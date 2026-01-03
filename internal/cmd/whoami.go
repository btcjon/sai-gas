package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/config"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/workspace"
)

var (
	whoamiJSON bool
)

var whoamiCmd = &cobra.Command{
	Use:     "whoami",
	GroupID: GroupDiag,
	Short:   "Show current identity for mail commands",
	Long: `Show the identity that will be used for mail commands.

Identity is determined by:
1. GT_ROLE env var (if set) - indicates an agent session
2. No GT_ROLE - you are the overseer (human)

Use --identity flag with mail commands to override.

Examples:
  gt whoami                      # Show current identity
  gt mail inbox                  # Check inbox for current identity
  gt mail inbox --identity mayor/  # Check Mayor's inbox instead`,
	RunE: runWhoami,
}

func init() {
	whoamiCmd.Flags().BoolVar(&whoamiJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(whoamiCmd)
}

func runWhoami(cmd *cobra.Command, args []string) error {
	// Get current identity using same logic as mail commands
	identity := detectSender()
	gtRole := os.Getenv("GT_ROLE")

	if whoamiJSON {
		res := map[string]string{
			"identity": identity,
			"role":     gtRole,
		}
		if gtRole == "" && identity == "overseer" {
			res["role"] = "overseer"
		}
		data, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("%s %s\n", style.Bold.Render("Identity:"), identity)

	// Show how it was determined
	if gtRole != "" {
		fmt.Printf("%s GT_ROLE=%s\n", style.Dim.Render("Source:"), gtRole)

		// Show additional env vars if present
		if rig := os.Getenv("GT_RIG"); rig != "" {
			fmt.Printf("%s GT_RIG=%s\n", style.Dim.Render("       "), rig)
		}
		if polecat := os.Getenv("GT_POLECAT"); polecat != "" {
			fmt.Printf("%s GT_POLECAT=%s\n", style.Dim.Render("       "), polecat)
		}
		if crew := os.Getenv("GT_CREW"); crew != "" {
			fmt.Printf("%s GT_CREW=%s\n", style.Dim.Render("       "), crew)
		}
	} else {
		fmt.Printf("%s no GT_ROLE set (human at terminal)\n", style.Dim.Render("Source:"))

		// If overseer, show their configured identity
		if identity == "overseer" {
			townRoot, err := workspace.FindFromCwd()
			if err == nil && townRoot != "" {
				if overseerConfig, err := config.LoadOverseerConfig(config.OverseerConfigPath(townRoot)); err == nil {
					fmt.Printf("\n%s\n", style.Bold.Render("Overseer Identity:"))
					fmt.Printf("  Name:  %s\n", overseerConfig.Name)
					if overseerConfig.Email != "" {
						fmt.Printf("  Email: %s\n", overseerConfig.Email)
					}
					if overseerConfig.Username != "" {
						fmt.Printf("  User:  %s\n", overseerConfig.Username)
					}
					fmt.Printf("  %s %s\n", style.Dim.Render("(detected via"), style.Dim.Render(overseerConfig.Source+")"))
				}
			}
		}
	}

	return nil
}
