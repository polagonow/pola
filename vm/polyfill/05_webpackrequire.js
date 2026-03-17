// Globals: __webpack_require__(id), __webpack_chunk_load__(chunkId),
//          __webpackModuleRegistry__
//
// Go code populates __webpackModuleRegistry__ before running the bundle.
// __webpack_require__ returns a fulfilled thenable for unknown ids so that
// requireAsyncModule() in the React Flight encoder short-circuits cleanly.
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
