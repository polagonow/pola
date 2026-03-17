// Package fixtures registers every VM for both the e2e and polyfill test
// suites. Importing this package (blank or otherwise) is all that is needed
// to make ForEachApp / ForEachVM aware of all VMs.
//
// To add a new VM: create <vmname>.go in this package, call
// e2efixture.Register and polyfilltest.Register from init(), and done.
package fixtures

import (
	_ "gojsx/bundler/esbuild"
	_ "gojsx/framework/assets/disk"
	_ "gojsx/render/react"
	_ "gojsx/render/react/discovery/nextjs"
	_ "gojsx/render/react/shell"
)
