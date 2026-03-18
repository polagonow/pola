// Package enginesuite contains engine-specific e2e test suites.
package enginesuite

import "testing"

// RunPolyfillTests verifies that polyfills are correctly injected per engine.
// TODO: implement engine-parameterized polyfill assertions.
func RunPolyfillTests(t *testing.T) {
	t.Helper()
	t.Skip("polyfill e2e suite not yet implemented")
}
