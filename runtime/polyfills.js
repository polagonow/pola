/**
 * runtime/polyfills.js — Goja VM polyfills.
 *
 * Loaded first in the server bundle, before the esbuild CJS output.
 * Provides Web APIs that react-server-dom-webpack/server.browser expects
 * in a browser-like environment, plus __webpack_require__ for the Flight
 * encoder's client-reference module loading.
 */

// ─── TextEncoder / TextDecoder ────────────────────────────────────────────────

var TextEncoder = (function () {
  function T() {}
  T.prototype.encode = function (s) {
    s = String(s);
    var b = [];
    for (var i = 0; i < s.length; i++) {
      var c = s.charCodeAt(i);
      if (c < 0x80) {
        b.push(c);
      } else if (c < 0x800) {
        b.push(0xC0 | (c >> 6));
        b.push(0x80 | (c & 0x3F));
      } else {
        b.push(0xE0 | (c >> 12));
        b.push(0x80 | ((c >> 6) & 0x3F));
        b.push(0x80 | (c & 0x3F));
      }
    }
    return new Uint8Array(b);
  };
  T.prototype.encodeInto = function (s, dest) {
    var e = this.encode(s);
    var w = Math.min(e.length, dest.length);
    for (var i = 0; i < w; i++) dest[i] = e[i];
    return { read: s.length, written: w };
  };
  return T;
})();

var TextDecoder = (function () {
  function T(enc) { this.encoding = enc || 'utf-8'; }
  T.prototype.decode = function (buf) {
    if (!buf) return '';
    var a = buf instanceof Uint8Array ? buf : new Uint8Array(buf);
    var s = '';
    for (var i = 0; i < a.length;) {
      var c = a[i++];
      if (c < 0x80) {
        s += String.fromCharCode(c);
      } else if ((c & 0xE0) === 0xC0) {
        s += String.fromCharCode(((c & 0x1F) << 6) | (a[i++] & 0x3F));
      } else {
        s += String.fromCharCode(((c & 0x0F) << 12) | ((a[i++] & 0x3F) << 6) | (a[i++] & 0x3F));
      }
    }
    return s;
  };
  return T;
})();

// ─── Microtask queue ──────────────────────────────────────────────────────────
// React's Flight encoder uses MessageChannel / queueMicrotask to schedule work.
// We capture those callbacks and expose __drainMicrotasks__() for Go to call
// between RunString() invocations to drain them.

var __microtaskQueue__ = [];

var queueMicrotask = function (fn) {
  __microtaskQueue__.push(fn);
};

function __drainMicrotasks__() {
  var safety = 0;
  while (__microtaskQueue__.length > 0 && safety++ < 5000) {
    var batch = __microtaskQueue__.splice(0);
    for (var i = 0; i < batch.length; i++) {
      try { batch[i](); } catch (e) { /* swallow — Flight handles its own errors */ }
    }
  }
}

// ─── MessageChannel ───────────────────────────────────────────────────────────

var MessageChannel = (function () {
  function Port() { this.onmessage = null; }
  Port.prototype.postMessage = function (data) {
    var partner = this._partner;
    __microtaskQueue__.push(function () {
      if (partner && typeof partner.onmessage === 'function') {
        partner.onmessage({ data: data });
      }
    });
  };
  Port.prototype.close = function () {};

  function MC() {
    this.port1 = new Port();
    this.port2 = new Port();
    this.port1._partner = this.port2;
    this.port2._partner = this.port1;
  }
  return MC;
})();

// ─── ReadableStream ───────────────────────────────────────────────────────────
// react-server-dom-webpack/server.browser returns a ReadableStream of
// Uint8Array chunks containing RSC Flight wire format text.

