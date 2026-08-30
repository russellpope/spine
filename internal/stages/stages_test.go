package stages_test

import (
	"fmt"
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

// I109, the operator-visible regression: a handoff that explains the cursor
// convention — quoting the fence in a sentence, and showing a whole block as
// an indented example — used to hijack its own parse. The scan matched the
// tag anywhere, so derivation opened the block at the prose quote, ran to the
// real closing fence, and reported a wall of malformed body lines. That took
// `spine audit stages` red (and doctor's D9 with it) against a byte-perfect
// committed snapshot.
//
// Fences are line-anchored at column 0 now, so prose and indented examples are
// skipped and this derives clean.
func TestHandoffWithProseFenceMentionDerivesClean(t *testing.T) {
	rep, err := stages.Derive(fixture("handoff-prose-fence"))
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Handoff.Applicable {
		t.Fatal("want Handoff.Applicable true — a cursor exists")
	}
	if !rep.Handoff.HasBlock {
		t.Errorf("want HasBlock true — the real fenced region is pristine: %s", rep.Handoff.Detail)
	}
	if rep.Handoff.Blocking() {
		t.Errorf("want Handoff.Blocking() false, got detail %q", rep.Handoff.Detail)
	}
	if rep.Blocking() {
		t.Errorf("a handoff that documents the convention must not block: handoff=%#v", rep.Handoff)
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

// M4 (I027): an I/O error reading docs/handoffs (as opposed to the
// directory legitimately having zero entries) must produce a distinct
// Detail — "no handoffs exist" and "handoffs unreadable" are different
// causes and must not share wording. Forced by making dir/docs a regular
// file, so os.ReadDir(dir/docs/handoffs) fails with ENOTDIR rather than
// ErrNotExist.
func TestHandoffReadErrorDetailDiffersFromAbsent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "WORKFLOW.md", "profile: library-cli\ntemplate_version: 8\nstages: [grill, prd, issues, implement]\n")
	writeFile(t, dir, ".superpowers/sdd/progress.md", "<!-- spine:cursor -->\n"+
		"effort: x\nprd: docs/specs/x.md\ntickets: I001\nstages: grill[<] prd[ ] issues[ ] implement[ ]\n"+
		"<!-- /spine:cursor -->\n")
	// dir/docs is a file, not a directory — os.ReadDir(dir/docs/handoffs)
	// fails with "not a directory", a genuine I/O error distinct from a
	// missing docs/handoffs dir (which handoff.List treats as zero entries,
	// not an error).
	if err := os.WriteFile(filepath.Join(dir, "docs"), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Handoff.Applicable || rep.Handoff.HasBlock {
		t.Errorf("Handoff = %#v, want applicable with no block found", rep.Handoff)
	}
	if !rep.Blocking() {
		t.Error("want Blocking() true — the handoff check cannot be satisfied when docs/handoffs is unreadable")
	}
	if strings.Contains(rep.Handoff.Detail, "no docs/handoffs entries found") {
		t.Errorf("Detail = %q, want it to distinguish a read error from zero entries, not reuse the absent-handoffs wording", rep.Handoff.Detail)
	}
	if !strings.Contains(strings.ToLower(rep.Handoff.Detail), "unreadable") && !strings.Contains(strings.ToLower(rep.Handoff.Detail), "error") {
		t.Errorf("Detail = %q, want it to mention the read error", rep.Handoff.Detail)
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
	writeFile(t, dir, "docs/handoffs/2026-01-02-x.md", "<!-- spine:cursor -->\neffort: x\nprd: docs/specs/x.md\ntickets: I001\nstages: grill[x] prd[<] issues[ ] implement[ ]\n<!-- /spine:cursor -->\n")
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

func TestAcceptanceSummaryUsesResolvedCursorTicketsOnly(t *testing.T) {
	dir := acceptanceStageRepo(t, "I001,I002")
	writeFile(t, dir, "docs/handoffs/2026-08-29-approval.md", "# Approval\n")
	writeAcceptanceStageTicket(t, dir, "I001-one.md", "I001", stageAcceptanceLine("reason one"))
	writeAcceptanceStageTicket(t, dir, "I002-two.md", "I002", stageAcceptanceLine(""))
	writeAcceptanceStageTicket(t, dir, "I003-unscoped.md", "I003", stageAcceptanceLine("reason three"))
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Acceptance.ValidCount() != 1 || rep.Acceptance.InvalidCount() != 1 || rep.Acceptance.CandidateCount() != 2 {
		t.Fatalf("scoped acceptance summary = %#v", rep.Acceptance)
	}
}

func TestAcceptanceSummarySkipsUnresolvableTickets(t *testing.T) {
	dir := acceptanceStageRepo(t, "not-a-ticket-expression")
	writeFile(t, dir, "docs/handoffs/2026-08-29-approval.md", "# Approval\n")
	writeAcceptanceStageTicket(t, dir, "I001-one.md", "I001", stageAcceptanceLine("reason one"))
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Notes) == 0 || rep.Acceptance.CandidateCount() != 0 {
		t.Fatalf("unresolvable tickets must warn and skip acceptance: notes=%#v summary=%#v", rep.Notes, rep.Acceptance)
	}
}

func TestAcceptanceProblemsNeverAffectBlocking(t *testing.T) {
	dir := acceptanceStageRepo(t, "I001")
	writeFile(t, dir, "docs/handoffs/2026-08-29-approval.md", "# Approval\n")
	writeAcceptanceStageTicket(t, dir, "I001-one.md", "I001", stageAcceptanceLine(""))
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Acceptance.InvalidCount() != 1 || rep.Blocking() {
		t.Fatalf("invalid acceptance must remain advisory: summary=%#v report=%#v", rep.Acceptance, rep)
	}
}

func TestAcceptanceReadErrorsAreAdvisoryAndSurfaced(t *testing.T) {
	dir := acceptanceStageRepo(t, "I001")
	if err := os.MkdirAll(filepath.Join(dir, "docs", "issues"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "missing-ticket.md"), filepath.Join(dir, "docs", "issues", "I001-broken.md")); err != nil {
		t.Fatal(err)
	}

	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Acceptance.ScanErrors) != 1 || rep.Acceptance.CandidateCount() != 0 || rep.Blocking() {
		t.Fatalf("acceptance read error contract = %#v", rep)
	}
}

func acceptanceStageRepo(t *testing.T, tickets string) string {
	t.Helper()
	dir := t.TempDir()
	block := "<!-- spine:cursor -->\n" +
		"effort: acceptance\n" +
		"prd: docs/specs/acceptance.md\n" +
		"tickets: " + tickets + "\n" +
		"stages: grill[x] issues[<]\n" +
		"<!-- /spine:cursor -->\n"
	writeFile(t, dir, "WORKFLOW.md", "stages: [grill, issues]\n")
	writeFile(t, dir, ".superpowers/sdd/progress.md", block)
	writeFile(t, dir, "docs/handoffs/2026-08-30-acceptance.md", block)
	return dir
}

func writeAcceptanceStageTicket(t *testing.T, dir, name, id, line string) {
	t.Helper()
	writeFile(t, dir, filepath.Join("docs", "issues", name), "---\nid: "+id+"\n---\n\n## Acceptance criteria\n"+line+"\n")
}

func stageAcceptanceLine(reason string) string {
	return "- [ ] Exercise failover -- APPROVED-UNTESTED 2026-08-29 by owner ref: docs/handoffs/2026-08-29-approval.md#failover reason: " + reason
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

// I114: a comma-list anchors exactly its listed ticket ids, in input order.
// With no ticket files present, the missing-id detail is the public evidence
// seam for both the resolved count and ordering.
func TestCommaListTicketGrammarResolvesInOrder(t *testing.T) {
	tests := []struct {
		name       string
		tickets    string
		wantDetail string
	}{
		{
			name:       "two tickets",
			tickets:    "I065,I106",
			wantDetail: "2/2 ticket file(s) missing: I065, I106",
		},
		{
			name:       "descending tickets",
			tickets:    "I106,I065",
			wantDetail: "2/2 ticket file(s) missing: I106, I065",
		},
		{
			name:       "three tickets",
			tickets:    "I065,I106,I114",
			wantDetail: "3/3 ticket file(s) missing: I065, I106, I114",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, "WORKFLOW.md", "profile: library-cli\ntemplate_version: 8\nstages: [grill, prd, issues, implement]\n")
			writeFile(t, dir, ".superpowers/sdd/progress.md", "<!-- spine:cursor -->\n"+
				"effort: x\nprd: docs/specs/x.md\ntickets: "+tt.tickets+"\nstages: grill[x] prd[ ] issues[x] implement[<]\n"+
				"<!-- /spine:cursor -->\n")

			rep, err := stages.Derive(dir)
			if err != nil {
				t.Fatal(err)
			}
			issues := rowByName(t, rep.Stages, "issues")
			if issues.Verdict != stages.VerdictTickedMissing {
				t.Fatalf("issues verdict = %s (%s), want ticked-missing", issues.Verdict, issues.Detail)
			}
			if !strings.Contains(issues.Detail, tt.wantDetail) {
				t.Errorf("issues detail = %q, want %q", issues.Detail, tt.wantDetail)
			}
			if len(rep.Notes) != 0 {
				t.Errorf("resolvable comma-list must not produce a Notes entry, got %#v", rep.Notes)
			}
		})
	}
}

// I114: malformed comma-lists are unresolvable as a whole. The valid ticket
// files make an accidental partial resolution observable as a judged stage.
func TestCommaListTicketGrammarRejectsWholeInvalidValue(t *testing.T) {
	tests := []struct {
		name    string
		tickets string
	}{
		{name: "internal whitespace", tickets: "I065, I106"},
		{name: "duplicate", tickets: "I065,I065"},
		{name: "malformed element", tickets: "I065,nope"},
		{name: "empty middle element", tickets: "I065,,I106"},
		{name: "trailing empty element", tickets: "I065,"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, "WORKFLOW.md", "profile: library-cli\ntemplate_version: 8\nstages: [grill, prd, issues, implement]\n")
			writeFile(t, dir, "docs/issues/I065-a.md", "---\nid: I065\n---\nx\n")
			writeFile(t, dir, "docs/issues/I106-b.md", "---\nid: I106\n---\nx\n")
			writeFile(t, dir, ".superpowers/sdd/progress.md", "<!-- spine:cursor -->\n"+
				"effort: x\nprd: docs/specs/x.md\ntickets: "+tt.tickets+"\nstages: grill[x] prd[ ] issues[x] implement[<]\n"+
				"<!-- /spine:cursor -->\n")

			rep, err := stages.Derive(dir)
			if err != nil {
				t.Fatal(err)
			}
			issues := rowByName(t, rep.Stages, "issues")
			if issues.Verdict != stages.VerdictNotJudged {
				t.Errorf("issues verdict = %s (%s), want not-judged for unresolvable list", issues.Verdict, issues.Detail)
			}
			if len(rep.Notes) != 1 {
				t.Fatalf("want exactly one Notes entry for unresolvable comma-list, got %#v", rep.Notes)
			}
			if !strings.Contains(rep.Notes[0], tt.tickets) {
				t.Errorf("Notes[0] = %q, want it to name %q", rep.Notes[0], tt.tickets)
			}
			if !strings.Contains(rep.Notes[0], "I0NN | I0NN,I0MM[,...] | I0NN-I0MM | prefix <str>") {
				t.Errorf("Notes[0] = %q, want the comma-list grammar summary", rep.Notes[0])
			}
		})
	}
}

