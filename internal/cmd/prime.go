package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/workspace"
)

// Role represents a detected agent role.
type Role string

const (
	RoleMayor    Role = "mayor"
	RoleWitness  Role = "witness"
	RoleRefinery Role = "refinery"
	RolePolecat  Role = "polecat"
	RoleUnknown  Role = "unknown"
)

var primeCmd = &cobra.Command{
	Use:   "prime",
	Short: "Output role context for current directory",
	Long: `Detect the agent role from the current directory and output context.

Role detection:
  - Town root or mayor/rig/ → Mayor context
  - <rig>/witness/rig/ → Witness context
  - <rig>/refinery/rig/ → Refinery context
  - <rig>/polecats/<name>/ → Polecat context

This command is typically used in shell prompts or agent initialization.`,
	RunE: runPrime,
}

func init() {
	rootCmd.AddCommand(primeCmd)
}

// RoleContext contains information about the detected role.
type RoleContext struct {
	Role     Role   `json:"role"`
	Rig      string `json:"rig,omitempty"`
	Polecat  string `json:"polecat,omitempty"`
	TownRoot string `json:"town_root"`
	WorkDir  string `json:"work_dir"`
}

func runPrime(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	// Find town root
	townRoot, err := workspace.FindFromCwd()
	if err != nil {
		return fmt.Errorf("finding workspace: %w", err)
	}
	if townRoot == "" {
		return fmt.Errorf("not in a Gas Town workspace")
	}

	// Detect role
	ctx := detectRole(cwd, townRoot)

	// Output context
	return outputPrimeContext(ctx)
}

func detectRole(cwd, townRoot string) RoleContext {
	ctx := RoleContext{
		Role:     RoleUnknown,
		TownRoot: townRoot,
		WorkDir:  cwd,
	}

	// Get relative path from town root
	relPath, err := filepath.Rel(townRoot, cwd)
	if err != nil {
		return ctx
	}

	// Normalize and split path
	relPath = filepath.ToSlash(relPath)
	parts := strings.Split(relPath, "/")

	// Check for mayor role
	// At town root, or in mayor/ or mayor/rig/
	if relPath == "." || relPath == "" {
		ctx.Role = RoleMayor
		return ctx
	}
	if len(parts) >= 1 && parts[0] == "mayor" {
		ctx.Role = RoleMayor
		return ctx
	}

	// At this point, first part should be a rig name
	if len(parts) < 1 {
		return ctx
	}
	rigName := parts[0]
	ctx.Rig = rigName

	// Check for witness: <rig>/witness/rig/
	if len(parts) >= 2 && parts[1] == "witness" {
		ctx.Role = RoleWitness
		return ctx
	}

	// Check for refinery: <rig>/refinery/rig/
	if len(parts) >= 2 && parts[1] == "refinery" {
		ctx.Role = RoleRefinery
		return ctx
	}

	// Check for polecat: <rig>/polecats/<name>/
	if len(parts) >= 3 && parts[1] == "polecats" {
		ctx.Role = RolePolecat
		ctx.Polecat = parts[2]
		return ctx
	}

	// Default: could be rig root - treat as unknown
	return ctx
}

func outputPrimeContext(ctx RoleContext) error {
	switch ctx.Role {
	case RoleMayor:
		outputMayorContext(ctx)
	case RoleWitness:
		outputWitnessContext(ctx)
	case RoleRefinery:
		outputRefineryContext(ctx)
	case RolePolecat:
		outputPolecatContext(ctx)
	default:
		outputUnknownContext(ctx)
	}
	return nil
}

