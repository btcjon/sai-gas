// Package cmd provides CLI commands for the gt tool.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/workspace"
)

var pluginCmd = &cobra.Command{
	Use:     "plugin",
	Aliases: []string{"plugins"},
	GroupID: GroupServices,
	Short:   "Manage plugins in rigs",
	Long: `Manage plugins (directories with config files) in rigs.

Plugins are subdirectories in the rig's plugins/ directory that contain
a plugin.yaml or similar configuration file.`,
	RunE: requireSubcommand,
}

var pluginsListCmd = &cobra.Command{
	Use:   "list <rig>",
	Short: "List all plugins in a rig",
	Long: `List all plugins in a rig.

Scans the <rig>/plugins/ directory and lists all plugin subdirectories
found. A plugin is a directory with a plugin.yaml, plugin.json, or
similar config file.

Example:
  gt plugins list greenplace`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginsList,
}

var pluginStatusCmd = &cobra.Command{
	Use:   "status <rig>/<name>",
	Short: "Show plugin status",
	Long: `Show detailed status for a plugin.

Displays:
  - Plugin name
  - Enabled/disabled status
  - Config file location
  - Last activity timestamp
  - Basic metadata from config

Example:
  gt plugin status greenplace/auth`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginStatus,
}

func init() {
	pluginCmd.AddCommand(pluginsListCmd)
	pluginCmd.AddCommand(pluginStatusCmd)
	rootCmd.AddCommand(pluginCmd)
}

// PluginInfo holds information about a plugin.
type PluginInfo struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	ConfigFile string   `json:"config_file"`
	Enabled   bool      `json:"enabled"`
	LastMod   time.Time `json:"last_modified"`
}

// findPluginsInRig scans the plugins directory for plugin subdirectories.
func findPluginsInRig(rigPath string) ([]PluginInfo, error) {
	pluginsDir := filepath.Join(rigPath, "plugins")

	// Check if plugins directory exists
	info, err := os.Stat(pluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []PluginInfo{}, nil // No plugins yet
		}
		return nil, fmt.Errorf("checking plugins directory: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("plugins path is not a directory: %s", pluginsDir)
	}

	// Read directory
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		return nil, fmt.Errorf("reading plugins directory: %w", err)
	}

	var plugins []PluginInfo

	for _, entry := range entries {
		// Only consider directories (plugins are directories)
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		pluginPath := filepath.Join(pluginsDir, name)

		// Check for config files
		configFile := findConfigFile(pluginPath)
		if configFile == "" {
			// No config file found, skip this directory
			continue
		}

		// Get last modified time
		stat, err := os.Stat(configFile)
		lastMod := time.Time{}
		if err == nil {
			lastMod = stat.ModTime()
		}

		// Check if enabled (non-hidden directory = enabled)
		enabled := !isHiddenPlugin(name)

		plugins = append(plugins, PluginInfo{
			Name:       name,
			Path:       pluginPath,
			ConfigFile: configFile,
			Enabled:    enabled,
			LastMod:    lastMod,
		})
	}

	// Sort by name
	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Name < plugins[j].Name
	})

	return plugins, nil
}