// An unresolvable tickets: value (neither bare id, range, nor prefix
// grammar) must degrade to no evidence, never a block — absence of evidence
// never blocks, even when the absence is "we couldn't even parse the ticket
// set." I026: the degradation must not be silent — a Notes entry names the
// exact bad value, distinguishing this from a legitimately empty "prefix"
// match (which resolves fine and produces no note).
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
	// Note: this fixture's overall rep.Blocking() is true regardless of the
	// tickets: value, because it has no docs/handoffs at all (the I014
	// newest-handoff backstop) — an orthogonal blocking condition already
	// covered by TestNoHandoffAtAllBlocksWhenCursorExists. Report.Notes is
	// never consulted by Blocking() (see the package doc), so an
	// unresolvable-tickets Note is non-blocking by construction; this test
	// only asserts the Note itself, not the report's overall Blocking().
	if len(rep.Notes) != 1 {
		t.Fatalf("want exactly one Notes entry for the unresolvable tickets: value, got %#v", rep.Notes)
	}
	if !strings.Contains(rep.Notes[0], "not-a-grammar") {
		t.Errorf("Notes[0] = %q, want it to name the bad tickets: value", rep.Notes[0])
	}
}

// A resolvable-but-empty tickets: value (a "prefix" that matches nothing on
// disk) must NOT produce an unresolvable-tickets Note — it resolved fine,
// vacuously, to an empty set. Only genuinely unparseable values get a note.
func TestResolvableEmptyPrefixProducesNoNote(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "WORKFLOW.md", "profile: library-cli\ntemplate_version: 8\nstages: [grill, prd, issues, implement]\n")
	writeFile(t, dir, ".superpowers/sdd/progress.md", "<!-- spine:cursor -->\n"+
		"effort: x\nprd: docs/specs/x.md\ntickets: prefix I9\nstages: grill[x] prd[ ] issues[x] implement[<]\n"+
		"<!-- /spine:cursor -->\n")
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Notes) != 0 {
		t.Errorf("resolvable (if empty) prefix must not produce a Notes entry, got %#v", rep.Notes)
	}
}

