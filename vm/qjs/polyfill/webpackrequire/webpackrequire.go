// Package webpackrequire stubs the webpack module system globals for the
// QuickJS VM.
package webpackrequire

import (
	"fmt"

	qjs "github.com/fastschema/qjs"
)

const src = `
(function() {
	var registry = {};
	globalThis.__webpackModuleRegistry__ = registry;

	function webpackRequire(id) {
		if (registry[id] !== undefined) return registry[id];
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
// __webpackModuleRegistry__ as globals into ctx.
func Enable(ctx *qjs.Context) error {
	val, err := ctx.Eval("webpackrequire.js", qjs.Code(src))
	if val != nil {
		val.Free()
	}
	if err != nil {
		return fmt.Errorf("webpackrequire: %w", err)
	}
	return nil
}
