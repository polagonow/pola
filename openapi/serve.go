package openapi

import (
	"net/http"

	"github.com/polagonow/pola/core"
)

type serveOptions struct {
	specPath string
	uiPath   string
}

// ServeOption configures the [Serve] middleware.
type ServeOption func(*serveOptions)

// WithSpecPath overrides the JSON spec path (default "/openapi.json").
func WithSpecPath(p string) ServeOption { return func(o *serveOptions) { o.specPath = p } }

// WithUIPath overrides the SwaggerUI page path (default "/openapi").
func WithUIPath(p string) ServeOption { return func(o *serveOptions) { o.uiPath = p } }

// Serve returns a middleware that serves the document as JSON at specPath
// (default "/openapi.json") and a SwaggerUI page at uiPath (default "/openapi").
// Like the health middleware it short-circuits its own paths and passes
// everything else through, so it needs no pipeline wiring:
//
//	specs, _ := routes.Discover(reg)
//	doc := openapi.Generate(specs, openapi.Info{Title: "Shop API", Version: "1.0.0"})
//	reg.AddMiddleware(openapi.Serve(doc))
func Serve(doc *Document, opts ...ServeOption) core.Middleware {
	o := serveOptions{specPath: "/openapi.json", uiPath: "/openapi"}
	for _, f := range opts {
		f(&o)
	}
	spec, _ := doc.JSON()
	return &serveMW{spec: spec, opts: o}
}

type serveMW struct {
	spec []byte
	opts serveOptions
}

func (m *serveMW) Name() string { return "openapi" }

func (m *serveMW) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			switch r.URL.Path {
			case m.opts.specPath:
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(m.spec)
				return
			case m.opts.uiPath:
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write([]byte(swaggerHTML(m.opts.specPath)))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// swaggerHTML is a minimal SwaggerUI page that loads the dist assets from a CDN
// and points them at the served spec.
func swaggerHTML(specURL string) string {
	return `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>API Docs</title>
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist/swagger-ui.css">
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist/swagger-ui-bundle.js"></script>
<script>window.onload = () => { SwaggerUIBundle({ url: "` + specURL + `", dom_id: "#swagger-ui" }); };</script>
</body>
</html>`
}
