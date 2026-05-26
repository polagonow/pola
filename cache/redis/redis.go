// Package redis provides a Redis-backed cache stub.
// Full implementation requires github.com/redis/go-redis/v9.
package redis

import (
	"context"
	"errors"

	"github.com/polagonow/pola/core"
)

// ErrNotImplemented is returned by all methods until fully implemented.
var ErrNotImplemented = errors.New("redis cache: not yet implemented")

// Cache is a stub Redis-backed cache.
type Cache struct{ Addr string }

// New creates a Redis cache stub targeting addr.
func New(addr string) *Cache { return &Cache{Addr: addr} }

func (c *Cache) Name() string { return "redis" }
func (c *Cache) Get(_ context.Context, _ string) ([]byte, bool, error) {
	return nil, false, ErrNotImplemented
}
func (c *Cache) Set(_ context.Context, _ string, _ []byte, _ core.CacheOptions) error {
	return ErrNotImplemented
}
func (c *Cache) Delete(_ context.Context, _ string) error     { return ErrNotImplemented }
func (c *Cache) Invalidate(_ context.Context, _ string) error { return ErrNotImplemented }
func (c *Cache) Clear(_ context.Context) error                { return ErrNotImplemented }
