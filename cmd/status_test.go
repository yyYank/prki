package cmd

import (
	"testing"
)

func TestReviewLabel(t *testing.T) {
	tests := []struct {
		decision string
		want     string
	}{
		{"APPROVED", "approved ✓"},
		{"approved", "approved ✓"},
		{"Approved", "approved ✓"},
		{"CHANGES_REQUESTED", "changes requested"},
		{"changes_requested", "changes requested"},
		{"Changes_Requested", "changes requested"},
		{"", "pending review"},
		{"REVIEW_REQUIRED", "pending review"},
		{"DISMISSED", "pending review"},
	}
	for _, tt := range tests {
		t.Run(tt.decision, func(t *testing.T) {
			if got := reviewLabel(tt.decision); got != tt.want {
				t.Errorf("reviewLabel(%q) = %q, want %q", tt.decision, got, tt.want)
			}
		})
	}
}

func TestNextActions(t *testing.T) {
	prs := []ChildPR{
		{Number: 101, Title: "review/config", ReviewDecision: "APPROVED"},
		{Number: 102, Title: "review/core", ReviewDecision: "CHANGES_REQUESTED"},
		{Number: 103, Title: "review/tests", ReviewDecision: ""},
	}
	toFix, toMerge := nextActions(prs)

	if len(toFix) != 1 {
		t.Fatalf("toFix: got %d items, want 1", len(toFix))
	}
	if toFix[0].Number != 102 {
		t.Errorf("toFix[0].Number = %d, want 102", toFix[0].Number)
	}

	if len(toMerge) != 1 {
		t.Fatalf("toMerge: got %d items, want 1", len(toMerge))
	}
	if toMerge[0].Number != 101 {
		t.Errorf("toMerge[0].Number = %d, want 101", toMerge[0].Number)
	}
}

func TestNextActions_AllApproved(t *testing.T) {
	prs := []ChildPR{
		{Number: 101, Title: "review/config", ReviewDecision: "APPROVED"},
		{Number: 102, Title: "review/core", ReviewDecision: "APPROVED"},
	}
	toFix, toMerge := nextActions(prs)

	if len(toFix) != 0 {
		t.Errorf("toFix: expected empty, got %v", toFix)
	}
	if len(toMerge) != 2 {
		t.Errorf("toMerge: got %d items, want 2", len(toMerge))
	}
}

func TestNextActions_AllChangesRequested(t *testing.T) {
	prs := []ChildPR{
		{Number: 101, Title: "review/config", ReviewDecision: "CHANGES_REQUESTED"},
		{Number: 102, Title: "review/core", ReviewDecision: "CHANGES_REQUESTED"},
	}
	toFix, toMerge := nextActions(prs)

	if len(toFix) != 2 {
		t.Errorf("toFix: got %d items, want 2", len(toFix))
	}
	if len(toMerge) != 0 {
		t.Errorf("toMerge: expected empty, got %v", toMerge)
	}
}

func TestNextActions_AllPending(t *testing.T) {
	prs := []ChildPR{
		{Number: 101, Title: "review/config", ReviewDecision: ""},
		{Number: 102, Title: "review/core", ReviewDecision: "REVIEW_REQUIRED"},
	}
	toFix, toMerge := nextActions(prs)

	if len(toFix) != 0 {
		t.Errorf("toFix: expected empty, got %v", toFix)
	}
	if len(toMerge) != 0 {
		t.Errorf("toMerge: expected empty, got %v", toMerge)
	}
}

func TestNextActions_Empty(t *testing.T) {
	toFix, toMerge := nextActions([]ChildPR{})
	if len(toFix) != 0 {
		t.Errorf("toFix: expected empty, got %v", toFix)
	}
	if len(toMerge) != 0 {
		t.Errorf("toMerge: expected empty, got %v", toMerge)
	}
}

func TestNextActions_CaseInsensitive(t *testing.T) {
	prs := []ChildPR{
		{Number: 101, Title: "review/config", ReviewDecision: "approved"},
		{Number: 102, Title: "review/core", ReviewDecision: "changes_requested"},
	}
	toFix, toMerge := nextActions(prs)

	if len(toFix) != 1 || toFix[0].Number != 102 {
		t.Errorf("toFix: got %v, want PR#102", toFix)
	}
	if len(toMerge) != 1 || toMerge[0].Number != 101 {
		t.Errorf("toMerge: got %v, want PR#101", toMerge)
	}
}
