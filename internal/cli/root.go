// Package cli implements the pola command-line tool.
package cli

import (
	"fmt"
	"os"

	autoloadall "github.com/polagonow/pola/internal/autoload/all"
	"github.com/polagonow/pola/internal/errs"
	generatorsall "github.com/polagonow/pola/internal/generators/all"
	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags.
var version = "dev"

var verbose bool
var cwd string

var rootCmd = &cobra.Command{
	Use:   "pola",
	Short: "Pola framework CLI",
	Long:  "CLI tool for creating, developing, and building Pola framework applications.",
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       version,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		if cwd == "" {
			return nil
		}
		// For "new", the --cwd directory may not exist yet; create it.
		if cmd.Name() == "new" {
			if err := os.MkdirAll(cwd, 0o755); err != nil {
				return fmt.Errorf("create directory %q: %w", cwd, err)
			}
		}
		if err := os.Chdir(cwd); err != nil {
			return fmt.Errorf("change directory to %q: %w", cwd, err)
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().StringVar(&cwd, "cwd", "", "run as if pola was started in this directory")
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(destroyCmd)
	rootCmd.AddCommand(dbCmd)
}

// Execute runs the root command. Errors are formatted with structured
// indentation (root cause first) and printed to stderr.
func Execute() error {
	generatorsall.Register()
	autoloadall.Register()
	addGeneratorCommands()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, errs.Format(err))
		return err
	}
	return nil
}
