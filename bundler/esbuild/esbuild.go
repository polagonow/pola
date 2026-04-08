// Package esbuild implements the Pola core.Bundler interface using esbuild.
//
// Build this package with the "esbuild" build tag:
//
//	go build -tags esbuild ./...
package esbuild

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/watcher"
)

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

	absAppDir, err := filepath.Abs(req.AppDir)
	if err != nil {
		return nil, fmt.Errorf("esbuild: abs appdir: %w", err)
	}

	// Probe: auto-discover "use client" files and CSS imports referenced from
	// the server entry (framework packages, third-party libraries, etc.).
	probe := probeServerEntry(req, absAppDir)
	if len(probe.clientFiles) > 0 {
		seen := make(map[string]bool, len(req.ClientComponents))
		for _, c := range req.ClientComponents {
			abs, _ := filepath.Abs(c)
			seen[abs] = true
		}
		for _, p := range probe.clientFiles {
			if !seen[p] {
				req.ClientComponents = append(req.ClientComponents, p)
				seen[p] = true
			}
		}
	}
	// Pass 1 — Client bundle (browser ESM).
	// CSS files discovered during the probe are passed as additional entries
	// so the CSS plugin can process them and esbuild emits hashed output.
	clientFiles, clientEntryOutput, metafile, cssURLs, err := buildClientBundle(req, absAppDir, probe.cssFiles)
	if err != nil {
		return nil, err
	}

	absOutDir, err := filepath.Abs(req.OutDir)
	if err != nil {
		return nil, fmt.Errorf("esbuild: abs outdir: %w", err)
	}
	inputChunkURLs := buildInputChunkURLs(metafile, absAppDir, absOutDir, req.AssetsURLPath)

	mfst, importURLs, err := buildManifest(
		req.ClientComponents, clientFiles, req.AppDir, req.AssetsURLPath, inputChunkURLs)
	if err != nil {
		return nil, fmt.Errorf("esbuild: manifest: %w", err)
	}
	manifestJSON, err := json.MarshalIndent(mfst, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("esbuild: marshal manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(req.OutDir, "manifest.json"), manifestJSON, 0o644); err != nil {
		return nil, fmt.Errorf("esbuild: write manifest: %w", err)
	}

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

	serverBundlePath, err := buildPagesBundle(req, absAppDir, string(manifestDefine), serverEntry)
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

	return &core.BundleOutput{
		ServerBundle:   serverBundle,
		ClientFiles:    clientFiles,
		ClientEntryURL: clientEntryOutput,
		ManifestJSON:   manifestJSON,
		ImportURLs:     importURLs,
		CSSURLs:        cssURLs,
	}, nil
}

