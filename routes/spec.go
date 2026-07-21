package routes

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/polagonow/pola/core"
)

// RouteSpec is a single discovered API route: an HTTP method, a URL pattern in
// Pola's ":param" / ":...rest" syntax, and a framework-neutral handler. The
// pipeline registers each spec onto the selected core.WebFramework.
type RouteSpec struct {
	Method  string
	Pattern string
	Handler core.HandlerFunc
	// Middleware are per-route wrappers declared via the Middlewarer interface,
	// composed around Handler at registration (index 0 outermost).
	Middleware []core.RouteMiddleware
	// Meta is arbitrary route metadata declared via the Metaer interface, for
	// tooling (e.g. OpenAPI) and meta-aware middleware. Nil when none is set.
	Meta map[string]any
}

// RouteSpecs is a named slice so it can be stored in / resolved from the DI
// registry (discovery runs once; dev hot-reload replays the cached specs).
type RouteSpecs []RouteSpec

// memberMethods always target a specific resource, so they are mounted on the
// base pattern + "/:id".
var memberMethods = map[string]bool{"PUT": true, "PATCH": true, "DELETE": true}

// Discover drains all registered struct factories and function-based
// registrations, materializes struct handlers (resolving DI from the registry),
// derives RESTful URL patterns, and returns the full set of route specs.
//
// It is called once per app build. The result is cached in the registry by the
// pipeline so dev hot-reload can replay it without re-draining the (now empty)
// registration queues.
func Discover(registry *core.Registry) (RouteSpecs, error) {
	factories := drain()
	funcRegs := drainFuncs()
	if len(factories) == 0 && len(funcRegs) == 0 {
		return nil, nil
	}

	// Materialize struct handlers by calling factories with the registry.
	handlers := make([]any, len(factories))
	for i, f := range factories {
		handlers[i] = f(registry)
	}

	// First pass: collect every registered package path (struct + function) so
	// pattern derivation knows which parent segments are resources.
	registeredPkgs := make(map[string]bool, len(handlers)+len(funcRegs))
	for _, h := range handlers {
		if rel := relativePackagePath(reflect.TypeOf(h).Elem().PkgPath()); rel != "" {
			registeredPkgs[rel] = true
		}
	}
	for _, fr := range funcRegs {
		if rel := relativePackagePath(funcPkgPath(fr.fn)); rel != "" {
			registeredPkgs[rel] = true
		}
	}

	var specs RouteSpecs

	// Struct-based routes.
	for _, h := range handlers {
		base, err := basePatternFor(h, registeredPkgs)
		if err != nil {
			return nil, err
		}
		actions, err := discoverActions(h)
		if err != nil {
			return nil, err
		}
		specs = append(specs, splitActions(base, actions, routeMiddleware(h), routeMeta(h))...)
	}

	// Function-based routes.
	for _, fr := range funcRegs {
		base := pathToPattern(funcPkgPath(fr.fn), registeredPkgs, nil)
		h, err := adaptHandler(fr.fn, fmt.Sprintf("%s %s", fr.method, funcName(fr.fn)))
		if err != nil {
			return nil, err
		}
		specs = append(specs, splitActions(base, []discoveredAction{{Method: fr.method, Handler: h}}, nil, nil)...)
	}

	return specs, nil
}

// basePatternFor derives a struct route's base URL pattern, honoring the
// optional Pather and Memberer overrides.
func basePatternFor(h any, registeredPkgs map[string]bool) (string, error) {
	if p, ok := h.(Pather); ok {
		base := p.Path()
		if !strings.HasPrefix(base, "/") {
			return "", fmt.Errorf("routes: %s.Path() must return a path starting with '/', got %q",
				reflect.TypeOf(h).Elem().Name(), base)
		}
		return base, nil
	}
	var memberOverride *bool
	if m, ok := h.(Memberer); ok {
		v := m.Member()
		memberOverride = &v
	}
	return pathToPattern(reflect.TypeOf(h).Elem().PkgPath(), registeredPkgs, memberOverride), nil
}

// splitActions expands discovered actions into specs, mounting member methods
// (PUT/PATCH/DELETE) on base + "/:id" and the rest on the base pattern. The
// route's per-route middleware and metadata (mws, meta) are attached to every
// action's spec.
func splitActions(base string, actions []discoveredAction, mws []core.RouteMiddleware, meta map[string]any) RouteSpecs {
	specs := make(RouteSpecs, 0, len(actions))
	for _, a := range actions {
		pattern := base
		if memberMethods[a.Method] {
			pattern = strings.TrimSuffix(base, "/") + "/:id"
		}
		specs = append(specs, RouteSpec{
			Method:     a.Method,
			Pattern:    pattern,
			Handler:    a.Handler,
			Middleware: mws,
			Meta:       meta,
		})
	}
	return specs
}

// funcPkgPath returns the import path of fn's package via the runtime symbol
// table, e.g. "github.com/owner/app/routes/health.GET" → ".../routes/health".
func funcPkgPath(fn any) string {
	rv := reflect.ValueOf(fn)
	if rv.Kind() != reflect.Func {
		return ""
	}
	full := runtime.FuncForPC(rv.Pointer()).Name()
	if idx := strings.LastIndex(full, "."); idx != -1 {
		return full[:idx]
	}
	return full
}

// funcName returns fn's short symbol name for error messages.
func funcName(fn any) string {
	rv := reflect.ValueOf(fn)
	if rv.Kind() != reflect.Func {
		return "<non-func>"
	}
	full := runtime.FuncForPC(rv.Pointer()).Name()
	if idx := strings.LastIndex(full, "/"); idx != -1 {
		return full[idx+1:]
	}
	return full
}
