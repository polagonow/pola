// Package project provides shared utilities for locating the user's Pola project.
package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ModulePath reads the module path from go.mod in the given directory.
func ModulePath(dir string) (string, error) {
	data, err := readGoMod(dir)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}
	return "", fmt.Errorf("module directive not found in go.mod")
}

// GoVersion reads the go version directive from go.mod in the given directory.
// Returns "1.25.0" as fallback if not found.
func GoVersion(dir string) string {
	data, err := readGoMod(dir)
	if err != nil {
		return "1.25.0"
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "go ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "go"))
		}
	}
	return "1.25.0"
}

func readGoMod(dir string) ([]byte, error) {
	return os.ReadFile(filepath.Join(dir, "go.mod"))
}

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
