// Package slog provides a Logger implementation backed by log/slog.
package slog

import (
	"log/slog"

	"github.com/samber/do/v2"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/di"
)

type logger struct {
	inner *slog.Logger
}

// New creates a Logger backed by the default slog logger.
func New() core.Logger { return &logger{inner: slog.Default()} }

// NewWith creates a Logger backed by the provided slog.Logger.
func NewWith(l *slog.Logger) core.Logger { return &logger{inner: l} }

func (l *logger) Info(msg string, args ...any)  { l.inner.Info(msg, args...) }
func (l *logger) Error(msg string, args ...any) { l.inner.Error(msg, args...) }
func (l *logger) Debug(msg string, args ...any) { l.inner.Debug(msg, args...) }
func (l *logger) Warn(msg string, args ...any)  { l.inner.Warn(msg, args...) }
func (l *logger) With(args ...any) core.Logger  { return &logger{inner: l.inner.With(args...)} }

func init() {
	di.Stage(func(i do.Injector) {
		do.Provide(i, func(_ do.Injector) (core.Logger, error) {
			return New(), nil
		})
	})
}
