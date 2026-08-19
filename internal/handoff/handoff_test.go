package handoff

import (
	"os"
	"path/filepath"
	"strconv"
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
	for _, want := range []string{"title: " + strconv.Quote("spine v2 spec"), "created: " + today, "## Context", "## Gotchas"} {
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

// I062: two handoffs created on the same day must use the persisted creation
// ordinal, not their filename, to decide which snapshot is newest.
func TestNewSameDateNewerCreationOutranksFilename(t *testing.T) {
	dir := t.TempDir()
	older, err := New(dir, "zeta older")
	if err != nil {
		t.Fatal(err)
	}
	newer, err := New(dir, "alpha newer")
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{older: "handoff_ordinal: 1", newer: "handoff_ordinal: 2"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), want) {
			t.Errorf("%s missing %q:\n%s", path, want, raw)
		}
	}
	latest, ok, err := Latest(dir)
	if err != nil || !ok || latest.Path != newer {
		t.Fatalf("Latest = %#v, %v, %v; want newer-created %q", latest, ok, err, newer)
	}
}

// I062: date remains the primary ordering key even when an older handoff has
// a greater persisted ordinal from a later repository-wide creation.
func TestListDifferentDatesRemainPrimaryOverOrdinal(t *testing.T) {
	dir := t.TempDir()
	hdir := filepath.Join(dir, "docs", "handoffs")
	if err := os.MkdirAll(hdir, 0o755); err != nil {
		t.Fatal(err)
	}
	older := filepath.Join(hdir, "2026-08-08-zeta.md")
	newer := filepath.Join(hdir, "2026-08-09-alpha.md")
	if err := os.WriteFile(older, []byte("---\nhandoff_ordinal: 99\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newer, []byte("---\nhandoff_ordinal: 1\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	latest, ok, err := Latest(dir)
	if err != nil || !ok || latest.Path != newer {
		t.Fatalf("Latest = %#v, %v, %v; want later-date %q", latest, ok, err, newer)
	}
}

// I062: existing handoffs have no persisted ordinal. Their same-date order
// retains the former filename rule, rather than inheriting checkout mtimes.
func TestListLegacySameDateFallsBackToFilenameDeterministically(t *testing.T) {
	dir := t.TempDir()
	hdir := filepath.Join(dir, "docs", "handoffs")
	if err := os.MkdirAll(hdir, 0o755); err != nil {
		t.Fatal(err)
	}
	older := filepath.Join(hdir, "2026-08-09-alpha.md")
	newer := filepath.Join(hdir, "2026-08-09-zeta.md")
	for _, path := range []string{older, newer} {
		if err := os.WriteFile(path, []byte("# legacy\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	latest, ok, err := Latest(dir)
	if err != nil || !ok || latest.Path != newer {
		t.Fatalf("Latest = %#v, %v, %v; want legacy filename fallback %q", latest, ok, err, newer)
	}
}

// I062: ordinals compare numerically, and every absent or malformed ordinal
// takes the deterministic legacy filename fallback rather than an mtime path.
func TestListOrdinalNumericAndFallbackOrdering(t *testing.T) {
	write := func(t *testing.T, dir, name, ordinal string) string {
		t.Helper()
		path := filepath.Join(dir, "docs", "handoffs", name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		content := "---\nhandoff_ordinal: " + ordinal + "\n---\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	t.Run("numeric 10 beats 9", func(t *testing.T) {
		dir := t.TempDir()
		want := write(t, dir, "2026-08-09-alpha-ten.md", "10")
		write(t, dir, "2026-08-09-zeta-nine.md", "9")
		got, ok, err := Latest(dir)
		if err != nil || !ok || got.Path != want {
			t.Fatalf("Latest = %#v, %v, %v; want numeric ordinal winner %q", got, ok, err, want)
		}
	})
	t.Run("equal ordinals fall back to filename", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "2026-08-09-alpha.md", "7")
		want := write(t, dir, "2026-08-09-zeta.md", "7")
		got, ok, err := Latest(dir)
		if err != nil || !ok || got.Path != want {
			t.Fatalf("Latest = %#v, %v, %v; want filename fallback %q", got, ok, err, want)
		}
	})
	for _, malformed := range []string{"", "0", "-1", "1.5", "junk", "18446744073709551616"} {
		t.Run("malformed "+strconv.Quote(malformed)+" falls back to filename", func(t *testing.T) {
			dir := t.TempDir()
			write(t, dir, "2026-08-09-alpha.md", malformed)
			want := write(t, dir, "2026-08-09-zeta.md", malformed)
			got, ok, err := Latest(dir)
			if err != nil || !ok || got.Path != want {
				t.Fatalf("Latest = %#v, %v, %v; want malformed fallback %q", got, ok, err, want)
			}
		})
	}
}

// I062: a fresh clone has no reliable mtime history. Re-reading identical
// committed frontmatter must retain the same same-date answer regardless of
// the files' checkout mtimes.
func TestListSameDateOrdinalIsFreshCloneDeterministic(t *testing.T) {
	makeClone := func(t *testing.T, root string, newerMtime time.Time) string {
		t.Helper()
		hdir := filepath.Join(root, "docs", "handoffs")
		if err := os.MkdirAll(hdir, 0o755); err != nil {
			t.Fatal(err)
		}
		newer := filepath.Join(hdir, "2026-08-09-alpha-newer.md")
		older := filepath.Join(hdir, "2026-08-09-zeta-older.md")
		if err := os.WriteFile(newer, []byte("---\nhandoff_ordinal: 42\n---\n# newer\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(older, []byte("---\nhandoff_ordinal: 41\n---\n# older\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(older, newerMtime, newerMtime); err != nil {
			t.Fatal(err)
		}
		return newer
	}

	first := t.TempDir()
	firstNewer := makeClone(t, first, time.Date(2026, 8, 10, 1, 0, 0, 0, time.UTC))
	second := t.TempDir()
	secondNewer := makeClone(t, second, time.Date(2020, 1, 1, 1, 0, 0, 0, time.UTC))
	for _, tc := range []struct {
		dir  string
		want string
	}{{first, firstNewer}, {second, secondNewer}} {
		latest, ok, err := Latest(tc.dir)
		if err != nil || !ok || latest.Path != tc.want {
			t.Fatalf("fresh-clone Latest(%q) = %#v, %v, %v; want %q", tc.dir, latest, ok, err, tc.want)
		}
	}
}

func TestNewRefusesWhenStatFails(t *testing.T) {
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

func TestNewSecondSameTopicSameDayFails(t *testing.T) {
	dir := t.TempDir()
	if _, err := New(dir, "same topic"); err != nil {
		t.Fatal(err)
	}
	_, err := New(dir, "same topic")
	if err == nil || !strings.Contains(err.Error(), "already exists — pick a more specific topic") {
		t.Fatalf("want already-exists error, got %v", err)
	}
}

func TestListSurfacesHandoffReadError(t *testing.T) {
	dir := t.TempDir()
	hdir := filepath.Join(dir, "docs", "handoffs")
	if err := os.MkdirAll(hdir, 0o755); err != nil {
		t.Fatal(err)
	}
	name := "2026-07-03-loop.md"
	if err := os.Symlink(name, filepath.Join(hdir, name)); err != nil {
		t.Fatal(err)
	}
	_, err := List(dir)
	if err == nil {
		t.Fatal("want read error surfaced, got nil (Title silently degraded before v3)")
	}
}

func TestLegacyUnquotedTitleListsVerbatim(t *testing.T) {
	dir := t.TempDir()
	hdir := filepath.Join(dir, "docs", "handoffs")
	if err := os.MkdirAll(hdir, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := `---
title: legacy: unquoted title
created: 2026-01-15
---

# Handoff — legacy: unquoted title (2026-01-15)
`
	if err := os.WriteFile(filepath.Join(hdir, "2026-01-15-legacy.md"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := List(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
	if entries[0].Title != "legacy: unquoted title" {
		t.Errorf("legacy Title = %q, want verbatim %q", entries[0].Title, "legacy: unquoted title")
	}
}

func TestNewQuotesTitleForYAML(t *testing.T) {
	dir := t.TempDir()
	topic := `spine v4: the "quoting" handoff`
	path, err := New(dir, topic)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "\ntitle: "+strconv.Quote(topic)+"\n") {
		t.Errorf("title not quoted/escaped:\n%s", s)
	}
	if !strings.Contains(s, "# Handoff — "+topic+" (") {
		t.Errorf("body H1 must keep the raw topic:\n%s", s)
	}
	entries, err := List(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
	if entries[0].Title != topic {
		t.Errorf("display Title = %q, want unquoted %q", entries[0].Title, topic)
	}
}

func TestNewBackslashTopicRoundtrip(t *testing.T) {
	dir := t.TempDir()
	topic := `back\slash and "quotes" and colon: all at once`
	if _, err := New(dir, topic); err != nil {
		t.Fatal(err)
	}
	entries, err := List(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
	if entries[0].Title != topic {
		t.Errorf("roundtrip Title = %q, want %q", entries[0].Title, topic)
	}
}
