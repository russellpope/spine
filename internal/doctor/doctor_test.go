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
