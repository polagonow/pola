// Package build compiles the GoJSX app using esbuild.
//
// # Architecture
//
// Two esbuild passes produce one JS bundle for the Goja server VM and one
// ESM bundle for the browser:
//
//  1. Pages pass  (react-server condition, CJS)
//     Entry: app/server-entry.tsx
//     Bundles: React (server variant) + react-server-dom-webpack/server.browser
//     Output: RSC Flight wire format from renderToReadableStream()
//
//  2. Client pass (browser condition, ESM)
//     Entry: app/_client.tsx + "use client" components
//     Bundles: React + react-dom/client + react-server-dom-esm/client
//     Output: ESM files written to public/
//
// Final Goja VM bundle = pages CJS bundle (polyfills are native Go in runtime/polyfill)
package build

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// BundleResult is the output of a successful build.
type BundleResult struct {
	// ServerBundlePath is the on-disk path of the emitted server CJS bundle.
	ServerBundlePath string

	// ClientFiles maps relative output path → bytes written to OutDir.
	ClientFiles map[string][]byte

	// ClientEntryOutput is the /public/... URL of the compiled client
	// injected as <script type="module" src="..."> in the HTML shell.
	ClientEntryOutput string

	// Manifest is the JSON client component manifest passed to __CLIENT_MANIFEST__
	// in the server bundle. Format: {moduleId: {id, chunks, name, async}}.
	// chunks is set to ["default"] so that react-server-dom-esm/client reads
	// metadata[1] as the export name ("default") rather than a chunk URL.
	Manifest []byte

	// ImportURLs maps module IDs to their browser-loadable chunk URLs.
	// Used by the HTML shell's <script type="importmap"> so that
	// import("components/Counter") resolves to /public/Counter-HASH.js.
	ImportURLs map[string]string
}

// PageEntry describes one server-rendered page.
// Every page must use `export default function` — the convention is enforced
// by DiscoverPages at discovery time.
type PageEntry struct {
	// PageComponentPath is the absolute or cwd-relative path to the page.tsx file.
	PageComponentPath string

	// Segments holds one entry per directory level from the pages/ root (outermost)
	// to the page's own directory (innermost). Each segment may have a LayoutPath
	// and/or ErrorPath. Empty if no ancestor has a layout.tsx or error.tsx.
	Segments []PageSegment

	// LoadingComponentPath is the path to the co-located loading.tsx, or "" if absent.
	LoadingComponentPath string

	// NotFoundComponentPath is the path to the co-located not-found.tsx (server component), or "" if absent.
	NotFoundComponentPath string
}

// BundlerConfig controls both build passes.
type BundlerConfig struct {
	// AppDir is the root of the app/ directory (absolute or relative to cwd).
	AppDir string

	// OutDir is where client bundles are written (e.g. "./public").
	OutDir string

	// Pages lists every server-rendered page. The bundler auto-generates the
	// server entry (imports + __render__ function) from this list, so no
	// hand-maintained server-entry.tsx is needed.
	Pages []PageEntry

	// ClientEntry is app/_client.tsx — the browser bootstrap.
	ClientEntry string

	// ServerEntry is the path where the server CJS bundle is emitted.
	// Defaults to filepath.Dir(OutDir)/_server.js when empty.
	ServerEntry string

	// ClientComponents are all "use client" TSX files.
	ClientComponents []string

	// External packages to mark external in all builds.
	External []string

	// AssetsURLPath is the URL prefix for client bundle files served by the
	// static file handler (e.g. "/public/assets"). No trailing slash.
	// Defaults to "/public/assets" when empty.
	AssetsURLPath string

	// GlobalNotFoundPath is the path to app/global-not-found.tsx (server component),
	// or "" if absent. Rendered when no route matches (HTTP 404).
	GlobalNotFoundPath string

	// GlobalErrorPath is the path to app/global-error.tsx ("use client"),
	// or "" if absent. Wraps all pages as the outermost error boundary.
	GlobalErrorPath string
}

