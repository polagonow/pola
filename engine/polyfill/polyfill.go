// Package polyfill defines well-known polyfill IDs and provides the default
// source registry pre-populated with the JavaScript polyfill implementations
// used by the Pola engine backends.
package polyfill

import "github.com/polagonow/pola/core"

// Well-known polyfill IDs.
const (
	Promise         core.PolyfillID = "promise"
	ReadableStream  core.PolyfillID = "readable-stream"
	MessageChannel  core.PolyfillID = "message-channel"
	AbortController core.PolyfillID = "abort-controller"
	MicrotaskQueue  core.PolyfillID = "microtask-queue"
	TextEncoding    core.PolyfillID = "text-encoding"
	WebpackRequire  core.PolyfillID = "webpack-require"
)

// microtaskSrc installs queueMicrotask and __drainMicrotasks__.
const microtaskSrc = `(function () {
	if (typeof globalThis.__drainMicrotasks__ !== 'undefined') { return; }
	var queue = [];

	globalThis.queueMicrotask = function (fn) {
		queue.push(fn);
	};

	globalThis.__drainMicrotasks__ = function () {
		var safety = 0;
		while (queue.length > 0 && safety < 5000) {
			safety++;
			var batch = queue.splice(0);
			for (var i = 0; i < batch.length; i++) {
				try { batch[i](); } catch (e) { }
			}
		}
	};
})();`

// textEncodingSrc installs TextEncoder and TextDecoder.
const textEncodingSrc = `(function() {
	if (typeof globalThis.TextEncoder !== 'undefined') { return; }
	function encodeUTF8(str) {
		var result = [];
		for (var i = 0; i < str.length; ) {
			var cp = str.codePointAt(i);
			if (cp < 0x80) {
				result.push(cp);
				i++;
			} else if (cp < 0x800) {
				result.push(0xC0 | (cp >> 6), 0x80 | (cp & 0x3F));
				i++;
			} else if (cp < 0x10000) {
				result.push(0xE0 | (cp >> 12), 0x80 | ((cp >> 6) & 0x3F), 0x80 | (cp & 0x3F));
				i++;
			} else {
				result.push(
					0xF0 | (cp >> 18),
					0x80 | ((cp >> 12) & 0x3F),
					0x80 | ((cp >> 6) & 0x3F),
					0x80 | (cp & 0x3F)
				);
				i += 2;
			}
		}
		return new Uint8Array(result);
	}

	function TextEncoder() {}
	TextEncoder.prototype.encoding = 'utf-8';
	TextEncoder.prototype.encode = function(str) {
		return encodeUTF8(str == null ? '' : String(str));
	};
	TextEncoder.prototype.encodeInto = function(str, dest) {
		var bytes = encodeUTF8(str == null ? '' : String(str));
		var written = Math.min(bytes.length, dest.length);
		for (var i = 0; i < written; i++) dest[i] = bytes[i];
		return { read: str.length, written: written };
	};

	function TextDecoder(enc) {
		this.encoding = enc || 'utf-8';
	}
	TextDecoder.prototype.decode = function(arr) {
		if (!arr) return '';
		if (arr.buffer) arr = new Uint8Array(arr.buffer, arr.byteOffset, arr.byteLength);
		else if (!(arr instanceof Uint8Array)) arr = new Uint8Array(arr);
		var result = [];
		var i = 0;
		while (i < arr.length) {
			var b = arr[i];
			if (b < 0x80) {
				result.push(String.fromCodePoint(b));
				i++;
			} else if ((b & 0xE0) === 0xC0) {
				result.push(String.fromCodePoint(((b & 0x1F) << 6) | (arr[i+1] & 0x3F)));
				i += 2;
			} else if ((b & 0xF0) === 0xE0) {
				result.push(String.fromCodePoint(
					((b & 0x0F) << 12) | ((arr[i+1] & 0x3F) << 6) | (arr[i+2] & 0x3F)
				));
				i += 3;
			} else {
				var cp = ((b & 0x07) << 18) | ((arr[i+1] & 0x3F) << 12) |
				         ((arr[i+2] & 0x3F) << 6) | (arr[i+3] & 0x3F);
				result.push(String.fromCodePoint(cp));
				i += 4;
			}
		}
		return result.join('');
	};

	globalThis.TextEncoder = TextEncoder;
	globalThis.TextDecoder = TextDecoder;
})();`

// messageChannelSrc installs MessageChannel.
const messageChannelSrc = `(function() {
	if (typeof globalThis.MessageChannel !== 'undefined') { return; }
	var portProto = {
		postMessage: function(data) {
			var partner = this._partner;
			if (!partner) return;
			queueMicrotask(function() {
				if (typeof partner.onmessage === 'function') {
					partner.onmessage({ data: data });
				}
			});
		},
		close: function() {}
	};

	function MessageChannel() {
		var port1 = Object.create(portProto);
		port1.onmessage = null;
		port1._partner = null;

		var port2 = Object.create(portProto);
		port2.onmessage = null;
		port2._partner = null;

		port1._partner = port2;
		port2._partner = port1;

		this.port1 = port1;
		this.port2 = port2;
	}

	globalThis.MessageChannel = MessageChannel;
})();`

