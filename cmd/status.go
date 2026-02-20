package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of parent and child PRs",
	Long: `Display the current branch and a summary of changes.

Examples:
  prki status`,
	RunE: func(cmd *cobra.Command, args []string) error {
		branch, err := getCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}

		fmt.Printf("Current branch: %s\n\n", branch)

		// TODO: fetch child PR status via GitHub API / gh CLI
		fmt.Println("Parent PR: (current branch)")
		fmt.Println("  ├─ Requires GitHub CLI (gh) to list child PRs")
		fmt.Println("  └─ Full status support coming in a future release")

		fmt.Println("\nCurrent changes:")
		files, err := getChangedFiles("")
		if err != nil {
			fmt.Println("  Could not retrieve changed files")
			return nil
		}
		if len(files) == 0 {
			fmt.Println("  No changes")
		} else {
			totalLines := 0
			for _, f := range files {
				totalLines += f.TotalLines()
			}
			fmt.Printf("  %d files, %d lines\n", len(files), totalLines)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
