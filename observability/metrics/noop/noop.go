// Package noop provides a no-op Metrics implementation.
// It discards all recorded metrics and serves a 404 from its handler.
package noop

import (
	"net/http"
	"time"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/reserved"
)

// Metrics is a no-op metrics implementation.
type Metrics struct{}

// New returns a no-op core.Metrics.
func New() core.Metrics { return &Metrics{} }

// Ensure Metrics satisfies core.Metrics.
var _ core.Metrics = (*Metrics)(nil)

// Name returns the metrics implementation name.
func (m *Metrics) Name() string { return "noop" }

// Path returns the metrics endpoint path.
func (m *Metrics) Path() string { return reserved.Metrics }

// RecordRequest is a no-op.
func (m *Metrics) RecordRequest(route, method string, statusCode int, d time.Duration) {}

// RecordRender is a no-op.
func (m *Metrics) RecordRender(route string, d time.Duration) {}

// Handler returns an HTTP 404 handler (no metrics endpoint exposed).
func (m *Metrics) Handler() http.Handler { return http.NotFoundHandler() }
