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
	pagesDir := filepath.Join(appDir, "pages")

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
	dir := filepath.Join(appDir, "pages", "bad")
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
	dir := filepath.Join(appDir, "pages", "products")
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
	// LayoutChain: only the co-located layout (no ancestor layouts in this temp tree).
	if len(p.LayoutChain) != 1 {
		t.Fatalf("expected LayoutChain length 1, got %d", len(p.LayoutChain))
	}
	if filepath.Base(p.LayoutChain[0]) != "layout.tsx" {
		t.Errorf("unexpected LayoutChain[0]: %s", p.LayoutChain[0])
	}
	if p.LoadingComponentPath == "" {
		t.Error("expected LoadingComponentPath to be set")
	}
	if p.ErrorComponentPath == "" {
		t.Error("expected ErrorComponentPath to be set")
	}
	if p.NotFoundComponentPath == "" {
		t.Error("expected NotFoundComponentPath to be set")
	}
	if filepath.Base(p.LoadingComponentPath) != "loading.tsx" {
		t.Errorf("unexpected LoadingComponentPath: %s", p.LoadingComponentPath)
	}
	if filepath.Base(p.ErrorComponentPath) != "error.tsx" {
		t.Errorf("unexpected ErrorComponentPath: %s", p.ErrorComponentPath)
	}
	if filepath.Base(p.NotFoundComponentPath) != "not-found.tsx" {
		t.Errorf("unexpected NotFoundComponentPath: %s", p.NotFoundComponentPath)
	}
}

func TestDiscoverPages_PartialCompanions(t *testing.T) {
	appDir := t.TempDir()
	dir := filepath.Join(appDir, "pages", "about")
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
	if len(p.LayoutChain) != 1 {
		t.Fatalf("expected LayoutChain length 1, got %d", len(p.LayoutChain))
	}
	if p.LoadingComponentPath != "" {
		t.Errorf("expected LoadingComponentPath to be empty, got %s", p.LoadingComponentPath)
	}
	if p.ErrorComponentPath != "" {
		t.Errorf("expected ErrorComponentPath to be empty, got %s", p.ErrorComponentPath)
	}
	if p.NotFoundComponentPath != "" {
		t.Errorf("expected NotFoundComponentPath to be empty, got %s", p.NotFoundComponentPath)
	}
}

func TestDiscoverPages_CompanionWithoutDefaultExport(t *testing.T) {
	appDir := t.TempDir()
	dir := filepath.Join(appDir, "pages", "user")
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
	if len(pages[0].LayoutChain) != 0 {
		t.Errorf("expected empty LayoutChain (no default export), got %v", pages[0].LayoutChain)
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

func TestCollectLayoutChain(t *testing.T) {
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
		wantDirs []string // expected filepath.Base of each layout's parent dir
	}{
		{pagesDir, 1, []string{filepath.Base(pagesDir)}},
		{filepath.Join(pagesDir, "about"), 2, []string{filepath.Base(pagesDir), "about"}},
		{filepath.Join(pagesDir, "products"), 2, []string{filepath.Base(pagesDir), "products"}},
		{filepath.Join(pagesDir, "products", "[id]"), 3, []string{filepath.Base(pagesDir), "products", "[id]"}},
		{filepath.Join(pagesDir, "user"), 2, []string{filepath.Base(pagesDir), "user"}},
	}
	for _, tc := range cases {
		chain := collectLayoutChain(pagesDir, tc.pageDir)
		if len(chain) != tc.wantLen {
			t.Errorf("pageDir=%s: chain length %d, want %d", tc.pageDir, len(chain), tc.wantLen)
			continue
		}
		for i, wantDir := range tc.wantDirs {
			got := filepath.Base(filepath.Dir(chain[i]))
			if got != wantDir {
				t.Errorf("pageDir=%s chain[%d] dir=%q, want %q", tc.pageDir, i, got, wantDir)
			}
		}
	}
}

func TestCollectLayoutChain_MissingLayouts(t *testing.T) {
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

	chain := collectLayoutChain(pagesDir, idDir)
	if len(chain) != 2 {
		t.Fatalf("expected 2 layouts (root + [id]), got %d: %v", len(chain), chain)
	}
	if filepath.Base(filepath.Dir(chain[0])) != filepath.Base(pagesDir) {
		t.Errorf("chain[0] should be root layout, got %s", chain[0])
	}
	if filepath.Base(filepath.Dir(chain[1])) != "[id]" {
		t.Errorf("chain[1] should be [id] layout, got %s", chain[1])
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
