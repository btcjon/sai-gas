package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/hooks"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/workspace"
)

var (
	hooksJSON    bool
	hooksVerbose bool
)

var hooksCmd = &cobra.Command{
	Use:     "hooks",
	GroupID: GroupConfig,
	Short:   "Manage Gas Town and Claude Code hooks",
	Long: `Manage hooks for both Gas Town and Claude Code.

Gas Town hooks (.gastown/hooks.json):
  gt hooks list [event]   List registered Gas Town hooks
  gt hooks fire <event>   Manually fire an event for testing
  gt hooks test           Validate hook configuration

Claude Code hooks (.claude/settings.json):
  gt hooks cc             List Claude Code hooks in workspace

Examples:
  gt hooks list                    # List all Gas Town hooks
  gt hooks list pre-shutdown       # List hooks for specific event
  gt hooks fire pre-shutdown       # Fire an event (for testing)
  gt hooks test                    # Validate configuration
  gt hooks cc                      # List Claude Code hooks`,
	RunE: requireSubcommand,
}

// Subcommand for listing Gas Town hooks
var hooksListCmd = &cobra.Command{
	Use:   "list [event]",
	Short: "List registered Gas Town hooks",
	Long: `List hooks registered in .gastown/hooks.json.

Optionally filter by event type.

Event types:
  pre-session-start   - Before session starts (can block)
  post-session-start  - After session starts
  pre-shutdown        - Before shutdown (can block)
  post-shutdown       - After shutdown
  mail-received       - When mail is received
  work-assigned       - When work is assigned

Examples:
  gt hooks list                    # List all hooks
  gt hooks list pre-shutdown       # List hooks for event`,
	Args: cobra.MaximumNArgs(1),
	RunE: runHooksList,
}

// Subcommand for firing a hook event (testing)
var hooksFireCmd = &cobra.Command{
	Use:   "fire <event>",
	Short: "Manually fire a hook event for testing",
	Long: `Fire a hook event manually to test hook scripts.

This is useful for debugging hooks without waiting for the actual event.
Hook output and results are displayed.

Examples:
  gt hooks fire pre-shutdown
  gt hooks fire mail-received`,
	Args: cobra.ExactArgs(1),
	RunE: runHooksFire,
}

// Subcommand for testing/validating hook config
var hooksTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Validate hook configuration",
	Long: `Validate the hooks configuration file.

Checks that:
- The config file is valid JSON
- Event types are recognized
- Hook definitions have required fields
- Timeout values are valid durations

Example:
  gt hooks test`,
	RunE: runHooksTest,
}

// Subcommand for Claude Code hooks (backwards compatibility)
var hooksCCCmd = &cobra.Command{
	Use:   "cc",
	Short: "List Claude Code hooks in workspace",
	Long: `List all Claude Code hooks configured in the workspace.

Scans for .claude/settings.json files and displays hooks by type.

Hook types:
  SessionStart     - Runs when Claude session starts
  PreCompact       - Runs before context compaction
  UserPromptSubmit - Runs before user prompt is submitted
  PreToolUse       - Runs before tool execution
  PostToolUse      - Runs after tool execution
  Stop             - Runs when Claude session stops

Examples:
  gt hooks cc              # List all hooks in workspace
  gt hooks cc --verbose    # Show hook commands
  gt hooks cc --json       # Output as JSON`,
	RunE: runHooksCC,
}

func init() {
	rootCmd.AddCommand(hooksCmd)
	hooksCmd.AddCommand(hooksListCmd)
	hooksCmd.AddCommand(hooksFireCmd)
	hooksCmd.AddCommand(hooksTestCmd)
	hooksCmd.AddCommand(hooksCCCmd)

	// Add flags to cc subcommand (for Claude Code hooks)
	hooksCCCmd.Flags().BoolVar(&hooksJSON, "json", false, "Output as JSON")
	hooksCCCmd.Flags().BoolVarP(&hooksVerbose, "verbose", "v", false, "Show hook commands")

	// Add JSON flag to list subcommand
	hooksListCmd.Flags().BoolVar(&hooksJSON, "json", false, "Output as JSON")
}

// ClaudeSettings represents the Claude Code settings.json structure.
type ClaudeSettings struct {
	EnabledPlugins map[string]bool                  `json:"enabledPlugins,omitempty"`
	Hooks          map[string][]ClaudeHookMatcher   `json:"hooks,omitempty"`
}

// ClaudeHookMatcher represents a hook matcher entry.
type ClaudeHookMatcher struct {
	Matcher string       `json:"matcher"`
	Hooks   []ClaudeHook `json:"hooks"`
}

