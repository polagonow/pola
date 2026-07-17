package react

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/globals"
	"github.com/polagonow/pola/serveraction"
)

// BundleDefines returns esbuild define keys the React renderer requires
// in the server bundle (e.g. the client manifest placeholder).
func (r *Renderer) BundleDefines() map[string]string {
	return map[string]string{ClientManifestDefine: "{}"}
}

// EntryGenConfig is the input to EntryGenerator.Generate.
type EntryGenConfig struct {
	Pages              []core.PageEntry
	AppDir             string
	GlobalErrorPath    string
	GlobalNotFoundPath string
	ManifestJSON       string // injected as a JS define value

	// RootLayoutReturnsHTML is true when the root layout contains <html,
	// indicating it returns a full HTML document. When set, the entry
	// generator emits __extractShell__ and excludes the root layout from
	// RSC wrapping (it becomes the document shell instead).
	RootLayoutReturnsHTML bool

	// ServerActions are the 'use server' modules discovered under AppDir. The
	// generated entry imports each, registers its exports in the global
	// server-action registry, and registers them with react-server-dom-webpack
	// so actions passed as props serialize as server references.
	ServerActions []serveraction.Module
}

var renderBlockTmpl = template.Must(template.New("renderBlock").Parse(
	`(globalThis as any).{{.RenderFn}} = function(exportName: string, propsJSON: string): ReadableStream {
  const Page = (__pages__ as any)[exportName];
  if (!Page) throw new Error('{{.RenderFn}}: unknown page: ' + exportName);
  return renderToReadableStream(React.createElement(Page, JSON.parse(propsJSON || "{}")), {{.ClientManifest}}, {
    onError(error: unknown) {
      return error instanceof Error ? error.message : String(error);
    },
  });
};
`))

// EntryGenerator produces the in-memory server-entry TypeScript source that
// the bundler compiles into the server VM bundle. It implements the Next.js-
// style file convention (app/page.tsx, layout.tsx, error.tsx, etc.).
type EntryGenerator struct{}

// BundleConditions returns the esbuild Conditions for the React RSC server bundle.
func (g *EntryGenerator) BundleConditions() []string {
	return []string{"react-server", "browser", "module", "default"}
}

// ServerGlobalName returns the JS global the RSCFlightProtocol calls to start a render.
func (g *EntryGenerator) ServerGlobalName() string { return globals.RenderFn }

