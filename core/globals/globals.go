// Package globals centralizes the framework's well-known JS global names.
//
// These are referenced across multiple VM/bundler/renderer implementations.
// Keeping them here avoids drift and makes renames deliberate.
package globals

// Core framework globals (request context + bridge + streaming).
const (
	// RequestContext holds per-request data injected by the VM before render.
	// Written by VM.SetRequestContext in each VM implementation; read by app code
	// (and typed in `ui/packages/jsi/index.ts`).
	RequestContext = "__REQUEST__"

	// BridgeObject is the per-request Go→JS dependency injection function table object.
	// Created during VM init (empty object) and populated per request by
	// RuntimeInjector.Inject; read by server components as `__DEPENDENCY_INJECTION__.fnName(...)`.
	BridgeObject = "__DEPENDENCY_INJECTION__"

	// StreamHandle is where streaming VMs stash the current RSC ReadableStream.
	// Used by v8go (and cleared by multiple VMs) so the Go side can call into
	// PullStreamFn repeatedly without passing opaque handles through Go↔JS.
	StreamHandle = "__pola_stream__"

	// OutputChunk is a per-request Go function used by non-streaming VMs to
	// receive decoded chunks from JS during the async render loop.
	// Installed for the duration of DrainStream and then cleared.
	OutputChunk = "__outputChunk__"
)

// React RSC render entry + stream pulling (polyfilled).
const (
	// RenderFn is the server bundle entrypoint global assigned by the renderer's
	// ServerEntryGenerator (React RSC uses `render/react/discovery/.../entry.go`).
	// VMs call this to start a render and obtain a ReadableStream.
	RenderFn = "__render__"

	// PullStreamFn is installed by the ReadableStream polyfill and drives a single
	// pull iteration, returning `{ chunks, done }` to the Go polling loop.
	PullStreamFn = "__pullStream__"
)

// Microtask queue helpers (polyfilled).
const (
	// DrainMicrotasksFn is installed by the microtask polyfill and is called by
	// stream-pulling loops to deterministically flush promise continuations.
	DrainMicrotasksFn = "__drainMicrotasks__"

	// RunMicrotasksFn is used by the Goja-only Suspense helper to prompt Goja's
	// internal microtask processing during Promise resolution.
	RunMicrotasksFn = "__runMicrotasks__"
)

// Shell extraction (document props from root layout).
const (
	// ExtractShellFn is the global function that calls the root layout component,
	// serializes its React element tree to HTML, and returns it as a string.
	// Used at startup to extract document-level props (html/body attrs, head elements).
	ExtractShellFn = "__extractShell__"
)

// Client component manifest injected as a bundler define.
const (
	// ClientManifest is injected into the server bundle by the bundler (esbuild
	// Define) and consumed by React's `renderToReadableStream(..., __CLIENT_MANIFEST__)`.
	ClientManifest = "__CLIENT_MANIFEST__"
)

// PolaLogFn is the standard Go-backed console bridge installed by every engine
// that uses the ConsoleBridge polyfill. Its signature is (level, msg string).
const PolaLogFn = "__pola_log__"

// QuickJS console helper used by qjs/quickjsgo (kept for backward compatibility).
// Deprecated: use PolaLogFn instead.
const (
	// QJSLogFn is a Go-backed function used to implement `console.*` in qjs and
	// quickjsgo (a minimal console polyfill).
	QJSLogFn = "__qjs_log__"
)

// QuickJS render helper installed by qjs/quickjsgo.
const (
	// RenderAsyncFn is the helper async function installed by qjs/quickjsgo that
	// polls PullStreamFn, calls DrainMicrotasksFn, and forwards each chunk to OutputChunk.
	RenderAsyncFn = "__renderAsync__"
)

// Webpack require polyfill globals used by React Flight encoder.
const (
	// WebpackRequireFn is a minimal `__webpack_require__` shim React Flight expects.
	WebpackRequireFn = "__webpack_require__"
	// WebpackChunkLoadFn is a minimal chunk loader shim React Flight expects.
	WebpackChunkLoadFn = "__webpack_chunk_load__"
	// WebpackModuleRegistry holds the module table used by WebpackRequireFn.
	WebpackModuleRegistry = "__webpackModuleRegistry__"
)

// moderncquickjs-specific bridge + render helper globals.
const (
	// MQJSCallFn is the Go-backed dispatcher moderncquickjs uses for both global
	// and per-request bridge calls.
	MQJSCallFn = "__mqjs_call__"

	// MQJSLogFn is the Go-backed logger used to implement `console.*` in moderncquickjs.
	MQJSLogFn = "__mqjs_log__"

	// MQJSErrorKey is the JSON key moderncquickjs uses to represent an error
	// payload returned from MQJSCallFn.
	MQJSErrorKey = "__mqjs_error__"

	// StartRenderFn stores the active stream/decoder in globals (moderncquickjs).
	StartRenderFn = "__startRender__"
	// PullOnceFn performs one pull iteration and calls OutputChunk for each chunk (moderncquickjs).
	PullOnceFn = "__pullOnce__"
	// ClearRenderStreamFn clears the stored stream/decoder globals (moderncquickjs).
	ClearRenderStreamFn = "__clearRenderStream__"

	// RSCStreamVar holds the active ReadableStream in moderncquickjs render helpers.
	RSCStreamVar = "__rscStream__"
	// RSCDecoderVar holds the active TextDecoder in moderncquickjs render helpers.
	RSCDecoderVar = "__rscDec__"
)
