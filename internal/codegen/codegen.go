package codegen

import (
	"fmt"
	"os"
	"path/filepath"
)

// RunResult holds the output paths from a codegen run.
type RunResult struct {
	// BridgePath is the temp file path of the generated Go bridge.
	BridgePath string
	// VirtualPath is the path the bridge should appear at in the overlay
	// (i.e. inside the actions/ package).
	VirtualPath string
	// TSOutPath is the path to the generated .d.ts file.
	TSOutPath string
}

// Run is the top-level codegen entry point. It parses the actions directory,
// generates the Go bridge into tmpDir, and writes TypeScript declarations to
// tsOutPath. The caller is responsible for creating/cleaning up tmpDir and
// building the overlay JSON.
func Run(actionsDir, tsOutPath, tmpDir string) (*RunResult, error) {
	result, err := Parse(actionsDir)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	if len(result.Actions) == 0 {
		return &RunResult{}, nil
	}

	// Generate Go bridge source.
	goSrc, err := GenerateGo(result)
	if err != nil {
		return nil, fmt.Errorf("generate go: %w", err)
	}

	// Write bridge to the caller-provided temp dir.
	tmpBridgePath := filepath.Join(tmpDir, "generated_bridge.go")
	if err := os.WriteFile(tmpBridgePath, goSrc, 0o644); err != nil {
		return nil, fmt.Errorf("write bridge: %w", err)
	}

	// Compute the virtual path (where it should appear in the actions/ package).
	absActionsDir, err := filepath.Abs(actionsDir)
	if err != nil {
		return nil, fmt.Errorf("abs actions dir: %w", err)
	}
	virtualPath := filepath.Join(absActionsDir, "generated_bridge.go")

	// Generate TypeScript declarations (this one goes to the user's project).
	tsSrc, err := GenerateTS(result)
	if err != nil {
		return nil, fmt.Errorf("generate ts: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(tsOutPath), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", filepath.Dir(tsOutPath), err)
	}
	if err := os.WriteFile(tsOutPath, tsSrc, 0o644); err != nil {
		return nil, fmt.Errorf("write %s: %w", tsOutPath, err)
	}

	return &RunResult{
		BridgePath:  tmpBridgePath,
		VirtualPath: virtualPath,
		TSOutPath:   tsOutPath,
	}, nil
}
