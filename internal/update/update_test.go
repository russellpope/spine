package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/scaffold"
)

const gen0HbmviewClaude = `# hbmview

Uses the **unified workflow** — see ` + "`WORKFLOW.md`" + ` for the active profile (` + "`rust`" + `) and stages.

- Specs / PRDs -> ` + "`docs/specs/`" + `
- Decisions (ADRs) -> ` + "`docs/adr/`" + `
- Issue / bug ledger -> ` + "`docs/issues/`" + ` (dependency convention in ` + "`docs/issues/README.md`" + `)
- Handoffs -> ` + "`docs/handoffs/`" + `

**Mandatory gates:** a PRD up front (run ` + "`/grill-with-docs`" + ` -> ` + "`/to-prd`" + `) and verification before completion.
**Model:** see ` + "`WORKFLOW.md`" + ` ` + "`model_default`" + ` (swappable).
`

func writeRepo(t *testing.T, workflow, claude string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if workflow != "" {
		if err := os.WriteFile(filepath.Join(dir, "WORKFLOW.md"), []byte(workflow), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if claude != "" {
		if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(claude), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func report(t *testing.T, reports []FileReport, path string) FileReport {
	t.Helper()
	for _, r := range reports {
		if r.Path == path {
			return r
		}
	}
	t.Fatalf("no report for %s in %#v", path, reports)
	return FileReport{}
}

func TestDiffEmptyWhenEqual(t *testing.T) {
	if d := Diff("x", "a\nb", "a\nb"); d != "" {
		t.Errorf("got %q", d)
	}
	d := Diff("x", "a\nb\nc", "a\nB\nc")
	if !strings.Contains(d, "- b") || !strings.Contains(d, "+ B") || !strings.Contains(d, "  a") {
		t.Errorf("diff:\n%s", d)
	}
}

func TestFreshInitIsUpToDate(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.State != UpToDate {
			t.Errorf("%s: state=%v diff:\n%s", r.Path, r.State, r.Diff)
		}
	}
}

func TestGen0CleanClaim(t *testing.T) {
	dir := writeRepo(t, gen0Hbmview, gen0HbmviewClaude)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != Pending {
		t.Fatalf("WORKFLOW state=%v unrec=%v", wf.State, wf.Unrecognized)
	}
	for _, want := range []string{"template_version: 1", "primary: claude-fable-5",
		"model_default: claude-fable-5", "profile: rust", "functional_harness: cli"} {
		if !strings.Contains(wf.newContent, want) {
			t.Errorf("regenerated WORKFLOW missing %q", want)
		}
	}
	if strings.Contains(wf.newContent, "model_default: claude-opus-4-8") {
		t.Error("stale gen0 default survived regeneration")
	}
	cl := report(t, reports, "CLAUDE.md")
	if cl.State != Pending {
		t.Fatalf("CLAUDE state=%v", cl.State)
	}
	if !strings.HasPrefix(cl.newContent, "<!-- spine:begin v1 -->") {
		t.Error("claimed CLAUDE.md lacks markers")
	}
	if got := strings.Count(cl.newContent, "# hbmview"); got != 1 {
		t.Errorf("clean claim duplicated content, %d title lines", got)
	}
}

func TestGen0WriteThenUpToDate(t *testing.T) {
	dir := writeRepo(t, gen0Hbmview, gen0HbmviewClaude)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.State != UpToDate {
			t.Errorf("after write, %s state=%v diff:\n%s", r.Path, r.State, r.Diff)
		}
	}
}

func TestUnrecognizedEditsSkipUnlessForce(t *testing.T) {
	custom := gen0Hbmview + "custom_rule: never deploy on fridays\n"
	dir := writeRepo(t, custom, gen0HbmviewClaude)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != SkippedUnrecognized || len(wf.Unrecognized) != 1 ||
		!strings.Contains(wf.Unrecognized[0], "custom_rule") {
		t.Fatalf("state=%v unrec=%v", wf.State, wf.Unrecognized)
	}
	// force regenerates (dropping the line) and write applies it
	if _, err := Run(Options{Dir: dir, Write: true, Force: true}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if strings.Contains(string(got), "custom_rule") {
		t.Error("force did not drop unrecognized line")
	}
	if !strings.Contains(string(got), "template_version: 1") {
		t.Error("force did not regenerate")
	}
}

func TestCustomChoiceSurvivesUpdate(t *testing.T) {
	custom := strings.Replace(gen0Hbmview, "functional_harness: cli", "functional_harness: rest", 1)
	dir := writeRepo(t, custom, gen0HbmviewClaude)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != Pending {
		t.Fatalf("state=%v unrec=%v", wf.State, wf.Unrecognized)
	}
	if !strings.Contains(wf.newContent, "functional_harness: rest") {
		t.Error("user harness choice lost")
	}
}

func TestLegacyClaudeWithUserContentPreserved(t *testing.T) {
	userClaude := gen0HbmviewClaude + "\n## Local invariants\n\n- verify with `lms ps --json`\n"
	dir := writeRepo(t, gen0Hbmview, userClaude)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	cl := report(t, reports, "CLAUDE.md")
	if cl.State != Pending {
		t.Fatalf("state=%v", cl.State)
	}
	if !strings.Contains(cl.newContent, "lms ps --json") {
		t.Error("user content dropped")
	}
	if !strings.HasPrefix(cl.newContent, "<!-- spine:begin") {
		t.Error("markers not inserted at top")
	}
}

func TestMarkerBlockReplacedUserTailKept(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "CLAUDE.md")
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, append(raw, []byte("\n## Notes\n\n- remote is github\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	cl := report(t, reports, "CLAUDE.md")
	if cl.State != UpToDate {
		t.Fatalf("same-version block should be up-to-date, state=%v diff:\n%s", cl.State, cl.Diff)
	}
}

func TestUnbalancedMarkersSkipped(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "CLAUDE.md")
	raw, _ := os.ReadFile(path)
	broken := strings.Replace(string(raw), "<!-- spine:end -->", "", 1)
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	cl := report(t, reports, "CLAUDE.md")
	if cl.State != SkippedUnrecognized {
		t.Fatalf("unbalanced markers must skip even with force, state=%v", cl.State)
	}
}

func TestMissingWorkflowIsHardError(t *testing.T) {
	if _, err := Run(Options{Dir: t.TempDir()}); err == nil {
		t.Fatal("want error when WORKFLOW.md missing")
	}
}
