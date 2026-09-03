package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/scaffold"
)

// I128 item 2: the plan prints a retired-override migration on its own
// line, distinct from an inherited refresh and from a preserved override,
// and the printed retired-model remedy (spine update --write) then leaves a
// validating row.
func TestUpdatePlanPrintsRetiredOverrideMigration(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	wfPath := filepath.Join(dir, "WORKFLOW.md")
	raw, err := os.ReadFile(wfPath)
	if err != nil {
		t.Fatal(err)
	}
	content := pinRow(t, string(raw), "claude.primary", "claude-fable-5 @ xhigh")
	if err := os.WriteFile(wfPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := runCmd(t, "model", "--dir", dir, "validate", "claude", "primary"); code != 1 || !strings.Contains(errOut, "retired-model") {
		t.Fatalf("precondition: validate code=%d err=%q, want a retired-model refusal", code, errOut)
	}

	code, out, _ := runCmd(t, "update", "--dir", dir)
	if code != 1 {
		t.Fatalf("dry-run code=%d out=%q", code, out)
	}
	wantLine := "model refresh (retired override): model_routing.claude.primary: claude-fable-5 @ xhigh -> claude-fable-5-1 @ xhigh"
	if !strings.Contains(out, wantLine) {
		t.Errorf("dry-run plan lacks %q, out=%q", wantLine, out)
	}
	if strings.Contains(out, "model override preserved:") || strings.Contains(out, "model refresh (inherited):") {
		t.Errorf("dry-run plan misreports the migration, out=%q", out)
	}

	code, out, _ = runCmd(t, "update", "--dir", dir, "--write")
	if code != 0 || !strings.Contains(out, wantLine) {
		t.Fatalf("write code=%d out=%q", code, out)
	}
	code, stdout, errOut := runCmd(t, "model", "--dir", dir, "validate", "claude", "primary")
	if code != 0 || strings.TrimSpace(stdout) != "claude-fable-5-1" {
		t.Fatalf("validate after the remedy: code=%d out=%q err=%q", code, stdout, errOut)
	}
}
