package nextjs

import (
	"fmt"
	"path/filepath"
	"strings"

	"gojsx/framework"
	"gojsx/framework/contract"
)

// ReactRSCEntryGenerator implements framework.ServerEntryGenerator for React
// Server Components using the react-server-dom-webpack/server.browser adapter.
type ReactRSCEntryGenerator struct{}

// BundleConditions returns the esbuild Conditions for the React RSC server bundle.
func (g *ReactRSCEntryGenerator) BundleConditions() []string {
	return []string{"react-server", "browser", "module", "default"}
}

// ServerGlobalName returns the JS global the RSCFlightProtocol calls to start a render.
func (g *ReactRSCEntryGenerator) ServerGlobalName() string { return "__render__" }

// Generate produces the in-memory server-entry TypeScript source from the
// discovered page list.
func (g *ReactRSCEntryGenerator) Generate(cfg framework.EntryGenConfig) (string, error) {
	absAppDir, _ := filepath.Abs(cfg.AppDir)
	absPagesDir := filepath.Join(absAppDir, "app")

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
		entry.WriteString("import __FrameworkErrorBoundary__ from \"@gojsx/react/components/ErrorBoundary\";\n")
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

	pageHasCompanions := func(p contract.PageEntry) bool {
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

	entry.WriteString(`
(globalThis as any).__render__ = function(exportName: string, propsJSON: string): ReadableStream {
  const Page = (__pages__ as any)[exportName];
  if (!Page) throw new Error('__render__: unknown page "' + exportName + '". Known: ' + Object.keys(__pages__).join(", "));
  return renderToReadableStream(React.createElement(Page, JSON.parse(propsJSON || "{}")), __CLIENT_MANIFEST__, {
    onError(error: unknown) {
      return error instanceof Error ? error.message : String(error);
    },
  });
};
`)

	return entry.String(), nil
}
