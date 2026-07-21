// Package health provides liveness and readiness HTTP endpoints as a Middleware,
// filling a gap for Kubernetes / Cloud Run style deployments (pola exposed
// /metrics and pprof but no health endpoint).
//
// It short-circuits its two paths before routing and passes everything else
// through, so it needs no pipeline wiring — register it like any middleware:
//
//	pola.Use(health.Plugin(
//	    health.WithCheck("db", func(ctx context.Context) error { return db.PingContext(ctx) }),
//	))
//
// Liveness (/healthz) always returns 200 while the process is up. Readiness
// (/readyz) runs the registered checks and returns 200 when all pass, or 503
// with the per-check status when any fails. Checks are user-supplied funcs, so
// this package depends on no database or other subsystem.
package health

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/polagonow/pola/core"
)

// Check is a named readiness probe. A non-nil error marks the dependency
// not-ready and fails /readyz with 503.
type Check struct {
	Name string
	Func func(ctx context.Context) error
}

// Option configures the health middleware.
type Option func(*mw)

// WithLivePath overrides the liveness path (default "/healthz").
func WithLivePath(p string) Option { return func(m *mw) { m.livePath = p } }

// WithReadyPath overrides the readiness path (default "/readyz").
func WithReadyPath(p string) Option { return func(m *mw) { m.readyPath = p } }

// WithCheck registers a readiness probe run by /readyz.
func WithCheck(name string, fn func(ctx context.Context) error) Option {
	return func(m *mw) { m.checks = append(m.checks, Check{Name: name, Func: fn}) }
}

type mw struct {
	livePath  string
	readyPath string
	checks    []Check
}

// New builds the health middleware.
func New(opts ...Option) core.Middleware {
	m := &mw{livePath: "/healthz", readyPath: "/readyz"}
	for _, o := range opts {
		o(m)
	}
	return m
}

func (m *mw) Name() string { return "health" }

func (m *mw) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			switch r.URL.Path {
			case m.livePath:
				writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
				return
			case m.readyPath:
				m.serveReady(w, r)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (m *mw) serveReady(w http.ResponseWriter, r *http.Request) {
	results := make(map[string]string, len(m.checks))
	ready := true
	for _, c := range m.checks {
		if err := c.Func(r.Context()); err != nil {
			results[c.Name] = err.Error()
			ready = false
		} else {
			results[c.Name] = "ok"
		}
	}

	status := http.StatusOK
	statusText := "ok"
	if !ready {
		status = http.StatusServiceUnavailable
		statusText = "unavailable"
	}
	writeJSON(w, status, map[string]any{"status": statusText, "checks": results})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