// Watch polls the app directory for file changes using esbuild's two-tier
// polling algorithm (100ms interval, recent files always checked, remaining
// files randomly sampled). When a change is detected the full Build pipeline
// is re-run and the new BundleOutput is sent on the returned channel.
// The channel is closed when ctx is canceled.
func (b *Bundler) Watch(ctx context.Context, req core.BundleInput) (<-chan *core.BundleOutput, error) {
	initial, err := b.Build(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("esbuild: watch initial build: %w", err)
	}

	outputCh := make(chan *core.BundleOutput, 1)
	triggerCh := make(chan struct{}, 1)

	watchExts := req.WatchExtensions
	if len(watchExts) == 0 {
		watchExts = []string{".ts", ".tsx", ".js", ".jsx", ".css"}
	}

	poller := watcher.NewWithCollect(func() []string {
		return watcher.CollectPaths(req.AppDir, watchExts)
	}, func() {
		select {
		case triggerCh <- struct{}{}:
		default:
		}
	})
	poller.Start()

	go func() {
		defer close(outputCh)
		defer poller.Stop()

		select {
		case outputCh <- initial:
		case <-ctx.Done():
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-triggerCh:
				if req.BeforeRebuild != nil {
					req.BeforeRebuild(&req)
				}
				out, buildErr := b.Build(ctx, req)
				if buildErr != nil {
					continue
				}
				// Replace any unread previous output so consumer always
				// gets the latest build.
				select {
				case <-outputCh:
				default:
				}
				select {
				case outputCh <- out:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return outputCh, nil
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
	absAppDir, err := filepath.Abs(appDir)
	if err != nil {
		return nil, nil, fmt.Errorf("esbuild: abs appdir: %w", err)
	}
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
	req core.BundleInput, absAppDir, manifestDefineJSON, serverOutFile string,
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
				var stub string
				if req.ClientModuleStub != nil {
					stub = req.ClientModuleStub(args.Path, moduleID)
				}
				if stub == "" {
					stub = "module.exports = {};\n"
				}
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
	}
	// Apply renderer-provided defines (e.g. __CLIENT_MANIFEST__ for React).
	for k, v := range req.ServerBundleDefines {
		if v == "{}" {
			// Replace the placeholder with the real manifest.
			defines[k] = manifestDefineJSON
		} else {
			defines[k] = v
		}
	}
	// Inject POLA_PUBLIC_* env vars.
	for k, v := range req.PublicEnvVars {
		quoted, _ := json.Marshal(v)
		defines[k] = string(quoted)
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
		AbsWorkingDir:     absAppDir,
		Outfile:           serverOutFile,
		Write:             true,
		Sourcemap:         api.SourceMapNone,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		Plugins:           serverPlugins(absAppDir, req.ServerPlugins, useClientPlugin),
		Conditions:        req.ServerBundleConditions,
		Define:            defines,
	})
	if len(r.Errors) > 0 {
		return "", fmtErrors("pages", r.Errors)
	}
	return serverOutFile, nil
}

func buildClientBundle(req core.BundleInput, absDir string, cssFiles []string) (map[string][]byte, string, string, []string, error) {
	if req.AssetsURLPath == "" {
		req.AssetsURLPath = "/public/assets"
	}
	absOutDir, err := filepath.Abs(req.OutDir)
	if err != nil {
		return nil, "", "", nil, fmt.Errorf("esbuild: abs outdir: %w", err)
	}

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
	entries = append(entries, cssFiles...)

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

	// Use the CSS processing plugin when a processor is available,
	// otherwise stub CSS imports as empty JS.
	var cssPlugin api.Plugin
	if req.CSSProcessor != nil {
		cssPlugin = newCSSPlugin(req.CSSProcessor)
	} else {
		cssPlugin = newCSSStubPlugin()
	}

	clientPlugins := []api.Plugin{newAtAliasPlugin(absDir)}
	clientPlugins = append(clientPlugins, castPlugins(req.ClientPlugins)...)
	clientPlugins = append(clientPlugins, cssPlugin)

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
		AssetNames:        "[name]-[hash]",
		Conditions:        []string{"browser", "import", "module", "default"},
		Plugins:           clientPlugins,
		Metafile:          true,
		Define:            clientDefines,
	})
	if len(r.Errors) > 0 {
		return nil, "", "", nil, fmtErrors("client", r.Errors)
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

	cssURLs := extractCSSURLs(r.Metafile, absDir, absOutDir, req.AssetsURLPath)
	return files, entryOutput, r.Metafile, cssURLs, nil
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

// probeResult holds the outputs of the server entry probe pass.
type probeResult struct {
	clientFiles []string // absolute paths of "use client" files
	cssFiles    []string // absolute paths of imported .css files
}

func probeServerEntry(req core.BundleInput, absDir string) probeResult {
	if req.ServerEntryContent == "" {
		return probeResult{}
	}
	// Skip probe when no probe plugins are provided — the renderer does not
	// need client boundary discovery (e.g. Svelte, Vue).
	if req.ProbePlugins == nil {
		return probeResult{}
	}
	absAppDir, _ := filepath.Abs(req.AppDir)
	var result probeResult

	// The renderer-provided probe plugins handle framework-specific detection
	// (e.g. "use client" for React). We wrap them with a result-collecting
	// plugin that captures discovered client files.
	clientCollector := api.Plugin{
		Name: "probe-client-collector",
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
				result.clientFiles = append(result.clientFiles, args.Path)
				empty := ""
				return api.OnLoadResult{Contents: &empty, Loader: api.LoaderJS}, nil
			})
		},
	}

	// Collect CSS file paths discovered during import resolution, then stub them.
	cssCollector := api.Plugin{
		Name: "probe-css",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `\.css$`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				resolved := args.Path
				if strings.HasPrefix(args.Path, ".") {
					resolved = filepath.Join(args.ResolveDir, args.Path)
				}
				resolved = filepath.Clean(resolved)
				result.cssFiles = append(result.cssFiles, resolved)
				return api.OnResolveResult{
					Path:      args.Path,
					Namespace: "css-stub",
				}, nil
			})
			build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "css-stub"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				empty := ""
				return api.OnLoadResult{Contents: &empty, Loader: api.LoaderJS}, nil
			})
		},
	}

	plugins := []api.Plugin{newAtAliasPlugin(absDir), clientCollector}
	plugins = append(plugins, castPlugins(req.ProbePlugins)...)
	plugins = append(plugins, cssCollector)

	defines := map[string]string{
		"process.env.NODE_ENV": `"production"`,
		"__DEV__":              "false",
	}
	// Apply renderer-provided defines for the probe pass.
	for k, v := range req.ServerBundleDefines {
		defines[k] = v
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
		Plugins:       plugins,
	})
	return result
}