// I026: a bare single-ticket id ("tickets: I001", previously undocumented
// and unresolvable) now resolves to that one ticket, anchoring evidence for
// both the issues and implement stages exactly like a one-element range
// would.
func TestBareTicketIDResolves(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "WORKFLOW.md", "profile: library-cli\ntemplate_version: 8\nstages: [grill, prd, issues, implement]\n")
	writeFile(t, dir, "docs/issues/I001-a.md", "---\nid: I001\n---\nx\n")
	writeFile(t, dir, ".superpowers/sdd/progress.md", "<!-- spine:cursor -->\n"+
		"effort: x\nprd: docs/specs/x.md\ntickets: I001\nstages: grill[x] prd[ ] issues[x] implement[<]\n"+
		"<!-- /spine:cursor -->\n")
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	issues := rowByName(t, rep.Stages, "issues")
	if issues.Verdict != stages.VerdictMatch {
		t.Errorf("issues verdict = %s (%s), want match — I001 exists on disk", issues.Verdict, issues.Detail)
	}
	if len(rep.Notes) != 0 {
		t.Errorf("a resolvable bare id must not produce an unresolvable-tickets note, got %#v", rep.Notes)
	}
}

// I026: a same-endpoint range ("I001-I001") already resolved structurally
// before this ticket (the range regex never required start < end) — this
// locks that behavior now that it's a documented, not just accidental, form.
func TestSameEndpointRangeResolves(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "WORKFLOW.md", "profile: library-cli\ntemplate_version: 8\nstages: [grill, prd, issues, implement]\n")
	writeFile(t, dir, "docs/issues/I001-a.md", "---\nid: I001\n---\nx\n")
	writeFile(t, dir, ".superpowers/sdd/progress.md", "<!-- spine:cursor -->\n"+
		"effort: x\nprd: docs/specs/x.md\ntickets: I001-I001\nstages: grill[x] prd[ ] issues[x] implement[<]\n"+
		"<!-- /spine:cursor -->\n")
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	issues := rowByName(t, rep.Stages, "issues")
	if issues.Verdict != stages.VerdictMatch {
		t.Errorf("issues verdict = %s (%s), want match — I001 exists on disk", issues.Verdict, issues.Detail)
	}
}

