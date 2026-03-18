// Package prometheus provides a Metrics implementation using Prometheus.
// It exposes a /metrics HTTP endpoint via the standard prometheus/client_golang library.
package prometheus

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/polagonow/pola/core"
)

// Metrics is a Prometheus-backed metrics collector.
type Metrics struct {
	requestDuration *prometheus.HistogramVec
	requestTotal    *prometheus.CounterVec
	reg             *prometheus.Registry
}

// New creates a Prometheus metrics collector.
func New() *Metrics {
	reg := prometheus.NewRegistry()
	m := &Metrics{
		reg: reg,
		requestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "pola_request_duration_seconds",
			Help:    "HTTP request latency",
			Buckets: prometheus.DefBuckets,
		}, []string{"route", "method", "status"}),
		requestTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "pola_requests_total",
			Help: "Total HTTP requests",
		}, []string{"route", "method", "status"}),
	}
	reg.MustRegister(m.requestDuration, m.requestTotal)
	return m
}

// Ensure Metrics satisfies core.Metrics.
var _ core.Metrics = (*Metrics)(nil)

// Name returns the metrics implementation name.
func (m *Metrics) Name() string { return "prometheus" }

// RecordRequest records a single HTTP request's route, method, status and duration.
func (m *Metrics) RecordRequest(route, method string, statusCode int, d time.Duration) {
	status := fmt.Sprintf("%d", statusCode)
	m.requestDuration.WithLabelValues(route, method, status).Observe(d.Seconds())
	m.requestTotal.WithLabelValues(route, method, status).Inc()
}

// Handler returns an HTTP handler that serves Prometheus metrics at /metrics.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{})
}

func init() {
	core.RegisterMetrics(func() core.Metrics { return New() })
}
