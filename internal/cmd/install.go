package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/config"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/templates"
	"github.com/steveyegge/gastown/internal/workspace"
)

var (
	installForce    bool
	installName     string
	installNoBeads  bool
)

var installCmd = &cobra.Command{
	Use:   "install [path]",
	Short: "Create a new Gas Town harness (workspace)",
	Long: `Create a new Gas Town harness at the specified path.

A harness is the top-level directory where Gas Town is installed. It contains:
  - config/town.json     Town configuration
  - config/rigs.json     Registry of managed rigs
  - mayor/               Mayor agent home
  - .beads/redirect      (optional) Default beads location

If path is omitted, uses the current directory.

Examples:
  gt install ~/gt                    # Create harness at ~/gt
  gt install . --name my-workspace   # Initialize current dir
  gt install ~/gt --no-beads         # Skip .beads/redirect setup`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInstall,
}

func init() {
	installCmd.Flags().BoolVarP(&installForce, "force", "f", false, "Overwrite existing harness")
	installCmd.Flags().StringVarP(&installName, "name", "n", "", "Town name (defaults to directory name)")
	installCmd.Flags().BoolVar(&installNoBeads, "no-beads", false, "Skip .beads/redirect setup")
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	// Determine target path
	targetPath := "."
	if len(args) > 0 {
		targetPath = args[0]
	}

	// Expand ~ and resolve to absolute path
	if targetPath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting home directory: %w", err)
		}
		targetPath = filepath.Join(home, targetPath[1:])
	}

	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Determine town name
	townName := installName
	if townName == "" {
		townName = filepath.Base(absPath)
	}

	// Check if already a workspace
	if isWS, _ := workspace.IsWorkspace(absPath); isWS && !installForce {
		return fmt.Errorf("directory is already a Gas Town harness (use --force to reinitialize)")
	}

	// Check if inside an existing workspace
	if existingRoot, _ := workspace.Find(absPath); existingRoot != "" && existingRoot != absPath {
		fmt.Printf("%s Warning: Creating harness inside existing workspace at %s\n",
			style.Dim.Render("⚠"), existingRoot)
	}

	fmt.Printf("%s Creating Gas Town harness at %s\n\n",
		style.Bold.Render("🏭"), style.Dim.Render(absPath))

	// Create directory structure
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Create config directory
	configDir := filepath.Join(absPath, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	fmt.Printf("   ✓ Created config/\n")

	// Create town.json
	townConfig := &config.TownConfig{
		Type:      "town",
		Version:   config.CurrentTownVersion,
		Name:      townName,
		CreatedAt: time.Now(),
	}
	townPath := filepath.Join(configDir, "town.json")
	if err := config.SaveTownConfig(townPath, townConfig); err != nil {
		return fmt.Errorf("writing town.json: %w", err)
	}
	fmt.Printf("   ✓ Created config/town.json\n")

	// Create rigs.json
	rigsConfig := &config.RigsConfig{
		Version: config.CurrentRigsVersion,
		Rigs:    make(map[string]config.RigEntry),
	}
	rigsPath := filepath.Join(configDir, "rigs.json")
	if err := config.SaveRigsConfig(rigsPath, rigsConfig); err != nil {
		return fmt.Errorf("writing rigs.json: %w", err)
	}
	fmt.Printf("   ✓ Created config/rigs.json\n")

	// Create mayor directory
	mayorDir := filepath.Join(absPath, "mayor")
	if err := os.MkdirAll(mayorDir, 0755); err != nil {
		return fmt.Errorf("creating mayor directory: %w", err)
	}
	fmt.Printf("   ✓ Created mayor/\n")

	// Create mayor mail directory
	mailDir := filepath.Join(mayorDir, "mail")
	if err := os.MkdirAll(mailDir, 0755); err != nil {
		return fmt.Errorf("creating mail directory: %w", err)
	}

	// Create empty inbox
	inboxPath := filepath.Join(mailDir, "inbox.jsonl")
	if err := os.WriteFile(inboxPath, []byte{}, 0644); err != nil {
		return fmt.Errorf("creating inbox: %w", err)
	}
	fmt.Printf("   ✓ Created mayor/mail/inbox.jsonl\n")

	// Create mayor state.json
	mayorState := &config.AgentState{
		Role:       "mayor",
		LastActive: time.Now(),
	}
	statePath := filepath.Join(mayorDir, "state.json")
	if err := config.SaveAgentState(statePath, mayorState); err != nil {
		return fmt.Errorf("writing mayor state: %w", err)
	}
	fmt.Printf("   ✓ Created mayor/state.json\n")

	// Create mayor config.json (this is what distinguishes town-level mayor)
	mayorConfig := map[string]interface{}{
		"type":    "mayor",
		"version": 1,
	}
	mayorConfigPath := filepath.Join(mayorDir, "config.json")
	if err := writeJSON(mayorConfigPath, mayorConfig); err != nil {
		return fmt.Errorf("writing mayor config: %w", err)
	}
	fmt.Printf("   ✓ Created mayor/config.json\n")

	// Create Mayor CLAUDE.md from template
	if err := createMayorCLAUDEmd(mayorDir, absPath); err != nil {
		fmt.Printf("   %s Could not create CLAUDE.md: %v\n", style.Dim.Render("⚠"), err)
	} else {
		fmt.Printf("   ✓ Created mayor/CLAUDE.md\n")
	}

	// Create .beads directory with redirect (optional)
	if !installNoBeads {
		beadsDir := filepath.Join(absPath, ".beads")
		if err := os.MkdirAll(beadsDir, 0755); err != nil {
			fmt.Printf("   %s Could not create .beads/: %v\n", style.Dim.Render("⚠"), err)
		} else {
			// Create redirect file with placeholder
			redirectPath := filepath.Join(beadsDir, "redirect")
			redirectContent := "# Redirect to your main rig's beads\n# Example: gastown/.beads\n"
			if err := os.WriteFile(redirectPath, []byte(redirectContent), 0644); err != nil {
				fmt.Printf("   %s Could not create redirect: %v\n", style.Dim.Render("⚠"), err)
			} else {
				fmt.Printf("   ✓ Created .beads/redirect (configure for your main rig)\n")
			}
		}
	}

	fmt.Printf("\n%s Harness created successfully!\n", style.Bold.Render("✓"))
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Add a rig: %s\n", style.Dim.Render("gt rig add <name> <git-url>"))
	fmt.Printf("  2. Configure beads redirect: %s\n", style.Dim.Render("edit .beads/redirect"))
	fmt.Printf("  3. Start the Mayor: %s\n", style.Dim.Render("cd "+absPath+" && gt prime"))

	return nil
}

func createMayorCLAUDEmd(mayorDir, townRoot string) error {
	tmpl, err := templates.New()
	if err != nil {
		return err
	}

	data := templates.RoleData{
		Role:     "mayor",
		TownRoot: townRoot,
		WorkDir:  mayorDir,
	}

	content, err := tmpl.RenderRole("mayor", data)
	if err != nil {
		return err
	}

	claudePath := filepath.Join(mayorDir, "CLAUDE.md")
	return os.WriteFile(claudePath, []byte(content), 0644)
}

func writeJSON(path string, data interface{}) error {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}
