package serveraction

import (
	"path/filepath"
	"strings"
)

// ModuleID returns a stable module ID for a source file: the path relative to
// the nearest ancestor of absAppDir that contains absPath without a leading
// "../", with the extension stripped and separators normalized to "/". Files
// inside node_modules get a package-relative id.
//
// This is the single source of truth for module IDs across the bundler, the
// server-action registry, and the client server-reference stubs — they must
// agree exactly. The esbuild bundler's computeModuleID delegates here.
func ModuleID(absAppDir, absPath string) string {
	if idx := strings.LastIndex(absPath, "/node_modules/"); idx != -1 {
		id := absPath[idx+len("/node_modules/"):]
		return filepath.ToSlash(strings.TrimSuffix(id, filepath.Ext(id)))
	}
	dir := absAppDir
	for {
		rel, err := filepath.Rel(dir, absPath)
		if err != nil {
			break
		}
		relSlash := strings.ReplaceAll(
			strings.TrimSuffix(rel, filepath.Ext(rel)),
			string(filepath.Separator), "/")
		if !strings.HasPrefix(relSlash, "../") {
			return relSlash
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Base(absPath)
}

// Key returns the registry key for an action export: "moduleId:exportName".
// rari uses ":" as the separator (client component manifests use "#"), keeping
// the two namespaces distinct.
func Key(moduleID, exportName string) string {
	return moduleID + ":" + exportName
}