// readableStreamSrc installs ReadableStream and __pullStream__.
const readableStreamSrc = `(function() {
	if (typeof globalThis.ReadableStream !== 'undefined') { return; }
	function ReadableStreamController() {
		this._chunks = [];
		this._closed = false;
		this._error = null;
	}
	ReadableStreamController.prototype.enqueue = function(chunk) {
		if (this._closed) return;
		this._chunks.push(chunk);
	};
	ReadableStreamController.prototype.close = function() {
		this._closed = true;
	};
	ReadableStreamController.prototype.error = function(err) {
		this._error = err;
		this._closed = true;
	};
	Object.defineProperty(ReadableStreamController.prototype, 'byobRequest', {
		get: function() { return null; },
		configurable: true
	});
	Object.defineProperty(ReadableStreamController.prototype, 'desiredSize', {
		get: function() { return this._closed ? 0 : 1; },
		configurable: true
	});

	function ReadableStream(src) {
		this._src = src || {};
		this._controller = new ReadableStreamController();
		this._started = false;
	}
	ReadableStream.prototype._start = function() {
		if (this._started) return;
		this._started = true;
		if (typeof this._src.start === 'function') {
			this._src.start(this._controller);
		}
	};
	ReadableStream.prototype._pull = function() {
		if (typeof this._src.pull === 'function') {
			this._src.pull(this._controller);
		}
	};

	globalThis.ReadableStream = ReadableStream;

	globalThis.__pullStream__ = function(s) {
		s._start();
		__drainMicrotasks__();
		s._pull();
		__drainMicrotasks__();
		var chunks = s._controller._chunks.splice(0);
		var closed = s._controller._closed;
		return { chunks: chunks, done: closed && chunks.length === 0 };
	};
})();`

// abortControllerSrc installs AbortSignal and AbortController.
const abortControllerSrc = `(function () {
	if (typeof globalThis.AbortController !== 'undefined') { return; }
	function AbortSignal() {
		this.aborted = false;
		this.reason = undefined;
		this._listeners = [];
	}
	AbortSignal.prototype.addEventListener = function (type, fn) {
		if (type !== 'abort') return;
		this._listeners.push(fn);
	};
	AbortSignal.prototype.removeEventListener = function (type, fn) {
		if (type !== 'abort') return;
		this._listeners = this._listeners.filter(function (f) { return f !== fn; });
	};
	AbortSignal.prototype.throwIfAborted = function () {
		if (this.aborted) throw this.reason;
	};
	globalThis.AbortSignal = AbortSignal;
})();

(function () {
	if (typeof globalThis.AbortController !== 'undefined') { return; }
	function AbortController() {
		this.signal = new AbortSignal();
	}
	AbortController.prototype.abort = function (reason) {
		if (this.signal.aborted) return;
		this.signal.aborted = true;
		if (reason === undefined) {
			try { reason = new Error('AbortError'); } catch (e) { reason = 'AbortError'; }
		}
		this.signal.reason = reason;
		var evt = { type: 'abort', target: this.signal };
		var listeners = this.signal._listeners.slice();
		for (var i = 0; i < listeners.length; i++) {
			try { listeners[i](evt); } catch (e) { }
		}
	};
	globalThis.AbortController = AbortController;
})();`

// promiseSrc installs a pure-JS Promise that routes through queueMicrotask.
const promiseSrc = `(function () {
	if (typeof Promise !== 'undefined') { return; }
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
})();`

// webpackRequireSrc installs __webpack_require__ and __webpack_chunk_load__.
const webpackRequireSrc = `(function() {
	if (typeof globalThis.__webpack_require__ !== 'undefined') { return; }
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
})();`

// DefaultRegistry returns a core.PolyfillRegistry pre-populated with all
// standard Pola polyfills. Each polyfill source is guarded so it only
// installs if the global it provides is not already defined.
func DefaultRegistry() core.PolyfillRegistry {
	reg := core.NewPolyfillRegistry()
	Register(reg)
	return reg
}

// Register populates reg with all standard Pola polyfills.
func Register(reg core.PolyfillRegistry) {
	reg.Register(core.PolyfillSource{ID: MicrotaskQueue, Source: microtaskSrc})
	reg.Register(core.PolyfillSource{ID: TextEncoding, Source: textEncodingSrc})
	reg.Register(core.PolyfillSource{ID: MessageChannel, Source: messageChannelSrc})
	reg.Register(core.PolyfillSource{ID: ReadableStream, Source: readableStreamSrc})
	reg.Register(core.PolyfillSource{ID: AbortController, Source: abortControllerSrc})
	reg.Register(core.PolyfillSource{ID: Promise, Source: promiseSrc})
	reg.Register(core.PolyfillSource{ID: WebpackRequire, Source: webpackRequireSrc})
}
