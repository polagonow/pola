// Package vm registers every VM engine for the e2e and polyfill test suites.
// Importing this package (blank or otherwise) triggers each VM's init() and
// calls fixture.RegisterVM for each engine.
//
// To add a new VM: create <vmname>.go in this package, implement VMFixture,
// and call fixture.RegisterVM from init().
package vm

import _ "gojsx/framework/assets/disk"
