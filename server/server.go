package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"

	"gojsx/framework/contract"
	renderreact "gojsx/render/react"
	ghtml "gojsx/render/react/shell"
	vmgoja "gojsx/vm/goja"
)

// SSRStreaming controls whether the HTML path streams RSC via a second
// client fetch (true, default) or buffers and inlines flight data in the
// HTML response (false). Streaming supports Suspense; inlining saves a round-trip.
var SSRStreaming = os.Getenv("SSR_STREAMING") != "false"

// Route maps a URL pattern to a Server Component export name.
// Pattern supports dynamic segments ("/products/:id"), catch-all ("/shop/:...path"),
// and optional catch-all ("/docs/:...slug?").
// Export is the JS key in __pages__ (e.g. "Products").
type Route = contract.Route

// App is the top-level application.
type App struct {
	Pool                 *vmgoja.VMPool
	Renderer             *renderreact.Renderer
	Routes               []Route
	PublicDir            string
	Manifest             renderreact.ClientManifest
	ImportURLs           map[string]string // moduleId → /public/chunk-HASH.js
	ClientEntryScript    string            // /public/client-[hash].js
	GlobalNotFoundExport string            // "GlobalNotFound" or ""

	sortOnce sync.Once
}

// routeScore returns a specificity score for a pattern: higher = tried first.
// Static segments score +1, single params score 0, catch-all -10, optional catch-all -20.
func routeScore(pattern string) int {
	score := 0
	for _, seg := range strings.Split(pattern, "/") {
		switch {
		case strings.HasPrefix(seg, ":...") && strings.HasSuffix(seg, "?"):
			score -= 20
		case strings.HasPrefix(seg, ":..."):
			score -= 10
		case strings.HasPrefix(seg, ":"):
			// dynamic single param: neutral
		default:
			score++
		}
	}
	return score
}

// MatchPattern matches a URL path against a route pattern.
// Supports:
//   - Static segments: exact match
//   - Dynamic segments (":name"): capture single segment as string
//   - Catch-all (":...name"): capture one or more remaining segments as []string
//   - Optional catch-all (":...name?"): capture zero or more remaining segments as []string
//     (key is absent from params when zero segments are matched)
//
// Returns (params, true) on match, (nil, false) otherwise.
func MatchPattern(pattern, path string) (map[string]any, bool) {
	pp := strings.Split(pattern, "/")
	rp := strings.Split(path, "/")
	params := map[string]any{}

	for i, seg := range pp {
		if strings.HasPrefix(seg, ":...") {
			// Catch-all or optional catch-all — must be last pattern segment.
			optional := strings.HasSuffix(seg, "?")
			name := seg[4:]
			if optional {
				name = name[:len(name)-1]
			}
			remaining := rp[i:]
			if len(remaining) == 0 || (len(remaining) == 1 && remaining[0] == "") {
				if !optional {
					return nil, false
				}
				// optional with zero segments: omit key (JS sees undefined)
				return params, true
			}
			params[name] = remaining
			return params, true
		}
		if i >= len(rp) {
			return nil, false
		}
		if strings.HasPrefix(seg, ":") {
			params[seg[1:]] = rp[i]
		} else if seg != rp[i] {
			return nil, false
		}
	}
	// For non-catch-all patterns, segment counts must match exactly.
	if len(pp) != len(rp) {
		return nil, false
	}
	return params, true
}

// Match returns the first route whose pattern matches path, plus captured params.
// Routes are sorted by specificity on the first call (most specific first).
func (a *App) Match(path string) (*Route, map[string]any) {
	a.sortOnce.Do(func() {
		sort.SliceStable(a.Routes, func(i, j int) bool {
			return routeScore(a.Routes[i].Pattern) > routeScore(a.Routes[j].Pattern)
		})
	})
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
//
// Param values are string for regular dynamic segments, []string for catch-all segments.
func BuildPageProps(r *http.Request, params map[string]any) map[string]any {
	searchParams := map[string]any{}
	for k, vs := range r.URL.Query() {
		if len(vs) == 1 {
			searchParams[k] = vs[0]
		} else {
			searchParams[k] = vs
		}
	}
	return map[string]any{"params": params, "searchParams": searchParams}
}

// HandleRoute serves all page routes. When the request carries
// Content-Type: text/x-component it returns the RSC Flight stream;
// otherwise it returns the HTML shell that bootstraps the client.
func (a *App) HandleRoute(w http.ResponseWriter, r *http.Request) {
	route, matchedParams := a.Match(r.URL.Path)
	if route == nil {
		if a.GlobalNotFoundExport != "" {
			if r.Header.Get("Content-Type") == "text/x-component" {
				w.Header().Set("Content-Type", "text/x-component; charset=utf-8")
				w.WriteHeader(http.StatusNotFound)
				fw := renderreact.NewFlightWriter(w)
				vm := a.Pool.Acquire()
				defer a.Pool.Release(vm)
				if err := a.Renderer.Render(fw, vm, renderreact.RenderOptions{
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

	props := BuildPageProps(r, matchedParams)

	if r.Header.Get("Content-Type") == "text/x-component" {
		w.Header().Set("Content-Type", "text/x-component; charset=utf-8")
		fw := renderreact.NewFlightWriter(w)
		vm := a.Pool.Acquire()
		defer a.Pool.Release(vm)
		if err := a.Renderer.Render(fw, vm, renderreact.RenderOptions{
			ExportName:     route.Export,
			Props:          props,
			RequestContext: RequestCtx(r),
			Bridge:         route.Bridge,
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
		bfw := renderreact.NewFlightWriterBuffer(&buf)
		vm := a.Pool.Acquire()
		defer a.Pool.Release(vm)
		if err := a.Renderer.Render(bfw, vm, renderreact.RenderOptions{
			ExportName:     route.Export,
			Props:          props,
			RequestContext: RequestCtx(r),
			Bridge:         route.Bridge,
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
