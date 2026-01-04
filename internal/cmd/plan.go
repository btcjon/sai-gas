package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/beads"
	"github.com/steveyegge/gastown/internal/plan"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/workspace"
)

// Plan command flags
var (
	planImportFrom    string
	planImportRepo    string
	planImportLabel   string
	planImportDryRun  bool
	planStructureDeps bool
)

var planCmd = &cobra.Command{
	Use:     "plan",
	GroupID: GroupWork,
	Short:   "Convert planning inputs into beads epics",
	RunE:    requireSubcommand,
	Long: `Convert planning inputs into beads epics.

This command helps convert external planning documents (markdown files,
GitHub issues) into structured beads epics with tasks and dependencies.

Commands:
  import     Import tasks from markdown or GitHub
  structure  Analyze epic children and add dependencies

Examples:
  gt plan import --from markdown plan.md
  gt plan import --from github --repo owner/repo --label epic
  gt plan structure gt-abc123`,
}

var planImportCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import tasks from markdown or GitHub into an epic",
	Long: `Import tasks from a planning document into a beads epic.

Sources:
  markdown  Parse a markdown file with phases and tasks
  github    Import issues from a GitHub repository by label

Markdown format:
  # Epic Title

  ## Phase 1: Setup
  - [ ] Task 1
  - [ ] Task 2 (depends on Task 1)

  ## Phase 2: Implementation
  - [ ] Task 3
  - [ ] Task 4

GitHub import:
  Uses the gh CLI to fetch issues by label and creates tasks.

Examples:
  gt plan import --from markdown plan.md
  gt plan import --from github --repo owner/repo --label epic`,
	RunE: runPlanImport,
}

var planStructureCmd = &cobra.Command{
	Use:   "structure <epic-id>",
	Short: "Analyze epic children and add dependencies",
	Long: `Analyze an epic's child tasks and add dependencies based on patterns.

This command examines the children of an epic and:
1. Looks for explicit "(depends on X)" mentions in titles/descriptions
2. Infers phase-based ordering from titles containing "Phase N"
3. Adds appropriate dependency relationships

Use --deps=false to only analyze without adding dependencies.

Examples:
  gt plan structure gt-abc123
  gt plan structure gt-abc123 --deps=false`,
	Args: cobra.ExactArgs(1),
	RunE: runPlanStructure,
}

func init() {
	// Import flags
	planImportCmd.Flags().StringVar(&planImportFrom, "from", "markdown", "Source type: markdown or github")
	planImportCmd.Flags().StringVar(&planImportRepo, "repo", "", "GitHub repository (owner/repo) for github source")
	planImportCmd.Flags().StringVar(&planImportLabel, "label", "", "GitHub label to filter issues")
	planImportCmd.Flags().BoolVar(&planImportDryRun, "dry-run", false, "Show what would be imported without creating")

	// Structure flags
	planStructureCmd.Flags().BoolVar(&planStructureDeps, "deps", true, "Add dependencies (set false to analyze only)")

	// Add subcommands
	planCmd.AddCommand(planImportCmd)
	planCmd.AddCommand(planStructureCmd)

	rootCmd.AddCommand(planCmd)
}

func runPlanImport(cmd *cobra.Command, args []string) error {
	var p *plan.Plan
	var err error

	switch planImportFrom {
	case "markdown":
		if len(args) < 1 {
			return fmt.Errorf("markdown source requires a file path")
		}
		p, err = importFromMarkdown(args[0])
		if err != nil {
			return err
		}

	case "github":
		if planImportRepo == "" {
			return fmt.Errorf("github source requires --repo flag")
		}
		p, err = importFromGitHub(planImportRepo, planImportLabel)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("unknown source type: %s (use: markdown or github)", planImportFrom)
	}

	// Display what we found
	fmt.Printf("%s Parsed plan: %s\n", style.Bold.Render("->"), p.Title)
	fmt.Printf("   Phases: %d, Tasks: %d\n\n", len(p.Phases), len(p.Tasks))

	for _, phase := range p.Phases {
		fmt.Printf("   %s %s\n", style.Dim.Render("Phase:"), phase.Name)
		for _, task := range phase.Tasks {
			status := "[ ]"
			if task.Done {
				status = "[x]"
			}
			deps := ""
			if len(task.DependsOn) > 0 {
				deps = fmt.Sprintf(" (depends on: %s)", strings.Join(task.DependsOn, ", "))
			}
			fmt.Printf("     %s %s%s\n", status, task.Title, deps)
		}
		fmt.Println()
	}

	// Dry run stops here
	if planImportDryRun {
		fmt.Printf("%s Dry run - no beads created\n", style.Dim.Render("Note:"))
		return nil
	}

	// Create the epic and tasks
	return createEpicFromPlan(p)
}

func importFromMarkdown(filePath string) (*plan.Plan, error) {
	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	// Parse markdown
	p, err := plan.ParseMarkdown(string(content))
	if err != nil {
		return nil, fmt.Errorf("parsing markdown: %w", err)
	}

	return p, nil
}

