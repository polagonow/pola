package core

import (
	"context"
	"net/http"
)

// paramsKeyType is an unexported context key type for route params.
type paramsKeyType struct{}

var paramsKey paramsKeyType

// WithParams returns a new request with route params stored in its context.
// WebFramework adapters call this so the framework-neutral Context.Param can
// resolve path parameters uniformly across adapters.
func WithParams(r *http.Request, params map[string]any) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), paramsKey, params))
}

// Params returns the route params stored on the request, or nil.
func Params(r *http.Request) map[string]any {
	if p, ok := r.Context().Value(paramsKey).(map[string]any); ok {
		return p
	}
	return nil
}

// Param returns a single route param by name as a string, or "".
func Param(r *http.Request, name string) string {
	p := Params(r)
	if p == nil {
		return ""
	}
	if v, ok := p[name].(string); ok {
		return v
	}
	return ""
}
