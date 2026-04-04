package core

import "context"

// renderContextKey is an unexported type for render request context keys,
// ensuring no collisions with keys from other packages.
type renderContextKey int

const (
	routeCtxKey renderContextKey = iota
	propsCtxKey
	statusCtxKey
	injectorsCtxKey
	requestContextCtxKey
)

// WithRenderRequest attaches per-request render data to the context.
// The orchestrator calls this before handing the request to the renderer.
func WithRenderRequest(ctx context.Context, route *Route, props map[string]any, requestContext map[string]any, status int, injectors []RuntimeInjector) context.Context {
	ctx = context.WithValue(ctx, routeCtxKey, route)
	ctx = context.WithValue(ctx, propsCtxKey, props)
	ctx = context.WithValue(ctx, requestContextCtxKey, requestContext)
	ctx = context.WithValue(ctx, statusCtxKey, status)
	ctx = context.WithValue(ctx, injectorsCtxKey, injectors)
	return ctx
}

// RenderRequestFrom extracts per-request render data from the context.
func RenderRequestFrom(ctx context.Context) (route *Route, props map[string]any, requestContext map[string]any, status int, injectors []RuntimeInjector) {
	if r, ok := ctx.Value(routeCtxKey).(*Route); ok {
		route = r
	}
	if p, ok := ctx.Value(propsCtxKey).(map[string]any); ok {
		props = p
	}
	if rc, ok := ctx.Value(requestContextCtxKey).(map[string]any); ok {
		requestContext = rc
	}
	if s, ok := ctx.Value(statusCtxKey).(int); ok {
		status = s
	}
	if inj, ok := ctx.Value(injectorsCtxKey).([]RuntimeInjector); ok {
		injectors = inj
	}
	return
}
