package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/scaffold"
	"github.com/russellpope/spine/internal/tmpl"
)

const gen0HbmviewClaude = `# hbmview

Uses the **unified workflow** — see ` + "`WORKFLOW.md`" + ` for the active profile (` + "`rust`" + `) and stages.

- Specs / PRDs -> ` + "`docs/specs/`" + `
- Decisions (ADRs) -> ` + "`docs/adr/`" + `
- Issue / bug ledger -> ` + "`docs/issues/`" + ` (dependency convention in ` + "`docs/issues/README.md`" + `)
- Handoffs -> ` + "`docs/handoffs/`" + `

**Mandatory gates:** a PRD up front (grill-with-docs -> to-prd) and verification before completion.
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

// Diff(path, "", content) is a brand-new file: every line is an addition,
// never a spurious "- " row for the phantom empty "old" line.
func TestDiffEmptyOldNoSpuriousMinusRow(t *testing.T) {
	d := Diff("x", "", "a\nb\n")
	for _, line := range strings.Split(d, "\n") {
		if strings.HasPrefix(line, "- ") {
			t.Errorf("unexpected '- ' row in empty-old diff:\n%s", d)
		}
	}
	if !strings.Contains(d, "+ a") || !strings.Contains(d, "+ b") {
		t.Errorf("missing expected '+' rows:\n%s", d)
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
	for _, want := range []string{"template_version: 7", "primary: claude-fable-5",
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
	if !strings.HasPrefix(cl.newContent, "<!-- spine:begin v7 -->") {
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
	if !strings.Contains(string(got), "template_version: 7") {
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

func TestUpdateKnowledgeManifest(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "knowledge", "vault"); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.Path == "docs/harness-interface.md" || r.Path == "docs/issues/README.md" || r.Path == "docs/issues/_template.md" {
			t.Errorf("knowledge update must not manage %s", r.Path)
		}
		if r.State != UpToDate {
			t.Errorf("%s not up-to-date after fresh init", r.Path)
		}
	}
}

func TestUpdateManagesEvalsReadmeOnlyWhenPresent(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.Path == "docs/evals/README.md" {
			t.Fatal("evals README managed without docs/evals/")
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs", "evals"), 0o755); err != nil {
		t.Fatal(err)
	}
	reports, err = Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range reports {
		if r.Path == "docs/evals/README.md" {
			found = true
			if r.State != Pending || !r.Created {
				t.Errorf("want Pending+Created, got state=%v created=%v", r.State, r.Created)
			}
		}
	}
	if !found {
		t.Fatal("evals README not planned despite docs/evals/ existing")
	}
}

func TestAdoptModeSynthesizesWorkflow(t *testing.T) {
	dir := t.TempDir()
	// pre-existing hand-authored CLAUDE.md, praxis-style
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("## Repo invariants\n\n- push with git push github main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir, AdoptProfile: "go-service", AdoptName: "praxis"})
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]FileReport{}
	for _, r := range reports {
		byPath[r.Path] = r
	}
	wf := byPath["WORKFLOW.md"]
	if wf.State != Pending || !wf.Created {
		t.Fatalf("WORKFLOW.md state=%v created=%v", wf.State, wf.Created)
	}
	if !strings.Contains(wf.Diff, "profile: go-service") || !strings.Contains(wf.Diff, "template_version: 7") || !strings.Contains(wf.Diff, "# Workflow — praxis") {
		t.Errorf("diff=%q", wf.Diff)
	}
	cl := byPath["CLAUDE.md"]
	if cl.State != Pending || cl.Created {
		t.Fatalf("CLAUDE.md state=%v created=%v (want claim of existing file)", cl.State, cl.Created)
	}
	if !strings.Contains(cl.Diff, "spine:begin") || !strings.Contains(cl.Diff, "Repo invariants") {
		t.Errorf("claim must insert markers and keep hand content; diff=%q", cl.Diff)
	}
}

func TestMissingWorkflowStillErrorsWithoutAdoptMode(t *testing.T) {
	if _, err := Run(Options{Dir: t.TempDir()}); err == nil {
		t.Fatal("want error")
	}
}

// C1: docs/adr/README.md is the one machine-owned file where unrecognized
// hand-authored content is preserved as-is, not skipped/warned. --force
// remains the explicit opt-in to regenerate it from the template.
func TestLegacyADRReadmePreserved(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "go-service", "demo"); err != nil {
		t.Fatal(err)
	}
	handAuthored := "# Architecture Decision Records\n\nSee the index below.\n\n| # | Decision |\n|---|---|\n| 0001 | Something |\n"
	if err := os.WriteFile(filepath.Join(dir, "docs", "adr", "README.md"), []byte(handAuthored), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, reports, "docs/adr/README.md")
	if r.State != UpToDate || !r.Preserved || r.Diff != "" {
		t.Fatalf("state=%v preserved=%v diff=%q", r.State, r.Preserved, r.Diff)
	}
	// preserved files must not count as outstanding work.
	reports2, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, rr := range reports2 {
		if rr.Path == "docs/adr/README.md" && rr.State != UpToDate {
			t.Errorf("preserved file counted as outstanding: state=%v", rr.State)
		}
	}
	// --force is the explicit opt-in to regenerate from the template.
	forced, err := Run(Options{Dir: dir, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	fr := report(t, forced, "docs/adr/README.md")
	if fr.State != Pending || fr.Preserved {
		t.Fatalf("force: state=%v preserved=%v", fr.State, fr.Preserved)
	}
}

// Absent or template-matched docs/adr/README.md keeps existing behavior:
// create when missing, up-to-date (not preserved) when it already matches
// the template.
func TestFreshInitADRReadmeNotPreserved(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "go-service", "demo"); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, reports, "docs/adr/README.md")
	if r.State != UpToDate || r.Preserved {
		t.Fatalf("state=%v preserved=%v (template-matched file must not be marked preserved)", r.State, r.Preserved)
	}
}

func TestVersionDowngradeGuard(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "WORKFLOW.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	bumped := strings.Replace(string(raw), "template_version: 7", "template_version: 8", 1)
	if bumped == string(raw) {
		t.Fatal("template_version: 7 not found in scaffolded WORKFLOW.md to bump")
	}
	if err := os.WriteFile(path, []byte(bumped), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Run(Options{Dir: dir})
	if err == nil || !strings.Contains(err.Error(), "generation") {
		t.Fatalf("want error mentioning generation, got %v", err)
	}
}

func TestAgentsMdCreatedWhenMissing(t *testing.T) {
	// A repo with WORKFLOW.md + CLAUDE.md but no AGENTS.md (the state every
	// existing gen-6 repo is in) gains AGENTS.md on update.
	dir := writeRepo(t, gen0Hbmview, gen0HbmviewClaude)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	a := report(t, reports, "AGENTS.md")
	if a.State != Pending || !a.Created {
		t.Fatalf("AGENTS.md state=%v created=%v", a.State, a.Created)
	}
	if !strings.Contains(a.Diff, "read by **Codex**") {
		t.Errorf("AGENTS.md diff missing Codex-tuned body:\n%s", a.Diff)
	}
}

func TestAgentsMdMarkerReplacePreservesHandContent(t *testing.T) {
	dir := writeRepo(t, gen0Hbmview, gen0HbmviewClaude)
	// pre-place an AGENTS.md with a stale block + hand-authored tail.
	stale := "<!-- spine:begin v5 -->\nold codex brief\n<!-- spine:end -->\n\n## Local notes\nkeep me\n"
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	_ = reports
	got, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(got)
	if strings.Contains(s, "old codex brief") {
		t.Error("stale block survived marker replacement")
	}
	if !strings.Contains(s, "## Local notes") || !strings.Contains(s, "keep me") {
		t.Error("hand-authored content outside markers was lost")
	}
	if strings.Count(s, "<!-- spine:begin") != 1 {
		t.Error("marker replacement duplicated the block")
	}
}

func TestAgentsMdUnbalancedMarkersFlagged(t *testing.T) {
	dir := writeRepo(t, gen0Hbmview, gen0HbmviewClaude)
	bad := "<!-- spine:begin v6 -->\nbody\n<!-- spine:begin v6 -->\n<!-- spine:end -->\n"
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	a := report(t, reports, "AGENTS.md")
	if len(a.Unrecognized) == 0 {
		t.Fatal("unbalanced markers should be flagged, not clobbered")
	}
	if !strings.Contains(a.Unrecognized[0], "AGENTS.md") {
		t.Errorf("unbalanced-marker message should name AGENTS.md, got %q", a.Unrecognized[0])
	}
}

func TestClaudeAndAgentsBlocksShareVocabulary(t *testing.T) {
	vals := tmpl.Values{Project: "demo", Profile: "go-service", Version: tmpl.Version()}
	claude, err := tmpl.Render("current", "CLAUDE.md.tmpl", vals)
	if err != nil {
		t.Fatal(err)
	}
	agents, err := tmpl.Render("current", "AGENTS.md.tmpl", vals)
	if err != nil {
		t.Fatal(err)
	}
	// The two harness briefs must not drift on the load-bearing vocabulary.
	for _, term := range []string{
		"WORKFLOW.md", "docs/specs/", "docs/adr/", "docs/issues/", "docs/handoffs",
		"primary", "routine", "mechanical", "fallback",
		"verification before completion",
	} {
		if !strings.Contains(claude, term) {
			t.Errorf("CLAUDE.md.tmpl missing shared term %q", term)
		}
		if !strings.Contains(agents, term) {
			t.Errorf("AGENTS.md.tmpl missing shared term %q", term)
		}
	}
}

func TestRunSurfacesEvalsDirStatError(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"WORKFLOW.md", "CLAUDE.md"} {
		raw, err := os.ReadFile(filepath.Join("testdata", "ccq", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	// docs/evals as a symlink loop: Stat fails with ELOOP, not ENOENT.
	if err := os.Symlink("evals", filepath.Join(dir, "docs", "evals")); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(Options{Dir: dir}); err == nil {
		t.Fatal("want Stat error surfaced, got nil (silently skipped evals-README before v3)")
	}
}

// A repo scaffolded at the *previous* generation must, on a plain update,
// gain AGENTS.md and advance its stamp — with no hand-written migration code.
func TestGen6RepoGainsAgentsMdOnUpdate(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "go-service", "demo"); err != nil {
		t.Fatal(err)
	}
	// Simulate a repo stamped at the prior generation.
	wfPath := filepath.Join(dir, "WORKFLOW.md")
	raw, err := os.ReadFile(wfPath)
	if err != nil {
		t.Fatal(err)
	}
	downgraded := strings.Replace(string(raw), "template_version: 7", "template_version: 6", 1)
	if downgraded == string(raw) {
		t.Fatal("could not stage a gen-6 fixture")
	}
	if err := os.WriteFile(wfPath, []byte(downgraded), 0o644); err != nil {
		t.Fatal(err)
	}
	// Remove AGENTS.md so we're truly in the pre-A state.
	if err := os.Remove(filepath.Join(dir, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	agents, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("update did not create AGENTS.md: %v", err)
	}
	if !strings.Contains(string(agents), "<!-- spine:begin v7 -->") {
		t.Error("AGENTS.md not stamped at v7")
	}
	wfAfter, err := os.ReadFile(wfPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(wfAfter), "template_version: 7") {
		t.Error("WORKFLOW.md did not advance to gen 7")
	}
}
