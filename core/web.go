package core

import (
	"context"
	"mime/multipart"
	"net/http"
)

// M is a convenience alias for building JSON response bodies, e.g.
// c.JSON(200, core.M{"status": "ok"}).
type M = map[string]any

// Context is the framework-neutral request/response abstraction handed to route
// handlers. Each WebFramework adapter implements it by wrapping its native
// context (gin.Context, echo.Context) or *http.Request/ResponseWriter (std/chi).
//
// User route handlers are written against this interface so swapping the
// selected web framework (gin → echo → chi → std) requires no code changes.
type Context interface {
	// Request returns the underlying *http.Request. Always available.
	Request() *http.Request
	// Writer returns the underlying http.ResponseWriter. Always available.
	Writer() http.ResponseWriter
	// Ctx is shorthand for Request().Context().
	Ctx() context.Context

	// Param returns a path parameter by name (e.g. "id" for /users/:id).
	Param(name string) string
	// Query returns a URL query-string value by name.
	Query(name string) string
	// Header returns a request header value by canonical name.
	Header(name string) string

	// Binding mirrors gin's two families. ShouldBind* decode into v and return
	// the error for the handler to deal with. Bind* (the "must" variants) do the
	// same but, on failure, also write a 400 JSON error response — so a handler
	// can `return err` and stop.
	//
	// Neither family validates: missing fields are left at their zero value, so
	// run validation.Validate(v) afterward using `validate:"..."` tags. Tag
	// reference: `uri` (path params), `header` (headers, case-insensitive),
	// `query` (query string), `form` (POST body); JSON binds the request body.

	// ShouldBind decodes the JSON request body into v.
	ShouldBind(v any) error
	// ShouldBindUri maps path parameters into v using `uri:"name"` tags.
	ShouldBindUri(v any) error
	// ShouldBindHeader maps request headers into v using `header:"Name"` tags.
	ShouldBindHeader(v any) error
	// ShouldBindQuery maps URL query values into v using `query:"name"` tags.
	// Repeated values bind to slice fields ([]string, []int, ...).
	ShouldBindQuery(v any) error
	// ShouldBindForm maps POST form-body values (urlencoded or multipart) into v
	// using `form:"name"` tags. File uploads stay on FormFile/MultipartForm.
	ShouldBindForm(v any) error

	// Bind is the must-variant of ShouldBind: it decodes the JSON request body
	// into v and writes a 400 JSON error response on failure.
	Bind(v any) error
	// BindUri is the must-variant of ShouldBindUri (writes 400 on failure).
	BindUri(v any) error
	// BindHeader is the must-variant of ShouldBindHeader (writes 400 on failure).
	BindHeader(v any) error
	// BindQuery is the must-variant of ShouldBindQuery (writes 400 on failure).
	BindQuery(v any) error
	// BindForm is the must-variant of ShouldBindForm (writes 400 on failure).
	BindForm(v any) error

	// FormValue returns the first value for the named form field (query string
	// or POST/PUT/PATCH form body).
	FormValue(name string) string
	// FormFile returns the first uploaded file for the named multipart form
	// field (mirrors gin/echo's c.FormFile).
	FormFile(name string) (*multipart.FileHeader, error)
	// MultipartForm parses and returns the full multipart form (fields + files).
	MultipartForm() (*multipart.Form, error)

	// JSON writes v as a JSON response with the given status code.
	JSON(status int, v any) error
	// String writes s as a text/plain response with the given status code.
	String(status int, s string) error
	// NoContent writes the given status code with an empty body.
	NoContent(status int) error
	// SetHeader sets a response header. Must be called before writing the body.
	SetHeader(name, value string)
	// Redirect issues an HTTP redirect to url with the given status code.
	Redirect(status int, url string) error

	// Status returns the response status code written so far (0 if none yet).
	Status() int
}

// HandlerFunc is a framework-neutral route handler. Returning a non-nil error
// lets adapters centralize error handling: if the handler has not already
// written a response, the adapter translates the error into a 500.
type HandlerFunc func(Context) error

// WebFramework is the pluggable HTTP engine. Exactly one implementation is
// registered in the DI container (selected via Polafile `framework`,
// `--framework`, or POLA_WEB_FRAMEWORK). The pipeline mounts Pola's machinery
// (API routes, page renderer, reserved endpoints, middleware) onto it and
// serves Handler().
//
// The default "std" adapter reproduces Pola's historical net/http behavior;
// gin/echo/chi adapters back the same surface with their respective engines.
type WebFramework interface {
	// Name returns the adapter name (e.g. "std", "gin", "echo", "chi").
	Name() string

	// Handle registers an API route. pattern uses Pola's ":param" / ":...rest"
	// syntax; the adapter translates it to the framework's native syntax.
	Handle(method, pattern string, h HandlerFunc)

	// Use registers net/http-compatible middleware applied to the whole engine.
	// core.Middleware.Wrap already returns func(http.Handler) http.Handler.
	Use(mw func(next http.Handler) http.Handler)

	// Mount attaches a plain http.Handler at a path prefix. Used for the
	// reserved /_pola/* endpoints, /public/, and server-action handlers.
	Mount(prefix string, h http.Handler)

	// Fallback sets the catch-all handler for requests that match no registered
	// route — the page renderer in full-stack mode, or a 404 handler otherwise.
	Fallback(h http.Handler)

	// Handler returns the fully-wired root http.Handler to serve.
	Handler() http.Handler
}

// WebFrameworkFactory creates a fresh WebFramework instance. Adapters register a
// factory (not a singleton) in the DI container so the pipeline can build a new
// engine on each (re)mount — e.g. dev hot-reload rebuilds the handler without
// re-registering routes onto an engine that already has them (which gin/echo
// would reject as duplicates).
type WebFrameworkFactory func() WebFramework
