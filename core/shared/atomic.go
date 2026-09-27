package shared

import (
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data to path so that a reader ever sees either the
// previous contents or the complete new contents, never a truncated file.
//
// os.WriteFile is O_WRONLY|O_CREATE|O_TRUNC followed by write(2): a crash,
// OOM kill, or power loss between the truncate and the final byte leaves a
// short file. For the files this project writes that is not cosmetic — a
// half-written state.json is silently discarded on the next read (the JSON
// fails to unmarshal and the process continues with empty state, which
// auto-discovery then repopulates as "running"), and a half-written project
// compose file still stats successfully, so every later resuscitate, update, or
// restart for that stack keeps failing against the broken file.
//
// The sequence is: write a temp file in the *same directory*, fsync it, rename
// it into place, then fsync the directory so the rename itself is durable.
//
// Two details that are easy to get wrong:
//
//   - The temp file must be created in the target's directory, not a shared
//     staging area. os.Rename fails with EXDEV across filesystems, and this
//     project writes to both the data dir (YANTR_DATA_DIR, default /data) and
//     the apps dir, which are separate mounts in the container image.
//   - Rename carries the *temp file's* inode, so the target's permissions are
//     replaced by the temp file's. os.CreateTemp always creates at 0600, so the
//     mode is applied explicitly with Chmod. Without this, a file written 0644
//     would silently become 0600, and the reverse would silently loosen it.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, ".tmp-"+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// No-op once the rename has succeeded; cleans up every error path.
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	// Durably record the rename itself. A failure here is not fatal: the data
	// is already written, and a missing directory fsync only weakens
	// crash-consistency, never correctness of the returned value.
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}
