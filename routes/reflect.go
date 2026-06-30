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
