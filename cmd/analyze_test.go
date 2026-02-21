package cmd

import (
	"regexp"
	"testing"
)

func TestFileChange_TotalLines(t *testing.T) {
	tests := []struct {
		name    string
		f       FileChange
		wantTotal int
	}{
		{"added only", FileChange{LinesAdded: 10, LinesDeleted: 0}, 10},
		{"deleted only", FileChange{LinesAdded: 0, LinesDeleted: 5}, 5},
		{"added and deleted", FileChange{LinesAdded: 30, LinesDeleted: 20}, 50},
		{"zero", FileChange{LinesAdded: 0, LinesDeleted: 0}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f.TotalLines(); got != tt.wantTotal {
				t.Errorf("TotalLines() = %d, want %d", got, tt.wantTotal)
			}
		})
	}
}

func TestFileGroup_TotalLines(t *testing.T) {
	tests := []struct {
		name  string
		group FileGroup
		want  int
	}{
		{
			"empty group",
			FileGroup{Files: []FileChange{}},
			0,
		},
		{
			"single file",
			FileGroup{Files: []FileChange{{LinesAdded: 10, LinesDeleted: 5}}},
			15,
		},
		{
			"multiple files",
			FileGroup{Files: []FileChange{
				{LinesAdded: 10, LinesDeleted: 5},
				{LinesAdded: 20, LinesDeleted: 0},
				{LinesAdded: 0, LinesDeleted: 3},
			}},
			38,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.group.TotalLines(); got != tt.want {
				t.Errorf("TotalLines() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFileGroup_RiskLevel(t *testing.T) {
	// RiskLevel is based on sum of Complexity values across files
	tests := []struct {
		name  string
		files []FileChange
		want  string
	}{
		{
			"low complexity (sum < 50)",
			[]FileChange{{Complexity: 10}, {Complexity: 20}},
			"低",
		},
		{
			"medium complexity (50 <= sum < 100)",
			[]FileChange{{Complexity: 50}, {Complexity: 40}},
			"中",
		},
		{
			"high complexity (sum >= 100)",
			[]FileChange{{Complexity: 60}, {Complexity: 60}},
			"高",
		},
		{
			"exactly 50 is medium",
			[]FileChange{{Complexity: 50}},
			"中",
		},
		{
			"exactly 100 is high",
			[]FileChange{{Complexity: 100}},
			"高",
		},
		{
			"zero is low",
			[]FileChange{},
			"低",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := FileGroup{Files: tt.files}
			if got := g.RiskLevel(); got != tt.want {
				t.Errorf("RiskLevel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCalculateComplexity(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		added      int
		deleted    int
		wantComplexity int
	}{
		// base = TotalLines / 10
		// .go: multiplier 0.8
		{"go file", "main.go", 50, 50, int(float64((50+50)/10) * 0.8)},
		// .ts: multiplier 1.2
		{"ts file", "src/app.ts", 50, 50, int(float64((50+50)/10) * 1.2)},
		// .tsx: multiplier 1.2
		{"tsx file", "src/App.tsx", 50, 50, int(float64((50+50)/10) * 1.2)},
		// .js: multiplier 1.2
		{"js file", "src/index.js", 50, 50, int(float64((50+50)/10) * 1.2)},
		// .jsx: multiplier 1.2
		{"jsx file", "src/Index.jsx", 50, 50, int(float64((50+50)/10) * 1.2)},
		// .py: multiplier 0.9
		{"py file", "script.py", 50, 50, int(float64((50+50)/10) * 0.9)},
		// unknown ext: multiplier 1.0
		{"md file", "README.md", 50, 50, int(float64((50+50)/10) * 1.0)},
		// zero lines: complexity = 0
		{"zero lines", "main.go", 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := []FileChange{{Path: tt.path, LinesAdded: tt.added, LinesDeleted: tt.deleted}}
			calculateComplexity(files)
			if files[0].Complexity != tt.wantComplexity {
				t.Errorf("Complexity = %d, want %d", files[0].Complexity, tt.wantComplexity)
			}
		})
	}
}

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"src/foo.test.ts", true},
		{"src/foo.spec.js", true},
		// /tests?/ パターンは先頭スラッシュが必要なのでサブディレクトリが必要
		{"src/tests/helper.go", true},
		{"src/test/helper.go", true},
		{"src/__tests__/component.tsx", true},
		{"cmd/analyze_test.go", true},
		// 先頭スラッシュなしは /tests?/ や /__tests__/ にマッチしない
		{"tests/helper.go", false},
		{"test/helper.go", false},
		{"__tests__/component.tsx", false},
		{"src/main.go", false},
		{"src/utils.ts", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isTestFile(tt.path); got != tt.want {
				t.Errorf("isTestFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsConfigFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"package.json", true},
		{"tsconfig.json", true},
		{"vite.config.js", true},
		{"vite.config.ts", true},
		{".github/workflows/ci.yml", true},
		{".github/workflows/ci.yaml", true},
		// パターンは小文字 "dockerfile" のみにマッチ（大文字はマッチしない）
		{"dockerfile", true},
		{"Dockerfile", false},
		{"go.mod", true},
		{"go.sum", true},
		{"src/main.go", false},
		{"README.md", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isConfigFile(tt.path); got != tt.want {
				t.Errorf("isConfigFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsDocFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"README.md", true},
		{"CONCEPT.md", true},
		{"src/docs/guide.md", true},
		// docs/guide.md は .md$ パターンにマッチするので true
		{"docs/guide.md", true},
		// /docs?/ パターンは先頭スラッシュが必要。.md 以外の拡張子はマッチしない
		{"docs/guide.txt", false},
		{"doc/spec.txt", false},
		{"readme.txt", true},
		{"src/main.go", false},
		{"src/app.ts", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isDocFile(tt.path); got != tt.want {
				t.Errorf("isDocFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsUIFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"src/components/Button.tsx", true},
		{"src/component/Input.jsx", true},
		{"src/pages/Home.tsx", true},
		{"src/page/About.vue", true},
		{"src/views/Dashboard.vue", true},
		{"src/view/Profile.tsx", true},
		{"App.tsx", true},
		{"App.jsx", true},
		{"App.vue", true},
		{"src/main.go", false},
		{"src/utils.ts", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isUIFile(tt.path); got != tt.want {
				t.Errorf("isUIFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestGroupBySemantic(t *testing.T) {
	files := []FileChange{
		{Path: "src/service.go"},
		{Path: "cmd/analyze_test.go"},
		{Path: "go.mod"},
		{Path: "README.md"},
		{Path: "src/components/Button.tsx"},
	}

	groups := groupBySemantic(files)

	// Find groups by name
	byName := map[string]FileGroup{}
	for _, g := range groups {
		byName[g.Name] = g
	}

	if _, ok := byName["Core Business Logic"]; !ok {
		t.Error("expected group 'Core Business Logic'")
	}
	if _, ok := byName["Tests"]; !ok {
		t.Error("expected group 'Tests'")
	}
	if _, ok := byName["Infrastructure & Config"]; !ok {
		t.Error("expected group 'Infrastructure & Config'")
	}
	if _, ok := byName["Documentation"]; !ok {
		t.Error("expected group 'Documentation'")
	}
	if _, ok := byName["UI & Components"]; !ok {
		t.Error("expected group 'UI & Components'")
	}

	if len(byName["Core Business Logic"].Files) != 1 {
		t.Errorf("Core Business Logic: got %d files, want 1", len(byName["Core Business Logic"].Files))
	}
	if len(byName["Tests"].Files) != 1 {
		t.Errorf("Tests: got %d files, want 1", len(byName["Tests"].Files))
	}
	if len(byName["Infrastructure & Config"].Files) != 1 {
		t.Errorf("Infrastructure & Config: got %d files, want 1", len(byName["Infrastructure & Config"].Files))
	}
	if len(byName["Documentation"].Files) != 1 {
		t.Errorf("Documentation: got %d files, want 1", len(byName["Documentation"].Files))
	}
	if len(byName["UI & Components"].Files) != 1 {
		t.Errorf("UI & Components: got %d files, want 1", len(byName["UI & Components"].Files))
	}
}

func TestGroupBySemantic_EmptyGroups_NotIncluded(t *testing.T) {
	// When only one category has files, only that group should appear
	files := []FileChange{
		{Path: "README.md"},
	}
	groups := groupBySemantic(files)
	if len(groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Name != "Documentation" {
		t.Errorf("expected group 'Documentation', got %q", groups[0].Name)
	}
}

func TestGroupByDirectory(t *testing.T) {
	files := []FileChange{
		{Path: "cmd/analyze.go"},
		{Path: "cmd/split.go"},
		{Path: "src/main.ts"},
	}
	groups := groupByDirectory(files)

	byName := map[string]FileGroup{}
	for _, g := range groups {
		byName[g.Name] = g
	}

	if len(byName["cmd"].Files) != 2 {
		t.Errorf("cmd: got %d files, want 2", len(byName["cmd"].Files))
	}
	if len(byName["src"].Files) != 1 {
		t.Errorf("src: got %d files, want 1", len(byName["src"].Files))
	}
}

func TestGroupByFileType(t *testing.T) {
	files := []FileChange{
		{Path: "cmd/analyze.go"},
		{Path: "cmd/split.go"},
		{Path: "src/main.ts"},
		{Path: "Makefile"},
	}
	groups := groupByFileType(files)

	byName := map[string]FileGroup{}
	for _, g := range groups {
		byName[g.Name] = g
	}

	if len(byName[".go"].Files) != 2 {
		t.Errorf(".go: got %d files, want 2", len(byName[".go"].Files))
	}
	if len(byName[".ts"].Files) != 1 {
		t.Errorf(".ts: got %d files, want 1", len(byName[".ts"].Files))
	}
	if len(byName["(no ext)"].Files) != 1 {
		t.Errorf("(no ext): got %d files, want 1", len(byName["(no ext)"].Files))
	}
}

func TestGroupFiles_Strategy(t *testing.T) {
	files := []FileChange{
		{Path: "cmd/analyze.go"},
		{Path: "src/main.ts"},
	}

	for _, strategy := range []string{"directory", "filetype", "semantic", ""} {
		groups := groupFiles(files, strategy)
		if len(groups) == 0 {
			t.Errorf("strategy=%q: expected non-empty groups", strategy)
		}
	}
}

func TestGroupBySemantic_Order(t *testing.T) {
	files := []FileChange{
		{Path: "README.md"},           // Documentation (order 5)
		{Path: "cmd/foo_test.go"},     // Tests (order 4)
		{Path: "src/App.tsx"},         // UI & Components (order 3)
		{Path: "src/service.go"},      // Core Business Logic (order 2)
		{Path: "go.mod"},              // Infrastructure & Config (order 1)
	}
	groups := groupBySemantic(files)

	expectedOrder := []string{
		"Infrastructure & Config",
		"Core Business Logic",
		"UI & Components",
		"Tests",
		"Documentation",
	}
	if len(groups) != len(expectedOrder) {
		t.Fatalf("expected %d groups, got %d", len(expectedOrder), len(groups))
	}
	for i, want := range expectedOrder {
		if groups[i].Name != want {
			t.Errorf("groups[%d].Name = %q, want %q", i, groups[i].Name, want)
		}
	}
}

func TestGroupBySemantic_EmptyInput(t *testing.T) {
	groups := groupBySemantic([]FileChange{})
	if len(groups) != 0 {
		t.Errorf("expected 0 groups for empty input, got %d", len(groups))
	}
}

func TestGroupByDirectory_RootFile(t *testing.T) {
	files := []FileChange{
		{Path: "main.go"},       // filepath.Dir → "."
		{Path: "cmd/split.go"},
	}
	groups := groupByDirectory(files)

	byName := map[string]FileGroup{}
	for _, g := range groups {
		byName[g.Name] = g
	}

	if len(byName["."].Files) != 1 {
		t.Errorf("root dir \".\": got %d files, want 1", len(byName["."].Files))
	}
	if len(byName["cmd"].Files) != 1 {
		t.Errorf("cmd: got %d files, want 1", len(byName["cmd"].Files))
	}
}

func TestMatchAny(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		patterns []*regexp.Regexp
		want     bool
	}{
		{
			"empty pattern list never matches",
			"src/main.go",
			[]*regexp.Regexp{},
			false,
		},
		{
			"single matching pattern",
			"main.go",
			[]*regexp.Regexp{regexp.MustCompile(`\.go$`)},
			true,
		},
		{
			"single non-matching pattern",
			"main.go",
			[]*regexp.Regexp{regexp.MustCompile(`\.ts$`)},
			false,
		},
		{
			"matches second pattern",
			"src/App.tsx",
			[]*regexp.Regexp{regexp.MustCompile(`\.go$`), regexp.MustCompile(`\.tsx$`)},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchAny(tt.path, tt.patterns); got != tt.want {
				t.Errorf("matchAny(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestCalculateComplexity_MultipleFiles(t *testing.T) {
	files := []FileChange{
		{Path: "main.go", LinesAdded: 100, LinesDeleted: 0},  // base=10, mult=0.8 → 8
		{Path: "app.ts", LinesAdded: 100, LinesDeleted: 0},   // base=10, mult=1.2 → 12
	}
	calculateComplexity(files)

	if files[0].Complexity != 8 {
		t.Errorf("files[0].Complexity = %d, want 8", files[0].Complexity)
	}
	if files[1].Complexity != 12 {
		t.Errorf("files[1].Complexity = %d, want 12", files[1].Complexity)
	}
}
