package nextjs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/fs/osfs"
)

// writeFile creates a file at dir/name with the given content.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

var defaultExts = []string{".tsx", ".jsx"}

func testFS() core.FS {
	return osfs.New(".")
}

// ── ScanRoutes ───────────────────────────────────────────────────────────────

func TestScanRoutes_Basic(t *testing.T) {
	appDir := t.TempDir()
	pagesDir := filepath.Join(appDir, "app")

	for _, route := range []string{"products", "about", "user"} {
		dir := filepath.Join(pagesDir, route)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, dir, "page.tsx", "export default function Page() { return null; }\n")
	}
	// Root index page
	if err := os.MkdirAll(pagesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, pagesDir, "page.tsx", "export default function Page() { return null; }\n")

	r := New()
	routes, err := r.ScanRoutes(context.Background(), testFS(), appDir, defaultExts)
	if err != nil {
		t.Fatalf("ScanRoutes: %v", err)
	}
	if len(routes) != 4 {
		t.Fatalf("expected 4 routes, got %d: %v", len(routes), routes)
	}

	patterns := make(map[string]bool)
	for _, rt := range routes {
		patterns[rt.Pattern] = true
	}
	for _, want := range []string{"/", "/products", "/about", "/user"} {
		if !patterns[want] {
			t.Errorf("route %q not discovered; got: %v", want, patterns)
		}
	}
}

func TestScanRoutes_DynamicSegments(t *testing.T) {
	appDir := t.TempDir()
	cases := []struct {
		rel     string
		pattern string
		export  string
	}{
		{filepath.Join("posts", "[slug]"), "/posts/:slug", "PostsSlug"},
		{filepath.Join("shop", "[...path]"), "/shop/:...path", "ShopPath"},
		{filepath.Join("docs", "[[...slug]]"), "/docs/:...slug?", "DocsSlug"},
	}
	for _, tc := range cases {
		dir := filepath.Join(appDir, "app", tc.rel)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, dir, "page.tsx", "export default function Page() { return null; }\n")
	}

	r := New()
	routes, err := r.ScanRoutes(context.Background(), testFS(), appDir, defaultExts)
	if err != nil {
		t.Fatalf("ScanRoutes: %v", err)
	}

	byPattern := make(map[string]string) // pattern → export
	for _, rt := range routes {
		byPattern[rt.Pattern] = rt.Export
	}
	for _, tc := range cases {
		got, ok := byPattern[tc.pattern]
		if !ok {
			t.Errorf("pattern %q not found; got %v", tc.pattern, byPattern)
			continue
		}
		if got != tc.export {
			t.Errorf("pattern %q: export=%q want %q", tc.pattern, got, tc.export)
		}
	}
}

func TestScanRoutes_RouteGroups(t *testing.T) {
	appDir := t.TempDir()
	cases := []struct {
		rel     string
		pattern string
		export  string
	}{
		{filepath.Join("(marketing)", "about"), "/about", "About"},
		{filepath.Join("(shop)", "products"), "/products", "Products"},
		{filepath.Join("(shop)", "products", "[id]"), "/products/:id", "ProductsId"},
		{filepath.Join("(a)", "(b)", "docs"), "/docs", "Docs"},
	}
	for _, tc := range cases {
		dir := filepath.Join(appDir, "app", tc.rel)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, dir, "page.tsx", "export default function Page() { return null; }\n")
	}

	r := New()
	routes, err := r.ScanRoutes(context.Background(), testFS(), appDir, defaultExts)
	if err != nil {
		t.Fatalf("ScanRoutes: %v", err)
	}

	byPattern := make(map[string]string)
	for _, rt := range routes {
		byPattern[rt.Pattern] = rt.Export
	}
	for _, tc := range cases {
		got, ok := byPattern[tc.pattern]
		if !ok {
			t.Errorf("pattern %q not found; patterns: %v", tc.pattern, byPattern)
			continue
		}
		if got != tc.export {
			t.Errorf("pattern %q: export=%q want %q", tc.pattern, got, tc.export)
		}
	}
}

func TestScanRoutes_MissingDefaultExport(t *testing.T) {
	appDir := t.TempDir()
	dir := filepath.Join(appDir, "app", "bad")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "page.tsx", "export function BadPage() { return null; }\n")

	r := New()
	_, err := r.ScanRoutes(context.Background(), testFS(), appDir, defaultExts)
	if err == nil {
		t.Fatal("expected error for missing default export, got nil")
	}
}

