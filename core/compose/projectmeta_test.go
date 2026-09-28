package compose

import (
	"os"
	"path/filepath"
	"testing"
)

func writeBase(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "compose.yml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestProjectMetaRoundTrip(t *testing.T) {
	dir := t.TempDir()
	const content = "services:\n  web:\n    image: example/web:latest\n"

	if _, ok := ReadProjectMeta(dir, "myapp"); ok {
		t.Error("ReadProjectMeta on a missing file should report ok=false")
	}
	if err := WriteProjectMetaForContent(dir, "myapp", content); err != nil {
		t.Fatal(err)
	}
	hash, ok := ReadProjectMeta(dir, "myapp")
	if !ok || hash == "" {
		t.Fatalf("ReadProjectMeta after write: hash=%q ok=%v", hash, ok)
	}

	// Corrupt baseline reads as absent, never as drifted-or-current.
	if err := os.WriteFile(ProjectMetaPath(dir, "myapp"), []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, ok := ReadProjectMeta(dir, "myapp"); ok {
		t.Error("ReadProjectMeta on corrupt JSON should report ok=false")
	}
}

func TestCheckDrift(t *testing.T) {
	dir := t.TempDir()
	const v1 = "services:\n  web:\n    image: example/web:latest\n"
	const v2 = "services:\n  web:\n    image: example/web:latest\n    ports:\n      - \"8080\"\n"

	// No baseline (pre-tracking deploy): untracked, never drifted.
	writeBase(t, dir, v1)
	if drifted, tracked, err := CheckDrift(dir, "myapp"); err != nil || drifted || tracked {
		t.Errorf("no baseline: drifted=%v tracked=%v err=%v, want false/false/nil", drifted, tracked, err)
	}

	// Baseline recorded: identical content is current.
	if err := WriteProjectMetaForContent(dir, "myapp", v1); err != nil {
		t.Fatal(err)
	}
	if drifted, tracked, err := CheckDrift(dir, "myapp"); err != nil || drifted || !tracked {
		t.Errorf("identical: drifted=%v tracked=%v err=%v, want false/true/nil", drifted, tracked, err)
	}

	// Catalog changed after deploy: drifted.
	writeBase(t, dir, v2)
	if drifted, tracked, err := CheckDrift(dir, "myapp"); err != nil || !drifted || !tracked {
		t.Errorf("changed: drifted=%v tracked=%v err=%v, want true/true/nil", drifted, tracked, err)
	}

	// Missing catalog file surfaces as an error, not a verdict.
	if drifted, tracked, err := CheckDrift(t.TempDir(), "myapp"); err == nil || drifted || tracked {
		t.Errorf("missing base: drifted=%v tracked=%v err=%v, want false/false/non-nil", drifted, tracked, err)
	}
}
