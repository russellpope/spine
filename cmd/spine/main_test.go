package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	if code != 0 || !strings.Contains(out, "spine template generation 7") {
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
	if code != 1 || !strings.Contains(out, "+ template_version: 7") {
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
	if !strings.Contains(out, "+ template_version: 7") {
		t.Errorf("dry-run text output missing diff content: out=%q", out)
	}
	// --json must never carry the diff text as loose prose in the payload
	// stream; the JSON test above already checks the stream is pure JSON,
	// this just confirms diffs are a text-mode-only addition.
	_, jsonOut, _ := runCmd(t, "adopt", "--dir", dir, "--json")
	if strings.Contains(jsonOut, "+ template_version: 7\n") {
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
	// clean fixture: all match, exit 0
	code, out, errs := runCmd(t, "audit", "routing",
		"--dir", fixture("clean", "repo"), "--transcripts", fixture("clean", "transcripts"))
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
		"--dir", fixture("mixed", "repo"), "--transcripts", fixture("mixed", "transcripts"))
	if code != 1 || !strings.Contains(out, "silent-descent") {
		t.Fatalf("mixed: code=%d (want 1) out=%q", code, out)
	}
	if !strings.Contains(out, "housekeeping") {
		t.Errorf("mixed out should list the unmatched dispatch: %q", out)
	}
	// degraded fixture: warnings on stderr, exit 0
	code, _, errs = runCmd(t, "audit", "routing",
		"--dir", fixture("degraded", "repo"), "--transcripts", fixture("degraded", "transcripts"))
	if code != 0 || !strings.Contains(errs, "warning:") || !strings.Contains(errs, "bad.jsonl") {
		t.Fatalf("degraded: code=%d errs=%q", code, errs)
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
		"derivation: n/a",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("out missing %q; out=%q", want, out)
		}
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

func TestCursorCommandOnRealRepoLedger(t *testing.T) {
	code, out, errs := runCmd(t, "cursor", "--dir", filepath.Join("..", ".."))
	if code != 0 {
		t.Fatalf("code=%d errs=%q", code, errs)
	}
	if !strings.Contains(out, "effort: stage-cursor-controls") || !strings.Contains(out, "derivation: n/a") {
		t.Errorf("out=%q", out)
	}
}
