package fsutil_test

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
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

func TestWriteFileExclusiveCreates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.md")
	if err := fsutil.WriteFileExclusive(path, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "hello" {
		t.Fatalf("content=%q err=%v", raw, err)
	}
	fi, err := os.Stat(path)
	if err != nil || fi.Mode().Perm() != 0o644 {
		t.Fatalf("mode=%v err=%v", fi.Mode(), err)
	}
	// No temp residue.
	des, _ := os.ReadDir(dir)
	if len(des) != 1 {
		t.Fatalf("residue in dir: %v", des)
	}
}

func TestWriteFileExclusiveRefusesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "taken.md")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := fsutil.WriteFileExclusive(path, []byte("usurper"))
	if !errors.Is(err, fs.ErrExist) {
		t.Fatalf("want fs.ErrExist, got %v", err)
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "original" {
		t.Fatalf("existing content clobbered: %q", raw)
	}
	des, _ := os.ReadDir(dir)
	if len(des) != 1 {
		t.Fatalf("residue in dir: %v", des)
	}
}

func TestWriteFileExclusiveRefusesDanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "link.md")
	if err := os.Symlink(filepath.Join(dir, "nowhere"), path); err != nil {
		t.Fatal(err)
	}
	// link(2) fails EEXIST on an existing path even when it is a dangling
	// symlink — the never-overwrite contract must hold for links too.
	if err := fsutil.WriteFileExclusive(path, []byte("x")); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("want fs.ErrExist, got %v", err)
	}
}

func TestWriteFileExclusiveConcurrentSingleWinner(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "raced.md")
	const n = 8
	start := make(chan struct{})
	errs := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs <- fsutil.WriteFileExclusive(path, []byte(fmt.Sprintf("writer-%d", i)))
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)
	wins, exists := 0, 0
	for err := range errs {
		switch {
		case err == nil:
			wins++
		case errors.Is(err, fs.ErrExist):
			exists++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if wins != 1 || exists != n-1 {
		t.Fatalf("wins=%d exists=%d (want 1 / %d)", wins, exists, n-1)
	}
	des, _ := os.ReadDir(dir)
	if len(des) != 1 {
		t.Fatalf("temp residue after race: %v", des)
	}
}
