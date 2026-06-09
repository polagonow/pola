package cli

import (
	"fmt"

	"github.com/polagonow/pola/internal/generators"
	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy [generator] [args...]",
	Short: "Remove files created by a generator",
	Long: `Reverse a generator by removing the files it would have created.
This is the inverse of 'pola generate'.

Example:
  pola destroy action Blog          # removes actions/blog_action.go + test
  pola destroy scaffold Product     # removes all scaffold files
  pola destroy route Posts          # removes routes/posts/route.go + test`,
	Args:    cobra.MinimumNArgs(1),
	RunE:    runDestroy,
	Aliases: []string{"d", "del"},
}

func init() {
	destroyCmd.Flags().Bool("force", false, "skip confirmation prompt")
	destroyCmd.Flags().Bool("dry-run", false, "print files that would be removed without deleting")
	destroyCmd.PersistentFlags().Bool("skip-tests", false, "exclude test files from removal")
	destroyCmd.PersistentFlags().Bool("skip-model", false, "skip model removal (scaffold)")
	destroyCmd.PersistentFlags().Bool("skip-repository", false, "skip repository removal (scaffold)")
	destroyCmd.PersistentFlags().Bool("skip-service", false, "skip service removal (scaffold)")
	destroyCmd.PersistentFlags().Bool("skip-action", false, "skip action removal (scaffold)")
	destroyCmd.PersistentFlags().Bool("skip-route", false, "skip route removal (scaffold)")
	destroyCmd.PersistentFlags().Bool("skip-zod", false, "skip zod removal (scaffold)")
	destroyCmd.PersistentFlags().Bool("skip-views", false, "skip page removal (scaffold)")
	destroyCmd.PersistentFlags().Bool("skip-migration", false, "propagated to model destroyer")
}

func runDestroy(cmd *cobra.Command, args []string) error {
	genName := args[0]
	genArgs := args[1:]

	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	fmt.Printf("Destroying %s...\n", genName)

	if err := generators.Destroy(genName, cmd, genArgs, projectDir, dryRun, force); err != nil {
		return err
	}

	return nil
}
