package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var mergePRs []int

var mergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Merge approved child PRs into the parent branch",
	Long: `Merge approved child PRs into the parent branch.

Examples:
  prki merge
  prki merge --pr 101
  prki merge --pr 101 --pr 103`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(mergePRs) > 0 {
			fmt.Printf("Merging specified child PR(s): %v\n", mergePRs)
		} else {
			fmt.Println("Merging all approved child PRs")
		}

		// TODO: fetch approved child PRs via GitHub API and merge
		fmt.Println("\nNot yet implemented. For now, run manually:")
		fmt.Println("  gh pr merge <child-pr-number> --merge --delete-branch")
		return nil
	},
}

func init() {
	mergeCmd.Flags().IntSliceVar(&mergePRs, "pr", nil, "Child PR number(s) to merge (repeatable)")

	rootCmd.AddCommand(mergeCmd)
}
