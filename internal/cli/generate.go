package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/polagonow/pola/internal/codegen"
	"github.com/spf13/cobra"
)

var generateFlags struct {
	actionsDir string
	tsOut      string
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate bridge code from actions/ directory",
	Long: `Parse Go structs in the actions/ directory and generate:
  - Go RuntimeInjector wiring (injected via -overlay, not written to your project)
  - TypeScript declaration file (typed di imports)`,
	RunE: runGenerate,
	Example: `  pola generate
  pola generate --actions-dir ./actions --ts-out ./ui/packages/di/src/generated.d.ts`,
	Aliases: []string{"gen"},
}

var generateActionCmd = &cobra.Command{
	Use:   "action [Name]",
	Short: "Scaffold a new action struct",
	Long:  "Create a new action file in the actions/ directory with boilerplate and comments.",
	Args:  cobra.ExactArgs(1),
	RunE:  runGenerateAction,
	Example: `  pola generate action Blog
  pola generate action Products`,
}

func init() {
	generateCmd.Flags().StringVar(&generateFlags.actionsDir, "actions-dir", "", "path to actions directory (default: ./actions)")
	generateCmd.Flags().StringVar(&generateFlags.tsOut, "ts-out", "", "path to generated .d.ts file")
	generateCmd.AddCommand(generateActionCmd)
}

func runGenerate(_ *cobra.Command, _ []string) error {
	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}
	// Use "tailwind" as default CSS for standalone generate (no CSS flag here).
	result, err := generateOverlay(projectDir, envOr("POLA_CSS", "tailwind"))
	if err != nil {
		return err
	}
	if result != nil && result.TmpDir != "" {
		defer os.RemoveAll(result.TmpDir)
	}
	if result != nil && result.OverlayPath != "" {
		fmt.Printf("Overlay: %s\n", result.OverlayPath)
		fmt.Println("Pass -overlay to go build/run to include the generated bridge.")
	}
	return nil
}

func runGenerateAction(_ *cobra.Command, args []string) error {
	name := args[0]
	if name[0] >= 'a' && name[0] <= 'z' {
		name = string(name[0]-32) + name[1:]
	}

	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	actionsDir := filepath.Join(projectDir, "actions")
	if err := os.MkdirAll(actionsDir, 0o755); err != nil {
		return fmt.Errorf("create actions dir: %w", err)
	}

	filename := strings.ToLower(name) + ".go"
	filePath := filepath.Join(actionsDir, filename)

	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("%s already exists", filePath)
	}

	src := fmt.Sprintf(`package actions

// %s exposes functions and variables to JavaScript via the Pola bridge.
//
// How it works:
//   - Each exported method becomes a callable function in JS
//   - Method names are automatically camelCased: GetItems → %s.getItems()
//   - Parameters are positional and type-safe (string, int, bool, structs)
//   - Return must be (T, error) or just error
//   - The special Vars() method exposes read-only constants to JS
//
// Usage in React:
//   import { %s } from "@pola/actions"
//   const items = await %s.getAll()
//
// Run "pola generate" to regenerate the bridge after editing this file.

// %s is your action struct. Add fields for dependencies (DB, services, etc.)
// that will be resolved via dependency injection.
type %s struct {
	// DB *sql.DB  // example: injected database connection
}

// Example: a simple item type returned by your methods.
// Struct fields map to TypeScript interfaces via json tags.
// type Item struct {
// 	ID    int    `+"`"+`json:"id"`+"`"+`
// 	Name  string `+"`"+`json:"name"`+"`"+`
// }

// GetAll returns all items. Becomes %s.getAll() in JavaScript.
func (a *%s) GetAll() ([]map[string]any, error) {
	// TODO: implement
	return nil, nil
}

// GetByID returns a single item by ID. Becomes %s.getByID(id) in JavaScript.
func (a *%s) GetByID(id string) (map[string]any, error) {
	// TODO: implement
	return nil, nil
}

// Vars exposes read-only values to JavaScript as properties on the action.
// Each key becomes accessible as %s.keyName in JS.
//
// func (a *%s) Vars() map[string]any {
// 	return map[string]any{
// 		"appName": "My App",
// 	}
// }
`, name, name, name, name, name, name, name, name, name, name, name, name)

	if err := os.WriteFile(filePath, []byte(src), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}

	fmt.Printf("Created %s\n", filePath)
	return nil
}

// overlayResult holds the output from generateOverlay.
type overlayResult struct {
	OverlayPath string
	TmpDir      string
	TSOutPath   string
}

