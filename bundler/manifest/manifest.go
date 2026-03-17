// Package manifest builds the webpack-format client component manifest used by
// react-server-dom-webpack/server and the browser import map.
package manifest

import (
	"path/filepath"
	"strings"
)

// Entry is a single entry in the webpack-format client manifest.
type Entry struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Chunks []string `json:"chunks"`
	Async  bool     `json:"async"`
}

// Build returns two things:
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
func Build(
	clientComponents []string, clientFiles map[string][]byte,
	appDir string, assetsURLPath string, inputChunkURLs map[string]string,
) (map[string]Entry, map[string]string, error) {
	if assetsURLPath == "" {
		assetsURLPath = "/public/assets"
	}
	absAppDir, _ := filepath.Abs(appDir)
	m := make(map[string]Entry)
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
		entry := Entry{ID: id, Name: "default", Chunks: []string{"default"}}
		m[id+"#"] = entry
		m[id+"#default"] = entry
	}
	return m, importURLs, nil
}

// ComputeModuleID returns a stable module ID for a client component file.
// Files inside node_modules get a package-path id (e.g. "@gojsx/react/ErrorBoundary").
// App files get a path id relative to the nearest ancestor of absAppDir that
// contains absPath without a leading "../" (e.g. "components/ThemeToggle" for
// files inside the app dir, "packages/react/components/ErrorBoundary" for
// monorepo sibling packages). This avoids "../"-prefixed ids, which break the
// HTML importmap when the document URL is at a different depth than the
// client bundle's URL.
func ComputeModuleID(absAppDir, absPath string) string {
	if idx := strings.LastIndex(absPath, "/node_modules/"); idx != -1 {
		id := absPath[idx+len("/node_modules/"):]
		return filepath.ToSlash(strings.TrimSuffix(id, filepath.Ext(id)))
	}
	// Walk up from absAppDir until we find an ancestor that contains absPath
	// (i.e. filepath.Rel produces a path without a leading "..").
	dir := absAppDir
	for {
		rel, err := filepath.Rel(dir, absPath)
		if err != nil {
			break
		}
		relSlash := strings.ReplaceAll(strings.TrimSuffix(rel, filepath.Ext(rel)), string(filepath.Separator), "/")
		if !strings.HasPrefix(relSlash, "../") {
			return relSlash
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}
	return filepath.Base(absPath)
}
