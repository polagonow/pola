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
//	app/page.tsx               → "Index"
//	app/products/page.tsx      → "Products"
//	app/products/[id]/page.tsx → "ProductsId"
func PageAlias(p PageEntry) string {
	base := strings.TrimSuffix(filepath.Base(p.PageComponentPath), filepath.Ext(p.PageComponentPath))
	if base != "page" {
		return titleCase(base)
	}
	// Collect directory segments above the "pages" root.
	dir := filepath.Dir(p.PageComponentPath)
	var segs []string
	for {
		name := filepath.Base(dir)
		parent := filepath.Dir(dir)
		if name == "app" || parent == dir {
			break
		}
		if !isRouteGroup(name) {
			segs = append([]string{name}, segs...)
		}
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
	// [[...slug]] → "slug"
	if strings.HasPrefix(s, "[[...") && strings.HasSuffix(s, "]]") {
		return s[5 : len(s)-2]
	}
	// [...slug] → "slug"
	if strings.HasPrefix(s, "[...") && strings.HasSuffix(s, "]") {
		return s[4 : len(s)-1]
	}
	// [slug] → "slug"
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		return s[1 : len(s)-1]
	}
	return s
}

// isRouteGroup reports whether s is a route group segment (wrapped in parentheses).
func isRouteGroup(s string) bool {
	return len(s) >= 2 && s[0] == '(' && s[len(s)-1] == ')'
}

// stripParens removes surrounding parentheses from a route group name.
func stripParens(s string) string {
	if isRouteGroup(s) {
		return s[1 : len(s)-1]
	}
	return s
}

// isCatchAll reports whether s is a required catch-all segment: [...name]
func isCatchAll(s string) bool {
	return strings.HasPrefix(s, "[...") && strings.HasSuffix(s, "]") && !strings.HasPrefix(s, "[[")
}

// isOptionalCatchAll reports whether s is an optional catch-all segment: [[...name]]
func isOptionalCatchAll(s string) bool {
	return strings.HasPrefix(s, "[[...") && strings.HasSuffix(s, "]]")
}

// RoutePattern converts a page file path to a URL pattern.
//
//	app/index/page.tsx         → "/"
//	app/products/page.tsx      → "/products"
//	app/products/[id]/page.tsx → "/products/:id"
func RoutePattern(appDir, file string) string {
	pagesDir := filepath.Join(appDir, "app")
	rel, err := filepath.Rel(pagesDir, filepath.Dir(file))
	if err != nil {
		return "/"
	}
	var parts []string
	for _, seg := range strings.Split(rel, string(filepath.Separator)) {
		if seg == "." || seg == "index" || isRouteGroup(seg) {
			continue
		}
		if isOptionalCatchAll(seg) {
			// [[...name]] → :...name?  (must be last; consumes all remaining segments optionally)
			name := seg[5 : len(seg)-2]
			parts = append(parts, ":..."+name+"?")
			break
		}
		if isCatchAll(seg) {
			// [...name] → :...name  (must be last; consumes all remaining segments)
			name := seg[4 : len(seg)-1]
			parts = append(parts, ":..."+name)
			break
		}
		if strings.HasPrefix(seg, "[") && strings.HasSuffix(seg, "]") {
			parts = append(parts, ":"+seg[1:len(seg)-1])
		} else {
			parts = append(parts, seg)
		}
	}
	return "/" + strings.Join(parts, "/")
}

// LayoutAlias returns the JS identifier prefix for a layout file.
// It derives the name from the layout's directory position relative to pagesDir,
// using the same capitalisation rules as PageAlias:
//
//	app/layout.tsx               → "Index"
//	app/products/layout.tsx      → "Products"
//	app/products/[id]/layout.tsx → "ProductsId"
func LayoutAlias(pagesDir, layoutPath string) string {
	dir := filepath.Dir(layoutPath)
	absPagesDir, _ := filepath.Abs(pagesDir)
	var segs []string
	for {
		absDir, _ := filepath.Abs(dir)
		if absDir == absPagesDir {
			break
		}
		name := filepath.Base(dir)
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		segs = append([]string{name}, segs...)
		dir = parent
	}
	if len(segs) == 0 {
		return "Index"
	}
	var b strings.Builder
	for _, s := range segs {
		b.WriteString(titleCase(stripParens(stripBrackets(s))))
	}
	return b.String()
}

