package auth

import "github.com/polagonow/pola/core"

// RouteMiddleware adapts an [Authenticator] into a per-route
// core.RouteMiddleware, so authentication can be attached to specific routes
// (via the routes package's Middlewarer interface) rather than the whole engine.
// On success it injects the user — read it in the handler with [UserFromContext]
// — and calls next; on failure it answers 401 unless [WithOptional] was set.
//
//	func (r *AdminRoutes) Middleware() []core.RouteMiddleware {
//	    return []core.RouteMiddleware{auth.RouteMiddleware(r.authenticator)}
//	}
func RouteMiddleware[T any](a Authenticator[T], opts ...Option) core.RouteMiddleware {
	c := config{unauthorized: defaultUnauthorized}
	for _, o := range opts {
		o(&c)
	}
	return func(next core.HandlerFunc) core.HandlerFunc {
		return func(ctx core.Context) error {
			u, err := a.Authenticate(ctx.Request())
			if err != nil {
				if c.optional {
					return next(ctx)
				}
				c.unauthorized(ctx.Writer(), ctx.Request(), err)
				return nil
			}
			ctx.SetContext(WithUser(ctx.Ctx(), u))
			return next(ctx)
		}
	}
}
