package supervisor

import (
	"os"
	"path/filepath"
	"testing"

	"core/compose"
)

// newTestState resets package state and points YANTR_DATA_DIR at a temp dir.
//
// Init only adopts state.json when the file exists, so in-memory state survives
// a failed read. Without an explicit reset on the way out, these tests would
// leak stacks into any test that runs after them.
func newTestState(t *testing.T) {
	t.Helper()
	t.Setenv("YANTR_DATA_DIR", t.TempDir())
	resetSupervisorGlobals()
	t.Cleanup(resetSupervisorGlobals)
}

func resetSupervisorGlobals() {
	stateMu.Lock()
	appState = State{Version: 1, Stacks: make(map[string]StackState)}
	stateMu.Unlock()
	removingMu.Lock()
	removing = make(map[string]bool)
	removingMu.Unlock()
	resetRestartHistory()
}

func stoppedServicesOf(t *testing.T, projectID string) []string {
	t.Helper()
	got := GetStackStoppedServices(projectID)
	if got == nil {
		return nil
	}
	out := make([]string, len(got))
	copy(out, got)
	return out
}

// Stopping one service of a multi-service stack must not disable the others.
//
// This is the regression that motivated per-service desired state: a single
// stop used to mark the whole project stopped, which turned off watchdog crash
// recovery and boot resuscitation for every sibling service.
func TestStopServiceKeepsProjectRunning(t *testing.T) {
	newTestState(t)
	RecordStackDeployed("immich", "immich")

	RecordStackServiceStopped("immich", "redis")

	if !IsStackRunning("immich") {
		t.Fatal("stopping one service must not mark the whole project stopped")
	}
	if !IsServiceStopped("immich", "redis") {
		t.Error("redis should be recorded as intentionally stopped")
	}
	for _, svc := range []string{"immich-server", "immich-machine-learning", "database"} {
		if IsServiceStopped("immich", svc) {
			t.Errorf("sibling service %q was wrongly marked stopped", svc)
		}
		if !ShouldAutoRestart("immich", svc) {
			t.Errorf("watchdog should still auto-restart sibling %q", svc)
		}
	}
	if ShouldAutoRestart("immich", "redis") {
		t.Error("watchdog must not auto-restart an intentionally stopped service")
	}
}

func TestStartServiceClearsStopMark(t *testing.T) {
	newTestState(t)
	RecordStackDeployed("immich", "immich")
	RecordStackServiceStopped("immich", "redis")
	RecordStackServiceStopped("immich", "database")

	RecordStackServiceStarted("immich", "redis")

	if IsServiceStopped("immich", "redis") {
		t.Error("starting redis should clear its stop mark")
	}
	if !IsServiceStopped("immich", "database") {
		t.Error("unrelated stop marks must survive")
	}
	if !ShouldAutoRestart("immich", "redis") {
		t.Error("watchdog should auto-restart redis after an explicit start")
	}
}

// Starting a service of a fully stopped project reopens the project, so a user
// can recover a stack one service at a time.
func TestStartServiceReopensStoppedProject(t *testing.T) {
	newTestState(t)
	RecordStackDeployed("immich", "immich")
	RecordStackStopped("immich")

	if ShouldAutoRestart("immich", "immich-server") {
		t.Fatal("a stopped project should not be auto-restarted")
	}

	RecordStackServiceStarted("immich", "immich-server")

	if !IsStackRunning("immich") {
		t.Error("starting a service should reopen the project")
	}
}

func TestShouldAutoRestartRespectsRemoving(t *testing.T) {
	newTestState(t)
	RecordStackDeployed("immich", "immich")

	MarkStackRemoving("immich")
	if ShouldAutoRestart("immich", "immich-server") {
		t.Error("a stack mid-teardown must not be auto-restarted")
	}
	UnmarkStackRemoving("immich")
	if !ShouldAutoRestart("immich", "immich-server") {
		t.Error("clearing the removing flag should re-enable auto-restart")
	}
}

func TestServiceStopIsIdempotentAndSorted(t *testing.T) {
	newTestState(t)
	RecordStackDeployed("immich", "immich")

	RecordStackServiceStopped("immich", "database")
	RecordStackServiceStopped("immich", "redis")
	RecordStackServiceStopped("immich", "database")

	got := stoppedServicesOf(t, "immich")
	if len(got) != 2 {
		t.Fatalf("expected 2 distinct stopped services, got %v", got)
	}
	if got[0] != "database" || got[1] != "redis" {
		t.Errorf("stopped services not sorted/deduped: %v", got)
	}
}

