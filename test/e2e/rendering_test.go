// Package e2e contains end-to-end tests for the Pola SSR framework.
//
// Tests are organised into testsuites (one file per feature area) and run
// against every registered VM×renderer×bundler fixture via the driver package.
//
// Run all suites:
//
//	go test ./test/e2e/...
package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	// Import to register all VM engines and bundler+renderer combos.
	"github.com/polagonow/pola/internal/actionbridge"
	"github.com/polagonow/pola/internal/stubpkgs"
	_ "github.com/polagonow/pola/test/combo"
	bundlersuite "github.com/polagonow/pola/test/e2e/bundler"
	enginesuite "github.com/polagonow/pola/test/e2e/engine"
	reactsuite "github.com/polagonow/pola/test/e2e/renderer/react"
	routersuite "github.com/polagonow/pola/test/e2e/router"
	"github.com/polagonow/pola/test/e2e/suite"
	"github.com/polagonow/pola/test/fixture"
	_ "github.com/polagonow/pola/test/vm"
)

// TestMain provisions the fixture app before running the suites. The e2e tests
// build the fixture through pola.BuildApp directly (bypassing the CLI), so two
// codegen steps the CLI normally performs must be replicated here:
//  1. Materialize the @pola/* workspace packages into the app's node_modules.
//  2. Generate @pola/actions' TypeScript client (generated.ts) from the app's
//     Go actions/ directory, so page.tsx imports like { Blog } resolve.
//
// The app's npm dependencies (react, tailwindcss, …) must already be installed
// (pnpm install in examples/blog-e2e-react/web).
func TestMain(m *testing.M) {
	if err := provisionFixture(); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: fixture provisioning failed: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func provisionFixture() error {
	// 1. Stub @pola/* into node_modules (this resets generated.ts to a placeholder).
	if err := stubpkgs.StubToNodeModules(fixture.AppDir); err != nil {
		return fmt.Errorf("stub @pola packages into %s: %w", fixture.AppDir, err)
	}

	// 2. Regenerate @pola/actions/src/generated.ts from the app's actions/ dir.
	actionsDir := filepath.Join(filepath.Dir(fixture.AppDir), "actions")
	if info, err := os.Stat(actionsDir); err != nil || !info.IsDir() {
		return nil // no actions to bridge
	}
	tsOut := filepath.Join(fixture.AppDir, "node_modules", "@pola", "actions", "src", "generated.ts")
	tmpDir, err := os.MkdirTemp("", "pola-actionbridge-*")
	if err != nil {
		return fmt.Errorf("tmpdir: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	if _, err := actionbridge.Run(actionsDir, tsOut, tmpDir, "github.com/polagonow/pola"); err != nil {
		return fmt.Errorf("actionbridge codegen: %w", err)
	}
	return nil
}

func TestHTMLShell(t *testing.T)                { suite.RunHTMLShellTests(t) }
func TestServerComponentRendering(t *testing.T) { suite.RunServerComponentRenderingTests(t) }
func TestNotFoundHandling(t *testing.T)         { suite.RunNotFoundHandlingTests(t) }
func TestConcurrentRendering(t *testing.T)      { suite.RunConcurrentRenderingTests(t) }

// React-specific suites.
func TestFlightProtocol(t *testing.T)    { reactsuite.RunFlightProtocolTests(t) }
func TestErrorBoundary(t *testing.T)     { reactsuite.RunErrorBoundaryTests(t) }
func TestLayoutComposition(t *testing.T) { reactsuite.RunLayoutCompositionTests(t) }

// Bundler suites.
func TestClientBundle(t *testing.T) { bundlersuite.RunClientBundleTests(t) }

// Engine suites.
func TestPolyfills(t *testing.T) { enginesuite.RunPolyfillTests(t) }

// Router suites.
func TestNextJSRouter(t *testing.T) { routersuite.RunNextJSRouterTests(t) }
