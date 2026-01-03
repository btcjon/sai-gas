package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

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
	FiveHour       int    `json:"five_hour"`
	FiveHourLeft   int    `json:"five_hour_left"`
	SevenDay       int    `json:"seven_day"`
	SevenDayLeft   int    `json:"seven_day_left"`
	PaceDelta      int    `json:"pace_delta"`
	Effective      int    `json:"effective"`
	FiveHourResets string `json:"five_hour_resets"`
	SevenDayResets string `json:"seven_day_resets"`
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

	// Determine if low remaining (<20% left = high usage)
	lowRemaining := data.FiveHourLeft < 20 || data.SevenDayLeft < 20

	if contextJSON {
		result := map[string]interface{}{
			"five_hour_left":   data.FiveHourLeft,
			"seven_day_left":   data.SevenDayLeft,
			"five_hour_used":   data.FiveHour,
			"seven_day_used":   data.SevenDay,
			"pace_delta":       data.PaceDelta,
			"effective":        data.Effective,
			"low_remaining":    lowRemaining,
			"five_hour_resets": data.FiveHourResets,
			"seven_day_resets": data.SevenDayResets,
		}
		jsonOut, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonOut))
		if lowRemaining {
			return NewSilentExit(1)
		}
		return nil
	}

	// Human-friendly output - show "% left" like CC UI
	fmt.Println(style.Bold.Render("Quota Remaining:"))
	fmt.Printf("  Session: %3d%% left  %s  %s\n", data.FiveHourLeft, renderBarLeft(data.FiveHourLeft), formatResetTime(data.FiveHourResets))
	fmt.Printf("  Weekly:  %3d%% left  %s  %s\n", data.SevenDayLeft, renderBarLeft(data.SevenDayLeft), formatResetTime(data.SevenDayResets))

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
	if lowRemaining {
		fmt.Printf("  %s  %s\n", style.Warning.Render("Status:"), style.Warning.Render("LOW - consider handoff"))
		return NewSilentExit(1)
	}
	fmt.Printf("  %s  %s\n", style.Dim.Render("Status:"), style.Success.Render("OK (>20% remaining)"))
	return nil
}

// renderBarLeft creates a progress bar for "% left" (higher = better)
func renderBarLeft(pct int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	// 10-character bar
	filled := pct / 10
	empty := 10 - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	// Color based on remaining (lower = worse)
	if pct <= 20 {
		return style.Warning.Render(bar)
	}
	if pct <= 50 {
		return style.Info.Render(bar)
	}
	return style.Success.Render(bar)
}

// formatResetTime converts ISO timestamp to human-readable duration
func formatResetTime(isoTime string) string {
	if isoTime == "" {
		return ""
	}

	t, err := time.Parse(time.RFC3339, isoTime)
	if err != nil {
		return ""
	}

	dur := time.Until(t)
	if dur < 0 {
		return style.Dim.Render("(resets now)")
	}

	hours := int(dur.Hours())
	minutes := int(dur.Minutes()) % 60

	if hours >= 24 {
		days := hours / 24
		hours = hours % 24
		return style.Dim.Render(fmt.Sprintf("(resets in %dd %dh)", days, hours))
	}
	if hours > 0 {
		return style.Dim.Render(fmt.Sprintf("(resets in %dh %dm)", hours, minutes))
	}
	return style.Dim.Render(fmt.Sprintf("(resets in %dm)", minutes))
}