func TestScanRoutes_MultipleExtensions(t *testing.T) {
	appDir := t.TempDir()
	dirs := []struct {
		path string
		ext  string
	}{
		{filepath.Join("app", "tsx-route"), ".tsx"},
		{filepath.Join("app", "jsx-route"), ".jsx"},
	}
	for _, d := range dirs {
		dir := filepath.Join(appDir, d.path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, dir, "page"+d.ext, "export default function Page() { return null; }\n")
	}

	r := New()
	routes, err := r.ScanRoutes(context.Background(), testFS(), appDir, []string{".tsx", ".jsx"})
	if err != nil {
		t.Fatalf("ScanRoutes: %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d: %v", len(routes), routes)
	}
}

// ── Resolve ──────────────────────────────────────────────────────────────────

func TestResolve_Static(t *testing.T) {
	appDir := t.TempDir()
	pagesDir := filepath.Join(appDir, "app")
	dirs := []string{"about", "contact"}
	for _, d := range dirs {
		dir := filepath.Join(pagesDir, d)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, dir, "page.tsx", "export default function Page() { return null; }\n")
	}

	r := New()
	if _, err := r.ScanRoutes(context.Background(), testFS(), appDir, defaultExts); err != nil {
		t.Fatalf("ScanRoutes: %v", err)
	}

	route, params := r.Resolve(context.Background(), "/about")
	if route == nil {
		t.Fatal("expected route for /about")
	}
	if route.Pattern != "/about" {
		t.Errorf("pattern=%q want /about", route.Pattern)
	}
	if len(params) != 0 {
		t.Errorf("expected no params, got %v", params)
	}

	route, _ = r.Resolve(context.Background(), "/nonexistent")
	if route != nil {
		t.Error("expected nil route for /nonexistent")
	}
}

func TestResolve_Dynamic(t *testing.T) {
	appDir := t.TempDir()
	dir := filepath.Join(appDir, "app", "posts", "[slug]")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "page.tsx", "export default function Page() { return null; }\n")

	r := New()
	if _, err := r.ScanRoutes(context.Background(), testFS(), appDir, defaultExts); err != nil {
		t.Fatalf("ScanRoutes: %v", err)
	}

	route, params := r.Resolve(context.Background(), "/posts/hello-world")
	if route == nil {
		t.Fatal("expected route for /posts/hello-world")
	}
	if params["slug"] != "hello-world" {
		t.Errorf("slug=%v want hello-world", params["slug"])
	}
}

func TestResolve_Sorting(t *testing.T) {
	// Ensure static routes beat dynamic ones.
	appDir := t.TempDir()
	staticDir := filepath.Join(appDir, "app", "posts", "featured")
	dynamicDir := filepath.Join(appDir, "app", "posts", "[slug]")
	for _, d := range []string{staticDir, dynamicDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, d, "page.tsx", "export default function Page() { return null; }\n")
	}

	r := New()
	if _, err := r.ScanRoutes(context.Background(), testFS(), appDir, defaultExts); err != nil {
		t.Fatalf("ScanRoutes: %v", err)
	}

	route, _ := r.Resolve(context.Background(), "/posts/featured")
	if route == nil {
		t.Fatal("expected match for /posts/featured")
	}
	if route.Export != "PostsFeatured" {
		t.Errorf("expected static route PostsFeatured, got %s", route.Export)
	}
}

// ── DiscoveryResult ──────────────────────────────────────────────────────────

func TestDiscoveryResult_ClientComponents(t *testing.T) {
	appDir := t.TempDir()
	pagesDir := filepath.Join(appDir, "app")
	compDir := filepath.Join(appDir, "components")
	for _, d := range []string{pagesDir, compDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, pagesDir, "page.tsx", "export default function Page() { return null; }\n")
	writeFile(t, compDir, "Counter.tsx", `"use client"`+"\nexport default function Counter() {}\n")
	writeFile(t, compDir, "Header.tsx", "export default function Header() {}\n")

	r := New()
	if _, err := r.ScanRoutes(context.Background(), testFS(), appDir, defaultExts); err != nil {
		t.Fatalf("ScanRoutes: %v", err)
	}

	disc := r.DiscoveryResult()
	if len(disc.ClientComponents) != 1 {
		t.Fatalf("expected 1 client component, got %d: %v", len(disc.ClientComponents), disc.ClientComponents)
	}
	if filepath.Base(disc.ClientComponents[0]) != "Counter.tsx" {
		t.Errorf("expected Counter.tsx, got %s", disc.ClientComponents[0])
	}
}

