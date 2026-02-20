package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// ChildPR represents a child pull request with its review status.
type ChildPR struct {
	Number         int    `json:"number"`
	Title          string `json:"title"`
	ReviewDecision string `json:"reviewDecision"`
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of parent and child PRs",
	Long: `Display the current branch, child PR statuses, and a summary of changes.

Examples:
  prki status`,
	RunE: func(cmd *cobra.Command, args []string) error {
		branch, err := getCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}

		fmt.Printf("Current branch: %s\n\n", branch)

		prs, err := fetchChildPRs(branch)
		if err != nil {
			fmt.Printf("Parent PR: %s\n", branch)
			fmt.Printf("  (Could not fetch child PR status: %v)\n", err)
		} else if len(prs) == 0 {
			fmt.Printf("Parent PR: %s\n", branch)
			fmt.Println("  (No child PRs found)")
		} else {
			fmt.Printf("Parent PR: %s\n", branch)
			for i, pr := range prs {
				connector := "├─"
				if i == len(prs)-1 {
					connector = "└─"
				}
				fmt.Printf("  %s Child PR #%d: %s [%s]\n", connector, pr.Number, pr.Title, reviewLabel(pr.ReviewDecision))
			}

			toFix, toMerge := nextActions(prs)
			if len(toFix) > 0 || len(toMerge) > 0 {
				fmt.Println("\nNext actions:")
				for _, pr := range toFix {
					fmt.Printf("  • Fix changes requested: %s\n", pr.Title)
				}
				for _, pr := range toMerge {
					fmt.Printf("  • Ready to merge: %s\n", pr.Title)
				}
			}
		}

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

// fetchChildPRs uses the gh CLI to list PRs whose base is the given branch.
func fetchChildPRs(branch string) ([]ChildPR, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, fmt.Errorf("gh CLI not found (https://cli.github.com)")
	}
	out, err := exec.Command("gh", "pr", "list",
		"--base", branch,
		"--json", "number,title,reviewDecision",
	).Output()
	if err != nil {
		return nil, err
	}
	var prs []ChildPR
	if err := json.Unmarshal(out, &prs); err != nil {
		return nil, fmt.Errorf("failed to parse gh output: %w", err)
	}
	return prs, nil
}

// reviewLabel converts a GitHub review decision string to a human-readable label.
func reviewLabel(decision string) string {
	switch strings.ToUpper(decision) {
	case "APPROVED":
		return "approved ✓"
	case "CHANGES_REQUESTED":
		return "changes requested"
	default:
		return "pending review"
	}
}

// nextActions separates PRs into those needing fixes and those ready to merge.
func nextActions(prs []ChildPR) (toFix []ChildPR, toMerge []ChildPR) {
	for _, pr := range prs {
		switch strings.ToUpper(pr.ReviewDecision) {
		case "CHANGES_REQUESTED":
			toFix = append(toFix, pr)
		case "APPROVED":
			toMerge = append(toMerge, pr)
		}
	}
	return
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
