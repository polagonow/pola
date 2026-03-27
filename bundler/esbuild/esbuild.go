//go:build esbuild

// Package esbuild implements the Pola core.Bundler interface using esbuild.
//
// Build this package with the "esbuild" build tag:
//
//	go build -tags esbuild ./...
//
// The init function registered under the esbuild build tag calls
// core.RegisterBundler so the framework can resolve the default bundler
// without a direct import cycle.
package esbuild

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/globals"
)

func init() { core.RegisterBundler(func() core.Bundler { return New() }) }

// Bundler implements core.Bundler using the two-pass esbuild pipeline.
type Bundler struct{}

// New returns a new esbuild Bundler.
func New() *Bundler { return &Bundler{} }

// Name returns the bundler name.
func (b *Bundler) Name() string { return "esbuild" }

// Build runs both esbuild passes and returns the combined output.
func (b *Bundler) Build(_ context.Context, req core.BundleInput) (*core.BundleOutput, error) {
	if err := os.MkdirAll(req.OutDir, 0o755); err != nil {
		return nil, fmt.Errorf("esbuild: mkdir %s: %w", req.OutDir, err)
	}

	absDir, err := filepath.Abs(".")
	if err != nil {
		return nil, fmt.Errorf("esbuild: abs cwd: %w", err)
	}

	absAppDir, _ := filepath.Abs(req.AppDir)

	// Probe: auto-discover "use client" files referenced from the server entry
	// (framework packages, third-party libraries, etc.).
	if probed := probeServerEntryClientFiles(req, absDir); len(probed) > 0 {
		seen := make(map[string]bool, len(req.ClientComponents))
		for _, c := range req.ClientComponents {
			abs, _ := filepath.Abs(c)
			seen[abs] = true
		}
		for _, p := range probed {
			if !seen[p] {
				req.ClientComponents = append(req.ClientComponents, p)
				seen[p] = true
			}
		}
	}

	// Probe: auto-discover "use server" files and compute action IDs.
	actionInfos := discoverServerActions(req, absDir, absAppDir)

	// Append server action dispatcher to server entry content.
	if len(actionInfos) > 0 {
		req.ServerEntryContent += generateActionDispatcher(actionInfos, absAppDir)
	}

	// Pass 1 — Client bundle (browser ESM).
	clientFiles, clientEntryOutput, metafile, err := buildClientBundle(req, absDir, actionInfos)
	if err != nil {
		return nil, err
	}

	absOutDir, err := filepath.Abs(req.OutDir)
	if err != nil {
		return nil, fmt.Errorf("esbuild: abs outdir: %w", err)
	}
	inputChunkURLs := buildInputChunkURLs(metafile, absDir, absOutDir, req.AssetsURLPath)

	mfst, importURLs, err := buildManifest(
		req.ClientComponents, clientFiles, req.AppDir, req.AssetsURLPath, inputChunkURLs)
	if err != nil {
		return nil, fmt.Errorf("esbuild: manifest: %w", err)
	}
	manifestJSON, _ := json.MarshalIndent(mfst, "", "  ")
	_ = os.WriteFile(filepath.Join(req.OutDir, "manifest.json"), manifestJSON, 0o644)

	// Pass 2 — Pages bundle (server conditions, CJS).
	manifestDefine, err := json.Marshal(mfst)
	if err != nil {
		return nil, fmt.Errorf("esbuild: manifest pass: %w", err)
	}

	serverEntry := req.ServerEntry
	if serverEntry == "" {
		serverEntry = filepath.Join(filepath.Dir(absOutDir), "_server.js")
	} else {
		serverEntry, err = filepath.Abs(serverEntry)
		if err != nil {
			return nil, fmt.Errorf("esbuild: abs server entry: %w", err)
		}
	}

	serverBundlePath, err := buildPagesBundle(req, absDir, absAppDir, string(manifestDefine), serverEntry)
	if err != nil {
		return nil, fmt.Errorf("esbuild: pages pass: %w", err)
	}

	var serverBundle []byte
	if serverBundlePath != "" {
		serverBundle, err = os.ReadFile(serverBundlePath)
		if err != nil {
			return nil, fmt.Errorf("esbuild: read server bundle: %w", err)
		}
	}

	// Build server action manifest.
	var actionManifest core.ServerActionManifest
	if len(actionInfos) > 0 {
		actionManifest = make(core.ServerActionManifest)
		for _, info := range actionInfos {
			for exportName, actionID := range info.actions {
				actionManifest[actionID] = core.ServerActionEntry{
					ModuleID:   info.moduleID,
					ExportName: exportName,
				}
			}
		}
	}

	return &core.BundleOutput{
		ServerBundle:         serverBundle,
		ClientFiles:          clientFiles,
		ClientEntryURL:       clientEntryOutput,
		ManifestJSON:         manifestJSON,
		ImportURLs:           importURLs,
		ServerActionManifest: actionManifest,
	}, nil
}

