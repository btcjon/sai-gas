package cmd

import (
	"bytes"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/epictemplate"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/workspace"
)

// Template command flags
var (
	templateShowJSON  bool
	templateListJSON  bool
	templateSpawnRig  string
	templateSpawnDry  bool
)

var templateCmd = &cobra.Command{
	Use:     "template",
	Aliases: []string{"templates", "tmpl"},
	GroupID: GroupWork,
	Short:   "Manage epic templates for batch work",
	RunE:    requireSubcommand,
	Long: `Manage epic templates for common batch work patterns.

Templates define reusable patterns for batch work scenarios that can be
instantiated as beads epics. Each template contains phases with predefined
issues that get created when the template is spawned.

Template search paths (in order):
  1. <rig>/templates/ (project)
  2. ~/.config/gastown/templates/ (user)
  3. <town>/templates/ (town-wide)

Commands:
  list    List available templates
  show    Display template details
  spawn   Instantiate a template as a beads epic

Examples:
  gt template list                          # List all templates
  gt template show basic-batch              # Show template details
  gt spawn --template basic-batch --epic gt-abc  # Create beads from template`,
}

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available templates",
	Long: `List available templates from all search paths.

Searches for template files (.yaml, .yml) in:
  1. <rig>/templates/ (project)
  2. ~/.config/gastown/templates/ (user)
  3. <town>/templates/ (town-wide)

Examples:
  gt template list            # List all templates
  gt template list --json     # JSON output`,
	RunE: runTemplateList,
}

var templateShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Display template details",
	Long: `Display detailed information about a template.

Shows:
  - Template metadata (name, description)
  - Phases and their issues
  - Total issue count

Examples:
  gt template show basic-batch
  gt template show release-prep --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTemplateShow,
}

var templateSpawnCmd = &cobra.Command{
	Use:   "spawn <template-name> <epic-id>",
	Short: "Instantiate a template as beads epic issues",
	Long: `Instantiate a template by creating beads issues within an epic.

This command:
  1. Loads the specified template
  2. Creates issues for each phase under the target epic
  3. Sets up dependencies between phases (later phases depend on earlier)

The epic-id should be an existing epic bead that will parent the template issues.

Options:
  --rig=NAME  Target rig for beads operations (default: auto-detect)
  --dry-run   Show what would be created without actually creating

Examples:
  gt template spawn basic-batch gt-abc          # Spawn template under epic
  gt template spawn release-prep gt-xyz --dry-run  # Preview what would be created`,
	Args: cobra.ExactArgs(2),
	RunE: runTemplateSpawn,
}

// Deprecated: gt spawn is now preferred, but keep this for backwards compat
var spawnCmd = &cobra.Command{
	Use:    "spawn",
	Hidden: true, // Hide from help since gt template spawn is preferred
	Short:  "Spawn a template (use 'gt template spawn' instead)",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Note: 'gt spawn' is deprecated. Use 'gt template spawn' instead.")
		return runTemplateSpawn(cmd, args)
	},
}

func init() {
	// List flags
	templateListCmd.Flags().BoolVar(&templateListJSON, "json", false, "Output as JSON")

	// Show flags
	templateShowCmd.Flags().BoolVar(&templateShowJSON, "json", false, "Output as JSON")

	// Spawn flags
	templateSpawnCmd.Flags().StringVar(&templateSpawnRig, "rig", "", "Target rig (default: auto-detect)")
	templateSpawnCmd.Flags().BoolVar(&templateSpawnDry, "dry-run", false, "Preview without creating")

	// Add deprecated spawn flags
	spawnCmd.Flags().StringVar(&templateSpawnRig, "rig", "", "Target rig")
	spawnCmd.Flags().BoolVar(&templateSpawnDry, "dry-run", false, "Preview without creating")
	var templateFlag, epicFlag string
	spawnCmd.Flags().StringVar(&templateFlag, "template", "", "Template name")
	spawnCmd.Flags().StringVar(&epicFlag, "epic", "", "Epic ID")

	// Add subcommands
	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateShowCmd)
	templateCmd.AddCommand(templateSpawnCmd)

	rootCmd.AddCommand(templateCmd)
	rootCmd.AddCommand(spawnCmd)
}

