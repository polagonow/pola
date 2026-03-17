// Package webpack_require stubs the webpack module system globals for the
// Goja VM.
//
// react-server-dom-webpack/server.browser calls __webpack_require__(moduleId)
// when serialising "use client" component references into the Flight stream.
// The registry is populated at startup from __CLIENT_MANIFEST__ (injected by
// esbuild Define). Async chunk loading never fires in our setup.
package webpackrequire

import (
	"github.com/dop251/goja"
)

// Enable installs __webpack_require__, __webpack_chunk_load__, and
// __webpackModuleRegistry__ as globals onto rt.
func Enable(rt *goja.Runtime) {
	registry := rt.NewObject()
	rt.Set("__webpackModuleRegistry__", registry) //nolint:errcheck

	requireFn := rt.ToValue(func(call goja.FunctionCall) goja.Value {
		id := call.Argument(0).String()
		entry := registry.Get(id)
		if entry != nil && !goja.IsUndefined(entry) && !goja.IsNull(entry) {
			return entry
		}
		// Return a thenable that is already "fulfilled: null" so
		// requireAsyncModule() in the Flight encoder short-circuits.
		p := rt.NewObject()
		p.Set("status", "fulfilled") //nolint:errcheck
		p.Set("value", goja.Null())  //nolint:errcheck
		return p
	}).(*goja.Object)

	// __webpack_require__.u(chunkId) → chunkId (identity).
	requireFn.Set("u", rt.ToValue(func(call goja.FunctionCall) goja.Value { //nolint:errcheck
		return call.Argument(0)
	}))

	rt.Set("__webpack_require__", requireFn) //nolint:errcheck

	// __webpack_chunk_load__(chunkId) → resolved Promise.
	rt.Set("__webpack_chunk_load__", func(call goja.FunctionCall) goja.Value { //nolint:errcheck
		p, resolve, _ := rt.NewPromise()
		resolve(goja.Undefined()) //nolint:errcheck
		return rt.ToValue(p)
	})
}
