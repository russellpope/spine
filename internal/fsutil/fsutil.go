// Package fsutil holds the write primitives every spine command uses.
package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeTemp stages data in a temp file in dir, mode 0644, ready for an
// atomic commit (rename or link). On any error the temp file is removed.
func writeTemp(dir string, data []byte) (string, error) {
	tmp, err := os.CreateTemp(dir, ".spine-*")
	if err != nil {
		return "", err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	if err := os.Chmod(name, 0o644); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}

// WriteFileAtomic writes data via temp-file + rename in the same directory,
// so a crash never leaves a partial file. The rename has POSIX rename(2)
// semantics: if path is a symlink, the link itself is replaced (not the
// file it points to) by a regular file — the symlink does not survive.
// The written file's mode is always normalized to 0644, regardless of any
// pre-existing file's mode at path.
func WriteFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	name, err := writeTemp(dir, data)
	if err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

// WriteFileExclusive writes data to path only if path does not already
// exist. The content lands via temp-file + link(2), so the create is atomic
// and a crash never leaves a partial file at path. Any pre-existing path —
// regular file, directory, or symlink (even dangling) — fails with an error
// satisfying errors.Is(err, fs.ErrExist); callers own the user-facing
// message. Mode is normalized to 0644, matching WriteFileAtomic.
//
// Requires a filesystem with hard-link support: os.Link fails with
// EPERM/ENOTSUP-class errors on filesystems without it (known constraint;
// the fleet is APFS). If removing the temp file AFTER a successful link
// fails, that error is returned even though path WAS written with full
// content — a re-run then truthfully reports "already exists".
func WriteFileExclusive(path string, data []byte) error {
	dir := filepath.Dir(path)
	name, err := writeTemp(dir, data)
	if err != nil {
		return err
	}
	if err := os.Link(name, path); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Remove(name); err != nil {
		return fmt.Errorf("%s was written, but removing temp file failed: %w", path, err)
	}
	return nil
}
