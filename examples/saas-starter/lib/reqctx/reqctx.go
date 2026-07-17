// Package reqctx carries request-scoped values (currently the client IP) into
// the context so bridged actions — which don't receive the *http.Request — can
// still record it (e.g. on activity-log entries).
package reqctx

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/polagonow/pola/core"
)

type ipKeyType struct{}

var ipKey = ipKeyType{}

// IP returns the client IP recorded for the request, or "" if none.
func IP(ctx context.Context) string {
	if v, ok := ctx.Value(ipKey).(string); ok {
		return v
	}
	return ""
}

type mw struct{}

func (mw) Name() string { return "client-ip" }

func (mw) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		ctx := context.WithValue(r.Context(), ipKey, ip)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Plugin registers the client-IP middleware.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "client-ip",
		Fn: func(r *core.Registry) {
			r.AddMiddleware(mw{})
		},
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
