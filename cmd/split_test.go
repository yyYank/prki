package cmd

import (
	"strings"
	"testing"
)

func TestToBranchName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Core Business Logic", "review/logic"},
		{"UI & Components", "review/components"},
		{"Infrastructure & Config", "review/config"},
		{"Tests", "review/tests"},
		{"Documentation", "review/documentation"},
		{"Single", "review/single"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := toBranchName(tt.input); got != tt.want {
				t.Errorf("toBranchName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDefaultCommitMsg(t *testing.T) {
	tests := []struct {
		groupName string
		want      string
	}{
		{"Core Business Logic", "[Review] Core Business Logic"},
		{"UI & Components", "[Review] UI & Components"},
		{"Tests", "[Review] Tests"},
	}
	for _, tt := range tests {
		t.Run(tt.groupName, func(t *testing.T) {
			if got := defaultCommitMsg(tt.groupName); got != tt.want {
				t.Errorf("defaultCommitMsg(%q) = %q, want %q", tt.groupName, got, tt.want)
			}
		})
	}
}

func TestBuildPRBody(t *testing.T) {
	g := FileGroup{
		Name: "Core Business Logic",
		Files: []FileChange{
			{Path: "cmd/analyze.go", LinesAdded: 100, LinesDeleted: 20},
			{Path: "cmd/split.go", LinesAdded: 50, LinesDeleted: 0},
		},
	}
	parentBranch := "feature/my-feature"
	body := buildPRBody(parentBranch, g)

	cases := []struct {
		desc    string
		contain string
	}{
		{"contains parent branch", "feature/my-feature"},
		{"contains group name", "Core Business Logic"},
		{"contains first file path", "cmd/analyze.go"},
		{"contains second file path", "cmd/split.go"},
		{"contains added lines for first file", "+100"},
		{"contains deleted lines for first file", "-20"},
		{"contains Review Purpose section", "## Review Purpose"},
		{"contains Files section", "## Files in This PR"},
		{"contains Context section", "## Context"},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			if !strings.Contains(body, c.contain) {
				t.Errorf("buildPRBody() does not contain %q\nBody:\n%s", c.contain, body)
			}
		})
	}
}

func TestBuildPRBody_EmptyFiles(t *testing.T) {
	g := FileGroup{Name: "Documentation", Files: []FileChange{}}
	body := buildPRBody("main", g)
	if !strings.Contains(body, "Documentation") {
		t.Error("buildPRBody() should contain group name even with no files")
	}
}
