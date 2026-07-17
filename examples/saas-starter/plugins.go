package main

import (
	"net/http"
	"os"
	"strings"

	"github.com/polagonow/pola"
	authjwt "github.com/polagonow/pola/auth/jwt"
	"github.com/polagonow/pola/core"

	"saas-starter/lib/reqctx"
)

// This hand-written file augments the framework-generated plugin wiring:
//   - reqctx.Plugin() records the client IP for activity logging.
//   - authGuardPlugin() protects /dashboard, the global-middleware equivalent of
//     the Next.js starter's middleware.ts. It verifies the JWT session cookie
//     directly (so it doesn't depend on middleware ordering) and redirects
//     unauthenticated visitors to /sign-in. The /api/* routes intentionally are
//     NOT guarded here — they return null when unauthenticated so client SWR
//     hooks work.
func init() {
	pola.Use(reqctx.Plugin())
	pola.Use(authGuardPlugin())
}

type authGuard struct {
	secret []byte
}

func (authGuard) Name() string { return "auth-guard" }

func (g authGuard) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isProtected(r.URL.Path) && !g.authenticated(r) {
			http.Redirect(w, r, "/sign-in", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isProtected(path string) bool {
	return strings.HasPrefix(path, "/dashboard")
}

func (g authGuard) authenticated(r *http.Request) bool {
	c, err := r.Cookie("session")
	if err != nil || c.Value == "" {
		return false
	}
	_, err = authjwt.Verify(c.Value, g.secret)
	return err == nil
}

func authGuardPlugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "auth-guard",
		Fn: func(reg *core.Registry) {
			reg.AddMiddleware(authGuard{secret: []byte(os.Getenv("AUTH_SECRET"))})
		},
	}
}
