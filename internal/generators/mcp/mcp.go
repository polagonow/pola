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
	"fmt"
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
	toolTestTmpl = template.Must(
		template.New("tool_test_go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/tool_test_go.tmpl"),
	)
	resourceTestTmpl = template.Must(
		template.New("resource_test_go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/resource_test_go.tmpl"),
	)
	promptTestTmpl = template.Must(
		template.New("prompt_test_go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/prompt_test_go.tmpl"),
	)
)

// Generator scaffolds MCP tools, resources, and prompts. It registers itself
// via init() into the global generator registry.
type Generator struct{}

func (g *Generator) Name() string { return "mcp" }

func (g *Generator) Description() string {
	return "Scaffold MCP tools, resources, and prompts"
}

func (g *Generator) AfterHooks() []generators.Hook {
	return []generators.Hook{
		generators.CmdHook("gofmt", "-w", "."),
	}
}

func (g *Generator) Artifacts(cmd *cobra.Command, args []string, projectDir string) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("mcp subcommand is required (tool, resource, or prompt)")
	}
	sub := args[0]
	subArgs := args[1:]
	switch sub {
	case "tool":
		return toolArtifacts(cmd, subArgs, projectDir)
	case "resource":
		return resourceArtifacts(cmd, subArgs, projectDir)
	case "prompt":
		return promptArtifacts(cmd, subArgs, projectDir)
	case "init":
		return nil, fmt.Errorf("mcp init modifies Polafile and cannot be reversed automatically")
	default:
		return nil, fmt.Errorf("unknown mcp subcommand %q; supported: tool, resource, prompt", sub)
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
