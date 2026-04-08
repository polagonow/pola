// Package actionbridge implements the action bridge overlay autoload.
// It generates Go bridge code and TypeScript declarations from the actions/ directory.
package actionbridge

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/polagonow/pola/internal/actionbridge"
	"github.com/polagonow/pola/internal/autoload"
)

type autoloadImpl struct{}

func init() {
	autoload.Register(&autoloadImpl{})
}

func (a *autoloadImpl) Name() string { return "actionbridge" }
func (a *autoloadImpl) Priority() int { return 100 }

func (a *autoloadImpl) Contribute(ctx *autoload.Context) error {
	actionsDir := ctx.Opts.ActionsDir
	if actionsDir == "" {
		actionsDir = filepath.Join(ctx.ProjectDir, "actions")
	}
	if !filepath.IsAbs(actionsDir) {
		actionsDir = filepath.Join(ctx.ProjectDir, actionsDir)
	}

	info, err := os.Stat(actionsDir)
	if err != nil || !info.IsDir() {
		if ctx.Verbose {
			fmt.Println("No actions/ directory found, skipping actionbridge.")
		}
		return nil
	}

	ctx.Discovery.HasActions = true

	tsOut := ctx.Opts.TSOut
	if tsOut == "" {
		tsOut = filepath.Join(ctx.ProjectDir, "node_modules", "@pola", "actions", "src", "generated.ts")
	}
	if !filepath.IsAbs(tsOut) {
		tsOut = filepath.Join(ctx.ProjectDir, tsOut)
	}

	fmt.Println("Generating action bridges...")
	bridgeResult, err := actionbridge.Run(actionsDir, tsOut, ctx.TmpDir, ctx.Opts.PolaPackage)
	if err != nil {
		return fmt.Errorf("actionbridge: %w", err)
	}

	if bridgeResult != nil && bridgeResult.BridgePath != "" {
		ctx.Replace[bridgeResult.VirtualPath] = bridgeResult.BridgePath
		ctx.Discovery.TSOutPath = bridgeResult.TSOutPath
	}

	if ctx.Verbose && bridgeResult != nil && bridgeResult.TSOutPath != "" {
		fmt.Printf("Generated types: %s\n", bridgeResult.TSOutPath)
	}

	// Resolve actions import path for pluginimports.
	if ctx.ModPath != "" {
		ctx.Discovery.ActionsImport = ctx.ModPath + "/actions"
	}

	return nil
}