// Bundle runs both esbuild passes and returns the combined result.
func Bundle(cfg BundlerConfig) (*BundleResult, error) {
	if err := os.MkdirAll(cfg.OutDir, 0o755); err != nil {
		return nil, fmt.Errorf("bundler: mkdir %s: %w", cfg.OutDir, err)
	}

	absDir, err := filepath.Abs(".")
	if err != nil {
		return nil, fmt.Errorf("bundler: abs cwd: %w", err)
	}

	// ------------------------------------------------------------------ //
	// Pass 1 — Client bundle (browser ESM)                                //
	// Build first so we know output filenames before building the manifest //
	// ------------------------------------------------------------------ //
	clientFiles, clientEntryOutput, err := buildClientBundle(cfg, absDir)
	if err != nil {
		return nil, err
	}

	manifest, importURLs, err := buildManifest(cfg.ClientComponents, clientFiles, cfg.AppDir, cfg.AssetsURLPath)
	if err != nil {
		return nil, fmt.Errorf("bundler: manifest: %w", err)
	}
	manifestJSON, _ := json.MarshalIndent(manifest, "", "  ")
	_ = os.WriteFile(filepath.Join(cfg.OutDir, "manifest.json"), manifestJSON, 0o644)

	// ------------------------------------------------------------------ //
	// Pass 2 — Pages bundle (react-server condition, CJS)                 //
	// Includes react-server-dom-webpack/server.browser which produces     //
	// RSC Flight wire format from renderToReadableStream().               //
	// __webpack_require__ is stubbed in polyfills.js.                     //
	// ------------------------------------------------------------------ //
	manifestDefine, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("bundler: manifest pass: %w", err)
	}

	absOutDir, err := filepath.Abs(cfg.OutDir)
	if err != nil {
		return nil, fmt.Errorf("bundler: abs outdir: %w", err)
	}
	serverEntry := cfg.ServerEntry
	if serverEntry == "" {
		serverEntry = filepath.Join(filepath.Dir(absOutDir), "_server.js")
	} else {
		serverEntry, err = filepath.Abs(serverEntry)
		if err != nil {
			return nil, fmt.Errorf("bundler: abs server entry: %w", err)
		}
	}

	serverBundlePath, err := buildPagesBundle(cfg, absDir, string(manifestDefine), serverEntry)
	if err != nil {
		return nil, fmt.Errorf("bundler: pages pass: %w", err)
	}

	return &BundleResult{
		ServerBundlePath:  serverBundlePath,
		ClientFiles:       clientFiles,
		ClientEntryOutput: clientEntryOutput,
		Manifest:          manifestJSON,
		ImportURLs:        importURLs,
	}, nil
}

// buildPagesBundle generates a server entry from cfg.Pages and compiles it
// with the react-server condition using esbuild's Stdin option — no
// hand-maintained server-entry.tsx file needed.
//
// "use client" files are intercepted by the useClientPlugin: instead of being
// bundled or marked external, each one is replaced with a synthetic module that
// calls createClientModuleProxy(moduleId). This is the standard RSC bundler
// pattern (used by Next.js, Parcel, etc.) — the server never runs client code,
// it only emits a ClientReference that the Flight encoder serialises as an
// I: row. The browser receives that row, imports the real chunk, and renders
// the component with full React state.
// newAtAliasPlugin returns an esbuild plugin that resolves @/ imports to absAppDir.
// e.g. import foo from "@/jsi"  →  absAppDir/jsi
//
// We delegate back to build.Resolve so esbuild handles extension lookup,
// directory index files, etc. rather than returning a bare path ourselves.
func newAtAliasPlugin(absAppDir string) api.Plugin {
	return api.Plugin{
		Name: "at-alias",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `^@/`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				// "@/foo/bar" → "./foo/bar" resolved from the app root
				rel := "." + args.Path[1:]
				result := build.Resolve(rel, api.ResolveOptions{
					ResolveDir: absAppDir,
					Kind:       args.Kind,
				})
				if len(result.Errors) > 0 {
					return api.OnResolveResult{}, fmt.Errorf("%s", result.Errors[0].Text)
				}
				return api.OnResolveResult{Path: result.Path}, nil
			})
		},
	}
}

