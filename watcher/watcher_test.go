package watcher

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// TestPoller_DetectsChange verifies that the poller fires onChange when a
// watched file is modified.
func TestPoller_DetectsChange(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "app.ts")
	if err := os.WriteFile(f, []byte("const x = 1;"), 0o644); err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Millisecond)

	var fired atomic.Int32
	p := New(func() { fired.Add(1) })
	p.SetPaths([]string{f})
	p.Start()
	defer p.Stop()

	// Modify the file.
	time.Sleep(150 * time.Millisecond)
	if err := os.WriteFile(f, []byte("const x = 2;"), 0o644); err != nil {
		t.Fatal(err)
	}

	deadline := time.After(2 * time.Second)
	for fired.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("poller did not detect file change within 2s")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// TestPoller_DetectsNewFile verifies that after SetPaths is called with a
// newly created file, the poller detects changes to it.
func TestPoller_DetectsNewFile(t *testing.T) {
	dir := t.TempDir()

	var fired atomic.Int32
	p := New(func() { fired.Add(1) })
	p.SetPaths(nil)
	p.Start()
	defer p.Stop()

	f := filepath.Join(dir, "new.tsx")
	if err := os.WriteFile(f, []byte("export default 1;"), 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	p.SetPaths([]string{f})

	time.Sleep(150 * time.Millisecond)
	if err := os.WriteFile(f, []byte("export default 2;"), 0o644); err != nil {
		t.Fatal(err)
	}

	deadline := time.After(2 * time.Second)
	for fired.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("poller did not detect new file change within 2s")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// TestPoller_StopIsIdempotent verifies that calling Stop multiple times
// does not panic.
func TestPoller_StopIsIdempotent(t *testing.T) {
	p := New(func() {})
	p.Start()
	p.Stop()
	p.Stop()
}

// TestCollectPaths verifies that the walker finds source files and
// excludes node_modules, vendor, and dot-directories.
func TestCollectPaths(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "components"), 0o755)
	os.WriteFile(filepath.Join(dir, "page.tsx"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "components", "Button.tsx"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "globals.css"), []byte("x"), 0o644)
	// Excluded paths.
	os.MkdirAll(filepath.Join(dir, "node_modules", "react"), 0o755)
	os.WriteFile(filepath.Join(dir, "node_modules", "react", "index.js"), []byte("x"), 0o644)
	os.MkdirAll(filepath.Join(dir, "vendor", "lib"), 0o755)
	os.WriteFile(filepath.Join(dir, "vendor", "lib", "main.go"), []byte("x"), 0o644)
	os.MkdirAll(filepath.Join(dir, ".hidden"), 0o755)
	os.WriteFile(filepath.Join(dir, ".hidden", "secret.ts"), []byte("x"), 0o644)
	// Non-matching extension.
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte("x"), 0o644)

	paths := CollectPaths(dir, []string{".tsx", ".css"})
	want := map[string]bool{
		filepath.Join(dir, "page.tsx"):                 true,
		filepath.Join(dir, "components", "Button.tsx"): true,
		filepath.Join(dir, "globals.css"):              true,
	}
	got := make(map[string]bool, len(paths))
	for _, p := range paths {
		got[p] = true
	}
	for w := range want {
		if !got[w] {
			t.Errorf("missing expected path: %s", w)
		}
	}
	for g := range got {
		if !want[g] {
			t.Errorf("unexpected path: %s", g)
		}
	}
}

// TestCollectPaths_GoFiles verifies that .go and .tmpl files are collected.
func TestCollectPaths_GoFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)
	os.WriteFile(filepath.Join(dir, "page.tmpl"), []byte("{{.}}"), 0o644)
	os.WriteFile(filepath.Join(dir, "style.css"), []byte("body{}"), 0o644)

	paths := CollectPaths(dir, []string{".go", ".tmpl"})
	want := map[string]bool{
		filepath.Join(dir, "main.go"):   true,
		filepath.Join(dir, "page.tmpl"): true,
	}
	got := make(map[string]bool, len(paths))
	for _, p := range paths {
		got[p] = true
	}
	if len(got) != len(want) {
		t.Errorf("got %d paths, want %d", len(got), len(want))
	}
	for w := range want {
		if !got[w] {
			t.Errorf("missing expected path: %s", w)
		}
	}
}
