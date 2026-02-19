package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

type FileChange struct {
	Path         string
	LinesAdded   int
	LinesDeleted int
	Complexity   int
}

func (f *FileChange) TotalLines() int {
	return f.LinesAdded + f.LinesDeleted
}

type FileGroup struct {
	Name  string
	Files []FileChange
	Order int
}

func (g *FileGroup) TotalLines() int {
	total := 0
	for _, f := range g.Files {
		total += f.TotalLines()
	}
	return total
}

func (g *FileGroup) RiskLevel() string {
	complexity := 0
	for _, f := range g.Files {
		complexity += f.Complexity
	}
	switch {
	case complexity < 50:
		return "低" // low
	case complexity < 100:
		return "中" // medium
	default:
		return "高" // high
	}
}

var (
	analyzeBranch    string
	analyzePR        int
	analyzeThreshold int
	analyzeStrategy  string
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze changes and propose a split plan",
	Long: `Analyze changes in the current branch and propose child PRs.

Examples:
  prki analyze
  prki analyze --branch feature/payment
  prki analyze --threshold 300
  prki analyze --strategy directory`,
	RunE: func(cmd *cobra.Command, args []string) error {
		files, err := getChangedFiles(analyzeBranch)
		if err != nil {
			return fmt.Errorf("failed to get changed files: %w", err)
		}

		if len(files) == 0 {
			fmt.Println("No changed files found.")
			return nil
		}

		calculateComplexity(files)
		groups := groupFiles(files, analyzeStrategy)

		totalLines := 0
		for _, f := range files {
			totalLines += f.TotalLines()
		}

		fmt.Print("\n🌳 Analyzing PR tree...\n\n")
		fmt.Printf("Current changes: %d files, %d lines\n\n", len(files), totalLines)

		if totalLines < analyzeThreshold {
			fmt.Printf("✓ Change size looks fine (%d lines < threshold %d)\n", totalLines, analyzeThreshold)
			return nil
		}

		riskLabel := map[string]string{"低": "low", "中": "medium", "高": "high"}
		fmt.Println("Split proposal:")
		for _, g := range groups {
			risk := g.RiskLevel()
			riskEmoji := map[string]string{"低": "✓", "中": "⚠️", "高": "🔴"}[risk]
			fmt.Printf("  ├─ %s %s\n", g.Name, riskEmoji)
			fmt.Printf("  │   - %d files, %d lines\n", len(g.Files), g.TotalLines())
			fmt.Printf("  │   - complexity: %s\n", riskLabel[risk])
			for i, f := range g.Files {
				if i >= 3 {
					fmt.Printf("  │     ... and %d more\n", len(g.Files)-3)
					break
				}
				fmt.Printf("  │     • %s\n", f.Path)
			}
			fmt.Println("  │")
		}

		fmt.Printf("\nRecommendation: split into %d child PR(s)\n", len(groups))
		return nil
	},
}

func getChangedFiles(branch string) ([]FileChange, error) {
	var diffCmd string
	if branch != "" {
		diffCmd = fmt.Sprintf("git diff main..%s --numstat", branch)
	} else {
		diffCmd = "git diff main..HEAD --numstat"
	}

	parts := strings.Fields(diffCmd)
	out, err := exec.Command(parts[0], parts[1:]...).Output()
	if err != nil {
		// stagingされた変更も試みる
		out, err = exec.Command("git", "diff", "--cached", "--numstat").Output()
		if err != nil {
			return nil, err
		}
	}

	var files []FileChange
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		if parts[0] == "-" || parts[1] == "-" {
			continue // バイナリファイルはスキップ
		}
		added, err1 := strconv.Atoi(parts[0])
		deleted, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			continue
		}
		files = append(files, FileChange{
			Path:         parts[2],
			LinesAdded:   added,
			LinesDeleted: deleted,
		})
	}
	return files, nil
}

func calculateComplexity(files []FileChange) {
	for i := range files {
		base := files[i].TotalLines() / 10
		ext := filepath.Ext(files[i].Path)
		multiplier := 1.0
		switch ext {
		case ".ts", ".tsx", ".js", ".jsx":
			multiplier = 1.2
		case ".py":
			multiplier = 0.9
		case ".go":
			multiplier = 0.8
		}
		files[i].Complexity = int(float64(base) * multiplier)
	}
}

func groupFiles(files []FileChange, strategy string) []FileGroup {
	switch strategy {
	case "directory":
		return groupByDirectory(files)
	case "filetype":
		return groupByFileType(files)
	default:
		return groupBySemantic(files)
	}
}

