// Package e2e contains end-to-end tests for the GoJSX SSR framework.
//
// Tests are organised into testsuites (one file per feature area) and run
// against every registered VM×renderer×bundler fixture via the driver package.
//
// Run all suites:
//
//	go test ./test/e2e/...
package e2e

import (
	"testing"

	// Import to register all VM engines and bundler+renderer combos.
	_ "gojsx/test/combo"
	_ "gojsx/test/vm"
	"gojsx/test/e2e/suite"
)

func TestHTMLShell(t *testing.T)                { suite.RunHTMLShellTests(t) }
func TestServerComponentRendering(t *testing.T) { suite.RunServerComponentRenderingTests(t) }
func TestNotFoundHandling(t *testing.T)         { suite.RunNotFoundHandlingTests(t) }
func TestLayoutComposition(t *testing.T)        { suite.RunLayoutCompositionTests(t) }
func TestFlightProtocol(t *testing.T)           { suite.RunFlightProtocolTests(t) }
func TestErrorBoundary(t *testing.T)            { suite.RunErrorBoundaryTests(t) }
func TestClientBundle(t *testing.T)             { suite.RunClientBundleTests(t) }
func TestConcurrentRendering(t *testing.T)      { suite.RunConcurrentRenderingTests(t) }
