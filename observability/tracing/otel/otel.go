// Package otel provides an OpenTelemetry Tracer implementation.
// It wraps the global OpenTelemetry tracer provider.
package otel

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/di"
)

func init() {
	di.Stage(func(i do.Injector) {
		do.Provide(i, func(_ do.Injector) (core.Tracer, error) {
			return New(), nil
		})
	})
}

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
// It converts the value to the appropriate OTEL attribute type.
func (s *otelSpan) SetAttribute(key string, value any) {
	s.span.SetAttributes(toAttribute(key, value))
}

// toAttribute converts a key/value pair to an OTEL attribute.KeyValue.
func toAttribute(key string, value any) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case bool:
		return attribute.Bool(key, v)
	case int:
		return attribute.Int(key, v)
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	default:
		return attribute.String(key, fmt.Sprintf("%v", v))
	}
}
