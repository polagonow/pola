package ratelimit

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/polagonow/pola/core"
	"golang.org/x/time/rate"
)

type Option func(*config)

type config struct {
	rps   rate.Limit
	burst int
}

func WithRequestsPerSecond(rps float64) Option {
	return func(c *config) { c.rps = rate.Limit(rps) }
}

func WithBurst(burst int) Option {
	return func(c *config) { c.burst = burst }
}

type mw struct {
	rps      rate.Limit
	burst    int
	limiters sync.Map
}

func New(opts ...Option) core.Middleware {
	cfg := &config{rps: 10, burst: 20}
	for _, o := range opts {
		o(cfg)
	}
	return &mw{rps: cfg.rps, burst: cfg.burst}
}

func (m *mw) Name() string { return "ratelimit" }

func (m *mw) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		v, _ := m.limiters.LoadOrStore(ip, rate.NewLimiter(m.rps, m.burst))
		limiter := v.(*rate.Limiter)
		if !limiter.Allow() {
			retryAfter := time.Second
			w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