// generateOverlay creates a unified overlay containing:
//  1. pola_plugins.go — blank imports for plugin self-registration (always)
//  2. generated_bridge.go — action bridge codegen (if actions/ exists)
//
// The caller should defer os.RemoveAll(result.TmpDir) after the build completes.
func generateOverlay(projectDir, css string) (*overlayResult, error) {
	tmpDir, err := os.MkdirTemp("", "pola-overlay-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	replace := make(map[string]string)

	// Determine actions directory.
	actionsDir := generateFlags.actionsDir
	if actionsDir == "" {
		actionsDir = filepath.Join(projectDir, "actions")
	}
	if !filepath.IsAbs(actionsDir) {
		actionsDir = filepath.Join(projectDir, actionsDir)
	}

	// 1. Run action bridge codegen if actions/ exists.
	var tsOutPath string
	var actionsImport string
	hasActions := false
	if info, err := os.Stat(actionsDir); err == nil && info.IsDir() {
		hasActions = true
		tsOut := generateFlags.tsOut
		if tsOut == "" {
			tsOut = filepath.Join(projectDir, "node_modules", "@pola", "actions", "src", "generated.ts")
		}
		if !filepath.IsAbs(tsOut) {
			tsOut = filepath.Join(projectDir, tsOut)
		}

		fmt.Println("Generating action bridges...")
		codegenResult, err := codegen.Run(actionsDir, tsOut, tmpDir)
		if err != nil {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("codegen: %w", err)
		}

		if codegenResult != nil && codegenResult.BridgePath != "" {
			replace[codegenResult.VirtualPath] = codegenResult.BridgePath
			tsOutPath = codegenResult.TSOutPath
		}

		if verbose && codegenResult != nil && codegenResult.TSOutPath != "" {
			fmt.Printf("Generated types: %s\n", codegenResult.TSOutPath)
		}
	} else if verbose {
		fmt.Println("No actions/ directory found, skipping codegen.")
	}

	// 2. Resolve the actions import path so pola_plugins.go can blank-import it.
	if hasActions {
		if modPath, err := readModulePath(projectDir); err == nil && modPath != "" {
			actionsImport = modPath + "/actions"
		}
	}

	// 3. Generate plugin imports (always).
	pluginsSrc := generatePluginImports(css, actionsImport)
	pluginsPath := filepath.Join(tmpDir, "pola_plugins.go")
	if err := os.WriteFile(pluginsPath, pluginsSrc, 0o644); err != nil {
		return nil, fmt.Errorf("write plugins: %w", err)
	}
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, fmt.Errorf("abs project dir: %w", err)
	}
	replace[filepath.Join(absProjectDir, "pola_plugins.go")] = pluginsPath

	// 3. Write unified overlay JSON.
	overlay := map[string]any{
		"Replace": replace,
	}
	overlayJSON, err := json.Marshal(overlay)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("marshal overlay: %w", err)
	}

	overlayPath := filepath.Join(tmpDir, "overlay.json")
	if err := os.WriteFile(overlayPath, overlayJSON, 0o644); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("write overlay: %w", err)
	}

	if verbose {
		fmt.Printf("Generated overlay: %s\n", overlayPath)
	}

	return &overlayResult{
		OverlayPath: overlayPath,
		TmpDir:      tmpDir,
		TSOutPath:   tsOutPath,
	}, nil
}

var pluginsTmpl = template.Must(template.New("plugins").Parse(`// Code generated by pola; DO NOT EDIT.
package main

import (
	_ "github.com/polagonow/pola/bundler"
	_ "github.com/polagonow/pola/cache"
{{- if .CSS}}
	_ "github.com/polagonow/pola/css"
{{- end}}
	_ "github.com/polagonow/pola/engine"
	_ "github.com/polagonow/pola/fs"
	_ "github.com/polagonow/pola/logger"
	_ "github.com/polagonow/pola/middleware/logging"
	_ "github.com/polagonow/pola/middleware/recovery"
	_ "github.com/polagonow/pola/renderer"
	_ "github.com/polagonow/pola/router"
{{- if .ActionsImport}}
	_ "{{.ActionsImport}}"
{{- end}}
)
`))

// generatePluginImports returns the source for pola_plugins.go containing
// blank imports that trigger plugin self-registration via init() → di.Stage().
func generatePluginImports(css, actionsImport string) []byte {
	var buf strings.Builder
	_ = pluginsTmpl.Execute(&buf, struct {
		CSS           bool
		ActionsImport string
	}{
		CSS:           css != "" && css != "none",
		ActionsImport: actionsImport,
	})
	return []byte(buf.String())
}

// readModulePath reads the module path from go.mod in the given directory.
func readModulePath(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}
	return "", fmt.Errorf("module directive not found in go.mod")
}
