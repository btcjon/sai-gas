package epictemplate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTemplate(t *testing.T) {
	// Create a temporary template file
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "test-template.yaml")

	content := `name: test-batch
description: A test batch template
phases:
  - name: startup
    issues:
      - title: "First task"
        type: task
  - name: work
    issues:
      - title: "Main work"
        description: "Do the main work"
        type: feature
`

	if err := os.WriteFile(templatePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test template: %v", err)
	}

	tmpl, err := LoadTemplate(templatePath)
	if err != nil {
		t.Fatalf("LoadTemplate failed: %v", err)
	}

	if tmpl.Name != "test-batch" {
		t.Errorf("Expected name 'test-batch', got '%s'", tmpl.Name)
	}

	if tmpl.Description != "A test batch template" {
		t.Errorf("Unexpected description: %s", tmpl.Description)
	}

	if len(tmpl.Phases) != 2 {
		t.Errorf("Expected 2 phases, got %d", len(tmpl.Phases))
	}

	if tmpl.TotalIssues() != 2 {
		t.Errorf("Expected 2 total issues, got %d", tmpl.TotalIssues())
	}
}

func TestListTemplates(t *testing.T) {
	// Create temp directory with templates
	dir := t.TempDir()

	// Write test templates
	templates := map[string]string{
		"batch-a.yaml": "name: batch-a\nphases: []",
		"batch-b.yml":  "name: batch-b\nphases: []",
		"not-yaml.txt": "this is not yaml",
	}

	for name, content := range templates {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", name, err)
		}
	}

	result, err := ListTemplates([]string{dir})
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 templates, got %d", len(result))
	}

	if _, ok := result["batch-a"]; !ok {
		t.Error("Expected batch-a template")
	}

	if _, ok := result["batch-b"]; !ok {
		t.Error("Expected batch-b template")
	}
}

func TestSortedNames(t *testing.T) {
	templates := map[string]string{
		"zebra": "/path/zebra.yaml",
		"alpha": "/path/alpha.yaml",
		"beta":  "/path/beta.yaml",
	}

	names := SortedNames(templates)

	if len(names) != 3 {
		t.Fatalf("Expected 3 names, got %d", len(names))
	}

	expected := []string{"alpha", "beta", "zebra"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("Position %d: expected '%s', got '%s'", i, expected[i], name)
		}
	}
}

func TestFindTemplate(t *testing.T) {
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "my-template.yaml")

	if err := os.WriteFile(templatePath, []byte("name: my-template\nphases: []"), 0644); err != nil {
		t.Fatal(err)
	}

	path, err := FindTemplate("my-template", []string{dir})
	if err != nil {
		t.Fatalf("FindTemplate failed: %v", err)
	}

	if path != templatePath {
		t.Errorf("Expected path '%s', got '%s'", templatePath, path)
	}

	// Test not found
	_, err = FindTemplate("nonexistent", []string{dir})
	if err == nil {
		t.Error("Expected error for nonexistent template")
	}
}

func TestSearchPaths(t *testing.T) {
	paths := SearchPaths("/project", "/town")

	// Should have 6 paths: 2 for project, 2 for user, 2 for town
	// (each location has base and epic subdirectory)
	if len(paths) < 4 {
		t.Errorf("Expected at least 4 paths, got %d", len(paths))
	}

	// Check that project templates comes first
	if paths[0] != "/project/templates" {
		t.Errorf("Expected first path to be /project/templates, got %s", paths[0])
	}

	// Check that epic subdirectory is included
	if paths[1] != "/project/templates/epic" {
		t.Errorf("Expected second path to be /project/templates/epic, got %s", paths[1])
	}
}
