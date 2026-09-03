package stages_test

import (
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/stages"
)

// I125: a ticket file closed by the ledger lifecycle (status: fixed plus a
// SHA-shaped commits: list) is a closure record and evidences implement for
// its id, OR'd with the progress-ledger done-word scan. These tests build
// the exact maikanban shape the ticket observed: implement ticked, zero
// progress-ledger lines for the ids, ticket files closed on disk.

const closureWorkflow = "profile: library-cli\ntemplate_version: 8\nstages: [grill, prd, issues, implement, functional-test]\n"

// closureRepo writes a repo whose cursor anchors the given tickets value
// with implement in implState ("x" or " ") and functional-test as the
// current stage, plus the given ledger tail after the cursor block.
func closureRepo(t *testing.T, tickets, implState, ledgerTail string) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "WORKFLOW.md", closureWorkflow)
	writeFile(t, dir, "docs/specs/x.md", "x\n")
	writeFile(t, dir, ".superpowers/sdd/progress.md", "<!-- spine:cursor -->\n"+
		"effort: x\nprd: docs/specs/x.md\ntickets: "+tickets+"\nstages: grill[x] prd[x] issues[x] implement["+implState+"] functional-test[<]\n"+
		"<!-- /spine:cursor -->\n"+ledgerTail)
	return dir
}

func ticketFile(id, status, commits string) string {
	s := "---\nid: " + id + "\ntitle: \"t\"\nseverity: low\nstatus: " + status + "\n"
	if commits != "" {
		s += "commits: " + commits + "\n"
	}
	return s + "affects: []\nblocked-by: []\n---\n\n## Problem\n\nx\n"
}

func implementRow(t *testing.T, dir string) stages.StageRow {
	t.Helper()
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	return rowByName(t, rep.Stages, "implement")
}

// Acceptance criterion 1: implement [x], no progress-md lines, ticket
// closed with commits — match, not ticked-missing. The issues row is
// asserted in the same derivation: the ticket's first aggravation was the
// two rows contradicting each other on one closed ticket.
func TestImplementClosureRecordEvidencesTickedStage(t *testing.T) {
	dir := closureRepo(t, "I001", "x", "\n- unrelated ledger prose\n")
	writeFile(t, dir, "docs/issues/I001-a.md", ticketFile("I001", "fixed", "[a9ddea5]"))
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	impl := rowByName(t, rep.Stages, "implement")
	if impl.Verdict != stages.VerdictMatch {
		t.Fatalf("implement verdict = %s (%s), want match from the closure record", impl.Verdict, impl.Detail)
	}
	if impl.Detail != "1/1 implement evidence present" {
		t.Errorf("Detail = %q, want the renamed implement-evidence label", impl.Detail)
	}
	if issues := rowByName(t, rep.Stages, "issues"); issues.Verdict != stages.VerdictMatch {
		t.Errorf("issues verdict = %s (%s), want match alongside implement — no contradictory rows", issues.Verdict, issues.Detail)
	}
	// The handoff backstop blocks on this fixture (no docs/handoffs), so
	// judge the stage rows alone: none may contradict the cursor.
	for _, row := range rep.Stages {
		if row.Verdict == stages.VerdictTickedMissing || row.Verdict == stages.VerdictPresentUnticked {
			t.Errorf("row %s = %s (%s), want no contradicting row", row.Name, row.Verdict, row.Detail)
		}
	}
}

// A quoted status is not the lifecycle's spelling and is not a closure
// record: the frontmatter walk strips no quotes, so id: resolution on the
// issues row stays byte-identical to the pre-I125 behaviour.
func TestImplementQuotedStatusIsNotClosureRecord(t *testing.T) {
	dir := closureRepo(t, "I001", "x", "\n")
	writeFile(t, dir, "docs/issues/I001-a.md", ticketFile("I001", "\"fixed\"", "[a9ddea5]"))
	impl := implementRow(t, dir)
	if impl.Verdict != stages.VerdictTickedMissing {
		t.Fatalf("implement verdict = %s (%s), want ticked-missing", impl.Verdict, impl.Detail)
	}
}

