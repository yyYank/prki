package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var (
	splitAuto      bool
	splitDraft     bool
	splitReviewers string
	splitStrategy  string
)

type splitResult struct {
	group  FileGroup
	branch string
	prURL  string
}

var splitCmd = &cobra.Command{
	Use:   "split",
	Short: "Split changes into child branches and create child PRs",
	Long: `Analyze changes in the current branch, create child branches for each group,
and open draft PRs against the current branch for focused review.

Each child branch is created from main with only the files in its group,
so reviewers see a clean, focused diff.

Examples:
  prki split
  prki split --auto
  prki split --draft=false
  prki split --reviewers alice,bob
  prki split --strategy directory`,
	RunE: runSplit,
}

func runSplit(cmd *cobra.Command, args []string) error {
	parentBranch, err := getCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	files, err := getChangedFiles("")
	if err != nil {
		return fmt.Errorf("failed to get changed files: %w", err)
	}
	if len(files) == 0 {
		fmt.Println("No changed files found.")
		return nil
	}

	calculateComplexity(files)
	groups := groupFiles(files, splitStrategy)

	totalLines := 0
	for _, f := range files {
		totalLines += f.TotalLines()
	}

	fmt.Println("\n🌳 Analyzing PR tree...\n")
	fmt.Printf("Current changes: %d files, %d lines\n\n", len(files), totalLines)
	fmt.Println("Split proposal:")
	for i, g := range groups {
		connector := "├─"
		if i == len(groups)-1 {
			connector = "└─"
		}
		fmt.Printf("  %s %s (%d files, %d lines)\n", connector, g.Name, len(g.Files), g.TotalLines())
	}

	if !splitAuto {
		fmt.Print("\nProceed? [Y/n] ")
		reader := bufio.NewReader(os.Stdin)
		ans, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(ans)) == "n" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	fmt.Println()
	var results []splitResult
	for _, g := range groups {
		r, err := createChildBranchAndPR(g, parentBranch)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", g.Name, err)
			// always return to parent before continuing
			_ = gitSilent("checkout", parentBranch)
			continue
		}
		results = append(results, *r)
	}

	// ensure we are back on the parent branch
	_ = gitSilent("checkout", parentBranch)

	fmt.Printf("\n%d child PR(s) created:\n", len(results))
	for _, r := range results {
		if r.prURL != "" {
			fmt.Printf("  ✓ [%s] branch: %s\n    %s\n", r.group.Name, r.branch, r.prURL)
		} else {
			fmt.Printf("  ✓ [%s] branch: %s  (push and create PR manually)\n", r.group.Name, r.branch)
		}
	}
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Request reviews on each child PR")
	fmt.Println("  2. After approval: prki merge")
	return nil
}

func createChildBranchAndPR(g FileGroup, parentBranch string) (*splitResult, error) {
	branch := toBranchName(g.Name)
	filePaths := make([]string, len(g.Files))
	for i, f := range g.Files {
		filePaths[i] = f.Path
	}

	// Create child branch from main
	fmt.Printf("  Creating branch %s...\n", branch)
	if err := gitSilent("checkout", "-b", branch, "main"); err != nil {
		return nil, fmt.Errorf("could not create branch %s (already exists?): %w", branch, err)
	}

	// Checkout only this group's files from the parent branch
	checkoutArgs := append([]string{"checkout", parentBranch, "--"}, filePaths...)
	if err := gitSilent(checkoutArgs...); err != nil {
		return nil, fmt.Errorf("failed to checkout files from %s: %w", parentBranch, err)
	}

	// Commit message
	commitMsg := defaultCommitMsg(g.Name)
	if !splitAuto {
		fmt.Printf("  Commit message for %s\n  (Enter to use default: %q): ", branch, commitMsg)
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		if msg := strings.TrimSpace(input); msg != "" {
			commitMsg = msg
		}
	}

	if err := gitSilent("commit", "-m", commitMsg); err != nil {
		return nil, fmt.Errorf("commit failed: %w", err)
	}

	// Push
	fmt.Printf("  Pushing %s...\n", branch)
	if err := gitSilent("push", "-u", "origin", branch); err != nil {
		fmt.Printf("  ⚠  Push failed — branch %s created locally only.\n", branch)
		return &splitResult{group: g, branch: branch}, nil
	}

	// Create PR via gh CLI
	prURL, err := ghCreatePR(branch, parentBranch, g)
	if err != nil {
		fmt.Printf("  ⚠  PR creation failed: %v\n    Create it manually: gh pr create --base %s --head %s\n", err, parentBranch, branch)
		return &splitResult{group: g, branch: branch}, nil
	}

	return &splitResult{group: g, branch: branch, prURL: prURL}, nil
}

func ghCreatePR(branch, base string, g FileGroup) (string, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", fmt.Errorf("gh CLI not found (https://cli.github.com)")
	}

	ghArgs := []string{
		"pr", "create",
		"--base", base,
		"--head", branch,
		"--title", fmt.Sprintf("[Review] %s", g.Name),
		"--body", buildPRBody(base, g),
	}
	if splitDraft {
		ghArgs = append(ghArgs, "--draft")
	}
	if splitReviewers != "" {
		for _, r := range strings.Split(splitReviewers, ",") {
			if r = strings.TrimSpace(r); r != "" {
				ghArgs = append(ghArgs, "--reviewer", r)
			}
		}
	}

	out, err := exec.Command("gh", ghArgs...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func buildPRBody(parentBranch string, g FileGroup) string {
	var sb strings.Builder
	sb.WriteString("## Review Purpose\n\n")
	sb.WriteString("This is a child PR for review purposes only.\n\n")
	sb.WriteString(fmt.Sprintf("**Parent Branch:** `%s`  \n", parentBranch))
	sb.WriteString(fmt.Sprintf("**Group:** %s\n\n", g.Name))
	sb.WriteString("## Files in This PR\n\n")
	for _, f := range g.Files {
		sb.WriteString(fmt.Sprintf("- `%s` (+%d/-%d lines)\n", f.Path, f.LinesAdded, f.LinesDeleted))
	}
	sb.WriteString("\n## Context\n\n")
	sb.WriteString("This PR is part of a larger feature split for easier review.  \n")
	sb.WriteString("Once approved, it will be merged back into the parent branch.\n\n")
	sb.WriteString("Please review this subset independently.\n")
	return sb.String()
}

func defaultCommitMsg(groupName string) string {
	return fmt.Sprintf("[Review] %s", groupName)
}

// gitSilent runs a git command, suppressing stdout but showing stderr.
func gitSilent(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getCurrentBranch() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func toBranchName(groupName string) string {
	lower := strings.ToLower(groupName)
	words := strings.Fields(lower)
	last := words[len(words)-1]
	last = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '-'
	}, last)
	return "review/" + last
}

func init() {
	splitCmd.Flags().BoolVar(&splitAuto, "auto", false, "Skip all confirmation prompts")
	splitCmd.Flags().BoolVar(&splitDraft, "draft", true, "Create child PRs as drafts")
	splitCmd.Flags().StringVar(&splitReviewers, "reviewers", "", "Comma-separated list of reviewers")
	splitCmd.Flags().StringVar(&splitStrategy, "strategy", "semantic", "Grouping strategy (semantic|directory|filetype)")

	rootCmd.AddCommand(splitCmd)
}
