package ratelimit

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/polagonow/pola/core"
	"golang.org/x/time/rate"
)

// idleTTL is how long a per-client limiter may sit unused before the background
// sweeper evicts it, bounding memory against attackers cycling through keys.
const idleTTL = 10 * time.Minute

// sweepInterval is how often the sweeper scans for idle limiters.
const sweepInterval = time.Minute

type Option func(*config)

type config struct {
	rps            rate.Limit
	burst          int
	trustedProxies []*net.IPNet
}

func WithRequestsPerSecond(rps float64) Option {
	return func(c *config) { c.rps = rate.Limit(rps) }
}

func WithBurst(burst int) Option {
	return func(c *config) { c.burst = burst }
}

// WithTrustedProxies enables use of the X-Forwarded-For / X-Real-IP headers, but
// only for requests whose direct peer (r.RemoteAddr) falls within one of the
// given CIDRs. Without this option the headers are ignored and the direct peer
// address is always used as the rate-limit key, so a client cannot spoof a fresh
// bucket per request. Invalid CIDRs are ignored.
func WithTrustedProxies(cidrs []string) Option {
	return func(c *config) {
		for _, cidr := range cidrs {
			if _, ipnet, err := net.ParseCIDR(cidr); err == nil {
				c.trustedProxies = append(c.trustedProxies, ipnet)
			}
		}
	}
}

// entry is a rate limiter plus the last time it served a request; lastSeen is
// stored as a Unix-nano int64 accessed under the sync.Map's per-key semantics.
type entry struct {
	limiter  *rate.Limiter
	lastSeen atomic.Int64
}

type mw struct {
	rps            rate.Limit
	burst          int
	trustedProxies []*net.IPNet
	limiters       sync.Map // key: client IP string -> *entry
}

func New(opts ...Option) core.Middleware {
	cfg := &config{rps: 10, burst: 20}
	for _, o := range opts {
		o(cfg)
	}
	m := &mw{rps: cfg.rps, burst: cfg.burst, trustedProxies: cfg.trustedProxies}
	m.startSweeper()
	return m
}

// startSweeper launches a single background goroutine that periodically evicts
// limiters idle longer than idleTTL. It is started once, at construction, so it
// never leaks a goroutine per request.
func (m *mw) startSweeper() {
	ticker := time.NewTicker(sweepInterval)
	go func() {
		for range ticker.C {
			cutoff := time.Now().Add(-idleTTL).UnixNano()
			m.limiters.Range(func(key, v any) bool {
				if v.(*entry).lastSeen.Load() < cutoff {
					m.limiters.Delete(key)
				}
				return true
			})
		}
	}()
}

func (m *mw) Name() string { return "ratelimit" }

func (m *mw) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := m.clientIP(r)
		v, _ := m.limiters.LoadOrStore(ip, &entry{limiter: rate.NewLimiter(m.rps, m.burst)})
		e := v.(*entry)
		e.lastSeen.Store(time.Now().UnixNano())
		if !e.limiter.Allow() {
			retryAfter := time.Second
			w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP resolves the rate-limit key. Forwarded headers are honored only when
// the direct peer is a configured trusted proxy; otherwise the direct peer
// address is used so a client cannot forge a fresh bucket per request.
func (m *mw) clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if m.trusted(host) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if i := strings.IndexByte(xff, ','); i > 0 {
				return strings.TrimSpace(xff[:i])
			}
			return strings.TrimSpace(xff)
		}
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return strings.TrimSpace(xri)
		}
	}
	return host
}

// trusted reports whether host (the direct peer IP) is a configured trusted
// proxy allowed to supply forwarded-for headers.
func (m *mw) trusted(host string) bool {
	if len(m.trustedProxies) == 0 {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, ipnet := range m.trustedProxies {
		if ipnet.Contains(ip) {
			return true
		}
	}
	return false
}
