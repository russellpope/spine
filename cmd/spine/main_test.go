package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/tmpl"
)

func runCmd(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestNoArgsShowsUsage(t *testing.T) {
	code, _, errs := runCmd(t)
	if code != 2 || !strings.Contains(errs, "usage: spine") {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
}

func TestUnknownCommand(t *testing.T) {
	code, _, errs := runCmd(t, "bogus")
	if code != 2 || !strings.Contains(errs, "unknown command") {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
}

func TestVersionCommand(t *testing.T) {
	code, out, _ := runCmd(t, "version")
	if code != 0 || !strings.Contains(out, "1") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestInitEndToEnd(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errs := runCmd(t, "init", "--dir", dir, "--name", "demo")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if !strings.Contains(out, "create: WORKFLOW.md") || !strings.Contains(out, "done: rust") {
		t.Errorf("out=%q", out)
	}
}

func TestInitUndetectableNeedsProfile(t *testing.T) {
	code, _, errs := runCmd(t, "init", "--dir", t.TempDir())
	if code != 2 || !strings.Contains(errs, "--profile") {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
}

func TestUpdateDryRunThenWrite(t *testing.T) {
	dir := t.TempDir()
	if code, _, errs := runCmd(t, "init", "--dir", dir, "--profile", "rust", "--name", "demo"); code != 0 {
		t.Fatal(errs)
	}
	// fresh scaffold: nothing pending
	code, out, _ := runCmd(t, "update", "--dir", dir)
	if code != 0 || !strings.Contains(out, "up-to-date") {
		t.Fatalf("code=%d out=%q", code, out)
	}
	// regress the repo to a TRUE gen0 state (rendering gen0 templates) —
	// merely deleting the stamp line would leave current-only lines that
	// read as unrecognized edits against gen0, i.e. Skipped, not Pending.
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
	code, out, _ = runCmd(t, "update", "--dir", dir)
	if code != 1 || !strings.Contains(out, "+ template_version: 1") {
		t.Fatalf("dry-run code=%d out=%q", code, out)
	}
	code, out, _ = runCmd(t, "update", "--dir", dir, "--write")
	if code != 0 || !strings.Contains(out, "updated: WORKFLOW.md") {
		t.Fatalf("write code=%d out=%q", code, out)
	}
	code, _, _ = runCmd(t, "update", "--dir", dir)
	if code != 0 {
		t.Fatalf("after write, code=%d", code)
	}
}

func TestUpdateMissingWorkflowExits2(t *testing.T) {
	code, _, errs := runCmd(t, "update", "--dir", t.TempDir())
	if code != 2 || !strings.Contains(errs, "spine init") {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
}

func TestADRNewAndList(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	code, out, errs := runCmd(t, "adr", "new", "--dir", dir, "Go with stdlib only")
	if code != 0 || !strings.Contains(out, "0001-go-with-stdlib-only.md") {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
	code, out, _ = runCmd(t, "adr", "list", "--dir", dir)
	if code != 0 || !strings.Contains(out, "0001  Accepted") {
		t.Fatalf("list code=%d out=%q", code, out)
	}
	code, _, errs = runCmd(t, "adr", "new", "--dir", dir, "--supersedes", "9", "X")
	if code != 2 || !strings.Contains(errs, "not found") {
		t.Fatalf("code=%d err=%q", code, errs)
	}
}

func TestDoctorInfoOnlyExitsZero(t *testing.T) {
	dir := t.TempDir()
	if code, _, errs := runCmd(t, "init", "--dir", dir, "--profile", "rust", "--name", "demo"); code != 0 {
		t.Fatal(errs)
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "internal", "doctor", "testdata", "legacy-adr.md"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "adr", "0001-legacy.md"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runCmd(t, "doctor", "--dir", dir)
	if code != 0 {
		t.Fatalf("want exit 0 for info-only findings, code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "D6") || !strings.Contains(out, "info") {
		t.Errorf("want D6 info finding printed, out=%q", out)
	}
}

func TestDoctorCleanAndJSON(t *testing.T) {
	dir := t.TempDir()
	runCmd(t, "init", "--dir", dir, "--profile", "rust", "--name", "demo")
	code, out, _ := runCmd(t, "doctor", "--dir", dir)
	if code != 0 || !strings.Contains(out, "ok") {
		t.Fatalf("code=%d out=%q", code, out)
	}
	code, out, _ = runCmd(t, "doctor", "--dir", dir, "--json")
	if code != 0 || !strings.Contains(out, `"findings":[]`) {
		t.Fatalf("json code=%d out=%q", code, out)
	}
	code, out, _ = runCmd(t, "doctor", "--dir", t.TempDir())
	if code != 1 || !strings.Contains(out, "D1") {
		t.Fatalf("empty-dir code=%d out=%q", code, out)
	}
}
