package plan

import (
	"testing"
)

func TestParseMarkdown(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantTitle  string
		wantPhases int
		wantTasks  int
		wantErr    bool
	}{
		{
			name: "basic plan",
			content: `# My Epic

## Phase 1: Setup
- [ ] Task 1
- [ ] Task 2

## Phase 2: Implementation
- [ ] Task 3
`,
			wantTitle:  "My Epic",
			wantPhases: 2,
			wantTasks:  3,
			wantErr:    false,
		},
		{
			name: "with dependencies",
			content: `# Epic with Deps

## Phase 1: Setup
- [ ] Task 1
- [ ] Task 2 (depends on Task 1)
`,
			wantTitle:  "Epic with Deps",
			wantPhases: 1,
			wantTasks:  2,
			wantErr:    false,
		},
		{
			name: "with completed tasks",
			content: `# Test Epic

## Phase 1
- [x] Done task
- [ ] Pending task
`,
			wantTitle:  "Test Epic",
			wantPhases: 1,
			wantTasks:  2,
			wantErr:    false,
		},
		{
			name:    "missing title",
			content: `## Phase 1
- [ ] Task 1
`,
			wantErr: true,
		},
		{
			name:    "no tasks",
			content: `# Epic Title`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := ParseMarkdown(tt.content)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseMarkdown() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseMarkdown() error = %v", err)
				return
			}

			if plan.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", plan.Title, tt.wantTitle)
			}

			if len(plan.Phases) != tt.wantPhases {
				t.Errorf("Phases count = %d, want %d", len(plan.Phases), tt.wantPhases)
			}

			if len(plan.Tasks) != tt.wantTasks {
				t.Errorf("Tasks count = %d, want %d", len(plan.Tasks), tt.wantTasks)
			}
		})
	}
}

func TestTaskDependency(t *testing.T) {
	content := `# Dep Test

## Phase 1
- [ ] Task A
- [ ] Task B (depends on Task A)
`

	plan, err := ParseMarkdown(content)
	if err != nil {
		t.Fatalf("ParseMarkdown() error = %v", err)
	}

	// Check Task B has dependency on Task A
	taskB := plan.Tasks[1]
	if len(taskB.DependsOn) != 1 {
		t.Fatalf("Task B DependsOn count = %d, want 1", len(taskB.DependsOn))
	}

	if taskB.DependsOn[0] != "Task A" {
		t.Errorf("Task B DependsOn[0] = %q, want %q", taskB.DependsOn[0], "Task A")
	}

	// Check dependency is stripped from title
	if taskB.Title != "Task B" {
		t.Errorf("Task B Title = %q, want %q", taskB.Title, "Task B")
	}
}

func TestResolveDependencies(t *testing.T) {
	content := `# Resolve Test

## Phase 1
- [ ] Task A
- [ ] Task B (depends on Task A)

## Phase 2
- [ ] Task C
`

	plan, err := ParseMarkdown(content)
	if err != nil {
		t.Fatalf("ParseMarkdown() error = %v", err)
	}

	// Test explicit deps only
	deps := plan.ResolveDependencies(false)

	// Task B (index 1) should depend on Task A (index 0)
	if len(deps[1]) != 1 || deps[1][0] != 0 {
		t.Errorf("Task B deps = %v, want [0]", deps[1])
	}

	// Test with phase deps
	deps = plan.ResolveDependencies(true)

	// Task C (index 2) should depend on Task A and Task B (phase 1 tasks)
	if len(deps[2]) < 1 {
		t.Errorf("Task C should have phase dependencies, got %v", deps[2])
	}
}

func TestParseRepoString(t *testing.T) {
	tests := []struct {
		repo      string
		wantOwner string
		wantName  string
		wantErr   bool
	}{
		{"owner/repo", "owner", "repo", false},
		{"anthropic/beads", "anthropic", "beads", false},
		{"invalid", "", "", true},
		{"/repo", "", "", true},
		{"owner/", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.repo, func(t *testing.T) {
			owner, name, err := ParseRepoString(tt.repo)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseRepoString(%q) expected error", tt.repo)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseRepoString(%q) error = %v", tt.repo, err)
				return
			}

			if owner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", owner, tt.wantOwner)
			}

			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
		})
	}
}
