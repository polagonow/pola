// Package openapi generates an OpenAPI 3 specification from pola's discovered API
// routes, and serves it (plus a SwaggerUI page) — giving external consumers a
// machine-readable contract that pola's typed @pola/actions bridge only provided
// to its own frontend.
//
// It derives paths, HTTP methods and path parameters from the routes, and reads
// optional per-route metadata (declared via the routes package's Metaer
// interface) for a human-readable summary and tags:
//
//	func (r *UserRoutes) Meta() map[string]any {
//	    return map[string]any{"summary": "Manage users", "tags": []string{"users"}}
//	}
//
// It does not infer request/response body schemas — pola routes don't yet
// declare their DTO types to the router — so the spec documents the API surface
// (endpoints, params) and is a foundation to enrich as routes gain schema
// metadata.
package openapi

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/polagonow/pola/routes"
)

// Document is a minimal OpenAPI 3 document — enough to describe pola's API
// surface and render in SwaggerUI.
type Document struct {
	OpenAPI string              `json:"openapi"`
	Info    Info                `json:"info"`
	Paths   map[string]PathItem `json:"paths"`
}

// Info is the spec's metadata block.
type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

// PathItem maps lowercase HTTP methods to their operation.
type PathItem map[string]Operation

// Operation describes one method on one path.
type Operation struct {
	Summary    string              `json:"summary,omitempty"`
	Tags       []string            `json:"tags,omitempty"`
	Parameters []Parameter         `json:"parameters,omitempty"`
	Responses  map[string]Response `json:"responses"`
}

// Parameter is a path parameter (the only kind derivable from the pattern).
type Parameter struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Required bool   `json:"required"`
	Schema   Schema `json:"schema"`
}

// Schema is a minimal JSON-schema node.
type Schema struct {
	Type string `json:"type"`
}

// Response is a single response entry.
type Response struct {
	Description string `json:"description"`
}

// Generate builds an OpenAPI 3 document from discovered route specs. Obtain the
// specs with routes.Discover(registry).
func Generate(specs routes.RouteSpecs, info Info) *Document {
	doc := &Document{OpenAPI: "3.0.3", Info: info, Paths: map[string]PathItem{}}
	for _, s := range specs {
		path, params := convertPath(s.Pattern)
		item, ok := doc.Paths[path]
		if !ok {
			item = PathItem{}
			doc.Paths[path] = item
		}

		op := Operation{Responses: map[string]Response{"200": {Description: "OK"}}}
		for _, p := range params {
			op.Parameters = append(op.Parameters, Parameter{
				Name: p, In: "path", Required: true, Schema: Schema{Type: "string"},
			})
		}
		applyMeta(&op, s.Meta)
		item[strings.ToLower(s.Method)] = op
	}
	return doc
}

// applyMeta pulls an optional summary and tags from a route's metadata.
func applyMeta(op *Operation, meta map[string]any) {
	if meta == nil {
		return
	}
	if v, ok := meta["summary"].(string); ok {
		op.Summary = v
	}
	if v, ok := meta["tags"].([]string); ok {
		op.Tags = v
	}
}

// convertPath rewrites a pola pattern (":id", ":...rest") into OpenAPI path
// syntax ("{id}", "{rest}") and returns the parameter names in order.
func convertPath(pattern string) (string, []string) {
	segs := strings.Split(pattern, "/")
	var params []string
	for i, seg := range segs {
		if strings.HasPrefix(seg, ":") {
			name := strings.TrimPrefix(strings.TrimPrefix(seg, ":"), "...")
			params = append(params, name)
			segs[i] = "{" + name + "}"
		}
	}
	return strings.Join(segs, "/"), params
}

// JSON renders the document as indented JSON.
func (d *Document) JSON() ([]byte, error) {
	return json.MarshalIndent(d, "", "  ")
}

// SortedPaths returns the document's paths in stable, sorted order — handy for
// deterministic output and tests.
func (d *Document) SortedPaths() []string {
	out := make([]string, 0, len(d.Paths))
	for p := range d.Paths {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
