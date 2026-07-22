// Package chi implements core.WebFramework on top of the chi router
// (github.com/go-chi/chi/v5). User route handlers are written against the
// framework-neutral core.Context, so selecting chi requires no changes to route
// code.
package chi

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/webframework/pattern"
)

// defaultMaxMemory bounds in-memory multipart parsing (matches net/http).
const defaultMaxMemory = 32 << 20

// defaultMaxBindBytes caps the JSON request body size accepted by ShouldBind,
// guarding against unbounded-body DoS.
const defaultMaxBindBytes = 10 << 20 // 10 MB

// Framework is the chi-backed core.WebFramework.
type Framework struct {
	router chi.Router
	mws    []func(http.Handler) http.Handler
}

// New returns a new chi Framework.
func New() *Framework {
	return &Framework{router: chi.NewRouter()}
}

// Name implements core.WebFramework.
func (f *Framework) Name() string { return "chi" }

// Handle implements core.WebFramework.
func (f *Framework) Handle(method, p string, h core.HandlerFunc) {
	f.router.MethodFunc(method, pattern.ToChi(p), func(w http.ResponseWriter, r *http.Request) {
		// Mirror chi route params onto the request context for legacy handlers.
		if rctx := chi.RouteContext(r.Context()); rctx != nil && len(rctx.URLParams.Keys) > 0 {
			m := make(map[string]any, len(rctx.URLParams.Keys))
			for i, k := range rctx.URLParams.Keys {
				m[k] = rctx.URLParams.Values[i]
			}
			r = core.WithParams(r, m)
		}
		c := newContext(w, r)
		if err := h(c); err != nil && !c.w.wrote {
			_ = c.JSON(http.StatusInternalServerError, core.M{"error": err.Error()})
		}
	})
}

// Use implements core.WebFramework. Pola middleware wraps the whole engine
// (applied in Handler) to avoid chi's "Use after route" ordering constraint.
func (f *Framework) Use(mw func(http.Handler) http.Handler) {
	f.mws = append(f.mws, mw)
}

// Mount implements core.WebFramework.
func (f *Framework) Mount(prefix string, h http.Handler) {
	prefix = strings.TrimSuffix(prefix, "/")
	if prefix == "" {
		prefix = "/"
	}
	f.router.Mount(prefix, h)
}

// Fallback implements core.WebFramework.
func (f *Framework) Fallback(h http.Handler) { f.router.NotFound(h.ServeHTTP) }

// Handler implements core.WebFramework.
func (f *Framework) Handler() http.Handler {
	var h http.Handler = f.router
	for i := len(f.mws) - 1; i >= 0; i-- {
		h = f.mws[i](h)
	}
	return h
}

// ── chiContext ───────────────────────────────────────────────────────────────

var _ core.Context = (*chiContext)(nil)

type chiContext struct {
	w *statusWriter
	r *http.Request
}

func newContext(w http.ResponseWriter, r *http.Request) *chiContext {
	return &chiContext{w: &statusWriter{ResponseWriter: w, code: http.StatusOK}, r: r}
}

func (c *chiContext) Request() *http.Request      { return c.r }
func (c *chiContext) Writer() http.ResponseWriter { return c.w }
func (c *chiContext) Ctx() context.Context        { return c.r.Context() }
func (c *chiContext) SetContext(ctx context.Context) { c.r = c.r.WithContext(ctx) }
func (c *chiContext) Status() int                 { return c.w.code }
func (c *chiContext) Param(name string) string    { return chi.URLParam(c.r, name) }
func (c *chiContext) Query(name string) string    { return c.r.URL.Query().Get(name) }
func (c *chiContext) Header(name string) string   { return c.r.Header.Get(name) }

func (c *chiContext) ShouldBind(v any) error {
	c.r.Body = http.MaxBytesReader(c.w, c.r.Body, defaultMaxBindBytes)
	dec := json.NewDecoder(c.r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

func (c *chiContext) ShouldBindUri(v any) error    { return core.ShouldBindUri(c, v) }
func (c *chiContext) ShouldBindHeader(v any) error { return core.ShouldBindHeader(c, v) }
func (c *chiContext) ShouldBindQuery(v any) error  { return core.ShouldBindQuery(c, v) }
func (c *chiContext) ShouldBindForm(v any) error   { return core.ShouldBindForm(c, v) }

func (c *chiContext) Bind(v any) error    { return core.MustBindValidate(c, v, c.ShouldBind(v)) }
func (c *chiContext) BindUri(v any) error { return core.MustBindValidate(c, v, c.ShouldBindUri(v)) }
func (c *chiContext) BindHeader(v any) error {
	return core.MustBindValidate(c, v, c.ShouldBindHeader(v))
}
func (c *chiContext) BindQuery(v any) error { return core.MustBindValidate(c, v, c.ShouldBindQuery(v)) }
func (c *chiContext) BindForm(v any) error  { return core.MustBindValidate(c, v, c.ShouldBindForm(v)) }

func (c *chiContext) FormValue(name string) string { return c.r.FormValue(name) }

func (c *chiContext) FormFile(name string) (*multipart.FileHeader, error) {
	_, fh, err := c.r.FormFile(name)
	return fh, err
}

func (c *chiContext) MultipartForm() (*multipart.Form, error) {
	if err := c.r.ParseMultipartForm(defaultMaxMemory); err != nil {
		return nil, err
	}
	return c.r.MultipartForm, nil
}

func (c *chiContext) JSON(status int, v any) error {
	c.w.Header().Set("Content-Type", "application/json")
	c.w.WriteHeader(status)
	return json.NewEncoder(c.w).Encode(v)
}

func (c *chiContext) String(status int, s string) error {
	c.w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.w.WriteHeader(status)
	_, err := c.w.Write([]byte(s))
	return err
}

func (c *chiContext) NoContent(status int) error {
	c.w.WriteHeader(status)
	return nil
}

func (c *chiContext) SetHeader(name, value string) { c.w.Header().Set(name, value) }

func (c *chiContext) Redirect(status int, url string) error {
	http.Redirect(c.w, c.r, url, status)
	return nil
}

// statusWriter records the status code and whether anything was written.
type statusWriter struct {
	http.ResponseWriter
	code  int
	wrote bool
}

func (s *statusWriter) WriteHeader(code int) {
	s.code = code
	s.wrote = true
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusWriter) Write(b []byte) (int, error) {
	s.wrote = true
	return s.ResponseWriter.Write(b)
}

func (s *statusWriter) Flush() {
	if fl, ok := s.ResponseWriter.(http.Flusher); ok {
		fl.Flush()
	}
}
