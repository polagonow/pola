// Package polyfilltest runs the shared polyfill test suite against every
// registered VM. Each VM registers itself via an init() call in its own
// fixture file. Tests call forEachVM to get a fresh, polyfill-enabled context
// for every VM in a table-driven sub-test.
package polyfilltest

import "testing"

// Fixture is a polyfill-enabled JS execution context for one VM.
type Fixture interface {
	// Name returns the VM identifier used as the sub-test name.
	Name() string
	// Enable installs all polyfills into the context.
	Enable() error
	// Eval executes src and returns an error if JS throws.
	Eval(src string) error
}

type vmEntry struct {
	name string
	new  func(t *testing.T) Fixture
}

var vmFactories []vmEntry

// register is called by each fixture file's init().
func register(name string, factory func(t *testing.T) Fixture) {
	vmFactories = append(vmFactories, vmEntry{name, factory})
}

// forEachVM runs fn in a sub-test for every registered VM, providing a fresh
// polyfill-enabled Fixture. The fixture lifetime is scoped to the sub-test.
func forEachVM(t *testing.T, fn func(t *testing.T, f Fixture)) {
	t.Helper()
	for _, e := range vmFactories {
		e := e
		t.Run(e.name, func(t *testing.T) {
			f := e.new(t)
			if err := f.Enable(); err != nil {
				t.Fatal(err)
			}
			fn(t, f)
		})
	}
}
