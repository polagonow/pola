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
		if p.Export != "default" {
			t.Errorf("%s: Export = %q, want default", p.File, p.Export)
		}
		dirs[filepath.Base(filepath.Dir(p.File))] = true
	}
	for _, want := range []string{"index", "products", "user", "about"} {
		if !dirs[want] {
			t.Errorf("route %q not discovered", want)
		}
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
