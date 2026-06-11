package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/polagonow/pola/polafile"
)

// ensureFrontendDeps installs frontend dependencies if they haven't been
// installed yet. It checks for package-manager sentinel files (.pnpm,
// .package-lock.json, .yarn-integrity) rather than just the node_modules
// directory, since stub packages may have created node_modules/@pola without
// a real install.
func ensureFrontendDeps(webDir string, pf polafile.Polafile) error {
	if _, err := os.Stat(filepath.Join(webDir, "package.json")); err != nil {
		return nil
	}
	if depsInstalled(webDir) {
		return nil
	}

	pm := pf.PackageManager
	if pm == "" {
		pm = detectPackageManager()
	}
	if i := strings.IndexByte(pm, '@'); i > 0 {
		pm = pm[:i]
	}
	fmt.Printf("Installing frontend dependencies (%s install)...\n", pm)
	if err := runInDir(webDir, pm, "install"); err != nil {
		return fmt.Errorf("%s install: %w", pm, err)
	}
	return nil
}

// depsInstalled reports whether a real package-manager install has been run
// inside webDir. It looks for sentinel files that only package managers create.
func depsInstalled(webDir string) bool {
	sentinels := []string{
		filepath.Join("node_modules", ".pnpm"),              // pnpm
		filepath.Join("node_modules", ".package-lock.json"), // npm
		filepath.Join("node_modules", ".yarn-integrity"),    // yarn
	}
	for _, s := range sentinels {
		if _, err := os.Stat(filepath.Join(webDir, s)); err == nil {
			return true
		}
	}
	return false
}
