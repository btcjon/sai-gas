// Package epictemplate provides templates for batch work patterns that can
// be instantiated as beads epics.
//
// Templates define reusable patterns for common batch work scenarios,
// such as multi-worker deployments, release workflows, or migration tasks.
// Each template contains phases with predefined issues that get created
// when the template is spawned.
package epictemplate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Template represents a batch work template.
type Template struct {
	Name        string  `yaml:"name"`
	Description string  `yaml:"description"`
	Phases      []Phase `yaml:"phases"`
}

// Phase represents a phase within a template.
type Phase struct {
	Name   string  `yaml:"name"`
	Issues []Issue `yaml:"issues,omitempty"`
}

// Issue represents an issue to be created within a phase.
type Issue struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description,omitempty"`
	Type        string `yaml:"type,omitempty"` // task, bug, feature
}

// SearchPaths returns the ordered list of directories to search for templates.
// Order: project templates, user config, town templates
// For each base path, we also check an "epic" subdirectory.
func SearchPaths(projectRoot, townRoot string) []string {
	var paths []string

	// Helper to add both base and epic subdirectory
	addPaths := func(base string) {
		paths = append(paths, base)
		paths = append(paths, filepath.Join(base, "epic"))
	}

	// 1. Project-level templates (if in a rig)
	if projectRoot != "" {
		addPaths(filepath.Join(projectRoot, "templates"))
	}

	// 2. User config templates
	if home, err := os.UserHomeDir(); err == nil {
		addPaths(filepath.Join(home, ".config", "gastown", "templates"))
	}

	// 3. Town-level templates
	if townRoot != "" {
		addPaths(filepath.Join(townRoot, "templates"))
	}

	return paths
}

// ListTemplates finds all available templates from all search paths.
// Returns a map of template name to file path.
func ListTemplates(searchPaths []string) (map[string]string, error) {
	templates := make(map[string]string)

	for _, dir := range searchPaths {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("reading %s: %w", dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			// Accept .yaml and .yml files
			if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
				continue
			}

			// Extract template name (without extension)
			templateName := strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml")

			// First match wins (project > user > town)
			if _, exists := templates[templateName]; !exists {
				templates[templateName] = filepath.Join(dir, name)
			}
		}
	}

	return templates, nil
}

// SortedNames returns template names sorted alphabetically.
func SortedNames(templates map[string]string) []string {
	names := make([]string, 0, len(templates))
	for name := range templates {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// LoadTemplate loads and parses a template from the given path.
func LoadTemplate(path string) (*Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading template: %w", err)
	}

	var tmpl Template
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	// Validate template
	if tmpl.Name == "" {
		tmpl.Name = strings.TrimSuffix(strings.TrimSuffix(filepath.Base(path), ".yaml"), ".yml")
	}

	return &tmpl, nil
}

// FindTemplate searches for a template by name in the search paths.
func FindTemplate(name string, searchPaths []string) (string, error) {
	templates, err := ListTemplates(searchPaths)
	if err != nil {
		return "", err
	}

	path, ok := templates[name]
	if !ok {
		return "", fmt.Errorf("template '%s' not found", name)
	}

	return path, nil
}

// TotalIssues returns the total number of issues across all phases.
func (t *Template) TotalIssues() int {
	count := 0
	for _, phase := range t.Phases {
		count += len(phase.Issues)
	}
	return count
}
