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
//     Entry: app/client-entry.tsx + "use client" components
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

	// ClientEntryOutput is the /public/... URL of the compiled client-entry
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
	// Export is the named export to render (e.g. "IndexPage").
	Export string
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

	// ClientEntry is app/client-entry.tsx — the browser bootstrap.
	ClientEntry string

	// ClientComponents are all "use client" TSX files.
	ClientComponents []string

	// PolyfillsJS is the path to runtime/polyfills.js.
	PolyfillsJS string

	// External packages to mark external in all builds.
	External []string
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

	manifest, importURLs, err := buildManifest(cfg.ClientComponents, clientFiles, cfg.AppDir)
	if err != nil {
		return nil, fmt.Errorf("bundler: manifest: %w", err)
	}
	manifestJSON, _ := json.MarshalIndent(manifest, "", "  ")
	_ = os.WriteFile(filepath.Join(cfg.OutDir, "client-manifest.json"), manifestJSON, 0o644)

	// ------------------------------------------------------------------ //
	// Pass 2 — Pages bundle (react-server condition, CJS)                 //
	// Includes react-server-dom-webpack/server.browser which produces     //
	// RSC Flight wire format from renderToReadableStream().               //
	// __webpack_require__ is stubbed in polyfills.js.                     //
	// ------------------------------------------------------------------ //
	manifestDefine, _ := json.Marshal(manifest)
	pagesJS, err := buildPagesBundle(cfg, absDir, string(manifestDefine))
	if err != nil {
		return nil, fmt.Errorf("bundler: pages pass: %w", err)
	}

	polyfills := loadPolyfills(cfg.PolyfillsJS)
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
		entry.WriteString(fmt.Sprintf("import { %s } from %q;\n", p.Export, importPath))
	}
	entry.WriteString("const __pages__ = {\n")
	for _, p := range cfg.Pages {
		entry.WriteString(fmt.Sprintf("  %s,\n", p.Export))
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
			entryOutput = "/public/" + rel
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
func buildManifest(clientComponents []string, clientFiles map[string][]byte, appDir string) (map[string]clientManifestEntry, map[string]string, error) {
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
		chunkURL := "/public/main.js"
		for out := range clientFiles {
			b := filepath.Base(out)
			if strings.HasPrefix(b, base+"-") || b == base+".js" {
				chunkURL = "/public/" + out
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

func loadPolyfills(path string) string {
	if path != "" {
		if b, err := os.ReadFile(path); err == nil {
			return string(b)
		}
	}
	return inlinePolyfills()
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

func inlinePolyfills() string {
	return `
var TextEncoder=(function(){function T(){}T.prototype.encode=function(s){s=String(s);var b=[];for(var i=0;i<s.length;i++){var c=s.charCodeAt(i);if(c<0x80)b.push(c);else if(c<0x800){b.push(0xC0|(c>>6));b.push(0x80|(c&0x3F));}else{b.push(0xE0|(c>>12));b.push(0x80|((c>>6)&0x3F));b.push(0x80|(c&0x3F));}}return new Uint8Array(b);};T.prototype.encodeInto=function(s,d){var e=this.encode(s);var w=Math.min(e.length,d.length);for(var i=0;i<w;i++)d[i]=e[i];return{read:s.length,written:w};};return T;})();
var TextDecoder=(function(){function T(e){this.encoding=e||'utf-8';}T.prototype.decode=function(b){if(!b)return'';var a=b instanceof Uint8Array?b:new Uint8Array(b);var s='';for(var i=0;i<a.length;){var c=a[i++];if(c<0x80)s+=String.fromCharCode(c);else if((c&0xE0)===0xC0)s+=String.fromCharCode(((c&0x1F)<<6)|(a[i++]&0x3F));else{s+=String.fromCharCode(((c&0x0F)<<12)|((a[i++]&0x3F)<<6)|(a[i++]&0x3F));}}return s;};return T;})();
var __microtaskQueue__=[];var queueMicrotask=function(fn){__microtaskQueue__.push(fn);};function __drainMicrotasks__(){while(__microtaskQueue__.length>0){var t=__microtaskQueue__.splice(0);for(var i=0;i<t.length;i++){try{t[i]();}catch(e){}}}};
var MessageChannel=(function(){function P(){this.onmessage=null;}P.prototype.postMessage=function(d){var s=this;__microtaskQueue__.push(function(){if(s._partner&&typeof s._partner.onmessage==='function')s._partner.onmessage({data:d});});};P.prototype.close=function(){};function MC(){this.port1=new P();this.port2=new P();this.port1._partner=this.port2;this.port2._partner=this.port1;}return MC;})();
var ReadableStream=(function(){function C(s){this._stream=s;this._chunks=[];this._closed=false;this._error=null;}C.prototype.enqueue=function(c){if(!this._closed)this._chunks.push(c);};C.prototype.close=function(){this._closed=true;};C.prototype.error=function(e){this._error=e;this._closed=true;};Object.defineProperty(C.prototype,'byobRequest',{get:function(){return null;}});Object.defineProperty(C.prototype,'desiredSize',{get:function(){return this._closed?0:1;}});function RS(src,_s){this._controller=new C(this);this._src=src||{};this._started=false;}RS.prototype._start=function(){if(this._started)return;this._started=true;if(typeof this._src.start==='function')this._src.start(this._controller);};RS.prototype._pull=function(){if(typeof this._src.pull==='function')this._src.pull(this._controller);};return RS;})();
function __pullStream__(s){s._start();__drainMicrotasks__();s._pull();__drainMicrotasks__();var c=s._controller._chunks.splice(0);return{chunks:c,done:s._controller._closed&&c.length===0};}
`
}
