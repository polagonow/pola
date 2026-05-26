// Package combo registers bundler+renderer+engine combinations for the e2e
// test suite. Each additional file in this package registers one combination
// via fixture.Register and is guarded by the required build tags.
//
// To add a new combination: create a new file here, implement AppFixture,
// and call fixture.Register from init(). Tests automatically run every
// registered fixture.
package combo
