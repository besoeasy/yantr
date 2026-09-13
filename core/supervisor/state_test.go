package supervisor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSupervisorState(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("YANTR_DATA_DIR", tempDir)

	// Test Init with empty directory
	Init("")

	if len(GetRunningStacks()) != 0 {
		t.Fatalf("expected 0 running stacks, got %d", len(GetRunningStacks()))
	}

	// Record deployment
	RecordStackDeployed("proj-1", "vaultwarden")
	if !IsStackRunning("proj-1") {
		t.Fatalf("expected proj-1 to be running")
	}
	stacks := GetRunningStacks()
	if len(stacks) != 1 || stacks[0].ProjectID != "proj-1" || stacks[0].AppID != "vaultwarden" {
		t.Fatalf("unexpected running stacks: %+v", stacks)
	}

	// Verify state.json was written to disk
	p := filepath.Join(tempDir, "state.json")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		t.Fatalf("state.json was not created at %s", p)
	}

	// Record another deployment
	RecordStackDeployed("proj-2", "uptime-kuma")
	if len(GetRunningStacks()) != 2 {
		t.Fatalf("expected 2 running stacks, got %d", len(GetRunningStacks()))
	}

	// Record stop on proj-1
	RecordStackStopped("proj-1")
	if IsStackRunning("proj-1") {
		t.Fatalf("expected proj-1 to not be running")
	}
	stacks = GetRunningStacks()
	if len(stacks) != 1 || stacks[0].ProjectID != "proj-2" {
		t.Fatalf("expected only proj-2 running, got %+v", stacks)
	}

	// Record removal of proj-2
	RecordStackRemoved("proj-2")
	if len(GetRunningStacks()) != 0 {
		t.Fatalf("expected 0 running stacks after removal, got %d", len(GetRunningStacks()))
	}

	// Re-init should load state from disk (proj-1 exists as stopped)
	Init("")
	if IsStackRunning("proj-1") {
		t.Fatalf("expected proj-1 to still be stopped after re-init")
	}
}

func TestStackRemovingRegistry(t *testing.T) {
	if IsStackRemoving("proj-x") {
		t.Fatalf("expected proj-x to not be removing initially")
	}
	MarkStackRemoving("proj-x")
	if !IsStackRemoving("proj-x") {
		t.Fatalf("expected proj-x to be removing after MarkStackRemoving")
	}
	UnmarkStackRemoving("proj-x")
	if IsStackRemoving("proj-x") {
		t.Fatalf("expected proj-x to not be removing after UnmarkStackRemoving")
	}
}

func TestSupervisorAutoDiscovery(t *testing.T) {
	tempDataDir := t.TempDir()
	t.Setenv("YANTR_DATA_DIR", tempDataDir)

	appsDir := t.TempDir()
	vaultAppDir := filepath.Join(appsDir, "vaultwarden")
	if err := os.MkdirAll(vaultAppDir, 0755); err != nil {
		t.Fatal(err)
	}
	composeFile := filepath.Join(vaultAppDir, "compose.discovered-proj.yml")
	if err := os.WriteFile(composeFile, []byte("services: {}"), 0644); err != nil {
		t.Fatal(err)
	}

	Init(appsDir)

	if !IsStackRunning("discovered-proj") {
		t.Fatalf("expected discovered-proj to be auto-discovered and marked running")
	}
}

func TestBootStatus(t *testing.T) {
	status := BootStatus{
		Resuscitating: true,
		Total:         5,
		Completed:     2,
		CurrentStack:  "vaultwarden",
		Message:       "Resuscitating 2/5",
	}
	SetBootStatus(status)

	retrieved := GetBootStatus()
	if retrieved.Total != 5 || retrieved.Completed != 2 || !retrieved.Resuscitating || retrieved.CurrentStack != "vaultwarden" {
		t.Fatalf("unexpected boot status: %+v", retrieved)
	}
}
