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

func promptArtifacts(cmd *cobra.Command, args []string, projectDir string) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("prompt name is required")
	}
	name := schema.PascalCase(args[0])
	dir := filepath.Join(projectDir, "mcp", "prompts")
	snake := schema.SnakeCase(name)
	paths := []string{filepath.Join(dir, snake+"_prompt.go")}

	pf, _ := polafile.Load(projectDir)
	if generators.ShouldGenerateTests(cmd, pf.GenerateTests()) {
		paths = append(paths, filepath.Join(dir, snake+"_prompt_test.go"))
	}
	return paths, nil
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

	pf, _ := polafile.Load(projectDir)
	if generators.ShouldGenerateTests(cmd, pf.GenerateTests()) {
		testPath := filepath.Join(dir, schema.SnakeCase(name)+"_prompt_test.go")
		if err := generators.CheckCollision(cmd, testPath); err != nil {
			return err
		}
		var testBuf strings.Builder
		if err := promptTestTmpl.Execute(&testBuf, data); err != nil {
			return fmt.Errorf("execute prompt test template: %w", err)
		}
		if err := os.WriteFile(testPath, []byte(testBuf.String()), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", testPath, err)
		}
		fmt.Printf("Created %s\n", testPath)
	}

	return generators.RunAfterHooks(&Generator{}, projectDir)
}
