package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var gen13ContentLines = map[string]bool{
	"DISCARDED <ticket-id> source:<claude|codex> session:<session-id> dispatch:<event-id> tier:<mechanical|routine|primary|fallback> reason: <one line>": true,
	"A record excuses exactly its to-tier, nothing else. Any record not matching the grammar exactly excuses nothing — spaced arrows, missing `reason:`, missing tokens, all of it. A discarded record covers one identified event only: its source, session, dispatch event, and resolved tier must all match an otherwise lower-tier token. Malformed, duplicate, zero-match, or multi-match discarded records are warned and excuse nothing. `discarded-with-reason` remains visible and advisory; a separate lower-tier event still reports blocking `silent-descent`.": true,
}

func isGen13ContentDiffLine(line string) bool {
	if len(line) == 0 || (line[0] != '+' && line[0] != '-') {
		return false
	}
	body := strings.TrimSpace(line[1:])
	if line[0] == '-' {
		return body == "A record excuses exactly its to-tier, nothing else. Any record not matching the grammar exactly excuses nothing — spaced arrows, missing `reason:`, missing tokens, all of it."
	}
	return gen13ContentLines[body]
}

func TestGen12To13PreservesI050AcceptanceContract(t *testing.T) {
	dir := stageGen12Repo(t)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != Pending || len(wf.Unrecognized) != 0 {
		t.Fatalf("WORKFLOW state=%v unrecognized=%v, want clean generation-12 migration", wf.State, wf.Unrecognized)
	}
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	assertFileContains(t, dir, "WORKFLOW.md", "template_version: 14", "DISCARDED <ticket-id>", "## Acceptance exceptions", "APPROVED-UNTESTED")
	raw, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range gen12I050Lines {
		if !strings.Contains(string(raw), line) {
			t.Errorf("generation-13 workflow did not preserve I050 line byte-for-byte: %q", line)
		}
	}
	reports, err = Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.State != UpToDate {
			t.Errorf("second pass %s state=%v diff=%s", r.Path, r.State, r.Diff)
		}
	}
}

func TestGen12To13RetainedI050EditIsUnrecognized(t *testing.T) {
	for _, tc := range []struct{ name, old, new string }{
		{"heading", "## Acceptance exceptions", "## Local acceptance exceptions"},
		{"grammar", "- [ ] <criterion> -- APPROVED-UNTESTED <YYYY-MM-DD> by <approver> ref: <docs/YYYY-MM-DD-artifact.md#anchor> reason: <one-line reason>", "- [ ] <criterion> -- LOCAL-APPROVED-UNTESTED <YYYY-MM-DD> by <approver> ref: <docs/YYYY-MM-DD-artifact.md#anchor> reason: <one-line reason>"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := stageGen12Repo(t)
			path := filepath.Join(dir, "WORKFLOW.md")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			changed := strings.Replace(string(raw), tc.old, tc.new, 1)
			if changed == string(raw) {
				t.Fatalf("generation-12 fixture lacks %q", tc.old)
			}
			if err := os.WriteFile(path, []byte(changed), 0o644); err != nil {
				t.Fatal(err)
			}
			reports, err := Run(Options{Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			wf := report(t, reports, "WORKFLOW.md")
			if wf.State != SkippedUnrecognized || len(wf.Unrecognized) == 0 {
				t.Fatalf("WORKFLOW state=%v unrecognized=%v, want retained I050 local-edit refusal", wf.State, wf.Unrecognized)
			}
		})
	}
}

var gen12I050Lines = []string{
	"## Acceptance exceptions",
	"- [ ] <criterion> -- APPROVED-UNTESTED <YYYY-MM-DD> by <approver> ref: <docs/YYYY-MM-DD-artifact.md#anchor> reason: <one-line reason>",
	"The dated Markdown reference must be a clean repository-relative `docs/` path to a regular file. Spine verifies durable local provenance but does not authenticate the approver or resolve the fragment. A complete record is silent in `spine doctor`; an incomplete or invalid record is a D15 warning. `spine audit stages` scans only cursor-resolved tickets, keeps acceptance warnings nonblocking, and prints `acceptance: approved-untested=<valid-count> invalid=<invalid-count>` when the scoped tickets contain candidates. Ordinary unchecked criteria and tickets with no uppercase candidate keep their existing behavior.",
}

func stageGen12Repo(t *testing.T) string {
	t.Helper()
	dir := stageGen11Repo(t)
	path := filepath.Join(dir, "WORKFLOW.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	workflow := strings.Replace(string(raw), "template_version: 11", "template_version: 12", 1)
	acceptance := "\n## Acceptance exceptions\n\nAn applicable acceptance criterion that was consciously approved without a test stays unchecked and records the decision on one physical line under the ticket's exact column-0 `## Acceptance criteria` heading:\n\n    - [ ] <criterion> -- APPROVED-UNTESTED <YYYY-MM-DD> by <approver> ref: <docs/YYYY-MM-DD-artifact.md#anchor> reason: <one-line reason>\n\nThe dated Markdown reference must be a clean repository-relative `docs/` path to a regular file. Spine verifies durable local provenance but does not authenticate the approver or resolve the fragment. A complete record is silent in `spine doctor`; an incomplete or invalid record is a D15 warning. `spine audit stages` scans only cursor-resolved tickets, keeps acceptance warnings nonblocking, and prints `acceptance: approved-untested=<valid-count> invalid=<invalid-count>` when the scoped tickets contain candidates. Ordinary unchecked criteria and tickets with no uppercase candidate keep their existing behavior.\n"
	workflow = strings.Replace(workflow, "\nReviewer floor:", acceptance+"\nReviewer floor:", 1)
	if err := os.WriteFile(path, []byte(workflow), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}
