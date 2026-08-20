package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/russellpope/spine/internal/handoff"
	"github.com/russellpope/spine/internal/tmpl"
)

// TestMain scrubs the gate stage's environment before any test runs. spine's
// own suite executes under the go@1 pack's `fast/test` stage, which exports
// MAIPIPE_RESULTS and the SPINE_GATE_* config; the gate tests below drive
// results-emitting commands, so an inherited MAIPIPE_RESULTS makes them write
// their fixture findings into the stage's real results file and fail the run.
// Tests that need either variable set it themselves with t.Setenv.
func TestMain(m *testing.M) {
	scrubStageEnv()
	os.Exit(m.Run())
}

// scrubStageEnv removes the gate stage's own variables from this process.
func scrubStageEnv() {
	os.Unsetenv("MAIPIPE_RESULTS")
	for _, kv := range os.Environ() {
		if k, _, ok := strings.Cut(kv, "="); ok && strings.HasPrefix(k, "SPINE_GATE_") {
			os.Unsetenv(k)
		}
	}
}

// TestScrubStageEnvRemovesStageVars is the unit half of the scrub: both
// families go, nothing else is touched.
func TestScrubStageEnvRemovesStageVars(t *testing.T) {
	t.Setenv("MAIPIPE_RESULTS", filepath.Join(t.TempDir(), "results.json"))
	t.Setenv("SPINE_GATE_N_PLUS_ONE_CLIENTS", "Query")
	t.Setenv("SPINE_KEEP", "keep")
	scrubStageEnv()
	if v := os.Getenv("MAIPIPE_RESULTS"); v != "" {
		t.Errorf("MAIPIPE_RESULTS=%q", v)
	}
	if v := os.Getenv("SPINE_GATE_N_PLUS_ONE_CLIENTS"); v != "" {
		t.Errorf("SPINE_GATE_N_PLUS_ONE_CLIENTS=%q", v)
	}
	if v := os.Getenv("SPINE_KEEP"); v != "keep" {
		t.Errorf("unrelated variable scrubbed: SPINE_KEEP=%q", v)
	}
}