// PageSegment represents one directory level in the route hierarchy.
// Both LayoutPath and ErrorPath are optional (empty string = absent).
type PageSegment struct {
	Dir        string // absolute path of this segment's directory
	LayoutPath string // layout.tsx path, or "" if none
	ErrorPath  string // error.tsx path, or "" if none
}

// collectSegments walks from pagesDir down to pageDir and returns one
// PageSegment per directory level (outermost first) that has at least a
// layout.tsx or error.tsx with a default export.
func collectSegments(pagesDir, pageDir string) ([]PageSegment, error) {
	absPagesDir, _ := filepath.Abs(pagesDir)
	absPageDir, _ := filepath.Abs(pageDir)
	rel, err := filepath.Rel(absPagesDir, absPageDir)
	if err != nil {
		return nil, err
	}
	current := absPagesDir
	dirs := []string{current}
	if rel != "." {
		for _, seg := range strings.Split(rel, string(filepath.Separator)) {
			current = filepath.Join(current, seg)
			dirs = append(dirs, current)
		}
	}
	var segments []PageSegment
	for _, d := range dirs {
		var seg PageSegment
		seg.Dir = d
		if candidate := filepath.Join(d, "layout.tsx"); fileExists(candidate) {
			if ok, _ := hasDefaultExport(candidate); ok {
				seg.LayoutPath = candidate
			}
		}
		if candidate := filepath.Join(d, "error.tsx"); fileExists(candidate) {
			if ok, _ := hasDefaultExport(candidate); ok {
				seg.ErrorPath = candidate
			}
		}
		if seg.LayoutPath != "" || seg.ErrorPath != "" {
			segments = append(segments, seg)
		}
	}
	return segments, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DiscoverPages walks appDir/app/ and returns one PageEntry per page.tsx file.
//
// Convention: each route lives in its own subdirectory with a page.tsx file
// that uses export default. The directory name becomes the JS alias and the
// URL segment (e.g. app/products/page.tsx → alias "Products", route /products).
//
// The root route uses app/index/page.tsx → alias "Index", route /.
func DiscoverPages(appDir string) ([]PageEntry, error) {
	pagesDir := filepath.Join(appDir, "app")
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
		segs, segErr := collectSegments(pagesDir, filepath.Dir(path))
		if segErr != nil {
			return segErr
		}
		entry := PageEntry{
			PageComponentPath: path,
			Segments:          segs,
		}
		entry.LoadingComponentPath, entry.NotFoundComponentPath =
			discoverCompanions(filepath.Dir(path))
		pages = append(pages, entry)
		return nil
	})
	return pages, err
}

// discoverCompanions checks the given page directory for optional co-located
// companion files (loading.tsx, not-found.tsx) that have a default export.
// Returns the absolute path for each companion found, or "" if absent.
// Layout and error files are handled per-segment via collectSegments.
func discoverCompanions(pageDir string) (loading, notFound string) {
	for _, name := range []string{"loading.tsx", "not-found.tsx"} {
		candidate := filepath.Join(pageDir, name)
		if !fileExists(candidate) {
			continue
		}
		ok, _ := hasDefaultExport(candidate)
		if !ok {
			continue
		}
		switch name {
		case "loading.tsx":
			loading = candidate
		case "not-found.tsx":
			notFound = candidate
		}
	}
	return
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
			strings.HasPrefix(line, "export default async function") ||
			strings.HasPrefix(line, "export default class") {
			return true, nil
		}
	}
	return false, sc.Err()
}

// GlobalComponents holds the optional global-level components found at the
// app/ root (not tied to any route).
type GlobalComponents struct {
	// NotFoundPath is the path to global-not-found.tsx, or "" if absent.
	NotFoundPath string
	// ErrorPath is the path to global-error.tsx ("use client"), or "" if absent.
	ErrorPath string
}

// DiscoverGlobalComponents checks appDir/app/ for global-not-found.tsx and
// global-error.tsx with a default export, returning their paths.
func DiscoverGlobalComponents(appDir string) (GlobalComponents, error) {
	pagesDir := filepath.Join(appDir, "app")
	var gc GlobalComponents
	for _, item := range []struct {
		name string
		dest *string
	}{
		{"global-not-found.tsx", &gc.NotFoundPath},
		{"global-error.tsx", &gc.ErrorPath},
	} {
		candidate := filepath.Join(pagesDir, item.name)
		if !fileExists(candidate) {
			continue
		}
		if ok, _ := hasDefaultExport(candidate); ok {
			*item.dest = candidate
		}
	}
	return gc, nil
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