func importFromGitHub(repo, label string) (*plan.Plan, error) {
	// Validate repo format
	owner, name, err := plan.ParseRepoString(repo)
	if err != nil {
		return nil, err
	}

	fmt.Printf("%s Fetching issues from %s/%s", style.Dim.Render("..."), owner, name)
	if label != "" {
		fmt.Printf(" (label: %s)", label)
	}
	fmt.Println()

	// Fetch issues
	issues, err := plan.FetchGitHubIssues(repo, label)
	if err != nil {
		return nil, fmt.Errorf("fetching GitHub issues: %w", err)
	}

	if len(issues) == 0 {
		return nil, fmt.Errorf("no issues found matching criteria")
	}

	// Build epic title
	epicTitle := fmt.Sprintf("%s/%s", owner, name)
	if label != "" {
		epicTitle = fmt.Sprintf("%s (%s)", epicTitle, label)
	}

	// Convert to plan
	return plan.GitHubIssuesToPlan(issues, epicTitle), nil
}

func createEpicFromPlan(p *plan.Plan) error {
	// Find working directory - use current rig's beads
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	// Try to find rig beads directory
	beadsDir := filepath.Join(cwd, ".beads")
	if _, err := os.Stat(beadsDir); os.IsNotExist(err) {
		// Try town root
		townRoot, townErr := workspace.FindFromCwd()
		if townErr != nil {
			return fmt.Errorf("not in a beads-enabled directory (no .beads found)")
		}
		beadsDir = filepath.Join(townRoot, ".beads")
		if _, err := os.Stat(beadsDir); os.IsNotExist(err) {
			return fmt.Errorf("no .beads directory found in town root")
		}
	}

	// Create the epic
	fmt.Printf("\n%s Creating epic and tasks...\n\n", style.Bold.Render("->"))

	epicArgs := []string{
		"create",
		"--type=epic",
		"--title=" + p.Title,
		"--json",
	}

	epicCmd := exec.Command("bd", epicArgs...)
	epicCmd.Dir = beadsDir
	epicOutput, err := epicCmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("creating epic: %s", string(exitErr.Stderr))
		}
		return fmt.Errorf("creating epic: %w", err)
	}

	// Parse epic ID from output
	epicID := extractIDFromJSON(string(epicOutput))
	if epicID == "" {
		return fmt.Errorf("could not parse epic ID from output")
	}

	fmt.Printf("   %s Created epic: %s\n", style.Bold.Render("[+]"), epicID)

	// Create tasks as children of the epic
	taskIDs := make(map[string]string) // task title -> bead ID
	taskOrder := make([]string, 0)      // ordered list of task titles

	for _, task := range p.Tasks {
		description := fmt.Sprintf("Phase: %s\nOrder: %d", task.Phase, task.Order)
		if len(task.DependsOn) > 0 {
			description += fmt.Sprintf("\nDepends on: %s", strings.Join(task.DependsOn, ", "))
		}

		taskArgs := []string{
			"create",
			"--type=task",
			"--title=" + task.Title,
			"--description=" + description,
			"--parent=" + epicID,
			"--json",
		}

		taskCmd := exec.Command("bd", taskArgs...)
		taskCmd.Dir = beadsDir
		taskOutput, err := taskCmd.Output()
		if err != nil {
			fmt.Printf("   %s Failed to create task %q: %v\n", style.Dim.Render("[!]"), task.Title, err)
			continue
		}

		taskID := extractIDFromJSON(string(taskOutput))
		if taskID == "" {
			fmt.Printf("   %s Could not parse ID for task %q\n", style.Dim.Render("[!]"), task.Title)
			continue
		}

		taskIDs[task.Title] = taskID
		taskOrder = append(taskOrder, task.Title)

		status := "[ ]"
		if task.Done {
			status = "[x]"
		}
		fmt.Printf("   %s Created: %s %s\n", style.Dim.Render("   "), status, taskID)
	}

	// Add dependencies
	deps := p.ResolveDependencies(true) // Include phase-based deps
	depCount := 0

	for taskIdx, depIdxs := range deps {
		if taskIdx >= len(p.Tasks) {
			continue
		}
		taskTitle := p.Tasks[taskIdx].Title
		taskID, ok := taskIDs[taskTitle]
		if !ok {
			continue
		}

		for _, depIdx := range depIdxs {
			if depIdx >= len(p.Tasks) {
				continue
			}
			depTitle := p.Tasks[depIdx].Title
			depID, ok := taskIDs[depTitle]
			if !ok {
				continue
			}

			// Add dependency: taskID depends on depID
			depArgs := []string{"dep", "add", taskID, depID}
			depCmd := exec.Command("bd", depArgs...)
			depCmd.Dir = beadsDir
			if err := depCmd.Run(); err != nil {
				fmt.Printf("   %s Failed to add dep %s -> %s: %v\n",
					style.Dim.Render("[!]"), taskID, depID, err)
				continue
			}
			depCount++
		}
	}

	// Summary
	fmt.Printf("\n%s Import complete!\n", style.Bold.Render("[/]"))
	fmt.Printf("   Epic:         %s\n", epicID)
	fmt.Printf("   Tasks:        %d created\n", len(taskIDs))
	fmt.Printf("   Dependencies: %d added\n", depCount)
	fmt.Printf("\n   View: bd show %s\n", epicID)

	return nil
}

