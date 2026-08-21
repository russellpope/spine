package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// I104 preserves maipipe as the validation authority: a reachable validator
// rejecting the rendered candidate must still refuse before the file is
// written. The fake isolates spine's propagation of maipipe's verdict.
func TestMaipipeValidationRejectsInvalidCandidate(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\necho 'invalid candidate: duplicate stage name' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(dir, "maipipe"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	err := checkMaipipeContent("/repo/"+MaipipeFile, "schema = 0\n")
	if err == nil {
		t.Fatal("maipipe rejection accepted the candidate")
	}
	for _, want := range []string{"maipipe validate rejected the result", "invalid candidate", "duplicate stage name"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal does not include %q:\n%s", want, err)
		}
	}
}

// I104's validator preflight applies to the rendered candidate through Run,
// not just the helper. A rejection aborts the complete write, including other
// pending files, after the plan has surfaced maipipe's verdict.
func TestUpdateRejectingMaipipeLeavesEveryPendingFileUnwritten(t *testing.T) {
	dir := gateRepo(t, "[]", nil)
	mpPath := filepath.Join(dir, MaipipeFile)
	const sentinel = "schema = 0\n# unchanged after validator refusal\n"
	if err := os.WriteFile(mpPath, []byte(sentinel), 0o644); err != nil {
		t.Fatal(err)
	}

	wfPath := filepath.Join(dir, "WORKFLOW.md")
	beforeWorkflow := setKey(readFile(t, wfPath), "template_version", "10")
	if err := os.WriteFile(wfPath, []byte(beforeWorkflow), 0o644); err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	script := "#!/bin/sh\necho 'invalid rendered candidate' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(binDir, "maipipe"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	reports, err := Run(Options{Dir: dir, Write: true})
	if err == nil {
		t.Fatal("validator rejection wrote the update")
	}
	mp := report(t, reports, MaipipeFile)
	if mp.State != Pending || mp.Preflight != maipipeValidatePreflight {
		t.Fatalf("maipipe report = state %v preflight %q, want pending maipipe validation", mp.State, mp.Preflight)
	}
	for _, want := range []string{"maipipe validate rejected the result", "invalid rendered candidate", "no files were written"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal does not include %q:\n%s", want, err)
		}
	}
	if got := readFile(t, mpPath); got != sentinel {
		t.Errorf("maipipe.toml changed after validator refusal:\nwant %q\n got %q", sentinel, got)
	}
	if got := readFile(t, wfPath); got != beforeWorkflow {
		t.Errorf("WORKFLOW.md changed after validator refusal:\nwant %q\n got %q", beforeWorkflow, got)
	}
}
