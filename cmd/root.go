package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "prki",
	Short: "prki (PR tree) - Split large PRs into manageable child PRs",
	Long: `prki (PR tree) splits large pull requests into smaller, reviewable child PRs.

In the AI coding era, it's easy to end up with a huge diff before you know it.
prki rescues already-large PRs by splitting them into meaningful groups.

Commands:
  prki analyze    Analyze changes and propose a split plan
  prki split      Execute the split and create child branches/PRs
  prki status     Show status of parent and child PRs
  prki merge      Merge approved child PRs into the parent branch`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
