// Package mcp implements the `pola generate mcp …` scaffold generator.
//
// It provides four subcommands:
//
//	pola generate mcp init                - add an mcp { … } block to Polafile so the autoload wires the plugin.
//	pola generate mcp tool <Name>         - scaffold an MCP tool. Defaults to DI flavor.
//	pola generate mcp resource <Name>     - scaffold an MCP resource.
//	pola generate mcp prompt <Name>       - scaffold an MCP prompt.
package mcp

import (
	"embed"
	"text/template"

	"github.com/polagonow/pola/internal/generators"
	"github.com/spf13/cobra"
)

//go:embed all:_templates
var templates embed.FS

var (
	toolDITmpl = template.Must(
		template.New("tool_di_go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/tool_di_go.tmpl"),
	)
	toolInitTmpl = template.Must(
		template.New("tool_init_go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/tool_init_go.tmpl"),
	)
	resourceTmpl = template.Must(
		template.New("resource_go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/resource_go.tmpl"),
	)
	promptTmpl = template.Must(
		template.New("prompt_go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/prompt_go.tmpl"),
	)
)

// Generator scaffolds MCP tools, resources, and prompts. It registers itself
// via init() into the global generator registry.
type Generator struct{}

func init() {
	generators.Register(&Generator{})
}

func (g *Generator) Name() string { return "mcp" }

func (g *Generator) Description() string {
	return "Scaffold MCP tools, resources, and prompts"
}

func (g *Generator) AfterHooks() []generators.Hook {
	return []generators.Hook{
		generators.CmdHook("gofmt", "-w", "."),
	}
}

func (g *Generator) Command() *cobra.Command {
	root := &cobra.Command{
		Use:   "mcp",
		Short: g.Description(),
		Long: `Scaffold Model Context Protocol artifacts.

Subcommands:
  init      Add an mcp { … } block to Polafile so the autoload wires the plugin.
  tool      Create an MCP tool. Use --no-di for a simple init()-registered tool.
  resource  Create an MCP resource.
  prompt    Create an MCP prompt.`,
	}
	root.AddCommand(initCmd())
	root.AddCommand(toolCmd())
	root.AddCommand(resourceCmd())
	root.AddCommand(promptCmd())
	return root
}
