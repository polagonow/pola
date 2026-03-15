// Package framework defines the pluggable interfaces for the GoJSX SSR
// framework. Every concrete technology choice (esbuild, Goja, React, etc.)
// implements one or more of these interfaces so the framework layer never
// imports any specific bundler, VM, or renderer package.
package framework

import (
	"context"
	"net/http"

	"gojsx/framework/contract"
)

// ── Discovery ────────────────────────────────────────────────────────────────

// Discoverer walks an app directory and returns all pages, client components,
// and global companion files. Implementations encode the file-naming convention
// (e.g. Next.js-style page.tsx, SvelteKit-style +page.svelte, etc.).
type Discoverer interface {
	Discover(appDir string) (contract.DiscoveryResult, error)
}

// ── Build ────────────────────────────────────────────────────────────────────

// Bundler compiles server and client bundles from discovered source files.
// Implementations own all bundler-specific options (plugins, conditions, format).
type Bundler interface {
	Bundle(input contract.BundleInput) (contract.BundleOutput, error)
}

// ServerEntryGenerator produces the in-memory server-entry source code that
// the Bundler compiles into the VM bundle. Implementations encode all
// framework-specific assumptions: imports, page wrappers, render entry point.
type ServerEntryGenerator interface {
	// Generate returns TypeScript/JavaScript source for the server entry.
	Generate(cfg EntryGenConfig) (string, error)

	// BundleConditions returns the esbuild Conditions (or equivalent) for
	// the server bundle. e.g. ["react-server", "browser", "module", "default"]
	BundleConditions() []string

	// ServerGlobalName is the JS global the StreamProtocol calls to start a
	// render. e.g. "__render__" for React RSC.
	ServerGlobalName() string
}

// EntryGenConfig is the input to ServerEntryGenerator.Generate.
type EntryGenConfig struct {
	Pages              []contract.PageEntry
	AppDir             string
	GlobalErrorPath    string
	GlobalNotFoundPath string
	ManifestJSON       string // injected as a JS define value
}

// RouteBuilder converts a discovered PageEntry into a Route (URL pattern +
// JS export name). Implementations encode the routing convention.
type RouteBuilder interface {
	Build(appDir string, page contract.PageEntry) contract.Route
}

// ── VM ───────────────────────────────────────────────────────────────────────

// VMInitContext is the opaque handle passed to PolyfillRegistry.Install.
// Concrete VM factories provide a VMInitContext that carries the underlying
// runtime reference. Polyfill packages type-assert it to the concrete type.
type VMInitContext interface {
	// SetGlobal sets a named global in the VM to the given value.
	// The accepted value types depend on the concrete implementation.
	SetGlobal(name string, value any) error

	// RunScript executes a JS snippet for polyfills easier to write in JS.
	RunScript(source string) error
}

// StreamHandle is the VM-side handle to an in-progress render stream.
// It is opaque to the framework layer; only the paired StreamProtocol knows
// how to consume it.
type StreamHandle interface {
	IsNil() bool
}

// VM is a single JavaScript runtime instance. Callers must not share a VM
// across goroutines; the VMPool guarantees one goroutine holds a VM at a time.
type VM interface {
	// SetRequestContext injects per-request data as a global in the VM.
	SetRequestContext(ctx map[string]any) error

	// SetBridgeFunctions installs the per-render Go→JS function table.
	SetBridgeFunctions(funcs map[string]contract.GoFunc) error

	// CallRenderFunction invokes the framework's named render entry point
	// with serialised props JSON and returns a stream handle.
	CallRenderFunction(exportName, propsJSON string) (StreamHandle, error)

	// ClearState resets all per-request globals so the VM can be pooled.
	ClearState() error
}

// VMPool manages a pool of pre-warmed VM instances.
type VMPool interface {
	Acquire() VM
	Release(vm VM)
}

// VMFactory creates and initialises a new VM instance. Implementations
// pre-compile the server bundle once (e.g. in a constructor) and create
// individual VM instances on each New call.
type VMFactory interface {
	New(bridge contract.BridgeConfig) (VM, error)
}

// PolyfillRegistry installs Web API polyfills into a VM during initialisation.
// Implementations are paired with a specific VM type.
type PolyfillRegistry interface {
	Install(ctx VMInitContext) error
}

// ── Streaming ────────────────────────────────────────────────────────────────

// StreamWriter is the destination for rendered output chunks.
type StreamWriter interface {
	WriteRaw(p []byte) (int, error)
	Flush()
}

// StreamProtocol knows how to pull rendered chunks from a VM's StreamHandle
// and write them to a StreamWriter. Implementations are paired 1:1 with a
// VM type and a renderer framework.
type StreamProtocol interface {
	// Drain reads all chunks from handle and writes them to w until done.
	Drain(vm VM, handle StreamHandle, w StreamWriter) error

	// ContentType is the MIME type for this protocol's wire format.
	// React RSC: "text/x-component", Svelte/Vue streaming: "text/html"
	ContentType() string

	// IsStreamingRequest reports whether r should receive the raw wire-format
	// stream (true) or the HTML shell (false).
	IsStreamingRequest(r *http.Request) bool
}

// ── Renderer ─────────────────────────────────────────────────────────────────

// Renderer drives a single page render using the VM pool.
type Renderer interface {
	Render(ctx context.Context, w StreamWriter, opts contract.RenderOpts) error
}

// ── Routing ──────────────────────────────────────────────────────────────────

// Router matches incoming request paths to registered routes.
type Router interface {
	Register(route contract.Route)
	Match(path string) (*contract.Route, map[string]any)
}

// ── HTML shell ───────────────────────────────────────────────────────────────

// HTMLShell renders the initial HTML document sent to the browser.
type HTMLShell interface {
	Render(p contract.ShellParams) string
}

// ── Assets ───────────────────────────────────────────────────────────────────

// AssetServer handles HTTP requests for static/compiled assets.
type AssetServer interface {
	// Handler returns an http.Handler for assets under the given URL prefix.
	Handler(prefix string) http.Handler
}