// I029: a partial ticked-missing set (some but not all resolved ids exist)
// must name the missing ticket ids in the detail, not just the raw
// missing/total count — the count alone gives no starting point for
// investigation. Existing ids must not be named as missing.
func TestTickedMissingNamesPartialMissingIDs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "WORKFLOW.md", "profile: library-cli\ntemplate_version: 8\nstages: [grill, prd, issues, implement]\n")
	writeFile(t, dir, "docs/issues/I001-a.md", "---\nid: I001\n---\nx\n")
	writeFile(t, dir, "docs/issues/I003-c.md", "---\nid: I003\n---\nx\n")
	writeFile(t, dir, ".superpowers/sdd/progress.md", "<!-- spine:cursor -->\n"+
		"effort: x\nprd: docs/specs/x.md\ntickets: I001-I004\nstages: grill[x] prd[x] issues[x] implement[<]\n"+
		"<!-- /spine:cursor -->\n")
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	issues := rowByName(t, rep.Stages, "issues")
	// Existing behavior unchanged: still a blocking ticked-missing verdict.
	if issues.Verdict != stages.VerdictTickedMissing {
		t.Fatalf("issues verdict = %s (%s), want ticked-missing", issues.Verdict, issues.Detail)
	}
	if !rep.Blocking() {
		t.Error("want Blocking() true — ticked-missing must still block")
	}
	if !strings.Contains(issues.Detail, "I002") || !strings.Contains(issues.Detail, "I004") {
		t.Errorf("Detail = %q, want it to name the missing ids I002 and I004", issues.Detail)
	}
	if strings.Contains(issues.Detail, "I001") || strings.Contains(issues.Detail, "I003") {
		t.Errorf("Detail = %q, must not name the present ids I001/I003 as missing", issues.Detail)
	}
}

