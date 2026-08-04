// Package redis provides a Redis-backed cache for Pola, implementing
// core.Cache on top of github.com/redis/go-redis/v9.
//
// Connections are established lazily by go-redis on first use, so constructing
// a Cache never blocks on or fails because of an unreachable server; operations
// return the underlying error instead. Keys are namespaced with a configurable
// prefix (default "pola:cache:") so Clear and Invalidate only ever touch this
// app's keys, never the rest of a shared Redis database.
package redis

import (
	"context"
	"errors"
	"fmt"

	goredis "github.com/redis/go-redis/v9"

	"github.com/polagonow/pola/core"
)

// defaultKeyPrefix namespaces every key so Clear/Invalidate stay scoped to this
// app even when the Redis database is shared.
const defaultKeyPrefix = "pola:cache:"

// scanBatch is the COUNT hint for SCAN and the DEL batch size used by the
// prefix-based Invalidate/Clear operations.
const scanBatch = 256

// Cache is a Redis-backed core.Cache.
type Cache struct {
	client    goredis.UniversalClient
	keyPrefix string
}

// Option configures a Cache.
type Option func(*config)

type config struct {
	addr      string
	password  string
	db        int
	keyPrefix string
	client    goredis.UniversalClient
}

// WithAddr sets the Redis address (host:port).
func WithAddr(addr string) Option { return func(c *config) { c.addr = addr } }

// WithPassword sets the Redis auth password.
func WithPassword(password string) Option { return func(c *config) { c.password = password } }

// WithDB selects the Redis logical database number.
func WithDB(db int) Option { return func(c *config) { c.db = db } }

// WithKeyPrefix overrides the key namespace prefix (default "pola:cache:").
func WithKeyPrefix(prefix string) Option { return func(c *config) { c.keyPrefix = prefix } }

// WithClient supplies a pre-built go-redis client, bypassing addr/password/db.
// Useful for tests and for sharing a client with other subsystems.
func WithClient(client goredis.UniversalClient) Option {
	return func(c *config) { c.client = client }
}

// New creates a Redis cache. When no client is supplied via WithClient, one is
// built from the address/password/db options (defaulting to localhost:6379).
func New(opts ...Option) (*Cache, error) {
	cfg := &config{
		addr:      "localhost:6379",
		keyPrefix: defaultKeyPrefix,
	}
	for _, o := range opts {
		o(cfg)
	}

	client := cfg.client
	if client == nil {
		client = goredis.NewClient(&goredis.Options{
			Addr:     cfg.addr,
			Password: cfg.password,
			DB:       cfg.db,
		})
	}
	return &Cache{client: client, keyPrefix: cfg.keyPrefix}, nil
}

// MustNew creates a Redis cache; panics on error. Used by generated plugin
// wiring.
func MustNew(opts ...Option) *Cache {
	c, err := New(opts...)
	if err != nil {
		panic(err)
	}
	return c
}

// Name returns the adapter name, "redis".
func (c *Cache) Name() string { return "redis" }

// k returns the namespaced Redis key for a logical cache key.
func (c *Cache) k(key string) string { return c.keyPrefix + key }

// Get returns the value for key. A miss (or expired key) yields (nil, false, nil).
func (c *Cache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	val, err := c.client.Get(ctx, c.k(key)).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, false, nil // miss is not an error
		}
		return nil, false, fmt.Errorf("redis cache: get %q: %w", key, err)
	}
	return val, true, nil
}

// Set stores val under key. opts.TTL of zero means no expiry.
func (c *Cache) Set(ctx context.Context, key string, val []byte, opts core.CacheOptions) error {
	// opts.TTL == 0 maps to go-redis expiration 0 → no expiry.
	if err := c.client.Set(ctx, c.k(key), val, opts.TTL).Err(); err != nil {
		return fmt.Errorf("redis cache: set %q: %w", key, err)
	}
	return nil
}

// Delete removes key from the cache.
func (c *Cache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, c.k(key)).Err(); err != nil {
		return fmt.Errorf("redis cache: delete %q: %w", key, err)
	}
	return nil
}

// Invalidate deletes every key whose logical name starts with prefix.
func (c *Cache) Invalidate(ctx context.Context, prefix string) error {
	return c.deleteMatching(ctx, c.k(prefix)+"*")
}

// Clear deletes every key in this cache's namespace. It is scoped to the key
// prefix (not FLUSHDB) so a shared Redis database is left untouched.
func (c *Cache) Clear(ctx context.Context) error {
	return c.deleteMatching(ctx, c.keyPrefix+"*")
}

// deleteMatching SCANs for keys matching pattern and deletes them in batches.
func (c *Cache) deleteMatching(ctx context.Context, pattern string) error {
	iter := c.client.Scan(ctx, 0, pattern, scanBatch).Iterator()
	batch := make([]string, 0, scanBatch)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := c.client.Del(ctx, batch...).Err(); err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}
	for iter.Next(ctx) {
		batch = append(batch, iter.Val())
		if len(batch) >= scanBatch {
			if err := flush(); err != nil {
				return fmt.Errorf("redis cache: delete matching %q: %w", pattern, err)
			}
		}
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("redis cache: scan %q: %w", pattern, err)
	}
	if err := flush(); err != nil {
		return fmt.Errorf("redis cache: delete matching %q: %w", pattern, err)
	}
	return nil
}
