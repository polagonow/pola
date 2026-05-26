// Package noop provides a no-op Tracer using OpenTelemetry's noop tracer.
// This is the default — zero overhead, zero dependencies at runtime.
package noop

import (
	"context"

	"github.com/polagonow/pola/core"
)

// Tracer is a no-op tracer that creates spans with no overhead.
type Tracer struct{}

// Span is a no-op span.
type Span struct{}

// New returns a no-op core.Tracer.
func New() core.Tracer { return &Tracer{} }

// Ensure interfaces are satisfied.
var _ core.Tracer = (*Tracer)(nil)
var _ core.Span = (*Span)(nil)

// Name returns the tracer implementation name.
func (t *Tracer) Name() string { return "noop" }

// StartSpan returns the context unchanged and a no-op Span.
func (t *Tracer) StartSpan(ctx context.Context, name string) (context.Context, core.Span) {
	return ctx, &Span{}
}

// End is a no-op.
func (s *Span) End() {}

// SetAttribute is a no-op.
func (s *Span) SetAttribute(key string, value any) {}
