// Package request provides request-time helpers for reading typed values out of
// a core.Context (path params, query params, multipart bodies). They are free
// functions over the framework-neutral core.Context, so they work unchanged
// across every WebFramework adapter (std/gin/echo/chi).
package request

import (
	"fmt"
	"strconv"

	"github.com/polagonow/pola/core"
)

// PathParam returns the named path parameter as a string ("" if absent).
func PathParam(c core.Context, name string) string {
	return c.Param(name)
}

// PathParamInt parses the named path parameter as an int.
func PathParamInt(c core.Context, name string) (int, error) {
	v := c.Param(name)
	if v == "" {
		return 0, fmt.Errorf("missing %s", name)
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s", name)
	}
	return n, nil
}

// PathParamUint parses the named path parameter as a uint.
func PathParamUint(c core.Context, name string) (uint, error) {
	v := c.Param(name)
	if v == "" {
		return 0, fmt.Errorf("missing %s", name)
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s", name)
	}
	return uint(n), nil
}

// QueryParam returns the named query parameter as a string ("" if absent).
func QueryParam(c core.Context, name string) string {
	return c.Query(name)
}

// QueryParamInt reads an integer query parameter, returning fallback when the
// value is missing, non-numeric, or less than 1.
func QueryParamInt(c core.Context, name string, fallback int) int {
	v := c.Query(name)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}
