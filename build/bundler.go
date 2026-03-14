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
// Final Goja VM bundle = polyfills.js + pages CJS bundle
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
	// ServerBundle is the JS for the Goja VM: polyfills + pages CJS bundle.
	ServerBundle string

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
type PageEntry struct {
	// File is the path to the TSX/TS file, relative to cwd (e.g. "./app/pages/index.tsx").
	File string
	// Export is either a named export (e.g. "IndexPage") or the sentinel "default"
	// to use the file's default export. When "default", the JS identifier used in
	// the generated server entry is derived from the file basename (e.g.
	// "pages/index.tsx" → "Index").
	Export string
}

// pageAlias returns the JS identifier used for p in the generated server entry.
// For named exports it is p.Export unchanged; for "default" it derives a name
// from the file basename (e.g. "pages/index.tsx" → "Index").
func pageAlias(p PageEntry) string {
	if p.Export != "default" {
		return p.Export
	}
	base := strings.TrimSuffix(filepath.Base(p.File), filepath.Ext(p.File))
	if len(base) == 0 {
		return "Page"
	}
	return strings.ToUpper(base[:1]) + base[1:]
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

	// ClientComponents are all "use client" TSX files.
	ClientComponents []string

	// PolyfillsJS is the path to runtime/polyfills.js.
	PolyfillsJS string

	// External packages to mark external in all builds.
	External []string

	// AssetsURLPath is the URL prefix for client bundle files served by the
	// static file handler (e.g. "/public/assets"). No trailing slash.
	// Defaults to "/public/assets" when empty.
	AssetsURLPath string
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

	pagesJS, err := buildPagesBundle(cfg, absDir, string(manifestDefine))
	if err != nil {
		return nil, fmt.Errorf("bundler: pages pass: %w", err)
	}

	polyfills, err := loadPolyfills(cfg.PolyfillsJS)
	if err != nil {
		return nil, err
	}
	serverBundle := polyfills + "\n" + pagesJS

	return &BundleResult{
		ServerBundle:      serverBundle,
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
func buildPagesBundle(cfg BundlerConfig, absDir string, manifestDefineJSON string) (string, error) {
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
		absFile, _ := filepath.Abs(p.File)
		// Import path relative to AppDir (the Stdin resolveDir).
		rel, err := filepath.Rel(absAppDir, absFile)
		if err != nil {
			rel = absFile
		}
		importPath := "./" + filepath.ToSlash(rel)
		alias := pageAlias(p)
		if p.Export == "default" {
			entry.WriteString(fmt.Sprintf("import %s from %q;\n", alias, importPath))
		} else {
			entry.WriteString(fmt.Sprintf("import { %s } from %q;\n", alias, importPath))
		}
	}
	entry.WriteString("const __pages__ = {\n")
	for _, p := range cfg.Pages {
		entry.WriteString(fmt.Sprintf("  %s,\n", pageAlias(p)))
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
		// moduleId matches the manifest key: path relative to appDir, no extension.
		rel, err := filepath.Rel(cfg.AppDir, src)
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
				trimmed := strings.TrimSpace(string(contents))
				if !strings.HasPrefix(trimmed, "'use client'") && !strings.HasPrefix(trimmed, `"use client"`) {
					// Not a client component — let esbuild handle normally.
					return api.OnLoadResult{}, nil
				}

				// Compute moduleId: prefer the pre-registered value from clientSet
				// (guarantees manifest alignment), fall back to deriving from appDir.
				moduleId, ok := clientSet[args.Path]
				if !ok {
					rel, relErr := filepath.Rel(cfg.AppDir, args.Path)
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
		Write:             false,
		Sourcemap:         api.SourceMapNone,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		Plugins:           []api.Plugin{useClientPlugin},

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
	var out string
	for _, f := range r.OutputFiles {
		out += string(f.Contents)
	}
	return out, nil
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
	m := make(map[string]clientManifestEntry)
	importURLs := make(map[string]string)

	for _, src := range clientComponents {
		rel, err := filepath.Rel(appDir, src)
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

		// Set chunks=["default"] so the ESM client uses metadata[1] as the
		// export name ("default") rather than treating it as a chunk path.
		m[id] = clientManifestEntry{ID: id, Name: "default", Chunks: []string{"default"}}
	}
	return m, importURLs, nil
}

// ------------------------------------------------------------------ //
// Polyfills                                                           //
// ------------------------------------------------------------------ //

func loadPolyfills(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err == nil {
		return string(b), nil
	}

	return "", fmt.Errorf("Could not load pollyfiles: %v", err)
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
