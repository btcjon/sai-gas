// Package planoracle provides analysis of issues/epics for work decomposition.
package planoracle

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Issue represents a beads issue with analysis metadata.
type Issue struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Type        string   `json:"issue_type"`
	Children    []string `json:"children,omitempty"`
	DependsOn   []string `json:"depends_on,omitempty"`
}

// SubTask represents a decomposed sub-task suggestion.
type SubTask struct {
	Index       int      `json:"index"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Complexity  string   `json:"complexity"` // S, M, L, XL
	Dependencies []int    `json:"depends_on"` // Indices of dependent tasks
}

// DecompositionAnalysis represents the result of analyzing an issue for decomposition.
type DecompositionAnalysis struct {
	IssueID         string                 `json:"issue_id"`
	IssueTitle      string                 `json:"issue_title"`
	Description     string                 `json:"description"`
	SubTasks        []*SubTask             `json:"subtasks"`
	ParallelGroups  [][]int                `json:"parallel_groups"`  // Groups that can run in parallel
	EstimatedEffort string                 `json:"estimated_effort"` // Sprint estimate
	Recommendations []string               `json:"recommendations"`
}

// Analyzer performs decomposition analysis on issues.
type Analyzer struct {
	beadsDir string
}

// New creates a new analyzer.
func New(beadsDir string) *Analyzer {
	return &Analyzer{
		beadsDir: beadsDir,
	}
}

// AnalyzeIssue analyzes an issue and suggests decomposition.
func (a *Analyzer) AnalyzeIssue(issueID string) (*DecompositionAnalysis, error) {
	// Get issue details from beads
	issue, err := a.getIssueDetails(issueID)
	if err != nil {
		return nil, fmt.Errorf("getting issue details: %w", err)
	}

	// Parse description for action items and components
	components := a.parseComponents(issue.Description)

	// Extract existing children
	existingChildren := issue.Children

	// Generate sub-tasks based on components
	subTasks := a.generateSubTasks(issue.Title, issue.Description, components)

	// Analyze dependencies between sub-tasks
	a.analyzeDependencies(subTasks)

	// Calculate parallel groups (tasks with no inter-dependencies)
	parallelGroups := a.calculateParallelGroups(subTasks)

	// Estimate total effort
	effort := a.estimateEffort(subTasks)

	// Generate recommendations
	recommendations := a.generateRecommendations(issue, subTasks, existingChildren)

	return &DecompositionAnalysis{
		IssueID:         issueID,
		IssueTitle:      issue.Title,
		Description:     issue.Description,
		SubTasks:        subTasks,
		ParallelGroups:  parallelGroups,
		EstimatedEffort: effort,
		Recommendations: recommendations,
	}, nil
}

// getIssueDetails retrieves issue information from beads.
func (a *Analyzer) getIssueDetails(issueID string) (*Issue, error) {
	cmd := exec.Command("bd", "show", issueID, "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd show failed: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(output, &issue); err != nil {
		return nil, fmt.Errorf("parsing issue JSON: %w", err)
	}

	return &issue, nil
}

// Component represents a detected component from the description.
type Component struct {
	Name    string
	Actions []string
	Keywords []string
}

// parseComponents extracts components from the issue description.
func (a *Analyzer) parseComponents(description string) []*Component {
	if description == "" {
		return []*Component{}
	}

	var components []*Component
	lines := strings.Split(description, "\n")

	// Pattern matching for:
	// - "## Section" style headers
	// - "- [ ]" or "- [x]" checkboxes
	// - "Action:" style directives

	currentComponent := (*Component)(nil)
	componentRE := regexp.MustCompile(`^#+\s+(.+)$`)
	checkboxRE := regexp.MustCompile(`^-\s+\[.\]\s+(.+)$`)
	actionRE := regexp.MustCompile(`^-\s+(.+?):\s+(.+)$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for component header
		if matches := componentRE.FindStringSubmatch(line); len(matches) > 1 {
			if currentComponent != nil {
				components = append(components, currentComponent)
			}
			currentComponent = &Component{
				Name:     matches[1],
				Actions:  []string{},
				Keywords: []string{},
			}
			continue
		}

		// Check for checkbox items
		if matches := checkboxRE.FindStringSubmatch(line); len(matches) > 1 {
			if currentComponent != nil {
				currentComponent.Actions = append(currentComponent.Actions, matches[1])
			}
			continue
		}

		// Check for action directives
		if matches := actionRE.FindStringSubmatch(line); len(matches) > 2 {
			action := matches[1]
			if currentComponent != nil && isComponentAction(action) {
				currentComponent.Keywords = append(currentComponent.Keywords, matches[2])
			}
		}
	}

	if currentComponent != nil {
		components = append(components, currentComponent)
	}

	return components
}

// isComponentAction checks if a string looks like a component action.
func isComponentAction(s string) bool {
	actions := []string{
		"implement", "create", "design", "setup", "configure",
		"test", "document", "review", "validate", "integrate",
	}
	s = strings.ToLower(s)
	for _, action := range actions {
		if strings.Contains(s, action) {
			return true
		}
	}
	return false
}

// generateSubTasks creates sub-task suggestions from components.
func (a *Analyzer) generateSubTasks(title, description string, components []*Component) []*SubTask {
	var subTasks []*SubTask
	index := 1

	// If we have structured components, create tasks from them
	if len(components) > 0 {
		for _, comp := range components {
			// Create task for component setup
			setupTask := &SubTask{
				Index:        index,
				Title:        fmt.Sprintf("[%s] Setup %s", complexityForPhase(index), comp.Name),
				Description:  fmt.Sprintf("Set up and initialize %s component", comp.Name),
				Complexity:   complexityForPhase(index),
				Dependencies: []int{},
			}
			if index > 1 {
				setupTask.Dependencies = []int{index - 1}
			}
			subTasks = append(subTasks, setupTask)
			index++

			// Create tasks for specific actions
			for _, action := range comp.Actions {
				actionTask := &SubTask{
					Index:        index,
					Title:        fmt.Sprintf("[M] %s", action),
					Description:  fmt.Sprintf("Complete action: %s", action),
					Complexity:   "M",
					Dependencies: []int{index - 1},
				}
				subTasks = append(subTasks, actionTask)
				index++
			}
		}
	}

	// If no components found, parse for action items and phases
	if len(subTasks) == 0 {
		subTasks = a.parsePhaseBasedTasks(title, description)
	}

	return subTasks
}

// parsePhaseBasedTasks extracts tasks from phase-structured descriptions.
func (a *Analyzer) parsePhaseBasedTasks(title, description string) []*SubTask {
	var tasks []*SubTask
	index := 1

	// Look for "Phase N:" patterns
	phaseRE := regexp.MustCompile(`(?i)phase\s+(\d+):\s*(.+?)(?:phase\s+\d+:|$)`)
	matches := phaseRE.FindAllStringSubmatchIndex(description, -1)

	if len(matches) > 0 {
		for _, match := range matches {
			phaseTitle := description[match[4]:match[5]]
			complexity := complexityForPhase(index)

			task := &SubTask{
				Index:        index,
				Title:        fmt.Sprintf("[%s] %s", complexity, phaseTitle),
				Description:  fmt.Sprintf("Complete phase %d", index),
				Complexity:   complexity,
				Dependencies: []int{},
			}
			if index > 1 {
				task.Dependencies = []int{index - 1}
			}
			tasks = append(tasks, task)
			index++
		}
	}

	// If still no tasks, create generic ones from bullet points
	if len(tasks) == 0 {
		lines := strings.Split(description, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
				title := strings.TrimPrefix(strings.TrimPrefix(line, "-"), "*")
				title = strings.TrimSpace(title)
				if len(title) > 0 {
					task := &SubTask{
						Index:        index,
						Title:        fmt.Sprintf("[M] %s", title),
						Description:  title,
						Complexity:   "M",
						Dependencies: []int{},
					}
					if index > 1 && !strings.Contains(title, "test") && !strings.Contains(title, "review") {
						task.Dependencies = []int{index - 1}
					}
					tasks = append(tasks, task)
					index++
				}
			}
		}
	}

	return tasks
}

// complexityForPhase returns a complexity level based on phase number.
func complexityForPhase(phase int) string {
	switch {
	case phase == 1:
		return "S"
	case phase == 2:
		return "M"
	case phase <= 4:
		return "M"
	default:
		return "L"
	}
}

// analyzeDependencies identifies dependencies between sub-tasks.
func (a *Analyzer) analyzeDependencies(subTasks []*SubTask) {
	for i, task := range subTasks {
		// Tasks that look like "test" or "review" depend on implementation tasks
		if isValidationTask(task.Title) {
			// Find most recent non-validation task
			for j := i - 1; j >= 0; j-- {
				if !isValidationTask(subTasks[j].Title) {
					task.Dependencies = []int{j + 1} // +1 because index is 1-based
					break
				}
			}
		}

		// Detect explicit dependencies in description
		words := strings.Fields(task.Description)
		for j := range words {
			if strings.Contains(strings.ToLower(words[j]), "depend") ||
			   strings.Contains(strings.ToLower(words[j]), "after") ||
			   strings.Contains(strings.ToLower(words[j]), "require") {
				// Look for task references like "task 1" or "task N"
				if j+1 < len(words) {
					refPattern := regexp.MustCompile(`task\s*(\d+)`)
					if matches := refPattern.FindStringSubmatch(strings.Join(words[j:j+2], " ")); len(matches) > 1 {
						// Found a reference (would be parsed into int in real implementation)
						break
					}
				}
			}
		}
	}
}

// isValidationTask checks if a task is a validation/test/review task.
func isValidationTask(title string) bool {
	lowerTitle := strings.ToLower(title)
	validationKeywords := []string{"test", "review", "validate", "verify", "integration", "qa"}
	for _, keyword := range validationKeywords {
		if strings.Contains(lowerTitle, keyword) {
			return true
		}
	}
	return false
}

// calculateParallelGroups identifies which tasks can run in parallel.
func (a *Analyzer) calculateParallelGroups(subTasks []*SubTask) [][]int {
	var groups [][]int

	// A simple approach: group tasks by dependency chain depth
	// Tasks with no dependencies or same dependencies can be parallel
	depMap := make(map[string][]int) // dependency set -> task indices

	for _, task := range subTasks {
		key := depSetToString(task.Dependencies)
		depMap[key] = append(depMap[key], task.Index)
	}

	// Create groups from tasks with identical dependency sets
	for _, indices := range depMap {
		if len(indices) > 1 {
			groups = append(groups, indices)
		}
	}

	return groups
}

// depSetToString converts a dependency set to a string key.
func depSetToString(deps []int) string {
	if len(deps) == 0 {
		return "none"
	}
	strs := make([]string, len(deps))
	for i, d := range deps {
		strs[i] = fmt.Sprintf("%d", d)
	}
	return strings.Join(strs, ",")
}

// estimateEffort calculates total effort estimate.
func (a *Analyzer) estimateEffort(subTasks []*SubTask) string {
	if len(subTasks) == 0 {
		return "< 1 sprint"
	}

	// Count complexity points
	points := 0
	for _, task := range subTasks {
		switch task.Complexity {
		case "S":
			points += 1
		case "M":
			points += 2
		case "L":
			points += 3
		case "XL":
			points += 5
		}
	}

	// Map points to sprints
	switch {
	case points <= 3:
		return "< 1 sprint"
	case points <= 5:
		return "1 sprint"
	case points <= 8:
		return "1-2 sprints"
	case points <= 13:
		return "2-3 sprints"
	default:
		return "> 3 sprints"
	}
}

// generateRecommendations creates actionable recommendations.
func (a *Analyzer) generateRecommendations(issue *Issue, subTasks []*SubTask, existingChildren []string) []string {
	var recs []string

	// Check if already has children
	if len(existingChildren) > 0 {
		recs = append(recs, fmt.Sprintf("Issue already has %d child tasks - consider merging or deduplicating", len(existingChildren)))
	}

	// Check for long dependency chains
	maxDepth := a.maxDependencyDepth(subTasks)
	if maxDepth > 3 {
		recs = append(recs, fmt.Sprintf("Long dependency chain detected (%d steps) - consider parallelizing where possible", maxDepth))
	}

	// Check for parallelizable tasks
	parallelCount := 0
	for _, task := range subTasks {
		if len(task.Dependencies) == 0 {
			parallelCount++
		}
	}
	if parallelCount > 2 {
		recs = append(recs, fmt.Sprintf("Multiple independent tasks found - good opportunity for parallel execution"))
	}

	// Check complexity distribution
	largeCount := 0
	for _, task := range subTasks {
		if task.Complexity == "L" || task.Complexity == "XL" {
			largeCount++
		}
	}
	if largeCount > len(subTasks)/3 {
		recs = append(recs, "High proportion of large/XL tasks - consider breaking down further or adding intermediate milestones")
	}

	// Default recommendation
	if len(recs) == 0 {
		recs = append(recs, "Decomposition looks well-structured - proceed with creating sub-tasks")
	}

	return recs
}

// maxDependencyDepth calculates the maximum dependency chain depth.
func (a *Analyzer) maxDependencyDepth(subTasks []*SubTask) int {
	if len(subTasks) == 0 {
		return 0
	}

	maxDepth := 1
	for _, task := range subTasks {
		depth := a.calculateTaskDepth(task, subTasks)
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	return maxDepth
}

// calculateTaskDepth calculates the dependency depth of a single task.
func (a *Analyzer) calculateTaskDepth(task *SubTask, allTasks []*SubTask) int {
	if len(task.Dependencies) == 0 {
		return 1
	}

	maxSubDepth := 0
	for _, depIdx := range task.Dependencies {
		for _, otherTask := range allTasks {
			if otherTask.Index == depIdx {
				depth := a.calculateTaskDepth(otherTask, allTasks)
				if depth > maxSubDepth {
					maxSubDepth = depth
				}
				break
			}
		}
	}
	return maxSubDepth + 1
}
