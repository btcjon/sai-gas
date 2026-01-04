package planoracle

import (
	"testing"
)

func TestParseComponents(t *testing.T) {
	analyzer := New("")

	tests := []struct {
		name     string
		desc     string
		expected int
	}{
		{
			name: "simple component",
			desc: `## Database Setup
- [ ] Create schema
- [ ] Add indexes

## API Implementation
- [ ] Build endpoints
- [ ] Add validation`,
			expected: 2,
		},
		{
			name:     "empty description",
			desc:     "",
			expected: 0,
		},
		{
			name: "single component",
			desc: `## Frontend
- [ ] Create components`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			components := analyzer.parseComponents(tt.desc)
			if len(components) != tt.expected {
				t.Errorf("got %d components, expected %d", len(components), tt.expected)
			}
		})
	}
}

func TestComplexityForPhase(t *testing.T) {
	tests := []struct {
		phase    int
		expected string
	}{
		{1, "S"},
		{2, "M"},
		{3, "M"},
		{4, "M"},
		{5, "L"},
		{6, "L"},
	}

	for _, tt := range tests {
		result := complexityForPhase(tt.phase)
		if result != tt.expected {
			t.Errorf("phase %d: got %q, expected %q", tt.phase, result, tt.expected)
		}
	}
}

func TestIsValidationTask(t *testing.T) {
	tests := []struct {
		title    string
		expected bool
	}{
		{"Test data loading", true},
		{"Review implementation", true},
		{"Validate API", true},
		{"Integration testing", true},
		{"Implement database schema", false},
		{"Create REST endpoints", false},
		{"Build UI components", false},
	}

	for _, tt := range tests {
		result := isValidationTask(tt.title)
		if result != tt.expected {
			t.Errorf("title %q: got %v, expected %v", tt.title, result, tt.expected)
		}
	}
}

func TestEstimateEffort(t *testing.T) {
	analyzer := New("")

	tests := []struct {
		name     string
		tasks    []*SubTask
		expected string
	}{
		{
			name:     "empty tasks",
			tasks:    []*SubTask{},
			expected: "< 1 sprint",
		},
		{
			name: "small effort",
			tasks: []*SubTask{
				{Complexity: "S"},
				{Complexity: "S"},
			},
			expected: "< 1 sprint",
		},
		{
			name: "medium effort",
			tasks: []*SubTask{
				{Complexity: "M"},
				{Complexity: "M"},
				{Complexity: "M"},
			},
			expected: "1-2 sprints",
		},
		{
			name: "large effort",
			tasks: []*SubTask{
				{Complexity: "L"},
				{Complexity: "L"},
				{Complexity: "L"},
				{Complexity: "L"},
				{Complexity: "L"},
			},
			expected: "> 3 sprints",
		},
	}

	for _, tt := range tests {
		result := analyzer.estimateEffort(tt.tasks)
		if result != tt.expected {
			t.Errorf("%s: got %q, expected %q", tt.name, result, tt.expected)
		}
	}
}

func TestCalculateParallelGroups(t *testing.T) {
	analyzer := New("")

	tasks := []*SubTask{
		{Index: 1, Dependencies: []int{}},
		{Index: 2, Dependencies: []int{}},
		{Index: 3, Dependencies: []int{1, 2}},
		{Index: 4, Dependencies: []int{1, 2}},
	}

	groups := analyzer.calculateParallelGroups(tasks)

	// Tasks 1 and 2 have the same dependencies (empty), so should be parallel
	// Tasks 3 and 4 have the same dependencies, so should be parallel
	if len(groups) != 2 {
		t.Errorf("got %d groups, expected 2", len(groups))
	}
}