// getSearchPaths returns template search paths based on current location.
func getSearchPaths() []string {
	var projectRoot, townRoot string

	// Try to find town root
	if root, err := workspace.FindFromCwd(); err == nil {
		townRoot = root
	}

	// Try to find project root (current rig)
	cwd, _ := os.Getwd()
	if cwd != "" {
		// Walk up looking for .beads directory
		dir := cwd
		for {
			if _, err := os.Stat(filepath.Join(dir, ".beads")); err == nil {
				projectRoot = dir
				break
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	return epictemplate.SearchPaths(projectRoot, townRoot)
}

func runTemplateList(cmd *cobra.Command, args []string) error {
	searchPaths := getSearchPaths()
	templates, err := epictemplate.ListTemplates(searchPaths)
	if err != nil {
		return fmt.Errorf("listing templates: %w", err)
	}

	if len(templates) == 0 {
		fmt.Println("No templates found.")
		fmt.Println("\nTemplate search paths:")
		for _, p := range searchPaths {
			fmt.Printf("  - %s\n", p)
		}
		fmt.Println("\nCreate a template with a .yaml file in one of these directories.")
		return nil
	}

	if templateListJSON {
		type templateInfo struct {
			Name string `json:"name"`
			Path string `json:"path"`
		}
		var list []templateInfo
		for _, name := range epictemplate.SortedNames(templates) {
			list = append(list, templateInfo{Name: name, Path: templates[name]})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	}

	fmt.Printf("%s\n\n", style.Bold.Render("Available Templates"))
	for _, name := range epictemplate.SortedNames(templates) {
		// Load template to get description
		tmpl, err := epictemplate.LoadTemplate(templates[name])
		if err != nil {
			fmt.Printf("  %s %s\n", name, style.Dim.Render("(error loading)"))
			continue
		}

		desc := tmpl.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		if desc == "" {
			desc = style.Dim.Render("(no description)")
		}

		fmt.Printf("  %s - %s\n", style.Bold.Render(name), desc)
	}

	fmt.Printf("\nUse 'gt template show <name>' for details.\n")
	return nil
}

func runTemplateShow(cmd *cobra.Command, args []string) error {
	templateName := args[0]
	searchPaths := getSearchPaths()

	path, err := epictemplate.FindTemplate(templateName, searchPaths)
	if err != nil {
		return err
	}

	tmpl, err := epictemplate.LoadTemplate(path)
	if err != nil {
		return err
	}

	if templateShowJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(tmpl)
	}

	// Human-readable output
	fmt.Printf("%s\n\n", style.Bold.Render(tmpl.Name))

	if tmpl.Description != "" {
		fmt.Printf("  %s\n\n", tmpl.Description)
	}

	fmt.Printf("  Source: %s\n", style.Dim.Render(path))
	fmt.Printf("  Phases: %d\n", len(tmpl.Phases))
	fmt.Printf("  Issues: %d total\n\n", tmpl.TotalIssues())

	for i, phase := range tmpl.Phases {
		phaseNum := i + 1
		fmt.Printf("  %s Phase %d: %s\n", style.Bold.Render(">>"), phaseNum, phase.Name)

		if len(phase.Issues) == 0 {
			fmt.Printf("     %s\n", style.Dim.Render("(placeholder - issues added at spawn time)"))
		} else {
			for _, issue := range phase.Issues {
				issueType := issue.Type
				if issueType == "" {
					issueType = "task"
				}
				fmt.Printf("     - %s [%s]\n", issue.Title, issueType)
			}
		}
		fmt.Println()
	}

	fmt.Printf("Spawn with: gt template spawn %s <epic-id>\n", templateName)
	return nil
}

func runTemplateSpawn(cmd *cobra.Command, args []string) error {
	// Handle old --template/--epic flag format
	var templateName, epicID string

	if len(args) >= 2 {
		templateName = args[0]
		epicID = args[1]
	} else {
		// Try flags (for deprecated gt spawn command)
		templateName, _ = cmd.Flags().GetString("template")
		epicID, _ = cmd.Flags().GetString("epic")

		if templateName == "" || epicID == "" {
			return fmt.Errorf("requires template name and epic ID\n\nUsage: gt template spawn <template-name> <epic-id>")
		}
	}

	searchPaths := getSearchPaths()
	path, err := epictemplate.FindTemplate(templateName, searchPaths)
	if err != nil {
		return err
	}

	tmpl, err := epictemplate.LoadTemplate(path)
	if err != nil {
		return err
	}

	// Verify epic exists
	epicDetails := getIssueDetails(epicID)
	if epicDetails == nil {
		return fmt.Errorf("epic '%s' not found", epicID)
	}

	// Dry run mode
	if templateSpawnDry {
		fmt.Printf("%s Would spawn template:\n", style.Dim.Render("[dry-run]"))
		fmt.Printf("  Template: %s\n", style.Bold.Render(tmpl.Name))
		fmt.Printf("  Epic:     %s (%s)\n", epicID, epicDetails.Title)
		fmt.Printf("  Phases:   %d\n", len(tmpl.Phases))
		fmt.Printf("  Issues:   %d would be created\n", tmpl.TotalIssues())

		fmt.Printf("\n  Issues to create:\n")
		for i, phase := range tmpl.Phases {
			fmt.Printf("    Phase %d (%s):\n", i+1, phase.Name)
			for _, issue := range phase.Issues {
				fmt.Printf("      - %s\n", issue.Title)
			}
		}
		return nil
	}

	// Find beads directory
	townBeads, err := getTownBeadsDir()
	if err != nil {
		return err
	}

	fmt.Printf("%s Spawning template: %s\n", style.Bold.Render(">>"), tmpl.Name)
	fmt.Printf("   Into epic: %s\n\n", epicID)

	// Track created issues by phase for dependency linking
	var prevPhaseIssues []string
	totalCreated := 0

	for i, phase := range tmpl.Phases {
		phaseNum := i + 1
		var currentPhaseIssues []string

		fmt.Printf("  Phase %d: %s\n", phaseNum, phase.Name)

		for _, issue := range phase.Issues {
			// Generate issue ID
			issueID := generateTemplateIssueID()

			issueType := issue.Type
			if issueType == "" {
				issueType = "task"
			}

			// Create the issue
			createArgs := []string{
				"create",
				"--id=" + issueID,
				"--type=" + issueType,
				"--title=" + issue.Title,
			}
			if issue.Description != "" {
				createArgs = append(createArgs, "--description="+issue.Description)
			}

			createCmd := exec.Command("bd", createArgs...)
			createCmd.Dir = townBeads
			var stderr bytes.Buffer
			createCmd.Stderr = &stderr

			if err := createCmd.Run(); err != nil {
				fmt.Printf("    %s Failed to create '%s': %v\n",
					style.Warning.Render("!"), issue.Title, err)
				continue
			}

			// Add as child of epic
			depArgs := []string{"dep", "add", epicID, issueID, "--type=parent"}
			depCmd := exec.Command("bd", depArgs...)
			depCmd.Dir = townBeads
			_ = depCmd.Run()

			// Add dependency on previous phase issues (if any)
			for _, prevID := range prevPhaseIssues {
				blockArgs := []string{"dep", "add", issueID, prevID}
				blockCmd := exec.Command("bd", blockArgs...)
				blockCmd.Dir = townBeads
				_ = blockCmd.Run()
			}

			currentPhaseIssues = append(currentPhaseIssues, issueID)
			totalCreated++

			fmt.Printf("    %s %s (%s)\n", style.Success.Render("+"), issue.Title, issueID)
		}

		prevPhaseIssues = currentPhaseIssues
		fmt.Println()
	}

	fmt.Printf("%s Created %d issues under epic %s\n", style.Bold.Render("Done!"), totalCreated, epicID)
	fmt.Printf("\nView epic: bd show %s\n", epicID)

	return nil
}

// generateTemplateIssueID generates a unique issue ID for template-spawned issues.
func generateTemplateIssueID() string {
	b := make([]byte, 4)
	rand.Read(b)
	suffix := strings.ToLower(base32.StdEncoding.EncodeToString(b)[:6])
	return fmt.Sprintf("gt-tmpl-%s", suffix)
}
