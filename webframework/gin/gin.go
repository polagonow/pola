// Package gin implements core.WebFramework on top of the Gin web framework
// (github.com/gin-gonic/gin). User route handlers are written against the
// framework-neutral core.Context, so selecting gin requires no changes to route
// code.
package gin

import (
	"context"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/webframework/pattern"
)

// Framework is the Gin-backed core.WebFramework.
type Framework struct {
	engine *gin.Engine
	mws    []func(http.Handler) http.Handler
}

// New returns a new gin Framework with no default middleware (Pola supplies its
// own logging/recovery via core.Middleware).
func New() *Framework {
	gin.SetMode(gin.ReleaseMode)
	return &Framework{engine: gin.New()}
}

// Name implements core.WebFramework.
func (f *Framework) Name() string { return "gin" }

// Handle implements core.WebFramework.
func (f *Framework) Handle(method, p string, h core.HandlerFunc) {
	f.engine.Handle(method, pattern.ToGin(p), func(gc *gin.Context) {
		// Mirror route params onto the request context so legacy net/http
		// handlers (routes.Param) observe them too.
		if len(gc.Params) > 0 {
			m := make(map[string]any, len(gc.Params))
			for _, pr := range gc.Params {
				m[pr.Key] = pr.Value
			}
			gc.Request = core.WithParams(gc.Request, m)
		}
		c := &ginContext{c: gc}
		if err := h(c); err != nil && !gc.Writer.Written() {
			gc.JSON(http.StatusInternalServerError, core.M{"error": err.Error()})
		}
	})
}

// Use implements core.WebFramework. Pola middleware is net/http-based, so it
// wraps the whole engine (applied in Handler).
func (f *Framework) Use(mw func(http.Handler) http.Handler) {
	f.mws = append(f.mws, mw)
}

// Mount implements core.WebFramework.
func (f *Framework) Mount(prefix string, h http.Handler) {
	f.engine.Any(prefix, gin.WrapH(h))
	f.engine.Any(joinWildcard(prefix), gin.WrapH(h))
}

// Fallback implements core.WebFramework.
func (f *Framework) Fallback(h http.Handler) { f.engine.NoRoute(gin.WrapH(h)) }

// Handler implements core.WebFramework.
func (f *Framework) Handler() http.Handler {
	var h http.Handler = f.engine
	for i := len(f.mws) - 1; i >= 0; i-- {
		h = f.mws[i](h)
	}
	return h
}

// joinWildcard appends gin's catch-all segment to a mount prefix.
func joinWildcard(prefix string) string {
	if prefix == "/" {
		return "/*any"
	}
	if prefix[len(prefix)-1] == '/' {
		return prefix + "*any"
	}
	return prefix + "/*any"
}

// ── ginContext ───────────────────────────────────────────────────────────────

var _ core.Context = (*ginContext)(nil)

type ginContext struct{ c *gin.Context }

func (g *ginContext) Request() *http.Request       { return g.c.Request }
func (g *ginContext) Writer() http.ResponseWriter  { return g.c.Writer }
func (g *ginContext) Ctx() context.Context         { return g.c.Request.Context() }
func (g *ginContext) SetContext(ctx context.Context) { g.c.Request = g.c.Request.WithContext(ctx) }
func (g *ginContext) Status() int                  { return g.c.Writer.Status() }
func (g *ginContext) Param(name string) string     { return g.c.Param(name) }
func (g *ginContext) Query(name string) string     { return g.c.Query(name) }
func (g *ginContext) Header(name string) string    { return g.c.GetHeader(name) }
func (g *ginContext) ShouldBind(v any) error       { return g.c.ShouldBindJSON(v) }
func (g *ginContext) ShouldBindUri(v any) error    { return core.ShouldBindUri(g, v) }
func (g *ginContext) ShouldBindHeader(v any) error { return core.ShouldBindHeader(g, v) }
func (g *ginContext) ShouldBindQuery(v any) error  { return core.ShouldBindQuery(g, v) }
func (g *ginContext) ShouldBindForm(v any) error   { return core.ShouldBindForm(g, v) }
func (g *ginContext) Bind(v any) error             { return core.MustBindValidate(g, v, g.ShouldBind(v)) }
func (g *ginContext) BindUri(v any) error          { return core.MustBindValidate(g, v, g.ShouldBindUri(v)) }
func (g *ginContext) BindHeader(v any) error {
	return core.MustBindValidate(g, v, g.ShouldBindHeader(v))
}
func (g *ginContext) BindQuery(v any) error        { return core.MustBindValidate(g, v, g.ShouldBindQuery(v)) }
func (g *ginContext) BindForm(v any) error         { return core.MustBindValidate(g, v, g.ShouldBindForm(v)) }
func (g *ginContext) FormValue(name string) string { return g.c.Request.FormValue(name) }
func (g *ginContext) FormFile(name string) (*multipart.FileHeader, error) {
	return g.c.FormFile(name)
}
func (g *ginContext) MultipartForm() (*multipart.Form, error) { return g.c.MultipartForm() }

func (g *ginContext) JSON(status int, v any) error {
	g.c.JSON(status, v)
	return nil
}

func (g *ginContext) String(status int, s string) error {
	g.c.Data(status, "text/plain; charset=utf-8", []byte(s))
	return nil
}

func (g *ginContext) NoContent(status int) error {
	g.c.Status(status)
	g.c.Writer.WriteHeaderNow()
	return nil
}

func (g *ginContext) SetHeader(name, value string) { g.c.Header(name, value) }

func (g *ginContext) Redirect(status int, url string) error {
	g.c.Redirect(status, url)
	return nil
}
