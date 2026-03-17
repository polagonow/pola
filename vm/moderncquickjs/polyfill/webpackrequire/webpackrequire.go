// Package webpackrequire stubs the webpack module system globals for the
// modernc.org/quickjs VM.
package webpackrequire

import (
	"fmt"

	mquickjs "modernc.org/quickjs"
)

const src = `
(function() {
	var registry = {};
	globalThis.__webpackModuleRegistry__ = registry;

	function webpackRequire(id) {
		if (registry[id] !== undefined) return registry[id];
		// Return a thenable that is already "fulfilled: null" so
		// requireAsyncModule() in the Flight encoder short-circuits.
		return { status: 'fulfilled', value: null };
	}
	webpackRequire.u = function(chunkId) { return chunkId; };

	globalThis.__webpack_require__ = webpackRequire;

	globalThis.__webpack_chunk_load__ = function(chunkId) {
		return Promise.resolve();
	};
})();
`

// Enable installs __webpack_require__, __webpack_chunk_load__, and
// __webpackModuleRegistry__ as globals into vm.
func Enable(vm *mquickjs.VM) error {
	if _, err := vm.Eval(src, mquickjs.EvalGlobal); err != nil {
		return fmt.Errorf("webpackrequire: %w", err)
	}
	return nil
}