// Watch stubs esbuild incremental watch mode.
// TODO: implement using esbuild's context-based rebuild API.
func (b *Bundler) Watch(_ context.Context, _ core.BundleInput, _ func(*core.BundleOutput)) error {
	return fmt.Errorf("esbuild: Watch not yet implemented")
}

// ── manifest (inlined from bundler/manifest) ─────────────────────────────────

// manifestEntry is a single entry in the webpack-format client manifest.
type manifestEntry struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Chunks []string `json:"chunks"`
	Async  bool     `json:"async"`
}

// buildManifest builds a webpack-format client component manifest and an
// importURLs map from absolute component path → browser chunk URL.
func buildManifest(
	clientComponents []string, clientFiles map[string][]byte,
	appDir string, assetsURLPath string, inputChunkURLs map[string]string,
) (map[string]manifestEntry, map[string]string, error) {
	if assetsURLPath == "" {
		assetsURLPath = "/public/assets"
	}
	absAppDir, _ := filepath.Abs(appDir)
	m := make(map[string]manifestEntry)
	importURLs := make(map[string]string)

	for _, src := range clientComponents {
		absSrc, _ := filepath.Abs(src)
		id := computeModuleID(absAppDir, absSrc)
		base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))

		chunkURL := assetsURLPath + "/main.js"
		if url, ok := inputChunkURLs[absSrc]; ok {
			chunkURL = url
		} else {
			for out := range clientFiles {
				b := filepath.Base(out)
				if strings.HasPrefix(b, base+"-") || b == base+".js" {
					chunkURL = assetsURLPath + "/" + out
					break
				}
			}
		}
		importURLs[id] = chunkURL

		entry := manifestEntry{ID: id, Name: "default", Chunks: []string{"default"}}
		m[id+"#"] = entry
		m[id+"#default"] = entry
	}
	return m, importURLs, nil
}

