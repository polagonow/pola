package htmx

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/htmlwriter"
	"github.com/polagonow/pola/shell"
)

type Renderer struct {
	deps atomic.Pointer[core.RenderDeps]

	mu        sync.RWMutex
	templates map[string]*template.Template
}

func New() *Renderer {
	return &Renderer{
		templates: make(map[string]*template.Template),
	}
}

func (r *Renderer) Name() string             { return "htmx" }
func (r *Renderer) FileExtensions() []string { return []string{".html", ".templ"} }
func (r *Renderer) Capabilities() []core.Capability {
	return []core.Capability{"server-side", "htmx"}
}

func (r *Renderer) SetRenderDeps(deps core.RenderDeps) { r.deps.Store(&deps) }

func (r *Renderer) loadDeps() core.RenderDeps {
	if d := r.deps.Load(); d != nil {
		return *d
	}
	return core.RenderDeps{}
}

func (r *Renderer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	deps := r.loadDeps()
	route, props, _, status, _ := core.RenderRequestFrom(req.Context())

	if route == nil {
		if status == 0 {
			status = http.StatusNotFound
		}
		if deps.NotFoundRoute != nil {
			route = deps.NotFoundRoute
			if props == nil {
				props = map[string]any{"params": map[string]any{}, "searchParams": map[string]any{}}
			}
		} else {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(status)
			r.serveFullPage(w, req, "<h1>404 — Not Found</h1>", &deps)
			return
		}
	}

	if status == 0 {
		status = http.StatusOK
	}

	content, err := r.renderTemplate(route, props)
	if err != nil {
		if deps.Logger != nil {
			deps.Logger.Error("pola: htmx render", "route", route.Pattern, "err", err)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "<h1>Internal Server Error</h1>")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if isHTMXRequest(req) {
		w.Header().Set("Vary", "HX-Request")
		w.WriteHeader(status)
		io.WriteString(w, content)
		return
	}

	w.WriteHeader(status)
	r.serveFullPage(w, req, content, &deps)
}

func isHTMXRequest(req *http.Request) bool {
	return req.Header.Get("HX-Request") == "true"
}

func (r *Renderer) renderTemplate(route *core.Route, props map[string]any) (string, error) {
	tmpl, err := r.loadTemplate(route.Export)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, props); err != nil {
		return "", fmt.Errorf("execute template %q: %w", route.Export, err)
	}
	return buf.String(), nil
}

func (r *Renderer) loadTemplate(path string) (*template.Template, error) {
	r.mu.RLock()
	if tmpl, ok := r.templates[path]; ok {
		r.mu.RUnlock()
		return tmpl, nil
	}
	r.mu.RUnlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read template %q: %w", path, err)
	}

	name := filepath.Base(path)
	tmpl, err := template.New(name).Funcs(htmxFuncMap()).Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse template %q: %w", path, err)
	}

	r.mu.Lock()
	r.templates[path] = tmpl
	r.mu.Unlock()
	return tmpl, nil
}

// InvalidateTemplates clears the template cache, called by hot-reload.
func (r *Renderer) InvalidateTemplates() {
	r.mu.Lock()
	r.templates = make(map[string]*template.Template)
	r.mu.Unlock()
}

func (r *Renderer) serveFullPage(w http.ResponseWriter, req *http.Request, content string, deps *core.RenderDeps) {
	params := core.ShellParams{
		Metadata:      defaultMetadata(),
		DocumentProps: deps.DocumentProps,
	}
	if len(deps.CSSURLs) > 0 {
		params.Stylesheets = deps.CSSURLs
	}
	if deps.DevScript != "" {
		params.Scripts = append(params.Scripts, deps.DevScript)
	}
	if nonce, ok := req.Context().Value(core.NonceContextKey).(string); ok && nonce != "" {
		params.Nonce = nonce
	}
	if token, ok := req.Context().Value(core.CSRFTokenContextKey).(string); ok && token != "" {
		if params.Metadata == nil {
			params.Metadata = &core.Metadata{}
		}
		if params.Metadata.Other == nil {
			params.Metadata.Other = make(map[string]string)
		}
		params.Metadata.Other["csrf-param"] = "authenticity_token"
		params.Metadata.Other["csrf-token"] = token
	}

	// Include the HTMX library
	params.Scripts = append(params.Scripts, htmxCDNScript)

	var html string
	if deps.Shell != nil {
		html = deps.Shell.Render(params)
	} else {
		html = "<!DOCTYPE html><html><head></head><body></body></html>"
	}

	// Inject the rendered content into the body.
	// The shell renders with empty innerHTML for HTMX — inject content before </body>.
	if idx := strings.LastIndex(html, "</body>"); idx >= 0 {
		html = html[:idx] + content + html[idx:]
	}

	io.WriteString(w, html)
}

func defaultMetadata() *core.Metadata {
	faviconURL := "/public/favicon.ico"
	return &core.Metadata{
		Title: core.Title{Default: "Pola"},
		Icons: &core.Icons{Icon: []core.Icon{{URL: faviconURL}}},
	}
}

func htmxFuncMap() template.FuncMap {
	return template.FuncMap{
		"attr": func(name string, value any) template.HTMLAttr {
			var buf bytes.Buffer
			hw := htmlwriter.New(&buf)
			hw.WriteAttr(name, value)
			return template.HTMLAttr(buf.String())
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"class": func(items ...any) template.HTMLAttr {
			var buf bytes.Buffer
			hw := htmlwriter.New(&buf)
			hw.WriteAttr("class", items)
			return template.HTMLAttr(buf.String())
		},
	}
}

const htmxCDNScript = `(function(){if(!window.htmx){var s=document.createElement("script");s.src="https://unpkg.com/htmx.org@2.0.4";document.head.appendChild(s)}})()`

// Plugin returns the HTMX renderer plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "htmx",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.Renderer](r, New())
			core.ProvideValue[core.HTMLShell](r, shell.New(""))
		},
	}
}

// Ensure Renderer satisfies the interfaces.
var (
	_ core.Renderer       = (*Renderer)(nil)
	_ core.RenderDepsAware = (*Renderer)(nil)
)
