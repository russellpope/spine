package doctor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/russellpope/spine/internal/adr"
	"github.com/russellpope/spine/internal/doctor"
	"github.com/russellpope/spine/internal/scaffold"
	"github.com/russellpope/spine/internal/tmpl"
)

func ids(fs []doctor.Finding) map[string]int {
	m := map[string]int{}
	for _, f := range fs {
		m[f.ID]++
	}
	return m
}

func TestCleanScaffoldNoFindings(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Fatalf("want clean, got %#v", fs)
	}
}

func TestMissingPiecesD1(t *testing.T) {
	fs, err := doctor.Run(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if ids(fs)["D1"] == 0 {
		t.Fatalf("want D1 findings, got %#v", fs)
	}
}

func TestStaleGen0D2AndD3(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	// regress to a TRUE gen0 repo by rendering the gen0 templates (stripping
	// the stamp from a current file would read as unrecognized edits instead)
	vals := tmpl.Values{Project: "demo", Profile: "rust",
		Reviewers: "rust-reviewer, security-review", Harness: "cli", Version: 1}
	for tmplName, rel := range map[string]string{
		"WORKFLOW.md.tmpl":     "WORKFLOW.md",
		"CLAUDE.md.tmpl":       "CLAUDE.md",
		"harness-interface.md": filepath.Join("docs", "harness-interface.md"),
	} {
		gen0, err := tmpl.Render("gen0", tmplName, vals)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(gen0), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := ids(fs)
	if got["D2"] == 0 || got["D3"] == 0 {
		t.Fatalf("want D2 (stale, pending update) + D3 (no markers), got %#v", fs)
	}
}

func TestSuperpowersDriftD5(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	sp := filepath.Join(dir, "docs", "superpowers", "plans")
	os.MkdirAll(sp, 0o755)
	os.WriteFile(filepath.Join(sp, "old-plan.md"), []byte("x"), 0o644)
	fs, _ := doctor.Run(dir)
	if ids(fs)["D5"] != 1 {
		t.Fatalf("want one D5, got %#v", fs)
	}
}

func TestUnrecognizedEditsD4(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	wf := filepath.Join(dir, "WORKFLOW.md")
	raw, err := os.ReadFile(wf)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wf, append(raw, []byte("custom_rule: never deploy fridays\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ids(fs)["D4"] == 0 {
		t.Fatalf("want D4 finding for unrecognized edit, got %#v", fs)
	}
}

func TestLegacyADRNoFrontMatterD6Info(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join("testdata", "legacy-adr.md"))
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "docs", "adr", "0001-legacy.md")
	if err := os.WriteFile(dst, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, f := range fs {
		if f.ID != "D6" || f.Path != dst {
			continue
		}
		found = true
		if f.Severity != "info" {
			t.Errorf("severity = %q, want info", f.Severity)
		}
	}
	if !found {
		t.Fatalf("want D6 finding for legacy (no front matter) ADR, got %#v", fs)
	}
}

func TestADRProblemsD6(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	adr.New(dir, "Real one", 0)
	// duplicate number + bogus status
	os.WriteFile(filepath.Join(dir, "docs", "adr", "0001-dupe.md"),
		[]byte("---\nid: 0001\ntitle: Dupe\nstatus: Draft\ndate: 2026-07-01\n---\n"), 0o644)
	fs, _ := doctor.Run(dir)
	got := ids(fs)
	if got["D6"] < 2 {
		t.Fatalf("want duplicate+status D6 findings, got %#v", fs)
	}
}