// computeModuleID returns a stable module ID for a client component file.
// Files inside node_modules get a package-path id. App files get a path id
// relative to the nearest ancestor of absAppDir that contains absPath without
// a leading "../".
func computeModuleID(absAppDir, absPath string) string {
	if idx := strings.LastIndex(absPath, "/node_modules/"); idx != -1 {
		id := absPath[idx+len("/node_modules/"):]
		return filepath.ToSlash(strings.TrimSuffix(id, filepath.Ext(id)))
	}
	dir := absAppDir
	for {
		rel, err := filepath.Rel(dir, absPath)
		if err != nil {
			break
		}
		relSlash := strings.ReplaceAll(
			strings.TrimSuffix(rel, filepath.Ext(rel)),
			string(filepath.Separator), "/")
		if !strings.HasPrefix(relSlash, "../") {
			return relSlash
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Base(absPath)
}

// ── passes ───────────────────────────────────────────────────────────────────

func buildPagesBundle(
	req core.BundleInput, absDir, absAppDir, manifestDefineJSON, serverOutFile string,
) (string, error) {
	if req.ServerEntryContent == "" {
		return "", nil
	}
	if len(req.ServerBundleConditions) == 0 {
		return "", fmt.Errorf("esbuild: ServerBundleConditions must not be empty")
	}

	// Build set of absolute paths for "use client" proxy generation.
	clientSet := make(map[string]string)
	for _, src := range req.ClientComponents {
		abs, err := filepath.Abs(src)
		if err != nil {
			abs = src
		}
		clientSet[abs] = computeModuleID(absAppDir, abs)
	}

	useClientPlugin := api.Plugin{
		Name: "resolve-client-imports",
		Setup: func(build api.PluginBuild) {
			build.OnLoad(api.OnLoadOptions{Filter: `\.(tsx|ts|jsx|js)$`}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				moduleID, ok := clientSet[args.Path]
				if !ok {
					return api.OnLoadResult{}, nil
				}
				stub := fmt.Sprintf(
					`import { createClientModuleProxy } from "react-server-dom-webpack/server.browser";`+"\n"+
						`module.exports = createClientModuleProxy(%q);`+"\n",
					moduleID,
				)
				return api.OnLoadResult{
					Contents:   &stub,
					Loader:     api.LoaderJS,
					ResolveDir: absAppDir,
				}, nil
			})
		},
	}

	if err := os.MkdirAll(filepath.Dir(serverOutFile), 0o755); err != nil {
		return "", fmt.Errorf("esbuild: mkdir server out: %w", err)
	}

	defines := map[string]string{
		"process.env.NODE_ENV": `"production"`,
		"__DEV__":              "false",
		globals.ClientManifest: manifestDefineJSON,
	}
	// Inject POLA_PUBLIC_* env vars.
	for k, v := range req.PublicEnvVars {
		quoted, _ := json.Marshal(v)
		defines[k] = string(quoted)
	}
	for k, v := range req.ServerBundleDefines {
		defines[k] = v
	}

	r := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   req.ServerEntryContent,
			ResolveDir: absAppDir,
			Loader:     api.LoaderTSX,
			Sourcefile: "<generated-server-entry>",
		},
		Bundle:            true,
		Format:            api.FormatCommonJS,
		Platform:          api.PlatformBrowser,
		JSX:               api.JSXAutomatic,
		Target:            api.ES2020,
		External:          req.External,
		AbsWorkingDir:     absDir,
		Outfile:           serverOutFile,
		Write:             true,
		Sourcemap:         api.SourceMapNone,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		Plugins:           []api.Plugin{newAtAliasPlugin(absAppDir), newPolaWorkspacePlugin(absAppDir), useClientPlugin},
		Conditions:        req.ServerBundleConditions,
		Define:            defines,
	})
	if len(r.Errors) > 0 {
		return "", fmtErrors("pages", r.Errors)
	}
	return serverOutFile, nil
}

// newUseServerClientPlugin returns an esbuild plugin that replaces "use server"
// files with createServerReference stubs in the client bundle.
func newUseServerClientPlugin(actionInfos []serverActionInfo) api.Plugin {
	actionSet := make(map[string]*serverActionInfo)
	for i := range actionInfos {
		actionSet[actionInfos[i].absPath] = &actionInfos[i]
	}
	return api.Plugin{
		Name: "resolve-server-actions",
		Setup: func(build api.PluginBuild) {
			build.OnLoad(api.OnLoadOptions{Filter: `\.(tsx|ts|jsx|js)$`}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				info, ok := actionSet[args.Path]
				if !ok {
					return api.OnLoadResult{}, nil
				}
				var sb strings.Builder
				sb.WriteString(`import { createServerReference } from "react-server-dom-webpack/client";` + "\n")
				for exportName, actionID := range info.actions {
					fmt.Fprintf(&sb, "export const %s = createServerReference(%q, globalThis.__callServer__);\n",
						exportName, actionID)
				}
				stub := sb.String()
				return api.OnLoadResult{
					Contents: &stub,
					Loader:   api.LoaderJS,
				}, nil
			})
		},
	}
}