// ClaudeHook represents an individual hook.
type ClaudeHook struct {
	Type    string `json:"type"`
	Command string `json:"command,omitempty"`
}

// HookInfo contains information about a discovered hook.
type HookInfo struct {
	Type     string   `json:"type"`     // Hook type (SessionStart, etc.)
	Location string   `json:"location"` // Path to the settings file
	Agent    string   `json:"agent"`    // Agent that owns this hook (e.g., "polecat/nux")
	Matcher  string   `json:"matcher"`  // Pattern matcher (empty = all)
	Commands []string `json:"commands"` // Hook commands
	Status   string   `json:"status"`   // "active" or "disabled"
}

// HooksOutput is the JSON output structure.
type HooksOutput struct {
	TownRoot string     `json:"town_root"`
	Hooks    []HookInfo `json:"hooks"`
	Count    int        `json:"count"`
}

// runHooksList lists Gas Town hooks from .gastown/hooks.json
func runHooksList(cmd *cobra.Command, args []string) error {
	townRoot, err := workspace.FindFromCwd()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	config, err := hooks.LoadConfig(townRoot)
	if err != nil {
		return fmt.Errorf("loading hooks config: %w", err)
	}

	// Filter by event if specified
	var filterEvent hooks.Event
	if len(args) > 0 {
		filterEvent = hooks.Event(args[0])
		// Validate event type
		valid := false
		for _, e := range hooks.AllEvents() {
			if e == filterEvent {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("unknown event type %q", args[0])
		}
	}

	if hooksJSON {
		return outputGTHooksJSON(townRoot, config, filterEvent)
	}

	return outputGTHooksHuman(townRoot, config, filterEvent)
}

// runHooksFire manually fires a hook event for testing
func runHooksFire(cmd *cobra.Command, args []string) error {
	townRoot, err := workspace.FindFromCwd()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	event := hooks.Event(args[0])

	// Validate event type
	valid := false
	for _, e := range hooks.AllEvents() {
		if e == event {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("unknown event type %q", args[0])
	}

	runner, err := hooks.NewRunnerFromTownRoot(townRoot)
	if err != nil {
		return fmt.Errorf("creating hook runner: %w", err)
	}

	if !runner.HasHooks(event) {
		fmt.Printf("%s No hooks registered for event: %s\n", style.Dim.Render("!"), event)
		return nil
	}

	ctx := hooks.NewEventContext(event, townRoot)

	fmt.Printf("%s Firing event: %s\n\n", style.Bold.Render(">"), event)

	results := runner.Fire(ctx)

	for i, result := range results {
		icon := style.Success.Render("OK")
		if !result.Success {
			icon = style.Error.Render("FAIL")
		}
		if result.Block {
			icon = style.Warning.Render("BLOCK")
		}

		fmt.Printf("%s Hook %d: %s\n", icon, i+1, result.Hook.Cmd)
		fmt.Printf("  Elapsed: %s\n", result.Elapsed)

		if result.Message != "" {
			fmt.Printf("  Message: %s\n", result.Message)
		}
		if result.Error != "" {
			fmt.Printf("  Error: %s\n", style.Error.Render(result.Error))
		}
		if result.Output != "" {
			fmt.Printf("  Output:\n")
			for _, line := range strings.Split(result.Output, "\n") {
				fmt.Printf("    %s\n", line)
			}
		}
		fmt.Println()
	}

	// Summary
	if hooks.AnyBlocked(results) {
		fmt.Printf("%s Event was blocked by a hook\n", style.Warning.Render("!"))
	} else if hooks.AllSucceeded(results) {
		fmt.Printf("%s All hooks completed successfully\n", style.Success.Render("OK"))
	} else {
		fmt.Printf("%s Some hooks failed\n", style.Error.Render("!"))
	}

	return nil
}

// runHooksTest validates the hooks configuration
func runHooksTest(cmd *cobra.Command, args []string) error {
	townRoot, err := workspace.FindFromCwd()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	configPath := filepath.Join(townRoot, hooks.ConfigFile)

	if !hooks.ConfigExists(townRoot) {
		fmt.Printf("%s No hooks configuration found at: %s\n", style.Dim.Render("!"), configPath)
		fmt.Println("\nTo create one, create a file like:")
		example, _ := json.MarshalIndent(hooks.ExampleConfig(), "", "  ")
		fmt.Println(string(example))
		return nil
	}

	config, err := hooks.LoadConfig(townRoot)
	if err != nil {
		fmt.Printf("%s Failed to load config: %v\n", style.Error.Render("X"), err)
		return err
	}

	if err := hooks.ValidateConfig(config); err != nil {
		fmt.Printf("%s Configuration validation failed: %v\n", style.Error.Render("X"), err)
		return err
	}

	// Count hooks
	total := 0
	for _, hks := range config.Hooks {
		total += len(hks)
	}

	fmt.Printf("%s Configuration valid!\n", style.Success.Render("OK"))
	fmt.Printf("  File: %s\n", configPath)
	fmt.Printf("  Events with hooks: %d\n", len(config.Hooks))
	fmt.Printf("  Total hooks: %d\n", total)

	return nil
}

// runHooksCC lists Claude Code hooks (renamed from old runHooks)
func runHooksCC(cmd *cobra.Command, args []string) error {
	townRoot, err := workspace.FindFromCwd()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	// Find all .claude/settings.json files
	ccHooks, err := discoverHooks(townRoot)
	if err != nil {
		return fmt.Errorf("discovering hooks: %w", err)
	}

	if hooksJSON {
		return outputHooksJSON(townRoot, ccHooks)
	}

	return outputHooksHuman(townRoot, ccHooks)
}

// outputGTHooksJSON outputs Gas Town hooks as JSON
func outputGTHooksJSON(townRoot string, config *hooks.HookConfig, filterEvent hooks.Event) error {
	type GTHookOutput struct {
		TownRoot string                    `json:"town_root"`
		Hooks    map[hooks.Event][]hooks.Hook `json:"hooks"`
	}

	output := GTHookOutput{
		TownRoot: townRoot,
		Hooks:    config.Hooks,
	}

	if filterEvent != "" {
		output.Hooks = map[hooks.Event][]hooks.Hook{
			filterEvent: config.GetHooks(filterEvent),
		}
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}

// outputGTHooksHuman outputs Gas Town hooks in human-readable format
func outputGTHooksHuman(townRoot string, config *hooks.HookConfig, filterEvent hooks.Event) error {
	if len(config.Hooks) == 0 {
		fmt.Println(style.Dim.Render("No Gas Town hooks configured"))
		fmt.Printf("\nConfig file: %s\n", filepath.Join(townRoot, hooks.ConfigFile))
		return nil
	}

	fmt.Printf("\n%s Gas Town Hooks\n", style.Bold.Render(">"))
	fmt.Printf("Config: %s\n\n", style.Dim.Render(filepath.Join(townRoot, hooks.ConfigFile)))

	// Get events to display
	var events []hooks.Event
	if filterEvent != "" {
		events = []hooks.Event{filterEvent}
	} else {
		events = hooks.AllEvents()
	}

	// Sort events alphabetically for consistent output
	sort.Slice(events, func(i, j int) bool {
		return string(events[i]) < string(events[j])
	})

	total := 0
	for _, event := range events {
		hks := config.GetHooks(event)
		if len(hks) == 0 {
			continue
		}

		blockable := ""
		if event.IsPreEvent() {
			blockable = style.Dim.Render(" (can block)")
		}

		fmt.Printf("%s %s%s\n", style.Bold.Render(">"), event, blockable)

		for i, h := range hks {
			total++
			timeout := ""
			if h.Timeout != "" {
				timeout = style.Dim.Render(fmt.Sprintf(" [timeout: %s]", h.Timeout))
			}
			fmt.Printf("  %d. %s%s\n", i+1, h.Cmd, timeout)
		}
		fmt.Println()
	}

	if total == 0 && filterEvent != "" {
		fmt.Printf("%s No hooks registered for event: %s\n", style.Dim.Render("!"), filterEvent)
	} else {
		fmt.Printf("%s %d hook(s) total\n", style.Dim.Render("Total:"), total)
	}

	return nil
}

// discoverHooks finds all Claude Code hooks in the workspace.
func discoverHooks(townRoot string) ([]HookInfo, error) {
	var hooks []HookInfo

	// Scan known locations for .claude/settings.json
	locations := []struct {
		path  string
		agent string
	}{
		{filepath.Join(townRoot, "mayor", ".claude", "settings.json"), "mayor/"},
		{filepath.Join(townRoot, ".claude", "settings.json"), "town-root"},
	}

	// Scan rigs
	entries, err := os.ReadDir(townRoot)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "mayor" || entry.Name() == ".beads" || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		rigName := entry.Name()
		rigPath := filepath.Join(townRoot, rigName)

		// Rig-level hooks
		locations = append(locations, struct {
			path  string
			agent string
		}{filepath.Join(rigPath, ".claude", "settings.json"), fmt.Sprintf("%s/rig", rigName)})

		// Polecats
		polecatsDir := filepath.Join(rigPath, "polecats")
		if polecats, err := os.ReadDir(polecatsDir); err == nil {
			for _, p := range polecats {
				if p.IsDir() {
					locations = append(locations, struct {
						path  string
						agent string
					}{filepath.Join(polecatsDir, p.Name(), ".claude", "settings.json"), fmt.Sprintf("%s/%s", rigName, p.Name())})
				}
			}
		}

		// Crew members
		crewDir := filepath.Join(rigPath, "crew")
		if crew, err := os.ReadDir(crewDir); err == nil {
			for _, c := range crew {
				if c.IsDir() {
					locations = append(locations, struct {
						path  string
						agent string
					}{filepath.Join(crewDir, c.Name(), ".claude", "settings.json"), fmt.Sprintf("%s/crew/%s", rigName, c.Name())})
				}
			}
		}

		// Witness
		witnessPath := filepath.Join(rigPath, "witness", ".claude", "settings.json")
		locations = append(locations, struct {
			path  string
			agent string
		}{witnessPath, fmt.Sprintf("%s/witness", rigName)})

		// Refinery
		refineryPath := filepath.Join(rigPath, "refinery", ".claude", "settings.json")
		locations = append(locations, struct {
			path  string
			agent string
		}{refineryPath, fmt.Sprintf("%s/refinery", rigName)})
	}

	// Process each location
	for _, loc := range locations {
		if _, err := os.Stat(loc.path); os.IsNotExist(err) {
			continue
		}

		found, err := parseHooksFile(loc.path, loc.agent)
		if err != nil {
			// Skip files that can't be parsed
			continue
		}
		hooks = append(hooks, found...)
	}

	return hooks, nil
}

// parseHooksFile parses a .claude/settings.json file and extracts hooks.
func parseHooksFile(path, agent string) ([]HookInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var settings ClaudeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	var hooks []HookInfo

	for hookType, matchers := range settings.Hooks {
		for _, matcher := range matchers {
			var commands []string
			for _, h := range matcher.Hooks {
				if h.Command != "" {
					commands = append(commands, h.Command)
				}
			}

			if len(commands) > 0 {
				hooks = append(hooks, HookInfo{
					Type:     hookType,
					Location: path,
					Agent:    agent,
					Matcher:  matcher.Matcher,
					Commands: commands,
					Status:   "active",
				})
			}
		}
	}

	return hooks, nil
}

func outputHooksJSON(townRoot string, hooks []HookInfo) error {
	output := HooksOutput{
		TownRoot: townRoot,
		Hooks:    hooks,
		Count:    len(hooks),
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}

func outputHooksHuman(townRoot string, hooks []HookInfo) error {
	if len(hooks) == 0 {
		fmt.Println(style.Dim.Render("No Claude Code hooks found in workspace"))
		return nil
	}

	fmt.Printf("\n%s Claude Code Hooks\n", style.Bold.Render("🪝"))
	fmt.Printf("Town root: %s\n\n", style.Dim.Render(townRoot))

	// Group by hook type
	byType := make(map[string][]HookInfo)
	typeOrder := []string{"SessionStart", "PreCompact", "UserPromptSubmit", "PreToolUse", "PostToolUse", "Stop"}

	for _, h := range hooks {
		byType[h.Type] = append(byType[h.Type], h)
	}

	// Add any types not in the predefined order
	for t := range byType {
		found := false
		for _, o := range typeOrder {
			if t == o {
				found = true
				break
			}
		}
		if !found {
			typeOrder = append(typeOrder, t)
		}
	}

	for _, hookType := range typeOrder {
		typeHooks := byType[hookType]
		if len(typeHooks) == 0 {
			continue
		}

		fmt.Printf("%s %s\n", style.Bold.Render("▸"), hookType)

		for _, h := range typeHooks {
			statusIcon := "●"
			if h.Status != "active" {
				statusIcon = "○"
			}

			matcherStr := ""
			if h.Matcher != "" {
				matcherStr = fmt.Sprintf(" [%s]", h.Matcher)
			}

			fmt.Printf("  %s %-25s%s\n", statusIcon, h.Agent, style.Dim.Render(matcherStr))

			if hooksVerbose {
				for _, cmd := range h.Commands {
					fmt.Printf("    %s %s\n", style.Dim.Render("→"), cmd)
				}
			}
		}
		fmt.Println()
	}

	fmt.Printf("%s %d hooks found\n", style.Dim.Render("Total:"), len(hooks))

	return nil
}