var ReadableStream = (function () {
  function Controller(stream) {
    this._stream = stream;
    this._chunks = [];
    this._closed = false;
    this._error = null;
  }
  Controller.prototype.enqueue = function (chunk) {
    if (!this._closed) this._chunks.push(chunk);
  };
  Controller.prototype.close = function () {
    this._closed = true;
  };
  Controller.prototype.error = function (err) {
    this._error = err;
    this._closed = true;
  };
  Object.defineProperty(Controller.prototype, 'byobRequest', { get: function () { return null; } });
  Object.defineProperty(Controller.prototype, 'desiredSize', {
    get: function () { return this._closed ? 0 : 1; }
  });

  function RS(src) {
    this._controller = new Controller(this);
    this._src = src || {};
    this._started = false;
  }
  RS.prototype._start = function () {
    if (this._started) return;
    this._started = true;
    if (typeof this._src.start === 'function') {
      this._src.start(this._controller);
    }
  };
  RS.prototype._pull = function () {
    if (typeof this._src.pull === 'function') {
      this._src.pull(this._controller);
    }
  };
  return RS;
})();

// __pullStream__(stream) → { chunks: Uint8Array[], done: bool }
// Called by Go (render.go) in a loop until done === true.
function __pullStream__(s) {
  s._start();
  __drainMicrotasks__();
  s._pull();
  __drainMicrotasks__();
  var chunks = s._controller._chunks.splice(0);
  return {
    chunks: chunks,
    done: s._controller._closed && chunks.length === 0,
  };
}

// ─── __webpack_require__ stub ─────────────────────────────────────────────────
// react-server-dom-webpack/server.browser calls __webpack_require__(moduleId)
// when serialising "use client" component references into the Flight stream.
// The module registry is populated at startup from __CLIENT_MANIFEST__
// (injected by esbuild Define in the pages bundle).
//
// In our setup client components are pre-bundled — async chunk loading never
// fires, so __webpack_require__ only needs to return synchronous module exports
// from the manifest registry.

var __webpackModuleRegistry__ = {};

var __webpack_require__ = function (id) {
  if (__webpackModuleRegistry__[id]) {
    return __webpackModuleRegistry__[id];
  }
  // Return a thenable Promise-like that is already "fulfilled: null"
  // so requireAsyncModule() short-circuits immediately.
  var p = { status: 'fulfilled', value: null };
  return p;
};

// __webpack_require__.u maps chunk IDs back to filenames (used by Flight to
// embed chunk URLs in the wire format so the browser knows what to preload).
__webpack_require__.u = function (chunkId) { return chunkId; };

// __webpack_chunk_load__ is called by preloadModule() for async chunks.
// We don't have async chunks — return a resolved Promise immediately.
var __webpack_chunk_load__ = function (chunkId) {
  var p = Promise.resolve();
  p.status = 'fulfilled';
  p.value = undefined;
  return p;
};

// ─── AbortController / AbortSignal ───────────────────────────────────────────
// react-server-dom-webpack/server.browser uses AbortController for request
// cancellation. We provide a minimal no-op implementation.

var AbortSignal = (function () {
  function S() {
    this.aborted = false;
    this.reason = undefined;
    this._listeners = [];
  }
  S.prototype.addEventListener = function (type, fn) {
    if (type === 'abort') this._listeners.push(fn);
  };
  S.prototype.removeEventListener = function (type, fn) {
    this._listeners = this._listeners.filter(function (f) { return f !== fn; });
  };
  S.prototype.throwIfAborted = function () {
    if (this.aborted) throw this.reason;
  };
  return S;
})();

var AbortController = (function () {
  function AC() {
    this.signal = new AbortSignal();
  }
  AC.prototype.abort = function (reason) {
    if (this.signal.aborted) return;
    this.signal.aborted = true;
    this.signal.reason = reason || new Error('AbortError');
    var fns = this.signal._listeners.slice();
    for (var i = 0; i < fns.length; i++) {
      try { fns[i]({ type: 'abort', target: this.signal }); } catch (e) {}
    }
  };
  return AC;
})();
