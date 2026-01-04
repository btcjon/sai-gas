// Package plan provides utilities for parsing planning documents into beads epics.
package plan

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

// Phase represents a phase in a plan (e.g., "## Phase 1: Setup").
type Phase struct {
	Name  string
	Order int
	Tasks []Task
}

// Task represents a task item from a markdown checklist.
type Task struct {
	Title      string
	Done       bool
	Phase      string
	Order      int
	DependsOn  []string // Task titles this task depends on
	RawLine    string   // Original markdown line
}

// Plan represents a parsed planning document.
type Plan struct {
	Title  string
	Phases []Phase
	Tasks  []Task // Flat list of all tasks for convenience
}

// TaskMention extracts task title from a "(depends on X)" reference.
var dependsPattern = regexp.MustCompile(`\(depends on ([^)]+)\)`)

// phasePattern matches "## Phase N: Title" or "## Title"
var phasePattern = regexp.MustCompile(`^##\s+(?:Phase\s+\d+:\s*)?(.+)$`)

// taskPattern matches "- [ ] Task" or "- [x] Task"
var taskPattern = regexp.MustCompile(`^[-*]\s+\[([ xX])\]\s+(.+)$`)

// ParseMarkdown parses a markdown planning document into a Plan.
// Expected format:
//
//	# Epic Title
//
//	## Phase 1: Setup
//	- [ ] Task 1
//	- [ ] Task 2 (depends on Task 1)
//
//	## Phase 2: Implementation
//	- [ ] Task 3
func ParseMarkdown(content string) (*Plan, error) {
	plan := &Plan{
		Phases: make([]Phase, 0),
		Tasks:  make([]Task, 0),
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	var currentPhase *Phase
	phaseOrder := 0
	taskOrder := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check for main title (# Title)
		if strings.HasPrefix(line, "# ") && !strings.HasPrefix(line, "## ") {
			plan.Title = strings.TrimPrefix(line, "# ")
			plan.Title = strings.TrimSpace(plan.Title)
			continue
		}

		// Check for phase header (## Phase N: Title)
		if match := phasePattern.FindStringSubmatch(line); match != nil {
			// Save current phase if exists
			if currentPhase != nil {
				plan.Phases = append(plan.Phases, *currentPhase)
			}

			phaseOrder++
			currentPhase = &Phase{
				Name:  match[1],
				Order: phaseOrder,
				Tasks: make([]Task, 0),
			}
			continue
		}

		// Check for task item (- [ ] Task or - [x] Task)
		if match := taskPattern.FindStringSubmatch(line); match != nil {
			done := strings.ToLower(match[1]) == "x"
			taskText := match[2]

			// Extract dependencies from "(depends on X)" pattern
			var dependsOn []string
			if depMatch := dependsPattern.FindStringSubmatch(taskText); depMatch != nil {
				// Clean the dependency reference
				depRef := strings.TrimSpace(depMatch[1])
				dependsOn = append(dependsOn, depRef)
				// Remove the "(depends on X)" from title
				taskText = dependsPattern.ReplaceAllString(taskText, "")
				taskText = strings.TrimSpace(taskText)
			}

			taskOrder++
			phaseName := ""
			if currentPhase != nil {
				phaseName = currentPhase.Name
			}

			task := Task{
				Title:     taskText,
				Done:      done,
				Phase:     phaseName,
				Order:     taskOrder,
				DependsOn: dependsOn,
				RawLine:   line,
			}

			// Add to current phase if exists
			if currentPhase != nil {
				currentPhase.Tasks = append(currentPhase.Tasks, task)
			}

			// Add to flat list
			plan.Tasks = append(plan.Tasks, task)
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning markdown: %w", err)
	}

	// Save the last phase
	if currentPhase != nil {
		plan.Phases = append(plan.Phases, *currentPhase)
	}

	// Validate
	if plan.Title == "" {
		return nil, fmt.Errorf("markdown must have a title (# Title)")
	}
	if len(plan.Tasks) == 0 {
		return nil, fmt.Errorf("markdown must have at least one task (- [ ] Task)")
	}

	return plan, nil
}

// ResolveDependencies resolves task title references to task indices.
// Returns a map from task index to list of dependency indices.
// Also adds phase-based implicit dependencies (tasks in phase N+1 depend on phase N tasks).
func (p *Plan) ResolveDependencies(addPhaseDeps bool) map[int][]int {
	deps := make(map[int][]int)

	// Build title -> index map
	titleToIndex := make(map[string]int)
	for i, task := range p.Tasks {
		titleToIndex[strings.ToLower(task.Title)] = i
	}

	// Resolve explicit dependencies
	for i, task := range p.Tasks {
		for _, depTitle := range task.DependsOn {
			if depIdx, ok := titleToIndex[strings.ToLower(depTitle)]; ok {
				deps[i] = append(deps[i], depIdx)
			}
		}
	}

	// Add implicit phase dependencies
	if addPhaseDeps && len(p.Phases) > 1 {
		for i := 1; i < len(p.Phases); i++ {
			prevPhase := p.Phases[i-1]
			currPhase := p.Phases[i]

			// First task in current phase depends on all tasks in previous phase
			if len(currPhase.Tasks) > 0 && len(prevPhase.Tasks) > 0 {
				// Find the indices in the flat task list
				for _, currTask := range currPhase.Tasks {
					currIdx := -1
					for j, t := range p.Tasks {
						if t.Title == currTask.Title && t.Phase == currTask.Phase {
							currIdx = j
							break
						}
					}
					if currIdx == -1 {
						continue
					}

					// Only add dep for first task in phase (others follow linearly)
					if currTask.Order == currPhase.Tasks[0].Order {
						for _, prevTask := range prevPhase.Tasks {
							for j, t := range p.Tasks {
								if t.Title == prevTask.Title && t.Phase == prevTask.Phase {
									deps[currIdx] = append(deps[currIdx], j)
									break
								}
							}
						}
					}
					break // Only process first task
				}
			}
		}
	}

	return deps
}
