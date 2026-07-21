package core

import (
	"errors"
	"net/http"
)

// StatusCoder is implemented by errors that carry an HTTP status code. When a
// route handler returns an error whose chain contains a StatusCoder, the web
// adapter responds with that status instead of the default 500 — so a handler
// turns a domain error into the right HTTP result simply by returning it.
type StatusCoder interface {
	StatusCode() int
}

// StatusError wraps an error with an HTTP status code. Construct it with
// [WithStatus]; it unwraps to the original error, so errors.Is/As still see the
// underlying cause.
type StatusError struct {
	Code int
	Err  error
}

// Error returns the wrapped error's message, or the status text when there is no
// wrapped error.
func (e *StatusError) Error() string {
	if e.Err == nil {
		return http.StatusText(e.Code)
	}
	return e.Err.Error()
}

// Unwrap exposes the wrapped error to errors.Is/As.
func (e *StatusError) Unwrap() error { return e.Err }

// StatusCode reports the HTTP status.
func (e *StatusError) StatusCode() int { return e.Code }

// WithStatus wraps err so the web adapter responds with the given HTTP status.
// It is the ORM/domain-neutral way to map an error to a response from a handler:
//
//	u, err := svc.Get(ctx, id)
//	if repository.IsNotFound(err) {
//	    return core.WithStatus(http.StatusNotFound, err)
//	}
func WithStatus(code int, err error) error {
	return &StatusError{Code: code, Err: err}
}

// StatusOf returns the HTTP status carried by err — the first [StatusCoder] in
// its chain — or http.StatusInternalServerError when it carries none. Web
// adapters use it to translate a returned handler error into a response status.
func StatusOf(err error) int {
	var sc StatusCoder
	if errors.As(err, &sc) {
		if code := sc.StatusCode(); code != 0 {
			return code
		}
	}
	return http.StatusInternalServerError
}
