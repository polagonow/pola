package noop_test

import (
	"testing"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/logger/noop"
)

func TestNew_SatisfiesInterface(t *testing.T) {
	var _ core.Logger = noop.New()
}

func TestMethods_NoPanic(t *testing.T) {
	l := noop.New()
	// All operations should silently discard without panicking.
	l.Info("info", "k", "v")
	l.Error("error", "k", "v")
	l.Debug("debug", "k", "v")
	l.Warn("warn", "k", "v")
	l2 := l.With("key", "value")
	if l2 == nil {
		t.Fatal("With() returned nil")
	}
	l2.Info("chained")
}
