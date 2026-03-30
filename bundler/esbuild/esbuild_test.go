//go:build esbuild

package esbuild

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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
