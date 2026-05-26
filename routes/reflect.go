package routes

import (
	"fmt"
	"net/http"
	"reflect"
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

// httpMethods lists the HTTP methods discovered on route structs.
var httpMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "CONNECT", "TRACE"}

// discoveredAction holds a discovered HTTP method handler on a route struct.
type discoveredAction struct {
	Method  string           // HTTP method (GET, POST, etc.)
	Handler http.HandlerFunc // bound method
}

// discoverActions inspects the route struct for methods named after HTTP verbs
// (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS, CONNECT, TRACE). Each must have the signature:
//
//	func(http.ResponseWriter, *http.Request)
func discoverActions(route any) ([]discoveredAction, error) {
	rv := reflect.ValueOf(route)
	rt := rv.Type()

	var result []discoveredAction

	for _, method := range httpMethods {
		m, ok := rt.MethodByName(method)
		if !ok {
			continue
		}

		// Validate signature: receiver + (http.ResponseWriter, *http.Request)
		mt := m.Type
		if mt.NumIn() != 3 || mt.NumOut() != 0 {
			return nil, fmt.Errorf("routes: %s.%s must have signature func(http.ResponseWriter, *http.Request), got %s",
				rt.Elem().Name(), method, mt)
		}

		writerType := reflect.TypeOf((*http.ResponseWriter)(nil)).Elem()
		requestType := reflect.TypeOf((*http.Request)(nil))
		if !mt.In(1).Implements(writerType) || mt.In(2) != requestType {
			return nil, fmt.Errorf("routes: %s.%s must have signature func(http.ResponseWriter, *http.Request), got %s",
				rt.Elem().Name(), method, mt)
		}

		fn := rv.MethodByName(method)
		handler := func(w http.ResponseWriter, r *http.Request) {
			fn.Call([]reflect.Value{reflect.ValueOf(w), reflect.ValueOf(r)})
		}

		result = append(result, discoveredAction{
			Method:  method,
			Handler: handler,
		})
	}

	return result, nil
}
