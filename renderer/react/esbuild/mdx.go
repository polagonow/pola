package reactesbuild

import (
	"os"
	"path/filepath"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/polagonow/pola/renderer/mdx"
)

// newMDXLoaderPlugin returns an esbuild plugin that compiles `.md`/`.mdx`
// content files into React modules via pola's Node-free goldmark pipeline
// (renderer/mdx). It is composed into all three React build passes
// (client/server/probe) so a content import resolves identically everywhere;
// esbuild's OnLoad overrides the loader map, so no `.mdx` entry is needed there.
//
// The returned module's default export is the page body component and its named
// exports (frontmatter/toc/structuredData) satisfy the fumadocs `.source`
// contract. For non-content files the filter simply does not match, so this is a
// no-op for ordinary `.tsx`/`.ts` sources and other examples are unaffected.
func newMDXLoaderPlugin() api.Plugin {
	return api.Plugin{
		Name: "pola-mdx",
		Setup: func(build api.PluginBuild) {
			build.OnLoad(api.OnLoadOptions{Filter: `\.mdx?$`}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				src, err := os.ReadFile(args.Path)
				if err != nil {
					return api.OnLoadResult{}, err
				}
				res, cerr := mdx.Compile(src, mdx.Options{})
				if cerr != nil {
					return api.OnLoadResult{}, cerr
				}
				contents := res.TSX
				return api.OnLoadResult{
					Contents:   &contents,
					Loader:     api.LoaderJS,
					ResolveDir: filepath.Dir(args.Path),
				}, nil
			})
		},
	}
}

// fumadocsSourceNamespace tags the generated content-source virtual module.
const fumadocsSourceNamespace = "pola-fumadocs-source"

// newFumadocsSourcePlugin resolves the bare specifier `@pola/fumadocs/source` to
// a Go-generated virtual module that lists every `content/docs/**` page and
// `meta.json`, shaped for fumadocs-core/source's loader(). This is the Node-free
// stand-in for fumadocs-mdx's `.source` output. It walks `appDir/content/docs`
// (appDir is the web root); a non-docs app that never imports the specifier is
// unaffected.
func newFumadocsSourcePlugin(appDir string) api.Plugin {
	return api.Plugin{
		Name: "pola-fumadocs-source",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `^@pola/fumadocs/source$`}, func(_ api.OnResolveArgs) (api.OnResolveResult, error) {
				return api.OnResolveResult{Path: "@pola/fumadocs/source", Namespace: fumadocsSourceNamespace}, nil
			})
			build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: fumadocsSourceNamespace}, func(_ api.OnLoadArgs) (api.OnLoadResult, error) {
				contentDir := filepath.Join(appDir, "content", "docs")
				src, err := mdx.GenerateSourceModule(contentDir)
				if err != nil {
					return api.OnLoadResult{}, err
				}
				return api.OnLoadResult{
					Contents:   &src,
					Loader:     api.LoaderJS,
					ResolveDir: contentDir,
				}, nil
			})
		},
	}
}
