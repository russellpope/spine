package adr_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/adr"
)

func adrDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestNewNumbersFromOne(t *testing.T) {
	dir := adrDir(t)
	path, err := adr.New(dir, "Go with stdlib only", 0)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "0001-go-with-stdlib-only.md" {
		t.Fatalf("path = %s", path)
	}
	raw, _ := os.ReadFile(path)
	for _, want := range []string{`id: "0001"`, `title: "Go with stdlib only"`, "status: Accepted",
		"# 0001: Go with stdlib only", "## Decision"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("missing %q in:\n%s", want, raw)
		}
	}
	path2, err := adr.New(dir, "Second decision", 0)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path2) != "0002-second-decision.md" {
		t.Fatalf("path2 = %s", path2)
	}
}

func TestListSorted(t *testing.T) {
	dir := adrDir(t)
	adr.New(dir, "First", 0)
	adr.New(dir, "Second", 0)
	entries, err := adr.List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].ID != 1 || entries[1].Title != "Second" ||
		entries[0].Status != "Accepted" {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestSupersedeFlipsStatus(t *testing.T) {
	dir := adrDir(t)
	first, _ := adr.New(dir, "Old way", 0)
	_, err := adr.New(dir, "New way", 1)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(first)
	if !strings.Contains(string(raw), "status: Superseded by 0002") {
		t.Errorf("old ADR not flipped:\n%s", raw)
	}
	entries, _ := adr.List(dir)
	if entries[1].Status != "Accepted" {
		t.Errorf("new ADR status = %q", entries[1].Status)
	}
	raw2, _ := os.ReadFile(filepath.Join(dir, "docs", "adr", "0002-new-way.md"))
	if !strings.Contains(string(raw2), `supersedes: "0001"`) {
		t.Errorf("new ADR missing supersedes line:\n%s", raw2)
	}
}

func TestSupersedeMissingTarget(t *testing.T) {
	if _, err := adr.New(adrDir(t), "X", 9); err == nil {
		t.Fatal("want error for missing supersede target")
	}
}

func TestSlugStripsPunctuation(t *testing.T) {
	dir := adrDir(t)
	path, err := adr.New(dir, "docs/specs absorbs plans, fleet-wide!", 0)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "0001-docs-specs-absorbs-plans-fleet-wide.md" {
		t.Fatalf("path = %s", path)
	}
}

func TestNewEmptySlugRejected(t *testing.T) {
	dir := adrDir(t)
	if _, err := adr.New(dir, "!!!", 0); err == nil {
		t.Fatal("want error for title that produces an empty slug")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "docs", "adr", "*.md"))
	if len(files) != 0 {
		t.Fatalf("want no files written after rejected New, got %v", files)
	}
}

func TestNewTitleWithNewlineRejected(t *testing.T) {
	dir := adrDir(t)
	if _, err := adr.New(dir, "x\nstatus: Evil", 0); err == nil {
		t.Fatal("want error for title containing a newline")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "docs", "adr", "*.md"))
	if len(files) != 0 {
		t.Fatalf("want no files written after rejected New, got %v", files)
	}
}

