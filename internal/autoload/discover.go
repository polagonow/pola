package autoload

import (
	"os"
	"path/filepath"
	"strings"
)

// DiscoverModulePlugins reads go.mod in projectDir and returns the import
// paths of direct dependencies whose module name starts with "pola-".
// These modules are expected to expose a func Plugin() core.Plugin at their
// package root. Returns nil (not an error) if go.mod is missing or contains
// no matching dependencies.
func DiscoverModulePlugins(projectDir string) []string {
	data, err := os.ReadFile(filepath.Join(projectDir, "go.mod"))
	if err != nil {
		return nil
	}

	var (
		inRequire bool
		plugins   []string
	)

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)

		if line == "require (" {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}

		var modPath string
		if inRequire {
			parts := strings.Fields(line)
			if len(parts) >= 2 && !strings.HasPrefix(parts[0], "//") {
				modPath = parts[0]
			}
		} else if strings.HasPrefix(line, "require ") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				modPath = parts[1]
			}
		}

		if modPath == "" {
			continue
		}

		// Match modules whose last path element starts with "pola-".
		base := modPath
		if idx := strings.LastIndex(modPath, "/"); idx >= 0 {
			base = modPath[idx+1:]
		}
		if strings.HasPrefix(base, "pola-") {
			plugins = append(plugins, modPath)
		}
	}
	return plugins
}
