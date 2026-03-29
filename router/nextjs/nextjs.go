// Package nextjs provides a Next.js-style file-system router.
// Routes are discovered by scanning the app directory for page files
// matching any of the configured extensions (provided by the active renderer).
package nextjs

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/polagonow/pola/core"
)

// Router is a Next.js-style file-system router that discovers routes by scanning
// the app directory for page files. It implements core.Router.
type Router struct {
	mu        sync.Mutex
	routes    []core.Route
	sorted    bool
	discovery core.DiscoveryResult
}

// New returns a new Router with no routes registered yet.
// Call ScanRoutes to populate it.
func New() *Router {
	return &Router{}
}

// Name returns the router name.
func (r *Router) Name() string { return "nextjs" }

// ScanRoutes walks appDir/app/ (or appDir directly when no app/ subdir exists),
// finds files named page.EXT for any ext in exts, and converts them to URL
// patterns using Next.js conventions. It also detects "use client" files in
// appDir/components/ and populates the internal DiscoveryResult.
func (r *Router) ScanRoutes(ctx context.Context, fsys core.FS, appDir string, exts []string) ([]core.Route, error) {
	// Determine the pages root: prefer appDir/app/, fall back to appDir.
	pagesDir := filepath.Join(appDir, "app")
	if !fsys.Exists(pagesDir) {
		pagesDir = appDir
	}

	extSet := make(map[string]bool, len(exts))
	for _, e := range exts {
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		extSet[e] = true
	}

	var routes []core.Route
	var pages []core.PageEntry

	err := walkDir(fsys, pagesDir, func(path string, info core.FSFileInfo) error {
		if info.IsDir {
			return nil
		}
		base := info.Name
		// Must be named "page.<ext>" for one of the allowed extensions.
		if !isPageFile(base, extSet) {
			return nil
		}

		ok, err := hasDefaultExport(fsys, path)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%s: missing \"export default function\"", path)
		}

		pattern := routePattern(pagesDir, path)
		export := pageExport(pagesDir, path)

		routes = append(routes, core.Route{Pattern: pattern, Export: export})

		// Collect PageEntry for the DiscoveryResult (used by renderers).
		segs, segErr := collectSegments(fsys, pagesDir, filepath.Dir(path), exts)
		if segErr != nil {
			return segErr
		}
		entry := core.PageEntry{
			PageComponentPath: path,
			Segments:          segs,
		}
		entry.LoadingComponentPath, entry.NotFoundComponentPath = discoverCompanions(fsys, filepath.Dir(path), exts)
		pages = append(pages, entry)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Discover client components from appDir/components/.
	clientComponents, err := discoverClientComponents(fsys, appDir, exts)
	if err != nil && !isNotExist(fsys, filepath.Join(appDir, "components")) {
		return nil, err
	}

	// Include error.tsx files from segments as client components (they use "use client").
	seen := make(map[string]bool, len(clientComponents))
	for _, cc := range clientComponents {
		seen[cc] = true
	}
	for _, p := range pages {
		for _, seg := range p.Segments {
			if seg.ErrorPath != "" && !seen[seg.ErrorPath] {
				seen[seg.ErrorPath] = true
				clientComponents = append(clientComponents, seg.ErrorPath)
			}
		}
	}

	// Discover global components (global-not-found, global-error).
	gc := discoverGlobalComponents(fsys, appDir, exts)
	if gc.ErrorPath != "" && !seen[gc.ErrorPath] {
		clientComponents = append(clientComponents, gc.ErrorPath)
	}

	// Check if root layout returns <html> (Next.js-style document control).
	rootLayoutReturnsHTML := detectRootLayoutHTML(fsys, pagesDir, exts)

	r.mu.Lock()
	r.routes = routes
	r.sorted = false
	r.discovery = core.DiscoveryResult{
		AppDir:                appDir,
		Pages:                 pages,
		ClientComponents:      clientComponents,
		GlobalNotFound:        gc.NotFoundPath,
		GlobalError:           gc.ErrorPath,
		RootLayoutReturnsHTML: rootLayoutReturnsHTML,
	}
	r.mu.Unlock()

	return routes, nil
}

