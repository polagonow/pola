// Package memory provides an in-memory LRU cache backed by github.com/hashicorp/golang-lru.
package memory

import (
	"context"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/polagonow/pola/core"
)

const defaultSize = 1024

// entry wraps a cached value with an optional expiration time.
type entry struct {
	val       []byte
	expiresAt time.Time // zero value = no expiry
}

// Cache is an LRU-backed in-memory cache.
type Cache struct {
	mu  sync.RWMutex
	lru *lru.Cache[string, entry]
}

// New creates an LRU cache with the given maximum entry count.
func New(size int) (*Cache, error) {
	if size <= 0 {
		size = defaultSize
	}
	l, err := lru.New[string, entry](size)
	if err != nil {
		return nil, err
	}
	return &Cache{lru: l}, nil
}

// MustNew creates an LRU cache; panics on error.
func MustNew(size int) *Cache {
	c, err := New(size)
	if err != nil {
		panic(err)
	}
	return c
}

func (c *Cache) Name() string { return "memory" }

func (c *Cache) Get(_ context.Context, key string) ([]byte, bool, error) {
	c.mu.RLock()
	e, ok := c.lru.Get(key)
	c.mu.RUnlock()
	if !ok {
		return nil, false, nil
	}
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		// Expired — remove lazily.
		c.mu.Lock()
		c.lru.Remove(key)
		c.mu.Unlock()
		return nil, false, nil
	}
	return e.val, true, nil
}

func (c *Cache) Set(_ context.Context, key string, val []byte, opts core.CacheOptions) error {
	e := entry{val: val}
	if opts.TTL > 0 {
		e.expiresAt = time.Now().Add(opts.TTL)
	}
	c.mu.Lock()
	c.lru.Add(key, e)
	c.mu.Unlock()
	return nil
}

func (c *Cache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	c.lru.Remove(key)
	c.mu.Unlock()
	return nil
}

func (c *Cache) Invalidate(_ context.Context, prefix string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, k := range c.lru.Keys() {
		if strings.HasPrefix(k, prefix) {
			c.lru.Remove(k)
		}
	}
	return nil
}

func (c *Cache) Clear(_ context.Context) error {
	c.mu.Lock()
	c.lru.Purge()
	c.mu.Unlock()
	return nil
}
