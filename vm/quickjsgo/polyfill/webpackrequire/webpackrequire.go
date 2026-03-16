// Package webpackrequire stubs the webpack module system globals for the
// QuickJS VM.
//
// react-server-dom-webpack/server.browser calls __webpack_require__(moduleId)
// when serialising "use client" component references into the Flight stream.
// The registry is populated at startup from __CLIENT_MANIFEST__ (injected by
// esbuild Define). Async chunk loading never fires in our setup.
package webpackrequire

import (
	"fmt"

	quickjs "github.com/buke/quickjs-go"
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
// __webpackModuleRegistry__ as globals into ctx.
func Enable(ctx *quickjs.Context) error {
	ret := ctx.Eval(src, quickjs.EvalFileName("webpackrequire.js"))
	defer ret.Free()
	if ret.IsException() {
		return fmt.Errorf("webpackrequire: %w", ctx.Exception())
	}
	return nil
}
