package shared

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileAtomicCreates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "thing.json")

	if err := WriteFileAtomic(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Errorf("contents = %q, want %q", got, "hello")
	}
}

// The permission argument must be honoured on the resulting file, not just the
// temp file. os.CreateTemp makes 0600 and rename carries the temp inode, so a
// naive temp+rename silently changes the mode of every rewritten file.
func TestWriteFileAtomicPreservesMode(t *testing.T) {
	dir := t.TempDir()

	cases := []struct {
		name string
		perm os.FileMode
	}{
		{"wide", 0644},
		{"narrow", 0600},
		{"exec", 0755},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name)
			if err := WriteFileAtomic(path, []byte("x"), tc.perm); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != tc.perm {
				t.Errorf("mode = %v, want %v", info.Mode().Perm(), tc.perm)
			}
		})
	}
}

// Rewriting must not leave the old mode in place either: os.WriteFile applies
// its perm only at creation, so a drifted file was never corrected.
func TestWriteFileAtomicCorrectsDriftedMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "drifted")
	if err := os.WriteFile(path, []byte("old"), 0666); err != nil {
		t.Fatal(err)
	}

	if err := WriteFileAtomic(path, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestWriteFileAtomicOverwrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "over.txt")

	if err := WriteFileAtomic(path, []byte("first-and-longer"), 0644); err != nil {
		t.Fatal(err)
	}
	// Shorter second write: a non-atomic truncate-then-write would be visible
	// as leftover tail bytes if the write were short.
	if err := WriteFileAtomic(path, []byte("second"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "second" {
		t.Errorf("contents = %q, want %q", got, "second")
	}
}

func TestWriteFileAtomicEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty")

	if err := WriteFileAtomic(path, nil, 0644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 0 {
		t.Errorf("size = %d, want 0", info.Size())
	}
}

// No temp files may survive a successful write.
func TestWriteFileAtomicLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clean.json")

	for i := 0; i < 5; i++ {
		if err := WriteFileAtomic(path, []byte("payload"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
	if len(entries) != 1 {
		t.Errorf("expected exactly 1 file in dir, got %d", len(entries))
	}
}

// A failed write must not damage the existing target, and must not leave a temp
// file behind.
//
// The failure is forced by renaming onto a non-empty directory, which the kernel
// rejects with ENOTEMPTY. The directory and its contents must survive.
func TestWriteFileAtomicFailureKeepsTarget(t *testing.T) {
	dir := t.TempDir()

	// "victim" is a non-empty directory standing where a file would be.
	victim := filepath.Join(dir, "victim")
	if err := os.MkdirAll(victim, 0755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(victim, "sentinel")
	if err := os.WriteFile(sentinel, []byte("intact"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := WriteFileAtomic(victim, []byte("bad"), 0644); err == nil {
		t.Fatal("expected an error renaming a file over a non-empty directory")
	}

	got, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatalf("target was damaged: %v", err)
	}
	if string(got) != "intact" {
		t.Errorf("target contents = %q, want %q", got, "intact")
	}

	// The temp file must have been cleaned up. It is created in the parent
	// directory, which is `dir`, not inside the victim directory.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-") {
			t.Errorf("temp file left behind after failure: %s", e.Name())
		}
	}
}

func TestWriteFileAtomicMissingDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-such-dir", "f")
	if err := WriteFileAtomic(path, []byte("x"), 0644); err == nil {
		t.Error("expected an error for a missing parent directory")
	}
}

// The temp file lives in the target's directory, which is what keeps rename
// from failing with EXDEV on a separate mount.
func TestWriteFileAtomicUsesTargetDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "f")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "f" {
		t.Errorf("unexpected dir contents after write: %v", entries)
	}
}

// A temp file in the app directory must not be picked up by the stack
// auto-discovery glob, which matches ".compose.*.yml".
func TestWriteFileAtomicTempNameAvoidsComposeGlob(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".compose.jellyfin.yml")

	if err := WriteFileAtomic(path, []byte("services: {}"), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".yml") && e.Name() != ".compose.jellyfin.yml" {
			t.Errorf("temp file would match the compose glob: %s", e.Name())
		}
	}
}
