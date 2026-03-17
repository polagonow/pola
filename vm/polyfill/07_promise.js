// Replaces the native QuickJS Promise with a pure-JS implementation that
// routes all continuations through queueMicrotask so __drainMicrotasks__()
// flushes them in the render loop.
//
// modernc.org/quickjs exposes no JS_ExecutePendingJob API, so native
// Promise.then callbacks would never fire without this polyfill.
//
// Requires: queueMicrotask (01_microtask.js loaded first)
(function () {
	if (Promise) return;
	var PENDING = 0, FULFILLED = 1, REJECTED = 2;

	function Promise(executor) {
		this._state = PENDING;
		this._value = undefined;
		this._handlers = [];
		var self = this;
		try {
			executor(
				function (v) { promiseResolve(self, v); },
				function (r) { promiseReject(self, r); }
			);
		} catch (e) {
			promiseReject(self, e);
		}
	}

	function promiseResolve(p, value) {
		if (p._state !== PENDING) return;
		if (value === p) {
			promiseReject(p, new TypeError('Promise resolved with itself'));
			return;
		}
		if (value && (typeof value === 'object' || typeof value === 'function')) {
			var then;
			try { then = value.then; } catch (e) { promiseReject(p, e); return; }
			if (typeof then === 'function') {
				var called = false;
				queueMicrotask(function () {
					try {
						then.call(value,
							function (v) { if (!called) { called = true; promiseResolve(p, v); } },
							function (r) { if (!called) { called = true; promiseReject(p, r); } }
						);
					} catch (e) {
						if (!called) { called = true; promiseReject(p, e); }
					}
				});
				return;
			}
		}
		p._state = FULFILLED;
		p._value = value;
		promiseFlush(p);
	}

	function promiseReject(p, reason) {
		if (p._state !== PENDING) return;
		p._state = REJECTED;
		p._value = reason;
		promiseFlush(p);
	}

	function promiseFlush(p) {
		var hs = p._handlers;
		p._handlers = [];
		for (var i = 0; i < hs.length; i++) promiseHandle(p, hs[i]);
	}

	function promiseHandle(p, h) {
		queueMicrotask(function () {
			var cb = p._state === FULFILLED ? h.onFulfilled : h.onRejected;
			if (typeof cb !== 'function') {
				if (p._state === FULFILLED) promiseResolve(h.promise, p._value);
				else promiseReject(h.promise, p._value);
				return;
			}
			try {
				promiseResolve(h.promise, cb(p._value));
			} catch (e) {
				promiseReject(h.promise, e);
			}
		});
	}

	Promise.prototype.then = function (onFulfilled, onRejected) {
		var p = new Promise(function () { });
		var h = { promise: p, onFulfilled: onFulfilled, onRejected: onRejected };
		if (this._state === PENDING) {
			this._handlers.push(h);
		} else {
			promiseHandle(this, h);
		}
		return p;
	};

	Promise.prototype['catch'] = function (onRejected) {
		return this.then(undefined, onRejected);
	};

	Promise.prototype['finally'] = function (onFinally) {
		return this.then(
			function (v) { return Promise.resolve(onFinally()).then(function () { return v; }); },
			function (r) { return Promise.resolve(onFinally()).then(function () { throw r; }); }
		);
	};

	Promise.resolve = function (v) {
		if (v instanceof Promise) return v;
		return new Promise(function (res) { res(v); });
	};

	Promise.reject = function (r) {
		return new Promise(function (_, rej) { rej(r); });
	};

	Promise.all = function (promises) {
		return new Promise(function (resolve, reject) {
			var n = promises.length;
			if (n === 0) { resolve([]); return; }
			var results = new Array(n);
			var done = 0;
			for (var i = 0; i < n; i++) {
				(function (idx) {
					Promise.resolve(promises[idx]).then(
						function (v) { results[idx] = v; if (++done === n) resolve(results); },
						reject
					);
				})(i);
			}
		});
	};

	Promise.allSettled = function (promises) {
		return new Promise(function (resolve) {
			var n = promises.length;
			if (n === 0) { resolve([]); return; }
			var results = new Array(n);
			var done = 0;
			for (var i = 0; i < n; i++) {
				(function (idx) {
					Promise.resolve(promises[idx]).then(
						function (v) { results[idx] = { status: 'fulfilled', value: v }; if (++done === n) resolve(results); },
						function (r) { results[idx] = { status: 'rejected', reason: r }; if (++done === n) resolve(results); }
					);
				})(i);
			}
		});
	};

	Promise.race = function (promises) {
		return new Promise(function (resolve, reject) {
			for (var i = 0; i < promises.length; i++) {
				Promise.resolve(promises[i]).then(resolve, reject);
			}
		});
	};

	Promise.any = function (promises) {
		return new Promise(function (resolve, reject) {
			var n = promises.length;
			if (n === 0) { reject(new AggregateError([], 'All promises were rejected')); return; }
			var errors = new Array(n);
			var done = 0;
			for (var i = 0; i < n; i++) {
				(function (idx) {
					Promise.resolve(promises[idx]).then(resolve, function (r) {
						errors[idx] = r;
						if (++done === n) reject(new AggregateError(errors, 'All promises were rejected'));
					});
				})(i);
			}
		});
	};

	globalThis.Promise = Promise;
})();
