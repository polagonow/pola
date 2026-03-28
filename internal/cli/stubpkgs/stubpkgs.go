// Package stubpkgs embeds the @pola/di and @pola/react source files and
// stubs them into a project's node_modules/@pola/ directory (Prisma-style).
package stubpkgs

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:di all:react
var packages embed.FS

// StubToNodeModules writes @pola/di and @pola/react into
// <projectDir>/node_modules/@pola/. Existing generated.d.ts files are
// preserved so that codegen output is not overwritten.
func StubToNodeModules(projectDir string) error {
	nmDir := filepath.Join(projectDir, "node_modules", "@pola")

	return fs.WalkDir(packages, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		outPath := filepath.Join(nmDir, path)

		// Don't overwrite generated.d.ts if it already exists —
		// codegen may have written real types there.
		if filepath.Base(path) == "generated.d.ts" {
			if _, err := os.Stat(outPath); err == nil {
				return nil
			}
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(outPath), err)
		}

		data, err := fs.ReadFile(packages, path)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", path, err)
		}

		return os.WriteFile(outPath, data, 0o644)
	})
}