func TestServiceStateIgnoresEmptyAndUnknown(t *testing.T) {
	newTestState(t)
	RecordStackDeployed("immich", "immich")

	// Must not panic or record anything.
	RecordStackServiceStopped("", "redis")
	RecordStackServiceStopped("immich", "")
	RecordStackServiceStarted("", "")
	RecordStackServiceStarted("immich", "")

	if IsServiceStopped("immich", "redis") {
		t.Error("empty service name should not have been recorded")
	}
	// Unknown project: no panic, and nothing reported as stopped.
	if IsServiceStopped("nope", "redis") {
		t.Error("unknown project should report nothing stopped")
	}
	if got := stoppedServicesOf(t, "nope"); got != nil {
		t.Errorf("unknown project should have no stopped services, got %v", got)
	}
}

// Older state.json files have no stoppedServices key. They must load as
// "nothing intentionally stopped" rather than erroring or defaulting to stopped.
func TestLegacyStateFileHasNoStoppedServices(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YANTR_DATA_DIR", dir)

	legacy := `{
  "version": 1,
  "stacks": {
    "jellyfin": {"projectId": "jellyfin", "appId": "jellyfin", "status": "running", "updatedAt": 1}
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte(legacy), 0644); err != nil {
		t.Fatal(err)
	}

	Init("")

	if !IsStackRunning("jellyfin") {
		t.Fatal("legacy state should load as running")
	}
	if IsServiceStopped("jellyfin", "jellyfin") {
		t.Error("legacy state must not imply any service is stopped")
	}
	if !ShouldAutoRestart("jellyfin", "jellyfin") {
		t.Error("legacy state should keep auto-restart enabled")
	}
}

func TestStoppedServicesRoundTripThroughStateFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YANTR_DATA_DIR", dir)

	RecordStackDeployed("immich", "immich")
	RecordStackServiceStopped("immich", "redis")

	// Re-init from disk, as a process restart would.
	Init("")

	if !IsStackRunning("immich") {
		t.Fatal("project should still be running after reload")
	}
	if !IsServiceStopped("immich", "redis") {
		t.Error("stop mark should survive a state.json reload")
	}
	if !ShouldAutoRestart("immich", "immich-server") {
		t.Error("siblings should still be auto-restartable after reload")
	}
}

// ─── Boot resuscitation service selection ────────────────────────────────────

const multiServiceCompose = `
services:
  immich-server:
    image: example/immich-server:latest
  redis:
    image: example/redis:latest
  database:
    image: example/postgres:latest
`

func writeProjectCompose(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, compose.ProjectComposeFileName("immich"))
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestServicesToResuscitate(t *testing.T) {
	newTestState(t)
	RecordStackDeployed("immich", "immich")
	path := writeProjectCompose(t, multiServiceCompose)

	t.Run("nothing stopped, nothing running", func(t *testing.T) {
		got := servicesToResuscitate(path, "immich", nil)
		if len(got) != 3 {
			t.Fatalf("expected all 3 services, got %v", got)
		}
	})

	t.Run("excludes intentionally stopped service", func(t *testing.T) {
		RecordStackServiceStopped("immich", "redis")
		got := servicesToResuscitate(path, "immich", nil)
		for _, name := range got {
			if name == "redis" {
				t.Fatalf("redis must not be resuscitated, got %v", got)
			}
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 services, got %v", got)
		}
		RecordStackServiceStarted("immich", "redis")
	})

	t.Run("excludes already running services", func(t *testing.T) {
		active := map[projectService]bool{
			{"immich", "redis"}:  true,
			{"immich", "database"}: true,
		}
		got := servicesToResuscitate(path, "immich", active)
		if len(got) != 1 || got[0] != "immich-server" {
			t.Fatalf("expected only immich-server, got %v", got)
		}
	})

	t.Run("all stopped yields nothing to do", func(t *testing.T) {
		for _, s := range []string{"immich-server", "redis", "database"} {
			RecordStackServiceStopped("immich", s)
		}
		if got := servicesToResuscitate(path, "immich", nil); len(got) != 0 {
			t.Fatalf("expected nothing to resuscitate, got %v", got)
		}
		for _, s := range []string{"immich-server", "redis", "database"} {
			RecordStackServiceStarted("immich", s)
		}
	})

	t.Run("missing compose file falls back to whole project", func(t *testing.T) {
		if got := servicesToResuscitate(filepath.Join(t.TempDir(), "nope.yml"), "immich", nil); got != nil {
			t.Fatalf("expected nil (revive whole project) for unreadable file, got %v", got)
		}
	})

	t.Run("malformed compose file falls back to whole project", func(t *testing.T) {
		bad := writeProjectCompose(t, "services: [this is not: valid: yaml")
		if got := servicesToResuscitate(bad, "immich", nil); got != nil {
			t.Fatalf("expected nil for malformed file, got %v", got)
		}
	})
}
