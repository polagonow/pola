// Package logging provides a request-logging Middleware.
package logging

import (
	"net/http"
	"time"

	"github.com/polagonow/pola/core"
)

type mw struct{ log core.Logger }

// New creates a logging middleware using the given Logger.
func New(log core.Logger) core.Middleware { return &mw{log: log} }

func (m *mw) Name() string { return "logging" }

func (m *mw) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(rw, r)
		m.log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.code,
			"duration", time.Since(start).String(),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	code int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.code = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
