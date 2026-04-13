// Package sass provides a CSS plugin that runs the Sass CLI to compile
// SCSS/Sass stylesheets into CSS. It supports both standalone binary and
// npx execution modes.
//
// Import this package to enable Sass processing:
//
//	import _ "github.com/polagonow/pola/css/sass"
package sass

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/polagonow/pola/core"
)

// Sass runs the sass CLI to compile SCSS/Sass stylesheets.
type Sass struct {
	// Bin is the path to the sass CLI binary (default: "sass").
	Bin    string
	logger core.Logger
}

// SetLogger implements core.LogAware.
func (s *Sass) SetLogger(l core.Logger) { s.logger = l }

// New creates a Sass CSS processor.
func New() *Sass { return &Sass{} }

// Ensure Sass satisfies core.CSS.
var _ core.CSS = (*Sass)(nil)

// Name returns the CSS implementation name.
func (s *Sass) Name() string { return "sass" }

// Process runs sass to compile inputPath → outputPath.
// In production mode (POLA_ENV != "development") it minifies the output.
func (s *Sass) Process(ctx context.Context, inputPath, outputPath string) error {
	bin, binArgs := s.resolvedBin(inputPath)
	args := append(binArgs, inputPath, outputPath, "--no-source-map", "--quiet-deps")

	// Add node_modules load path for package resolution (e.g. @carbon/react).
	nmRoot := nodeModulesRoot(inputPath)
	args = append(args, "--load-path", filepath.Join(nmRoot, "node_modules"))

	if os.Getenv("POLA_ENV") != "development" {
		args = append(args, "--style=compressed")
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = nmRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sass: %w", err)
	}
	return nil
}

// Watch runs sass in watch mode (--watch), calling onChange on each rebuild.
// The watcher process is terminated when ctx is cancelled.
func (s *Sass) Watch(ctx context.Context, inputPath, outputPath string, onChange func()) error {
	bin, binArgs := s.resolvedBin(inputPath)
	args := append(binArgs, "--watch", inputPath+":"+outputPath, "--no-source-map", "--quiet-deps")

	nmRoot := nodeModulesRoot(inputPath)
	args = append(args, "--load-path", filepath.Join(nmRoot, "node_modules"))

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = nmRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	go func() {
		if err := cmd.Run(); err != nil && ctx.Err() == nil {
			if s.logger != nil {
				s.logger.Error("sass: watch error", "err", err)
			}
		}
	}()
	return nil
}

// resolvedBin returns the binary and any extra args needed to invoke sass.
// It checks: explicit Bin field → PATH → node_modules/.bin/ near inputPath → npx.
func (s *Sass) resolvedBin(inputPath string) (string, []string) {
	if s.Bin != "" {
		if _, err := exec.LookPath(s.Bin); err == nil {
			return s.Bin, nil
		}
	}
	if _, err := exec.LookPath("sass"); err == nil {
		return "sass", nil
	}
	// Walk up from inputPath looking for node_modules/.bin/sass.
	dir := filepath.Dir(inputPath)
	for {
		candidate := filepath.Join(dir, "node_modules", ".bin", "sass")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "npx", []string{"sass"}
}

// nodeModulesRoot walks up from the CSS/SCSS file's directory to find the
// nearest ancestor containing node_modules. This is used as the Sass CLI
// working directory so that @use/@import paths resolve correctly.
func nodeModulesRoot(inputPath string) string {
	dir := filepath.Dir(inputPath)
	for {
		if info, err := os.Stat(filepath.Join(dir, "node_modules")); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Dir(inputPath)
}