func buildPagesBundle(cfg BundlerConfig, absDir string, manifestDefineJSON string, serverOutFile string) (string, error) {
	if len(cfg.Pages) == 0 {
		return "", nil
	}

	// Generate the server entry in memory from the page list.
	// The entry imports each page file and exposes __render__ as a global.
	absAppDir, _ := filepath.Abs(cfg.AppDir)
	var entry strings.Builder
	entry.WriteString(`import React from "react";` + "\n")
	entry.WriteString(`import { renderToReadableStream } from "react-server-dom-webpack/server.browser";` + "\n")
	for _, p := range cfg.Pages {
		absFile, _ := filepath.Abs(p.PageComponentPath)
		// Import path relative to AppDir (the Stdin resolveDir).
		rel, err := filepath.Rel(absAppDir, absFile)
		if err != nil {
			rel = absFile
		}
		importPath := "./" + filepath.ToSlash(rel)
		alias := PageAlias(p)
		entry.WriteString(fmt.Sprintf("import %s from %q;\n", alias, importPath))
	}
	// Import layout files — deduplicated and in deterministic (page-walk) order.
	// Each unique layout path is imported once with alias LayoutAlias(...)+"Layout".
	absPagesDir := filepath.Join(absAppDir, "app")
	seenLayout := make(map[string]bool)
	var layoutPaths []string
	for _, p := range cfg.Pages {
		for _, seg := range p.Segments {
			if seg.LayoutPath == "" {
				continue
			}
			abs, _ := filepath.Abs(seg.LayoutPath)
			if !seenLayout[abs] {
				seenLayout[abs] = true
				layoutPaths = append(layoutPaths, abs)
			}
		}
	}
	for _, abs := range layoutPaths {
		rel, _ := filepath.Rel(absAppDir, abs)
		alias := LayoutAlias(absPagesDir, abs) + "Layout"
		entry.WriteString(fmt.Sprintf("import %s from %q;\n", alias, "./"+filepath.ToSlash(rel)))
	}

	// Import the shared framework ErrorBoundary once if any segment has an error component
	// or if a global error boundary is configured.
	{
		hasErrorPage := cfg.GlobalErrorPath != ""
	outer:
		for _, p := range cfg.Pages {
			for _, seg := range p.Segments {
				if seg.ErrorPath != "" {
					hasErrorPage = true
					break outer
				}
			}
		}
		if hasErrorPage {
			entry.WriteString("import __FrameworkErrorBoundary__ from \"./components/ErrorBoundary\";\n")
		}
	}

	// Import error components from segments — deduplicated.
	seenError := make(map[string]bool)
	for _, p := range cfg.Pages {
		for _, seg := range p.Segments {
			if seg.ErrorPath == "" {
				continue
			}
			abs, _ := filepath.Abs(seg.ErrorPath)
			if seenError[abs] {
				continue
			}
			seenError[abs] = true
			rel, _ := filepath.Rel(absAppDir, abs)
			alias := LayoutAlias(absPagesDir, abs) + "Error"
			entry.WriteString(fmt.Sprintf("import %s from %q;\n", alias, "./"+filepath.ToSlash(rel)))
		}
	}

	// Import global components (global-error, global-not-found).
	if cfg.GlobalErrorPath != "" {
		abs, _ := filepath.Abs(cfg.GlobalErrorPath)
		rel, _ := filepath.Rel(absAppDir, abs)
		fmt.Fprintf(&entry, "import GlobalError from %q;\n", "./"+filepath.ToSlash(rel))
	}
	if cfg.GlobalNotFoundPath != "" {
		abs, _ := filepath.Abs(cfg.GlobalNotFoundPath)
		rel, _ := filepath.Rel(absAppDir, abs)
		fmt.Fprintf(&entry, "import GlobalNotFound from %q;\n", "./"+filepath.ToSlash(rel))
	}

	// Import co-located companions (loading, not-found) per page.
	for _, p := range cfg.Pages {
		alias := PageAlias(p)
		for _, companion := range []struct{ path, suffix string }{
			{p.LoadingComponentPath, "Loading"},
			{p.NotFoundComponentPath, "NotFound"},
		} {
			if companion.path == "" {
				continue
			}
			abs, _ := filepath.Abs(companion.path)
			rel, _ := filepath.Rel(absAppDir, abs)
			entry.WriteString(fmt.Sprintf("import %s%s from %q;\n",
				alias, companion.suffix, "./"+filepath.ToSlash(rel)))
		}
	}

	// Generate per-page wrapper functions.
	// Wrapping order (innermost → outermost):
	//   Page (with notFound prop) → Suspense/Loading → per segment (EB then Layout), innermost first → GlobalEB
	pageHasCompanions := func(p PageEntry) bool {
		return len(p.Segments) > 0 || p.LoadingComponentPath != "" || p.NotFoundComponentPath != ""
	}
	for _, p := range cfg.Pages {
		alias := PageAlias(p)
		needsWrapper := pageHasCompanions(p) || cfg.GlobalErrorPath != ""
		if !needsWrapper {
			continue
		}
		// not-found is a server component passed as a prop, not a wrapper.
		pageProps := "props"
		if p.NotFoundComponentPath != "" {
			pageProps = fmt.Sprintf("{...props, notFound: %sNotFound}", alias)
		}
		inner := fmt.Sprintf("React.createElement(%s, %s)", alias, pageProps)
		if p.LoadingComponentPath != "" {
			inner = fmt.Sprintf(
				"React.createElement(React.Suspense,{fallback:React.createElement(%sLoading,null)},%s)",
				alias, inner)
		}
		// Wrap segments innermost-first: at each level apply EB (if ErrorPath) then Layout (if LayoutPath).
		for i := len(p.Segments) - 1; i >= 0; i-- {
			seg := p.Segments[i]
			if seg.ErrorPath != "" {
				abs, _ := filepath.Abs(seg.ErrorPath)
				errAlias := LayoutAlias(absPagesDir, abs) + "Error"
				inner = fmt.Sprintf(
					"React.createElement(__FrameworkErrorBoundary__,{fallback:%s},%s)",
					errAlias, inner)
			}
			if seg.LayoutPath != "" {
				abs, _ := filepath.Abs(seg.LayoutPath)
				layoutAlias := LayoutAlias(absPagesDir, abs) + "Layout"
				inner = fmt.Sprintf("React.createElement(%s,null,%s)", layoutAlias, inner)
			}
		}
		// Wrap with global error boundary at the outermost level (above all layouts).
		if cfg.GlobalErrorPath != "" {
			inner = fmt.Sprintf(
				"React.createElement(__FrameworkErrorBoundary__,{fallback:GlobalError},%s)", inner)
		}
		fmt.Fprintf(&entry, "function __wrap_%s__(props: any){return %s;}\n", alias, inner)
	}

	entry.WriteString("const __pages__ = {\n")
	for _, p := range cfg.Pages {
		alias := PageAlias(p)
		needsWrapper := pageHasCompanions(p) || cfg.GlobalErrorPath != ""
		if needsWrapper {
			fmt.Fprintf(&entry, "  %s: __wrap_%s__,\n", alias, alias)
		} else {
			fmt.Fprintf(&entry, "  %s,\n", alias)
		}
	}
	if cfg.GlobalNotFoundPath != "" {
		entry.WriteString("  GlobalNotFound,\n")
	}
	entry.WriteString("};\n")
	entry.WriteString(`
(globalThis as any).__render__ = function(exportName: string, propsJSON: string): ReadableStream {
  const Page = (__pages__ as any)[exportName];
  if (!Page) throw new Error('__render__: unknown page "' + exportName + '". Known: ' + Object.keys(__pages__).join(", "));
  return renderToReadableStream(React.createElement(Page, JSON.parse(propsJSON || "{}")), __CLIENT_MANIFEST__);
};
`)

	// Build a set of absolute paths for quick lookup inside the plugin.
	clientSet := make(map[string]string) // absPath → moduleId
	for _, src := range cfg.ClientComponents {
		abs, err := filepath.Abs(src)
		if err != nil {
			abs = src
		}
		// moduleId matches the manifest key: path relative to absAppDir, no extension.
		rel, err := filepath.Rel(absAppDir, abs)
		if err != nil {
			rel = filepath.Base(src)
		}
		moduleId := strings.TrimSuffix(rel, filepath.Ext(rel))
		moduleId = strings.ReplaceAll(moduleId, string(filepath.Separator), "/")
		clientSet[abs] = moduleId
	}

	// useClientPlugin intercepts "use client" files during the server pass and
	// replaces them with a synthetic proxy stub.
	//
	// Why OnLoad (not OnResolve): TypeScript imports omit file extensions
	// (e.g. `import Counter from "./Counter"`). OnResolve sees the raw import
	// specifier before extension resolution, so a filter like `\.(tsx|ts)$`
	// would never match. OnLoad fires after esbuild has fully resolved the path
	// (args.Path is always the absolute path with extension), so the filter works
	// correctly and we can read the real file to detect the directive.
	useClientPlugin := api.Plugin{
		Name: "resolve-client-imports",
		Setup: func(build api.PluginBuild) {
			build.OnLoad(api.OnLoadOptions{Filter: `\.(tsx|ts|jsx|js)$`}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				// Read the file to check for the 'use client' directive,
				// mirroring how the JS plugin calls readFile on each resolved import.
				contents, err := os.ReadFile(args.Path)
				if err != nil {
					return api.OnLoadResult{}, nil
				}
				// A file is a client component if it is explicitly registered in
				// clientSet OR if its content begins with the "use client" directive.
				// Checking clientSet first handles files whose directive is preceded
				// by comments (e.g. global-error.tsx).
				_, inClientSet := clientSet[args.Path]
				trimmed := strings.TrimSpace(string(contents))
				hasDirective := strings.HasPrefix(trimmed, "'use client'") || strings.HasPrefix(trimmed, `"use client"`)
				if !inClientSet && !hasDirective {
					// Not a client component — let esbuild handle normally.
					return api.OnLoadResult{}, nil
				}

				// Compute moduleId: prefer the pre-registered value from clientSet
				// (guarantees manifest alignment), fall back to deriving from appDir.
				moduleId, ok := clientSet[args.Path]
				if !ok {
					rel, relErr := filepath.Rel(absAppDir, args.Path)
					if relErr != nil {
						rel = filepath.Base(args.Path)
					}
					moduleId = strings.TrimSuffix(rel, filepath.Ext(rel))
					moduleId = strings.ReplaceAll(moduleId, string(filepath.Separator), "/")
				}

				// Emit the synthetic server-side stub:
				//   import { createClientModuleProxy } from "react-server-dom-webpack/server.browser";
				//   module.exports = createClientModuleProxy("<moduleId>");
				// The proxy intercepts any property access and returns a registered
				// ClientReference, which the Flight encoder serialises as an I: row.
				stub := fmt.Sprintf(
					`import { createClientModuleProxy } from "react-server-dom-webpack/server.browser";`+"\n"+
						`module.exports = createClientModuleProxy(%q);`+"\n",
					moduleId,
				)
				return api.OnLoadResult{
					Contents: &stub,
					Loader:   api.LoaderJS,
				}, nil
			})
		},
	}

	if err := os.MkdirAll(filepath.Dir(serverOutFile), 0o755); err != nil {
		return "", fmt.Errorf("bundler: mkdir server out: %w", err)
	}

	entryStr := entry.String()
	r := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   entryStr,
			ResolveDir: absAppDir, // imports resolve relative to app/
			Loader:     api.LoaderTSX,
			Sourcefile: "<generated-server-entry>",
		},
		Bundle:            true,
		Format:            api.FormatCommonJS,
		Platform:          api.PlatformBrowser,
		JSX:               api.JSXAutomatic,
		Target:            api.ES2020,
		External:          cfg.External,
		AbsWorkingDir:     absDir,
		Outfile:           serverOutFile,
		Write:             true,
		Sourcemap:         api.SourceMapNone,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		Plugins:           []api.Plugin{newAtAliasPlugin(absAppDir), useClientPlugin},

		// react-server → React resolves to its server-safe variant.
		// browser → selects react-server-dom-webpack-server.browser.production.js
		Conditions: []string{"react-server", "browser", "module", "default"},
		Define: map[string]string{
			"process.env.NODE_ENV": `"production"`,
			"__DEV__":              "false",
			"__CLIENT_MANIFEST__":  manifestDefineJSON,
		},
	})
	if len(r.Errors) > 0 {
		return "", fmtErrors("pages", r.Errors)
	}
	return serverOutFile, nil
}

