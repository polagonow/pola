package nativersc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/polagonow/pola/core"
	reactrenderer "github.com/polagonow/pola/renderer/react"
	"github.com/polagonow/pola/serveraction"
)

// EntryGenConfig is the input to EntryGenerator.Generate.
type EntryGenConfig struct {
	Pages              []core.PageEntry
	AppDir             string
	GlobalErrorPath    string
	GlobalNotFoundPath string

	// RootLayoutReturnsHTML is true when the root layout contains <html,
	// indicating it returns a full HTML document. When set, the entry generator
	// emits __extractShell__ and excludes the root layout from RSC wrapping.
	RootLayoutReturnsHTML bool

	// ServerActions are the 'use server' modules discovered under AppDir. The
	// generated entry imports each module and registers its exported functions
	// in the global server-action registry so the Go handler can invoke them.
	ServerActions []ServerActionModule
}

// ServerActionModule describes one discovered 'use server' module. It is an
// alias of serveraction.Module so the entry config and the shared registry
// builder use a single type.
type ServerActionModule = serveraction.Module

// EntryGenerator produces the in-memory server-entry TypeScript source for the
// nativersc renderer. It mirrors the react renderer's Next.js-style page/layout
// wrapping (reusing react.PageAlias/LayoutAlias), but instead of importing
// react-server-dom-webpack and emitting __render__, it exposes the primitives the
// Go-driven reconciler drives: __createRoot__ and the __rsc__ classification
// helper. No react-server-dom-webpack import is emitted.
type EntryGenerator struct{}

// Generate produces the server-entry TypeScript source from the discovered pages.
func (g *EntryGenerator) Generate(cfg EntryGenConfig) (string, error) { //nolint:gocyclo
	absAppDir, _ := filepath.Abs(cfg.AppDir)
	absPagesDir := absAppDir
	if info, err := os.Stat(filepath.Join(absAppDir, "app")); err == nil && info.IsDir() {
		absPagesDir = filepath.Join(absAppDir, "app")
	}

	var entry strings.Builder

	// React only — the Flight serialization happens in Go, not via
	// react-server-dom-webpack.
	entry.WriteString(`import React from "react";` + "\n")

	for _, p := range cfg.Pages {
		absFile, _ := filepath.Abs(p.PageComponentPath)
		rel, err := filepath.Rel(absAppDir, absFile)
		if err != nil {
			rel = absFile
		}
		alias := reactrenderer.PageAlias(p)
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
		alias := reactrenderer.LayoutAlias(absPagesDir, abs) + "Layout"
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
			alias := reactrenderer.LayoutAlias(absPagesDir, abs) + "Error"
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
		alias := reactrenderer.PageAlias(p)
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
		alias := reactrenderer.PageAlias(p)
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
			if cfg.RootLayoutReturnsHTML && i == 0 && isRootSegment(absPagesDir, seg) {
				continue
			}
			if seg.ErrorPath != "" {
				abs, _ := filepath.Abs(seg.ErrorPath)
				errAlias := reactrenderer.LayoutAlias(absPagesDir, abs) + "Error"
				inner = fmt.Sprintf(
					"React.createElement(__FrameworkErrorBoundary__,{fallback:%s},%s)",
					errAlias, inner)
			}
			if seg.LayoutPath != "" {
				abs, _ := filepath.Abs(seg.LayoutPath)
				layoutAlias := reactrenderer.LayoutAlias(absPagesDir, abs) + "Layout"
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
		alias := reactrenderer.PageAlias(p)
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

	entry.WriteString(walkSupportJS)

	if cfg.RootLayoutReturnsHTML {
		entry.WriteString(shellExtractorJS)
	}

	// Server-action registry + invocation helpers (reference the __sa_N__ imports).
	if len(cfg.ServerActions) > 0 {
		entry.WriteString(serveraction.RegistryJS(cfg.ServerActions))
	}

	return entry.String(), nil
}

// isRootSegment returns true when the segment's directory is the pages root.
func isRootSegment(absPagesDir string, seg core.PageSegment) bool {
	absDir, _ := filepath.Abs(seg.Dir)
	return absDir == absPagesDir
}

// walkSupportJS installs the globals the Go-driven reconciler drives:
//   - __createRoot__(exportName, propsJSON): the root React element for a page.
//   - __rsc__: classification + server-component execution helpers. All node
//     inspection lives here so React's symbol identities are compared inside the
//     bundle's own React registry rather than across the Go/JS boundary.
//
// This is intentionally plain JavaScript (no TypeScript annotations) so it is
// valid both as esbuild TSX input and when executed directly in a goja unit test.
const walkSupportJS = `
globalThis.__createRoot__ = function (exportName, propsJSON) {
  var Page = __pages__[exportName];
  if (!Page) throw new Error('__createRoot__: unknown page: ' + exportName);
  return React.createElement(Page, JSON.parse(propsJSON || "{}"));
};

globalThis.__rsc__ = (function () {
  var ELEMENT_A = Symbol.for("react.transitional.element");
  var ELEMENT_B = Symbol.for("react.element");
  var FRAGMENT = Symbol.for("react.fragment");
  var SUSPENSE = Symbol.for("react.suspense");
  var CLIENT_REF = Symbol.for("react.client.reference");
  var SERVER_REF = Symbol.for("react.server.reference");

  function isElement(v) {
    return v != null && (v.$$typeof === ELEMENT_A || v.$$typeof === ELEMENT_B);
  }

  return {
    kind: function (v) {
      if (v === null) return "null";
      if (v === undefined) return "undefined";
      var t = typeof v;
      if (t === "string") return "string";
      if (t === "number") return "number";
      if (t === "boolean") return "boolean";
      if (t === "bigint") return "bigint";
      if (t === "function") {
        if (v.$$typeof === SERVER_REF) return "serverref";
        return "function";
      }
      if (t === "symbol") return "symbol";
      if (Array.isArray(v)) return "array";
      if (isElement(v)) {
        var type = v.type;
        var tt = typeof type;
        if (tt === "string") return "host";
        if (type === FRAGMENT) return "fragment";
        if (type === SUSPENSE) return "suspense";
        if (type && type.$$typeof === CLIENT_REF) return "client";
        return "component";
      }
      if (v.$$typeof === CLIENT_REF) return "clientref";
      if (typeof v.then === "function") return "promise";
      if (v instanceof Date) return "date";
      return "object";
    },
    clientId: function (node) {
      var refObj = node.$$typeof === CLIENT_REF ? node : node.type;
      return String(refObj.$$id);
    },
    serverRefId: function (node) {
      return String(node.$$id);
    },
    callComponent: function (node) {
      return node.type(node.props);
    },
    dateISO: function (node) {
      return node.toISOString();
    },
  };
})();
`

// shellExtractorJS is appended when the root layout returns <html>. It mirrors
// renderer/react's extractor: a minimal React element -> HTML serializer plus the
// __extractShell__ global the Go pipeline calls at startup.
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

