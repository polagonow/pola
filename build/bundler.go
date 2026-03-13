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

	// Manifest is the JSON client component manifest.
	Manifest []byte
}

// BundlerConfig controls both build passes.
type BundlerConfig struct {
	// AppDir is the root of the app/ directory (absolute or relative to cwd).
	AppDir string

	// OutDir is where client bundles are written (e.g. "./public").
	OutDir string

	// ServerEntry is app/server-entry.tsx — the pages bundle entry point.
	ServerEntry string

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

	manifest, err := buildManifest(cfg.ClientComponents, clientFiles, cfg.AppDir)
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
	}, nil
}

// buildPagesBundle compiles server-entry.tsx with the react-server condition.
// react-server-dom-webpack/server.browser is bundled in (not external) so
// renderToReadableStream is available in Goja at runtime.
func buildPagesBundle(cfg BundlerConfig, absDir string, manifestDefineJSON string) (string, error) {
	if cfg.ServerEntry == "" {
		return "", nil
	}

	// Mark "use client" files external — they become ClientReference objects.
	// react-server-dom-webpack/server.browser is bundled IN (not external).
	external := append([]string{}, cfg.External...)
	external = append(external, cfg.ClientComponents...)

	r := api.Build(api.BuildOptions{
		EntryPoints:       []string{cfg.ServerEntry},
		Bundle:            true,
		Format:            api.FormatCommonJS,
		Platform:          api.PlatformBrowser,
		JSX:               api.JSXAutomatic,
		Target:            api.ES2020,
		External:          external,
		AbsWorkingDir:     absDir,
		Write:             false,
		Sourcemap:         api.SourceMapNone,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		// react-server resolves React to its server-safe variant
		// browser selects the .browser.js Flight encoder
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

func buildManifest(clientComponents []string, clientFiles map[string][]byte, appDir string) (map[string]clientManifestEntry, error) {
	m := make(map[string]clientManifestEntry)
	for _, src := range clientComponents {
		rel, err := filepath.Rel(appDir, src)
		if err != nil {
			rel = filepath.Base(src)
		}
		id := strings.TrimSuffix(rel, filepath.Ext(rel))
		id = strings.ReplaceAll(id, string(filepath.Separator), "/")
		base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
		var chunks []string
		for out := range clientFiles {
			b := filepath.Base(out)
			if strings.HasPrefix(b, base+"-") || b == base+".js" {
				chunks = append(chunks, "/public/"+out)
			}
		}
		if len(chunks) == 0 {
			chunks = []string{"/public/main.js"}
		}
		entry := clientManifestEntry{ID: id, Name: "default", Chunks: chunks}
		m[id] = entry
		m[id+"/default"] = entry
	}
	return m, nil
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