// ── plugin helpers ────────────────────────────────────────────────────────────

// castPlugins type-asserts a []any slice to []api.Plugin, silently skipping
// values that are not esbuild plugins.
func castPlugins(anyPlugins []any) []api.Plugin {
	plugins := make([]api.Plugin, 0, len(anyPlugins))
	for _, p := range anyPlugins {
		if ep, ok := p.(api.Plugin); ok {
			plugins = append(plugins, ep)
		}
	}
	return plugins
}

// serverPlugins assembles the plugin list for the server bundle pass:
// [atAlias] + renderer-provided plugins + [useClientPlugin] + [cssStub].
func serverPlugins(absDir string, rendererPlugins []any, useClientPlugin api.Plugin) []api.Plugin {
	plugins := []api.Plugin{newAtAliasPlugin(absDir)}
	plugins = append(plugins, castPlugins(rendererPlugins)...)
	plugins = append(plugins, useClientPlugin, newCSSStubPlugin())
	return plugins
}

// ── esbuild plugins ──────────────────────────────────────────────────────────

func newAtAliasPlugin(absProjectDir string) api.Plugin {
	return api.Plugin{
		Name: "at-alias",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `^@/`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				rel := "." + args.Path[1:]
				result := build.Resolve(rel, api.ResolveOptions{
					ResolveDir: absProjectDir,
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

// newCSSStubPlugin returns an esbuild plugin that stubs out .css imports
// as empty JS. Used by the server and probe passes where CSS output is not needed.
func newCSSStubPlugin() api.Plugin {
	return api.Plugin{
		Name: "css-stub",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `\.css$`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				return api.OnResolveResult{
					Path:      args.Path,
					Namespace: "css-stub",
				}, nil
			})
			build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "css-stub"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				empty := ""
				return api.OnLoadResult{Contents: &empty, Loader: api.LoaderJS}, nil
			})
		},
	}
}

// newCSSPlugin returns an esbuild plugin that processes .css imports through
// a core.CSS processor (e.g. Tailwind). The processor runs on each CSS file
// that esbuild discovers through its normal import resolution, and the
// processed output is returned with LoaderCSS so esbuild emits it as a
// hashed CSS output file.
func newCSSPlugin(processor core.CSS) api.Plugin {
	return api.Plugin{
		Name: "css-process",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `\.css$`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				// Skip if already in our namespace (avoid infinite recursion).
				if args.Namespace == "css-process" {
					return api.OnResolveResult{}, nil
				}
				// Resolve relative paths manually to avoid re-triggering OnResolve.
				resolved := args.Path
				if strings.HasPrefix(args.Path, ".") {
					resolved = filepath.Join(args.ResolveDir, args.Path)
				}
				resolved = filepath.Clean(resolved)
				return api.OnResolveResult{
					Path:      resolved,
					Namespace: "css-process",
				}, nil
			})

			build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "css-process"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				tmpOut := args.Path + ".pola-processed.css"
				defer os.Remove(tmpOut)

				ctx := context.Background()
				if err := processor.Process(ctx, args.Path, tmpOut); err != nil {
					return api.OnLoadResult{}, fmt.Errorf("css-process: %w", err)
				}

				processed, err := os.ReadFile(tmpOut)
				if err != nil {
					return api.OnLoadResult{}, fmt.Errorf("css-process: read output: %w", err)
				}
				contents := string(processed)
				return api.OnLoadResult{
					Contents:   &contents,
					Loader:     api.LoaderCSS,
					ResolveDir: filepath.Dir(args.Path),
				}, nil
			})
		},
	}
}

// extractCSSURLs finds all emitted CSS files in the esbuild metafile and
// returns their public URLs.
func extractCSSURLs(metafile, absDir, absOutDir, assetsURLPath string) []string {
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
	var urls []string
	for outPath := range meta.Outputs {
		if strings.HasSuffix(outPath, ".css") {
			absOut := filepath.Join(absDir, outPath)
			relOut, err := filepath.Rel(absOutDir, absOut)
			if err != nil {
				continue
			}
			urls = append(urls, assetsURLPath+"/"+filepath.ToSlash(relOut))
		}
	}
	return urls
}

func isPackageSpecifier(s string) bool {
	return s != "" && !filepath.IsAbs(s) && !strings.HasPrefix(s, ".") && !strings.HasPrefix(s, "/")
}
