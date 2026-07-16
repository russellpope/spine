package cursor_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/cursor"
)

func fixture(scenario string) string {
	return filepath.Join("testdata", scenario, "repo")
}

// writeFixtureFiles writes rel-path -> content pairs under dir, creating
// parent directories as needed.
func writeFixtureFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestValidCursorParses(t *testing.T) {
	res, err := cursor.Load(fixture("valid"))
	if err != nil {
		t.Fatal(err)
	}
	if !res.HasCursor {
		t.Fatal("want HasCursor true")
	}
	if len(res.Findings) != 0 {
		t.Fatalf("want no findings, got %#v", res.Findings)
	}
	c := res.Cursor
	if c.Effort != "fixture-effort" {
		t.Errorf("Effort = %q", c.Effort)
	}
	if c.PRD != "docs/specs/2026-01-01-fixture-design.md" {
		t.Errorf("PRD = %q", c.PRD)
	}
	if c.Tickets != "I001-I005" {
		t.Errorf("Tickets = %q", c.Tickets)
	}
	if len(c.Stages) != 11 {
		t.Fatalf("want 11 stages, got %d: %#v", len(c.Stages), c.Stages)
	}
	if c.Stages[0].Name != "grill" || c.Stages[0].State != cursor.Done {
		t.Errorf("Stages[0] = %#v", c.Stages[0])
	}
	if c.Stages[3].Name != "implement" || c.Stages[3].State != cursor.Here {
		t.Errorf("Stages[3] = %#v", c.Stages[3])
	}
	if c.Stages[4].Name != "functional-test" || c.Stages[4].State != cursor.Pending {
		t.Errorf("Stages[4] = %#v", c.Stages[4])
	}
}

func TestMalformedBlockMissingKeyIsFinding(t *testing.T) {
	res, err := cursor.Load(fixture("malformed"))
	if err != nil {
		t.Fatal(err)
	}
	if !res.HasCursor {
		t.Fatal("want HasCursor true — block was present, just malformed")
	}
	if len(res.Findings) == 0 {
		t.Fatal("want a finding for the missing `tickets` key")
	}
	joined := strings.Join(res.Findings, "; ")
	if !strings.Contains(joined, "tickets") {
		t.Errorf("findings = %#v, want mention of missing tickets key", res.Findings)
	}
}

func TestMissingFileNoCursorNoError(t *testing.T) {
	res, err := cursor.Load(fixture("missing"))
	if err != nil {
		t.Fatal(err)
	}
	if res.HasCursor {
		t.Fatalf("want HasCursor false, got %#v", res)
	}
	if len(res.Findings) != 0 {
		t.Errorf("want no findings for a repo with no progress.md, got %#v", res.Findings)
	}
}

func TestNotASpineRepoNoCursorNoError(t *testing.T) {
	res, err := cursor.Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if res.HasCursor {
		t.Fatalf("want HasCursor false for an empty dir, got %#v", res)
	}
}

func TestTwoHereMarkersIsFinding(t *testing.T) {
	res, err := cursor.Load(fixture("two-here"))
	if err != nil {
		t.Fatal(err)
	}
	if !res.HasCursor {
		t.Fatal("want HasCursor true")
	}
	var found bool
	for _, f := range res.Findings {
		if strings.Contains(f, "YOU-ARE-HERE") || strings.Contains(f, "[<]") {
			found = true
		}
	}
	if !found {
		t.Fatalf("want a finding about multiple HERE markers, got %#v", res.Findings)
	}
}

func TestUnknownStageNameIsFinding(t *testing.T) {
	res, err := cursor.Load(fixture("unknown-stage"))
	if err != nil {
		t.Fatal(err)
	}
	if !res.HasCursor {
		t.Fatal("want HasCursor true")
	}
	var found bool
	for _, f := range res.Findings {
		if strings.Contains(f, "packaging") {
			found = true
		}
	}
	if !found {
		t.Fatalf("want a finding naming the unknown stage %q, got %#v", "packaging", res.Findings)
	}
}

// This repo's own ledger dogfoods the I018 grammar; the parser must accept
// it cleanly, matching the plan's requirement that the parser reconcile
// against the real ledger, not just synthetic fixtures.
func TestDogfoodLedgerParsesCleanly(t *testing.T) {
	res, err := cursor.Load(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if !res.HasCursor {
		t.Fatal("want the repo's own .superpowers/sdd/progress.md to parse as a cursor")
	}
	if len(res.Findings) != 0 {
		t.Fatalf("want the dogfood ledger to parse with no findings, got %#v", res.Findings)
	}
	if res.Cursor.Effort != "stage-cursor-controls" {
		t.Errorf("Effort = %q", res.Cursor.Effort)
	}
}

func TestUnterminatedBlockIsFinding(t *testing.T) {
	dir := t.TempDir()
	writeFixtureFiles(t, dir, map[string]string{
		"WORKFLOW.md": "stages: [grill, prd, issues, implement]\n",
		".superpowers/sdd/progress.md": "<!-- spine:cursor -->\n" +
			"effort: x\nprd: docs/specs/x.md\ntickets: I001\nstages: grill[<] prd[ ] issues[ ] implement[ ]\n",
	})
	res, err := cursor.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !res.HasCursor {
		t.Fatal("want HasCursor true — an open marker was found")
	}
	if len(res.Findings) == 0 {
		t.Fatal("want a finding for the missing closing marker")
	}
}