// I029: a long missing set must truncate the named ids with a "+N more"
// count rather than dumping every missing id onto one line.
func TestTickedMissingTruncatesLongMissingSet(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "WORKFLOW.md", "profile: library-cli\ntemplate_version: 8\nstages: [grill, prd, issues, implement]\n")
	// Only I001 exists; derive the range from the production naming cap so the
	// missing set is always exactly one larger than that cap.
	rangeEnd := stages.MaxNamedMissingIDsForTest + 2
	writeFile(t, dir, "docs/issues/I001-a.md", "---\nid: I001\n---\nx\n")
	writeFile(t, dir, ".superpowers/sdd/progress.md", "<!-- spine:cursor -->\n"+
		fmt.Sprintf("effort: x\nprd: docs/specs/x.md\ntickets: I001-I%03d\nstages: grill[x] prd[x] issues[x] implement[<]\n", rangeEnd)+
		"<!-- /spine:cursor -->\n")
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	issues := rowByName(t, rep.Stages, "issues")
	if issues.Verdict != stages.VerdictTickedMissing {
		t.Fatalf("issues verdict = %s (%s), want ticked-missing", issues.Verdict, issues.Detail)
	}
	if !rep.Blocking() {
		t.Error("want Blocking() true — ticked-missing must still block")
	}
	if !strings.Contains(issues.Detail, "more") {
		t.Errorf("Detail = %q, want a truncated \"+N more\" tail for a long missing set", issues.Detail)
	}
	lastID := fmt.Sprintf("I%03d", rangeEnd)
	if strings.Contains(issues.Detail, lastID) {
		t.Errorf("Detail = %q, want the tail id %s folded into the +N more count, not named", issues.Detail, lastID)
	}
}

