// Package echo implements core.WebFramework on top of the Echo web framework
// (github.com/labstack/echo/v4). User route handlers are written against the
// framework-neutral core.Context, so selecting echo requires no changes to
// route code.
package echo

import (
	"context"
	"mime/multipart"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/webframework/pattern"
)

// Framework is the Echo-backed core.WebFramework.
type Framework struct {
	echo *echo.Echo
	mws  []func(http.Handler) http.Handler
}

// New returns a new echo Framework.
func New() *Framework {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	return &Framework{echo: e}
}

// Name implements core.WebFramework.
func (f *Framework) Name() string { return "echo" }

// Handle implements core.WebFramework.
func (f *Framework) Handle(method, p string, h core.HandlerFunc) {
	f.echo.Add(method, pattern.ToEcho(p), func(ec echo.Context) error {
		// Mirror route params onto the request context for legacy handlers.
		if names := ec.ParamNames(); len(names) > 0 {
			m := make(map[string]any, len(names))
			for _, n := range names {
				m[n] = ec.Param(n)
			}
			ec.SetRequest(core.WithParams(ec.Request(), m))
		}
		return h(&echoContext{c: ec})
	})
}

// Use implements core.WebFramework. Pola middleware is net/http-based, so it
// wraps the whole engine (applied in Handler).
func (f *Framework) Use(mw func(http.Handler) http.Handler) {
	f.mws = append(f.mws, mw)
}

// Mount implements core.WebFramework.
func (f *Framework) Mount(prefix string, h http.Handler) {
	wrapped := echo.WrapHandler(h)
	f.echo.Any(prefix, wrapped)
	f.echo.Any(joinWildcard(prefix), wrapped)
}

// Fallback implements core.WebFramework. The catch-all "/*" route has the
// lowest matching priority, so explicit routes still win.
func (f *Framework) Fallback(h http.Handler) {
	f.echo.Any("/*", echo.WrapHandler(h))
}

// Handler implements core.WebFramework.
func (f *Framework) Handler() http.Handler {
	var h http.Handler = f.echo
	for i := len(f.mws) - 1; i >= 0; i-- {
		h = f.mws[i](h)
	}
	return h
}

func joinWildcard(prefix string) string {
	if prefix == "/" {
		return "/*"
	}
	if prefix[len(prefix)-1] == '/' {
		return prefix + "*"
	}
	return prefix + "/*"
}

// ── echoContext ──────────────────────────────────────────────────────────────

var _ core.Context = (*echoContext)(nil)

type echoContext struct{ c echo.Context }

func (e *echoContext) Request() *http.Request       { return e.c.Request() }
func (e *echoContext) Writer() http.ResponseWriter  { return e.c.Response() }
func (e *echoContext) Ctx() context.Context         { return e.c.Request().Context() }
func (e *echoContext) Status() int                  { return e.c.Response().Status }
func (e *echoContext) Param(name string) string     { return e.c.Param(name) }
func (e *echoContext) Query(name string) string     { return e.c.QueryParam(name) }
func (e *echoContext) Header(name string) string    { return e.c.Request().Header.Get(name) }
func (e *echoContext) ShouldBind(v any) error       { return e.c.Bind(v) }
func (e *echoContext) ShouldBindUri(v any) error    { return core.ShouldBindUri(e, v) }
func (e *echoContext) ShouldBindHeader(v any) error { return core.ShouldBindHeader(e, v) }
func (e *echoContext) ShouldBindQuery(v any) error  { return core.ShouldBindQuery(e, v) }
func (e *echoContext) ShouldBindForm(v any) error   { return core.ShouldBindForm(e, v) }
func (e *echoContext) Bind(v any) error             { return core.MustBindValidate(e, v, e.ShouldBind(v)) }
func (e *echoContext) BindUri(v any) error          { return core.MustBindValidate(e, v, e.ShouldBindUri(v)) }
func (e *echoContext) BindHeader(v any) error {
	return core.MustBindValidate(e, v, e.ShouldBindHeader(v))
}
func (e *echoContext) BindQuery(v any) error {
	return core.MustBindValidate(e, v, e.ShouldBindQuery(v))
}
func (e *echoContext) BindForm(v any) error         { return core.MustBindValidate(e, v, e.ShouldBindForm(v)) }
func (e *echoContext) FormValue(name string) string { return e.c.FormValue(name) }
func (e *echoContext) FormFile(name string) (*multipart.FileHeader, error) {
	return e.c.FormFile(name)
}
func (e *echoContext) MultipartForm() (*multipart.Form, error) { return e.c.MultipartForm() }
func (e *echoContext) JSON(status int, v any) error            { return e.c.JSON(status, v) }
func (e *echoContext) String(status int, s string) error       { return e.c.String(status, s) }
func (e *echoContext) NoContent(status int) error              { return e.c.NoContent(status) }
func (e *echoContext) SetHeader(name, value string) {
	e.c.Response().Header().Set(name, value)
}
func (e *echoContext) Redirect(status int, url string) error {
	return e.c.Redirect(status, url)
}
