package react

import (
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/globals"
)

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

	var entry strings.Builder

	entry.WriteString(`import React from "react";` + "\n")
	entry.WriteString(`import { renderToReadableStream } from "react-server-dom-webpack/server.browser";` + "\n")

	for _, p := range cfg.Pages {
		absFile, _ := filepath.Abs(p.PageComponentPath)
		rel, err := filepath.Rel(absAppDir, absFile)
		if err != nil {
			rel = absFile
		}
		alias := PageAlias(p)
		fmt.Fprintf(&entry, "import %s from %q;\n", alias, "./"+filepath.ToSlash(rel))
	}

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
		fmt.Fprintf(&entry, "import %s from %q;\n", alias, "./"+filepath.ToSlash(rel))
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
		globals.ClientManifest,
	})
	entry.WriteString(renderBlockBuf.String())

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

// shellExtractorJS is appended to the generated server entry when the root
// layout returns <html>. It provides a minimal React element → HTML serializer
// and a __extractShell__ global that the Go pipeline calls at startup.
const shellExtractorJS = `
function __elementToHTML__(el: any): string {
  if (el == null || el === false || el === true) return '';
  if (typeof el === 'string' || typeof el === 'number') return String(el);
  if (Array.isArray(el)) return el.map(__elementToHTML__).join('');
  if (typeof el === 'object' && el.type == null && el.props == null) return '';
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
