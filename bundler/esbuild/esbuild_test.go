//go:build esbuild

package esbuild

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/polagonow/pola/core"
)

// TestBuild_MinimalInput verifies that Build succeeds for a trivial server entry
// with no pages and produces a non-empty BundleOutput.
func TestBuild_MinimalInput(t *testing.T) {
	appDir := t.TempDir()
	outDir := filepath.Join(appDir, "public", "assets")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Minimal server entry — no React imports, just a global assignment.
	serverEntry := `(globalThis as any).__render__ = function() { return null; };`

	b := New()
	out, err := b.Build(context.Background(), core.BundleInput{
		AppDir:                 appDir,
		OutDir:                 outDir,
		AssetsURLPath:          "/public/assets",
		ServerEntryContent:     serverEntry,
		ServerBundleConditions: []string{"react-server", "browser", "module", "default"},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if out == nil {
		t.Fatal("expected non-nil BundleOutput")
	}
	if len(out.ServerBundle) == 0 {
		t.Error("expected non-empty ServerBundle")
	}
	// ManifestJSON should be a valid (possibly empty) JSON object.
	if len(out.ManifestJSON) == 0 {
		t.Error("expected non-empty ManifestJSON")
	}
}

// TestBuild_Name verifies the bundler name.
func TestBuild_Name(t *testing.T) {
	b := New()
	if b.Name() != "esbuild" {
		t.Errorf("Name() = %q, want esbuild", b.Name())
	}
}

// TestWatch_ReturnsInitialOutput verifies that Watch returns a channel
// and the first value is a valid BundleOutput.
func TestWatch_ReturnsInitialOutput(t *testing.T) {
	appDir := t.TempDir()
	outDir := filepath.Join(appDir, "public", "assets")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	serverEntry := `(globalThis as any).__render__ = function() { return null; };`

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b := New()
	ch, err := b.Watch(ctx, core.BundleInput{
		AppDir:                 appDir,
		OutDir:                 outDir,
		AssetsURLPath:          "/public/assets",
		ServerEntryContent:     serverEntry,
		ServerBundleConditions: []string{"react-server", "browser", "module", "default"},
	})
	if err != nil {
		t.Fatalf("Watch: %v", err)
	}
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}

	out := <-ch
	if out == nil {
		t.Fatal("expected non-nil initial BundleOutput")
	}
	if len(out.ServerBundle) == 0 {
		t.Error("expected non-empty ServerBundle")
	}
}

// TestWatch_CancelClosesChannel verifies that canceling the context closes the channel.
func TestWatch_CancelClosesChannel(t *testing.T) {
	appDir := t.TempDir()
	outDir := filepath.Join(appDir, "public", "assets")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	serverEntry := `(globalThis as any).__render__ = function() { return null; };`

	ctx, cancel := context.WithCancel(context.Background())

	b := New()
	ch, err := b.Watch(ctx, core.BundleInput{
		AppDir:                 appDir,
		OutDir:                 outDir,
		AssetsURLPath:          "/public/assets",
		ServerEntryContent:     serverEntry,
		ServerBundleConditions: []string{"react-server", "browser", "module", "default"},
	})
	if err != nil {
		t.Fatalf("Watch: %v", err)
	}

	// Drain initial output.
	<-ch

	// Cancel and verify channel closes.
	cancel()
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after context cancel")
	}
}

// TestWatch_InitialBuildError verifies that Watch returns an error when the
// initial build fails.
func TestWatch_InitialBuildError(t *testing.T) {
	b := New()
	_, err := b.Watch(context.Background(), core.BundleInput{
		AppDir:                 "/nonexistent",
		OutDir:                 "/nonexistent/out",
		ServerEntryContent:     `import "totally-bogus-nonexistent-pkg";`,
		ServerBundleConditions: []string{"browser"},
	})
	if err == nil {
		t.Error("expected Watch to return an error for bad input")
	}
}

