// Package bundler defines the agnostic types and utilities shared by all
// bundler implementations (esbuild, Vite, etc.).
package bundler

import (
	"path/filepath"
	"strings"
)

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

// ClientManifestEntry is a single entry in the webpack-format client manifest.
type ClientManifestEntry struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Chunks []string `json:"chunks"`
	Async  bool     `json:"async"`
}

// BuildManifest returns two things:
//
//  1. manifest — flat webpack-format map for __CLIENT_MANIFEST__ in the server
//     bundle.  react-server-dom-webpack/server emits I rows as the 3-element
//     array [id, chunks, name]. react-server-dom-esm/client reads that array
//     as [specifier, name] — so metadata[1] (chunks) is used as the export
//     name on the loaded module. Setting chunks=["default"] makes the ESM
//     client call moduleExports["default"] which is the correct export.
//
//  2. importURLs — moduleId → real chunk URL, used by the HTML import map so
//     the browser can resolve import("components/Counter") to the hashed file.
func BuildManifest(clientComponents []string, clientFiles map[string][]byte, appDir string, assetsURLPath string, inputChunkURLs map[string]string) (map[string]ClientManifestEntry, map[string]string, error) {
	if assetsURLPath == "" {
		assetsURLPath = "/public/assets"
	}
	absAppDir, _ := filepath.Abs(appDir)
	m := make(map[string]ClientManifestEntry)
	importURLs := make(map[string]string)

	for _, src := range clientComponents {
		absSrc, _ := filepath.Abs(src)
		id := ComputeModuleID(absAppDir, absSrc)
		base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))

		// Find the real chunk URL for the import map.
		// Prefer the exact metafile-based mapping; fall back to filename heuristic.
		chunkURL := assetsURLPath + "/main.js"
		if url, ok := inputChunkURLs[absSrc]; ok {
			chunkURL = url
		} else {
			for out := range clientFiles {
				b := filepath.Base(out)
				if strings.HasPrefix(b, base+"-") || b == base+".js" {
					chunkURL = assetsURLPath + "/" + out
					break
				}
			}
		}
		importURLs[id] = chunkURL

		// React RSC looks up the manifest using "moduleId#exportName".
		// The proxy stub is CJS (module.exports = createClientModuleProxy(id)),
		// so React encodes an empty export name → key = id + "#".
		// We also register id + "#default" for any ESM default-import paths.
		entry := ClientManifestEntry{ID: id, Name: "default", Chunks: []string{"default"}}
		m[id+"#"] = entry
		m[id+"#default"] = entry
	}
	return m, importURLs, nil
}

// ComputeModuleID returns a stable module ID for a client component file.
// Files inside node_modules get a package-path id (e.g. "@gojsx/react-renderer/components/ErrorBoundary").
// App files get a relative path id (e.g. "components/ThemeToggle").
func ComputeModuleID(absAppDir, absPath string) string {
	if idx := strings.LastIndex(absPath, "/node_modules/"); idx != -1 {
		id := absPath[idx+len("/node_modules/"):]
		return filepath.ToSlash(strings.TrimSuffix(id, filepath.Ext(id)))
	}
	rel, err := filepath.Rel(absAppDir, absPath)
	if err != nil {
		return filepath.Base(absPath)
	}
	return strings.ReplaceAll(strings.TrimSuffix(rel, filepath.Ext(rel)), string(filepath.Separator), "/")
}
