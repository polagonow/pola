package codegen

import (
	"fmt"
	"os"
	"path/filepath"
)

// Run is the top-level codegen entry point. It parses the actions directory,
// generates the Go bridge and TypeScript declaration files.
//
// actionsDir: absolute path to the actions/ directory
// tsOutPath: absolute path to the generated .d.ts file (e.g. ui/packages/di/src/generated.d.ts)
func Run(actionsDir, tsOutPath string) error {
	// Parse the actions package.
	result, err := Parse(actionsDir)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	if len(result.Actions) == 0 {
		return nil // nothing to generate
	}

	// Generate Go bridge.
	goSrc, err := GenerateGo(result)
	if err != nil {
		return fmt.Errorf("generate go: %w", err)
	}

	goOutPath := filepath.Join(actionsDir, "generated_bridge.go")
	if err := os.WriteFile(goOutPath, goSrc, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", goOutPath, err)
	}

	// Generate TypeScript declarations.
	tsSrc, err := GenerateTS(result)
	if err != nil {
		return fmt.Errorf("generate ts: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(tsOutPath), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(tsOutPath), err)
	}
	if err := os.WriteFile(tsOutPath, tsSrc, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tsOutPath, err)
	}

	return nil
}