func groupBySemantic(files []FileChange) []FileGroup {
	buckets := map[string][]FileChange{
		"Infrastructure & Config": {},
		"Core Business Logic":     {},
		"UI & Components":         {},
		"Tests":                   {},
		"Documentation":           {},
	}
	order := map[string]int{
		"Infrastructure & Config": 1,
		"Core Business Logic":     2,
		"UI & Components":         3,
		"Tests":                   4,
		"Documentation":           5,
	}

	for _, f := range files {
		p := strings.ToLower(f.Path)
		switch {
		case isTestFile(p):
			buckets["Tests"] = append(buckets["Tests"], f)
		case isConfigFile(p):
			buckets["Infrastructure & Config"] = append(buckets["Infrastructure & Config"], f)
		case isDocFile(p):
			buckets["Documentation"] = append(buckets["Documentation"], f)
		case isUIFile(p):
			buckets["UI & Components"] = append(buckets["UI & Components"], f)
		default:
			buckets["Core Business Logic"] = append(buckets["Core Business Logic"], f)
		}
	}

	var groups []FileGroup
	names := []string{"Infrastructure & Config", "Core Business Logic", "UI & Components", "Tests", "Documentation"}
	for _, name := range names {
		if len(buckets[name]) > 0 {
			groups = append(groups, FileGroup{Name: name, Files: buckets[name], Order: order[name]})
		}
	}
	return groups
}

func groupByDirectory(files []FileChange) []FileGroup {
	buckets := map[string][]FileChange{}
	for _, f := range files {
		dir := filepath.Dir(f.Path)
		buckets[dir] = append(buckets[dir], f)
	}
	var groups []FileGroup
	i := 1
	for dir, fs := range buckets {
		groups = append(groups, FileGroup{Name: dir, Files: fs, Order: i})
		i++
	}
	return groups
}

func groupByFileType(files []FileChange) []FileGroup {
	buckets := map[string][]FileChange{}
	for _, f := range files {
		ext := filepath.Ext(f.Path)
		if ext == "" {
			ext = "(no ext)"
		}
		buckets[ext] = append(buckets[ext], f)
	}
	var groups []FileGroup
	i := 1
	for ext, fs := range buckets {
		groups = append(groups, FileGroup{Name: ext, Files: fs, Order: i})
		i++
	}
	return groups
}

var testPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\.test\.`),
	regexp.MustCompile(`\.spec\.`),
	regexp.MustCompile(`/tests?/`),
	regexp.MustCompile(`/__tests__/`),
	regexp.MustCompile(`_test\.go$`),
}

var configPatterns = []*regexp.Regexp{
	regexp.MustCompile(`package\.json$`),
	regexp.MustCompile(`tsconfig\.json$`),
	regexp.MustCompile(`\.config\.(js|ts)$`),
	regexp.MustCompile(`\.(yml|yaml)$`),
	regexp.MustCompile(`/\.github/`),
	regexp.MustCompile(`dockerfile`),
	regexp.MustCompile(`go\.mod$`),
	regexp.MustCompile(`go\.sum$`),
}

var docPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\.md$`),
	regexp.MustCompile(`/docs?/`),
	regexp.MustCompile(`readme`),
}

var uiPatterns = []*regexp.Regexp{
	regexp.MustCompile(`/components?/`),
	regexp.MustCompile(`/pages?/`),
	regexp.MustCompile(`/views?/`),
	regexp.MustCompile(`\.(tsx|jsx|vue)$`),
}

func matchAny(path string, patterns []*regexp.Regexp) bool {
	for _, p := range patterns {
		if p.MatchString(path) {
			return true
		}
	}
	return false
}

func isTestFile(p string) bool   { return matchAny(p, testPatterns) }
func isConfigFile(p string) bool { return matchAny(p, configPatterns) }
func isDocFile(p string) bool    { return matchAny(p, docPatterns) }
func isUIFile(p string) bool     { return matchAny(p, uiPatterns) }

func init() {
	analyzeCmd.Flags().StringVar(&analyzeBranch, "branch", "", "Branch to analyze (default: current branch vs main)")
	analyzeCmd.Flags().IntVar(&analyzePR, "pr", 0, "GitHub PR number to analyze")
	analyzeCmd.Flags().IntVar(&analyzeThreshold, "threshold", 500, "Line count threshold to suggest splitting")
	analyzeCmd.Flags().StringVar(&analyzeStrategy, "strategy", "semantic", "Grouping strategy (semantic|directory|filetype)")

	rootCmd.AddCommand(analyzeCmd)
}
