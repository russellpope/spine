package stages_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/cursor"
	"github.com/russellpope/spine/internal/stages"
)

func fixture(scenario string) string {
	return filepath.Join("testdata", scenario, "repo")
}

func rowByName(t *testing.T, rows []stages.StageRow, name string) stages.StageRow {
	t.Helper()
	for _, r := range rows {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("no stage row named %q in %#v", name, rows)
	return stages.StageRow{}
}

// Acceptance: a cursor whose ticked stages all have matching artifacts and
// whose newest handoff carries the cursor block derives cleanly — nothing
// blocks.
func TestCleanCursorDerivesMatchNotBlocking(t *testing.T) {
	rep, err := stages.Derive(fixture("clean"))
	if err != nil {
		t.Fatal(err)
	}
	if !rep.HasCursor {
		t.Fatal("want HasCursor true")
	}
	for _, name := range []string{"prd", "issues", "implement"} {
		row := rowByName(t, rep.Stages, name)
		if row.Verdict != stages.VerdictMatch {
			t.Errorf("%s: verdict = %s (%s), want match", name, row.Verdict, row.Detail)
		}
	}
	if !rep.Handoff.Applicable || !rep.Handoff.HasBlock {
		t.Errorf("Handoff = %#v, want applicable with the cursor block present", rep.Handoff)
	}
	if rep.Blocking() {
		t.Errorf("clean fixture must not be blocking: stages=%#v handoff=%#v", rep.Stages, rep.Handoff)
	}
}

// Story 6: a cursor that claims prd done with no PRD file on disk is a
// contradiction — ticked-but-missing blocks.
func TestTickedButMissingBlocks(t *testing.T) {
	rep, err := stages.Derive(fixture("ticked-missing"))
	if err != nil {
		t.Fatal(err)
	}
	prd := rowByName(t, rep.Stages, "prd")
	if prd.Verdict != stages.VerdictTickedMissing {
		t.Errorf("prd verdict = %s (%s), want ticked-missing", prd.Verdict, prd.Detail)
	}
	if !rep.Blocking() {
		t.Error("want Blocking() true when a ticked stage's artifact is missing")
	}
	// issues is pending with no ticket files on disk — absence never blocks.
	issues := rowByName(t, rep.Stages, "issues")
	if issues.Verdict != stages.VerdictMatch {
		t.Errorf("issues verdict = %s (%s), want match (absence never blocks)", issues.Verdict, issues.Detail)
	}
}

// Story 7: tickets exist on disk but the issues stage is still marked
// pending — a stale cursor, present-but-unticked blocks.
func TestPresentButUntickedBlocks(t *testing.T) {
	rep, err := stages.Derive(fixture("present-unticked"))
	if err != nil {
		t.Fatal(err)
	}
	issues := rowByName(t, rep.Stages, "issues")
	if issues.Verdict != stages.VerdictPresentUnticked {
		t.Errorf("issues verdict = %s (%s), want present-unticked", issues.Verdict, issues.Detail)
	}
	if !rep.Blocking() {
		t.Error("want Blocking() true when artifacts exist for an unticked stage")
	}
	// prd is ticked and its file exists — must not also report a problem.
	prd := rowByName(t, rep.Stages, "prd")
	if prd.Verdict != stages.VerdictMatch {
		t.Errorf("prd verdict = %s (%s), want match", prd.Verdict, prd.Detail)
	}
}

// Story 8: no progress.md at all is a dormant/non-SDD repo — warn only,
// never blocking.
func TestNoLedgerWarnsNotBlocking(t *testing.T) {
	rep, err := stages.Derive(fixture("no-ledger-warn"))
	if err != nil {
		t.Fatal(err)
	}
	if rep.HasCursor {
		t.Fatal("want HasCursor false — no progress.md present")
	}
	if len(rep.Notes) == 0 {
		t.Fatal("want a Notes entry explaining the missing ledger")
	}
	if !strings.Contains(rep.Notes[0], "progress.md") {
		t.Errorf("Notes[0] = %q, want it to mention progress.md", rep.Notes[0])
	}
	if rep.Blocking() {
		t.Error("no-ledger case must never be blocking")
	}
}

// The newest-handoff backstop (I014): when a cursor exists, the newest
// docs/handoffs/* entry must carry a spine:cursor block. Here it exists but
// is stale prose — must block even though every stage row matches.
func TestHandoffMissingBlockBlocks(t *testing.T) {
	rep, err := stages.Derive(fixture("handoff-missing-block"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"prd", "issues"} {
		row := rowByName(t, rep.Stages, name)
		if row.Verdict != stages.VerdictMatch {
			t.Errorf("%s verdict = %s (%s), want match (this fixture isolates the handoff problem)", name, row.Verdict, row.Detail)
		}
	}
	if !rep.Handoff.Applicable {
		t.Fatal("want Handoff.Applicable true — a cursor exists")
	}
	if rep.Handoff.HasBlock {
		t.Error("want HasBlock false — the newest handoff has no spine:cursor block")
	}
	if !rep.Handoff.Blocking() {
		t.Error("want Handoff.Blocking() true")
	}
	if !rep.Blocking() {
		t.Error("want Blocking() true overall")
	}
}

// I025: the newest-handoff backstop is not satisfied by mere presence of a
// spine:cursor block — the block's effort: must match the live cursor's
// effort. Here the newest handoff carries a well-formed block, but for a
// different (previous) effort; this must block exactly like an absent
// block, and the detail must name both efforts.
func TestHandoffStaleEffortBlocks(t *testing.T) {
	rep, err := stages.Derive(fixture("handoff-stale-effort"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"prd", "issues"} {
		row := rowByName(t, rep.Stages, name)
		if row.Verdict != stages.VerdictMatch {
			t.Errorf("%s verdict = %s (%s), want match (this fixture isolates the handoff problem)", name, row.Verdict, row.Detail)
		}
	}
	if !rep.Handoff.Applicable {
		t.Fatal("want Handoff.Applicable true — a cursor exists")
	}
	if rep.Handoff.HasBlock {
		t.Error("want HasBlock false — a stale-effort block must be treated the same as an absent one")
	}
	if !rep.Handoff.Blocking() {
		t.Error("want Handoff.Blocking() true")
	}
	if !strings.Contains(rep.Handoff.Detail, "fixture-effort") || !strings.Contains(rep.Handoff.Detail, "previous-effort") {
		t.Errorf("want Detail naming both the live effort and the stale effort, got %q", rep.Handoff.Detail)
	}
	if !rep.Blocking() {
		t.Error("want Blocking() true overall")
	}
}

// Bonus (beyond the required fixture matrix): zero docs/handoffs entries at
// all, with a cursor present, must also block — there is nothing to embed
// the cursor in yet. Built inline rather than as a new testdata fixture
// (matches Task 1's convention of an inline bonus case for a distinct code
// path already covered by the fixture matrix in spirit).
func TestNoHandoffAtAllBlocksWhenCursorExists(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "WORKFLOW.md", "profile: library-cli\ntemplate_version: 8\nstages: [grill, prd, issues, implement]\n")
	writeFile(t, dir, ".superpowers/sdd/progress.md", "<!-- spine:cursor -->\n"+
		"effort: x\nprd: docs/specs/x.md\ntickets: I001\nstages: grill[<] prd[ ] issues[ ] implement[ ]\n"+
		"<!-- /spine:cursor -->\n")
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Handoff.Applicable || rep.Handoff.HasBlock {
		t.Errorf("Handoff = %#v, want applicable with no block found", rep.Handoff)
	}
	if !rep.Blocking() {
		t.Error("want Blocking() true — a cursor exists but there is no handoff at all")
	}
}

// The "here" (current) stage is exempt from both directions of the
// bidirectional check: partial evidence while actively working a stage is
// expected, not a contradiction.
func TestHereStageNeverBlocks(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "WORKFLOW.md", "profile: library-cli\ntemplate_version: 8\nstages: [grill, prd, issues, implement]\n")
	// prd is HERE despite the PRD file being present on disk already (would
	// block as present-but-unticked if prd were Pending instead).
	writeFile(t, dir, "docs/specs/x.md", "# x\n")
	writeFile(t, dir, ".superpowers/sdd/progress.md", "<!-- spine:cursor -->\n"+
		"effort: x\nprd: docs/specs/x.md\ntickets: I001\nstages: grill[x] prd[<] issues[ ] implement[ ]\n"+
		"<!-- /spine:cursor -->\n")
	writeFile(t, dir, "docs/handoffs/2026-01-02-x.md", "<!-- spine:cursor -->\neffort: x\n<!-- /spine:cursor -->\n")
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	prd := rowByName(t, rep.Stages, "prd")
	if prd.Verdict != stages.VerdictNotJudged {
		t.Errorf("prd (here) verdict = %s (%s), want not-judged", prd.Verdict, prd.Detail)
	}
	if rep.Blocking() {
		t.Errorf("here-stage must never block: stages=%#v", rep.Stages)
	}
}

// Stages with no derivation rule (grill, functional-test, review, verify,
// ship, deploy, docs, handoff) never carry evidence and can never block,
// whatever their marker.
func TestStagesWithoutARuleAreNeverJudged(t *testing.T) {
	rep, err := stages.Derive(fixture("clean"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"grill", "review", "verify", "ship", "deploy", "docs", "handoff"} {
		row := rowByName(t, rep.Stages, name)
		if row.Verdict != stages.VerdictNotJudged {
			t.Errorf("%s verdict = %s, want not-judged (no rule)", name, row.Verdict)
		}
	}
}

// The tickets: field's "prefix I0" grammar form resolves against every
// docs/issues ticket id sharing that prefix, not just a numeric range.
func TestPrefixTicketGrammarResolves(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "WORKFLOW.md", "profile: library-cli\ntemplate_version: 8\nstages: [grill, prd, issues, implement]\n")
	writeFile(t, dir, "docs/issues/I001-a.md", "---\nid: I001\n---\nx\n")
	writeFile(t, dir, "docs/issues/I002-b.md", "---\nid: I002\n---\nx\n")
	writeFile(t, dir, ".superpowers/sdd/progress.md", "<!-- spine:cursor -->\n"+
		"effort: x\nprd: docs/specs/x.md\ntickets: prefix I0\nstages: grill[x] prd[ ] issues[x] implement[<]\n"+
		"<!-- /spine:cursor -->\n")
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	issues := rowByName(t, rep.Stages, "issues")
	if issues.Verdict != stages.VerdictMatch {
		t.Errorf("issues verdict = %s (%s), want match — both I001 and I002 exist", issues.Verdict, issues.Detail)
	}
}

// An unresolvable tickets: value (neither range nor prefix grammar) must
// degrade to no evidence, never a block — absence of evidence never blocks,
// even when the absence is "we couldn't even parse the ticket set."
func TestUnresolvableTicketsNeverBlocks(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "WORKFLOW.md", "profile: library-cli\ntemplate_version: 8\nstages: [grill, prd, issues, implement]\n")
	writeFile(t, dir, ".superpowers/sdd/progress.md", "<!-- spine:cursor -->\n"+
		"effort: x\nprd: docs/specs/x.md\ntickets: not-a-grammar\nstages: grill[x] prd[ ] issues[x] implement[<]\n"+
		"<!-- /spine:cursor -->\n")
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	issues := rowByName(t, rep.Stages, "issues")
	if issues.Verdict != stages.VerdictNotJudged {
		t.Errorf("issues verdict = %s (%s), want not-judged", issues.Verdict, issues.Detail)
	}
}

// FromResult must accept an already-loaded cursor.Result (cmd/spine's
// cursor command has one already; it must not need to re-read the repo).
func TestFromResultMatchesDerive(t *testing.T) {
	res, err := cursor.Load(fixture("clean"))
	if err != nil {
		t.Fatal(err)
	}
	rep := stages.FromResult(fixture("clean"), res)
	if !rep.HasCursor || rep.Blocking() {
		t.Errorf("FromResult report = %#v, want a clean non-blocking report", rep)
	}
}

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
