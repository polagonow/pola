package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/polagonow/pola/internal/generators"
	"github.com/polagonow/pola/internal/generators/model/schema"
	"github.com/polagonow/pola/internal/project"
	"github.com/spf13/cobra"
)

func promptCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prompt [Name]",
		Short: "Scaffold an MCP prompt",
		Long: `Create an MCP prompt struct under mcp/prompts/. The generated
constructor takes *core.Registry so the prompt can resolve services via DI.`,
		Args:    cobra.ExactArgs(1),
		RunE:    runPrompt,
		Example: `  pola generate mcp prompt Summarize`,
	}
}

func runPrompt(cmd *cobra.Command, args []string) error {
	name := schema.PascalCase(args[0])

	projectDir, err := project.FindRoot()
	if err != nil {
		return err
	}
	modulePath, err := project.ModulePath(projectDir)
	if err != nil {
		return fmt.Errorf("read module path: %w", err)
	}

	dir := filepath.Join(projectDir, "mcp", "prompts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create prompts dir: %w", err)
	}

	filePath := filepath.Join(dir, schema.SnakeCase(name)+"_prompt.go")
	if err := generators.CheckCollision(cmd, filePath); err != nil {
		return err
	}

	data := struct {
		Name       string
		SnakeName  string
		PromptName string
		ModulePath string
	}{
		Name:       name,
		SnakeName:  schema.SnakeCase(name),
		PromptName: strings.ReplaceAll(schema.SnakeCase(name), "_", "-"),
		ModulePath: modulePath,
	}

	var buf strings.Builder
	if err := promptTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute prompt template: %w", err)
	}
	if err := os.WriteFile(filePath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}
	fmt.Printf("Created %s\n", filePath)
	return generators.RunAfterHooks(&Generator{}, projectDir)
}
