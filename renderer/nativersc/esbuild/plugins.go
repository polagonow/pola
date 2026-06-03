package nativerscesbuild

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// pluginProvider implements core.BundlePluginProvider for nativersc + esbuild.
// It mirrors the React provider (workspace resolution, auto-dedupe, "use client"
// probe) since nativersc reuses the same client runtime and client-reference
// markers. The only difference is the fallback client-module stub, which avoids
// react-server-dom-webpack.
type pluginProvider struct{}

func (p *pluginProvider) ClientPlugins(appDir string) []any {
	return []any{
		newPolaWorkspacePlugin(appDir),
		newAutoDedupePlugin(appDir),
	}
}

func (p *pluginProvider) ServerPlugins(appDir string) []any {
	return []any{
		newPolaWorkspacePlugin(appDir),
	}
}

func (p *pluginProvider) ProbePlugins(appDir string) []any {
	return []any{
		newPolaWorkspacePlugin(appDir),
		newUseClientProbePlugin(),
	}
}

// ClientModuleStub returns the fallback stub for client modules whose exports
// could not be discovered from the client metafile. Unlike the React provider it
// does not import react-server-dom-webpack: it builds the client-reference marker
// directly. The nativersc reconciler reads the same {$$typeof, $$id} shape.
//
// Note: the common path (known exports) is handled inside the esbuild bundler's
// buildESMClientStub, which still uses createClientModuleProxy. Fully removing
// react-server-dom-webpack from the server bundle requires an additive bundler
// change and is tracked separately.
func (p *pluginProvider) ClientModuleStub() func(absPath, moduleID string) string {
	return func(_, moduleID string) string {
		return fmt.Sprintf(`module.exports = new Proxy({}, {
  get: function (_t, name) {
    if (name === "$$typeof") return undefined;
    if (name === "then") return undefined;
    if (name === "__esModule") return true;
    if (typeof name === "symbol") return undefined;
    return { $$typeof: Symbol.for("react.client.reference"), $$id: %q + "#" + name, $$async: false };
  }
});
`, moduleID)
	}
}

// ── esbuild plugins (copied from renderer/react/esbuild) ─────────────────────

func newPolaWorkspacePlugin(absAppDir string) api.Plugin {
	return api.Plugin{
		Name: "pola-workspace",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `^@pola/(react|actions)(/.*)?$`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				pkgRoot := findPolaPkgRoot(absAppDir)
				if pkgRoot == "" {
					return api.OnResolveResult{}, nil
				}
				path := args.Path
				if strings.HasPrefix(path, "@pola/react") {
					sub := strings.TrimPrefix(path, "@pola/react")
					switch sub {
					case "", "/":
						return api.OnResolveResult{}, nil
					case "/client", "/Client":
						return api.OnResolveResult{Path: filepath.Join(pkgRoot, "react", "components", "Client.tsx")}, nil
					case "/error-boundary", "/ErrorBoundary":
						return api.OnResolveResult{Path: filepath.Join(pkgRoot, "react", "components", "ErrorBoundary.tsx")}, nil
					case "/link", "/Link":
						return api.OnResolveResult{Path: filepath.Join(pkgRoot, "react", "components", "Link.tsx")}, nil
					case "/types/page":
						return api.OnResolveResult{Path: filepath.Join(pkgRoot, "react", "types", "page.ts")}, nil
					default:
						return api.OnResolveResult{}, nil
					}
				}
				if path == "@pola/actions" {
					return api.OnResolveResult{Path: filepath.Join(pkgRoot, "actions", "src", "index.ts")}, nil
				}
				return api.OnResolveResult{}, nil
			})
		},
	}
}

func newAutoDedupePlugin(absAppDir string) api.Plugin {
	return api.Plugin{
		Name: "auto-dedupe",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `^[^./]`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				if !strings.Contains(filepath.ToSlash(args.ResolveDir), "/node_modules/") {
					return api.OnResolveResult{}, nil
				}
				if strings.HasSuffix(args.Path, ".css") {
					return api.OnResolveResult{}, nil
				}
				if strings.HasPrefix(args.Path, "#") ||
					strings.HasPrefix(args.Path, "data:") ||
					strings.HasPrefix(args.Path, "http:") ||
					strings.HasPrefix(args.Path, "https:") {
					return api.OnResolveResult{}, nil
				}
				result := build.Resolve(args.Path, api.ResolveOptions{
					ResolveDir: absAppDir,
					Kind:       args.Kind,
				})
				if len(result.Errors) == 0 && result.Path != "" {
					return api.OnResolveResult{Path: result.Path}, nil
				}
				return api.OnResolveResult{}, nil
			})
		},
	}
}

// newUseClientProbePlugin detects files starting with "use client" and stubs
// them during the probe pass so the bundler can collect client boundaries.
func newUseClientProbePlugin() api.Plugin {
	return api.Plugin{
		Name: "probe-use-client",
		Setup: func(build api.PluginBuild) {
			build.OnLoad(api.OnLoadOptions{Filter: `\.(tsx|ts|jsx|js)$`}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				contents, err := os.ReadFile(args.Path)
				if err != nil {
					return api.OnLoadResult{}, nil
				}
				trimmed := strings.TrimSpace(string(contents))
				if !strings.HasPrefix(trimmed, `"use client"`) && !strings.HasPrefix(trimmed, `'use client'`) {
					return api.OnLoadResult{}, nil
				}
				empty := ""
				return api.OnLoadResult{Contents: &empty, Loader: api.LoaderJS}, nil
			})
		},
	}
}

// findPolaPkgRoot walks up from absAppDir looking for node_modules/@pola.
func findPolaPkgRoot(absAppDir string) string {
	dir := absAppDir
	for {
		candidate := filepath.Join(dir, "node_modules", "@pola")
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
