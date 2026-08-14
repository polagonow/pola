package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/polagonow/pola/cache/redis"
	"github.com/polagonow/pola/core"
)

// newCache spins up an in-process miniredis and returns a Cache wired to it.
func newCache(t *testing.T) (*redis.Cache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	c, err := redis.New(redis.WithClient(client), redis.WithKeyPrefix("test:"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c, mr
}

func TestGet_Miss(t *testing.T) {
	c, _ := newCache(t)
	_, ok, err := c.Get(context.Background(), "missing")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected cache miss")
	}
}

func TestSet_Get(t *testing.T) {
	c, mr := newCache(t)
	ctx := context.Background()

	if err := c.Set(ctx, "k", []byte("v"), core.CacheOptions{}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	val, ok, err := c.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(val) != "v" {
		t.Fatalf("expected 'v', got %q", val)
	}
	// Key is namespaced with the configured prefix.
	if !mr.Exists("test:k") {
		t.Fatal("expected namespaced key test:k to exist")
	}
}

func TestSet_TTL(t *testing.T) {
	c, mr := newCache(t)
	ctx := context.Background()

	if err := c.Set(ctx, "k", []byte("v"), core.CacheOptions{TTL: time.Minute}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if _, ok, _ := c.Get(ctx, "k"); !ok {
		t.Fatal("expected hit before expiry")
	}
	mr.FastForward(2 * time.Minute)
	if _, ok, _ := c.Get(ctx, "k"); ok {
		t.Fatal("expected miss after TTL expiry")
	}
}

func TestSet_NoTTL_Persists(t *testing.T) {
	c, mr := newCache(t)
	ctx := context.Background()

	_ = c.Set(ctx, "k", []byte("v"), core.CacheOptions{})
	mr.FastForward(24 * time.Hour)
	if _, ok, _ := c.Get(ctx, "k"); !ok {
		t.Fatal("expected key with no TTL to persist")
	}
}

func TestDelete(t *testing.T) {
	c, _ := newCache(t)
	ctx := context.Background()

	_ = c.Set(ctx, "k", []byte("v"), core.CacheOptions{})
	if err := c.Delete(ctx, "k"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok, _ := c.Get(ctx, "k"); ok {
		t.Fatal("expected cache miss after delete")
	}
}

func TestInvalidate(t *testing.T) {
	c, _ := newCache(t)
	ctx := context.Background()

	_ = c.Set(ctx, "prefix/a", []byte("a"), core.CacheOptions{})
	_ = c.Set(ctx, "prefix/b", []byte("b"), core.CacheOptions{})
	_ = c.Set(ctx, "other/c", []byte("c"), core.CacheOptions{})

	if err := c.Invalidate(ctx, "prefix/"); err != nil {
		t.Fatalf("Invalidate: %v", err)
	}

	if _, ok, _ := c.Get(ctx, "prefix/a"); ok {
		t.Fatal("prefix/a should be evicted")
	}
	if _, ok, _ := c.Get(ctx, "prefix/b"); ok {
		t.Fatal("prefix/b should be evicted")
	}
	if _, ok, _ := c.Get(ctx, "other/c"); !ok {
		t.Fatal("other/c should still be present")
	}
}

func TestClear(t *testing.T) {
	c, mr := newCache(t)
	ctx := context.Background()

	// A key outside this cache's namespace must survive Clear.
	if err := mr.Set("outside", "keep"); err != nil {
		t.Fatal(err)
	}
	_ = c.Set(ctx, "a", []byte("1"), core.CacheOptions{})
	_ = c.Set(ctx, "b", []byte("2"), core.CacheOptions{})

	if err := c.Clear(ctx); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if _, ok, _ := c.Get(ctx, "a"); ok {
		t.Fatal("expected a to be cleared")
	}
	if _, ok, _ := c.Get(ctx, "b"); ok {
		t.Fatal("expected b to be cleared")
	}
	if !mr.Exists("outside") {
		t.Fatal("Clear must not touch keys outside the namespace")
	}
}

// TestClear_ManyKeys exercises the batched SCAN/DEL path beyond one batch.
func TestClear_ManyKeys(t *testing.T) {
	c, _ := newCache(t)
	ctx := context.Background()

	for i := 0; i < 600; i++ {
		key := "k" + string(rune('0'+i%10)) + string(rune('a'+i/10%26)) + string(rune('A'+i/260))
		_ = c.Set(ctx, key, []byte("v"), core.CacheOptions{})
	}
	if err := c.Clear(ctx); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if _, ok, _ := c.Get(ctx, "k0aA"); ok {
		t.Fatal("expected all namespaced keys cleared")
	}
}

func TestName(t *testing.T) {
	c, _ := newCache(t)
	if c.Name() != "redis" {
		t.Fatalf("expected 'redis', got %q", c.Name())
	}
}