func buildClientBundle(req core.BundleInput, absDir string, actionInfos []serverActionInfo) (map[string][]byte, string, string, error) {
	if req.AssetsURLPath == "" {
		req.AssetsURLPath = "/public/assets"
	}
	absOutDir, err := filepath.Abs(req.OutDir)
	if err != nil {
		return nil, "", "", fmt.Errorf("esbuild: abs outdir: %w", err)
	}
	absAppDir, _ := filepath.Abs(req.AppDir)

	var entries []string
	entryBase := ""
	if req.ClientEntry != "" {
		if isPackageSpecifier(req.ClientEntry) {
			synthPath := filepath.Join(filepath.Dir(absOutDir), "_client.tsx")
			content := fmt.Sprintf("import %q;\n", req.ClientEntry)
			if werr := os.WriteFile(synthPath, []byte(content), 0o644); werr == nil {
				defer func() { _ = os.Remove(synthPath) }()
				entries = append(entries, synthPath)
				entryBase = "_client"
			}
		} else {
			entries = append(entries, req.ClientEntry)
			entryBase = strings.TrimSuffix(filepath.Base(req.ClientEntry), filepath.Ext(req.ClientEntry))
		}
	}
	entries = append(entries, req.ClientComponents...)

	clientNodeEnv := `"production"`
	clientIsDev := "false"
	if req.Dev {
		clientNodeEnv = `"development"`
		clientIsDev = "true"
	}

	// Inject POLA_PUBLIC_* vars into the client bundle.
	clientDefines := map[string]string{
		"process.env.NODE_ENV": clientNodeEnv,
		"__DEV__":              clientIsDev,
	}
	for k, v := range req.PublicEnvVars {
		quoted, _ := json.Marshal(v)
		clientDefines[k] = string(quoted)
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
		MinifyWhitespace:  !req.Dev,
		MinifyIdentifiers: !req.Dev,
		MinifySyntax:      !req.Dev,
		EntryNames:        "[name]-[hash]",
		ChunkNames:        "chunks/[name]-[hash]",
		Conditions:        []string{"browser", "import", "module", "default"},
		Plugins:           appendServerPlugin([]api.Plugin{newAtAliasPlugin(absAppDir), newPolaWorkspacePlugin(absAppDir), newAutoDedupePlugin(absAppDir)}, actionInfos),
		Metafile:          true,
		Define:            clientDefines,
	})
	if len(r.Errors) > 0 {
		return nil, "", "", fmtErrors("client", r.Errors)
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
			entryOutput = req.AssetsURLPath + "/" + rel
		}
		_ = os.MkdirAll(filepath.Dir(f.Path), 0o755)
		_ = os.WriteFile(f.Path, f.Contents, 0o644)
	}
	return files, entryOutput, r.Metafile, nil
}

func buildInputChunkURLs(metafile, absDir, absOutDir, assetsURLPath string) map[string]string {
	if metafile == "" {
		return nil
	}
	var meta struct {
		Outputs map[string]struct {
			EntryPoint string `json:"entryPoint"`
		} `json:"outputs"`
	}
	if err := json.Unmarshal([]byte(metafile), &meta); err != nil {
		return nil
	}
	if assetsURLPath == "" {
		assetsURLPath = "/public/assets"
	}
	result := make(map[string]string)
	for outPath, info := range meta.Outputs {
		if info.EntryPoint == "" {
			continue
		}
		absOut := filepath.Join(absDir, outPath)
		relOut, err := filepath.Rel(absOutDir, absOut)
		if err != nil {
			continue
		}
		absInput := filepath.Join(absDir, info.EntryPoint)
		result[absInput] = assetsURLPath + "/" + filepath.ToSlash(relOut)
	}
	return result
}

func probeServerEntryClientFiles(req core.BundleInput, absDir string) []string {
	if req.ServerEntryContent == "" {
		return nil
	}
	absAppDir, _ := filepath.Abs(req.AppDir)
	var collected []string
	probePlugin := api.Plugin{
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
				collected = append(collected, args.Path)
				empty := ""
				return api.OnLoadResult{Contents: &empty, Loader: api.LoaderJS}, nil
			})
		},
	}
	defines := map[string]string{
		"process.env.NODE_ENV": `"production"`,
		"__DEV__":              "false",
		globals.ClientManifest: "{}",
	}
	api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents: req.ServerEntryContent, ResolveDir: absAppDir,
			Loader: api.LoaderTSX, Sourcefile: "<probe>",
		},
		Bundle:        true,
		Write:         false,
		Outfile:       "_probe.js",
		Format:        api.FormatCommonJS,
		Platform:      api.PlatformBrowser,
		AbsWorkingDir: absDir,
		Conditions:    req.ServerBundleConditions,
		External:      req.External,
		Define:        defines,
		Plugins:       []api.Plugin{newAtAliasPlugin(absAppDir), newPolaWorkspacePlugin(absAppDir), probePlugin},
	})
	return collected
}

