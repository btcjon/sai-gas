package convoy

import (
	"testing"
)

func TestExtractRigFromIssueID(t *testing.T) {
	tests := []struct {
		issueID  string
		expected string
	}{
		{"gt-abc123", "gastown"},
		{"bd-xyz456", "beads"},
		{"gp-feature", "greenplace"},
		{"hq-cv-convoy", ""}, // town-level, no specific rig
		{"invalid", ""},      // no hyphen
		{"x-short", ""},      // unknown prefix
	}

	for _, test := range tests {
		t.Run(test.issueID, func(t *testing.T) {
			result := extractRigFromIssueID(test.issueID)
			if result != test.expected {
				t.Errorf("extractRigFromIssueID(%q) = %q, want %q", test.issueID, result, test.expected)
			}
		})
	}
}

func TestFindTargetForIssue(t *testing.T) {
	polecats := []idlePolecat{
		{Rig: "gastown", Name: "toast", Target: "gastown/polecats/toast"},
		{Rig: "beads", Name: "obsidian", Target: "beads/polecats/obsidian"},
	}

	tests := []struct {
		name      string
		issue     ReadyIssue
		polecats  []idlePolecat
		targetRig string
		expected  string
	}{
		{
			name:     "match issue prefix to rig",
			issue:    ReadyIssue{ID: "gt-abc123", Title: "Test issue"},
			polecats: polecats,
			expected: "gastown/polecats/toast",
		},
		{
			name:     "specific rig requested",
			issue:    ReadyIssue{ID: "gt-abc123", Title: "Test issue"},
			polecats: polecats,
			targetRig: "beads",
			expected: "beads/polecats/obsidian",
		},
		{
			name:     "no matching rig uses first available",
			issue:    ReadyIssue{ID: "unknown-abc", Title: "Unknown prefix"},
			polecats: polecats,
			expected: "gastown/polecats/toast",
		},
		{
			name:     "empty polecats returns empty",
			issue:    ReadyIssue{ID: "gt-abc123", Title: "Test issue"},
			polecats: nil,
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := findTargetForIssue(test.issue, test.polecats, test.targetRig)
			if result != test.expected {
				t.Errorf("findTargetForIssue() = %q, want %q", result, test.expected)
			}
		})
	}
}
