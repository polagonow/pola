// Package mcp discovers MCP tool/resource/prompt constructors under
// mcp/tools, mcp/resources, mcp/prompts and emits per-package overlay files
// containing a <Name>Plugin() function for each entry.
//
// Generated plugins resolve *mcp.Server from the DI container in their Start
// hook and register the constructed provider with it. The order is enforced
// by pluginimports: mcp.Plugin() is placed before any of these generated
// plugins so *mcp.Server is available when they run.
package mcp

import (
	"embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/polagonow/pola/internal/autoload"
	"github.com/polagonow/pola/internal/generators/model/schema"
)

//go:embed _templates/*.tmpl
var templates embed.FS

var (
	toolsTmpl     = template.Must(template.New("tool_plugins_go.tmpl").ParseFS(templates, "_templates/tool_plugins_go.tmpl"))
	resourcesTmpl = template.Must(template.New("resource_plugins_go.tmpl").ParseFS(templates, "_templates/resource_plugins_go.tmpl"))
	promptsTmpl   = template.Must(template.New("prompt_plugins_go.tmpl").ParseFS(templates, "_templates/prompt_plugins_go.tmpl"))
)

type autoloadImpl struct{}

// New returns this autoload stage for explicit registration in autoload/all.
func New() autoload.Autoload { return &autoloadImpl{} }

func (a *autoloadImpl) Name() string  { return "mcp" }
func (a *autoloadImpl) Priority() int { return 500 }

func (a *autoloadImpl) Contribute(ctx *autoload.Context) error {
	if ctx.ModPath == "" || !ctx.Opts.HasMCP {
		return nil
	}

	disco := &autoload.MCPDiscovery{ModulePath: ctx.Opts.PolaPackage}

	tools := discover(ctx.ProjectDir, "tools", "Tool")
	res := discover(ctx.ProjectDir, "resources", "Resource")
	prompts := discover(ctx.ProjectDir, "prompts", "Prompt")

	disco.Tools = tools
	disco.Resources = res
	disco.Prompts = prompts

	if len(tools) > 0 {
		disco.ToolsImport = ctx.ModPath + "/mcp/tools"
		if err := emit(ctx, "tools", "tools", toolsTmpl, tools); err != nil {
			return err
		}
	}
	if len(res) > 0 {
		disco.ResourceImport = ctx.ModPath + "/mcp/resources"
		if err := emit(ctx, "resources", "resources", resourcesTmpl, res); err != nil {
			return err
		}
	}
	if len(prompts) > 0 {
		disco.PromptImport = ctx.ModPath + "/mcp/prompts"
		if err := emit(ctx, "prompts", "prompts", promptsTmpl, prompts); err != nil {
			return err
		}
	}

	ctx.Discovery.MCPDisco = disco
	return nil
}

// discover scans <projectDir>/mcp/<sub> for exported functions matching
// New<Name><Suffix> with a single *core.Registry parameter. Returns the
// PluginEntry list sorted by name.
func discover(projectDir, sub, suffix string) []autoload.PluginEntry {
	absDir := filepath.Join(projectDir, "mcp", sub)
	info, err := os.Stat(absDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil
	}

	var out []autoload.PluginEntry
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(absDir, e.Name()), nil, 0)
		if err != nil {
			continue
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv != nil || !fd.Name.IsExported() {
				continue
			}
			name := fd.Name.Name
			if !strings.HasPrefix(name, "New") || !strings.HasSuffix(name, suffix) {
				continue
			}
			short := strings.TrimPrefix(name, "New")
			short = strings.TrimSuffix(short, suffix)
			if short == "" {
				continue
			}
			out = append(out, autoload.PluginEntry{
				Name:      short,
				SnakeName: schema.SnakeCase(short),
			})
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// emit writes a per-package plugin overlay file into ctx.TmpDir and adds
// a Replace entry mapping <absDir>/pola_plugins.go → tmp path.
func emit(ctx *autoload.Context, sub, pkgName string, tmpl *template.Template, entries []autoload.PluginEntry) error {
	data := struct {
		PolaPackage string
		PkgName     string
		Suffix      string
		Entries     []autoload.PluginEntry
	}{
		PolaPackage: ctx.Opts.PolaPackage,
		PkgName:     pkgName,
		Suffix:      strings.Title(strings.TrimSuffix(sub, "s")), //nolint:staticcheck
		Entries:     entries,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute %s plugins template: %w", sub, err)
	}

	overlayPath := filepath.Join(ctx.TmpDir, "pola_mcp_"+sub+"_plugins.go")
	if err := os.WriteFile(overlayPath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write %s plugins overlay: %w", sub, err)
	}

	absDir, _ := filepath.Abs(filepath.Join(ctx.ProjectDir, "mcp", sub))
	ctx.Replace[filepath.Join(absDir, "pola_plugins.go")] = overlayPath
	return nil
}
