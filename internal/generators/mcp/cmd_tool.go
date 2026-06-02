package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/polagonow/pola/internal/generators"
	"github.com/polagonow/pola/internal/generators/model/schema"
	"github.com/polagonow/pola/internal/project"
	"github.com/polagonow/pola/polafile"
	"github.com/spf13/cobra"
)

func toolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tool [Name]",
		Short: "Scaffold an MCP tool",
		Long: `Create an MCP tool struct under mcp/tools/. The default flavor declares a
constructor that takes *core.Registry so the tool can resolve services via DI.

Use --no-di for an init()-registered tool that needs no dependencies.`,
		Args: cobra.ExactArgs(1),
		RunE: runTool,
		Example: `  pola generate mcp tool Users
  pola generate mcp tool Greet --no-di`,
	}
	cmd.Flags().Bool("no-di", false, "scaffold an init()-registered tool with no DI")
	return cmd
}

func runTool(cmd *cobra.Command, args []string) error {
	name := schema.PascalCase(args[0])

	projectDir, err := project.FindRoot()
	if err != nil {
		return err
	}
	modulePath, err := project.ModulePath(projectDir)
	if err != nil {
		return fmt.Errorf("read module path: %w", err)
	}

	toolsDir := filepath.Join(projectDir, "mcp", "tools")
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		return fmt.Errorf("create tools dir: %w", err)
	}

	filePath := filepath.Join(toolsDir, schema.SnakeCase(name)+"_tool.go")
	if err := generators.CheckCollision(cmd, filePath); err != nil {
		return err
	}

	noDI, _ := cmd.Flags().GetBool("no-di")

	data := struct {
		Name        string
		SnakeName   string
		ToolName    string
		ModulePath  string
	}{
		Name:       name,
		SnakeName:  schema.SnakeCase(name),
		ToolName:   strings.ReplaceAll(schema.SnakeCase(name), "_", "-"),
		ModulePath: modulePath,
	}

	var buf strings.Builder
	tmpl := toolDITmpl
	if noDI {
		tmpl = toolInitTmpl
	}
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute tool template: %w", err)
	}
	if err := os.WriteFile(filePath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}
	fmt.Printf("Created %s\n", filePath)

	// Tests only make sense for the DI-flavored tool — the init() flavor
	// registers globally and has no exported constructor to call from a test.
	if !noDI {
		pf, _ := polafile.Load(projectDir)
		if generators.ShouldGenerateTests(cmd, pf.GenerateTests()) {
			testPath := filepath.Join(toolsDir, schema.SnakeCase(name)+"_tool_test.go")
			if err := generators.CheckCollision(cmd, testPath); err != nil {
				return err
			}
			var testBuf strings.Builder
			if err := toolTestTmpl.Execute(&testBuf, data); err != nil {
				return fmt.Errorf("execute tool test template: %w", err)
			}
			if err := os.WriteFile(testPath, []byte(testBuf.String()), 0o644); err != nil {
				return fmt.Errorf("write %s: %w", testPath, err)
			}
			fmt.Printf("Created %s\n", testPath)
		}
	}

	return generators.RunAfterHooks(&Generator{}, projectDir)
}
