// Package bundlersuite contains bundler-specific e2e test suites.
package bundlersuite

import (
	"os"
	"strings"
	"testing"

	"github.com/polagonow/pola/core/reserved"
	"github.com/polagonow/pola/test/fixture"
)

// assetsDir is where the bundler writes compiled client assets: the framework's
// default public dir ("./public"), resolved relative to the test's working
// directory (the test/e2e package dir) — not the fixture app's own public dir.
const assetsDir = "public/assets/"

// RunClientBundleTests verifies that the client-side JS bundle is built, served,
// and free of bundler internals that should not be exposed to the browser.
func RunClientBundleTests(t *testing.T) {
	t.Helper()

	t.Run("EntryPointHasReservedAssetsURL", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			app := f.GetApp(t)
			if app.Artifacts().Output.ClientEntryURL == "" {
				t.Fatal("client entry URL is empty — bundle was not built")
			}
			if !strings.HasPrefix(app.Artifacts().Output.ClientEntryURL, reserved.Assets+"/") {
				t.Errorf("client entry URL should be under %s/, got %q", reserved.Assets, app.Artifacts().Output.ClientEntryURL)
			}
		})
	})

	t.Run("BundleFileIsPresentOnDisk", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			app := f.GetApp(t)
			rel := strings.TrimPrefix(app.Artifacts().Output.ClientEntryURL, reserved.Assets+"/")
			path := assetsDir + rel
			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("client bundle file not found at %q: %v", path, err)
			}
			if info.Size() < 10_000 {
				t.Errorf("client bundle suspiciously small (%d bytes) — may be empty or incomplete", info.Size())
			}
		})
	})

	t.Run("BundleDoesNotContainBundlerInternals", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			app := f.GetApp(t)
			rel := strings.TrimPrefix(app.Artifacts().Output.ClientEntryURL, reserved.Assets+"/")
			data, err := os.ReadFile(assetsDir + rel)
			if err != nil {
				t.Fatalf("read client bundle: %v", err)
			}
			if strings.Contains(string(data), "__webpack_require__") {
				t.Error("client bundle exposes __webpack_require__ — bundler internals leaked into output")
			}
		})
	})
}
