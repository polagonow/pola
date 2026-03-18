package slog_test

import (
	"log/slog"
	"testing"

	pkgslog "github.com/polagonow/pola/logger/slog"
)

func TestNew_NonNil(t *testing.T) {
	l := pkgslog.New()
	if l == nil {
		t.Fatal("New() returned nil")
	}
}

func TestNewWith_NonNil(t *testing.T) {
	l := pkgslog.NewWith(slog.Default())
	if l == nil {
		t.Fatal("NewWith() returned nil")
	}
}

func TestWith_ReturnsDifferentLogger(t *testing.T) {
	l := pkgslog.New()
	l2 := l.With("key", "value")
	if l2 == nil {
		t.Fatal("With() returned nil")
	}
	if l == l2 {
		t.Fatal("With() should return a new logger instance")
	}
}

func TestMethods_NoPanic(t *testing.T) {
	l := pkgslog.New()
	// These should not panic.
	l.Info("info message")
	l.Error("error message")
	l.Debug("debug message")
	l.Warn("warn message")
}
