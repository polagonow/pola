package routes

import "net/http"

type contextKey int

const paramsKey contextKey = iota

// Params returns the URL parameters extracted from the route match,
// or nil if no parameters were set.
func Params(r *http.Request) map[string]any {
	if p, ok := r.Context().Value(paramsKey).(map[string]any); ok {
		return p
	}
	return nil
}

// Param returns a single URL parameter by name as a string.
// Returns "" if the parameter doesn't exist or is not a string.
func Param(r *http.Request, name string) string {
	params := Params(r)
	if params == nil {
		return ""
	}
	if v, ok := params[name].(string); ok {
		return v
	}
	return ""
}
