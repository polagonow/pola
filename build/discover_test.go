package build

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// ── hasUseClient ────────────────────────────────────────────────────────────

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
			got, err := hasUseClient(path)
			if err != nil {
				t.Fatalf("hasUseClient: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// ── DiscoverPages ───────────────────────────────────────────────────────────

func TestDiscoverPages(t *testing.T) {
	appDir := t.TempDir()
	pagesDir := filepath.Join(appDir, "app")

	// Create route subdirectories each with a page.tsx
	for _, route := range []string{"index", "products", "user", "about"} {
		dir := filepath.Join(pagesDir, route)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, dir, "page.tsx", "export default function Page() { return null; }\n")
	}
	// A non-page file that should be ignored
	writeFile(t, filepath.Join(pagesDir, "index"), "helpers.ts", "export const x = 1;\n")

	pages, err := DiscoverPages(appDir)
	if err != nil {
		t.Fatalf("DiscoverPages: %v", err)
	}
	if len(pages) != 4 {
		t.Fatalf("expected 4 pages, got %d", len(pages))
	}

	dirs := make(map[string]bool)
	for _, p := range pages {
		dirs[filepath.Base(filepath.Dir(p.PageComponentPath))] = true
	}
	for _, want := range []string{"index", "products", "user", "about"} {
		if !dirs[want] {
			t.Errorf("route %q not discovered", want)
		}
	}
}

func TestDiscoverPages_MissingDefaultExport(t *testing.T) {
	appDir := t.TempDir()
	dir := filepath.Join(appDir, "app", "bad")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Named export — violates the convention
	writeFile(t, dir, "page.tsx", "export function BadPage() { return null; }\n")

	_, err := DiscoverPages(appDir)
	if err == nil {
		t.Fatal("expected error for missing default export, got nil")
	}
}

// ── discoverCompanions / companion fields ────────────────────────────────────

func TestDiscoverPages_WithCompanions(t *testing.T) {
	appDir := t.TempDir()
	dir := filepath.Join(appDir, "app", "products")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "page.tsx", "export default function Page() { return null; }\n")
	writeFile(t, dir, "layout.tsx", "export default function Layout({children}: any) { return children; }\n")
	writeFile(t, dir, "loading.tsx", "export default function Loading() { return null; }\n")
	writeFile(t, dir, "error.tsx", "\"use client\";\nimport React from \"react\";\nexport default class E extends React.Component<any,any> { render() { return null; } }\n")
	writeFile(t, dir, "not-found.tsx", "export default function NotFound() { return null; }\n")

	pages, err := DiscoverPages(appDir)
	if err != nil {
		t.Fatalf("DiscoverPages: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
	p := pages[0]
	// Segments: only the co-located segment (no ancestor dirs with layout/error in this temp tree).
	if len(p.Segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(p.Segments))
	}
	seg := p.Segments[0]
	if filepath.Base(seg.LayoutPath) != "layout.tsx" {
		t.Errorf("unexpected Segments[0].LayoutPath: %s", seg.LayoutPath)
	}
	if filepath.Base(seg.ErrorPath) != "error.tsx" {
		t.Errorf("unexpected Segments[0].ErrorPath: %s", seg.ErrorPath)
	}
	if p.LoadingComponentPath == "" {
		t.Error("expected LoadingComponentPath to be set")
	}
	if p.NotFoundComponentPath == "" {
		t.Error("expected NotFoundComponentPath to be set")
	}
	if filepath.Base(p.LoadingComponentPath) != "loading.tsx" {
		t.Errorf("unexpected LoadingComponentPath: %s", p.LoadingComponentPath)
	}
	if filepath.Base(p.NotFoundComponentPath) != "not-found.tsx" {
		t.Errorf("unexpected NotFoundComponentPath: %s", p.NotFoundComponentPath)
	}
}

func TestDiscoverPages_PartialCompanions(t *testing.T) {
	appDir := t.TempDir()
	dir := filepath.Join(appDir, "app", "about")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "page.tsx", "export default function Page() { return null; }\n")
	writeFile(t, dir, "layout.tsx", "export default function Layout({children}: any) { return children; }\n")
	// No loading.tsx, error.tsx, or not-found.tsx

	pages, err := DiscoverPages(appDir)
	if err != nil {
		t.Fatalf("DiscoverPages: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
	p := pages[0]
	if len(p.Segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(p.Segments))
	}
	if filepath.Base(p.Segments[0].LayoutPath) != "layout.tsx" {
		t.Errorf("unexpected Segments[0].LayoutPath: %s", p.Segments[0].LayoutPath)
	}
	if p.LoadingComponentPath != "" {
		t.Errorf("expected LoadingComponentPath to be empty, got %s", p.LoadingComponentPath)
	}
	if p.NotFoundComponentPath != "" {
		t.Errorf("expected NotFoundComponentPath to be empty, got %s", p.NotFoundComponentPath)
	}
}

func TestDiscoverPages_CompanionWithoutDefaultExport(t *testing.T) {
	appDir := t.TempDir()
	dir := filepath.Join(appDir, "app", "user")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "page.tsx", "export default function Page() { return null; }\n")
	// layout.tsx with no default export — should be ignored
	writeFile(t, dir, "layout.tsx", "export function Layout({children}: any) { return children; }\n")

	pages, err := DiscoverPages(appDir)
	if err != nil {
		t.Fatalf("DiscoverPages: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
	if len(pages[0].Segments) != 0 {
		t.Errorf("expected empty Segments (no default export), got %v", pages[0].Segments)
	}
}

func TestLayoutAlias(t *testing.T) {
	appDir := t.TempDir()
	pagesDir := filepath.Join(appDir, "pages")
	cases := []struct {
		rel  string
		want string
	}{
		{"layout.tsx", "Index"},
		{filepath.Join("about", "layout.tsx"), "About"},
		{filepath.Join("products", "layout.tsx"), "Products"},
		{filepath.Join("products", "[id]", "layout.tsx"), "ProductsId"},
		{filepath.Join("user", "layout.tsx"), "User"},
	}
	for _, tc := range cases {
		got := LayoutAlias(pagesDir, filepath.Join(pagesDir, tc.rel))
		if got != tc.want {
			t.Errorf("LayoutAlias(%q) = %q, want %q", tc.rel, got, tc.want)
		}
	}
}

func TestCollectSegments(t *testing.T) {
	appDir := t.TempDir()
	pagesDir := filepath.Join(appDir, "pages")
	// Create directories with layout.tsx at each level.
	for _, dir := range []string{
		pagesDir,
		filepath.Join(pagesDir, "products"),
		filepath.Join(pagesDir, "products", "[id]"),
		filepath.Join(pagesDir, "about"),
		filepath.Join(pagesDir, "user"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, dir, "layout.tsx", "export default function L({children}: any){return children}\n")
	}

	cases := []struct {
		pageDir  string
		wantLen  int
		wantDirs []string // expected filepath.Base of each segment's Dir
	}{
		{pagesDir, 1, []string{filepath.Base(pagesDir)}},
		{filepath.Join(pagesDir, "about"), 2, []string{filepath.Base(pagesDir), "about"}},
		{filepath.Join(pagesDir, "products"), 2, []string{filepath.Base(pagesDir), "products"}},
		{filepath.Join(pagesDir, "products", "[id]"), 3, []string{filepath.Base(pagesDir), "products", "[id]"}},
		{filepath.Join(pagesDir, "user"), 2, []string{filepath.Base(pagesDir), "user"}},
	}
	for _, tc := range cases {
		segs, err := collectSegments(pagesDir, tc.pageDir)
		if err != nil {
			t.Errorf("pageDir=%s: collectSegments error: %v", tc.pageDir, err)
			continue
		}
		if len(segs) != tc.wantLen {
			t.Errorf("pageDir=%s: segments length %d, want %d", tc.pageDir, len(segs), tc.wantLen)
			continue
		}
		for i, wantDir := range tc.wantDirs {
			got := filepath.Base(segs[i].Dir)
			if got != wantDir {
				t.Errorf("pageDir=%s segs[%d].Dir base=%q, want %q", tc.pageDir, i, got, wantDir)
			}
			if segs[i].LayoutPath == "" {
				t.Errorf("pageDir=%s segs[%d].LayoutPath should be set", tc.pageDir, i)
			}
		}
	}
}

func TestCollectSegments_MissingLayouts(t *testing.T) {
	// Only the root and deepest level have layouts; middle level does not.
	appDir := t.TempDir()
	pagesDir := filepath.Join(appDir, "pages")
	idDir := filepath.Join(pagesDir, "products", "[id]")
	if err := os.MkdirAll(idDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// root layout exists, products layout does NOT, [id] layout exists
	writeFile(t, pagesDir, "layout.tsx", "export default function L({c}: any){return c}\n")
	writeFile(t, idDir, "layout.tsx", "export default function L({c}: any){return c}\n")

	segs, err := collectSegments(pagesDir, idDir)
	if err != nil {
		t.Fatalf("collectSegments error: %v", err)
	}
	if len(segs) != 2 {
		t.Fatalf("expected 2 segments (root + [id]), got %d: %v", len(segs), segs)
	}
	if filepath.Base(segs[0].Dir) != filepath.Base(pagesDir) {
		t.Errorf("segs[0] should be root dir, got %s", segs[0].Dir)
	}
	if filepath.Base(segs[1].Dir) != "[id]" {
		t.Errorf("segs[1] should be [id] dir, got %s", segs[1].Dir)
	}
}

func TestCollectSegments_ErrorPaths(t *testing.T) {
	// Root has layout only; products has both layout and error; [id] has error only.
	appDir := t.TempDir()
	pagesDir := filepath.Join(appDir, "pages")
	productsDir := filepath.Join(pagesDir, "products")
	idDir := filepath.Join(productsDir, "[id]")
	for _, d := range []string{pagesDir, productsDir, idDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, pagesDir, "layout.tsx", "export default function L({c}: any){return c}\n")
	writeFile(t, productsDir, "layout.tsx", "export default function L({c}: any){return c}\n")
	writeFile(t, productsDir, "error.tsx", "\"use client\";\nexport default function E() { return null; }\n")
	writeFile(t, idDir, "error.tsx", "\"use client\";\nexport default function E() { return null; }\n")

	segs, err := collectSegments(pagesDir, idDir)
	if err != nil {
		t.Fatalf("collectSegments error: %v", err)
	}
	// Expect 3 segments: root (layout), products (layout+error), [id] (error)
	if len(segs) != 3 {
		t.Fatalf("expected 3 segments, got %d: %v", len(segs), segs)
	}
	if segs[0].LayoutPath == "" || segs[0].ErrorPath != "" {
		t.Errorf("segs[0]: want layout only, got layout=%q error=%q", segs[0].LayoutPath, segs[0].ErrorPath)
	}
	if segs[1].LayoutPath == "" || segs[1].ErrorPath == "" {
		t.Errorf("segs[1]: want layout+error, got layout=%q error=%q", segs[1].LayoutPath, segs[1].ErrorPath)
	}
	if segs[2].LayoutPath != "" || segs[2].ErrorPath == "" {
		t.Errorf("segs[2]: want error only, got layout=%q error=%q", segs[2].LayoutPath, segs[2].ErrorPath)
	}
}

// ── hasDefaultExport ────────────────────────────────────────────────────────

func TestHasDefaultExport(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name    string
		content string
		want    bool
	}{
		{"sync function", "export default function Page() { return null; }", true},
		{"async function", "export default async function Page() { return null; }", true},
		{"named export only", "export function Page() { return null; }", false},
		{"re-export", "export { Page as default } from './other'", false},
		{"empty file", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeFile(t, dir, tc.name+".tsx", tc.content)
			got, err := hasDefaultExport(path)
			if err != nil {
				t.Fatalf("hasDefaultExport: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// ── DiscoverGlobalComponents ────────────────────────────────────────────────

func TestDiscoverGlobalComponents(t *testing.T) {
	appDir := t.TempDir()
	pagesDir := filepath.Join(appDir, "app")
	if err := os.MkdirAll(pagesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Run("both present", func(t *testing.T) {
		writeFile(t, pagesDir, "global-not-found.tsx", "export default function GlobalNotFound() { return null; }\n")
		writeFile(t, pagesDir, "global-error.tsx", "'use client';\nexport default function GlobalError() { return null; }\n")

		gc, err := DiscoverGlobalComponents(appDir)
		if err != nil {
			t.Fatalf("DiscoverGlobalComponents: %v", err)
		}
		if filepath.Base(gc.NotFoundPath) != "global-not-found.tsx" {
			t.Errorf("NotFoundPath = %q, want global-not-found.tsx", gc.NotFoundPath)
		}
		if filepath.Base(gc.ErrorPath) != "global-error.tsx" {
			t.Errorf("ErrorPath = %q, want global-error.tsx", gc.ErrorPath)
		}
	})

	t.Run("neither present", func(t *testing.T) {
		emptyDir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(emptyDir, "app"), 0o755); err != nil {
			t.Fatal(err)
		}
		gc, err := DiscoverGlobalComponents(emptyDir)
		if err != nil {
			t.Fatalf("DiscoverGlobalComponents: %v", err)
		}
		if gc.NotFoundPath != "" {
			t.Errorf("expected empty NotFoundPath, got %q", gc.NotFoundPath)
		}
		if gc.ErrorPath != "" {
			t.Errorf("expected empty ErrorPath, got %q", gc.ErrorPath)
		}
	})

	t.Run("no default export ignored", func(t *testing.T) {
		dir := t.TempDir()
		pd := filepath.Join(dir, "app")
		if err := os.MkdirAll(pd, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, pd, "global-not-found.tsx", "export function GlobalNotFound() { return null; }\n")

		gc, err := DiscoverGlobalComponents(dir)
		if err != nil {
			t.Fatalf("DiscoverGlobalComponents: %v", err)
		}
		if gc.NotFoundPath != "" {
			t.Errorf("expected empty NotFoundPath (no default export), got %q", gc.NotFoundPath)
		}
	})
}

// ── DiscoverClientComponents ────────────────────────────────────────────────

func TestDiscoverClientComponents(t *testing.T) {
	appDir := t.TempDir()
	compDir := filepath.Join(appDir, "components")
	subDir := filepath.Join(compDir, "ui")
	for _, d := range []string{compDir, subDir} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	writeFile(t, compDir, "Counter.tsx", `"use client"`+"\nexport default function Counter() {}")
	writeFile(t, compDir, "Header.tsx", "export default function Header() {}")
	writeFile(t, subDir, "Button.tsx", `'use client'`+"\nexport default function Button() {}")
	writeFile(t, subDir, "Label.tsx", "export default function Label() {}")

	got, err := DiscoverClientComponents(appDir)
	if err != nil {
		t.Fatalf("DiscoverClientComponents: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 client components, got %d: %v", len(got), got)
	}

	names := make(map[string]bool)
	for _, p := range got {
		names[filepath.Base(p)] = true
	}
	if !names["Counter.tsx"] {
		t.Error("Counter.tsx not found")
	}
	if !names["Button.tsx"] {
		t.Error("Button.tsx not found")
	}
}

// ── Route Groups ──────────────────────────────────────────────────────────────

func TestRoutePattern_RouteGroups(t *testing.T) {
	appDir := t.TempDir()
	cases := []struct {
		rel  string
		want string
	}{
		{filepath.Join("(marketing)", "about", "page.tsx"), "/about"},
		{filepath.Join("(shop)", "products", "page.tsx"), "/products"},
		{filepath.Join("(shop)", "products", "[id]", "page.tsx"), "/products/:id"},
		{filepath.Join("(a)", "(b)", "docs", "page.tsx"), "/docs"},
	}
	for _, tc := range cases {
		file := filepath.Join(appDir, "app", tc.rel)
		got := RoutePattern(appDir, file)
		if got != tc.want {
			t.Errorf("RoutePattern(%q) = %q, want %q", tc.rel, got, tc.want)
		}
	}
}

func TestPageAlias_RouteGroups(t *testing.T) {
	appDir := t.TempDir()
	cases := []struct {
		rel  string
		want string
	}{
		{filepath.Join("(shop)", "products", "page.tsx"), "Products"},
		{filepath.Join("(marketing)", "about", "page.tsx"), "About"},
		{filepath.Join("(shop)", "products", "[id]", "page.tsx"), "ProductsId"},
		{filepath.Join("(a)", "(b)", "docs", "page.tsx"), "Docs"},
	}
	for _, tc := range cases {
		dir := filepath.Join(appDir, "app", filepath.Dir(tc.rel))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, dir, "page.tsx", "export default function Page() { return null; }\n")
	}
	pages, err := DiscoverPages(appDir)
	if err != nil {
		t.Fatalf("DiscoverPages: %v", err)
	}
	aliases := make(map[string]bool)
	for _, p := range pages {
		aliases[PageAlias(p)] = true
	}
	for _, tc := range cases {
		if !aliases[tc.want] {
			t.Errorf("expected alias %q (for %q), got aliases: %v", tc.want, tc.rel, aliases)
		}
	}
}

func TestLayoutAlias_RouteGroups(t *testing.T) {
	appDir := t.TempDir()
	pagesDir := filepath.Join(appDir, "pages")
	cases := []struct {
		rel  string
		want string
	}{
		{filepath.Join("(shop)", "layout.tsx"), "Shop"},
		{filepath.Join("(marketing)", "about", "layout.tsx"), "MarketingAbout"},
		{filepath.Join("(shop)", "products", "[id]", "layout.tsx"), "ShopProductsId"},
	}
	for _, tc := range cases {
		got := LayoutAlias(pagesDir, filepath.Join(pagesDir, tc.rel))
		if got != tc.want {
			t.Errorf("LayoutAlias(%q) = %q, want %q", tc.rel, got, tc.want)
		}
	}
}
