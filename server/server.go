package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	ghtml "gojsx/html"
	"gojsx/runtime"
)

// SSRStreaming controls whether the HTML path streams RSC via a second
// client fetch (true, default) or buffers and inlines flight data in the
// HTML response (false). Streaming supports Suspense; inlining saves a round-trip.
var SSRStreaming = os.Getenv("SSR_STREAMING") != "false"

// Route maps a URL pattern to a Server Component export name.
// Pattern supports dynamic segments: "/products/:id".
// Export is the JS key in __pages__ (e.g. "Products").
type Route struct {
	Pattern string
	Export  string
}

// App is the top-level application.
type App struct {
	Pool                 *runtime.VMPool
	Renderer             *runtime.Renderer
	Routes               []Route
	PublicDir            string
	Manifest             runtime.ClientManifest
	ImportURLs           map[string]string // moduleId → /public/chunk-HASH.js
	ClientEntryScript    string            // /public/client-[hash].js
	GlobalNotFoundExport string            // "GlobalNotFound" or ""
}

// MatchPattern matches a URL path against a route pattern.
// Dynamic segments (":name") capture into the returned map.
// Returns (params, true) on match, (nil, false) otherwise.
func MatchPattern(pattern, path string) (map[string]string, bool) {
	pp := strings.Split(pattern, "/")
	rp := strings.Split(path, "/")
	if len(pp) != len(rp) {
		return nil, false
	}
	params := map[string]string{}
	for i, seg := range pp {
		if strings.HasPrefix(seg, ":") {
			params[seg[1:]] = rp[i]
		} else if seg != rp[i] {
			return nil, false
		}
	}
	return params, true
}

// Match returns the first route whose pattern matches path, plus captured params.
func (a *App) Match(path string) (*Route, map[string]string) {
	for i := range a.Routes {
		if params, ok := MatchPattern(a.Routes[i].Pattern, path); ok {
			return &a.Routes[i], params
		}
	}
	return nil, nil
}

// BuildPageProps constructs the standard PageProps passed to every page component:
//
//	{ params: { <path segments> }, searchParams: { <query string> } }
func BuildPageProps(r *http.Request, params map[string]string) map[string]any {
	searchParams := map[string]any{}
	for k, vs := range r.URL.Query() {
		if len(vs) == 1 {
			searchParams[k] = vs[0]
		} else {
			searchParams[k] = vs
		}
	}
	p := map[string]any{}
	for k, v := range params {
		p[k] = v
	}
	return map[string]any{"params": p, "searchParams": searchParams}
}

// HandleRoute serves all page routes. When the request carries
// Content-Type: text/x-component it returns the RSC Flight stream;
// otherwise it returns the HTML shell that bootstraps the client.
func (a *App) HandleRoute(w http.ResponseWriter, r *http.Request) {
	route, params := a.Match(r.URL.Path)
	if route == nil {
		if a.GlobalNotFoundExport != "" {
			if r.Header.Get("Content-Type") == "text/x-component" {
				w.Header().Set("Content-Type", "text/x-component; charset=utf-8")
				w.WriteHeader(http.StatusNotFound)
				fw := runtime.NewFlightWriter(w)
				vm := a.Pool.Acquire()
				defer a.Pool.Release(vm)
				if err := a.Renderer.Render(fw, vm, runtime.RenderOptions{
					ExportName: a.GlobalNotFoundExport,
					Props:      map[string]any{},
				}); err != nil {
					fmt.Printf("rsc 404: %v\n", err)
				}
			} else {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, ghtml.Render(ghtml.Params{
					ImportURLs:   a.ImportURLs,
					ClientScript: a.ClientEntryScript,
				}))
			}
		} else {
			http.NotFound(w, r)
		}
		return
	}

	props := BuildPageProps(r, params)

	if r.Header.Get("Content-Type") == "text/x-component" {
		w.Header().Set("Content-Type", "text/x-component; charset=utf-8")
		fw := runtime.NewFlightWriter(w)
		vm := a.Pool.Acquire()
		defer a.Pool.Release(vm)
		if err := a.Renderer.Render(fw, vm, runtime.RenderOptions{
			ExportName:     route.Export,
			Props:          props,
			RequestContext: RequestCtx(r),
		}); err != nil {
			fmt.Printf("rsc %s: %v\n", r.URL.Path, err)
			fmt.Fprintf(w, `<div class="rsc-err">%s</div>`, err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	htmlParams := ghtml.Params{
		ImportURLs:   a.ImportURLs,
		ClientScript: a.ClientEntryScript,
	}

	if !SSRStreaming {
		var buf bytes.Buffer
		bfw := runtime.NewFlightWriterBuffer(&buf)
		vm := a.Pool.Acquire()
		defer a.Pool.Release(vm)
		if err := a.Renderer.Render(bfw, vm, runtime.RenderOptions{
			ExportName:     route.Export,
			Props:          props,
			RequestContext: RequestCtx(r),
		}); err != nil {
			fmt.Printf("html render %s: %v\n", r.URL.Path, err)
		}
		flightJSON, _ := json.Marshal(buf.String())
		htmlParams.Scripts = append(htmlParams.Scripts, "self.__flight_data="+string(flightJSON))
	}

	fmt.Fprint(w, ghtml.Render(htmlParams))
}

// RequestCtx extracts request metadata for the JS request context object.
func RequestCtx(r *http.Request) map[string]any {
	headers := make(map[string]string)
	for k, v := range r.Header {
		headers[k] = strings.Join(v, ", ")
	}
	return map[string]any{
		"url":     r.URL.String(),
		"path":    r.URL.Path,
		"query":   r.URL.RawQuery,
		"method":  r.Method,
		"headers": headers,
	}
}

// Flush flushes the response writer if it supports http.Flusher.
func Flush(w http.ResponseWriter) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