func TestDiscoveryResult_Pages(t *testing.T) {
	appDir := t.TempDir()
	dir := filepath.Join(appDir, "app", "blog")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "page.tsx", "export default function Page() { return null; }\n")
	writeFile(t, dir, "layout.tsx", "export default function Layout({children}: any) { return children; }\n")

	r := New()
	if _, err := r.ScanRoutes(context.Background(), testFS(), appDir, defaultExts); err != nil {
		t.Fatalf("ScanRoutes: %v", err)
	}

	disc := r.DiscoveryResult()
	if len(disc.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(disc.Pages))
	}
	if len(disc.Pages[0].Segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(disc.Pages[0].Segments))
	}
	if filepath.Base(disc.Pages[0].Segments[0].LayoutPath) != "layout.tsx" {
		t.Errorf("unexpected layout: %s", disc.Pages[0].Segments[0].LayoutPath)
	}
}

// ── hasUseClient ──────────────────────────────────────────────────────────────

func TestHasUseClient(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name    string
		content string
		want    bool
	}{
		{"double-quoted directive", `"use client"` + "\nexport default function C() {}", true},
		{"single-quoted directive", `'use client'` + "\nexport default function C() {}", true},
		{"no directive", "export default function C() {}", false},
		{"blank lines then directive", "\n\n\"use client\"\nexport default function C() {}", true},
		{"with semicolon", `"use client";` + "\nexport default function C() {}", true},
		{"use server is not use client", `"use server"` + "\nexport default function C() {}", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeFile(t, dir, tc.name+".tsx", tc.content)
			got, err := hasUseClient(testFS(), path)
			if err != nil {
				t.Fatalf("hasUseClient: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// ── hasDefaultExport ──────────────────────────────────────────────────────────

func TestHasDefaultExport(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name    string
		content string
		want    bool
	}{
		{"sync function", "export default function Page() { return null; }", true},
		{"async function", "export default async function Page() { return null; }", true},
		{"class", "export default class Page {}", true},
		{"named export only", "export function Page() { return null; }", false},
		{"re-export", "export { Page as default } from './other'", false},
		{"empty file", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeFile(t, dir, tc.name+".tsx", tc.content)
			got, err := hasDefaultExport(testFS(), path)
			if err != nil {
				t.Fatalf("hasDefaultExport: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// ── routePattern ──────────────────────────────────────────────────────────────

func TestRoutePattern(t *testing.T) {
	appDir := t.TempDir()
	pagesDir := filepath.Join(appDir, "app")
	cases := []struct {
		rel  string
		want string
	}{
		{"page.tsx", "/"},
		{filepath.Join("posts", "page.tsx"), "/posts"},
		{filepath.Join("posts", "[slug]", "page.tsx"), "/posts/:slug"},
		{filepath.Join("shop", "[...path]", "page.tsx"), "/shop/:...path"},
		{filepath.Join("docs", "[[...slug]]", "page.tsx"), "/docs/:...slug?"},
		{filepath.Join("(marketing)", "about", "page.tsx"), "/about"},
		{filepath.Join("(shop)", "products", "page.tsx"), "/products"},
		{filepath.Join("(shop)", "products", "[id]", "page.tsx"), "/products/:id"},
		{filepath.Join("(a)", "(b)", "docs", "page.tsx"), "/docs"},
	}
	for _, tc := range cases {
		file := filepath.Join(pagesDir, tc.rel)
		got := routePattern(pagesDir, file)
		if got != tc.want {
			t.Errorf("routePattern(%q) = %q, want %q", tc.rel, got, tc.want)
		}
	}
}

// ── pageExport ────────────────────────────────────────────────────────────────

func TestPageExport(t *testing.T) {
	appDir := t.TempDir()
	pagesDir := filepath.Join(appDir, "app")
	cases := []struct {
		rel  string
		want string
	}{
		{"page.tsx", "Index"},
		{filepath.Join("posts", "page.tsx"), "Posts"},
		{filepath.Join("posts", "[slug]", "page.tsx"), "PostsSlug"},
		{filepath.Join("shop", "[...path]", "page.tsx"), "ShopPath"},
		{filepath.Join("docs", "[[...slug]]", "page.tsx"), "DocsSlug"},
		{filepath.Join("(shop)", "products", "page.tsx"), "Products"},
		{filepath.Join("(marketing)", "about", "page.tsx"), "About"},
		{filepath.Join("(shop)", "products", "[id]", "page.tsx"), "ProductsId"},
		{filepath.Join("(a)", "(b)", "docs", "page.tsx"), "Docs"},
	}
	for _, tc := range cases {
		file := filepath.Join(pagesDir, tc.rel)
		got := pageExport(pagesDir, file)
		if got != tc.want {
			t.Errorf("pageExport(%q) = %q, want %q", tc.rel, got, tc.want)
		}
	}
}
