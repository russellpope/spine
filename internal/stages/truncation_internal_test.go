package stages

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// I029: a long missing set must truncate the named ids with a "+N more"
// count rather than dumping every missing id onto one line. This is a
// package-internal test so the fixture can derive its boundary from the
// production cap without exposing a test-only export to other packages.
func TestTickedMissingTruncatesLongMissingSet(t *testing.T) {
	dir := t.TempDir()
	writeTruncationTestFile(t, dir, "WORKFLOW.md", "profile: library-cli\ntemplate_version: 8\nstages: [grill, prd, issues, implement]\n")
	// Only I001 exists; the inclusive range ends two ids after the first, so
	// the missing set is always exactly maxNamedMissingIDs+1.
	rangeEnd := maxNamedMissingIDs + 2
	writeTruncationTestFile(t, dir, "docs/issues/I001-a.md", "---\nid: I001\n---\nx\n")
	writeTruncationTestFile(t, dir, ".superpowers/sdd/progress.md", "<!-- spine:cursor -->\n"+
		fmt.Sprintf("effort: x\nprd: docs/specs/x.md\ntickets: I001-I%03d\nstages: grill[x] prd[x] issues[x] implement[<]\n", rangeEnd)+
		"<!-- /spine:cursor -->\n")
	rep, err := Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	var issues StageRow
	for _, row := range rep.Stages {
		if row.Name == "issues" {
			issues = row
			break
		}
	}
	if issues.Name == "" {
		t.Fatal("no stage row named \"issues\"")
	}
	if issues.Verdict != VerdictTickedMissing {
		t.Fatalf("issues verdict = %s (%s), want ticked-missing", issues.Verdict, issues.Detail)
	}
	if !rep.Blocking() {
		t.Error("want Blocking() true — ticked-missing must still block")
	}
	if !strings.Contains(issues.Detail, "+1 more") {
		t.Errorf("Detail = %q, want the exact cap+1 missing set folded into a \"+1 more\" tail", issues.Detail)
	}
	lastID := fmt.Sprintf("I%03d", rangeEnd)
	if strings.Contains(issues.Detail, lastID) {
		t.Errorf("Detail = %q, want the tail id %s folded into the +1 more count, not named", issues.Detail, lastID)
	}
}

func writeTruncationTestFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
