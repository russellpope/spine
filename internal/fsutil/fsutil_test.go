package fsutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/russellpope/spine/internal/fsutil"
)

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.md")
	if err := fsutil.WriteFileAtomic(path, []byte("hello\n")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "hello\n" {
		t.Fatalf("read back %q, %v", got, err)
	}
	// overwrite works and leaves no temp litter
	if err := fsutil.WriteFileAtomic(path, []byte("two\n")); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("temp file litter: %v", entries)
	}
}
