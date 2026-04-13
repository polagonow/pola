// Package reactesbuild provides esbuild-specific build plugins for the React
// renderer. It implements core.BundlePluginProvider, keeping the React renderer
// itself free of esbuild dependencies.
package reactesbuild

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// pluginProvider implements core.BundlePluginProvider for React + esbuild.
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

func (p *pluginProvider) ClientModuleStub() func(absPath, moduleID string) string {
	return func(_, moduleID string) string {
		return fmt.Sprintf(
			`import { createClientModuleProxy } from "react-server-dom-webpack/server.browser";`+"\n"+
				`module.exports = createClientModuleProxy(%q);`+"\n",
			moduleID,
		)
	}
}

// ── esbuild plugins ──────────────────────────────────────────────────────────

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
				// Skip CSS imports — they are literal file paths, not npm
				// modules, and re-resolving from the app root breaks them.
				if strings.HasSuffix(args.Path, ".css") {
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

// newUseClientProbePlugin returns an esbuild plugin that detects files
// starting with "use client" and stubs them out during the probe pass.
// The detected files are collected by the bundler's probe infrastructure.
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
