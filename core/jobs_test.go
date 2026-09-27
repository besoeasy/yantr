package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestJobLifecycle(t *testing.T) {
	store := &JobStore{
		jobs: make(map[string]*Job),
		seq:  make([]string, 0),
	}

	j := store.Create("deploy", "nextcloud", "Deploy nextcloud")
	if j.Status != JobStatusRunning {
		t.Fatalf("expected running, got %s", j.Status)
	}

	j.AddLog("Pulling image...")
	j.AddLog("Extracting layers...")
	j.SetProgress("In progress")

	snap := j.Snapshot()
	if len(snap.Logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(snap.Logs))
	}
	if snap.Progress != "In progress" {
		t.Fatalf("expected 'In progress', got '%s'", snap.Progress)
	}

	active := store.FindActiveByTarget("nextcloud")
	if active == nil || active.ID != j.ID {
		t.Fatalf("expected to find active job for nextcloud")
	}

	res := map[string]interface{}{"container": "nextcloud-app"}
	j.Complete(res)

	if j.Status != JobStatusCompleted {
		t.Fatalf("expected completed, got %s", j.Status)
	}
	if j.CompletedAt == nil {
		t.Fatalf("expected non-nil completedAt")
	}

	// Should no longer be active
	active = store.FindActiveByTarget("nextcloud")
	if active != nil {
		t.Fatalf("expected nil active job after completion")
	}
}

func TestJobFailure(t *testing.T) {
	store := &JobStore{
		jobs: make(map[string]*Job),
		seq:  make([]string, 0),
	}

	j := store.Create("deploy", "failapp", "Deploy failapp")
	j.Fail(errors.New("image pull failed"), 1)

	if j.Status != JobStatusFailed {
		t.Fatalf("expected failed status, got %s", j.Status)
	}
	if j.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", j.ExitCode)
	}
	if j.Error != "image pull failed" {
		t.Fatalf("expected 'image pull failed', got '%s'", j.Error)
	}
}

func TestJobStoreRetention(t *testing.T) {
	store := &JobStore{
		jobs: make(map[string]*Job),
		seq:  make([]string, 0),
	}

	for i := 0; i < 60; i++ {
		store.Create("deploy", fmt.Sprintf("app-%d", i), "title")
	}

	list := store.List(100)
	if len(list) != maxRetainedJobs {
		t.Fatalf("expected %d retained jobs, got %d", maxRetainedJobs, len(list))
	}
	// Newest should be app-59
	if list[0].Target != "app-59" {
		t.Fatalf("expected newest target app-59, got %s", list[0].Target)
	}
}

func TestLineWriter(t *testing.T) {
	var sb strings.Builder
	var lines []string

	lw := newLineWriter(&sb, func(line string) {
		lines = append(lines, line)
	})

	_, _ = lw.Write([]byte("first line\nsec"))
	_, _ = lw.Write([]byte("ond line\nthird"))
	lw.Flush()

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "first line" || lines[1] != "second line" || lines[2] != "third" {
		t.Fatalf("unexpected line contents: %v", lines)
	}
	if sb.String() != "first line\nsecond line\nthird" {
		t.Fatalf("unexpected sb content: %s", sb.String())
	}
}
