// Package contract contains the shared types that flow between the pluggable
// modules of the GoJSX framework (bundler, VM, renderer, router, HTML shell).
//
// This package has zero external dependencies — only the Go standard library —
// so every other package can import it without creating cycles.
package contract

// GoFunc is the signature for Go functions exposed to the JS VM.
// Arguments are already exported to plain Go values (string, float64, bool,
// map[string]interface{}, []interface{}, etc.) so bridge functions never
// touch VM internals and are safe to call from goroutines.
type GoFunc func(args []interface{}) (any, error)

// BridgeConfig describes Go functions to expose inside the VM.
type BridgeConfig struct {
	// Globals are injected as bare global functions: fetchJSON("url")
	Globals map[string]GoFunc
	// Context functions are injected on a `jsi` object: jsi.getProducts()
	Context map[string]GoFunc
}

// ClientRef is the wire representation of a Client Component reference.
// The browser's RSC runtime resolves this to the actual React component
// by looking up the manifest produced at build time.
type ClientRef struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Chunks []string `json:"chunks"`
	Async  bool     `json:"async"`
}

// ClientManifest maps moduleId → ClientRef.
type ClientManifest map[string]ClientRef

// PageSegment represents one directory level in the route hierarchy.
// Both LayoutPath and ErrorPath are optional (empty string = absent).
type PageSegment struct {
	Dir        string // absolute path of this segment's directory
	LayoutPath string // layout.tsx path, or "" if none
	ErrorPath  string // error.tsx path, or "" if none
}

// PageEntry describes one server-rendered page.
// Every page must use `export default function`.
type PageEntry struct {
	// PageComponentPath is the absolute or cwd-relative path to the page.tsx file.
	PageComponentPath string

	// Segments holds one entry per directory level from the pages/ root (outermost)
	// to the page's own directory (innermost). Each segment may have a LayoutPath
	// and/or ErrorPath.
	Segments []PageSegment

	// LoadingComponentPath is the path to the co-located loading.tsx, or "" if absent.
	LoadingComponentPath string

	// NotFoundComponentPath is the path to the co-located not-found.tsx, or "" if absent.
	NotFoundComponentPath string
}

// ComponentFile identifies a discovered source file with its computed module ID.
type ComponentFile struct {
	// AbsPath is the absolute filesystem path to the component file.
	AbsPath string
	// ModuleID is the path relative to appDir without extension, slash-separated.
	// Used as the key in the client manifest.
	ModuleID string
}

// DiscoveryResult is the output of the Discoverer interface.
type DiscoveryResult struct {
	Pages            []PageEntry
	ClientComponents []string // absolute paths of "use client" files
	GlobalNotFound   string   // abs path to global-not-found.tsx, or ""
	GlobalError      string   // abs path to global-error.tsx, or ""
}

// Route maps a URL pattern to a Server Component export name.
// Pattern supports dynamic segments ("/products/:id"), catch-all ("/shop/:...path"),
// and optional catch-all ("/docs/:...slug?").
type Route struct {
	Pattern string
	Export  string
	Bridge  *BridgeConfig // nil = use pool-level bridge
}

// ShellParams holds the dynamic values needed to render the HTML shell.
type ShellParams struct {
	// ImportURLs maps RSC module IDs to their hashed browser chunk URLs,
	// used to generate the <script type="importmap"> block.
	ImportURLs map[string]string

	// ClientScript is the URL of the compiled client entry module
	// (e.g. "/public/assets/_client-HASH.js").
	ClientScript string

	// Scripts holds bare JS expressions to embed as inline <script> blocks
	// before the client module tag (e.g. "self.__flight_data=...").
	Scripts []string
}

// RenderOpts controls a single page render.
type RenderOpts struct {
	ExportName     string
	Props          map[string]any
	RequestContext map[string]any
	Bridge         *BridgeConfig // nil = use pool's global bridge
}

// BundleInput is the normalised input to the Bundler interface.
// It consolidates the fields of the build-package-specific BundlerConfig.
type BundleInput struct {
	AppDir        string
	OutDir        string
	AssetsURLPath string
	ClientEntry   string // path to _client.tsx
	ServerEntry   string // output path for server bundle; defaults auto-set

	ClientComponents []string // abs paths of "use client" files

	External []string
	Dev      bool

	// ServerEntryContent is the pre-generated server entry TypeScript source
	// produced by ServerEntryGenerator.Generate. The bundler uses this directly
	// instead of generating an entry itself.
	ServerEntryContent string

	// ServerBundleConditions are the esbuild conditions for the server pass
	// (e.g. ["react-server","browser","module","default"] for React RSC).
	ServerBundleConditions []string

	// ServerBundleDefines are additional esbuild defines for the server pass
	// (merged with the bundler's base defines such as __CLIENT_MANIFEST__).
	ServerBundleDefines map[string]string
}

// BundleOutput is the result of a successful build.
type BundleOutput struct {
	// ServerBundle is the compiled server CJS bundle in memory.
	// Passed directly to the VMFactory — no file read needed.
	ServerBundle []byte

	// ClientFiles maps relative output path → file bytes written to OutDir.
	ClientFiles map[string][]byte

	// ClientEntryURL is the /public/... URL of the compiled client entry
	// injected as <script type="module" src="..."> in the HTML shell.
	ClientEntryURL string

	// ManifestJSON is the raw JSON of the client component manifest.
	ManifestJSON []byte

	// ImportURLs maps module IDs to their browser-loadable chunk URLs.
	ImportURLs map[string]string
}
