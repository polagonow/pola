package std

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"

	"github.com/polagonow/pola/core"
)

// defaultMaxMemory bounds in-memory multipart parsing (matches net/http).
const defaultMaxMemory = 32 << 20

var _ core.Context = (*stdContext)(nil)

// stdContext implements core.Context over a plain http.ResponseWriter and
// *http.Request. Route params are carried on the request context (see
// core.WithParams) so legacy net/http handlers see them too.
type stdContext struct {
	w *statusWriter
	r *http.Request
}

func newContext(w http.ResponseWriter, r *http.Request) *stdContext {
	return &stdContext{w: &statusWriter{ResponseWriter: w, code: http.StatusOK}, r: r}
}

func (c *stdContext) Request() *http.Request      { return c.r }
func (c *stdContext) Writer() http.ResponseWriter { return c.w }
func (c *stdContext) Ctx() context.Context        { return c.r.Context() }
func (c *stdContext) Status() int                 { return c.w.code }

func (c *stdContext) Param(name string) string { return core.Param(c.r, name) }

func (c *stdContext) Query(name string) string  { return c.r.URL.Query().Get(name) }
func (c *stdContext) Header(name string) string { return c.r.Header.Get(name) }

func (c *stdContext) ShouldBind(v any) error {
	if err := json.NewDecoder(c.r.Body).Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

func (c *stdContext) ShouldBindUri(v any) error    { return core.ShouldBindUri(c, v) }
func (c *stdContext) ShouldBindHeader(v any) error { return core.ShouldBindHeader(c, v) }
func (c *stdContext) ShouldBindQuery(v any) error  { return core.ShouldBindQuery(c, v) }
func (c *stdContext) ShouldBindForm(v any) error   { return core.ShouldBindForm(c, v) }

func (c *stdContext) Bind(v any) error       { return core.MustBind(c, c.ShouldBind(v)) }
func (c *stdContext) BindUri(v any) error    { return core.MustBind(c, c.ShouldBindUri(v)) }
func (c *stdContext) BindHeader(v any) error { return core.MustBind(c, c.ShouldBindHeader(v)) }
func (c *stdContext) BindQuery(v any) error  { return core.MustBind(c, c.ShouldBindQuery(v)) }
func (c *stdContext) BindForm(v any) error   { return core.MustBind(c, c.ShouldBindForm(v)) }

func (c *stdContext) FormValue(name string) string { return c.r.FormValue(name) }

func (c *stdContext) FormFile(name string) (*multipart.FileHeader, error) {
	_, fh, err := c.r.FormFile(name)
	return fh, err
}

func (c *stdContext) MultipartForm() (*multipart.Form, error) {
	if err := c.r.ParseMultipartForm(defaultMaxMemory); err != nil {
		return nil, err
	}
	return c.r.MultipartForm, nil
}

func (c *stdContext) JSON(status int, v any) error {
	c.w.Header().Set("Content-Type", "application/json")
	c.w.WriteHeader(status)
	return json.NewEncoder(c.w).Encode(v)
}

func (c *stdContext) String(status int, s string) error {
	c.w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.w.WriteHeader(status)
	_, err := c.w.Write([]byte(s))
	return err
}

func (c *stdContext) NoContent(status int) error {
	c.w.WriteHeader(status)
	return nil
}

func (c *stdContext) SetHeader(name, value string) { c.w.Header().Set(name, value) }

func (c *stdContext) Redirect(status int, url string) error {
	http.Redirect(c.w, c.r, url, status)
	return nil
}

// statusWriter records the response status code and whether anything was written
// so the dispatcher can decide whether to emit a 500 for a returned error.
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
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
