package mcp

import (
	"fmt"
	"path/filepath"

	"github.com/polagonow/pola/internal/generators"
	"github.com/polagonow/pola/internal/project"
	"github.com/polagonow/pola/polafile"
	"github.com/spf13/cobra"
)

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Add an mcp { … } block to Polafile so the MCP plugin is wired in",
		Args:  cobra.NoArgs,
		RunE:  runInit,
		Example: `  pola generate mcp init`,
	}
}

func runInit(_ *cobra.Command, _ []string) error {
	projectDir, err := project.FindRoot()
	if err != nil {
		return err
	}

	pf, err := polafile.Load(projectDir)
	if err != nil {
		return fmt.Errorf("load Polafile: %w", err)
	}
	if pf == nil {
		fmt.Println("No Polafile found; skipping mcp { } block. Re-run after `pola init`.")
		return nil
	}
	if pf.MCP == nil {
		pf.MCP = &polafile.MCP{
			Enabled:   true,
			Transport: "http",
			Mount:     "/mcp",
		}
		if err := polafile.Save(projectDir, pf); err != nil {
			return fmt.Errorf("save Polafile: %w", err)
		}
		fmt.Printf("Updated %s with mcp { … } block\n", filepath.Join(projectDir, polafile.Filename))
	} else {
		fmt.Println("Polafile already has an mcp block; left unchanged.")
	}

	return generators.RunAfterHooks(&Generator{}, projectDir)
}
