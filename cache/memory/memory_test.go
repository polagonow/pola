package memory_test

import (
	"context"
	"testing"

	"github.com/polagonow/pola/cache/memory"
	"github.com/polagonow/pola/core"
)

func newCache(t *testing.T) *memory.Cache {
	t.Helper()
	c, err := memory.New(16)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestGet_Miss(t *testing.T) {
	c := newCache(t)
	_, ok, err := c.Get(context.Background(), "missing")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected cache miss")
	}
}

func TestSet_Get(t *testing.T) {
	c := newCache(t)
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
}

func TestDelete(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	_ = c.Set(ctx, "k", []byte("v"), core.CacheOptions{})
	if err := c.Delete(ctx, "k"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, ok, _ := c.Get(ctx, "k")
	if ok {
		t.Fatal("expected cache miss after delete")
	}
}

func TestInvalidate(t *testing.T) {
	c := newCache(t)
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
	c := newCache(t)
	ctx := context.Background()

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
}

func TestName(t *testing.T) {
	c := newCache(t)
	if c.Name() != "memory" {
		t.Fatalf("expected 'memory', got %q", c.Name())
	}
}

func TestMustNew_DefaultSize(t *testing.T) {
	c := memory.MustNew(0) // 0 triggers default size
	if c == nil {
		t.Fatal("MustNew returned nil")
	}
}
