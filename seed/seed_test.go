package seed_test

import (
	"context"
	"errors"
	"testing"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/seed"
)

func TestRegisterAndRun(t *testing.T) {
	if seed.HasSeeds() {
		t.Skip("seed registry already populated by another test/build")
	}
	var calls int
	seed.Register(func(ctx context.Context, r *core.Registry) error {
		calls++
		return nil
	})
	seed.Register(nil) // ignored
	if !seed.HasSeeds() {
		t.Fatal("expected HasSeeds() true after Register")
	}
	if err := seed.Run(context.Background(), nil); err != nil {
		t.Fatalf("run: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 seed call, got %d", calls)
	}
}

func TestRun_PropagatesError(t *testing.T) {
	want := errors.New("boom")
	seed.Register(func(ctx context.Context, r *core.Registry) error { return want })
	if err := seed.Run(context.Background(), nil); !errors.Is(err, want) {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}
