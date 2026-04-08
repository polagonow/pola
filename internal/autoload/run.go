package autoload

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Run creates the overlay by iterating all registered autoloads in priority
// order. The caller should defer os.RemoveAll(result.TmpDir) after use.
func Run(projectDir string, opts PluginOpts, verbose bool) (*Result, error) {
	tmpDir, err := os.MkdirTemp("", "pola-overlay-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	modPath, _ := ReadModulePath(projectDir)

	ctx := &Context{
		ProjectDir: projectDir,
		TmpDir:     tmpDir,
		ModPath:    modPath,
		Opts:       opts,
		Replace:    make(map[string]string),
		Discovery:  &Discovery{},
		Verbose:    verbose,
	}

	for _, a := range All() {
		if err := a.Contribute(ctx); err != nil {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("autoload %s: %w", a.Name(), err)
		}
	}

	// Write unified overlay JSON.
	overlay := map[string]any{
		"Replace": ctx.Replace,
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

	return &Result{
		OverlayPath: overlayPath,
		TmpDir:      tmpDir,
		TSOutPath:   ctx.Discovery.TSOutPath,
	}, nil
}
