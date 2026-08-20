package doctor_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/russellpope/spine/internal/adr"
	"github.com/russellpope/spine/internal/doctor"
	"github.com/russellpope/spine/internal/eval"
	"github.com/russellpope/spine/internal/scaffold"
	"github.com/russellpope/spine/internal/tmpl"
	"github.com/russellpope/spine/internal/update"
)

func ids(fs []doctor.Finding) map[string]int {
	m := map[string]int{}
	for _, f := range fs {
		m[f.ID]++
	}
	return m
}

func TestCleanScaffoldNoFindings(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Fatalf("want clean, got %#v", fs)
	}
}

func TestMissingPiecesD1(t *testing.T) {
	fs, err := doctor.Run(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if ids(fs)["D1"] == 0 {
		t.Fatalf("want D1 findings, got %#v", fs)
	}
}

func TestStaleGen0D2AndD3(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	// regress to a TRUE gen0 repo by rendering the gen0 templates (stripping
	// the stamp from a current file would read as unrecognized edits instead)
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
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := ids(fs)
	if got["D2"] == 0 || got["D3"] == 0 {
		t.Fatalf("want D2 (stale, pending update) + D3 (no markers), got %#v", fs)
	}
}

// Both markers present exactly once but in swapped order must be treated as
// damage — counts alone (begins==1, ends==1) previously passed silently.
func TestOutOfOrderMarkersD3Error(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "CLAUDE.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(raw), "\n")
	var beginIdx, endIdx = -1, -1
	for i, l := range lines {
		if strings.HasPrefix(l, "<!-- spine:begin") {
			beginIdx = i
		}
		if strings.HasPrefix(l, "<!-- spine:end -->") {
			endIdx = i
		}
	}
	if beginIdx == -1 || endIdx == -1 {
		t.Fatalf("scaffolded CLAUDE.md missing markers: %q", string(raw))
	}
	lines[beginIdx], lines[endIdx] = lines[endIdx], lines[beginIdx]
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, f := range fs {
		if f.ID != "D3" {
			continue
		}
		found = true
		if f.Severity != "error" || f.Message != "spine markers out of order — fix by hand" {
			t.Errorf("D3 finding = %#v", f)
		}
	}
	if !found {
		t.Fatalf("want D3 finding, got %#v", fs)
	}
}

// Marker damage (unbalanced) must not suggest --force in the D4 message,
// since --force cannot repair CLAUDE.md's marker block.
func TestMarkerDamageD4Message(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "CLAUDE.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(raw), "<!-- spine:end -->\n", "", 1)
	if broken == string(raw) {
		t.Fatal("end marker line not found to delete")
	}
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, f := range fs {
		if f.ID != "D4" || f.Path != "CLAUDE.md" {
			continue
		}
		found = true
		want := "spine markers damaged — fix by hand (--force cannot repair)"
		if f.Message != want {
			t.Errorf("D4 message = %q, want %q", f.Message, want)
		}
		if strings.Contains(f.Message, "--force") && !strings.Contains(f.Message, "cannot repair") {
			t.Errorf("D4 message must not offer --force as a repair: %q", f.Message)
		}
	}
	if !found {
		t.Fatalf("want D4 finding for CLAUDE.md, got %#v", fs)
	}
}

// Analog of TestOutOfOrderMarkersD3Error for AGENTS.md — the marker check
// must run over AGENTS.md too, not just CLAUDE.md.
func TestOutOfOrderMarkersD3ErrorAgents(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "AGENTS.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(raw), "\n")
	var beginIdx, endIdx = -1, -1
	for i, l := range lines {
		if strings.HasPrefix(l, "<!-- spine:begin") {
			beginIdx = i
		}
		if strings.HasPrefix(l, "<!-- spine:end -->") {
			endIdx = i
		}
	}
	if beginIdx == -1 || endIdx == -1 {
		t.Fatalf("scaffolded AGENTS.md missing markers: %q", string(raw))
	}
	lines[beginIdx], lines[endIdx] = lines[endIdx], lines[beginIdx]
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, f := range fs {
		if f.ID != "D3" || f.Path != "AGENTS.md" {
			continue
		}
		found = true
		if f.Severity != "error" || f.Message != "spine markers out of order — fix by hand" {
			t.Errorf("D3 finding = %#v", f)
		}
	}
	if !found {
		t.Fatalf("want D3 finding for AGENTS.md, got %#v", fs)
	}
}

// Analog of TestMarkerDamageD4Message for AGENTS.md — --force cannot repair
// AGENTS.md's marker block either, so the hint must not be the generic one.
func TestMarkerDamageD4MessageAgents(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "AGENTS.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(raw), "<!-- spine:end -->\n", "", 1)
	if broken == string(raw) {
		t.Fatal("end marker line not found to delete")
	}
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, f := range fs {
		if f.ID != "D4" || f.Path != "AGENTS.md" {
			continue
		}
		found = true
		want := "spine markers damaged — fix by hand (--force cannot repair)"
		if f.Message != want {
			t.Errorf("D4 message = %q, want %q", f.Message, want)
		}
		if strings.Contains(f.Message, "--force") && !strings.Contains(f.Message, "cannot repair") {
			t.Errorf("D4 message must not offer --force as a repair: %q", f.Message)
		}
	}
	if !found {
		t.Fatalf("want D4 finding for AGENTS.md, got %#v", fs)
	}
}

func TestSuperpowersDriftD5(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	sp := filepath.Join(dir, "docs", "superpowers", "plans")
	os.MkdirAll(sp, 0o755)
	os.WriteFile(filepath.Join(sp, "old-plan.md"), []byte("x"), 0o644)
	fs, _ := doctor.Run(dir)
	if ids(fs)["D5"] != 1 {
		t.Fatalf("want one D5, got %#v", fs)
	}
}

func TestUnrecognizedEditsD4(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	wf := filepath.Join(dir, "WORKFLOW.md")
	raw, err := os.ReadFile(wf)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wf, append(raw, []byte("custom_rule: never deploy fridays\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ids(fs)["D4"] == 0 {
		t.Fatalf("want D4 finding for unrecognized edit, got %#v", fs)
	}
}

// C1: a hand-authored docs/adr/README.md (praxis-style index) must be
// reported as D4 info — "preserved", not warn/skip — and must not also
// trigger the generic unrecognized-edits warn.
func TestPreservedADRReadmeD4Info(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "go-service", "demo"); err != nil {
		t.Fatal(err)
	}
	handAuthored := "# Architecture Decision Records\n\nSee the index below.\n\n| # | Decision |\n|---|---|\n| 0001 | Something |\n"
	if err := os.WriteFile(filepath.Join(dir, "docs", "adr", "README.md"), []byte(handAuthored), 0o644); err != nil {
		t.Fatal(err)
	}
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, f := range fs {
		if f.Path != "docs/adr/README.md" {
			continue
		}
		found = true
		if f.ID != "D4" || f.Severity != "info" {
			t.Errorf("finding = %#v, want D4 info", f)
		}
		if !strings.Contains(f.Message, "preserved") || !strings.Contains(f.Message, "--force") {
			t.Errorf("message = %q, want mention of preserved + --force", f.Message)
		}
	}
	if !found {
		t.Fatalf("want a finding for docs/adr/README.md, got %#v", fs)
	}
	for _, f := range fs {
		if f.Severity == "warn" || f.Severity == "error" {
			t.Errorf("preserved ADR README must not also warn/error: %#v", f)
		}
	}
}

func TestLegacyADRNoFrontMatterD6Info(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join("testdata", "legacy-adr.md"))
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "docs", "adr", "0001-legacy.md")
	if err := os.WriteFile(dst, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, f := range fs {
		if f.ID != "D6" || f.Path != dst {
			continue
		}
		found = true
		if f.Severity != "info" {
			t.Errorf("severity = %q, want info", f.Severity)
		}
	}
	if !found {
		t.Fatalf("want D6 finding for legacy (no front matter) ADR, got %#v", fs)
	}
}