// buildClientBundle compiles the browser-side entry + client components.
func buildClientBundle(cfg BundlerConfig, absDir string) (map[string][]byte, string, error) {
	if cfg.AssetsURLPath == "" {
		cfg.AssetsURLPath = "/public/assets"
	}
	entries := []string{}
	if cfg.ClientEntry != "" {
		entries = append(entries, cfg.ClientEntry)
	}
	entries = append(entries, cfg.ClientComponents...)

	entryBase := ""
	if cfg.ClientEntry != "" {
		entryBase = strings.TrimSuffix(filepath.Base(cfg.ClientEntry), filepath.Ext(cfg.ClientEntry))
	}

	absOutDir, err := filepath.Abs(cfg.OutDir)
	if err != nil {
		return nil, "", fmt.Errorf("bundler: abs outdir: %w", err)
	}
	absAppDir, _ := filepath.Abs(cfg.AppDir)

	r := api.Build(api.BuildOptions{
		EntryPoints:       entries,
		Bundle:            true,
		Format:            api.FormatESModule,
		Platform:          api.PlatformBrowser,
		JSX:               api.JSXAutomatic,
		Target:            api.ES2020,
		Splitting:         true,
		Outdir:            absOutDir,
		AbsWorkingDir:     absDir,
		Write:             false,
		Sourcemap:         api.SourceMapNone,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		EntryNames:        "[name]-[hash]",
		ChunkNames:        "chunks/[name]-[hash]",
		Conditions:        []string{"browser", "import", "module", "default"},
		Plugins:           []api.Plugin{newAtAliasPlugin(absAppDir)},
		Define: map[string]string{
			"process.env.NODE_ENV": `"production"`,
			"__DEV__":              "false",
		},
	})
	if len(r.Errors) > 0 {
		return nil, "", fmtErrors("client", r.Errors)
	}

	files := make(map[string][]byte)
	var entryOutput string

	for _, f := range r.OutputFiles {
		rel, _ := filepath.Rel(absOutDir, f.Path)
		if rel == "" {
			rel = filepath.Base(f.Path)
		}
		files[rel] = f.Contents
		if entryBase != "" && strings.HasPrefix(filepath.Base(rel), entryBase+"-") {
			entryOutput = cfg.AssetsURLPath + "/" + rel
		}
		_ = os.MkdirAll(filepath.Dir(f.Path), 0o755)
		_ = os.WriteFile(f.Path, f.Contents, 0o644)
	}
	return files, entryOutput, nil
}

