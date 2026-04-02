// Package project provides shared utilities for locating the user's Pola project.
package project

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindRoot walks up from cwd looking for a go.mod file.
// Falls back to cwd if no go.mod is found.
func FindRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get cwd: %w", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Fall back to cwd.
	return os.Getwd()
}
