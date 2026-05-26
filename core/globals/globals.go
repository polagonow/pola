// Package globals centralizes the framework's well-known JS global names.
//
// These are referenced across multiple VM/bundler/renderer implementations.
// Keeping them here avoids drift and makes renames deliberate.
//
// React-specific globals (SSRData, RootElementID, ExtractShellFn,
// ClientManifestDefine, RunMicrotasksFn) live in renderer/react.
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

	// StreamHandle is where streaming VMs stash the current ReadableStream.
	// Used by v8go (and cleared by multiple VMs) so the Go side can call into
	// PullStreamFn repeatedly without passing opaque handles through Go↔JS.
	StreamHandle = "__pola_stream__"

	// OutputChunk is a per-request Go function used by non-streaming VMs to
	// receive decoded chunks from JS during the async render loop.
	// Installed for the duration of DrainStream and then cleared.
	OutputChunk = "__outputChunk__"
)

// Render entry + stream pulling (polyfilled).
const (
	// RenderFn is the server bundle entrypoint global assigned by the renderer's
	// ServerEntryGenerator. VMs call this to start a render and obtain a
	// ReadableStream.
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
)

// PolaLogFn is the standard Go-backed console bridge installed by every engine
// that uses the ConsoleBridge polyfill. Its signature is (level, msg string).
const PolaLogFn = "__pola_log__"