// TestSuiteWritesNoResultsFileUnderAStage is the end-to-end control for the
// scrub: this package's own suite, run the way the `fast/test` stage runs it
// (MAIPIPE_RESULTS exported), must leave that path untouched. The child runs
// the seeded gate test that leaked before — it emits two findings and, without
// TestMain's scrub, writes them into the stage's results file, turning a green
// `go test ./...` into a failed gate stage. The -run filter excludes this test,
// so the child does not recurse.
func TestSuiteWritesNoResultsFileUnderAStage(t *testing.T) {
	results := filepath.Join(t.TempDir(), "results.json")
	cmd := exec.Command("go", "test", "-count=1",
		"-run", "^TestGateSyntacticClassesTolerateTestdata$", ".")
	cmd.Env = append(os.Environ(), "MAIPIPE_RESULTS="+results)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("child suite failed: %v\n%s", err, out)
	}
	if _, err := os.Stat(results); !os.IsNotExist(err) {
		got, _ := os.ReadFile(results)
		t.Fatalf("stage results path written by the tree's own suite: err=%v contents=%s", err, got)
	}
}

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
	if code != 0 || !strings.Contains(out, "spine template generation 11") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestAgeDaysIsCalendarLocal(t *testing.T) {
	defer func() { now = time.Now }()
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	// 17:00 PDT on 2026-07-03 == 2026-07-04 00:00 UTC: the old
	// hours/24-since-UTC-midnight math reported a today-dated handoff as 1d.
	now = func() time.Time { return time.Date(2026, 7, 3, 17, 0, 0, 0, la) }
	cases := []struct {
		filenameDate string
		want         int
	}{
		{"2026-07-03", 0}, // today — the observed off-by-one
		{"2026-07-02", 1}, // yesterday
		{"2026-06-26", 7},
		{"2026-07-04", 0}, // future-dated clamps to 0
	}
	for _, c := range cases {
		d, err := time.Parse("2006-01-02", c.filenameDate)
		if err != nil {
			t.Fatal(err)
		}
		if got := ageDays(d); got != c.want {
			t.Errorf("ageDays(%s) = %d, want %d", c.filenameDate, got, c.want)
		}
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
	if code != 1 || !strings.Contains(out, "+ template_version: 11") {
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

// I035: an inherited stale model default is itemized in the update plan —
// named with old value, new value, and inherited provenance, distinct from
// the content diff — and a preserved override is reported as such.
func TestUpdateItemizesModelRefreshAndOverride(t *testing.T) {
	dir := t.TempDir()
	if code, _, errs := runCmd(t, "init", "--dir", dir, "--profile", "rust", "--name", "demo"); code != 0 {
		t.Fatal(errs)
	}
	wfPath := filepath.Join(dir, "WORKFLOW.md")
	raw, err := os.ReadFile(wfPath)
	if err != nil {
		t.Fatal(err)
	}
	// Regress the claude fallback to the previous shipped default (inherited)
	// and pin the claude routine to a value no default ever shipped
	// (override) — value-only replacements, so the dotted mirror rows'
	// alignment padding (I036) is irrelevant.
	content := strings.Replace(string(raw), "claude.fallback:   claude-opus-5", "claude.fallback:   claude-opus-4-8", 1)
	content = strings.Replace(content, "claude.routine:    claude-opus-5 @ low", "claude.routine:    local-llama-70b", 1)
	if content == string(raw) {
		t.Fatal("could not stage fallback/routine values in scaffolded WORKFLOW.md")
	}
	if err := os.WriteFile(wfPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out, _ := runCmd(t, "update", "--dir", dir)
	if code != 1 {
		t.Fatalf("dry-run code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "model refresh (inherited): model_routing.claude.fallback: claude-opus-4-8 -> claude-opus-5") {
		t.Errorf("dry-run plan missing itemized refresh, out=%q", out)
	}
	if !strings.Contains(out, "model override preserved: model_routing.claude.routine: local-llama-70b") {
		t.Errorf("dry-run plan missing override report, out=%q", out)
	}

	code, out, _ = runCmd(t, "update", "--dir", dir, "--write")
	if code != 0 || !strings.Contains(out, "model refresh (inherited): model_routing.claude.fallback: claude-opus-4-8 -> claude-opus-5") {
		t.Fatalf("write code=%d out=%q", code, out)
	}
	got, err := os.ReadFile(wfPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "claude-opus-5") || strings.Contains(string(got), "claude-opus-4-8") {
		t.Errorf("written WORKFLOW.md not refreshed:\n%s", got)
	}
	if !strings.Contains(string(got), "routine: local-llama-70b") {
		t.Errorf("override lost on write:\n%s", got)
	}
}

// I036 review Important 2: an override minted by the effort: migration is
// announced as created, never as "preserved" — the itemized lines are the
// net a sweep reviewer trusts, so a just-created value must not read as
// pre-existing. A genuinely pre-existing override keeps the preserved
// wording (asserted by TestUpdateItemizesModelRefreshAndOverride above).
func TestUpdateAnnouncesMigratedEffortOverridesAsCreated(t *testing.T) {
	dir := t.TempDir()
	if code, _, errs := runCmd(t, "init", "--dir", dir, "--profile", "rust", "--name", "demo"); code != 0 {
		t.Fatal(errs)
	}
	wfPath := filepath.Join(dir, "WORKFLOW.md")
	raw, err := os.ReadFile(wfPath)
	if err != nil {
		t.Fatal(err)
	}
	// Reintroduce a customized retired effort: key above stages:, gen-9 style.
	content := strings.Replace(string(raw), "\nstages:", "\neffort: xhigh\nstages:", 1)
	if content == string(raw) {
		t.Fatal("could not stage a top-level effort: key in scaffolded WORKFLOW.md")
	}
	if err := os.WriteFile(wfPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runCmd(t, "update", "--dir", dir)
	if code != 1 {
		t.Fatalf("dry-run code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "model override created (migrated from retired effort:): model_routing.claude.primary: claude-fable-5 @ xhigh") {
		t.Errorf("plan missing created-wording for the minted override, out=%q", out)
	}
	if strings.Contains(out, "model override preserved:") {
		t.Errorf("a migration-minted override announced as preserved, out=%q", out)
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
	if !strings.Contains(out, "note: no spine cursor found") {
		t.Fatalf("new without cursor must explain the omitted block: out=%q", out)
	}
	path := strings.Split(strings.TrimSpace(out), "\n")[0]
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "<!-- spine:cursor -->") {
		t.Fatalf("new without cursor must not add a cursor block:\n%s", raw)
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

func TestHandoffNewEmbedsCurrentCursorSnapshot(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "WORKFLOW.md"), []byte("stages: [grill, prd]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".superpowers", "sdd"), 0o755); err != nil {
		t.Fatal(err)
	}
	block := "<!-- spine:cursor -->\n" +
		"effort: handoff-embed\n" +
		"prd: docs/specs/2026-08-06-handoff-embed-design.md\n" +
		"tickets: \n" +
		"stages: grill[x] prd[<]\n" +
		"<!-- /spine:cursor -->\n"
	if err := os.WriteFile(filepath.Join(dir, ".superpowers", "sdd", "progress.md"), []byte(block), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out, errs := runCmd(t, "handoff", "new", "--dir", dir, "cursor snapshot")
	if code != 0 || errs != "" {
		t.Fatalf("new: code=%d out=%q err=%q", code, out, errs)
	}
	if strings.Contains(out, "note:") {
		t.Fatalf("new with a cursor must not report it missing: out=%q", out)
	}
	path := strings.TrimSpace(out)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), block) {
		t.Fatalf("created handoff must embed the current cursor block verbatim:\n%s", raw)
	}
	code, auditOut, auditErr := runCmd(t, "audit", "stages", "--dir", dir)
	if code != 0 {
		t.Fatalf("embedded snapshot must leave the pair fresh: code=%d out=%q err=%q", code, auditOut, auditErr)
	}
}

// I062: both the public latest command and the stages newest-handoff
// backstop must select the handoff that was created later on the same day,
// even when its filename sorts earlier.
func TestSameDateHandoffOrdinalDrivesLatestAndStagesAudit(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "WORKFLOW.md"), []byte("stages: [grill]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	block := "<!-- spine:cursor -->\n" +
		"effort: i062\n" +
		"prd: \n" +
		"tickets: I062\n" +
		"stages: grill[<]\n" +
		"<!-- /spine:cursor -->\n"
	ledger := filepath.Join(dir, ".superpowers", "sdd", "progress.md")
	if err := os.MkdirAll(filepath.Dir(ledger), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ledger, []byte(block), 0o644); err != nil {
		t.Fatal(err)
	}
	hdir := filepath.Join(dir, "docs", "handoffs")
	if err := os.MkdirAll(hdir, 0o755); err != nil {
		t.Fatal(err)
	}
	older := filepath.Join(hdir, "2026-08-09-zeta-older.md")
	newer := filepath.Join(hdir, "2026-08-09-alpha-newer.md")
	if err := os.WriteFile(older, []byte("---\nhandoff_ordinal: 7\n---\n# stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newer, []byte("---\nhandoff_ordinal: 8\n---\n# current\n\n"+block), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out, errs := runCmd(t, "handoff", "latest", "--dir", dir)
	if code != 0 || errs != "" || strings.TrimSpace(out) != newer {
		t.Fatalf("latest: code=%d out=%q stderr=%q; want %q", code, out, errs, newer)
	}
	code, out, errs = runCmd(t, "audit", "stages", "--dir", dir)
	if code != 0 || errs != "" || !strings.Contains(out, newer) || !strings.Contains(out, "carries the spine:cursor block") {
		t.Fatalf("audit stages: code=%d out=%q stderr=%q", code, out, errs)
	}
	code, out, errs = runCmd(t, "handoff", "latest", "--dir", dir, "--json")
	if code != 0 || errs != "" {
		t.Fatalf("latest --json: code=%d out=%q stderr=%q", code, out, errs)
	}
	var got map[string]string
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("latest --json must be valid JSON: %v; out=%q", err, out)
	}
	if got["path"] != newer || got["date"] != "2026-08-09" || got["topic"] != "alpha-newer" || got["title"] != "alpha-newer" || len(got) != 4 {
		t.Fatalf("latest --json schema/selection = %#v, want existing four-field schema for %q", got, newer)
	}
}

// I062: when ordinal selection picks a higher-ordinal stale handoff, both
// consumers must name that same path; audit blocks and doctor remains D9 warn.
func TestSameDateHandoffOrdinalAuditAndDoctorInverseAgree(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "WORKFLOW.md"), []byte("stages: [grill]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	block := "<!-- spine:cursor -->\n" +
		"effort: i062\n" +
		"prd: \n" +
		"tickets: I062\n" +
		"stages: grill[<]\n" +
		"<!-- /spine:cursor -->\n"
	ledger := filepath.Join(dir, ".superpowers", "sdd", "progress.md")
	if err := os.MkdirAll(filepath.Dir(ledger), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ledger, []byte(block), 0o644); err != nil {
		t.Fatal(err)
	}
	hdir := filepath.Join(dir, "docs", "handoffs")
	if err := os.MkdirAll(hdir, 0o755); err != nil {
		t.Fatal(err)
	}
	validOlder := filepath.Join(hdir, "2026-08-09-zeta-valid.md")
	staleNewer := filepath.Join(hdir, "2026-08-09-alpha-stale.md")
	if err := os.WriteFile(validOlder, []byte("---\nhandoff_ordinal: 7\n---\n"+block), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(staleNewer, []byte("---\nhandoff_ordinal: 8\n---\n# stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errs := runCmd(t, "audit", "stages", "--dir", dir)
	if code != 1 || errs != "" || !strings.Contains(out, staleNewer) || !strings.Contains(out, "missing the spine:cursor block") {
		t.Fatalf("audit inverse: code=%d out=%q stderr=%q", code, out, errs)
	}
	code, out, errs = runCmd(t, "doctor", "--dir", dir)
	if code != 1 || errs != "" || !strings.Contains(out, "D9 warn") || !strings.Contains(out, staleNewer) || !strings.Contains(out, "missing the spine:cursor block") {
		t.Fatalf("doctor inverse: code=%d out=%q stderr=%q", code, out, errs)
	}
}

// I062 child helper. Each child executes the public CLI command in a distinct
// process after the parent releases its common start barrier.
func TestI062HandoffNewSubprocessHelper(t *testing.T) {
	if os.Getenv("I062_HANDOFF_NEW_CHILD") != "1" {
		return
	}
	for {
		if _, err := os.Stat(os.Getenv("I062_HANDOFF_NEW_START")); err == nil {
			break
		} else if !os.IsNotExist(err) {
			os.Exit(2)
		}
		time.Sleep(time.Millisecond)
	}
	os.Exit(run([]string{"handoff", "new", "--dir", os.Getenv("I062_HANDOFF_NEW_DIR"), os.Getenv("I062_HANDOFF_NEW_TOPIC")}, os.Stdout, os.Stderr))
}

// I062: allocation must be collision-safe across independent CLI processes,
// not merely callers sharing an in-process mutex.
func TestHandoffNewConcurrentCLIProcessesReserveUniqueOrdinals(t *testing.T) {
	const creators = 32
	dir := t.TempDir()
	start := filepath.Join(dir, "start")
	cmds := make([]*exec.Cmd, 0, creators)
	for i := 0; i < creators; i++ {
		cmd := exec.Command(os.Args[0], "-test.run=^TestI062HandoffNewSubprocessHelper$")
		cmd.Env = append(os.Environ(), "I062_HANDOFF_NEW_CHILD=1", "I062_HANDOFF_NEW_START="+start, "I062_HANDOFF_NEW_DIR="+dir, fmt.Sprintf("I062_HANDOFF_NEW_TOPIC=concurrent-%03d", i))
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		cmds = append(cmds, cmd)
	}
	if err := os.WriteFile(start, []byte("go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			t.Fatalf("concurrent handoff new failed: %v", err)
		}
	}
	entries, err := handoff.List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != creators {
		t.Fatalf("created %d handoffs, want %d", len(entries), creators)
	}
	seen := map[uint64]bool{}
	for _, entry := range entries {
		if entry.Ordinal == 0 || seen[entry.Ordinal] {
			t.Fatalf("ordinal collision or invalid ordinal in %#v", entries)
		}
		seen[entry.Ordinal] = true
	}
}

func TestHandoffNewCanonicalizesValidCursorAndRejectsMalformedCursor(t *testing.T) {
	t.Run("valid non-canonical working block", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "WORKFLOW.md"), []byte("stages: [grill, prd]\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		ledger := filepath.Join(dir, ".superpowers", "sdd", "progress.md")
		if err := os.MkdirAll(filepath.Dir(ledger), 0o755); err != nil {
			t.Fatal(err)
		}
		messy := "<!-- spine:cursor -->\n effort :  canonical-snapshot  \n prd : docs/specs/example.md \n tickets :  I058  \n stages :  grill[x]   prd[<]  \n<!-- /spine:cursor -->\n"
		if err := os.WriteFile(ledger, []byte(messy), 0o644); err != nil {
			t.Fatal(err)
		}

		code, out, errs := runCmd(t, "handoff", "new", "--dir", dir, "canonical snapshot")
		if code != 0 || errs != "" {
			t.Fatalf("new: code=%d out=%q err=%q", code, out, errs)
		}
		raw, err := os.ReadFile(strings.TrimSpace(out))
		if err != nil {
			t.Fatal(err)
		}
		canonical := "<!-- spine:cursor -->\neffort: canonical-snapshot\nprd: docs/specs/example.md\ntickets: I058\nstages: grill[x] prd[<]\n<!-- /spine:cursor -->\n"
		if !strings.Contains(string(raw), canonical) || strings.Contains(string(raw), " effort :") {
			t.Fatalf("handoff snapshot was not canonical:\n%s", raw)
		}
	})

	t.Run("malformed working block", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "WORKFLOW.md"), []byte("stages: [grill, prd]\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		ledger := filepath.Join(dir, ".superpowers", "sdd", "progress.md")
		if err := os.MkdirAll(filepath.Dir(ledger), 0o755); err != nil {
			t.Fatal(err)
		}
		malformed := "<!-- spine:cursor -->\neffort: malformed\nprd: \ntickets: \nstages: grill[<] prd[ ]\n"
		if err := os.WriteFile(ledger, []byte(malformed), 0o644); err != nil {
			t.Fatal(err)
		}

		code, _, errs := runCmd(t, "handoff", "new", "--dir", dir, "must refuse")
		if code == 0 || !strings.Contains(errs, "cursor block is malformed") {
			t.Fatalf("malformed cursor: code=%d stderr=%q", code, errs)
		}
		if _, err := os.Stat(filepath.Join(dir, "docs", "handoffs")); !os.IsNotExist(err) {
			t.Fatalf("malformed cursor created a handoff directory: %v", err)
		}
	})
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

// I3: adopt's text dry-run must show the actual diff for each pending file
// (same diff `spine update` dry-run shows) — the T15 human review gate needs
// to see what would land, not just a one-line "create WORKFLOW.md".
func TestAdoptDryRunShowsDiffs(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runCmd(t, "adopt", "--dir", dir)
	if code != 1 {
		t.Fatalf("want pending exit 1, got %d out=%q", code, out)
	}
	if !strings.Contains(out, "+ template_version: 11") {
		t.Errorf("dry-run text output missing diff content: out=%q", out)
	}
	// --json must never carry the diff text as loose prose in the payload
	// stream; the JSON test above already checks the stream is pure JSON,
	// this just confirms diffs are a text-mode-only addition.
	_, jsonOut, _ := runCmd(t, "adopt", "--dir", dir, "--json")
	if strings.Contains(jsonOut, "+ template_version: 11\n") {
		t.Errorf("json output should not contain raw diff text: out=%q", jsonOut)
	}
}

// I1: adopt --json in a pending state must emit ONLY the JSON payload — no
// trailing "rerun with --write to apply" prose corrupting the stream — and
// the payload itself must carry the pending-ness that the exit code used to
// be the only signal for.
func TestAdoptJSONNoTrailingProse(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runCmd(t, "adopt", "--dir", dir, "--json")
	if code != 1 {
		t.Fatalf("want exit 1 (pending), got %d out=%q", code, out)
	}
	dec := json.NewDecoder(strings.NewReader(out))
	var payload struct {
		Pending bool `json:"pending"`
	}
	if err := dec.Decode(&payload); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nout=%q", err, out)
	}
	if dec.More() {
		t.Fatalf("trailing content after JSON payload: out=%q", out)
	}
	if !payload.Pending {
		t.Errorf("payload.pending = false, want true (adopt is pending)")
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

func TestAuditRoutingEndToEnd(t *testing.T) {
	fixture := func(parts ...string) string {
		return filepath.Join(append([]string{"..", "..", "internal", "audit", "testdata"}, parts...)...)
	}
	// --codex-sessions is pinned to a controlled, absent dir throughout:
	// otherwise the un-overridden default (I041) would resolve to this
	// machine's real ~/.codex/sessions and scan it on every call, exactly
	// the kind of non-hermetic, environment-dependent slowdown --transcripts
	// is already pinned everywhere below to avoid.
	noCodex := filepath.Join(t.TempDir(), "no-codex-sessions")
	// clean fixture: all match, exit 0
	code, out, errs := runCmd(t, "audit", "routing",
		"--dir", fixture("clean", "repo"), "--transcripts", fixture("clean", "transcripts"), "--codex-sessions", noCodex)
	if code != 0 {
		t.Fatalf("clean: code=%d out=%q err=%q", code, out, errs)
	}
	first := strings.SplitN(out, "\n", 2)[0]
	if !strings.HasPrefix(first, "ticket") || !strings.Contains(first, "tier") ||
		!strings.Contains(first, "actual") || !strings.Contains(first, "verdict") {
		t.Errorf("header missing/wrong: %q", first)
	}
	if !strings.Contains(out, "I101") || !strings.Contains(out, "match") {
		t.Errorf("clean out=%q", out)
	}
	// mixed fixture: contains a silent-descent, exit 1
	code, out, _ = runCmd(t, "audit", "routing",
		"--dir", fixture("mixed", "repo"), "--transcripts", fixture("mixed", "transcripts"), "--codex-sessions", noCodex)
	if code != 1 || !strings.Contains(out, "silent-descent") {
		t.Fatalf("mixed: code=%d (want 1) out=%q", code, out)
	}
	if !strings.Contains(out, "housekeeping") {
		t.Errorf("mixed out should list the unmatched dispatch: %q", out)
	}
	// team fixture (I090): claude-team Bash spawns are judged — the
	// under-tiered one exits 1 — and an unmatched spawn renders the effort
	// its command declared, while one that declared none renders bare.
	code, out, _ = runCmd(t, "audit", "routing",
		"--dir", fixture("team", "repo"), "--transcripts", fixture("team", "transcripts"), "--codex-sessions", noCodex)
	if code != 1 || !strings.Contains(out, "silent-descent") {
		t.Fatalf("team: code=%d (want 1) out=%q", code, out)
	}
	if !strings.Contains(out, "[claude-fable-5 @ high]") {
		t.Errorf("team out should render the unmatched spawn's effort: %q", out)
	}
	if !strings.Contains(out, "[claude-opus-4-8]") {
		t.Errorf("a spawn declaring no effort must render bare: %q", out)
	}
	// I090's residual gap is stated, not silent: unattributable team spawns
	// are counted in the footer and point at the follow-up ticket.
	if !strings.Contains(out, "team spawn(s) recognised but not attributed") || !strings.Contains(out, "I101") {
		t.Errorf("team out should carry the unattributable-spawn footer: %q", out)
	}
	// A multi-line spawn command (brief written and worker started in one
	// call) must not print its continuations at column 0.
	code, out, _ = runCmd(t, "audit", "routing",
		"--dir", fixture("teamnoise", "repo"), "--transcripts", fixture("teamnoise", "transcripts"), "--codex-sessions", noCodex)
	if code != 0 {
		t.Fatalf("teamnoise: code=%d (want 0) out=%q", code, out)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "herdr ") || strings.HasPrefix(line, "cat ") {
			t.Errorf("unmatched dispatch broke the indent: %q", line)
		}
	}
	// The same footer fires here, where no brief was delivered by file
	// reference at all (the spawn simply names no ticket outside its
	// heredoc). It must therefore state the fact without guessing a cause.
	if !strings.Contains(out, "team spawn(s) recognised but not attributed") {
		t.Errorf("teamnoise out should carry the footer: %q", out)
	}
	if strings.Contains(out, "$(cat") {
		t.Errorf("footer must not attribute a cause it cannot know: %q", out)
	}
	// degraded fixture: warnings on stderr, exit 0
	code, _, errs = runCmd(t, "audit", "routing",
		"--dir", fixture("degraded", "repo"), "--transcripts", fixture("degraded", "transcripts"), "--codex-sessions", noCodex)
	if code != 0 || !strings.Contains(errs, "warning:") || !strings.Contains(errs, "bad.jsonl") {
		t.Fatalf("degraded: code=%d errs=%q", code, errs)
	}
}

// Acceptance (I041): --codex-sessions overrides discovery, mirroring
// --transcripts. A missing/nonexistent dir degrades to a warning, never an
// error, and never changes the exit code driven by claude-side evidence.
func TestAuditRoutingCodexSessionsFlag(t *testing.T) {
	fixture := func(parts ...string) string {
		return filepath.Join(append([]string{"..", "..", "internal", "audit", "testdata"}, parts...)...)
	}
	code, out, errs := runCmd(t, "audit", "routing",
		"--dir", fixture("clean", "repo"), "--transcripts", fixture("clean", "transcripts"),
		"--codex-sessions", filepath.Join(t.TempDir(), "no-such-codex-dir"))
	if code != 0 {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
	if !strings.Contains(errs, "codex sessions dir unreadable") {
		t.Errorf("want a codex-sessions warning, got errs=%q", errs)
	}
	if !strings.Contains(out, "I101") || !strings.Contains(out, "match") {
		t.Errorf("claude-side evidence must still judge normally: out=%q", out)
	}
}

// Acceptance (RA1/M1, ratified at I041 review — design D-doc "Flavor
// threading"): a missing EXPLICITLY-requested --codex-sessions dir warns
// (proven above); a missing UN-OVERRIDDEN default must be a silent skip —
// otherwise every audit on a codex-less machine gets a standing warning,
// the exact permanent-noise failure the design's problem statement decries.
// CODEX_HOME is pointed at a fresh, sessions-less temp dir so the derived
// default is deterministically absent, independent of this machine's real
// ~/.codex state.
func TestAuditRoutingSilentlySkipsMissingDefaultCodexSessionsDir(t *testing.T) {
	t.Setenv("CODEX_HOME", t.TempDir())
	fixture := func(parts ...string) string {
		return filepath.Join(append([]string{"..", "..", "internal", "audit", "testdata"}, parts...)...)
	}
	code, out, errs := runCmd(t, "audit", "routing",
		"--dir", fixture("clean", "repo"), "--transcripts", fixture("clean", "transcripts"))
	if code != 0 {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
	if strings.Contains(errs, "codex sessions dir unreadable") {
		t.Errorf("a missing un-overridden default must not warn, got errs=%q", errs)
	}
}

func TestAuditUsageErrors(t *testing.T) {
	if code, _, errs := runCmd(t, "audit"); code != 2 || !strings.Contains(errs, "usage: spine audit") {
		t.Fatalf("bare audit: code=%d errs=%q", code, errs)
	}
	if code, _, errs := runCmd(t, "audit", "bogus"); code != 2 || !strings.Contains(errs, "unknown audit subcommand") {
		t.Fatalf("bogus sub: code=%d errs=%q", code, errs)
	}
	// a repo that is not scaffolded (no docs/issues) is a usage error
	if code, _, _ := runCmd(t, "audit", "routing", "--dir", t.TempDir(), "--transcripts", t.TempDir()); code != 2 {
		t.Fatalf("unscaffolded repo: want exit 2, got %d", code)
	}
}

// writeAuditFixtureRepo writes the minimal scaffold `spine audit routing`
// needs at the CLI layer: a WORKFLOW.md carrying a claude model_routing
// table, and one docs/issues/<id>.md per ticket with just the id/tier
// frontmatter fields readTickets actually consumes (see
// internal/audit/audit.go readTickets).
func writeAuditFixtureRepo(t *testing.T, dir string, tickets map[string]string) {
	t.Helper()
	workflow := "profile: go-service\ntemplate_version: 9\nmodel_routing:\n" +
		"  primary: claude-fable-5\n  routine: claude-sonnet-5\n" +
		"  mechanical: claude-haiku-4-5\n  fallback: claude-opus-4-8\n"
	if err := os.WriteFile(filepath.Join(dir, "WORKFLOW.md"), []byte(workflow), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs", "issues"), 0o755); err != nil {
		t.Fatal(err)
	}
	for id, tier := range tickets {
		body := fmt.Sprintf("---\nid: %s\ntier: %s\n---\n", id, tier)
		if err := os.WriteFile(filepath.Join(dir, "docs", "issues", id+".md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// writeAuditDispatch writes one session file (mirroring
// internal/audit/i047_test.go's writeSingleDispatch) carrying one Task
// dispatch for ticketID, with an explicit cwd so it repo-qualifies (D28).
func writeAuditDispatch(t *testing.T, path, repoDir, ticketID, model string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	line := fmt.Sprintf(
		`{"type":"assistant","cwd":%q,"message":{"model":"claude-fable-5","role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Task","input":{"description":%q,"model":%q,"prompt":"You are implementing ticket %s."}}]}}`+"\n",
		repoDir, ticketID+": fixture work", model, ticketID)
	if err := os.WriteFile(path, []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Acceptance (Testing Decisions CLI clause): --since parses at the CLI layer
// and threads through to audit.Run, observably excluding an older session's
// evidence — mirroring internal/audit/i047_test.go's
// TestSinceAndSessionFiltersComposeWithDefaults at the command-runner layer.
func TestAuditRoutingSinceFlagExcludesOlderSession(t *testing.T) {
	dir := t.TempDir()
	writeAuditFixtureRepo(t, dir, map[string]string{"I900": "routine"})
	tdir := t.TempDir()
	oldPath := filepath.Join(tdir, "old.jsonl")
	newPath := filepath.Join(tdir, "new.jsonl")
	writeAuditDispatch(t, oldPath, dir, "I900", "claude-haiku-4-5") // descent
	writeAuditDispatch(t, newPath, dir, "I900", "claude-sonnet-5")  // clean
	if err := os.Chtimes(oldPath, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newPath, time.Date(2020, 1, 5, 0, 0, 0, 0, time.UTC), time.Date(2020, 1, 5, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	noCodex := filepath.Join(t.TempDir(), "no-codex-sessions")

	code, out, _ := runCmd(t, "audit", "routing",
		"--dir", dir, "--transcripts", tdir, "--codex-sessions", noCodex)
	if code != 1 || !strings.Contains(out, "silent-descent") {
		t.Fatalf("no filter: code=%d out=%q, want blocking silent-descent from old.jsonl", code, out)
	}

	code, out, errs := runCmd(t, "audit", "routing",
		"--dir", dir, "--transcripts", tdir, "--codex-sessions", noCodex, "--since", "2020-01-03")
	if code != 0 {
		t.Fatalf("--since 2020-01-03: code=%d out=%q errs=%q, want exit 0 once old.jsonl is excluded", code, out, errs)
	}
	if strings.Contains(out, "silent-descent") {
		t.Errorf("--since 2020-01-03 should exclude old.jsonl's descent, out=%q", out)
	}
	if !strings.Contains(out, "match") {
		t.Errorf("want new.jsonl's surviving clean evidence to read as a match, out=%q", out)
	}
}

// Acceptance (Testing Decisions CLI clause, ratified exit-2 rule): an
// unparseable --since value is a usage error at the CLI layer, exit 2 with a
// usage-style message — the load-bearing case, since exit codes are the
// CLI's contract (mirrors internal/audit/i047_test.go's
// TestSinceUnparseableIsUsageError one layer up).
func TestAuditRoutingSinceUnparseableExitsUsageError(t *testing.T) {
	dir := t.TempDir()
	writeAuditFixtureRepo(t, dir, map[string]string{"I901": "routine"})
	tdir := t.TempDir()
	writeAuditDispatch(t, filepath.Join(tdir, "s1.jsonl"), dir, "I901", "claude-sonnet-5")
	noCodex := filepath.Join(t.TempDir(), "no-codex-sessions")

	code, _, errs := runCmd(t, "audit", "routing",
		"--dir", dir, "--transcripts", tdir, "--codex-sessions", noCodex, "--since", "garbage")
	if code != 2 {
		t.Fatalf("code=%d errs=%q, want exit 2 (usage error) for an unparseable --since value", code, errs)
	}
	if !strings.Contains(errs, "--since") || !strings.Contains(errs, "garbage") {
		t.Errorf("want a usage-style error naming --since and the bad value, errs=%q", errs)
	}
}

// Acceptance (Testing Decisions CLI clause): --session restricts evidence to
// one session at the CLI layer, and a non-matching id surfaces the
// matched-no-sessions warning on stderr (mirrors internal/audit/i047_test.go's
// TestSinceAndSessionFiltersComposeWithDefaults and
// TestSessionMatchingNothingWarns one layer up).
func TestAuditRoutingSessionFlagRestrictsAndWarnsOnNoMatch(t *testing.T) {
	dir := t.TempDir()
	writeAuditFixtureRepo(t, dir, map[string]string{"I902": "routine"})
	tdir := t.TempDir()
	writeAuditDispatch(t, filepath.Join(tdir, "s1.jsonl"), dir, "I902", "claude-sonnet-5")
	noCodex := filepath.Join(t.TempDir(), "no-codex-sessions")

	code, out, errs := runCmd(t, "audit", "routing",
		"--dir", dir, "--transcripts", tdir, "--codex-sessions", noCodex, "--session", "s1")
	if code != 0 {
		t.Fatalf("code=%d out=%q errs=%q", code, out, errs)
	}
	if !strings.Contains(out, "match") {
		t.Errorf("want I902 to match via the isolated session, out=%q", out)
	}
	if strings.Contains(errs, "matched no sessions") {
		t.Errorf("a real --session match must not warn, errs=%q", errs)
	}

	code, _, errs = runCmd(t, "audit", "routing",
		"--dir", dir, "--transcripts", tdir, "--codex-sessions", noCodex, "--session", "typo-session")
	if code != 0 {
		t.Fatalf("code=%d errs=%q, want exit 0 (no-transcript warns, does not block)", code, errs)
	}
	if !strings.Contains(errs, `--session "typo-session" matched no sessions`) {
		t.Errorf("want the unmatched --session warning, errs=%q", errs)
	}
}

func stagesFixture(scenario string) string {
	return filepath.Join("..", "..", "internal", "stages", "testdata", scenario, "repo")
}

func TestAuditStagesCleanExitsZero(t *testing.T) {
	code, out, errs := runCmd(t, "audit", "stages", "--dir", stagesFixture("clean"))
	if code != 0 {
		t.Fatalf("code=%d out=%q errs=%q", code, out, errs)
	}
	first := strings.SplitN(out, "\n", 2)[0]
	if !strings.HasPrefix(first, "stage") || !strings.Contains(first, "state") ||
		!strings.Contains(first, "verdict") || !strings.Contains(first, "detail") {
		t.Errorf("header missing/wrong: %q", first)
	}
	if !strings.Contains(out, "match") {
		t.Errorf("out=%q", out)
	}
	if !strings.Contains(out, "handoff: applicable=true blocking=false") {
		t.Errorf("want a non-blocking handoff line, out=%q", out)
	}
}

// I059: formatting a valid cursor block by hand is a sole-writer violation,
// not malformed grammar. Audit stages must block it and name the built-in
// rewrite path; the canonical clean fixture above remains an exit-0 control.
func TestAuditStagesNonCanonicalCursorBlocksWithRewriteRemediation(t *testing.T) {
	code, out, errs := runCmd(t, "audit", "stages", "--dir", stagesFixture("noncanonical-cursor"))
	if code != 1 {
		t.Fatalf("want exit 1 for a valid but non-canonical cursor, got %d, out=%q errs=%q", code, out, errs)
	}
	for _, want := range []string{"non-canonical", "spine cursor", "spine cursor set"} {
		if !strings.Contains(out, want) {
			t.Errorf("want remediation %q in audit finding, out=%q", want, out)
		}
	}
	if strings.Contains(out, "malformed cursor block") {
		t.Errorf("valid formatting drift must remain distinct from malformed grammar, out=%q", out)
	}
}

// I059: doctor consumes the same fixture at its CLI boundary. It stays a
// warning (and therefore preserves the normal doctor non-zero health exit),
// rather than becoming a second blocking gate with different remediation.
func TestDoctorAdvisesOnNonCanonicalCursor(t *testing.T) {
	code, out, errs := runCmd(t, "doctor", "--dir", stagesFixture("noncanonical-cursor"))
	if code != 1 {
		t.Fatalf("doctor warnings must retain the usual exit 1, got %d, out=%q errs=%q", code, out, errs)
	}
	for _, want := range []string{"D9 warn", "non-canonical", "spine cursor set"} {
		if !strings.Contains(out, want) {
			t.Errorf("want doctor advisory %q, out=%q errs=%q", want, out, errs)
		}
	}
}

// The canonical control fixture may have unrelated scaffold-health findings,
// but its cursor must not acquire a D9 warning from this gate.
func TestDoctorLeavesCanonicalCursorWithoutD9(t *testing.T) {
	_, out, errs := runCmd(t, "doctor", "--dir", stagesFixture("clean"))
	if strings.Contains(out, "D9 ") || strings.Contains(errs, "D9 ") {
		t.Fatalf("canonical cursor must not produce a D9 finding, out=%q errs=%q", out, errs)
	}
}

func TestAuditStagesTickedMissingBlocks(t *testing.T) {
	code, out, _ := runCmd(t, "audit", "stages", "--dir", stagesFixture("ticked-missing"))
	if code != 1 {
		t.Fatalf("want exit 1 on a blocking mismatch, got %d, out=%q", code, out)
	}
	if !strings.Contains(out, "ticked-missing") {
		t.Errorf("out=%q", out)
	}
}

func TestAuditStagesPresentUntickedBlocks(t *testing.T) {
	code, out, _ := runCmd(t, "audit", "stages", "--dir", stagesFixture("present-unticked"))
	if code != 1 {
		t.Fatalf("want exit 1 on a blocking mismatch, got %d, out=%q", code, out)
	}
	if !strings.Contains(out, "present-unticked") {
		t.Errorf("out=%q", out)
	}
}

func TestAuditStagesNoLedgerWarnsExitZero(t *testing.T) {
	code, out, errs := runCmd(t, "audit", "stages", "--dir", stagesFixture("no-ledger-warn"))
	if code != 0 {
		t.Fatalf("want exit 0 (warn only, no progress.md), got %d out=%q errs=%q", code, out, errs)
	}
	if !strings.Contains(errs, "warning:") || !strings.Contains(errs, "progress.md") {
		t.Errorf("want a warning mentioning progress.md, errs=%q", errs)
	}
	if !strings.Contains(out, "nothing to audit") {
		t.Errorf("out=%q", out)
	}
}

func TestAuditStagesHandoffMissingBlockBlocks(t *testing.T) {
	code, out, _ := runCmd(t, "audit", "stages", "--dir", stagesFixture("handoff-missing-block"))
	if code != 1 {
		t.Fatalf("want exit 1 (newest handoff lacks the cursor block), got %d, out=%q", code, out)
	}
	if !strings.Contains(out, "handoff: applicable=true blocking=true") {
		t.Errorf("out=%q", out)
	}
}

// A cursor block whose stages: line is garbage (grammar findings, zero
// stage rows) must not sail through audit stages at exit 0 — that is a
// silent gate bypass (the whole point of a stage cursor is to gate).
// audit stages is the ONLY caller that must turn CursorFindings blocking;
// spine cursor (below) and doctor D9 stay advisory.
func TestAuditStagesMalformedCursorBlocks(t *testing.T) {
	code, out, errs := runCmd(t, "audit", "stages", "--dir", stagesFixture("malformed-cursor"))
	if code != 1 {
		t.Fatalf("want exit 1 (malformed cursor grammar findings must block audit stages), got %d, out=%q errs=%q", code, out, errs)
	}
	if !strings.Contains(out, "malformed") {
		t.Errorf("want the blocking cursor-malformed finding surfaced in the report table, out=%q", out)
	}
	if strings.Contains(out, "non-canonical") {
		t.Errorf("malformed cursor must retain its distinct finding, out=%q", out)
	}
}

// spine cursor stays exit-0-always even on the same malformed fixture — it
// is a read-only printer, never a gate. Only audit stages gates.
func TestCursorCommandStaysExitZeroOnMalformedCursor(t *testing.T) {
	code, out, errs := runCmd(t, "cursor", "--dir", stagesFixture("malformed-cursor"))
	if code != 0 {
		t.Fatalf("spine cursor must stay exit-0-always even on a malformed cursor, got %d out=%q errs=%q", code, out, errs)
	}
}

// I024: this fixture's stages: line ("??? *** !!!") parses to zero stage
// rows and its handoff carries the cursor block, so nothing else blocks —
// before the fix this sailed through to "derivation: clean", which is
// incoherent with `spine audit stages` blocking on the same fixture
// (TestAuditStagesMalformedCursorBlocks). The derivation line must instead
// name the cursor as malformed, still at exit 0 (read-only printer contract
// unchanged).
func TestCursorCommandMalformedGrammarPrintsNAWording(t *testing.T) {
	code, out, errs := runCmd(t, "cursor", "--dir", stagesFixture("malformed-cursor"))
	if code != 0 {
		t.Fatalf("spine cursor must stay exit-0-always even on a malformed cursor, got %d out=%q errs=%q", code, out, errs)
	}
	if !strings.Contains(out, "derivation: n/a (cursor malformed)") {
		t.Errorf("want the n/a (cursor malformed) wording, out=%q", out)
	}
	if strings.Contains(out, "derivation: clean") {
		t.Errorf("must not claim clean on a grammar-malformed cursor, out=%q errs=%q", out, errs)
	}
}

// F1(a) (final whole-branch review, I024-I027 batch): before this fix, an
// unresolvable tickets: value (I026's Report.Notes entry) was computed but
// never printed by `spine cursor` — this fixture's stages otherwise resolve
// cleanly, so the command surfaced an ambient "derivation: clean" with the
// unresolvable-tickets warning nowhere on the hook surface (the
// SessionStart hook wires this command's stdout into session context).
// spine audit stages already prints the equivalent Notes entries (as
// "warning: <note>") — spine cursor must match that.
func TestCursorCommandSurfacesUnresolvableTicketsNote(t *testing.T) {
	code, out, errs := runCmd(t, "cursor", "--dir", stagesFixture("unresolvable-tickets"))
	if code != 0 {
		t.Fatalf("spine cursor stays exit-0-always (advisory), got %d out=%q errs=%q", code, out, errs)
	}
	if !strings.Contains(out, "warning:") || !strings.Contains(out, "not-a-grammar") {
		t.Errorf("want a warning naming the unresolvable tickets: value on stdout, out=%q", out)
	}
}

// F1(b) (final whole-branch review, I024-I027 batch): the "n/a (cursor
// malformed)" branch (I024) returned early without ever checking
// rep.Handoff — the info-loss corner the I024 review found. A malformed
// stages: grammar AND a blocking (missing-block) newest handoff both need
// attention from the human/agent reading the hook surface, but only the
// malformed-cursor header made it through. Both must be visible together.
func TestCursorCommandMalformedBranchStillPrintsBlockingHandoff(t *testing.T) {
	code, out, errs := runCmd(t, "cursor", "--dir", stagesFixture("malformed-cursor-handoff-missing"))
	if code != 0 {
		t.Fatalf("spine cursor stays exit-0-always (advisory), got %d out=%q errs=%q", code, out, errs)
	}
	if !strings.Contains(out, "derivation: n/a (cursor malformed)") {
		t.Errorf("want the n/a (cursor malformed) header, out=%q", out)
	}
	if !strings.Contains(out, "handoff:") || !strings.Contains(out, "missing the spine:cursor block") {
		t.Errorf("want the blocking handoff detail printed alongside the malformed header, out=%q", out)
	}
}

func TestHandoffListTextHasHeaderAndPath(t *testing.T) {
	dir := t.TempDir()
	if code, _, errs := runCmd(t, "handoff", "new", "-dir", dir, "v3 cosmetics"); code != 0 {
		t.Fatal(errs)
	}
	code, out, errs := runCmd(t, "handoff", "list", "-dir", dir)
	if code != 0 {
		t.Fatal(errs)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want header + 1 row, got %d lines: %q", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "date") || !strings.Contains(lines[0], "topic") || !strings.Contains(lines[0], "path") {
		t.Errorf("header missing/wrong: %q", lines[0])
	}
	if !strings.Contains(lines[1], "v3-cosmetics") || !strings.Contains(lines[1], filepath.Join(dir, "docs", "handoffs")) {
		t.Errorf("row missing topic or path: %q", lines[1])
	}
}

func TestEvalListTextHasHeader(t *testing.T) {
	dir := t.TempDir()
	if code, _, errs := runCmd(t, "eval", "new", "-dir", dir, "header eval"); code != 0 {
		t.Fatal(errs)
	}
	code, out, errs := runCmd(t, "eval", "list", "-dir", dir)
	if code != 0 {
		t.Fatal(errs)
	}
	first := strings.SplitN(out, "\n", 2)[0]
	if !strings.HasPrefix(first, "eval") || !strings.Contains(first, "run") ||
		!strings.Contains(first, "stage") || !strings.Contains(first, "score") {
		t.Errorf("header missing/wrong: %q", first)
	}
}

func TestUpdateTextNamesPreservedFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"WORKFLOW.md", "CLAUDE.md"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "internal", "update", "testdata", "ccq", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Hand-authored ADR index: ADR-0009 territory — update must SAY so.
	if err := os.WriteFile(filepath.Join(dir, "docs", "adr", "README.md"), []byte("# my hand-rolled index\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, out, _ := runCmd(t, "update", "-dir", dir)
	if !strings.Contains(out, "preserved (hand-authored): docs/adr/README.md") {
		t.Errorf("no preservation notice in:\n%s", out)
	}
}

func TestHandoffLatestRejectsFlagLikeDirValues(t *testing.T) {
	cases := []struct {
		args    []string
		wantMsg string
	}{
		{[]string{"handoff", "latest", "-fleet", "--dir"}, "--fleet needs a directory value"},
		{[]string{"handoff", "latest", "-dir", "--json"}, "--dir needs a directory value"},
	}
	for _, c := range cases {
		code, _, errs := runCmd(t, c.args...)
		if code != 2 {
			t.Errorf("%v: code = %d, want 2 (stderr %q)", c.args, code, errs)
		}
		if !strings.Contains(errs, c.wantMsg) {
			t.Errorf("%v: stderr = %q, want it to contain %q", c.args, errs, c.wantMsg)
		}
	}
	// A legitimate fleet dir (no handoffs anywhere) still parses and runs.
	if code, _, errs := runCmd(t, "handoff", "latest", "-fleet", t.TempDir()); code != 0 {
		t.Errorf("legit -fleet dir: code = %d, stderr %q", code, errs)
	}
}

func TestHandoffListAlignsPathColumnPastDefaultWidth(t *testing.T) {
	dir := t.TempDir()
	for _, topic := range []string{"short", "extremely long handoff topic exceeding twenty eight chars"} {
		if code, _, errs := runCmd(t, "handoff", "new", "-dir", dir, topic); code != 0 {
			t.Fatal(errs)
		}
	}
	code, out, errs := runCmd(t, "handoff", "list", "-dir", dir)
	if code != 0 {
		t.Fatal(errs)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("want header + 2 rows, got %d lines: %q", len(lines), out)
	}
	want := strings.Index(lines[0], "path")
	if want < 0 {
		t.Fatalf("no path header: %q", lines[0])
	}
	prefix := filepath.Join(dir, "docs", "handoffs")
	for _, row := range lines[1:] {
		if got := strings.Index(row, prefix); got != want {
			t.Errorf("path column at %d, want %d: %q", got, want, row)
		}
	}
}

func cursorFixture(scenario string) string {
	return filepath.Join("..", "..", "internal", "cursor", "testdata", scenario, "repo")
}

func TestCursorCommandPrintsValidCursor(t *testing.T) {
	code, out, errs := runCmd(t, "cursor", "--dir", cursorFixture("valid"))
	if code != 0 {
		t.Fatalf("code=%d err=%q", code, errs)
	}
	for _, want := range []string{
		"effort: fixture-effort",
		"prd: docs/specs/2026-01-01-fixture-design.md",
		"tickets: I001-I005",
		"implement[<]",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("out missing %q; out=%q", want, out)
		}
	}
	// This grammar-only fixture (internal/cursor/testdata) has no
	// docs/specs, docs/issues, or docs/handoffs on disk — prd and issues
	// are ticked done with nothing to back them, so the live verdict must
	// be blocking, and it must be exit 0 regardless (advisory here).
	if !strings.Contains(out, "derivation: blocking") {
		t.Errorf("want a live blocking verdict (this fixture's ticked stages have no artifacts), out=%q", out)
	}
	if !strings.Contains(out, "prd (ticked-missing)") || !strings.Contains(out, "issues (ticked-missing)") {
		t.Errorf("want prd/issues ticked-missing detail lines, out=%q", out)
	}
}

func TestCursorCommandExitsZeroOnMalformedAndPrintsFindings(t *testing.T) {
	code, out, _ := runCmd(t, "cursor", "--dir", cursorFixture("malformed"))
	if code != 0 {
		t.Fatalf("want exit 0 (advisory), got %d, out=%q", code, out)
	}
	if !strings.Contains(out, "tickets") {
		t.Errorf("want finding naming the missing tickets key, out=%q", out)
	}
}

func TestCursorQuietSilentWhenNoCursor(t *testing.T) {
	code, out, errs := runCmd(t, "cursor", "--quiet", "--dir", t.TempDir())
	if code != 0 || out != "" || errs != "" {
		t.Fatalf("code=%d out=%q errs=%q, want silent exit 0", code, out, errs)
	}
}

func TestCursorQuietSilentWhenSpineRepoHasNoLedgerYet(t *testing.T) {
	// A spine repo (WORKFLOW.md present) that hasn't started an SDD effort
	// yet has no progress.md at all — same "nothing to report" case as not
	// being a spine repo.
	code, out, errs := runCmd(t, "cursor", "--quiet", "--dir", cursorFixture("missing"))
	if code != 0 || out != "" || errs != "" {
		t.Fatalf("code=%d out=%q errs=%q, want silent exit 0", code, out, errs)
	}
}

func TestCursorQuietDoesNotSuppressAPresentCursor(t *testing.T) {
	// --quiet is for hook use: silent when there's nothing to report, but a
	// SessionStart hook wiring "spine cursor --quiet" into session context
	// (I021) still needs real output when a cursor exists.
	code, out, errs := runCmd(t, "cursor", "--quiet", "--dir", cursorFixture("valid"))
	if code != 0 {
		t.Fatalf("code=%d errs=%q", code, errs)
	}
	if !strings.Contains(out, "effort: fixture-effort") {
		t.Errorf("want cursor still printed under --quiet when one exists, out=%q", out)
	}
}

// cursorWriteRepo creates a small on-disk spine repo for the cursor write
// verbs. Tests drive run through its public command boundary; this helper
// deliberately does not call cursor parsing or serialization APIs.
func cursorWriteRepo(t *testing.T, stages string) string {
	t.Helper()
	dir := t.TempDir()
	workflow := "profile: library-cli\nstages: [" + stages + "]\n"
	if err := os.WriteFile(filepath.Join(dir, "WORKFLOW.md"), []byte(workflow), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func cursorLedger(t *testing.T, dir string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, ".superpowers", "sdd", "progress.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestCursorTickTripwireUsesAuditFindingAndForceWritesAtomically(t *testing.T) {
	dir := cursorWriteRepo(t, "grill, prd, issues")
	if code, _, errs := runCmd(t, "cursor", "start", "--dir", dir, "--effort", "tripwire", "--prd", "docs/specs/missing.md"); code != 0 {
		t.Fatalf("start: code=%d stderr=%q", code, errs)
	}
	if code, _, errs := runCmd(t, "cursor", "tick", "--dir", dir, "grill"); code != 0 {
		t.Fatalf("tick grill: code=%d stderr=%q", code, errs)
	}
	code, _, refusal := runCmd(t, "cursor", "tick", "--dir", dir, "prd")
	if code != 1 {
		t.Fatalf("tick without artifact: code=%d stderr=%q", code, refusal)
	}
	if !strings.Contains(refusal, "marked done but 1/1 PRD file docs/specs/missing.md missing") {
		t.Fatalf("tripwire did not print the audit finding text: %q", refusal)
	}
	if got := cursorLedger(t, dir); strings.Contains(got, "prd[x]") {
		t.Fatalf("refused tick wrote the ledger: %q", got)
	}
	if code, _, errs := runCmd(t, "cursor", "tick", "--dir", dir, "prd", "--force"); code != 0 {
		t.Fatalf("forced tick: code=%d stderr=%q", code, errs)
	}
	code, auditOut, auditErr := runCmd(t, "audit", "stages", "--dir", dir)
	if code != 1 {
		t.Fatalf("forced tick must remain audit-blocking: code=%d out=%q stderr=%q", code, auditOut, auditErr)
	}
	if !strings.Contains(auditOut, strings.TrimSpace(refusal)) {
		t.Fatalf("audit finding differs from tripwire: refusal=%q audit=%q", refusal, auditOut)
	}
	ledger := cursorLedger(t, dir)
	want := "<!-- spine:cursor -->\neffort: tripwire\nprd: docs/specs/missing.md\ntickets: \nstages: grill[x] prd[x] issues[<]\n<!-- /spine:cursor -->\n"
	if ledger != want {
		t.Fatalf("forced tick bytes:\n%s\nwant:\n%s", ledger, want)
	}
	entries, err := os.ReadDir(filepath.Join(dir, ".superpowers", "sdd"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "progress.md" {
		t.Fatalf("atomic cursor write left temporary residue: %#v", entries)
	}
}

// A present derivation artifact must let the ordinary (unforced) write path
// succeed, retain canonical bytes, and produce a matching non-blocking row.
func TestCursorTickWithPresentArtifactSucceedsAndAuditsMatching(t *testing.T) {
	dir := cursorWriteRepo(t, "grill, prd, issues")
	if err := os.MkdirAll(filepath.Join(dir, "docs", "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "specs", "ready.md"), []byte("# Ready\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := runCmd(t, "cursor", "start", "--dir", dir, "--effort", "artifact", "--prd", "docs/specs/ready.md"); code != 0 {
		t.Fatalf("start: code=%d stderr=%q", code, errs)
	}
	if code, _, errs := runCmd(t, "cursor", "tick", "--dir", dir, "grill"); code != 0 {
		t.Fatalf("tick grill: code=%d stderr=%q", code, errs)
	}
	code, out, errs := runCmd(t, "cursor", "tick", "--dir", dir, "prd")
	if code != 0 || out != "cursor ticked: prd\n" || errs != "" {
		t.Fatalf("unforced artifact-present tick: code=%d out=%q stderr=%q", code, out, errs)
	}
	want := "<!-- spine:cursor -->\neffort: artifact\nprd: docs/specs/ready.md\ntickets: \nstages: grill[x] prd[x] issues[<]\n<!-- /spine:cursor -->\n"
	if got := cursorLedger(t, dir); got != want {
		t.Fatalf("canonical tick bytes:\n%s\nwant:\n%s", got, want)
	}
	if code, _, errs := runCmd(t, "handoff", "new", "--dir", dir, "artifact snapshot"); code != 0 || errs != "" {
		t.Fatalf("handoff: code=%d stderr=%q", code, errs)
	}
	code, auditOut, auditErr := runCmd(t, "audit", "stages", "--dir", dir)
	if code != 0 || auditErr != "" || !strings.Contains(auditOut, "prd") || !strings.Contains(auditOut, "match") || !strings.Contains(auditOut, "1/1 PRD file docs/specs/ready.md present") {
		t.Fatalf("matching audit row: code=%d out=%q stderr=%q", code, auditOut, auditErr)
	}
}

// The newest handoff is a complete state snapshot. Same-effort cursor writes
// must therefore be visible to audit and doctor's D9, until a fresh handoff
// preserves the new state without rewriting historical snapshots.
func TestHandoffSnapshotDetectsSameEffortCursorDriftAndDoctorAdvises(t *testing.T) {
	dir := cursorWriteRepo(t, "grill, prd, review")
	if err := os.MkdirAll(filepath.Join(dir, "docs", "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "specs", "ready.md"), []byte("# Ready\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := runCmd(t, "cursor", "start", "--dir", dir, "--effort", "same-effort", "--prd", "docs/specs/ready.md"); code != 0 {
		t.Fatalf("start: code=%d stderr=%q", code, errs)
	}
	for _, stage := range []string{"grill", "prd"} {
		if code, _, errs := runCmd(t, "cursor", "tick", "--dir", dir, stage); code != 0 {
			t.Fatalf("tick %s: code=%d stderr=%q", stage, code, errs)
		}
	}
	if code, _, errs := runCmd(t, "handoff", "new", "--dir", dir, "snapshot 01 initial"); code != 0 || errs != "" {
		t.Fatalf("initial handoff: code=%d stderr=%q", code, errs)
	}
	assertClean := func(label string) {
		t.Helper()
		if code, out, errs := runCmd(t, "audit", "stages", "--dir", dir); code != 0 || errs != "" {
			t.Fatalf("%s audit must be clean: code=%d out=%q stderr=%q", label, code, out, errs)
		}
	}
	assertBlocked := func(label string) {
		t.Helper()
		code, out, _ := runCmd(t, "audit", "stages", "--dir", dir)
		if code != 1 || !strings.Contains(out, "stale cursor snapshot") || !strings.Contains(out, "spine handoff new") {
			t.Fatalf("%s audit must report actionable stale snapshot: code=%d out=%q", label, code, out)
		}
		code, out, _ = runCmd(t, "doctor", "--dir", dir)
		if code != 1 || !strings.Contains(out, "D9") || !strings.Contains(out, "stale cursor snapshot") {
			t.Fatalf("%s doctor must advise through D9: code=%d out=%q", label, code, out)
		}
	}
	assertClean("initial")

	if code, _, errs := runCmd(t, "cursor", "here", "--dir", dir, "grill"); code != 0 {
		t.Fatalf("here: code=%d stderr=%q", code, errs)
	}
	assertBlocked("here")
	if code, _, errs := runCmd(t, "handoff", "new", "--dir", dir, "snapshot 02 here"); code != 0 || errs != "" {
		t.Fatalf("here refresh: code=%d stderr=%q", code, errs)
	}
	assertClean("here refresh")

	if code, _, errs := runCmd(t, "cursor", "set", "--dir", dir, "--tickets", "I058"); code != 0 {
		t.Fatalf("set: code=%d stderr=%q", code, errs)
	}
	assertBlocked("set")
	if code, _, errs := runCmd(t, "handoff", "new", "--dir", dir, "snapshot 03 set"); code != 0 || errs != "" {
		t.Fatalf("set refresh: code=%d stderr=%q", code, errs)
	}
	assertClean("set refresh")

	if code, _, errs := runCmd(t, "cursor", "tick", "--dir", dir, "review"); code != 0 {
		t.Fatalf("tick review: code=%d stderr=%q", code, errs)
	}
	assertBlocked("tick")
	if code, _, errs := runCmd(t, "handoff", "new", "--dir", dir, "snapshot 04 tick"); code != 0 || errs != "" {
		t.Fatalf("tick refresh: code=%d stderr=%q", code, errs)
	}
	assertClean("tick refresh")
}

func TestHandoffSnapshotMalformedBlockBlocksAuditAndDoctor(t *testing.T) {
	dir := cursorWriteRepo(t, "grill")
	if code, _, errs := runCmd(t, "cursor", "start", "--dir", dir, "--effort", "malformed"); code != 0 {
		t.Fatalf("start: code=%d stderr=%q", code, errs)
	}
	path := filepath.Join(dir, "docs", "handoffs", "2026-08-06-malformed.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("# malformed\n\n<!-- spine:cursor -->\neffort: malformed\n<!-- /spine:cursor -->\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runCmd(t, "audit", "stages", "--dir", dir)
	if code != 1 || !strings.Contains(out, "malformed spine:cursor block") || !strings.Contains(out, "spine handoff new") {
		t.Fatalf("malformed snapshot audit: code=%d out=%q", code, out)
	}
	code, out, _ = runCmd(t, "doctor", "--dir", dir)
	if code != 1 || !strings.Contains(out, "D9") || !strings.Contains(out, "malformed spine:cursor block") {
		t.Fatalf("malformed snapshot doctor: code=%d out=%q", code, out)
	}
}

func TestCursorVerbsMarkerRegressionAndStartGuard(t *testing.T) {
	dir := cursorWriteRepo(t, "grill, review, verify")
	if code, _, errs := runCmd(t, "cursor", "start", "--dir", dir, "--effort", "first"); code != 0 {
		t.Fatalf("start: code=%d stderr=%q", code, errs)
	}
	startWant := "<!-- spine:cursor -->\neffort: first\nprd: \ntickets: \nstages: grill[<] review[ ] verify[ ]\n<!-- /spine:cursor -->\n"
	if got := cursorLedger(t, dir); got != startWant {
		t.Fatalf("start did not seed canonical bytes:\n%s\nwant:\n%s", got, startWant)
	}
	if code, out, errs := runCmd(t, "cursor", "--dir", dir); code != 0 || strings.Contains(out, "finding:") {
		t.Fatalf("started cursor did not parse cleanly: code=%d out=%q stderr=%q", code, out, errs)
	}
	before := cursorLedger(t, dir)
	if code, _, errs := runCmd(t, "cursor", "start", "--dir", dir, "--effort", "second"); code != 1 || !strings.Contains(errs, "mid-flight") {
		t.Fatalf("unguarded start: code=%d stderr=%q", code, errs)
	}
	if got := cursorLedger(t, dir); got != before {
		t.Fatalf("guarded start changed cursor:\n%s", got)
	}
	if code, _, errs := runCmd(t, "cursor", "start", "--dir", dir, "--effort", "second", "--force"); code != 0 {
		t.Fatalf("forced start: code=%d stderr=%q", code, errs)
	}
	if code, _, errs := runCmd(t, "cursor", "tick", "--dir", dir, "verify"); code != 0 {
		t.Fatalf("non-marker tick: code=%d stderr=%q", code, errs)
	}
	if got := cursorLedger(t, dir); !strings.Contains(got, "grill[<] review[ ] verify[x]") {
		t.Fatalf("non-marker tick moved marker: %q", got)
	}
	if code, _, errs := runCmd(t, "cursor", "tick", "--dir", dir, "grill"); code != 0 {
		t.Fatalf("marker tick: code=%d stderr=%q", code, errs)
	}
	if got := cursorLedger(t, dir); !strings.Contains(got, "grill[x] review[<] verify[x]") {
		t.Fatalf("marker did not advance: %q", got)
	}
	if code, _, errs := runCmd(t, "cursor", "tick", "--dir", dir, "review"); code != 0 {
		t.Fatalf("final marker tick: code=%d stderr=%q", code, errs)
	}
	if got := cursorLedger(t, dir); !strings.Contains(got, "grill[x] review[x] verify[x]") {
		t.Fatalf("final marker was not dropped: %q", got)
	}
	if code, _, errs := runCmd(t, "cursor", "here", "--dir", dir, "grill"); code != 0 {
		t.Fatalf("here done stage: code=%d stderr=%q", code, errs)
	}
	if got := cursorLedger(t, dir); !strings.Contains(got, "grill[<] review[x] verify[x]") {
		t.Fatalf("here did not regress completed stage: %q", got)
	}
	if code, _, errs := runCmd(t, "cursor", "start", "--dir", dir, "--effort", "cyclic", "--force"); code != 0 {
		t.Fatalf("cyclic start: code=%d stderr=%q", code, errs)
	}
	if code, _, errs := runCmd(t, "cursor", "here", "--dir", dir, "verify"); code != 0 {
		t.Fatalf("forward here: code=%d stderr=%q", code, errs)
	}
	if code, _, errs := runCmd(t, "cursor", "tick", "--dir", dir, "verify"); code != 0 {
		t.Fatalf("terminal tick with earlier pending stages: code=%d stderr=%q", code, errs)
	}
	if got := cursorLedger(t, dir); !strings.Contains(got, "grill[<] review[ ] verify[x]") {
		t.Fatalf("terminal tick did not cycle to the next unticked stage: %q", got)
	}
	if code, out, errs := runCmd(t, "cursor", "--dir", dir); code != 0 || strings.Contains(out, "finding:") {
		t.Fatalf("cyclic marker result did not parse cleanly: code=%d out=%q stderr=%q", code, out, errs)
	}
}

func TestCursorSetNormalizesMessyValidBlockAndEditsFields(t *testing.T) {
	dir := cursorWriteRepo(t, "grill, prd")
	ledgerPath := filepath.Join(dir, ".superpowers", "sdd", "progress.md")
	if err := os.MkdirAll(filepath.Dir(ledgerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	messy := "<!-- spine:cursor -->\n effort:  old-effort  \nprd:   docs/specs/old.md\ntickets:  I001  \nstages:   grill[x]    prd[<]  \n<!-- /spine:cursor -->\n\nnotes stay below\n"
	if err := os.WriteFile(ledgerPath, []byte(messy), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := runCmd(t, "cursor", "set", "--dir", dir); code != 0 {
		t.Fatalf("initial no-op set: code=%d stderr=%q", code, errs)
	}
	canonicalOld := "<!-- spine:cursor -->\neffort: old-effort\nprd: docs/specs/old.md\ntickets: I001\nstages: grill[x] prd[<]\n<!-- /spine:cursor -->\n\nnotes stay below\n"
	if got := cursorLedger(t, dir); got != canonicalOld {
		t.Fatalf("no-op set did not canonicalize messy input:\n%s\nwant:\n%s", got, canonicalOld)
	}
	if code, _, errs := runCmd(t, "cursor", "set", "--dir", dir, "--prd", "docs/specs/new.md", "--tickets", "I002-I003"); code != 0 {
		t.Fatalf("field set: code=%d stderr=%q", code, errs)
	}
	want := "<!-- spine:cursor -->\neffort: old-effort\nprd: docs/specs/new.md\ntickets: I002-I003\nstages: grill[x] prd[<]\n<!-- /spine:cursor -->\n\nnotes stay below\n"
	if got := cursorLedger(t, dir); got != want {
		t.Fatalf("set bytes:\n%s\nwant:\n%s", got, want)
	}
	if code, _, errs := runCmd(t, "cursor", "set", "--dir", dir); code != 0 {
		t.Fatalf("no-op set: code=%d stderr=%q", code, errs)
	}
	if got := cursorLedger(t, dir); got != want {
		t.Fatalf("no-op set was not canonical: %q", got)
	}
}

func TestCursorWritersRejectNewlineValuesWithoutWriting(t *testing.T) {
	startDir := cursorWriteRepo(t, "grill, prd")
	code, _, errs := runCmd(t, "cursor", "start", "--dir", startDir, "--effort", "bad\neffort")
	if code == 0 || !strings.Contains(errs, "contains a newline") {
		t.Fatalf("newline start: code=%d stderr=%q", code, errs)
	}
	if _, err := os.Stat(filepath.Join(startDir, ".superpowers", "sdd", "progress.md")); !os.IsNotExist(err) {
		t.Fatalf("rejected start wrote a ledger: %v", err)
	}

	setDir := cursorWriteRepo(t, "grill, prd")
	if code, _, errs := runCmd(t, "cursor", "start", "--dir", setDir, "--effort", "valid"); code != 0 {
		t.Fatalf("setup start: code=%d stderr=%q", code, errs)
	}
	before := cursorLedger(t, setDir)
	code, _, errs = runCmd(t, "cursor", "set", "--dir", setDir, "--prd", "docs/specs/bad\rpath.md")
	if code == 0 || !strings.Contains(errs, "contains a newline") {
		t.Fatalf("newline set: code=%d stderr=%q", code, errs)
	}
	if got := cursorLedger(t, setDir); got != before {
		t.Fatalf("rejected set changed ledger:\n%s", got)
	}
}

// writeModelWorkflow mirrors internal/model's writeWorkflow fixture helper:
// a bare tempdir with only the dotted model_routing mirror the resolver
// reads, no other WORKFLOW.md scaffolding.
func writeModelWorkflow(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "WORKFLOW.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestModelBareIDIsDefaultOutsideAnyRepo(t *testing.T) {
	// t.TempDir() has no WORKFLOW.md at all — the "outside a spine repo"
	// case (design D11), which must resolve to the embedded default rather
	// than erroring.
	code, out, errs := runCmd(t, "model", "--dir", t.TempDir(), "claude", "primary")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if out != "claude-fable-5\n" {
		t.Fatalf("out=%q, want bare id on its own line with no decoration", out)
	}
}

func TestModelEffortFlagPrintsResolvedEffort(t *testing.T) {
	code, out, errs := runCmd(t, "model", "--dir", t.TempDir(), "--effort", "claude", "primary")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if out != "high\n" {
		t.Fatalf("out=%q, want the tier's resolved effort", out)
	}
}

func TestModelJSONFlagPrintsIDAndEffortTogether(t *testing.T) {
	code, out, errs := runCmd(t, "model", "--dir", t.TempDir(), "--json", "claude", "primary")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	var got struct {
		Flavor     string   `json:"flavor"`
		Tier       string   `json:"tier"`
		ID         string   `json:"id"`
		Effort     string   `json:"effort"`
		Aliases    []string `json:"aliases"`
		Provenance string   `json:"provenance"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("out=%q not valid JSON: %v", out, err)
	}
	if got.ID != "claude-fable-5" || got.Effort != "high" || got.Flavor != "claude" || got.Tier != "primary" {
		t.Errorf("got=%+v, out=%q", got, out)
	}
	if got.Provenance != "default" {
		t.Errorf("provenance=%q, want default (no repo override present)", got.Provenance)
	}
}

func TestModelOverrideInRepoWinsOverDefault(t *testing.T) {
	dir := writeModelWorkflow(t, "model_routing:\n  claude.primary: claude-custom-model\n")
	code, out, errs := runCmd(t, "model", "--dir", dir, "claude", "primary")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if out != "claude-custom-model\n" {
		t.Fatalf("out=%q, want the repo override id", out)
	}
}

func TestModelUnknownFlavorExitsNonZeroWithMessage(t *testing.T) {
	code, _, errs := runCmd(t, "model", "--dir", t.TempDir(), "bogus", "primary")
	if code == 0 || !strings.Contains(errs, "unknown flavor") {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
}

func TestModelUnknownTierExitsNonZeroWithMessage(t *testing.T) {
	code, _, errs := runCmd(t, "model", "--dir", t.TempDir(), "claude", "bogus")
	if code == 0 || !strings.Contains(errs, "unknown tier") {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
}

func TestModelMissingArgsExitsNonZero(t *testing.T) {
	for _, args := range [][]string{{"model"}, {"model", "claude"}, {"model", "claude", "primary", "extra"}} {
		code, _, errs := runCmd(t, args...)
		if code == 0 || !strings.Contains(errs, "usage: spine model") {
			t.Errorf("run(%v): code=%d stderr=%q", args, code, errs)
		}
	}
}

// I079 AC1: the pi harness resolves from the table today — each tier at its
// spec'd explicit effort, with the owner-tuned alternate available.
func TestModelPiTiersResolveWithExplicitEfforts(t *testing.T) {
	for _, tc := range []struct{ tier, effort string }{
		{"primary", "xhigh"},
		{"routine", "medium"},
		{"mechanical", "low"},
		{"fallback", "xhigh"},
	} {
		dir := t.TempDir()
		code, out, errs := runCmd(t, "model", "--dir", dir, "pi", tc.tier)
		if code != 0 || out != "qwen3.8-27b-q8_0\n" {
			t.Errorf("pi %s: code=%d out=%q stderr=%q", tc.tier, code, out, errs)
		}
		code, out, errs = runCmd(t, "model", "--dir", dir, "--effort", "pi", tc.tier)
		if code != 0 || out != tc.effort+"\n" {
			t.Errorf("pi %s --effort: code=%d out=%q stderr=%q", tc.tier, code, out, errs)
		}
	}
}

// I079 AC1: --alternate answers from the cell's alternate half — qwen @
// xhigh on every pi cell — so an evaluator's critic differs from its author
// by table data rather than a dispatch-time heuristic.
func TestModelPiAlternateReturnsOwnerTunedPair(t *testing.T) {
	dir := t.TempDir()
	code, out, errs := runCmd(t, "model", "--dir", dir, "--alternate", "pi", "routine")
	if code != 0 || out != "qwen3.8-27b-q8_0\n" {
		t.Fatalf("code=%d out=%q stderr=%q", code, out, errs)
	}
	code, out, errs = runCmd(t, "model", "--dir", dir, "--alternate", "--effort", "pi", "routine")
	if code != 0 || out != "xhigh\n" {
		t.Fatalf("--effort: code=%d out=%q stderr=%q", code, out, errs)
	}
}

// I079 AC1: --json carries the alternate when the cell has one.
func TestModelJSONCarriesAlternateForPi(t *testing.T) {
	code, out, errs := runCmd(t, "model", "--dir", t.TempDir(), "--json", "pi", "routine")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	var got struct {
		ID        string `json:"id"`
		Effort    string `json:"effort"`
		Alternate *struct {
			ID     string `json:"id"`
			Effort string `json:"effort"`
		} `json:"alternate"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("out=%q not valid JSON: %v", out, err)
	}
	if got.ID != "qwen3.8-27b-q8_0" || got.Effort != "medium" {
		t.Errorf("got=%+v, out=%q", got, out)
	}
	if got.Alternate == nil || got.Alternate.ID != "qwen3.8-27b-q8_0" || got.Alternate.Effort != "xhigh" {
		t.Errorf("alternate=%+v, want qwen3.8-27b-q8_0 @ xhigh", got.Alternate)
	}
}

// I079 AC1 negative control: a cell without an alternate says so rather than
// silently answering with its primary id — an evaluator that asked for the
// critic and got the author would run a model against itself.
func TestModelAlternateAbsentIsAnError(t *testing.T) {
	code, out, errs := runCmd(t, "model", "--dir", t.TempDir(), "--alternate", "claude", "primary")
	if code != 2 || !strings.Contains(errs, "has no alternate") {
		t.Fatalf("code=%d out=%q stderr=%q", code, out, errs)
	}
}

// I079 AC1 negative control: claude's --json output gains no alternate key.
func TestModelJSONOmitsAlternateWhenAbsent(t *testing.T) {
	code, out, errs := runCmd(t, "model", "--dir", t.TempDir(), "--json", "claude", "primary")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if strings.Contains(out, "alternate") {
		t.Fatalf("out=%q, want no alternate key for a cell that ships none", out)
	}
}

// I079 AC2: pi speaks low | medium | xhigh. A per-repo override asking pi
// for "high" fails with a message naming the vocabulary rather than being
// mapped onto some neighbouring level.
func TestModelPiRejectsHighEffortNamingItsVocabulary(t *testing.T) {
	dir := writeModelWorkflow(t, "model_routing:\n  pi.routine: qwen3.8-27b-q8_0 @ high\n")
	code, _, errs := runCmd(t, "model", "--dir", dir, "--effort", "pi", "routine")
	if code != 2 {
		t.Fatalf("code=%d, want 2; stderr=%q", code, errs)
	}
	for _, want := range []string{`effort "high"`, "pi effort vocabulary", "low, medium, xhigh"} {
		if !strings.Contains(errs, want) {
			t.Errorf("stderr=%q, want it to contain %q", errs, want)
		}
	}
}

// I079 AC2 negative control: the same override at a vocabulary effort
// resolves cleanly — the check rejects the word, not the override path.
func TestModelPiAcceptsVocabularyEffortOverride(t *testing.T) {
	dir := writeModelWorkflow(t, "model_routing:\n  pi.routine: qwen3.8-27b-q8_0 @ low\n")
	code, out, errs := runCmd(t, "model", "--dir", dir, "--effort", "pi", "routine")
	if code != 0 || out != "low\n" {
		t.Fatalf("code=%d out=%q stderr=%q", code, out, errs)
	}
}

// I079 AC3 at the resolver's own seam: an on-disk pi row spelled as the
// table ships it reports inherited; the same row with an edited alternate
// reports override, so the refresh rule leaves the edit alone.
func TestModelPiAlternateProvenance(t *testing.T) {
	for _, tc := range []struct{ row, want string }{
		{"qwen3.8-27b-q8_0 @ xhigh alt: qwen3.8-27b-q8_0 @ xhigh", "inherited"},
		{"qwen3.8-27b-q8_0 @ xhigh alt: qwen3.8-27b-q8_0 @ low", "override"},
		{"qwen3.8-27b-q8_0 @ xhigh", "override"},
	} {
		dir := writeModelWorkflow(t, "model_routing:\n  pi.primary: "+tc.row+"\n")
		code, out, errs := runCmd(t, "model", "--dir", dir, "--json", "pi", "primary")
		if code != 0 {
			t.Fatalf("row %q: code=%d stderr=%q", tc.row, code, errs)
		}
		var got struct {
			Provenance string `json:"provenance"`
		}
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("out=%q not valid JSON: %v", out, err)
		}
		if got.Provenance != tc.want {
			t.Errorf("row %q: provenance=%q, want %q", tc.row, got.Provenance, tc.want)
		}
	}
}

// I079 fix round 1: a bare-id pi mirror row — a plausible hand edit that
// just pins an id — inherits pi's OWN tier default (xhigh on primary and
// fallback), not the global "high" its vocabulary has no word for, so the
// row resolves instead of erroring. It reports override because dropping
// the cell's alternate is itself a deliberate edit.
func TestModelPiBareIDRowInheritsPiTierDefault(t *testing.T) {
	dir := writeModelWorkflow(t, "model_routing:\n  pi.primary: qwen3.8-27b-q8_0\n  pi.fallback: qwen3.8-27b-q8_0\n")
	for _, tier := range []string{"primary", "fallback"} {
		code, out, errs := runCmd(t, "model", "--dir", dir, "--effort", "pi", tier)
		if code != 0 || out != "xhigh\n" {
			t.Errorf("pi %s --effort: code=%d out=%q stderr=%q", tier, code, out, errs)
		}
		code, out, errs = runCmd(t, "model", "--dir", dir, "--json", "pi", tier)
		if code != 0 {
			t.Fatalf("pi %s --json: code=%d stderr=%q", tier, code, errs)
		}
		var got struct {
			ID         string `json:"id"`
			Effort     string `json:"effort"`
			Provenance string `json:"provenance"`
		}
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("out=%q not valid JSON: %v", out, err)
		}
		if got.ID != "qwen3.8-27b-q8_0" || got.Effort != "xhigh" || got.Provenance != "override" {
			t.Errorf("pi %s: got=%+v (want xhigh/override — the row dropped the cell's alternate)", tier, got)
		}
	}
}

// I079 fix round 1 negative control: claude has no per-flavor tier default,
// so a bare-id claude row still inherits the global "high" — the override is
// scoped to the flavor that declares one.
func TestModelClaudeBareIDRowStillInheritsGlobalTierDefault(t *testing.T) {
	dir := writeModelWorkflow(t, "model_routing:\n  claude.primary: claude-custom-model\n")
	code, out, errs := runCmd(t, "model", "--dir", dir, "--effort", "claude", "primary")
	if code != 0 || out != "high\n" {
		t.Fatalf("code=%d out=%q stderr=%q", code, out, errs)
	}
}

// AC (I085): docs/remediation/README.md is scaffolded by init, restored by
// update when missing, and states the remediation convention.
func TestRemediationReadmeScaffolded(t *testing.T) {
	dir := t.TempDir()
	if code, _, errs := runCmd(t, "init", "--dir", dir, "--profile", "rust", "--name", "demo"); code != 0 {
		t.Fatal(errs)
	}
	rel := filepath.Join("docs", "remediation", "README.md")
	raw, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		t.Fatalf("init did not scaffold %s: %v", rel, err)
	}
	for _, want := range []string{
		"docs/remediation/<effort>/round-N.md",
		"3 rounds per effort",
		"extension-ratified-by:",
		"spine audit stages",
		"findings-only",
		"prescriptive",
		"raw-review",
		"results-contract `code`",
		"hitlist.tmpl.md",
		"remediation-round.tmpl.md",
		"ADR 0007",
	} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("remediation README missing %q", want)
		}
	}
	if err := os.Remove(filepath.Join(dir, rel)); err != nil {
		t.Fatal(err)
	}
	if code, out, errs := runCmd(t, "update", "--dir", dir, "--write"); code != 0 {
		t.Fatalf("code=%d out=%q stderr=%q", code, out, errs)
	}
	if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
		t.Errorf("update did not restore %s: %v", rel, err)
	}
}

// AC (I085) at the CLI seam: opting into the pack renders the gate-go region
// into maipipe.toml; a repo that never opts in gets no maipipe.toml.
func TestGatePackRegionAtCLISeam(t *testing.T) {
	dir := t.TempDir()
	if code, _, errs := runCmd(t, "init", "--dir", dir, "--profile", "rust", "--name", "demo"); code != 0 {
		t.Fatal(errs)
	}
	if code, out, errs := runCmd(t, "update", "--dir", dir, "--write"); code != 0 {
		t.Fatalf("code=%d out=%q stderr=%q", code, out, errs)
	}
	if _, err := os.Stat(filepath.Join(dir, "maipipe.toml")); !os.IsNotExist(err) {
		t.Fatalf("maipipe.toml written without gate_pack (err=%v)", err)
	}
	wfPath := filepath.Join(dir, "WORKFLOW.md")
	raw, err := os.ReadFile(wfPath)
	if err != nil {
		t.Fatal(err)
	}
	opted := strings.Replace(string(raw), "gate_pack: ", "gate_pack: go@1", 1)
	if opted == string(raw) {
		t.Fatal("gate_pack: row not found in the scaffolded WORKFLOW.md")
	}
	if err := os.WriteFile(wfPath, []byte(opted), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errs := runCmd(t, "update", "--dir", dir, "--write")
	if code != 0 {
		t.Fatalf("code=%d out=%q stderr=%q", code, out, errs)
	}
	if !strings.Contains(out, "maipipe.toml") {
		t.Errorf("update output does not name maipipe.toml: %q", out)
	}
	got, err := os.ReadFile(filepath.Join(dir, "maipipe.toml"))
	if err != nil {
		t.Fatalf("region not rendered: %v", err)
	}
	for _, want := range []string{
		"# spine:begin gate-pack go@1\n",
		"[pipelines.gate-go]\nprofile = \"full\"\n",
		"run = \"spine gate go tskip\"\n",
		"# spine:end\n",
	} {
		if !strings.Contains(string(got), want) {
			t.Errorf("rendered region missing %q:\n%s", want, got)
		}
	}
	if code, out, _ := runCmd(t, "doctor", "--dir", dir); code != 0 {
		t.Errorf("doctor on a canonical region: code=%d out=%q", code, out)
	}
}

// remediationRepo stages a repo whose cursor names effort E, plus a newest
// handoff carrying the same cursor block so the stages backstop is satisfied
// — the fixture the round-budget advisory tests measure exit codes against.
func remediationRepo(t *testing.T, effort, prd, stages string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "WORKFLOW.md"), []byte("stages: [grill, prd]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	block := "<!-- spine:cursor -->\n" +
		"effort: " + effort + "\n" +
		"prd: " + prd + "\n" +
		"tickets: \n" +
		"stages: " + stages + "\n" +
		"<!-- /spine:cursor -->\n"
	ledger := filepath.Join(dir, ".superpowers", "sdd", "progress.md")
	if err := os.MkdirAll(filepath.Dir(ledger), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ledger, []byte(block), 0o644); err != nil {
		t.Fatal(err)
	}
	handoffs := filepath.Join(dir, "docs", "handoffs")
	if err := os.MkdirAll(handoffs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(handoffs, "2026-08-18-"+effort+".md"), []byte("# handoff\n\n"+block), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeRoundRecord(t *testing.T, dir, effort, name, frontmatter string) {
	t.Helper()
	roundDir := filepath.Join(dir, "docs", "remediation", effort)
	if err := os.MkdirAll(roundDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\n" + frontmatter + "hitlist: docs/remediation/" + effort + "/hitlist-1.md\n---\n\n# round\n"
	if err := os.WriteFile(filepath.Join(roundDir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// AC (I087): audit stages reports a round-4 record without ratification, and
// the exit code is identical to the same repo without the file — the rule is
// advisory and can never gate. Both a non-blocking and a blocking cursor are
// measured, so "unchanged" is proved in both directions.
func TestAuditStagesRoundBudgetAdvisoryNeverChangesExitCode(t *testing.T) {
	for _, tc := range []struct {
		name     string
		prd      string
		stages   string
		wantCode int
	}{
		{"non-blocking", "", "grill[<] prd[ ]", 0},
		{"blocking", "docs/specs/missing.md", "grill[x] prd[x]", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := remediationRepo(t, "budget-"+tc.name, tc.prd, tc.stages)
			baseCode, _, _ := runCmd(t, "audit", "stages", "--dir", base)
			if baseCode != tc.wantCode {
				t.Fatalf("fixture without the round record: code=%d want %d", baseCode, tc.wantCode)
			}

			dir := remediationRepo(t, "budget-"+tc.name, tc.prd, tc.stages)
			writeRoundRecord(t, dir, "budget-"+tc.name, "round-4.md", "round: 4\ndose: findings-only\n")
			code, out, errs := runCmd(t, "audit", "stages", "--dir", dir)
			if code != baseCode {
				t.Errorf("advisory changed the exit code: %d vs %d (out=%q err=%q)", code, baseCode, out, errs)
			}
			for _, want := range []string{
				"warning:",
				"docs/remediation/budget-" + tc.name + "/round-4.md",
				"beyond the 3-round remediation budget",
				"add `extension-ratified-by: <owner>` to its frontmatter",
			} {
				if !strings.Contains(errs, want) {
					t.Errorf("stderr missing %q: %q", want, errs)
				}
			}
		})
	}
}

// AC (I087) negative controls: rounds 1-3, a ratified round-4+, and no
// remediation directory at all are all silent.
func TestAuditStagesRoundBudgetNegativeControls(t *testing.T) {
	for _, tc := range []struct {
		name, file, frontmatter string
	}{
		{"round-3-is-within-budget", "round-3.md", "round: 3\ndose: findings-only\n"},
		{"round-4-ratified", "round-4.md", "round: 4\ndose: prescriptive\nextension-ratified-by: someone\n"},
		{"round-5-ratified", "round-5.md", "round: 5\ndose: raw-review\nextension-ratified-by: someone\n"},
		{"unnumbered-file-ignored", "notes.md", "round: 9\n"},
		{"commented-out-key-in-round-3", "round-3.md", "round: 3\n# extension-ratified-by: <owner>\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := remediationRepo(t, "quiet", "", "grill[<] prd[ ]")
			writeRoundRecord(t, dir, "quiet", tc.file, tc.frontmatter)
			code, out, errs := runCmd(t, "audit", "stages", "--dir", dir)
			if code != 0 {
				t.Fatalf("code=%d out=%q err=%q", code, out, errs)
			}
			if strings.Contains(errs, "remediation budget") {
				t.Errorf("negative control produced an advisory: %q", errs)
			}
		})
	}
	dir := remediationRepo(t, "no-dir", "", "grill[<] prd[ ]")
	if code, out, errs := runCmd(t, "audit", "stages", "--dir", dir); code != 0 || strings.Contains(errs, "remediation budget") {
		t.Errorf("no docs/remediation dir must be silent: code=%d out=%q err=%q", code, out, errs)
	}
}

// AC (I087): both templates are scaffolded into docs/remediation/ by init and
// restored by update — the same machine-owned simple-file ownership as the
// README — and the README names them and how to instantiate them.
func TestRemediationTemplatesScaffolded(t *testing.T) {
	dir := t.TempDir()
	if code, _, errs := runCmd(t, "init", "--dir", dir, "--profile", "rust", "--name", "demo"); code != 0 {
		t.Fatal(errs)
	}
	rels := []string{
		filepath.Join("docs", "remediation", "_hitlist.template.md"),
		filepath.Join("docs", "remediation", "_round.template.md"),
	}
	for _, rel := range rels {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Fatalf("init did not scaffold %s: %v", rel, err)
		}
		if err := os.Remove(filepath.Join(dir, rel)); err != nil {
			t.Fatal(err)
		}
	}
	if code, out, errs := runCmd(t, "update", "--dir", dir, "--write"); code != 0 {
		t.Fatalf("code=%d out=%q stderr=%q", code, out, errs)
	}
	for _, rel := range rels {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("update did not restore %s: %v", rel, err)
		}
	}
	readme, err := os.ReadFile(filepath.Join(dir, "docs", "remediation", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"docs/remediation/_hitlist.template.md",
		"docs/remediation/_round.template.md",
		"docs/remediation/<effort>/hitlist-N.md",
		"There is no\n`spine remediation new` verb",
	} {
		if !strings.Contains(string(readme), want) {
			t.Errorf("remediation README missing %q", want)
		}
	}
}

// AC (I087): the advisory fires when extension-ratified-by: is present but
// empty — the load-bearing half of the ratified negative control — and
// `spine doctor` stays untouched: the round-budget advisory is not a doctor
// finding and does not move doctor's exit code.
func TestRoundBudgetEmptyRatificationAdvisesAndDoctorIsUntouched(t *testing.T) {
	dir := remediationRepo(t, "empty-ratify", "", "grill[<] prd[ ]")
	doctorBefore, _, _ := runCmd(t, "doctor", "--dir", dir)
	writeRoundRecord(t, dir, "empty-ratify", "round-4.md", "round: 4\nextension-ratified-by:   \n")
	code, _, errs := runCmd(t, "audit", "stages", "--dir", dir)
	if code != 0 || !strings.Contains(errs, "beyond the 3-round remediation budget") {
		t.Errorf("empty ratification must still advise: code=%d err=%q", code, errs)
	}
	doctorAfter, out, _ := runCmd(t, "doctor", "--dir", dir)
	if doctorAfter != doctorBefore {
		t.Errorf("doctor exit code moved: %d -> %d (out=%q)", doctorBefore, doctorAfter, out)
	}
	if strings.Contains(out, "remediation budget") {
		t.Errorf("doctor must not report the round-budget advisory: %q", out)
	}
}

// Final review, Important 1: the dry-run plan prints the refusal --write
// would hit, before that file's diff — the plan is the review surface
// (ADR 0017), so the blocker cannot be something the reader discovers by
// running --write. Minor 6: a pass with no maipipe on PATH says which half of
// the pre-flight ran.
func TestUpdateDryRunPrintsTheRefusalBeforeTheDiff(t *testing.T) {
	dir := t.TempDir()
	if code, _, errs := runCmd(t, "init", "--dir", dir, "--profile", "go-service", "--name", "demo"); code != 0 {
		t.Fatal(errs)
	}
	setGateKeys(t, dir, "go@1", "[]")

	// A [pipelines.gate-go] table of the repo's own, which the region spine
	// appends then declares a second time.
	path := filepath.Join(dir, "maipipe.toml")
	if err := os.WriteFile(path, []byte("schema = 0\n\n[pipelines.gate-go]\nprofile = \"fast\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	code, out, _ := runCmd(t, "update", "--dir", dir)
	if code != 1 {
		t.Fatalf("dry-run code=%d out=%q", code, out)
	}
	refusal := strings.Index(out, "refusing to write")
	if refusal < 0 {
		t.Fatalf("dry-run plan does not show the refusal, out=%q", out)
	}
	if !strings.Contains(out, "duplicate table") {
		t.Errorf("refusal does not name the problem, out=%q", out)
	}
	if !strings.Contains(out, "--write would refuse this plan as a whole") {
		t.Errorf("plan does not say the refusal aborts the whole run, out=%q", out)
	}
	if diff := strings.Index(out, "--- maipipe.toml"); diff < 0 || diff < refusal {
		t.Errorf("refusal at %d is not printed before the diff at %d, out=%q", refusal, diff, out)
	}
	if after, err := os.ReadFile(path); err != nil || string(after) != string(before) {
		t.Errorf("a dry-run touched maipipe.toml (err=%v)", err)
	}
	// And --write does refuse, so the plan told the truth.
	if code, _, errs := runCmd(t, "update", "--dir", dir, "--write"); code != 2 ||
		!strings.Contains(errs, "no files were written") {
		t.Errorf("write code=%d err=%q; want a refusal naming the whole-run abort", code, errs)
	}
}

// setGateKeys rewrites the two gate_pack keys in a scaffolded WORKFLOW.md.
// Each key's value is replaced to the end of *its own* line, with any trailing
// comment kept: the earlier cut cut at the next "#" anywhere after the key and
// worked only because the template happens to carry a trailing comment on both
// of these lines (final review, Minor 5).
func setGateKeys(t *testing.T, dir, pack, disabled string) {
	t.Helper()
	path := filepath.Join(dir, "WORKFLOW.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"gate_pack:": pack, "gate_pack_disabled:": disabled}
	set := map[string]bool{}
	lines := strings.Split(string(raw), "\n")
	for i, line := range lines {
		for key, val := range want {
			if set[key] || !strings.HasPrefix(strings.TrimSpace(line), key) {
				continue
			}
			trailer := ""
			if j := strings.Index(line, "#"); j >= 0 {
				trailer = "    " + line[j:]
			}
			lines[i] = key + " " + val + trailer
			set[key] = true
		}
	}
	for key := range want {
		if !set[key] {
			t.Fatalf("scaffolded WORKFLOW.md has no %s key", key)
		}
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
}

// AC (I098): the update plan names the stages a render adds to the region
// already on disk, and the definition_hash re-approval they cost, before
// --write — a new stage in the gating lane is not something to discover
// from a diff.
func TestUpdatePlanFlagsAddedGateStages(t *testing.T) {
	dir := t.TempDir()
	if code, _, errs := runCmd(t, "init", "--dir", dir, "--profile", "go-service", "--name", "demo"); code != 0 {
		t.Fatal(errs)
	}
	setGateKeys(t, dir, "go@1", "[tskip, n-plus-one]")
	if code, out, errs := runCmd(t, "update", "--dir", dir, "--write"); code != 0 {
		t.Fatalf("write code=%d out=%q err=%q", code, out, errs)
	}

	setGateKeys(t, dir, "go@1", "[]")
	code, out, _ := runCmd(t, "update", "--dir", dir)
	if code != 1 {
		t.Fatalf("dry-run code=%d out=%q", code, out)
	}
	want := "maipipe.toml: this render adds 2 stage(s) not previously present: n-plus-one, tskip"
	if !strings.Contains(out, want) {
		t.Errorf("plan missing the added-stage line %q, out=%q", want, out)
	}
	if !strings.Contains(out, "maipipe gate approve-definition") {
		t.Errorf("plan does not name the definition_hash re-approval, out=%q", out)
	}
	// A plan that adds stages says nothing about dropping any.
	if strings.Contains(out, "stage(s) present today") {
		t.Errorf("plan reports dropped stages for an add-only render, out=%q", out)
	}
}
