package redis

import (
	"context"
	"fmt"

	"github.com/gorilla/sessions"
	"github.com/rbcervilla/redisstore/v9"
	"github.com/redis/go-redis/v9"
)

type Option func(*config)

type config struct {
	addr      string
	password  string
	db        int
	keyPrefix string
}

func WithAddr(addr string) Option {
	return func(c *config) { c.addr = addr }
}

func WithPassword(password string) Option {
	return func(c *config) { c.password = password }
}

func WithDB(db int) Option {
	return func(c *config) { c.db = db }
}

func WithKeyPrefix(prefix string) Option {
	return func(c *config) { c.keyPrefix = prefix }
}

func NewStore(opts ...Option) (sessions.Store, error) {
	cfg := &config{
		addr:      "localhost:6379",
		keyPrefix: "session:",
	}
	for _, o := range opts {
		o(cfg)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.addr,
		Password: cfg.password,
		DB:       cfg.db,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("session/redis: connect: %w", err)
	}

	store, err := redisstore.NewRedisStore(context.Background(), client)
	if err != nil {
		return nil, fmt.Errorf("session/redis: create store: %w", err)
	}

	store.KeyPrefix(cfg.keyPrefix)
	return store, nil
}

func MustNew(opts ...Option) sessions.Store {
	s, err := NewStore(opts...)
	if err != nil {
		panic(err)
	}
	return s
}
