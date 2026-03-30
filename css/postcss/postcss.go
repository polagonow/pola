// Package postcss provides a CSS plugin that runs the PostCSS CLI.
// It supports both standalone binary and npx execution modes.
//
// Import this package with the postcss build tag to enable PostCSS processing:
//
//	import _ "github.com/polagonow/pola/css/postcss"
package postcss

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/polagonow/pola/core"
)

// PostCSS runs the postcss CLI to process CSS.
type PostCSS struct {
	// Bin is the path to the postcss CLI binary.
	Bin    string
	logger core.Logger
}

// SetLogger implements core.LogAware.
func (p *PostCSS) SetLogger(l core.Logger) { p.logger = l }

// New creates a PostCSS processor.
func New() *PostCSS { return &PostCSS{} }

// Ensure PostCSS satisfies core.CSS.
var _ core.CSS = (*PostCSS)(nil)

// Name returns the CSS implementation name.
func (p *PostCSS) Name() string { return "postcss" }

// Process runs postcss to process inputPath → outputPath.
func (p *PostCSS) Process(ctx context.Context, inputPath, outputPath string) error {
	bin, binArgs := p.resolvedBin(inputPath)
	args := append(binArgs, inputPath, "-o", outputPath)
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = filepath.Dir(inputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("postcss: %w", err)
	}
	return nil
}

// Watch runs postcss in watch mode (--watch), calling onChange on each rebuild.
// The watcher process is terminated when ctx is cancelled.
func (p *PostCSS) Watch(ctx context.Context, inputPath, outputPath string, onChange func()) error {
	bin, binArgs := p.resolvedBin(inputPath)
	args := append(binArgs, inputPath, "-o", outputPath, "--watch")
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = filepath.Dir(inputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	go func() {
		if err := cmd.Run(); err != nil && ctx.Err() == nil {
			if p.logger != nil {
				p.logger.Error("postcss: watch error", "err", err)
			}
		}
	}()
	return nil
}

// resolvedBin returns the binary and any extra args needed to invoke postcss.
// It checks: explicit Bin field → PATH → node_modules/.bin/ near inputPath → npx.
func (p *PostCSS) resolvedBin(inputPath string) (string, []string) {
	if p.Bin != "" {
		if _, err := exec.LookPath(p.Bin); err == nil {
			return p.Bin, nil
		}
	}
	if _, err := exec.LookPath("postcss"); err == nil {
		return "postcss", nil
	}
	// Walk up from inputPath looking for node_modules/.bin/postcss.
	dir := filepath.Dir(inputPath)
	for range 10 {
		candidate := filepath.Join(dir, "node_modules", ".bin", "postcss")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "npx", []string{"postcss"}
}
