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

func resourceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resource [Name]",
		Short: "Scaffold an MCP resource",
		Long: `Create an MCP resource struct under mcp/resources/. The generated
constructor takes *core.Registry so the resource can resolve services via DI.`,
		Args:    cobra.ExactArgs(1),
		RunE:    runResource,
		Example: `  pola generate mcp resource Config`,
	}
}

func runResource(cmd *cobra.Command, args []string) error {
	name := schema.PascalCase(args[0])

	projectDir, err := project.FindRoot()
	if err != nil {
		return err
	}
	modulePath, err := project.ModulePath(projectDir)
	if err != nil {
		return fmt.Errorf("read module path: %w", err)
	}

	dir := filepath.Join(projectDir, "mcp", "resources")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create resources dir: %w", err)
	}

	filePath := filepath.Join(dir, schema.SnakeCase(name)+"_resource.go")
	if err := generators.CheckCollision(cmd, filePath); err != nil {
		return err
	}

	data := struct {
		Name       string
		SnakeName  string
		URI        string
		ModulePath string
	}{
		Name:       name,
		SnakeName:  schema.SnakeCase(name),
		URI:        "pola://" + strings.ReplaceAll(schema.SnakeCase(name), "_", "-"),
		ModulePath: modulePath,
	}

	var buf strings.Builder
	if err := resourceTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute resource template: %w", err)
	}
	if err := os.WriteFile(filePath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}
	fmt.Printf("Created %s\n", filePath)

	pf, _ := polafile.Load(projectDir)
	if generators.ShouldGenerateTests(cmd, pf.GenerateTests()) {
		testPath := filepath.Join(dir, schema.SnakeCase(name)+"_resource_test.go")
		if err := generators.CheckCollision(cmd, testPath); err != nil {
			return err
		}
		var testBuf strings.Builder
		if err := resourceTestTmpl.Execute(&testBuf, data); err != nil {
			return fmt.Errorf("execute resource test template: %w", err)
		}
		if err := os.WriteFile(testPath, []byte(testBuf.String()), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", testPath, err)
		}
		fmt.Printf("Created %s\n", testPath)
	}

	return generators.RunAfterHooks(&Generator{}, projectDir)
}