func TestD1ProfileAwareKnowledge(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "knowledge", "vault"); err != nil {
		t.Fatal(err)
	}
	findings, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		if f.ID == "D1" {
			t.Errorf("unexpected D1 on fresh knowledge repo: %+v", f)
		}
	}
}

func TestADRProblemsD6(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	adr.New(dir, "Real one", 0)
	// duplicate number + bogus status
	os.WriteFile(filepath.Join(dir, "docs", "adr", "0001-dupe.md"),
		[]byte("---\nid: 0001\ntitle: Dupe\nstatus: Draft\ndate: 2026-07-01\n---\n"), 0o644)
	fs, _ := doctor.Run(dir)
	got := ids(fs)
	if got["D6"] < 2 {
		t.Fatalf("want duplicate+status D6 findings, got %#v", fs)
	}
}

func TestD7EvalStructure(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	if _, err := eval.New(dir, "demo eval"); err != nil {
		t.Fatal(err)
	}
	// well-formed: no D7
	findings, _ := doctor.Run(dir)
	for _, f := range findings {
		if f.ID == "D7" {
			t.Fatalf("unexpected D7: %+v", f)
		}
	}
	// malformed run: D7 warn
	today := time.Now().Format("2006-01-02")
	bad := filepath.Join(dir, "docs", "evals", today+"-demo-eval", "runs", "broken.md")
	if err := os.WriteFile(bad, []byte("no front matter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, _ = doctor.Run(dir)
	found := false
	for _, f := range findings {
		if f.ID == "D7" && f.Severity == "warn" {
			found = true
		}
	}
	if !found {
		t.Fatalf("want D7 warn, findings=%+v", findings)
	}
}

// D9 (the stage-derivation advisory, I019) must stay silent on a freshly
// scaffolded repo: no progress.md at all is a dormant/non-SDD-effort repo,
// not unhealthy.
func TestD9SilentWithNoCursor(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	findings, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ids(findings)["D9"] != 0 {
		t.Fatalf("want no D9 on a dormant repo, got %#v", findings)
	}
}

// A cursor whose ticked stages all have matching artifacts, and whose
// newest handoff carries the cursor block, must not produce a D9 finding.
func TestD9SilentOnCleanCursor(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	seedCleanCursor(t, dir)
	findings, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ids(findings)["D9"] != 0 {
		t.Fatalf("want no D9 on a clean cursor, got %#v", findings)
	}
}

// A stage-cursor mismatch (ticked done with no matching artifact) must
// produce a D9 finding, severity warn — never error — and must not change
// doctor's existing warn/error-drives-exit-code rule.
func TestD9WarnOnTickedMissingStage(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	seedCleanCursor(t, dir)
	// Remove the PRD file the cursor claims is done — a ticked-but-missing
	// contradiction.
	if err := os.Remove(filepath.Join(dir, "docs", "specs", "2026-01-01-fixture-design.md")); err != nil {
		t.Fatal(err)
	}
	findings, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, f := range findings {
		if f.ID != "D9" {
			continue
		}
		found = true
		if f.Severity != "warn" {
			t.Errorf("D9 severity = %q, want warn (never error)", f.Severity)
		}
	}
	if !found {
		t.Fatalf("want a D9 finding for the ticked-but-missing PRD, got %#v", findings)
	}
	code := 0
	for _, f := range findings {
		if f.Severity == "warn" || f.Severity == "error" {
			code = 1
		}
	}
	if code != 1 {
		t.Error("existing doctor exit rule (warn/error -> 1) must still apply with D9 present")
	}
}

// I024: a cursor block whose stages: line is grammar-garbage (zero parsed
// stage rows, cursor.Result.Findings non-empty) must surface as a D9 warn —
// previously doctor never surfaced grammar-level CursorFindings at all,
// even though `spine audit stages` was already blocking on the equivalent
// fixture (internal/stages/testdata/malformed-cursor). The handoff still
// carries the cursor block here, so the newest-handoff backstop stays
// non-blocking — the only D9 finding this repo should produce is the new
// grammar one.
func TestD9WarnOnMalformedCursorGrammar(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	cursorBlock := "<!-- spine:cursor -->\n" +
		"effort: fixture-effort\n" +
		"prd: docs/specs/2026-01-01-fixture-design.md\n" +
		"tickets: I001-I001\n" +
		"stages: ??? *** !!!\n" +
		"<!-- /spine:cursor -->\n"
	writeUnder(t, dir, ".superpowers/sdd/progress.md", "# ledger\n\n"+cursorBlock)
	writeUnder(t, dir, "docs/handoffs/2026-01-02-fixture.md", "---\ntitle: \"fixture\"\n---\n\n"+cursorBlock)
	findings, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, f := range findings {
		if f.ID != "D9" {
			continue
		}
		if f.Severity != "warn" {
			t.Errorf("D9 finding severity = %q, want warn (never error): %#v", f.Severity, f)
		}
		if strings.Contains(f.Message, "malformed stage token") {
			found = true
		}
	}
	if !found {
		t.Fatalf("want a D9 warn naming the malformed cursor grammar, got %#v", findings)
	}
}

// F1 (final whole-branch review, I024-I027 batch): an unresolvable
// tickets: value (I026's Report.Notes entry) previously never reached
// doctor at all — this fixture's stages: grammar and handoff both resolve
// cleanly, so before this fix doctor reported zero D9 findings on a repo
// whose tickets: value silently degraded issues/implement evidence to
// not-judged. D9 stays warn-only, matching every other check.
func TestD9WarnOnUnresolvableTicketsNote(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	cursorBlock := "<!-- spine:cursor -->\n" +
		"effort: fixture-effort\n" +
		"prd: docs/specs/2026-01-01-fixture-design.md\n" +
		"tickets: not-a-grammar\n" +
		"stages: grill[x] prd[x] issues[x] implement[<]\n" +
		"<!-- /spine:cursor -->\n"
	writeUnder(t, dir, ".superpowers/sdd/progress.md", "# ledger\n\n"+cursorBlock)
	writeUnder(t, dir, "docs/specs/2026-01-01-fixture-design.md", "# fixture design\n")
	writeUnder(t, dir, "docs/handoffs/2026-01-02-fixture.md", "---\ntitle: \"fixture\"\n---\n\n"+cursorBlock)
	findings, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, f := range findings {
		if f.ID != "D9" {
			continue
		}
		if f.Severity != "warn" {
			t.Errorf("D9 finding severity = %q, want warn (never error): %#v", f.Severity, f)
		}
		if strings.Contains(f.Message, "not-a-grammar") {
			found = true
		}
	}
	if !found {
		t.Fatalf("want a D9 warn naming the unresolvable tickets: value, got %#v", findings)
	}
}

// seedCleanCursor writes a matching cursor + PRD + ticket files + a handoff
// carrying the cursor block into a scaffolded dir, so a stage-derivation
// check over it comes back clean.
func seedCleanCursor(t *testing.T, dir string) {
	t.Helper()
	cursorBlock := "<!-- spine:cursor -->\n" +
		"effort: fixture-effort\n" +
		"prd: docs/specs/2026-01-01-fixture-design.md\n" +
		"tickets: I001-I001\n" +
		"stages: grill[x] prd[x] issues[x] implement[<]\n" +
		"<!-- /spine:cursor -->\n"
	writeUnder(t, dir, ".superpowers/sdd/progress.md", "# ledger\n\n"+cursorBlock)
	writeUnder(t, dir, "docs/specs/2026-01-01-fixture-design.md", "# fixture design\n")
	writeUnder(t, dir, "docs/issues/I001-a.md", "---\nid: I001\ntitle: fixture\nseverity: low\nstatus: fixed\n---\nx\n")
	writeUnder(t, dir, "docs/handoffs/2026-01-02-fixture.md", "---\ntitle: \"fixture\"\n---\n\n"+cursorBlock)
}

func writeUnder(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestD8HandoffNaming(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "handoffs", "notes.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, _ := doctor.Run(dir)
	found := false
	for _, f := range findings {
		if f.ID == "D8" {
			found = true
			if f.Severity != "info" {
				t.Errorf("D8 must be info, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Fatalf("want D8, findings=%+v", findings)
	}
}

// gatePackRepo scaffolds a repo, opts it into the gate pack, and renders the
// region — the canonical starting point for the D10 checks.
func gatePackRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "WORKFLOW.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	opted := strings.Replace(string(raw), "gate_pack: ", "gate_pack: go@1", 1)
	if opted == string(raw) {
		t.Fatal("gate_pack: row not found in the scaffolded WORKFLOW.md")
	}
	if err := os.WriteFile(path, []byte(opted), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := update.Run(update.Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	return dir
}

// AC (I085) negative control: a canonical gate-pack region is silent.
func TestD10SilentOnCanonicalRegion(t *testing.T) {
	fs, err := doctor.Run(gatePackRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Fatalf("want clean, got %#v", fs)
	}
}

// AC (I085): doctor fires on a broken marker (begin without end).
func TestD10BrokenMarker(t *testing.T) {
	dir := gatePackRepo(t)
	path := filepath.Join(dir, update.MaipipeFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(raw), "# spine:end\n", "", 1)
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []doctor.Finding
	for _, f := range fs {
		if f.ID == "D10" {
			got = append(got, f)
		}
	}
	if len(got) != 1 || got[0].Severity != "error" || !strings.Contains(got[0].Message, "unbalanced") {
		t.Fatalf("want one D10 error about unbalanced markers, got %#v (all: %#v)", got, fs)
	}
	if ids(fs)["D4"] != 0 {
		t.Errorf("region health double-reported as D4: %#v", fs)
	}
}

// AC (I085): drifted region content is non-canonical for the pinned pack.
func TestD10NonCanonicalRegionContent(t *testing.T) {
	dir := gatePackRepo(t)
	path := filepath.Join(dir, update.MaipipeFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	drifted := strings.Replace(string(raw), `run = "spine gate go tskip"`, `run = "echo tskip"`, 1)
	if drifted == string(raw) {
		t.Fatal("tskip stage not found in the rendered region")
	}
	if err := os.WriteFile(path, []byte(drifted), 0o644); err != nil {
		t.Fatal(err)
	}
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range fs {
		if f.ID == "D10" {
			found = true
			if f.Severity != "warn" || !strings.Contains(f.Message, "not canonical") {
				t.Errorf("D10 = %#v, want a warn about non-canonical content", f)
			}
		}
	}
	if !found {
		t.Fatalf("want a D10 finding, got %#v", fs)
	}
}

// Negative control: a repo that never opted in has no region and no D10.
func TestD10SilentWithoutGatePack(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ids(fs)["D10"] != 0 {
		t.Fatalf("D10 fired without gate_pack: %#v", fs)
	}
}

// setWorkflowKey rewrites the value of one `key: value` row of a repo's
// WORKFLOW.md, preserving any trailing `#` comment so the row stays the
// template's — the point of these fixtures is a changed value, not a
// changed file shape.
func setWorkflowKey(t *testing.T, dir, key, value string) {
	t.Helper()
	path := filepath.Join(dir, "WORKFLOW.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(raw), "\n")
	found := false
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), key+":") {
			row := key + ": " + value
			// Reproduce the renderer's spacing for a value-filled row —
			// four spaces before the trailing comment — so WORKFLOW.md
			// stays canonical and its own drift finding cannot mask the
			// gate-pack finding under test.
			if c := strings.Index(l, "#"); c >= 0 {
				row += "    " + l[c:]
			}
			lines[i] = row
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("%s: row not found in WORKFLOW.md", key)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
}

// AC (I099): gate_pack set but no region at all — D10 warn, region missing.
func TestD10RegionMissing(t *testing.T) {
	dir := gatePackRepo(t)
	if err := os.Remove(filepath.Join(dir, update.MaipipeFile)); err != nil {
		t.Fatal(err)
	}
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []doctor.Finding
	for _, f := range fs {
		if f.ID == "D10" {
			got = append(got, f)
		}
	}
	if len(got) != 1 || got[0].Severity != "warn" || !strings.Contains(got[0].Message, "region is missing") {
		t.Fatalf("want one D10 warn about a missing region, got %#v (all: %#v)", got, fs)
	}
	if got[0].Path != update.MaipipeFile {
		t.Errorf("D10 path = %q, want %q", got[0].Path, update.MaipipeFile)
	}
}

// AC (I099): a region that is well formed and made of recognized pack lines
// but is no longer what the pack renders (here: WORKFLOW.md disabled a class
// after the region was written) — D10 warn, phrased as a difference rather
// than a cause, since under I095 reading (A) doctor keeps no record of what
// spine last rendered and so cannot tell the two causes apart.
func TestD10RegionDiffersFromRendering(t *testing.T) {
	dir := gatePackRepo(t)
	setWorkflowKey(t, dir, "gate_pack_disabled", "[tskip]")
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []doctor.Finding
	for _, f := range fs {
		if f.ID == "D10" {
			got = append(got, f)
		}
	}
	if len(got) != 1 || got[0].Severity != "warn" ||
		!strings.Contains(got[0].Message, "differs from what the pinned pack renders") {
		t.Fatalf("want one D10 warn about a differing region, got %#v (all: %#v)", got, fs)
	}
	if len(fs) != 1 {
		t.Errorf("region drift produced collateral findings: %#v", fs)
	}
}

// AC (I099) negative control: an unknown gate_pack: value is a WORKFLOW.md
// defect with a WORKFLOW.md remedy, so it surfaces as a D4 error on
// WORKFLOW.md and no longer as a D10 error wearing a maipipe.toml path.
func TestUnknownGatePackIsD4OnWorkflowNotD10(t *testing.T) {
	dir := gatePackRepo(t)
	setWorkflowKey(t, dir, "gate_pack", "go@99")
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []doctor.Finding
	for _, f := range fs {
		if strings.Contains(f.Message, "is not a pack this spine binary ships") {
			got = append(got, f)
		}
	}
	if len(got) != 1 {
		t.Fatalf("want exactly one unknown-pack finding, got %#v (all: %#v)", got, fs)
	}
	if got[0].ID != "D4" || got[0].Severity != "error" || got[0].Path != "WORKFLOW.md" {
		t.Errorf("unknown-pack finding = %#v, want D4 error on WORKFLOW.md", got[0])
	}
	for _, f := range fs {
		if f.ID == "D10" {
			t.Errorf("D10 still fires for an unknown pack: %#v", f)
		}
	}
	if len(fs) != 1 {
		t.Errorf("unknown pack produced collateral findings: %#v", fs)
	}
}
