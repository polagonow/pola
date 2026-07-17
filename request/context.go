package request

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/polagonow/pola/core"
)

// Info is a lightweight snapshot of the incoming HTTP request, captured by the
// request-info middleware so that bridged actions — which receive a
// context.Context but not the *http.Request — can still read request metadata
// (client IP, user-agent, headers).
type Info struct {
	IP        string
	UserAgent string
	Host      string
	Method    string
	Path      string
	header    http.Header
}

// Header returns a request header value (case-insensitive), or "".
func (i *Info) Header(name string) string {
	if i == nil || i.header == nil {
		return ""
	}
	return i.header.Get(name)
}

type infoCtxKey struct{}

// FromContext returns the request Info captured for this request, if the
// request-info middleware ran.
func FromContext(ctx context.Context) (*Info, bool) {
	i, ok := ctx.Value(infoCtxKey{}).(*Info)
	return i, ok
}

// ClientIP returns the best-guess client IP: the first hop of X-Forwarded-For,
// else the connection's remote address.
func ClientIP(ctx context.Context) string {
	if i, ok := FromContext(ctx); ok {
		return i.IP
	}
	return ""
}

// UserAgent returns the request User-Agent, or "".
func UserAgent(ctx context.Context) string {
	if i, ok := FromContext(ctx); ok {
		return i.UserAgent
	}
	return ""
}

// Host returns the request Host, or "".
func Host(ctx context.Context) string {
	if i, ok := FromContext(ctx); ok {
		return i.Host
	}
	return ""
}

// Header returns a request header value by name (case-insensitive), or "".
func Header(ctx context.Context, name string) string {
	if i, ok := FromContext(ctx); ok {
		return i.Header(name)
	}
	return ""
}

type infoMW struct{}

func (infoMW) Name() string { return "request-info" }

func (infoMW) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := &Info{
			IP:        clientIP(r),
			UserAgent: r.UserAgent(),
			Host:      r.Host,
			Method:    r.Method,
			Path:      r.URL.Path,
			header:    r.Header,
		}
		ctx := context.WithValue(r.Context(), infoCtxKey{}, info)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Middleware returns the request-info middleware directly (for apps that wire
// middleware manually or for tests).
func Middleware() core.Middleware { return infoMW{} }

// Plugin registers the request-info middleware, which captures request metadata
// into the context so request.ClientIP/UserAgent/Header work inside actions.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "request-info",
		Fn: func(r *core.Registry) {
			r.AddMiddleware(infoMW{})
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
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return strings.TrimSpace(xr)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
