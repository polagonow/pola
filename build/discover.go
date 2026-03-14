package build

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DiscoverPages walks appDir/pages/ and returns one PageEntry per page.tsx file.
//
// Convention: each route lives in its own subdirectory with a page.tsx file
// that uses export default. The directory name becomes the JS alias and the
// URL segment (e.g. pages/products/page.tsx → alias "Products", route /products).
//
// The root route uses pages/index/page.tsx → alias "Index", route /.
func DiscoverPages(appDir string) ([]PageEntry, error) {
	pagesDir := filepath.Join(appDir, "pages")
	var pages []PageEntry

	err := filepath.WalkDir(pagesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "page.tsx" {
			return nil
		}
		pages = append(pages, PageEntry{File: path, Export: "default"})
		return nil
	})
	return pages, err
}

// DiscoverClientComponents walks appDir/components/ recursively and returns
// the path of every .tsx/.ts file whose first non-empty line is "use client".
func DiscoverClientComponents(appDir string) ([]string, error) {
	componentsDir := filepath.Join(appDir, "components")
	var paths []string
	err := filepath.WalkDir(componentsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !isTSFile(d.Name()) {
			return nil
		}
		ok, err := hasUseClient(path)
		if err != nil {
			return err
		}
		if ok {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}

// isTSFile reports whether the filename has a .ts or .tsx extension.
func isTSFile(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".ts" || ext == ".tsx"
}

// hasUseClient reports whether path starts with a "use client" directive.
func hasUseClient(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		line = strings.TrimSuffix(strings.TrimSuffix(line, ";"), ";")
		return line == `"use client"` || line == `'use client'`, nil
	}
	return false, nil
}