func TestListIgnoresBodyLinesOutsideFrontMatter(t *testing.T) {
	dir := adrDir(t)
	path, err := adr.New(dir, "Scoped front matter", 0)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a body that happens to contain lines shaped like front
	// matter keys; List must not let these override the real status.
	if err := os.WriteFile(path, append(raw, []byte("\nstatus: Draft\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := adr.List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Status != "Accepted" {
		t.Fatalf("entries = %#v, want status Accepted (front matter is scoped to the --- block)", entries)
	}
}

func TestSupersedeScopedToFrontMatterBlockBodyIntact(t *testing.T) {
	dir := adrDir(t)
	target := filepath.Join(dir, "docs", "adr", "0001-x.md")
	// Front matter has the real status line; the body ALSO has a column-0
	// "status: " line (as pre-spine ADRs sometimes do). Only the front-matter
	// line may flip.
	body := "---\nid: 0001\ntitle: X\nstatus: Accepted\ndate: 2026-01-01\n---\n\n# 0001: X\n\nstatus: draft\n"
	if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := adr.New(dir, "y", 1); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	if !strings.Contains(content, "status: Superseded by 0002") {
		t.Errorf("front-matter status not flipped:\n%s", content)
	}
	if !strings.Contains(content, "\nstatus: draft\n") {
		t.Errorf("body status line was mutated (should be untouched):\n%s", content)
	}
}

func TestSupersedeNoFrontMatterTargetErrorsNoWrite(t *testing.T) {
	dir := adrDir(t)
	target := filepath.Join(dir, "docs", "adr", "0001-legacy.md")
	// Pre-spine style: capitalized "Status:" (not machine front matter), plus
	// a column-0 lowercase "status: " line buried in a body code sample. A
	// naive whole-file scan for the first "status: " line would find and
	// rewrite that body line; the fix must refuse instead since there is no
	// --- ... --- front-matter block at all.
	body := "# ADR 0001 — Legacy\n\nStatus: Accepted\nDate: 2026-01-01\n\n## Example config\n\n```\nstatus: draft\n```\n"
	if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := adr.New(dir, "y", 1); err == nil {
		t.Fatal("want error when supersede target has no front matter")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "docs", "adr", "0002-*.md"))
	if len(files) != 0 {
		t.Fatalf("want no new ADR written when supersede target has no front matter, got %v", files)
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != body {
		t.Fatalf("target must be byte-identical when validation fails:\n%s", raw)
	}
}

func TestSupersedeValidatesBeforeWriting(t *testing.T) {
	dir := adrDir(t)
	// Hand-write a target ADR with no status line, so the supersede flip
	// cannot be computed. New must fail before writing anything.
	target := filepath.Join(dir, "docs", "adr", "0001-x.md")
	body := "---\nid: 0001\ntitle: X\ndate: 2026-01-01\n---\n\n# 0001: X\n"
	if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := adr.New(dir, "y", 1); err == nil {
		t.Fatal("want error when supersede target has no status line")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "docs", "adr", "0002-*.md"))
	if len(files) != 0 {
		t.Fatalf("want no new ADR written when supersede validation fails, got %v", files)
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "Superseded") {
		t.Fatalf("target must be untouched when validation fails:\n%s", raw)
	}
}

func TestNewQuotesFrontMatterScalars(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	title := `spine v3: the "sweep" release`
	path, err := adr.New(dir, title, 0)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "\nid: \"0001\"\n") {
		t.Errorf("id not quoted:\n%s", s)
	}
	if !strings.Contains(s, "\ntitle: "+strconv.Quote(title)+"\n") {
		t.Errorf("title not quoted/escaped:\n%s", s)
	}
	if !strings.Contains(s, "# 0001: "+title+"\n") {
		t.Errorf("body H1 must keep the raw title:\n%s", s)
	}
	entries, err := adr.List(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
	if entries[0].Title != title {
		t.Errorf("display Title = %q, want unquoted %q", entries[0].Title, title)
	}
}

func TestNewQuotesSupersedes(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := adr.New(dir, "first", 0); err != nil {
		t.Fatal(err)
	}
	path, err := adr.New(dir, "second", 1)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "\nsupersedes: \"0001\"\n") {
		t.Errorf("supersedes not quoted (octal quirk lives):\n%s", raw)
	}
}

func TestNewBackslashTitleRoundtrip(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	title := `back\slash and "quotes" and colon: all at once`
	if _, err := adr.New(dir, title, 0); err != nil {
		t.Fatal(err)
	}
	entries, err := adr.List(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
	if entries[0].Title != title {
		t.Errorf("roundtrip Title = %q, want %q", entries[0].Title, title)
	}
}

func TestLegacyUnquotedTitleListsVerbatim(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := `---
id: "0001"
title: legacy: unquoted title
status: Accepted
date: 2026-01-15
---

# 0001: legacy: unquoted title
`
	if err := os.WriteFile(filepath.Join(dir, "docs", "adr", "0001-legacy.md"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := adr.List(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
	if entries[0].Title != "legacy: unquoted title" {
		t.Errorf("legacy Title = %q, want verbatim %q", entries[0].Title, "legacy: unquoted title")
	}
}
