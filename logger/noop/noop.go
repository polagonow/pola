// Package noop provides a no-op Logger that discards all log output.
package noop

import "github.com/polagonow/pola/core"

type logger struct{}

// New returns a no-op Logger.
func New() core.Logger { return logger{} }

func (logger) Info(msg string, args ...any)  {}
func (logger) Error(msg string, args ...any) {}
func (logger) Debug(msg string, args ...any) {}
func (logger) Warn(msg string, args ...any)  {}
func (logger) With(args ...any) core.Logger  { return logger{} }
