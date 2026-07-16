package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// End-to-end against copies of hbmview's REAL stranded files (gen0, pristine).
func TestHbmviewUnstranding(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	copyFixture := func(src, dst string) {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join("testdata", "hbmview", src))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, dst), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	copyFixture("WORKFLOW.md", "WORKFLOW.md")
	copyFixture("CLAUDE.md", "CLAUDE.md")
	copyFixture("harness-interface.md", filepath.Join("docs", "harness-interface.md"))

	// dry run: everything claimable, nothing skipped
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.State == SkippedUnrecognized {
			t.Fatalf("%s skipped: %v", r.Path, r.Unrecognized)
		}
	}

	// write, then verify the outcome
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	wf, _ := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	for _, want := range []string{"# Workflow — hbmview", "profile: rust", "template_version: 8",
		"primary: claude-fable-5", "model_default: claude-fable-5",
		"reviewers: [rust-reviewer, security-review]", "functional_harness: cli",
		"## Execution modes"} {
		if !strings.Contains(string(wf), want) {
			t.Errorf("WORKFLOW.md missing %q", want)
		}
	}
	if strings.Contains(string(wf), "model_default: claude-opus-4-8") {
		t.Error("stale model_default survived")
	}
	cl, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if !strings.HasPrefix(string(cl), "<!-- spine:begin v8 -->") ||
		strings.Count(string(cl), "# hbmview") != 1 {
		t.Errorf("CLAUDE.md claim wrong:\n%s", cl)
	}
	hi, _ := os.ReadFile(filepath.Join(dir, "docs", "harness-interface.md"))
	if !strings.Contains(string(hi), "fresh-context") {
		t.Error("harness-interface.md not upgraded to current generation")
	}

	// idempotence: second run is all up-to-date
	reports, err = Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.State != UpToDate {
			t.Errorf("second pass %s state=%v diff:\n%s", r.Path, r.State, r.Diff)
		}
	}
}
