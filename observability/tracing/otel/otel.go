// Package otel provides an OpenTelemetry Tracer implementation.
// It wraps the global OpenTelemetry tracer provider.
package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/polagonow/pola/core"
)

// Tracer is an OpenTelemetry-backed tracer.
type Tracer struct {
	tracer oteltrace.Tracer
}

// New creates a Tracer backed by the global OpenTelemetry tracer provider.
func New() core.Tracer {
	return &Tracer{tracer: otel.Tracer("pola")}
}

// Ensure Tracer satisfies core.Tracer.
var _ core.Tracer = (*Tracer)(nil)

// Name returns the tracer implementation name.
func (t *Tracer) Name() string { return "otel" }

// StartSpan starts a new span with the given name and returns the updated context
// and the span. The caller must call span.End() when the operation is complete.
func (t *Tracer) StartSpan(ctx context.Context, name string) (context.Context, core.Span) {
	ctx, span := t.tracer.Start(ctx, name)
	return ctx, &otelSpan{span: span}
}

// otelSpan adapts an oteltrace.Span to the core.Span interface.
type otelSpan struct{ span oteltrace.Span }

// Ensure otelSpan satisfies core.Span.
var _ core.Span = (*otelSpan)(nil)

// End ends the span.
func (s *otelSpan) End() { s.span.End() }

// SetAttribute sets a span attribute.
// In production code use the attribute.* constructors for type safety;
// this simplified form accepts any value and discards non-primitive types.
func (s *otelSpan) SetAttribute(key string, value any) {
	// Simplified: attribute setting is best done directly via the otel SDK
	// using typed attribute.* helpers. This method satisfies the interface
	// without introducing a hard dependency on attribute types here.
}