// Resolve matches path against the registered routes and returns the matching
// route and extracted URL parameters. Returns (nil, nil) when no route matches.
func (r *Router) Resolve(_ context.Context, path string) (*core.Route, map[string]any) {
	r.mu.Lock()
	if !r.sorted {
		sortRoutes(r.routes)
		r.sorted = true
	}
	routes := r.routes
	r.mu.Unlock()

	for i := range routes {
		if params, ok := MatchPattern(routes[i].Pattern, path); ok {
			return &routes[i], params
		}
	}
	return nil, nil
}

// LoadRoutes seeds the router's internal routing table from a pre-built route
// list (e.g. loaded from prebuild-meta.json in embed mode). This allows
// Resolve to work without calling ScanRoutes first.
func (r *Router) LoadRoutes(routes []core.Route) {
	r.mu.Lock()
	r.routes = routes
	r.sorted = false
	r.mu.Unlock()
}

// DiscoveryResult returns the core.DiscoveryResult populated during the last
// ScanRoutes call. Renderers use this to generate server entries.
func (r *Router) DiscoveryResult() core.DiscoveryResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.discovery
}

// ── FS helpers ───────────────────────────────────────────────────────────────

// walkDir recursively walks a directory using core.FS, calling fn for each entry.
func walkDir(fsys core.FS, root string, fn func(path string, info core.FSFileInfo) error) error {
	entries, err := fsys.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		path := filepath.Join(root, entry.Name)
		if err := fn(path, entry); err != nil {
			return err
		}
		if entry.IsDir {
			if err := walkDir(fsys, path, fn); err != nil {
				return err
			}
		}
	}
	return nil
}

// isNotExist returns true when the given path does not exist on the FS.
func isNotExist(fsys core.FS, path string) bool {
	return !fsys.Exists(path)
}

// hasDefaultExport scans a file for an "export default function/class" line.
func hasDefaultExport(fsys core.FS, path string) (bool, error) {
	data, err := fsys.ReadFile(path)
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "export default function") ||
			strings.HasPrefix(line, "export default async function") ||
			strings.HasPrefix(line, "export default class") {
			return true, nil
		}
	}
	return false, nil
}

// hasUseClient checks whether the first non-empty line of a file is "use client".
func hasUseClient(fsys core.FS, path string) (bool, error) {
	data, err := fsys.ReadFile(path)
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimSuffix(line, ";")
		return line == `"use client"` || line == `'use client'`, nil
	}
	return false, nil
}

// ── path conversion ──────────────────────────────────────────────────────────