// probeServerActionFiles is the "use server" analogue of probeServerEntryClientFiles.
// It discovers files with the "use server" directive that are reachable from the
// server entry.
func probeServerActionFiles(req core.BundleInput, absDir string) []string {
	if req.ServerEntryContent == "" {
		return nil
	}
	absAppDir, _ := filepath.Abs(req.AppDir)
	var collected []string
	probePlugin := api.Plugin{
		Name: "probe-use-server",
		Setup: func(build api.PluginBuild) {
			build.OnLoad(api.OnLoadOptions{Filter: `\.(tsx|ts|jsx|js)$`}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				contents, err := os.ReadFile(args.Path)
				if err != nil {
					return api.OnLoadResult{}, nil
				}
				trimmed := strings.TrimSpace(string(contents))
				if !strings.HasPrefix(trimmed, `"use server"`) && !strings.HasPrefix(trimmed, `'use server'`) {
					return api.OnLoadResult{}, nil
				}
				collected = append(collected, args.Path)
				empty := ""
				return api.OnLoadResult{Contents: &empty, Loader: api.LoaderJS}, nil
			})
		},
	}
	defines := map[string]string{
		"process.env.NODE_ENV": `"production"`,
		"__DEV__":              "false",
		globals.ClientManifest: "{}",
	}
	api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents: req.ServerEntryContent, ResolveDir: absAppDir,
			Loader: api.LoaderTSX, Sourcefile: "<probe-server>",
		},
		Bundle:        true,
		Write:         false,
		Outfile:       "_probe_server.js",
		Format:        api.FormatCommonJS,
		Platform:      api.PlatformBrowser,
		AbsWorkingDir: absDir,
		Conditions:    req.ServerBundleConditions,
		External:      req.External,
		Define:        defines,
		Plugins:       []api.Plugin{newAtAliasPlugin(absAppDir), newPolaWorkspacePlugin(absAppDir), probePlugin},
	})
	return collected
}

// discoverExports runs a quick esbuild build to discover the export names of a file.
func discoverExports(filePath, absDir string) []string {
	r := api.Build(api.BuildOptions{
		EntryPoints:   []string{filePath},
		Bundle:        false,
		Write:         false,
		Outfile:       "_exports.js",
		Format:        api.FormatESModule,
		Platform:      api.PlatformBrowser,
		AbsWorkingDir: absDir,
		Metafile:      true,
	})
	if r.Metafile == "" {
		return nil
	}
	var meta struct {
		Outputs map[string]struct {
			Exports []string `json:"exports"`
		} `json:"outputs"`
	}
	if err := json.Unmarshal([]byte(r.Metafile), &meta); err != nil {
		return nil
	}
	for _, o := range meta.Outputs {
		if len(o.Exports) > 0 {
			return o.Exports
		}
	}
	return nil
}

// computeActionID returns a deterministic action ID for a module+export pair.
func computeActionID(moduleID, exportName string) string {
	h := sha256.Sum256([]byte(moduleID + "#" + exportName))
	return hex.EncodeToString(h[:8])
}

// serverActionInfo holds the discovered exports and computed action IDs for a
// single "use server" file.
type serverActionInfo struct {
	absPath  string
	moduleID string
	// exportName → actionID
	actions map[string]string
}

// discoverServerActions probes for "use server" files, discovers their exports,
// and computes stable action IDs.
func discoverServerActions(req core.BundleInput, absDir, absAppDir string) []serverActionInfo {
	probed := probeServerActionFiles(req, absDir)
	if len(probed) == 0 {
		return nil
	}

	var infos []serverActionInfo
	for _, p := range probed {
		modID := computeModuleID(absAppDir, p)
		exports := discoverExports(p, absDir)
		if len(exports) == 0 {
			continue
		}
		actions := make(map[string]string, len(exports))
		for _, exp := range exports {
			if exp == "default" {
				continue // skip default for now — typically not used for actions
			}
			actions[exp] = computeActionID(modID, exp)
		}
		if len(actions) == 0 {
			continue
		}
		infos = append(infos, serverActionInfo{
			absPath:  p,
			moduleID: modID,
			actions:  actions,
		})
	}
	return infos
}

// ── esbuild plugins ──────────────────────────────────────────────────────────

