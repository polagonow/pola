// Package structs defines the agnostic types shared by all bundler
// implementations (esbuild, Vite, etc.).
package structs

// BundleResult is the output of a successful build.
type BundleResult struct {
	// ServerBundlePath is the on-disk path of the emitted server CJS bundle.
	ServerBundlePath string

	// ServerBundle is the compiled server CJS bundle in memory.
	// Populated alongside ServerBundlePath so callers can skip os.ReadFile.
	ServerBundle []byte

	// ClientFiles maps relative output path → bytes written to OutDir.
	ClientFiles map[string][]byte

	// ClientEntryOutput is the /public/... URL of the compiled client
	// injected as <script type="module" src="..."> in the HTML shell.
	ClientEntryOutput string

	// Manifest is the JSON client component manifest passed to __CLIENT_MANIFEST__
	// in the server bundle. Format: {moduleId: {id, chunks, name, async}}.
	// chunks is set to ["default"] so that react-server-dom-esm/client reads
	// metadata[1] as the export name ("default") rather than a chunk URL.
	Manifest []byte

	// ImportURLs maps module IDs to their browser-loadable chunk URLs.
	// Used by the HTML shell's <script type="importmap"> so that
	// import("components/Counter") resolves to /public/Counter-HASH.js.
	ImportURLs map[string]string
}

// BundlerConfig controls both build passes.
type BundlerConfig struct {
	// AppDir is the root of the app/ directory (absolute or relative to cwd).
	AppDir string

	// OutDir is where client bundles are written (e.g. "./public").
	OutDir string

	// ClientEntry is app/_client.tsx — the browser bootstrap.
	ClientEntry string

	// ServerEntry is the path where the server CJS bundle is emitted.
	// Defaults to filepath.Dir(OutDir)/_server.js when empty.
	ServerEntry string

	// ClientComponents are all "use client" TSX files.
	ClientComponents []string

	// External packages to mark external in all builds.
	External []string

	// AssetsURLPath is the URL prefix for client bundle files served by the
	// static file handler (e.g. "/public/assets"). No trailing slash.
	// Defaults to "/public/assets" when empty.
	AssetsURLPath string

	// Dev enables development mode: real error messages are preserved in the
	// RSC Flight stream and bundles are not minified.
	Dev bool

	// ServerEntryContent is the pre-generated server entry TypeScript source
	// produced by ServerEntryGenerator.Generate.
	ServerEntryContent string

	// ServerBundleConditions are the esbuild conditions for the server pass.
	ServerBundleConditions []string

	// ServerBundleDefines are additional esbuild defines merged into the server
	// pass (e.g. {"__DEV__": "true"}). __CLIENT_MANIFEST__ is always injected
	// from the manifest produced by the client pass.
	ServerBundleDefines map[string]string
}