// TestBuild_EmptyServerEntry verifies that passing no ServerEntryContent
// returns an empty ServerBundle without error.
func TestBuild_EmptyServerEntry(t *testing.T) {
	appDir := t.TempDir()
	outDir := filepath.Join(appDir, "public", "assets")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	b := New()
	out, err := b.Build(context.Background(), core.BundleInput{
		AppDir:        appDir,
		OutDir:        outDir,
		AssetsURLPath: "/public/assets",
		// No ServerEntryContent — server pass is skipped.
	})
	if err != nil {
		t.Fatalf("Build with empty entry: %v", err)
	}
	if out == nil {
		t.Fatal("expected non-nil BundleOutput")
	}
	// ServerBundle should be empty since there was no entry.
	if len(out.ServerBundle) != 0 {
		t.Errorf("expected empty ServerBundle, got %d bytes", len(out.ServerBundle))
	}
}

// TestFilePoller_DetectsChange verifies that the poller fires onChange when a
// watched file is modified.
func TestFilePoller_DetectsChange(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "app.ts")
	if err := os.WriteFile(f, []byte("const x = 1;"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Let the file age past the safety gap so the poller uses ModKey comparison.
	time.Sleep(10 * time.Millisecond)

	var fired atomic.Int32
	p := newFilePoller(func() { fired.Add(1) })
	p.SetPaths([]string{f})
	p.Start()
	defer p.Stop()

	// Modify the file.
	time.Sleep(150 * time.Millisecond) // let at least one poll pass
	if err := os.WriteFile(f, []byte("const x = 2;"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for the poller to detect the change.
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

// TestFilePoller_DetectsNewFile verifies that after SetPaths is called with a
// newly created file, the poller detects changes to it.
func TestFilePoller_DetectsNewFile(t *testing.T) {
	dir := t.TempDir()

	var fired atomic.Int32
	p := newFilePoller(func() { fired.Add(1) })
	p.SetPaths(nil) // start with no paths
	p.Start()
	defer p.Stop()

	// Create a new file and add it to watched paths.
	f := filepath.Join(dir, "new.tsx")
	if err := os.WriteFile(f, []byte("export default 1;"), 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	p.SetPaths([]string{f})

	// Modify it.
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

// TestFilePoller_StopIsIdempotent verifies that calling Stop multiple times
// does not panic.
func TestFilePoller_StopIsIdempotent(t *testing.T) {
	p := newFilePoller(func() {})
	p.Start()
	p.Stop()
	p.Stop() // should not panic
}

// TestCollectSourcePaths verifies that the walker finds source files and
// excludes node_modules and dot-directories.
func TestCollectSourcePaths(t *testing.T) {
	dir := t.TempDir()
	// Create source files.
	os.MkdirAll(filepath.Join(dir, "components"), 0o755)
	os.WriteFile(filepath.Join(dir, "page.tsx"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "components", "Button.tsx"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "globals.css"), []byte("x"), 0o644)
	// Create excluded paths.
	os.MkdirAll(filepath.Join(dir, "node_modules", "react"), 0o755)
	os.WriteFile(filepath.Join(dir, "node_modules", "react", "index.js"), []byte("x"), 0o644)
	os.MkdirAll(filepath.Join(dir, ".hidden"), 0o755)
	os.WriteFile(filepath.Join(dir, ".hidden", "secret.ts"), []byte("x"), 0o644)
	// Non-source file.
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte("x"), 0o644)

	paths := collectSourcePaths(dir)
	want := map[string]bool{
		filepath.Join(dir, "page.tsx"):                    true,
		filepath.Join(dir, "components", "Button.tsx"):    true,
		filepath.Join(dir, "globals.css"):                 true,
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

// TestComputeModuleID checks the manifest module ID derivation.
func TestComputeModuleID(t *testing.T) {
	appDir := "/app"
	cases := []struct {
		path string
		want string
	}{
		{"/app/components/Counter.tsx", "components/Counter"},
		{"/app/components/ui/Button.tsx", "components/ui/Button"},
		{"/some/node_modules/@pola/react/components/Client.tsx", "@pola/react/components/Client"},
	}
	for _, tc := range cases {
		got := computeModuleID(appDir, tc.path)
		if got != tc.want {
			t.Errorf("computeModuleID(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}
