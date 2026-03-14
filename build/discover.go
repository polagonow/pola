package build

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// PageAlias returns the JS identifier used for p in the generated server entry.
// It walks every directory segment between the "pages" root and the file,
// capitalises each one, and strips dynamic-segment brackets:
//
//	pages/page.tsx               → "Index"
//	pages/products/page.tsx      → "Products"
//	pages/products/[id]/page.tsx → "ProductsId"
func PageAlias(p PageEntry) string {
	base := strings.TrimSuffix(filepath.Base(p.File), filepath.Ext(p.File))
	if base != "page" {
		return titleCase(base)
	}
	// Collect directory segments above the "pages" root.
	dir := filepath.Dir(p.File)
	var segs []string
	for {
		name := filepath.Base(dir)
		parent := filepath.Dir(dir)
		if name == "pages" || parent == dir {
			break
		}
		segs = append([]string{name}, segs...)
		dir = parent
	}
	if len(segs) == 0 {
		return "Index"
	}
	var alias strings.Builder
	for _, s := range segs {
		alias.WriteString(titleCase(stripBrackets(s)))
	}
	return alias.String()
}

func titleCase(s string) string {
	if len(s) == 0 {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func stripBrackets(s string) string {
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		return s[1 : len(s)-1]
	}
	return s
}

// RoutePattern converts a page file path to a URL pattern.
//
//	app/pages/index/page.tsx         → "/"
//	app/pages/products/page.tsx      → "/products"
//	app/pages/products/[id]/page.tsx → "/products/:id"
func RoutePattern(appDir, file string) string {
	pagesDir := filepath.Join(appDir, "pages")
	rel, err := filepath.Rel(pagesDir, filepath.Dir(file))
	if err != nil {
		return "/"
	}
	var parts []string
	for _, seg := range strings.Split(rel, string(filepath.Separator)) {
		if seg == "." || seg == "index" {
			continue
		}
		if strings.HasPrefix(seg, "[") && strings.HasSuffix(seg, "]") {
			parts = append(parts, ":"+seg[1:len(seg)-1])
		} else {
			parts = append(parts, seg)
		}
	}
	return "/" + strings.Join(parts, "/")
}

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
		ok, err := hasDefaultExport(path)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%s: missing \"export default function\"", path)
		}
		pages = append(pages, PageEntry{File: path})
		return nil
	})
	return pages, err
}

// hasDefaultExport reports whether path contains an "export default function"
// or "export default async function" declaration.
func hasDefaultExport(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "export default function") ||
			strings.HasPrefix(line, "export default async function") {
			return true, nil
		}
	}
	return false, sc.Err()
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
