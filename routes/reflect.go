package routes

import (
	"fmt"
	"reflect"

	"github.com/polagonow/pola/core"
)

// Pather is optionally implemented by route structs to override the
// auto-derived URL pattern. The returned string must start with "/".
type Pather interface {
	Path() string
}

// Memberer is optionally implemented by route structs to override the
// auto-detected member/collection nesting behavior.
// Return true to force :id param insertion from parent segment.
// Return false to suppress it (flat path).
type Memberer interface {
	Member() bool
}

// Middlewarer is optionally implemented by route structs to attach per-route
// middleware to every action on the route. The middleware run after the route
// is matched, wrapping the handler in the returned order (index 0 outermost),
// and may short-circuit (e.g. an auth check answering 401).
type Middlewarer interface {
	Middleware() []core.RouteMiddleware
}

// Metaer is optionally implemented by route structs to attach arbitrary metadata
// to every action on the route (e.g. an auth flag or a rate-limit tier). The
// metadata rides on the discovered RouteSpec for tooling such as OpenAPI
// generation and meta-aware middleware.
type Metaer interface {
	Meta() map[string]any
}

// routeMiddleware returns the per-route middleware declared by h, or nil.
func routeMiddleware(h any) []core.RouteMiddleware {
	if m, ok := h.(Middlewarer); ok {
		return m.Middleware()
	}
	return nil
}

// routeMeta returns the metadata declared by h, or nil.
func routeMeta(h any) map[string]any {
	if m, ok := h.(Metaer); ok {
		return m.Meta()
	}
	return nil
}

// httpMethods lists the HTTP methods discovered on route structs / packages.
var httpMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "CONNECT", "TRACE"}

// discoveredAction holds a discovered HTTP method handler.
type discoveredAction struct {
	Method  string
	Handler core.HandlerFunc
}

// discoverActions inspects the route struct for methods named after HTTP verbs
// (GET, POST, PUT, …). Each must have the signature:
//
//	func(core.Context) error
func discoverActions(route any) ([]discoveredAction, error) {
	rv := reflect.ValueOf(route)
	rt := rv.Type()

	var result []discoveredAction
	for _, method := range httpMethods {
		if _, ok := rt.MethodByName(method); !ok {
			continue
		}
		fn := rv.MethodByName(method)
		h, err := adaptHandler(fn.Interface(), fmt.Sprintf("%s.%s", rt.Elem().Name(), method))
		if err != nil {
			return nil, err
		}
		result = append(result, discoveredAction{Method: method, Handler: h})
	}
	return result, nil
}

// adaptHandler converts a handler into a core.HandlerFunc, or returns an error
// describing the expected signature.
func adaptHandler(fn any, name string) (core.HandlerFunc, error) {
	switch h := fn.(type) {
	case core.HandlerFunc:
		return h, nil
	case func(core.Context) error:
		return h, nil
	default:
		return nil, fmt.Errorf("routes: %s must be func(core.Context) error, got %T", name, fn)
	}
}
