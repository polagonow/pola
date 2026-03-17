package abortcontroller

// signalSrc is the pure-JS AbortSignal implementation.
const signalSrc = `
(function() {
	function AbortSignal() {
		this.aborted = false;
		this.reason = undefined;
		this._listeners = [];
	}
	AbortSignal.prototype.addEventListener = function(type, fn) {
		if (type !== 'abort') return;
		this._listeners.push(fn);
	};
	AbortSignal.prototype.removeEventListener = function(type, fn) {
		if (type !== 'abort') return;
		this._listeners = this._listeners.filter(function(f) { return f !== fn; });
	};
	AbortSignal.prototype.throwIfAborted = function() {
		if (this.aborted) throw this.reason;
	};
	globalThis.AbortSignal = AbortSignal;
})();
`
