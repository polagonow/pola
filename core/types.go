// Package core contains the shared types and interfaces that flow between the
// pluggable modules of the Pola framework (bundler, engine, renderer, router,
// HTML shell, cache, etc.).
//
// This package has minimal external dependencies so every other package can
// import it without creating cycles.
package core

import "time"

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
	AppDir           string   // absolute path to the web app root; used by entry generators
	Pages            []PageEntry
	ClientComponents []string // absolute paths of "use client" files
	GlobalNotFound   string   // abs path to global-not-found.tsx, or ""
	GlobalError      string   // abs path to global-error.tsx, or ""

	// RootLayoutReturnsHTML is true when the root layout file contains <html,
	// indicating it returns a full HTML document (Next.js-style).
	RootLayoutReturnsHTML bool
}

// Route maps a URL pattern to a Server Component export name.
// Pattern supports dynamic segments ("/products/:id"), catch-all ("/shop/:...path"),
// and optional catch-all ("/docs/:...slug?").
type Route struct {
	Pattern string
	Export  string
}

// DocumentProps holds the document-level properties extracted from the root
// layout when it returns <html>. This gives users the same DX as Next.js
// App Router where the root layout controls the full HTML document.
type DocumentProps struct {
	// HTMLAttributes are rendered on the <html> element (e.g. {"lang": "fr", "dir": "rtl"}).
	HTMLAttributes map[string]string `json:"htmlAttributes,omitempty"`
	// BodyAttributes are rendered on the <body> element (e.g. {"class": "dark"}).
	BodyAttributes map[string]string `json:"bodyAttributes,omitempty"`
	// HeadElements are raw HTML strings injected into <head> (e.g. font preconnect links).
	HeadElements []string `json:"headElements,omitempty"`
	// BodyPrefix is HTML rendered before the root div (e.g. a header/nav from the root layout).
	BodyPrefix string `json:"bodyPrefix,omitempty"`
	// BodySuffix is HTML rendered after the root div (e.g. a footer from the root layout).
	BodySuffix string `json:"bodySuffix,omitempty"`
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

	// Stylesheets holds URLs of external CSS files to load via <link> tags
	// (e.g. "/public/assets/styles.css" produced by a CSS processor).
	Stylesheets []string

	// Metadata contains SEO/social metadata rendered as <head> tags.
	// When nil the shell emits no title, meta, or link tags beyond
	// the built-in charset and viewport declarations.
	Metadata *Metadata

	// DocumentProps holds document-level properties extracted from the root
	// layout (html/body attributes, head elements, body prefix/suffix).
	// When nil the shell uses default attributes (lang="en").
	DocumentProps *DocumentProps
}

// RenderOpts controls a single page render.
type RenderOpts struct {
	ExportName     string
	Props          map[string]any
	RequestContext map[string]any
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

	// PublicEnvVars holds POLA_PUBLIC_* environment variables to be injected
	// into the client bundle at build time.
	PublicEnvVars map[string]string

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

	// CSSProcessor is the optional CSS processor (e.g. Tailwind) to use
	// during the client bundle pass. When nil, CSS imports are stubbed.
	CSSProcessor CSS
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

	// CSSURLs are the public URLs of emitted CSS bundles (e.g.
	// "/public/assets/globals-HASH.css"). Empty when no CSS was emitted.
	CSSURLs []string
}

// FSFileInfo describes a single entry returned by FS.ReadDir.
type FSFileInfo struct {
	Name    string
	IsDir   bool
	Size    int64
	ModTime time.Time
}

// CacheOptions controls how a cache entry is stored.
type CacheOptions struct {
	// TTL is the time-to-live for the cache entry. Zero means no expiry.
	TTL time.Duration
}

// InjectionCapability describes a single capability exposed by a RuntimeInjector.
type InjectionCapability struct {
	Name        string
	Description string
}

// RenderRequest is the input to a Renderer.Render call.
type RenderRequest struct {
	Route          Route
	Props          map[string]any
	RequestContext map[string]any
	// Injectors are applied to the VM before rendering, exposing Go services
	// to the JS runtime as __DEPENDENCY_INJECTION__ async functions.
	Injectors []RuntimeInjector
}

// RenderResult is the output of a Renderer.Render call.
type RenderResult struct {
	ContentType string
	Body        []byte
	Streaming   bool
}