// Acceptance criteria 2 and 3: the closure path is load-bearing. An open
// ticket, a fixed ticket with an empty or absent commits list, a placeholder
// token, and the wontfix/superseded closures all leave implement
// ticked-missing.
func TestImplementClosureNegativeControls(t *testing.T) {
	cases := []struct {
		name, status, commits string
	}{
		{"open with sha", "open", "[a9ddea5]"},
		{"in-progress with sha", "in-progress", "[a9ddea5]"},
		{"fixed empty commits", "fixed", "[]"},
		{"fixed absent commits", "fixed", ""},
		{"fixed placeholder commits", "fixed", "[pending]"},
		{"wontfix with sha", "wontfix", "[a9ddea5]"},
		{"superseded with sha", "superseded", "[a9ddea5]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := closureRepo(t, "I001", "x", "\n- unrelated ledger prose\n")
			writeFile(t, dir, "docs/issues/I001-a.md", ticketFile("I001", tc.status, tc.commits))
			impl := implementRow(t, dir)
			if impl.Verdict != stages.VerdictTickedMissing {
				t.Fatalf("implement verdict = %s (%s), want ticked-missing — %s must not evidence implement", impl.Verdict, impl.Detail, tc.name)
			}
		})
	}
}

// The two sources OR per id: one ticket evidenced only by a ledger line,
// the other only by its closure record, derive a full match together.
func TestImplementLedgerAndClosureRecordOr(t *testing.T) {
	dir := closureRepo(t, "I001,I002", "x", "\n- I001: implementation complete\n")
	writeFile(t, dir, "docs/issues/I001-a.md", ticketFile("I001", "open", ""))
	writeFile(t, dir, "docs/issues/I002-b.md", ticketFile("I002", "fixed", "[b81292d, 947a87a]"))
	impl := implementRow(t, dir)
	if impl.Verdict != stages.VerdictMatch || impl.Detail != "2/2 implement evidence present" {
		t.Fatalf("implement = %s (%s), want 2/2 match from ledger OR closure", impl.Verdict, impl.Detail)
	}
}

// An anchored ledger line without a done-word (the I117 shape) is outranked
// by a closure record for the same id: the ticket file is the lifecycle's
// own close artifact.
func TestImplementAnchoredNoDoneWordClosureRecordWins(t *testing.T) {
	dir := closureRepo(t, "I001", "x", "\n- I001: shipped and declared\n")
	writeFile(t, dir, "docs/issues/I001-a.md", ticketFile("I001", "fixed", "[a9ddea5]"))
	impl := implementRow(t, dir)
	if impl.Verdict != stages.VerdictMatch {
		t.Fatalf("implement verdict = %s (%s), want match — closure record outranks the wording miss", impl.Verdict, impl.Detail)
	}
}

// Symmetry: closure evidence feeds the pending direction exactly as a stray
// ledger line does — a pending implement over a closed ticket is a stale
// cursor and derives present-unticked.
func TestImplementPendingWithClosureRecordIsPresentUnticked(t *testing.T) {
	dir := closureRepo(t, "I001", " ", "\n")
	writeFile(t, dir, "docs/issues/I001-a.md", ticketFile("I001", "fixed", "[a9ddea5]"))
	impl := implementRow(t, dir)
	if impl.Verdict != stages.VerdictPresentUnticked {
		t.Fatalf("implement verdict = %s (%s), want present-unticked", impl.Verdict, impl.Detail)
	}
}

// Acceptance criterion 4: ticket files present for every anchored id, zero
// progress-md lines — the ticked-missing detail names both evidence sources
// and carries no typo hint and no tickets: mention.
func TestImplementZeroEvidenceNoAnchoredLinesNamesBothSources(t *testing.T) {
	dir := closureRepo(t, "I001,I002", "x", "\n- unrelated ledger prose\n")
	writeFile(t, dir, "docs/issues/I001-a.md", ticketFile("I001", "open", ""))
	writeFile(t, dir, "docs/issues/I002-b.md", ticketFile("I002", "in-progress", ""))
	impl := implementRow(t, dir)
	if impl.Verdict != stages.VerdictTickedMissing {
		t.Fatalf("implement verdict = %s (%s), want ticked-missing", impl.Verdict, impl.Detail)
	}
	for _, want := range []string{"I001, I002", "no progress-ledger implement line", "no closure record", "status: fixed"} {
		if !strings.Contains(impl.Detail, want) {
			t.Errorf("Detail = %q, want it to contain %q", impl.Detail, want)
		}
	}
	for _, banned := range []string{"typo", "tickets:", "as a whole word"} {
		if strings.Contains(impl.Detail, banned) {
			t.Errorf("Detail = %q, must not contain %q", impl.Detail, banned)
		}
	}
}

