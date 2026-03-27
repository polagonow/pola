// Package prometheus provides a Metrics implementation using Prometheus.
// It exposes a /metrics HTTP endpoint via the standard prometheus/client_golang library.
package prometheus

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	samberdo "github.com/samber/do/v2"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/di"
)

// Metrics is a Prometheus-backed metrics collector.
type Metrics struct {
	requestDuration *prometheus.HistogramVec
	requestTotal    *prometheus.CounterVec
	renderDuration  *prometheus.HistogramVec
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
		renderDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "pola_render_duration_seconds",
			Help:    "SSR render latency per route",
			Buckets: prometheus.DefBuckets,
		}, []string{"route"}),
	}
	reg.MustRegister(m.requestDuration, m.requestTotal, m.renderDuration)
	return m
}

// Ensure Metrics satisfies core.Metrics.
var _ core.Metrics = (*Metrics)(nil)

// Name returns the metrics implementation name.
func (m *Metrics) Name() string { return "prometheus" }

// Path returns the metrics endpoint path.
func (m *Metrics) Path() string { return "/metrics" }

// RecordRequest records a single HTTP request's route, method, status and duration.
func (m *Metrics) RecordRequest(route, method string, statusCode int, d time.Duration) {
	status := fmt.Sprintf("%d", statusCode)
	m.requestDuration.WithLabelValues(route, method, status).Observe(d.Seconds())
	m.requestTotal.WithLabelValues(route, method, status).Inc()
}

// RecordRender records the SSR render duration for a given route.
func (m *Metrics) RecordRender(route string, d time.Duration) {
	m.renderDuration.WithLabelValues(route).Observe(d.Seconds())
}

// Handler returns an HTTP handler that serves Prometheus metrics at /metrics.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{})
}

func init() {
	di.Stage(func(i samberdo.Injector) {
		samberdo.Provide(i, func(_ samberdo.Injector) (core.Metrics, error) {
			return New(), nil
		})
	})
}