func outputMayorContext(ctx RoleContext) {
	fmt.Printf("%s\n\n", style.Bold.Render("# Mayor Context"))
	fmt.Println("You are the **Mayor** - the global coordinator of Gas Town.")
	fmt.Println()
	fmt.Println("## Responsibilities")
	fmt.Println("- Coordinate work across all rigs")
	fmt.Println("- Delegate to Refineries, not directly to polecats")
	fmt.Println("- Monitor overall system health")
	fmt.Println()
	fmt.Println("## Key Commands")
	fmt.Println("- `gt status` - Show overall town status")
	fmt.Println("- `gt rigs` - List all rigs")
	fmt.Println("- `bd ready` - Issues ready to work")
	fmt.Println()
	fmt.Printf("Town root: %s\n", style.Dim.Render(ctx.TownRoot))
}

func outputWitnessContext(ctx RoleContext) {
	fmt.Printf("%s\n\n", style.Bold.Render("# Witness Context"))
	fmt.Printf("You are the **Witness** for rig: %s\n\n", style.Bold.Render(ctx.Rig))
	fmt.Println("## Responsibilities")
	fmt.Println("- Monitor polecat health via heartbeat")
	fmt.Println("- Spawn ephemeral agents for stuck polecats")
	fmt.Println("- Report rig status to Mayor")
	fmt.Println()
	fmt.Println("## Key Commands")
	fmt.Println("- `gt witness status` - Show witness status")
	fmt.Println("- `gt polecats` - List polecats in this rig")
	fmt.Println()
	fmt.Printf("Rig: %s\n", style.Dim.Render(ctx.Rig))
}

func outputRefineryContext(ctx RoleContext) {
	fmt.Printf("%s\n\n", style.Bold.Render("# Refinery Context"))
	fmt.Printf("You are the **Refinery** for rig: %s\n\n", style.Bold.Render(ctx.Rig))
	fmt.Println("## Responsibilities")
	fmt.Println("- Process the merge queue for this rig")
	fmt.Println("- Merge polecat work to integration branch")
	fmt.Println("- Resolve merge conflicts")
	fmt.Println("- Land completed swarms to main")
	fmt.Println()
	fmt.Println("## Key Commands")
	fmt.Println("- `gt merge queue` - Show pending merges")
	fmt.Println("- `gt merge next` - Process next merge")
	fmt.Println()
	fmt.Printf("Rig: %s\n", style.Dim.Render(ctx.Rig))
}

func outputPolecatContext(ctx RoleContext) {
	fmt.Printf("%s\n\n", style.Bold.Render("# Polecat Context"))
	fmt.Printf("You are polecat **%s** in rig: %s\n\n",
		style.Bold.Render(ctx.Polecat), style.Bold.Render(ctx.Rig))
	fmt.Println("## Responsibilities")
	fmt.Println("- Work on assigned issues")
	fmt.Println("- Commit work to your branch")
	fmt.Println("- Signal completion for merge queue")
	fmt.Println()
	fmt.Println("## Key Commands")
	fmt.Println("- `bd show <issue>` - View your assigned issue")
	fmt.Println("- `bd close <issue>` - Mark issue complete")
	fmt.Println("- `gt done` - Signal work ready for merge")
	fmt.Println()
	fmt.Printf("Polecat: %s | Rig: %s\n",
		style.Dim.Render(ctx.Polecat), style.Dim.Render(ctx.Rig))
}

func outputUnknownContext(ctx RoleContext) {
	fmt.Printf("%s\n\n", style.Bold.Render("# Gas Town Context"))
	fmt.Println("Could not determine specific role from current directory.")
	fmt.Println()
	if ctx.Rig != "" {
		fmt.Printf("You appear to be in rig: %s\n\n", style.Bold.Render(ctx.Rig))
	}
	fmt.Println("Navigate to a specific agent directory:")
	fmt.Println("- `<rig>/polecats/<name>/` - Polecat role")
	fmt.Println("- `<rig>/witness/rig/` - Witness role")
	fmt.Println("- `<rig>/refinery/rig/` - Refinery role")
	fmt.Println("- Town root or `mayor/` - Mayor role")
	fmt.Println()
	fmt.Printf("Town root: %s\n", style.Dim.Render(ctx.TownRoot))
}
