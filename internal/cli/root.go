// Package cli implements the pola command-line tool.
package cli

import (
	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags.
var version = "dev"

var verbose bool

var rootCmd = &cobra.Command{
	Use:   "pola",
	Short: "Pola framework CLI",
	Long:  "CLI tool for creating, developing, and building Pola framework applications.",
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
	SilenceUsage: true,
	Version:      version,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(generateCmd)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
