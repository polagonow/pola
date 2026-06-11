package session

import (
	"context"
	"crypto/rand"
	"encoding/gob"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/polagonow/pola/core"
)

func init() {
	gob.Register(uint(0))
	gob.Register(uint64(0))
	gob.Register(map[string]string{})
	gob.Register(map[string]any{})
}

type Option func(*config)

type config struct {
	store      sessions.Store
	cookieName string
	maxAge     int
	secure     bool
	httpOnly   bool
	path       string
}

// WithStore sets the gorilla/sessions Store implementation.
// Built-in: sessions.NewCookieStore, sessions.NewFilesystemStore.
// Third-party: redistore, pgstore, gormstore, etc.
func WithStore(store sessions.Store) Option {
	return func(c *config) { c.store = store }
}

func WithCookieName(name string) Option {
	return func(c *config) { c.cookieName = name }
}

func WithMaxAge(seconds int) Option {
	return func(c *config) { c.maxAge = seconds }
}

func WithSecure(secure bool) Option {
	return func(c *config) { c.secure = secure }
}

func WithPath(path string) Option {
	return func(c *config) { c.path = path }
}

type ctxData struct {
	sess *sessions.Session
	req  *http.Request
	w    http.ResponseWriter
}

// saveWriter intercepts Write/WriteHeader to save the session cookie
// before the response body is flushed.
type saveWriter struct {
	http.ResponseWriter
	sess  *sessions.Session
	req   *http.Request
	saved bool
}

func (sw *saveWriter) save() {
	if !sw.saved {
		sw.saved = true
		_ = sw.sess.Save(sw.req, sw.ResponseWriter)
	}
}

func (sw *saveWriter) WriteHeader(code int) {
	sw.save()
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *saveWriter) Write(b []byte) (int, error) {
	sw.save()
	return sw.ResponseWriter.Write(b)
}

type mw struct {
	cfg config
}

func New(opts ...Option) core.Middleware {
	cfg := config{
		cookieName: "_pola_session",
		maxAge:     86400,
		secure:     true,
		httpOnly:   true,
		path:       "/",
	}
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.store == nil {
		key := make([]byte, 32)
		_, _ = rand.Read(key)
		cs := sessions.NewCookieStore(key)
		cs.Options = &sessions.Options{
			Path:     cfg.path,
			MaxAge:   cfg.maxAge,
			Secure:   cfg.secure,
			HttpOnly: cfg.httpOnly,
			SameSite: http.SameSiteLaxMode,
		}
		cfg.store = cs
	}
	return &mw{cfg: cfg}
}

func (m *mw) Name() string { return "session" }

func (m *mw) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, _ := m.cfg.store.Get(r, m.cfg.cookieName)
		sw := &saveWriter{ResponseWriter: w, sess: sess, req: r}
		cd := &ctxData{sess: sess, req: r, w: w}
		ctx := context.WithValue(r.Context(), core.SessionContextKey, cd)
		next.ServeHTTP(sw, r.WithContext(ctx))
		sw.save()
	})
}

func getCtx(ctx context.Context) *ctxData {
	cd, _ := ctx.Value(core.SessionContextKey).(*ctxData)
	return cd
}

// Get returns a copy of all session values as map[string]any.
func Get(ctx context.Context) map[string]any {
	cd := getCtx(ctx)
	if cd == nil || cd.sess == nil {
		return make(map[string]any)
	}
	result := make(map[string]any, len(cd.sess.Values))
	for k, v := range cd.sess.Values {
		if key, ok := k.(string); ok {
			result[key] = v
		}
	}
	return result
}

// Set stores a value in the session.
func Set(ctx context.Context, key string, value any) {
	cd := getCtx(ctx)
	if cd == nil || cd.sess == nil {
		return
	}
	cd.sess.Values[key] = value
}

// Delete removes a key from the session.
func Delete(ctx context.Context, key string) {
	cd := getCtx(ctx)
	if cd == nil || cd.sess == nil {
		return
	}
	delete(cd.sess.Values, key)
}

// Clear removes all data from the session.
func Clear(ctx context.Context) {
	cd := getCtx(ctx)
	if cd == nil || cd.sess == nil {
		return
	}
	for k := range cd.sess.Values {
		delete(cd.sess.Values, k)
	}
}

// AddFlash adds a flash message (gorilla built-in flash support).
func AddFlash(ctx context.Context, value any, vars ...string) {
	cd := getCtx(ctx)
	if cd == nil || cd.sess == nil {
		return
	}
	cd.sess.AddFlash(value, vars...)
}

// Flashes returns and clears flash messages (gorilla built-in flash support).
func Flashes(ctx context.Context, vars ...string) []any {
	cd := getCtx(ctx)
	if cd == nil || cd.sess == nil {
		return nil
	}
	return cd.sess.Flashes(vars...)
}
