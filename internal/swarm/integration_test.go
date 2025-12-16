package swarm

import (
	"testing"

	"github.com/steveyegge/gastown/internal/rig"
)

func TestGetIntegrationBranch(t *testing.T) {
	r := &rig.Rig{
		Name: "test-rig",
		Path: "/tmp/test-rig",
	}
	m := NewManager(r)

	swarm, _ := m.Create("epic-1", []string{"Toast"}, "main")

	branch, err := m.GetIntegrationBranch(swarm.ID)
	if err != nil {
		t.Fatalf("GetIntegrationBranch failed: %v", err)
	}

	expected := "swarm/epic-1"
	if branch != expected {
		t.Errorf("branch = %q, want %q", branch, expected)
	}
}

func TestGetIntegrationBranchNotFound(t *testing.T) {
	r := &rig.Rig{
		Name: "test-rig",
		Path: "/tmp/test-rig",
	}
	m := NewManager(r)

	_, err := m.GetIntegrationBranch("nonexistent")
	if err != ErrSwarmNotFound {
		t.Errorf("GetIntegrationBranch = %v, want ErrSwarmNotFound", err)
	}
}

func TestGetWorkerBranch(t *testing.T) {
	r := &rig.Rig{
		Name: "test-rig",
		Path: "/tmp/test-rig",
	}
	m := NewManager(r)

	branch := m.GetWorkerBranch("sw-1", "Toast", "task-123")
	expected := "sw-1/Toast/task-123"
	if branch != expected {
		t.Errorf("branch = %q, want %q", branch, expected)
	}
}

func TestCreateIntegrationBranchSwarmNotFound(t *testing.T) {
	r := &rig.Rig{
		Name: "test-rig",
		Path: "/tmp/test-rig",
	}
	m := NewManager(r)

	err := m.CreateIntegrationBranch("nonexistent")
	if err != ErrSwarmNotFound {
		t.Errorf("CreateIntegrationBranch = %v, want ErrSwarmNotFound", err)
	}
}

func TestMergeToIntegrationSwarmNotFound(t *testing.T) {
	r := &rig.Rig{
		Name: "test-rig",
		Path: "/tmp/test-rig",
	}
	m := NewManager(r)

	err := m.MergeToIntegration("nonexistent", "branch")
	if err != ErrSwarmNotFound {
		t.Errorf("MergeToIntegration = %v, want ErrSwarmNotFound", err)
	}
}

func TestLandToMainSwarmNotFound(t *testing.T) {
	r := &rig.Rig{
		Name: "test-rig",
		Path: "/tmp/test-rig",
	}
	m := NewManager(r)

	err := m.LandToMain("nonexistent")
	if err != ErrSwarmNotFound {
		t.Errorf("LandToMain = %v, want ErrSwarmNotFound", err)
	}
}

func TestCleanupBranchesSwarmNotFound(t *testing.T) {
	r := &rig.Rig{
		Name: "test-rig",
		Path: "/tmp/test-rig",
	}
	m := NewManager(r)

	err := m.CleanupBranches("nonexistent")
	if err != ErrSwarmNotFound {
		t.Errorf("CleanupBranches = %v, want ErrSwarmNotFound", err)
	}
}