// I029: when ALL resolved ids in the set are missing (0 present out of N),
// the detail must also mention the live tickets: value — that all-missing
// shape is exactly what a resolvable-but-wrong tickets: typo (e.g.
// "I01-I04" for "I001-I004") produces, so the reader should be pointed at
// the likely cause.
func TestTickedMissingAllMissingMentionsTicketsValue(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "WORKFLOW.md", "profile: library-cli\ntemplate_version: 8\nstages: [grill, prd, issues, implement]\n")
	// A real ticket exists, but under the correct I0NN (3-digit) width — the
	// typo'd 2-digit range I01-I04 resolves to a disjoint, entirely-missing
	// set.
	writeFile(t, dir, "docs/issues/I001-a.md", "---\nid: I001\n---\nx\n")
	writeFile(t, dir, ".superpowers/sdd/progress.md", "<!-- spine:cursor -->\n"+
		"effort: x\nprd: docs/specs/x.md\ntickets: I01-I04\nstages: grill[x] prd[x] issues[x] implement[<]\n"+
		"<!-- /spine:cursor -->\n")
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	issues := rowByName(t, rep.Stages, "issues")
	if issues.Verdict != stages.VerdictTickedMissing {
		t.Fatalf("issues verdict = %s (%s), want ticked-missing", issues.Verdict, issues.Detail)
	}
	if !rep.Blocking() {
		t.Error("want Blocking() true — ticked-missing must still block")
	}
	if !strings.Contains(issues.Detail, "tickets:") || !strings.Contains(issues.Detail, "I01-I04") {
		t.Errorf("Detail = %q, want it to mention the live tickets: value I01-I04 as the likely typo", issues.Detail)
	}
}

// I117: when implement is ticked with zero evidence but ledger lines for
// the anchored ids DO exist, the miss is the done-word requirement, not a
// tickets: typo — the detail must name the rule and suppress the typo hint.
func TestImplementAllMissingWithAnchoredLinesNamesDoneWordRule(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "WORKFLOW.md", "profile: library-cli\ntemplate_version: 8\nstages: [grill, prd, issues, implement, functional-test]\n")
	writeFile(t, dir, "docs/issues/I001-a.md", "---\nid: I001\n---\nx\n")
	writeFile(t, dir, ".superpowers/sdd/progress.md", "<!-- spine:cursor -->\n"+
		"effort: x\nprd: docs/specs/x.md\ntickets: I001\nstages: grill[x] prd[x] issues[x] implement[x] functional-test[<]\n"+
		"<!-- /spine:cursor -->\n\n- I001: shipped and declared\n")
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	impl := rowByName(t, rep.Stages, "implement")
	if impl.Verdict != stages.VerdictTickedMissing {
		t.Fatalf("implement verdict = %s (%s), want ticked-missing", impl.Verdict, impl.Detail)
	}
	if !strings.Contains(impl.Detail, "done/complete/completed") || !strings.Contains(impl.Detail, "as a whole word") {
		t.Errorf("Detail = %q, want the done-word whole-word requirement named", impl.Detail)
	}
	if strings.Contains(impl.Detail, "typo") {
		t.Errorf("Detail = %q, want the typo hint suppressed — the ids demonstrably resolved", impl.Detail)
	}
}