// The I117 wording message keeps precedence: when an anchored line exists
// without a done-word, the detail names the done-word rule, not the
// both-sources rule.
func TestImplementAnchoredNoDoneWordKeepsWordingMessage(t *testing.T) {
	dir := closureRepo(t, "I001", "x", "\n- I001: shipped and declared\n")
	writeFile(t, dir, "docs/issues/I001-a.md", ticketFile("I001", "open", ""))
	impl := implementRow(t, dir)
	if impl.Verdict != stages.VerdictTickedMissing {
		t.Fatalf("implement verdict = %s (%s), want ticked-missing", impl.Verdict, impl.Detail)
	}
	if !strings.Contains(impl.Detail, "as a whole word") || strings.Contains(impl.Detail, "no closure record") {
		t.Errorf("Detail = %q, want the done-word wording message alone", impl.Detail)
	}
}

// The frontmatter walk is first-occurrence-wins, exactly like the pre-I125
// id: lookup: an indented duplicate id: deeper in the fence must neither
// re-point the file at another ticket (a false issues-row block) nor lose
// the first id's closure record.
func TestImplementFirstIDWinsOnRepeatedKey(t *testing.T) {
	dir := closureRepo(t, "I001", "x", "\n")
	writeFile(t, dir, "docs/issues/I001-a.md", "---\nid: I001\nrelated:\n  id: I002\nstatus: fixed\ncommits: [a9ddea5]\n---\n\nx\n")
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	if issues := rowByName(t, rep.Stages, "issues"); issues.Verdict != stages.VerdictMatch {
		t.Errorf("issues verdict = %s (%s), want match — the first id: line resolves the file", issues.Verdict, issues.Detail)
	}
	if impl := rowByName(t, rep.Stages, "implement"); impl.Verdict != stages.VerdictMatch {
		t.Errorf("implement verdict = %s (%s), want match from I001's closure record", impl.Verdict, impl.Detail)
	}
}

// An unclosed fence still resolves the id (pre-I125 leniency kept) but is
// never a closure record: body prose read as frontmatter must not
// manufacture presence evidence over a pending implement stage.
func TestImplementUnclosedFenceIsNotClosureRecord(t *testing.T) {
	dir := closureRepo(t, "I001", " ", "\n")
	writeFile(t, dir, "docs/issues/I001-a.md", "---\nid: I001\ntitle: \"t\"\n\n## Problem\n\nstatus: fixed\ncommits: [a9ddea5]\n")
	rep, err := stages.Derive(dir)
	if err != nil {
		t.Fatal(err)
	}
	if issues := rowByName(t, rep.Stages, "issues"); issues.Verdict != stages.VerdictMatch {
		t.Errorf("issues verdict = %s (%s), want match — an unclosed fence still resolves id:", issues.Verdict, issues.Detail)
	}
	if impl := rowByName(t, rep.Stages, "implement"); impl.Verdict != stages.VerdictMatch {
		t.Errorf("implement verdict = %s (%s), want match (pending, no evidence) — prose must not be a closure record", impl.Verdict, impl.Detail)
	}
}

// Conservative rule: a ticket file with a broken frontmatter fence is no
// evidence and no error — the derivation degrades to ticked-missing, the
// same verdict an absent record gives.
func TestImplementMalformedTicketFrontmatterIsNoEvidence(t *testing.T) {
	dir := closureRepo(t, "I001", "x", "\n")
	writeFile(t, dir, "docs/issues/I001-a.md", "id: I001\nstatus: fixed\ncommits: [a9ddea5]\n")
	writeFile(t, dir, "docs/issues/I001-b.md", "---\nid: I001\n---\n")
	impl := implementRow(t, dir)
	if impl.Verdict != stages.VerdictTickedMissing {
		t.Fatalf("implement verdict = %s (%s), want ticked-missing — an unfenced file is not a closure record", impl.Verdict, impl.Detail)
	}
}
