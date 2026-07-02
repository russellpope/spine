// Package fsutil holds the one write primitive every spine command uses.
package fsutil

import (
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data via temp-file + rename in the same directory,
// so a crash never leaves a partial file. The rename has POSIX rename(2)
// semantics: if path is a symlink, the link itself is replaced (not the
// file it points to) by a regular file — the symlink does not survive.
// The written file's mode is always normalized to 0644, regardless of any
// pre-existing file's mode at path.
func WriteFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".spine-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Chmod(name, 0o644); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}
