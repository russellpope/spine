package adr_test

import (
	"os"
	"path/filepath"
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
	for _, want := range []string{"id: 0001", "title: Go with stdlib only", "status: Accepted",
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
	if !strings.Contains(string(raw2), "supersedes: 0001") {
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