// Generate produces the in-memory server-entry TypeScript source from the
// discovered page list.
func (g *EntryGenerator) Generate(cfg EntryGenConfig) (string, error) { //nolint:gocyclo
	absAppDir, _ := filepath.Abs(cfg.AppDir)
	absPagesDir := absAppDir
	// The router uses appDir/app as the pages root when the "app" subdirectory
	// exists. Segment directories are relative to that path, so isRootSegment
	// must compare against the same directory.
	if info, err := os.Stat(filepath.Join(absAppDir, "app")); err == nil && info.IsDir() {
		absPagesDir = filepath.Join(absAppDir, "app")
	}

	var entry strings.Builder

	entry.WriteString(`import React from "react";` + "\n")
	entry.WriteString(`import { renderToReadableStream } from "react-server-dom-webpack/server.browser";` + "\n")
	if len(cfg.ServerActions) > 0 {
		// registerServerReference lets RSDW's serializer treat 'use server'
		// functions as server references when passed as props to client components.
		entry.WriteString(`import { registerServerReference as __pola_registerServerReference__ } from "react-server-dom-webpack/server.browser";` + "\n")
	}

	for _, p := range cfg.Pages {
		absFile, _ := filepath.Abs(p.PageComponentPath)
		rel, err := filepath.Rel(absAppDir, absFile)
		if err != nil {
			rel = absFile
		}
		alias := PageAlias(p)
		named := MetadataImportSuffix(alias, p.HasMetadata, p.HasGenMetadata)
		if named != "" {
			fmt.Fprintf(&entry, "import %s, %s from %q;\n", alias, named, "./"+filepath.ToSlash(rel))
		} else {
			fmt.Fprintf(&entry, "import %s from %q;\n", alias, "./"+filepath.ToSlash(rel))
		}
	}

	type layoutInfo struct {
		absPath        string
		hasMetadata    bool
		hasGenMetadata bool
	}
	seenLayout := make(map[string]bool)
	var layouts []layoutInfo
	for _, p := range cfg.Pages {
		for _, seg := range p.Segments {
			if seg.LayoutPath == "" {
				continue
			}
			abs, _ := filepath.Abs(seg.LayoutPath)
			if !seenLayout[abs] {
				seenLayout[abs] = true
				layouts = append(layouts, layoutInfo{abs, seg.HasMetadata, seg.HasGenMetadata})
			}
		}
	}
	for _, li := range layouts {
		rel, _ := filepath.Rel(absAppDir, li.absPath)
		alias := LayoutAlias(absPagesDir, li.absPath) + "Layout"
		named := MetadataImportSuffix(alias, li.hasMetadata, li.hasGenMetadata)
		if named != "" {
			fmt.Fprintf(&entry, "import %s, %s from %q;\n", alias, named, "./"+filepath.ToSlash(rel))
		} else {
			fmt.Fprintf(&entry, "import %s from %q;\n", alias, "./"+filepath.ToSlash(rel))
		}
	}

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
		entry.WriteString("import __FrameworkErrorBoundary__ from \"@pola/react/error-boundary\";\n")
	}

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
			fmt.Fprintf(&entry, "import %s from %q;\n", alias, "./"+filepath.ToSlash(rel))
		}
	}

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
			fmt.Fprintf(&entry, "import %s%s from %q;\n",
				alias, companion.suffix, "./"+filepath.ToSlash(rel))
		}
	}

	// Server-action module imports (registered after the page wiring).
	entry.WriteString(serveraction.ImportLines(cfg.ServerActions, absAppDir))

	pageHasCompanions := func(p core.PageEntry) bool {
		return len(p.Segments) > 0 || p.LoadingComponentPath != "" || p.NotFoundComponentPath != ""
	}
	for _, p := range cfg.Pages {
		alias := PageAlias(p)
		needsWrapper := pageHasCompanions(p) || cfg.GlobalErrorPath != ""
		if !needsWrapper {
			continue
		}
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
		for i := len(p.Segments) - 1; i >= 0; i-- {
			seg := p.Segments[i]
			// When the root layout returns <html>, skip the root-level
			// segment (index 0) from RSC wrapping — it becomes the
			// document shell instead.
			if cfg.RootLayoutReturnsHTML && i == 0 && isRootSegment(absPagesDir, seg) {
				continue
			}
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
		if cfg.GlobalErrorPath != "" {
			inner = fmt.Sprintf(
				"React.createElement(__FrameworkErrorBoundary__,{fallback:GlobalError},%s)", inner)
		}
		fmt.Fprintf(&entry, "function __wrap_%s__(props: any){return %s;}\n", alias, inner)
	}

	entry.WriteString("const __pages__ = {\n")
	for _, p := range cfg.Pages {
		alias := PageAlias(p)
		if pageHasCompanions(p) || cfg.GlobalErrorPath != "" {
			fmt.Fprintf(&entry, "  %s: __wrap_%s__,\n", alias, alias)
		} else {
			fmt.Fprintf(&entry, "  %s,\n", alias)
		}
	}
	if cfg.GlobalNotFoundPath != "" {
		entry.WriteString("  GlobalNotFound,\n")
	}
	entry.WriteString("};\n")

	var renderBlockBuf strings.Builder
	_ = renderBlockTmpl.Execute(&renderBlockBuf, struct{ RenderFn, ClientManifest string }{
		globals.RenderFn,
		ClientManifestDefine,
	})
	entry.WriteString(renderBlockBuf.String())

	// Metadata collection: emit __metadataChain__ + __collectMetadata__.
	entry.WriteString(GenerateMetadataJS(cfg.Pages, absPagesDir))

	// Server-action registry + invocation helpers (reference the __sa_N__ imports).
	if len(cfg.ServerActions) > 0 {
		entry.WriteString(serveraction.RegistryJS(cfg.ServerActions))
	}

	// When the root layout returns <html>, generate __extractShell__ to
	// serialize the root layout's React element tree to an HTML string.
	if cfg.RootLayoutReturnsHTML {
		entry.WriteString(shellExtractorJS)
	}

	return entry.String(), nil
}