// ------------------------------------------------------------------ //
// Manifest                                                            //
// ------------------------------------------------------------------ //

type clientManifestEntry struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Chunks []string `json:"chunks"`
	Async  bool     `json:"async"`
}

// buildManifest returns two things:
//
//  1. manifest — flat webpack-format map for __CLIENT_MANIFEST__ in the server
//     bundle.  react-server-dom-webpack/server emits I rows as the 3-element
//     array [id, chunks, name]. react-server-dom-esm/client reads that array
//     as [specifier, name] — so metadata[1] (chunks) is used as the export
//     name on the loaded module. Setting chunks=["default"] makes the ESM
//     client call moduleExports["default"] which is the correct export.
//
//  2. importURLs — moduleId → real chunk URL, used by the HTML import map so
//     the browser can resolve import("components/Counter") to the hashed file.
func buildManifest(clientComponents []string, clientFiles map[string][]byte, appDir string, assetsURLPath string) (map[string]clientManifestEntry, map[string]string, error) {
	if assetsURLPath == "" {
		assetsURLPath = "/public/assets"
	}
	absAppDir, _ := filepath.Abs(appDir)
	m := make(map[string]clientManifestEntry)
	importURLs := make(map[string]string)

	for _, src := range clientComponents {
		absSrc, _ := filepath.Abs(src)
		rel, err := filepath.Rel(absAppDir, absSrc)
		if err != nil {
			rel = filepath.Base(src)
		}
		id := strings.TrimSuffix(rel, filepath.Ext(rel))
		id = strings.ReplaceAll(id, string(filepath.Separator), "/")
		base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))

		// Find the real chunk URL for the import map.
		chunkURL := assetsURLPath + "/main.js"
		for out := range clientFiles {
			b := filepath.Base(out)
			if strings.HasPrefix(b, base+"-") || b == base+".js" {
				chunkURL = assetsURLPath + "/" + out
				break
			}
		}
		importURLs[id] = chunkURL

		// React RSC looks up the manifest using "moduleId#exportName".
		// The proxy stub is CJS (module.exports = createClientModuleProxy(id)),
		// so React encodes an empty export name → key = id + "#".
		// We also register id + "#default" for any ESM default-import paths.
		entry := clientManifestEntry{ID: id, Name: "default", Chunks: []string{"default"}}
		m[id+"#"] = entry
		m[id+"#default"] = entry
	}
	return m, importURLs, nil
}

func fmtErrors(pass string, errs []api.Message) error {
	msgs := make([]string, len(errs))
	for i, e := range errs {
		if e.Location != nil {
			msgs[i] = fmt.Sprintf("%s:%d: %s", e.Location.File, e.Location.Line, e.Text)
		} else {
			msgs[i] = e.Text
		}
	}
	return fmt.Errorf("esbuild [%s]:\n%s", pass, strings.Join(msgs, "\n"))
}
