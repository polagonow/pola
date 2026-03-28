package codegen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// RunResult holds the output paths from a codegen run.
type RunResult struct {
	// OverlayPath is the path to the overlay JSON file that maps the
	// generated bridge into the actions/ package. Empty if no actions found.
	OverlayPath string
	// TSOutPath is the path to the generated .d.ts file.
	TSOutPath string
}

// Run is the top-level codegen entry point. It parses the actions directory,
// generates the Go bridge to a temp dir (with an overlay JSON for -overlay flag),
// and writes TypeScript declarations to tsOutPath.
func Run(actionsDir, tsOutPath string) (*RunResult, error) {
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

	// Write bridge to a temp dir (invisible to user).
	tmpDir, err := os.MkdirTemp("", "pola-bridge-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	tmpBridgePath := filepath.Join(tmpDir, "generated_bridge.go")
	if err := os.WriteFile(tmpBridgePath, goSrc, 0o644); err != nil {
		return nil, fmt.Errorf("write bridge: %w", err)
	}

	// Create overlay JSON that maps the temp file into the actions/ package.
	// Go's -overlay flag reads this to virtually inject the file.
	absActionsDir, err := filepath.Abs(actionsDir)
	if err != nil {
		return nil, fmt.Errorf("abs actions dir: %w", err)
	}
	virtualPath := filepath.Join(absActionsDir, "generated_bridge.go")

	overlay := map[string]any{
		"Replace": map[string]string{
			virtualPath: tmpBridgePath,
		},
	}
	overlayJSON, err := json.Marshal(overlay)
	if err != nil {
		return nil, fmt.Errorf("marshal overlay: %w", err)
	}

	overlayPath := filepath.Join(tmpDir, "overlay.json")
	if err := os.WriteFile(overlayPath, overlayJSON, 0o644); err != nil {
		return nil, fmt.Errorf("write overlay: %w", err)
	}

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
		OverlayPath: overlayPath,
		TSOutPath:   tsOutPath,
	}, nil
}
