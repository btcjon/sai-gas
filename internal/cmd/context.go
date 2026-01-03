package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/style"
)

var (
	contextUsage bool
	contextJSON  bool
)

var contextCmd = &cobra.Command{
	Use:     "context",
	GroupID: GroupDiag,
	Short:   "Check Claude context and quota usage",
	Long: `Check Claude API context and quota usage.

The --usage flag shows current quota utilization:
  - 5-hour utilization percentage
  - 7-day utilization percentage
  - Pace-adjusted effective threshold
  - Status indicator (OK or HIGH)

Exit code is 1 if any utilization exceeds 80% (useful for scripting).

Examples:
  gt context --usage        # Show quota utilization
  gt context --usage --json # Output as JSON for automation`,
	RunE: runContext,
}

func init() {
	contextCmd.Flags().BoolVar(&contextUsage, "usage", false, "Show quota utilization")
	contextCmd.Flags().BoolVar(&contextJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(contextCmd)
}

// usageData represents the JSON response from claude-usage.sh
type usageData struct {
	FiveHour  int `json:"five_hour"`
	SevenDay  int `json:"seven_day"`
	PaceDelta int `json:"pace_delta"`
	Effective int `json:"effective"`
}

func runContext(cmd *cobra.Command, args []string) error {
	if !contextUsage {
		return cmd.Help()
	}

	// Call the usage script
	scriptPath := os.ExpandEnv("$HOME/.claude/scripts/usage-fetchers/claude-usage.sh")
	out, err := exec.Command(scriptPath, "--json").Output()
	if err != nil {
		return fmt.Errorf("failed to fetch usage: %w", err)
	}

	var data usageData
	if err := json.Unmarshal(out, &data); err != nil {
		return fmt.Errorf("failed to parse usage data: %w", err)
	}

	// Determine if high usage (>80%)
	highUsage := data.FiveHour > 80 || data.SevenDay > 80

	if contextJSON {
		result := map[string]interface{}{
			"five_hour":  data.FiveHour,
			"seven_day":  data.SevenDay,
			"pace_delta": data.PaceDelta,
			"effective":  data.Effective,
			"high_usage": highUsage,
		}
		jsonOut, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonOut))
		if highUsage {
			return NewSilentExit(1)
		}
		return nil
	}

	// Human-friendly output
	fmt.Println(style.Bold.Render("Context Usage:"))
	fmt.Printf("  5-hour:  %3d%%  %s\n", data.FiveHour, renderBar(data.FiveHour))
	fmt.Printf("  7-day:   %3d%%  %s\n", data.SevenDay, renderBar(data.SevenDay))

	// Show pace info
	var paceIndicator string
	if data.PaceDelta < 0 {
		paceIndicator = fmt.Sprintf("%d%% behind pace", -data.PaceDelta)
	} else if data.PaceDelta > 0 {
		paceIndicator = fmt.Sprintf("%d%% ahead of pace", data.PaceDelta)
	} else {
		paceIndicator = "on pace"
	}
	fmt.Printf("  %s %s\n", style.Dim.Render("Pace:"), paceIndicator)

	// Status line
	if highUsage {
		fmt.Printf("  %s  %s\n", style.Warning.Render("Status:"), style.Warning.Render("HIGH - consider handoff"))
		return NewSilentExit(1)
	}
	fmt.Printf("  %s  %s\n", style.Dim.Render("Status:"), style.Success.Render("OK (below 80% threshold)"))
	return nil
}

// renderBar creates a simple progress bar
func renderBar(pct int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	// 10-character bar
	filled := pct / 10
	empty := 10 - filled

	bar := strings.Repeat("#", filled) + strings.Repeat("-", empty)

	// Color based on threshold
	if pct > 80 {
		return style.Warning.Render(bar)
	}
	if pct > 50 {
		return style.Info.Render(bar)
	}
	return style.Success.Render(bar)
}
