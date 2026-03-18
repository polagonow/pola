// Package routersuite contains router-specific e2e test suites.
package routersuite

import "testing"

// RunNextJSRouterTests verifies Next.js-style file-system routing behaviour.
// TODO: implement router-parameterized route matching assertions.
func RunNextJSRouterTests(t *testing.T) {
	t.Helper()
	t.Skip("nextjs router e2e suite not yet implemented")
}
