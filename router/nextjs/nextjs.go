// Package nextjs provides a Next.js-style file-system router.
// Routes are discovered by scanning the app directory for page files
// matching any of the configured extensions (provided by the active renderer).
package nextjs

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
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
func (r *Router) ScanRoutes(ctx context.Context, _ core.FS, appDir string, exts []string) ([]core.Route, error) {
	// Determine the pages root: prefer appDir/app/, fall back to appDir.
	pagesDir := filepath.Join(appDir, "app")
	if !dirExists(pagesDir) {
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

	err := filepath.WalkDir(pagesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		base := d.Name()
		// Must be named "page.<ext>" for one of the allowed extensions.
		if !isPageFile(base, extSet) {
			return nil
		}

		ok, err := hasDefaultExport(path)
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
		segs, segErr := collectSegments(pagesDir, filepath.Dir(path))
		if segErr != nil {
			return segErr
		}
		entry := core.PageEntry{
			PageComponentPath: path,
			Segments:          segs,
		}
		entry.LoadingComponentPath, entry.NotFoundComponentPath = discoverCompanions(filepath.Dir(path))
		pages = append(pages, entry)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Discover client components from appDir/components/.
	clientComponents, err := discoverClientComponents(appDir, exts)
	if err != nil && !os.IsNotExist(err) {
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
	gc := discoverGlobalComponents(appDir, exts)
	if gc.ErrorPath != "" && !seen[gc.ErrorPath] {
		clientComponents = append(clientComponents, gc.ErrorPath)
	}

	r.mu.Lock()
	r.routes = routes
	r.sorted = false
	r.discovery = core.DiscoveryResult{
		AppDir:           appDir,
		Pages:            pages,
		ClientComponents: clientComponents,
		GlobalNotFound:   gc.NotFoundPath,
		GlobalError:      gc.ErrorPath,
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

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ── discovery helpers ────────────────────────────────────────────────────────

func collectSegments(pagesDir, pageDir string) ([]core.PageSegment, error) {
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

type globalComponents struct {
	NotFoundPath string
	ErrorPath    string
}

func discoverGlobalComponents(appDir string, exts []string) globalComponents {
	_ = exts // reserved for future multi-extension support
	pagesDir := filepath.Join(appDir, "app")
	var gc globalComponents
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
	return gc
}

func discoverClientComponents(appDir string, exts []string) ([]string, error) {
	_ = exts // currently scans .ts and .tsx regardless of exts (renderer-specific)
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

func isTSFile(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".ts" || ext == ".tsx"
}

func hasDefaultExport(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()
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

func hasUseClient(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		line = strings.TrimSuffix(line, ";")
		return line == `"use client"` || line == `'use client'`, nil
	}
	return false, nil
}