// routePattern converts a page file path to a URL pattern.
//
//	app/page.tsx              → /
//	app/posts/page.tsx        → /posts
//	app/posts/[slug]/page.tsx → /posts/:slug
//	app/posts/[...slug]/page.tsx       → /posts/:...slug
//	app/posts/[[...slug]]/page.tsx     → /posts/:...slug?
//	app/(group)/page.tsx      → / (groups stripped)
func routePattern(pagesDir, file string) string {
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
			name := seg[5 : len(seg)-2]
			parts = append(parts, ":..."+name+"?")
			break
		}
		if isCatchAll(seg) {
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

// pageExport derives the component export name from the page's path.
// app/page.tsx                  → Index
// app/posts/page.tsx            → Posts
// app/posts/[slug]/page.tsx     → PostsSlug
// app/(shop)/products/page.tsx  → Products  (groups stripped)
func pageExport(pagesDir, file string) string {
	dir := filepath.Dir(file)
	var segs []string
	for {
		name := filepath.Base(dir)
		parent := filepath.Dir(dir)
		if name == filepath.Base(pagesDir) || parent == dir {
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

// ── segment helpers ──────────────────────────────────────────────────────────

func isPageFile(name string, extSet map[string]bool) bool {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	return base == "page" && extSet[filepath.Ext(name)]
}

func isRouteGroup(s string) bool {
	return len(s) >= 2 && s[0] == '(' && s[len(s)-1] == ')'
}

func isCatchAll(s string) bool {
	return strings.HasPrefix(s, "[...") && strings.HasSuffix(s, "]") && !strings.HasPrefix(s, "[[")
}

func isOptionalCatchAll(s string) bool {
	return strings.HasPrefix(s, "[[...") && strings.HasSuffix(s, "]]")
}

func stripBrackets(s string) string {
	if strings.HasPrefix(s, "[[...") && strings.HasSuffix(s, "]]") {
		return s[5 : len(s)-2]
	}
	if strings.HasPrefix(s, "[...") && strings.HasSuffix(s, "]") {
		return s[4 : len(s)-1]
	}
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		return s[1 : len(s)-1]
	}
	return s
}

func titleCase(s string) string {
	if len(s) == 0 {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func isTSFile(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".ts" || ext == ".tsx"
}

// findWithExts looks for dir/base.ext for each ext in exts, returning the
// first match found. Returns "" if none exist.
func findWithExts(fsys core.FS, dir, base string, exts []string) string {
	for _, ext := range exts {
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		candidate := filepath.Join(dir, base+ext)
		if fsys.Exists(candidate) {
			return candidate
		}
	}
	return ""
}

// ── discovery helpers ────────────────────────────────────────────────────────

func collectSegments(fsys core.FS, pagesDir, pageDir string, exts []string) ([]core.PageSegment, error) {
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
	var segments []core.PageSegment
	for _, d := range dirs {
		var seg core.PageSegment
		seg.Dir = d
		if candidate := findWithExts(fsys, d, "layout", exts); candidate != "" {
			if ok, _ := hasDefaultExport(fsys, candidate); ok {
				seg.LayoutPath = candidate
			}
		}
		if candidate := findWithExts(fsys, d, "error", exts); candidate != "" {
			if ok, _ := hasDefaultExport(fsys, candidate); ok {
				seg.ErrorPath = candidate
			}
		}
		if seg.LayoutPath != "" || seg.ErrorPath != "" {
			segments = append(segments, seg)
		}
	}
	return segments, nil
}

func discoverCompanions(fsys core.FS, pageDir string, exts []string) (loading, notFound string) {
	for _, base := range []string{"loading", "not-found"} {
		candidate := findWithExts(fsys, pageDir, base, exts)
		if candidate == "" {
			continue
		}
		ok, _ := hasDefaultExport(fsys, candidate)
		if !ok {
			continue
		}
		switch base {
		case "loading":
			loading = candidate
		case "not-found":
			notFound = candidate
		}
	}
	return
}

type globalComponents struct {
	NotFoundPath string
	ErrorPath    string
}

func discoverGlobalComponents(fsys core.FS, appDir string, exts []string) globalComponents {
	pagesDir := filepath.Join(appDir, "app")
	var gc globalComponents
	for _, item := range []struct {
		base string
		dest *string
	}{
		{"global-not-found", &gc.NotFoundPath},
		{"global-error", &gc.ErrorPath},
	} {
		candidate := findWithExts(fsys, pagesDir, item.base, exts)
		if candidate == "" {
			continue
		}
		if ok, _ := hasDefaultExport(fsys, candidate); ok {
			*item.dest = candidate
		}
	}
	return gc
}

// detectRootLayoutHTML checks whether the root layout file contains "<html"
// in its source, indicating it returns a full HTML document (Next.js-style).
func detectRootLayoutHTML(fsys core.FS, pagesDir string, exts []string) bool {
	layoutPath := findWithExts(fsys, pagesDir, "layout", exts)
	if layoutPath == "" {
		return false
	}
	data, err := fsys.ReadFile(layoutPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "<html")
}

func discoverClientComponents(fsys core.FS, appDir string, exts []string) ([]string, error) {
	componentsDir := filepath.Join(appDir, "components")
	if !fsys.Exists(componentsDir) {
		return nil, nil
	}
	extSet := make(map[string]bool, len(exts))
	for _, e := range exts {
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		extSet[e] = true
	}
	var paths []string
	err := walkDir(fsys, componentsDir, func(path string, info core.FSFileInfo) error {
		if info.IsDir || !extSet[filepath.Ext(info.Name)] {
			return nil
		}
		ok, err := hasUseClient(fsys, path)
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
