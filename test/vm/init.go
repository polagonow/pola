// Package vm registers polyfill VM fixtures for the e2e and polyfill test suites.
// Importing this package (blank or otherwise) triggers each VM's init() and
// calls fixture.RegisterPolyfillVM for each engine.
//
// To add a new VM: create <vmname>.go in this package, call
// fixture.RegisterPolyfillVM from init().
package vm

import _ "github.com/polagonow/pola/fs/osfs"

// errPolyfillFixture is returned by engine-specific init() functions when
// the runtime cannot be created (e.g. unsupported platform).
type errPolyfillFixture struct{ err error }

func (e *errPolyfillFixture) Enable() error       { return e.err }
func (e *errPolyfillFixture) Eval(_ string) error { return e.err }
