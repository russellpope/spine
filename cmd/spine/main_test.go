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

func TestHelpAndDashHShowUsageOnStdout(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"-h"}} {
		code, out, _ := runCmd(t, args...)
		if code != 0 || !strings.Contains(out, "usage: spine") {
			t.Errorf("run(%v): code=%d out=%q", args, code, out)
		}
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
	if code != 0 || !strings.Contains(out, "spine template generation 2") {
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
	if code != 1 || !strings.Contains(out, "+ template_version: 2") {
		t.Fatalf("dry-run code=%d out=%q", code, out)
	}
	// also remove a simple machine-owned file entirely, so --write must
	// report it as created: (missing on disk), not updated:
	adrReadme := filepath.Join(dir, "docs", "adr", "README.md")
	if err := os.Remove(adrReadme); err != nil {
		t.Fatal(err)
	}
	code, out, _ = runCmd(t, "update", "--dir", dir, "--write")
	if code != 0 || !strings.Contains(out, "updated: WORKFLOW.md") {
		t.Fatalf("write code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "created: docs/adr/README.md") {
		t.Errorf("want created: docs/adr/README.md in --write output, out=%q", out)
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

func TestADRListJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Empty ledger must encode as [], never null — a regression to
	// `var out []entryJSON` would emit "null" and fail this.
	code, out, _ := runCmd(t, "adr", "list", "--dir", dir, "--json")
	if code != 0 || strings.TrimSpace(out) != "[]" {
		t.Fatalf("empty ledger: code=%d out=%q, want []", code, out)
	}
	if code, _, errs := runCmd(t, "adr", "new", "--dir", dir, "Some Decision"); code != 0 {
		t.Fatal(errs)
	}
	code, out, _ = runCmd(t, "adr", "list", "--dir", dir, "--json")
	if code != 0 || !strings.Contains(out, `"title":"Some Decision"`) || !strings.Contains(out, `"has_front_matter":true`) {
		t.Fatalf("code=%d out=%q", code, out)
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

func TestHandoffEndToEnd(t *testing.T) {
	dir := t.TempDir()
	code, out, errs := runCmd(t, "handoff", "new", "--dir", dir, "spine v2 wrap")
	if code != 0 || !strings.Contains(out, "-spine-v2-wrap.md") {
		t.Fatalf("new: code=%d out=%q err=%q", code, out, errs)
	}
	code, out, _ = runCmd(t, "handoff", "list", "--dir", dir)
	if code != 0 || !strings.Contains(out, "spine-v2-wrap") {
		t.Fatalf("list: code=%d out=%q", code, out)
	}
	code, out, _ = runCmd(t, "handoff", "latest", "--dir", dir)
	if code != 0 || !strings.HasSuffix(strings.TrimSpace(out), "-spine-v2-wrap.md") {
		t.Fatalf("latest: code=%d out=%q", code, out)
	}
	code, out, _ = runCmd(t, "handoff", "latest", "--dir", dir, "--json")
	if code != 0 || !strings.Contains(out, `"topic":"spine-v2-wrap"`) {
		t.Fatalf("latest --json: code=%d out=%q", code, out)
	}
	code, _, _ = runCmd(t, "handoff", "latest", "--dir", t.TempDir())
	if code != 1 {
		t.Fatalf("latest on empty repo: want exit 1, got %d", code)
	}
}

func TestEvalEndToEnd(t *testing.T) {
	dir := t.TempDir()
	code, out, errs := runCmd(t, "eval", "new", "--dir", dir, "govmomi cli")
	if code != 0 || !strings.Contains(out, "-govmomi-cli") {
		t.Fatalf("new: code=%d out=%q err=%q", code, out, errs)
	}
	code, out, errs = runCmd(t, "eval", "add-run", "--dir", dir, "--eval", "govmomi-cli", "--name", "qwen-3.6-27b")
	if code != 0 || !strings.Contains(out, "qwen-3.6-27b.md") {
		t.Fatalf("add-run: code=%d out=%q err=%q", code, out, errs)
	}
	code, out, _ = runCmd(t, "eval", "list", "--dir", dir)
	if code != 0 || !strings.Contains(out, "qwen-3.6-27b") {
		t.Fatalf("list: code=%d out=%q", code, out)
	}
	code, out, _ = runCmd(t, "eval", "list", "--dir", dir, "--json")
	if code != 0 || !strings.Contains(out, `"name":"qwen-3.6-27b"`) {
		t.Fatalf("list --json: code=%d out=%q", code, out)
	}
	code, _, errs = runCmd(t, "eval", "add-run", "--dir", dir, "--eval", "nope", "--name", "m")
	if code != 2 || !strings.Contains(errs, "no eval matches") {
		t.Fatalf("code=%d errs=%q", code, errs)
	}
}

func TestHandoffFleet(t *testing.T) {
	parent := t.TempDir()
	repo := filepath.Join(parent, "demo")
	if err := os.MkdirAll(filepath.Join(repo, "docs", "handoffs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "docs", "handoffs", "2026-07-01-x.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runCmd(t, "handoff", "latest", "--fleet", parent)
	if code != 0 || !strings.Contains(out, "demo") {
		t.Fatalf("code=%d out=%q", code, out)
	}
	code, _, _ = runCmd(t, "handoff", "latest", "--fleet", filepath.Join(parent, "nope"))
	if code != 2 {
		t.Fatalf("want 2, got %d", code)
	}
}

func TestAdoptEndToEnd(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("## Invariants\n- keep me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runCmd(t, "adopt", "--dir", dir)
	if code != 1 || !strings.Contains(out, "profile: go-service") || !strings.Contains(out, "WORKFLOW.md") {
		t.Fatalf("dry-run: code=%d out=%q", code, out)
	}
	code, out, errs := runCmd(t, "adopt", "--dir", dir, "--write")
	if code != 0 {
		t.Fatalf("write: code=%d out=%q err=%q", code, out, errs)
	}
	code, _, _ = runCmd(t, "adopt", "--dir", dir)
	if code != 0 {
		t.Fatalf("idempotency: want 0, got %d", code)
	}
	code, _, _ = runCmd(t, "doctor", "--dir", dir)
	if code != 0 {
		t.Fatalf("doctor after adopt: want 0, got %d", code)
	}
	code, out, _ = runCmd(t, "adopt", "--dir", dir, "--json")
	if code != 0 || !strings.Contains(out, `"profile":"go-service"`) {
		t.Fatalf("json: code=%d out=%q", code, out)
	}
}

// C1: adopt reports a hand-authored docs/adr/README.md as "preserve" (text
// and JSON), with an info line, rather than warning or destroying it.
func TestAdoptPreservedADRReadmeCmd(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	handAuthored := "# Architecture Decision Records\n\nSee the index below.\n\n| # | Decision |\n|---|---|\n| 0001 | Something |\n"
	if err := os.WriteFile(filepath.Join(dir, "docs", "adr", "README.md"), []byte(handAuthored), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runCmd(t, "adopt", "--dir", dir)
	if !strings.Contains(out, "preserve") || !strings.Contains(out, "docs/adr/README.md") {
		t.Fatalf("text: code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "preserved") {
		t.Fatalf("text missing preserved info line: out=%q", out)
	}
	_, out, _ = runCmd(t, "adopt", "--dir", dir, "--json")
	if !strings.Contains(out, `"action":"preserve"`) {
		t.Fatalf("json missing preserve action: out=%q", out)
	}
}