// PageAlias returns the JS identifier used for p in the generated server entry.
func PageAlias(p core.PageEntry) string {
	base := strings.TrimSuffix(filepath.Base(p.PageComponentPath), filepath.Ext(p.PageComponentPath))
	if base != "page" {
		return titleCase(base)
	}
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

// LayoutAlias returns the JS identifier prefix for a layout file.
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

// titleCase converts a string to PascalCase, handling hyphens and underscores
// so that e.g. "my-page" becomes "MyPage" (a valid JS identifier).
func titleCase(s string) string {
	if len(s) == 0 {
		return ""
	}
	var b strings.Builder
	upper := true
	for _, r := range s {
		if r == '-' || r == '_' {
			upper = true
			continue
		}
		if upper {
			b.WriteString(strings.ToUpper(string(r)))
			upper = false
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
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

func isRouteGroup(s string) bool {
	return len(s) >= 2 && s[0] == '(' && s[len(s)-1] == ')'
}

func stripParens(s string) string {
	if isRouteGroup(s) {
		return s[1 : len(s)-1]
	}
	return s
}

// isRootSegment returns true when the segment's directory is the pages root.
func isRootSegment(absPagesDir string, seg core.PageSegment) bool {
	absDir, _ := filepath.Abs(seg.Dir)
	return absDir == absPagesDir
}

// MetadataImportSuffix returns the named import clause for metadata exports,
// e.g. `{ metadata as Foo_metadata }`, or "" if neither flag is set.
func MetadataImportSuffix(alias string, hasMetadata, hasGenMetadata bool) string {
	var parts []string
	if hasMetadata {
		parts = append(parts, fmt.Sprintf("metadata as %s_metadata", alias))
	}
	if hasGenMetadata {
		parts = append(parts, fmt.Sprintf("generateMetadata as %s_generateMetadata", alias))
	}
	if len(parts) == 0 {
		return ""
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

// GenerateMetadataJS builds the __metadataChain__, __mergeMetadata__, and
// __collectMetadata__ JS blocks for the server entry.
func GenerateMetadataJS(pages []core.PageEntry, absPagesDir string) string {
	var b strings.Builder

	b.WriteString(metadataMergeJS)

	b.WriteString("const __metadataChain__: Record<string, Array<{s?: any, g?: Function}>> = {\n")
	for _, p := range pages {
		alias := PageAlias(p)
		var chain []string

		for _, seg := range p.Segments {
			if seg.LayoutPath == "" && !seg.HasMetadata && !seg.HasGenMetadata {
				continue
			}
			if seg.LayoutPath == "" {
				continue
			}
			abs, _ := filepath.Abs(seg.LayoutPath)
			la := LayoutAlias(absPagesDir, abs) + "Layout"
			if seg.HasGenMetadata {
				chain = append(chain, fmt.Sprintf("{g:%s_generateMetadata}", la))
			} else if seg.HasMetadata {
				chain = append(chain, fmt.Sprintf("{s:%s_metadata}", la))
			}
		}

		if p.HasGenMetadata {
			chain = append(chain, fmt.Sprintf("{g:%s_generateMetadata}", alias))
		} else if p.HasMetadata {
			chain = append(chain, fmt.Sprintf("{s:%s_metadata}", alias))
		}

		if len(chain) > 0 {
			fmt.Fprintf(&b, "  %s:[%s],\n", alias, strings.Join(chain, ","))
		}
	}
	b.WriteString("};\n")

	b.WriteString(metadataCollectorJS)

	return b.String()
}

const metadataMergeJS = `
function __mergeMetadata__(parent: any, child: any): any {
  if (!child || typeof child !== 'object') return parent;
  const result: any = { ...parent };
  for (const [k, v] of Object.entries(child)) {
    if (v === undefined || v === null) continue;
    if (k === 'title') {
      if (typeof v === 'string') {
        if (typeof result.title === 'object' && result.title?.template) {
          const tmpl = result.title.template;
          result.title = tmpl.includes('%s') ? tmpl.replace('%s', v) : v;
        } else {
          result.title = v;
        }
      } else {
        result.title = { ...(typeof result.title === 'object' ? result.title : {}), ...v };
      }
    } else if (k === 'openGraph' || k === 'twitter' || k === 'robots' || k === 'icons') {
      result[k] = typeof result[k] === 'object' && result[k] ? { ...result[k], ...v } : v;
    } else {
      result[k] = v;
    }
  }
  return result;
}
`

const metadataCollectorJS = `
(globalThis as any).__collectMetadata__ = async function(exportName: string, propsJSON: string): Promise<string> {
  const chain = (__metadataChain__ as any)[exportName];
  if (!chain || chain.length === 0) return JSON.stringify(null);
  const props = JSON.parse(propsJSON || "{}");
  let merged: any = {};
  for (const entry of chain) {
    try {
      let meta: any = null;
      if (entry.g) {
        meta = await entry.g(props);
      } else if (entry.s) {
        meta = entry.s;
      }
      if (meta) merged = __mergeMetadata__(merged, meta);
    } catch(e) {}
  }
  return JSON.stringify(merged);
};
`

// shellExtractorJS is appended to the generated server entry when the root
// layout returns <html>. It provides a minimal React element → HTML serializer
// and a __extractShell__ global that the Go pipeline calls at startup.
const shellExtractorJS = `
function __elementToHTML__(el: any): string {
  if (el == null || el === false || el === true) return '';
  if (typeof el === 'string' || typeof el === 'number') return String(el);
  if (Array.isArray(el)) return el.map(__elementToHTML__).join('');
  if (typeof el === 'object' && el.type == null && el.props == null) return '';
  // forwardRef / memo components (e.g. lucide-react icons) are objects carrying
  // a render fn (forwardRef) or an inner type (memo), not plain functions — so
  // without this they'd fall through to the "special types" branch below and
  // vanish from the shell. memo(forwardRef(...)) resolves via the recursion.
  if (el.type != null && typeof el.type === 'object') {
    const t: any = el.type;
    if (t.$$typeof === Symbol.for('react.forward_ref') && typeof t.render === 'function') {
      try { return __elementToHTML__(t.render(el.props || {}, null)); } catch { return ''; }
    }
    if (t.$$typeof === Symbol.for('react.memo') && t.type != null) {
      try { return __elementToHTML__({ type: t.type, props: el.props || {} }); } catch { return ''; }
    }
  }
  if (typeof el.type === 'function') {
    try { return __elementToHTML__(el.type(el.props || {})); } catch { return ''; }
  }
  if (typeof el.type === 'symbol' || typeof el.type !== 'string') {
    // Fragment or other special types — just render children
    const c = el.props?.children;
    return c != null ? __elementToHTML__(c) : '';
  }
  const tag = el.type as string;
  const props = el.props || {};
  const attrMap: Record<string,string> = { className: 'class', htmlFor: 'for', tabIndex: 'tabindex', autoFocus: 'autofocus' };
  let attrs = '';
  for (const [k, v] of Object.entries(props)) {
    if (k === 'children' || k === 'key' || k === 'ref' || k === 'dangerouslySetInnerHTML' || v == null || v === false) continue;
    if (k === 'style' && typeof v === 'object') {
      const css = Object.entries(v as Record<string,any>)
        .map(([p, val]) => p.replace(/[A-Z]/g, m => '-' + m.toLowerCase()) + ':' + val)
        .join(';');
      attrs += ' style="' + css.replace(/"/g, '&quot;') + '"';
      continue;
    }
    const name = attrMap[k] || k;
    if (v === true) { attrs += ' ' + name; continue; }
    attrs += ' ' + name + '="' + String(v).replace(/"/g, '&quot;') + '"';
  }
  const voidTags = new Set(['meta','link','br','hr','img','input','area','base','col','embed','source','track','wbr']);
  if (voidTags.has(tag)) return '<' + tag + attrs + '/>';
  let children = '';
  if (props.dangerouslySetInnerHTML?.__html) {
    children = props.dangerouslySetInnerHTML.__html;
  } else if (props.children != null) {
    children = __elementToHTML__(props.children);
  }
  return '<' + tag + attrs + '>' + children + '</' + tag + '>';
}

(globalThis as any).__extractShell__ = function(): string | null {
  try {
    if (typeof IndexLayout !== 'function') return null;
    const tree = IndexLayout({ children: '<!--POLA_CHILDREN-->' });
    if (!tree || tree.type !== 'html') return null;
    return __elementToHTML__(tree);
  } catch(e) { return null; }
};
`
