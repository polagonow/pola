//go:build tailwind

// Package tailwind provides a CSS plugin that runs the TailwindCSS CLI.
// It supports both standalone binary and npx execution modes.
//
// Import this package with the tailwind build tag to enable TailwindCSS processing:
//
//	import _ "github.com/polagonow/pola/css/tailwind"
package tailwind

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/polagonow/pola/core"
)

func init() { core.RegisterCSS(func() core.CSS { return New() }) }

// Tailwind runs the tailwindcss CLI to process CSS.
type Tailwind struct {
	// Bin is the path to the tailwindcss CLI binary (default: "tailwindcss").
	Bin string
	// ConfigPath is the tailwind config file path (optional, for v3; v4 is CSS-first).
	ConfigPath string
	logger     core.Logger
}

// SetLogger implements core.LogAware.
func (t *Tailwind) SetLogger(l core.Logger) { t.logger = l }

// New creates a Tailwind CSS processor.
func New() *Tailwind { return &Tailwind{} }

// Ensure Tailwind satisfies core.CSS.
var _ core.CSS = (*Tailwind)(nil)

// Name returns the CSS implementation name.
func (t *Tailwind) Name() string { return "tailwind" }

// Process runs tailwindcss to process inputPath → outputPath.
// In production mode (POLA_DEV != "true") it minifies the output.
func (t *Tailwind) Process(ctx context.Context, inputPath, outputPath string) error {
	bin, binArgs := t.resolvedBin(inputPath)
	args := append(binArgs, "-i", inputPath, "-o", outputPath)
	if t.ConfigPath != "" {
		args = append(args, "--config", t.ConfigPath)
	}
	if os.Getenv("POLA_DEV") != "true" {
		args = append(args, "--minify")
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = filepath.Dir(inputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tailwind: %w", err)
	}
	return nil
}

// Watch runs tailwindcss in watch mode (--watch), calling onChange on each rebuild.
// The watcher process is terminated when ctx is cancelled.
func (t *Tailwind) Watch(ctx context.Context, inputPath, outputPath string, onChange func()) error {
	bin, binArgs := t.resolvedBin(inputPath)
	args := append(binArgs, "-i", inputPath, "-o", outputPath, "--watch")
	if t.ConfigPath != "" {
		args = append(args, "--config", t.ConfigPath)
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = filepath.Dir(inputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	go func() {
		if err := cmd.Run(); err != nil && ctx.Err() == nil {
			if t.logger != nil {
				t.logger.Error("tailwind: watch error", "err", err)
			}
		}
	}()
	return nil
}

// resolvedBin returns the binary and any extra args needed to invoke tailwindcss.
// It checks: explicit Bin field → PATH → node_modules/.bin/ near inputPath → npx.
func (t *Tailwind) resolvedBin(inputPath string) (string, []string) {
	if t.Bin != "" {
		if _, err := exec.LookPath(t.Bin); err == nil {
			return t.Bin, nil
		}
	}
	if _, err := exec.LookPath("tailwindcss"); err == nil {
		return "tailwindcss", nil
	}
	// Walk up from inputPath looking for node_modules/.bin/tailwindcss.
	dir := filepath.Dir(inputPath)
	for range 10 {
		candidate := filepath.Join(dir, "node_modules", ".bin", "tailwindcss")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "npx", []string{"tailwindcss"}
}