func runPlanStructure(cmd *cobra.Command, args []string) error {
	epicID := args[0]

	// Find working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	b := beads.New(cwd)

	// Get the epic
	epic, err := b.Show(epicID)
	if err != nil {
		return fmt.Errorf("getting epic: %w", err)
	}

	if epic.Type != "epic" {
		return fmt.Errorf("%s is not an epic (type: %s)", epicID, epic.Type)
	}

	fmt.Printf("%s Analyzing epic: %s\n", style.Bold.Render("->"), epic.Title)
	fmt.Printf("   Children: %d\n\n", len(epic.Children))

	if len(epic.Children) == 0 {
		fmt.Printf("   No children to analyze.\n")
		return nil
	}

	// Fetch all children
	children, err := b.ShowMultiple(epic.Children)
	if err != nil {
		return fmt.Errorf("fetching children: %w", err)
	}

	// Group by phase (look for "Phase N" pattern in title or description)
	phases := make(map[string][]*beads.Issue)
	phaseOrder := make([]string, 0)
	noPhase := make([]*beads.Issue, 0)

	phasePattern := regexp.MustCompile(`Phase\s+(\d+)`)

	for _, child := range children {
		if child == nil {
			continue
		}

		match := phasePattern.FindStringSubmatch(child.Title)
		if match == nil {
			match = phasePattern.FindStringSubmatch(child.Description)
		}

		if match != nil {
			phaseName := "Phase " + match[1]
			if _, exists := phases[phaseName]; !exists {
				phaseOrder = append(phaseOrder, phaseName)
			}
			phases[phaseName] = append(phases[phaseName], child)
		} else {
			noPhase = append(noPhase, child)
		}
	}

	// Display analysis
	for _, phaseName := range phaseOrder {
		issues := phases[phaseName]
		fmt.Printf("   %s %s (%d tasks)\n", style.Bold.Render("Phase:"), phaseName, len(issues))
		for _, issue := range issues {
			fmt.Printf("      - %s (%s)\n", issue.Title, issue.ID)
		}
	}

	if len(noPhase) > 0 {
		fmt.Printf("   %s Unphased (%d tasks)\n", style.Bold.Render("Other:"), len(noPhase))
		for _, issue := range noPhase {
			fmt.Printf("      - %s (%s)\n", issue.Title, issue.ID)
		}
	}

	// Add dependencies if requested
	if !planStructureDeps {
		fmt.Printf("\n%s Analyze only mode - no dependencies added\n", style.Dim.Render("Note:"))
		return nil
	}

	if len(phaseOrder) < 2 {
		fmt.Printf("\n%s Not enough phases to add cross-phase dependencies\n", style.Dim.Render("Note:"))
		return nil
	}

	// Add dependencies: first task in phase N depends on all tasks in phase N-1
	fmt.Printf("\n%s Adding phase dependencies...\n", style.Bold.Render("->"))
	depCount := 0

	for i := 1; i < len(phaseOrder); i++ {
		prevPhase := phaseOrder[i-1]
		currPhase := phaseOrder[i]

		prevTasks := phases[prevPhase]
		currTasks := phases[currPhase]

		if len(prevTasks) == 0 || len(currTasks) == 0 {
			continue
		}

		// First task in current phase depends on all previous phase tasks
		firstTask := currTasks[0]

		for _, prevTask := range prevTasks {
			if err := b.AddDependency(firstTask.ID, prevTask.ID); err != nil {
				fmt.Printf("   %s Failed: %s -> %s\n", style.Dim.Render("[!]"), firstTask.ID, prevTask.ID)
				continue
			}
			fmt.Printf("   %s %s depends on %s\n", style.Dim.Render("[+]"), firstTask.ID, prevTask.ID)
			depCount++
		}
	}

	fmt.Printf("\n%s Structure complete! Added %d dependencies\n", style.Bold.Render("[/]"), depCount)

	return nil
}

// extractIDFromJSON extracts the "id" field from JSON output.
// Simple extraction without full JSON parsing.
func extractIDFromJSON(jsonStr string) string {
	// Look for "id":"..." or "id": "..."
	idStart := strings.Index(jsonStr, `"id"`)
	if idStart == -1 {
		return ""
	}

	// Find the colon
	colonPos := strings.Index(jsonStr[idStart:], ":")
	if colonPos == -1 {
		return ""
	}

	// Find the opening quote
	start := idStart + colonPos + 1
	for start < len(jsonStr) && (jsonStr[start] == ' ' || jsonStr[start] == '\t') {
		start++
	}

	if start >= len(jsonStr) || jsonStr[start] != '"' {
		return ""
	}
	start++ // Skip opening quote

	// Find the closing quote
	end := strings.Index(jsonStr[start:], `"`)
	if end == -1 {
		return ""
	}

	return jsonStr[start : start+end]
}