func newAtAliasPlugin(absAppDir string) api.Plugin {
	return api.Plugin{
		Name: "at-alias",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `^@/`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
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

func newPolaWorkspacePlugin(absAppDir string) api.Plugin {
	return api.Plugin{
		Name: "pola-workspace",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `^@pola/(react|di)(/.*)?$`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				uiRoot := findUIRoot(absAppDir)
				if uiRoot == "" {
					return api.OnResolveResult{}, nil
				}
				path := args.Path
				if strings.HasPrefix(path, "@pola/react") {
					sub := strings.TrimPrefix(path, "@pola/react")
					switch sub {
					case "", "/":
						return api.OnResolveResult{}, nil
					case "/client", "/Client":
						return api.OnResolveResult{Path: filepath.Join(uiRoot, "packages", "react", "components", "Client.tsx")}, nil
					case "/error-boundary", "/ErrorBoundary":
						return api.OnResolveResult{Path: filepath.Join(uiRoot, "packages", "react", "components", "ErrorBoundary.tsx")}, nil
					case "/types/page":
						return api.OnResolveResult{Path: filepath.Join(uiRoot, "packages", "react", "types", "page.ts")}, nil
					default:
						return api.OnResolveResult{}, nil
					}
				}
				if path == "@pola/di" {
					return api.OnResolveResult{Path: filepath.Join(uiRoot, "packages", "di", "src", "index.ts")}, nil
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

func findUIRoot(absAppDir string) string {
	dir := absAppDir
	for range 10 {
		if filepath.Base(dir) == "ui" {
			if st, err := os.Stat(filepath.Join(dir, "packages")); err == nil && st.IsDir() {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// ── helpers ──────────────────────────────────────────────────────────────────

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

// generateActionDispatcher produces TypeScript code that imports "use server"
// files, builds an action map, and exposes __callServerAction__ as a global.
func generateActionDispatcher(infos []serverActionInfo, absAppDir string) string {
	var sb strings.Builder
	sb.WriteString("\n// ── Server Actions (generated) ──\n")
	sb.WriteString(`import { registerServerReference as __registerServerReference__ } from "react-server-dom-webpack/server.browser";` + "\n")

	for i, info := range infos {
		rel, err := filepath.Rel(absAppDir, info.absPath)
		if err != nil {
			rel = info.absPath
		}
		fmt.Fprintf(&sb, "import * as __actions_%d__ from %q;\n", i, "./"+filepath.ToSlash(rel))
	}

	// Register each exported function as a server reference so React's Flight
	// encoder serialises them as action references instead of erroring.
	for i, info := range infos {
		for exportName, actionID := range info.actions {
			fmt.Fprintf(&sb, "__registerServerReference__(__actions_%d__.%s, %q, %q);\n",
				i, exportName, actionID, exportName)
		}
	}

	sb.WriteString("const __actionMap__: Record<string, Function> = {\n")
	for i, info := range infos {
		for exportName, actionID := range info.actions {
			fmt.Fprintf(&sb, "  %q: __actions_%d__.%s,\n", actionID, i, exportName)
		}
	}
	sb.WriteString("};\n")

	fmt.Fprintf(&sb, `(globalThis as any).%s = function(actionId: string, bodyStr: string): ReadableStream {
  const fn = __actionMap__[actionId];
  if (!fn) throw new Error('%s: unknown action: ' + actionId);
  const args = JSON.parse(bodyStr || "[]");
  const resultPromise = Promise.resolve(fn.apply(null, args));
  return renderToReadableStream(resultPromise, %s, {
    onError(error: unknown) {
      return error instanceof Error ? error.message : String(error);
    },
  });
};
`, globals.CallServerActionFn, globals.CallServerActionFn, globals.ClientManifest)

	return sb.String()
}

func appendServerPlugin(plugins []api.Plugin, actionInfos []serverActionInfo) []api.Plugin {
	if len(actionInfos) > 0 {
		plugins = append(plugins, newUseServerClientPlugin(actionInfos))
	}
	return plugins
}

func isPackageSpecifier(s string) bool {
	return s != "" && !filepath.IsAbs(s) && !strings.HasPrefix(s, ".") && !strings.HasPrefix(s, "/")
}