// findConfigFile looks for a plugin config file in the given directory.
// Supports: plugin.yaml, plugin.json, plugin.yml
func findConfigFile(pluginDir string) string {
	configNames := []string{
		"plugin.yaml",
		"plugin.yml",
		"plugin.json",
		"config.yaml",
		"config.yml",
		"config.json",
	}

	for _, name := range configNames {
		path := filepath.Join(pluginDir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// isHiddenPlugin checks if a plugin name indicates it's disabled.
// Disabled plugins start with a dot or have "_disabled" suffix.
func isHiddenPlugin(name string) bool {
	return name[0] == '.' || len(name) > 9 && name[len(name)-9:] == "_disabled"
}

func runPluginsList(cmd *cobra.Command, args []string) error {
	rigName := args[0]

	// Find workspace
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	// Get rig path
	rigPath := filepath.Join(townRoot, rigName)

	// Check if rig exists
	if _, err := os.Stat(rigPath); err != nil {
		return fmt.Errorf("rig not found: %s", rigName)
	}

	// Find plugins
	plugins, err := findPluginsInRig(rigPath)
	if err != nil {
		return fmt.Errorf("scanning plugins: %w", err)
	}

	// Display results
	if len(plugins) == 0 {
		fmt.Printf("No plugins found in rig %s\n", style.Bold.Render(rigName))
		return nil
	}

	fmt.Printf("%s\n\n", style.Bold.Render(fmt.Sprintf("Plugins in %s", rigName)))

	for _, p := range plugins {
		// Status indicator
		statusStr := "enabled"
		statusStyle := style.Success
		if !p.Enabled {
			statusStr = "disabled"
			statusStyle = style.Dim
		}

		// Get relative path for display
		relPath, _ := filepath.Rel(rigPath, p.ConfigFile)

		fmt.Printf("  %s %s (%s)\n",
			style.Bold.Render(p.Name),
			statusStyle.Render(fmt.Sprintf("[%s]", statusStr)),
			style.Dim.Render(relPath),
		)

		// Show last modified if not too long ago
		if !p.LastMod.IsZero() {
			ago := time.Since(p.LastMod)
			fmt.Printf("    Last modified: %s\n", formatDurationAgo(ago))
		}
	}

	fmt.Printf("\nTotal: %d plugin(s)\n", len(plugins))
	return nil
}

func runPluginStatus(cmd *cobra.Command, args []string) error {
	spec := args[0]

	// Parse spec: rig/name
	parts := filepath.SplitList(spec)
	if len(parts) == 1 {
		// Try splitting by /
		parts = splitPluginSpec(spec)
	}

	if len(parts) != 2 {
		return fmt.Errorf("plugin spec must be in format: <rig>/<name>")
	}

	rigName := parts[0]
	pluginName := parts[1]

	// Find workspace
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	// Get rig path
	rigPath := filepath.Join(townRoot, rigName)

	// Check if rig exists
	if _, err := os.Stat(rigPath); err != nil {
		return fmt.Errorf("rig not found: %s", rigName)
	}

	// Find plugins
	plugins, err := findPluginsInRig(rigPath)
	if err != nil {
		return fmt.Errorf("scanning plugins: %w", err)
	}

	// Find the requested plugin
	var plugin *PluginInfo
	for i := range plugins {
		if plugins[i].Name == pluginName {
			plugin = &plugins[i]
			break
		}
	}

	if plugin == nil {
		return fmt.Errorf("plugin not found: %s/%s", rigName, pluginName)
	}

	// Display plugin status
	fmt.Printf("%s\n\n", style.Bold.Render(fmt.Sprintf("Plugin: %s", plugin.Name)))
	fmt.Printf("  Rig: %s\n", rigName)
	fmt.Printf("  Location: %s\n", plugin.Path)
	fmt.Printf("  Config: %s\n", plugin.ConfigFile)

	// Status
	statusStr := "enabled"
	statusStyle := style.Success
	if !plugin.Enabled {
		statusStr = "disabled"
		statusStyle = style.Dim
	}
	fmt.Printf("  Status: %s\n", statusStyle.Render(statusStr))

	// Last activity
	if !plugin.LastMod.IsZero() {
		fmt.Printf("  Last modified: %s (%s)\n",
			plugin.LastMod.Format("2006-01-02 15:04:05"),
			formatDurationAgo(time.Since(plugin.LastMod)))
	}

	return nil
}

// splitPluginSpec splits a plugin spec "rig/name" into parts.
func splitPluginSpec(spec string) []string {
	var result []string
	var current string

	for i, ch := range spec {
		if ch == '/' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(ch)
		}

		// Handle last character
		if i == len(spec)-1 && current != "" {
			result = append(result, current)
		}
	}

	return result
}