// I032: the all-missing tickets typo hint belongs to the issues row, not the
// implement row, whose missing evidence is in the ledger rather than ticket
// files.
func TestImplementAllMissingDoesNotMentionTicketsTypo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "WORKFLOW.md", "profile: library-cli\ntemplate_version: 8\nstages: [grill, prd, issues, implement, functional-test]\n")
	writeFile(t, dir, "docs/issues/I001-a.md", "---\nid: I001\n---\nx\n")
	writeFile(t, dir, ".superpowers/sdd/progress.md", "<!-- spine:cursor -->\n"+
		"effort: x\nprd: docs/specs/x.md\ntickets: I001\nstages: grill[x] prd[x] issues[x] implement[x] functional-test[<]\n"+
		"<!-- /spine:cursor -->\n")
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	impl := rowByName(t, rep.Stages, "implement")
	if impl.Verdict != stages.VerdictTickedMissing {
		t.Fatalf("implement verdict = %s (%s), want ticked-missing", impl.Verdict, impl.Detail)
	}
	if strings.Contains(impl.Detail, "tickets:") || strings.Contains(impl.Detail, "typo") {
		t.Errorf("implement detail = %q, want the tickets typo hint scoped to issues", impl.Detail)
	}
}

// I032 supersedes I117's negative control: with no ledger line for the id at
// all, the implement row still must not receive the issues-row typo hint.
func TestImplementAllMissingNoAnchoredLinesSuppressesTypoHint(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "WORKFLOW.md", "profile: library-cli\ntemplate_version: 8\nstages: [grill, prd, issues, implement, functional-test]\n")
	writeFile(t, dir, "docs/issues/I001-a.md", "---\nid: I001\n---\nx\n")
	writeFile(t, dir, ".superpowers/sdd/progress.md", "<!-- spine:cursor -->\n"+
		"effort: x\nprd: docs/specs/x.md\ntickets: I001\nstages: grill[x] prd[x] issues[x] implement[x] functional-test[<]\n"+
		"<!-- /spine:cursor -->\n\n- unrelated ledger prose\n")
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	impl := rowByName(t, rep.Stages, "implement")
	if impl.Verdict != stages.VerdictTickedMissing {
		t.Fatalf("implement verdict = %s (%s), want ticked-missing", impl.Verdict, impl.Detail)
	}
	if strings.Contains(impl.Detail, "typo") {
		t.Errorf("Detail = %q, want the typo hint scoped to the issues row", impl.Detail)
	}
	if strings.Contains(impl.Detail, "tickets:") {
		t.Errorf("Detail = %q, want no tickets hint on the implement row", impl.Detail)
	}
}

// I117 mixed case: one id anchored without a done-word, one absent
// entirely. Any anchored line proves the tickets value is not a typo, so
// the wording message wins — while the missing-ids list still names both.
func TestImplementMixedAnchoredAndAbsentGetsWordingMessage(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "WORKFLOW.md", "profile: library-cli\ntemplate_version: 8\nstages: [grill, prd, issues, implement, functional-test]\n")
	writeFile(t, dir, "docs/issues/I001-a.md", "---\nid: I001\n---\nx\n")
	writeFile(t, dir, "docs/issues/I002-b.md", "---\nid: I002\n---\nx\n")
	writeFile(t, dir, ".superpowers/sdd/progress.md", "<!-- spine:cursor -->\n"+
		"effort: x\nprd: docs/specs/x.md\ntickets: I001,I002\nstages: grill[x] prd[x] issues[x] implement[x] functional-test[<]\n"+
		"<!-- /spine:cursor -->\n\n- I001: work declared\n")
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	impl := rowByName(t, rep.Stages, "implement")
	if impl.Verdict != stages.VerdictTickedMissing {
		t.Fatalf("implement verdict = %s (%s), want ticked-missing", impl.Verdict, impl.Detail)
	}
	if !strings.Contains(impl.Detail, "as a whole word") || strings.Contains(impl.Detail, "typo") {
		t.Errorf("Detail = %q, want the wording message to win the mixed case", impl.Detail)
	}
	if !strings.Contains(impl.Detail, "I001") || !strings.Contains(impl.Detail, "I002") {
		t.Errorf("Detail = %q, want the missing-ids list still naming both ids", impl.Detail)
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
