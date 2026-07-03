package handoff

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestParseName(t *testing.T) {
	d, topic, ok := ParseName("2026-07-02-spine-v2-brainstorm.md")
	if !ok || topic != "spine-v2-brainstorm" || d.Format("2006-01-02") != "2026-07-02" {
		t.Fatalf("d=%v topic=%q ok=%v", d, topic, ok)
	}
	for _, bad := range []string{"README.md", "2026-13-45-x.md", "2026-07-02-x.txt", "notes.md"} {
		if _, _, ok := ParseName(bad); ok {
			t.Errorf("ParseName(%q) should fail", bad)
		}
	}
}

func TestNewListLatest(t *testing.T) {
	dir := t.TempDir()
	older := filepath.Join(dir, "docs", "handoffs", "2020-01-01-ancient-work.md")
	if err := os.MkdirAll(filepath.Dir(older), 0o755); err != nil {
		t.Fatal(err)
	}
	// legacy handoff: no front matter at all
	if err := os.WriteFile(older, []byte("# some legacy handoff\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path, err := New(dir, "spine v2 spec")
	if err != nil {
		t.Fatal(err)
	}
	today := time.Now().Format("2006-01-02")
	if !strings.HasSuffix(path, today+"-spine-v2-spec.md") {
		t.Fatalf("path=%q", path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"title: spine v2 spec", "created: " + today, "## Context", "## Gotchas"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("missing %q in %q", want, raw)
		}
	}
	if _, err := New(dir, "spine v2 spec"); err == nil {
		t.Fatal("same-day collision must error")
	}
	entries, err := List(dir)
	if err != nil || len(entries) != 2 {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
	if entries[0].Topic != "spine-v2-spec" || entries[1].Topic != "ancient-work" {
		t.Fatalf("order wrong: %v", entries)
	}
	if entries[0].Title != "spine v2 spec" {
		t.Errorf("title from front matter, got %q", entries[0].Title)
	}
	if entries[1].Title != "ancient-work" {
		t.Errorf("legacy title falls back to topic, got %q", entries[1].Title)
	}
	latest, ok, err := Latest(dir)
	if err != nil || !ok || latest.Topic != "spine-v2-spec" {
		t.Fatalf("latest=%v ok=%v err=%v", latest, ok, err)
	}
}

func TestNewRefusesWhenStatFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation may require privileges on Windows")
	}
	dir := t.TempDir()
	hdir := filepath.Join(dir, "docs", "handoffs")
	if err := os.MkdirAll(hdir, 0o755); err != nil {
		t.Fatal(err)
	}
	today := time.Now().Format("2006-01-02")
	path := filepath.Join(hdir, today+"-self-loop.md")
	// A self-referential symlink makes os.Stat fail with ELOOP — an error
	// that is neither nil nor IsNotExist. New must surface it instead of
	// falling through to WriteFileAtomic, whose POSIX rename would silently
	// replace the existing directory entry ("never overwrites" contract).
	if err := os.Symlink(path, path); err != nil {
		t.Fatal(err)
	}
	if _, err := New(dir, "self loop"); err == nil {
		t.Fatal("New must fail when Stat on the target errors")
	}
	fi, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("target was replaced (mode %v) — New overwrote on Stat failure", fi.Mode())
	}
}

func TestListMissingDirIsEmpty(t *testing.T) {
	entries, err := List(t.TempDir())
	if err != nil || entries != nil {
		t.Fatalf("want nil,nil got %v,%v", entries, err)
	}
}

func TestFleet(t *testing.T) {
	parent := t.TempDir()
	mk := func(repo, name string) {
		p := filepath.Join(parent, repo, "docs", "handoffs")
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p, name), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mk("alpha", "2026-07-01-older.md")
	mk("beta", "2026-07-02-newer.md")
	if err := os.MkdirAll(filepath.Join(parent, "no-handoffs-repo"), 0o755); err != nil {
		t.Fatal(err)
	}
	// broken-repo: docs/handoffs is a regular FILE, so List's os.ReadDir
	// fails ENOTDIR — a real (non-NotExist) error, unlike no-handoffs-repo
	// which exercises the ok=false path. Fleet must skip it silently via
	// its err != nil branch, not fail the whole scan.
	if err := os.MkdirAll(filepath.Join(parent, "broken-repo", "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "broken-repo", "docs", "handoffs"), []byte("not a dir\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Guard: the fixture must produce a real List error, or this case would
	// silently degrade into a duplicate of the ok=false path above.
	if _, err := List(filepath.Join(parent, "broken-repo")); err == nil {
		t.Fatal("fixture broken: List on broken-repo must error (ENOTDIR)")
	}
	rows, err := Fleet(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Repo != "beta" || rows[1].Repo != "alpha" {
		t.Fatalf("rows=%v", rows)
	}
	if _, err := Fleet(filepath.Join(parent, "does-not-exist")); err == nil {
		t.Fatal("missing parent must error")
	}
}
