package audit

import (
	"path/filepath"
	"strings"
	"testing"
)

func runFixture(t *testing.T, name string) Report {
	t.Helper()
	rep, err := Run(filepath.Join("testdata", name, "repo"), filepath.Join("testdata", name, "transcripts"))
	if err != nil {
		t.Fatalf("Run(%s): %v", name, err)
	}
	return rep
}

func rowsByID(t *testing.T, rep Report) map[string]TicketRow {
	t.Helper()
	m := map[string]TicketRow{}
	for _, r := range rep.Tickets {
		if _, dup := m[r.ID]; dup {
			t.Fatalf("duplicate row for %s", r.ID)
		}
		m[r.ID] = r
	}
	return m
}

// Acceptance: clean fixture (annotations match transcript) -> all-match
// report, nothing blocking, no warnings.
func TestCleanFixtureAllMatch(t *testing.T) {
	rep := runFixture(t, "clean")
	if len(rep.Tickets) != 2 {
		t.Fatalf("want 2 tickets, got %+v", rep.Tickets)
	}
	for _, r := range rep.Tickets {
		if r.Verdict != VerdictMatch {
			t.Errorf("%s: verdict = %s (%s), want match", r.ID, r.Verdict, r.Detail)
		}
	}
	if rep.Blocking() {
		t.Error("clean report must not be blocking")
	}
	if len(rep.Warnings) != 0 {
		t.Errorf("clean report must have no warnings, got %q", rep.Warnings)
	}
	if len(rep.Unmatched) != 0 {
		t.Errorf("clean report must have no unmatched dispatches, got %+v", rep.Unmatched)
	}
}

// The subagent transcript, not the dispatch alias, is the actual: I101 was
// dispatched with alias "sonnet" but its linked subagent transcript names
// the full model id.
func TestSubagentTranscriptIsTheActual(t *testing.T) {
	rows := rowsByID(t, runFixture(t, "clean"))
	got := strings.Join(rows["I101"].Actuals, ",")
	if got != "claude-sonnet-5" {
		t.Errorf("I101 actuals = %q, want claude-sonnet-5 (from subagent transcript)", got)
	}
	// I102 has no subagent transcript: the dispatch alias is the evidence.
	if got := strings.Join(rows["I102"].Actuals, ","); got != "fable" {
		t.Errorf("I102 actuals = %q, want fable (dispatch alias)", got)
	}
}

// Acceptance: escalation with a recorded reason -> advisory verdict.
func TestEscalationWithReasonAdvisory(t *testing.T) {
	rows := rowsByID(t, runFixture(t, "mixed"))
	r := rows["I201"]
	if r.Verdict != VerdictEscalatedWithReason {
		t.Fatalf("I201 verdict = %s (%s), want escalated-with-reason", r.Verdict, r.Detail)
	}
	if !strings.Contains(r.Detail, "integration teeth") {
		t.Errorf("I201 detail should carry the recorded reason, got %q", r.Detail)
	}
}

// Acceptance: dispatch below the annotated tier with no recorded reason ->
// silent-descent, and the report is blocking. The actual here comes from the
// subagent transcript (claude-sonnet-5) even though the dispatch alias said
// "fable" — the transcript is ground truth.
func TestSilentDescentBlocks(t *testing.T) {
	rep := runFixture(t, "mixed")
	rows := rowsByID(t, rep)
	r := rows["I202"]
	if r.Verdict != VerdictSilentDescent {
		t.Fatalf("I202 verdict = %s (%s), want silent-descent", r.Verdict, r.Detail)
	}
	if got := strings.Join(r.Actuals, ","); got != "claude-sonnet-5" {
		t.Errorf("I202 actuals = %q, want claude-sonnet-5 (transcript beats alias)", got)
	}
	if !rep.Blocking() {
		t.Error("a silent-descent verdict must make the report blocking")
	}
}

// Acceptance: model id absent from the repo's tier mapping -> warn.
func TestUnmappedDispatchWarns(t *testing.T) {
	rows := rowsByID(t, runFixture(t, "mixed"))
	if r := rows["I203"]; r.Verdict != VerdictUnmappedDispatch {
		t.Errorf("I203 verdict = %s (%s), want unmapped-dispatch", r.Verdict, r.Detail)
	}
}

// An annotated ticket with no dispatch and no transcript evidence -> warn.
// Its effort-escalation ledger record is not model evidence.
func TestNoTranscriptTicket(t *testing.T) {
	rows := rowsByID(t, runFixture(t, "mixed"))
	if r := rows["I204"]; r.Verdict != VerdictNoTranscript {
		t.Errorf("I204 verdict = %s (%s), want no-transcript", r.Verdict, r.Detail)
	}
}

// Acceptance: unannotated tickets are reported, never judged — even when
// their dispatch ran below every ordered tier.
func TestUnannotatedNeverJudged(t *testing.T) {
	rows := rowsByID(t, runFixture(t, "mixed"))
	r := rows["I205"]
	if r.Verdict != VerdictUnannotated {
		t.Errorf("I205 verdict = %s (%s), want unannotated", r.Verdict, r.Detail)
	}
	if got := strings.Join(r.Actuals, ","); got != "haiku" {
		t.Errorf("I205 actuals = %q, want the evidence still listed", got)
	}
	// An annotated-but-unknown tier value is reported, not judged.
	if r := rows["I209"]; r.Verdict != VerdictUnannotated || !strings.Contains(r.Detail, "turbo") {
		t.Errorf("I209 verdict = %s (%s), want unannotated naming the unknown tier", r.Verdict, r.Detail)
	}
}

// Fallback is lateral: covered by a FALLBACK ledger record -> advisory;
// uncovered -> warn-level unexplained-fallback, never blocking.
func TestFallbackCoverage(t *testing.T) {
	rows := rowsByID(t, runFixture(t, "mixed"))
	if r := rows["I206"]; r.Verdict != VerdictEscalatedWithReason || !strings.Contains(r.Detail, "security-framed") {
		t.Errorf("I206 verdict = %s (%s), want escalated-with-reason carrying the FALLBACK reason", r.Verdict, r.Detail)
	}
	if r := rows["I207"]; r.Verdict != VerdictUnexplainedFallback {
		t.Errorf("I207 verdict = %s (%s), want unexplained-fallback", r.Verdict, r.Detail)
	}
}

// Above-tier dispatch without a ledger record is surfaced as a warn-level
// verdict of its own — not blocking (quality went up), not silently a match.
func TestEscalationWithoutReasonWarns(t *testing.T) {
	rows := rowsByID(t, runFixture(t, "mixed"))
	if r := rows["I208"]; r.Verdict != VerdictEscalatedNoReason {
		t.Errorf("I208 verdict = %s (%s), want escalated-no-reason", r.Verdict, r.Detail)
	}
}

// An escalation record excuses only its recorded to-tier: I210 carries a
// recorded routine->primary escalation but was later re-dispatched on the
// mechanical tier — below the annotation and unrelated to the record. That
// is a genuine silent descent and must block at the Run boundary.
func TestEscalationRecordDoesNotExcuseUnrelatedDescent(t *testing.T) {
	rep := runFixture(t, "mixed")
	rows := rowsByID(t, rep)
	r := rows["I210"]
	if r.Verdict != VerdictSilentDescent {
		t.Fatalf("I210 verdict = %s (%s), want silent-descent — the routine->primary record must not excuse a mechanical dispatch", r.Verdict, r.Detail)
	}
	if !rep.Blocking() {
		t.Error("I210's descent must make the report blocking")
	}
}

// A reasoned DOWNWARD record excuses exactly its to-tier: recorded
// primary->routine descent stays advisory, never blocking.
func TestReasonedDescentStaysAdvisory(t *testing.T) {
	rows := rowsByID(t, runFixture(t, "mixed"))
	r := rows["I211"]
	if r.Verdict != VerdictEscalatedWithReason || !strings.Contains(r.Detail, "verbatim") {
		t.Errorf("I211 verdict = %s (%s), want escalated-with-reason carrying the recorded reason", r.Verdict, r.Detail)
	}
}

// Template and README files in docs/issues are not tickets.
func TestNonTicketFilesIgnored(t *testing.T) {
	rows := rowsByID(t, runFixture(t, "mixed"))
	want := []string{"I201", "I202", "I203", "I204", "I205", "I206", "I207", "I208", "I209", "I210", "I211"}
	if len(rows) != len(want) {
		t.Fatalf("want %d rows, got %v", len(want), rows)
	}
	for _, id := range want {
		if _, ok := rows[id]; !ok {
			t.Errorf("missing row for %s", id)
		}
	}
}

// Correlation: dispatches matching no ticket id are listed once as
// informational entries — deduped across session files, never judged.
func TestUnmatchedDispatchListedOnce(t *testing.T) {
	rep := runFixture(t, "mixed")
	if len(rep.Unmatched) != 1 {
		t.Fatalf("want exactly 1 unmatched dispatch, got %+v", rep.Unmatched)
	}
	if d := rep.Unmatched[0]; !strings.Contains(d.Description, "housekeeping") || d.Model != "sonnet" {
		t.Errorf("unmatched = %+v", d)
	}
}

// Acceptance: missing transcript dir -> warning + no-transcript verdicts,
// never an error and never blocking.
func TestMissingTranscriptDirDegrades(t *testing.T) {
	rep, err := Run(filepath.Join("testdata", "clean", "repo"), filepath.Join("testdata", "clean", "no-such-dir"))
	if err != nil {
		t.Fatalf("missing transcript dir must not error: %v", err)
	}
	if len(rep.Warnings) == 0 {
		t.Error("want a warning about the missing transcript dir")
	}
	for _, r := range rep.Tickets {
		if r.Verdict != VerdictNoTranscript {
			t.Errorf("%s: verdict = %s, want no-transcript", r.ID, r.Verdict)
		}
	}
	if rep.Blocking() {
		t.Error("missing transcripts must never block")
	}
}

// Acceptance: malformed JSONL -> per-file warning, remaining files still
// audited, never an error (parser rot must not fail builds).
func TestMalformedJSONLWarnsNeverFails(t *testing.T) {
	rep := runFixture(t, "degraded")
	var found bool
	for _, w := range rep.Warnings {
		if strings.Contains(w, "bad.jsonl") {
			found = true
		}
	}
	if !found {
		t.Errorf("want a warning naming bad.jsonl, got %q", rep.Warnings)
	}
	rows := rowsByID(t, rep)
	if r := rows["I301"]; r.Verdict != VerdictMatch {
		t.Errorf("I301 verdict = %s (%s), want match from the well-formed file", r.Verdict, r.Detail)
	}
	if r := rows["I302"]; r.Verdict != VerdictNoTranscript {
		t.Errorf("I302 verdict = %s, want no-transcript", r.Verdict)
	}
	if rep.Blocking() {
		t.Error("degraded fixture must not block")
	}
}

// A repo without docs/issues is a usage error, not a report.
func TestMissingIssuesDirErrors(t *testing.T) {
	if _, err := Run(t.TempDir(), filepath.Join("testdata", "clean", "transcripts")); err == nil {
		t.Fatal("want an error for a repo with no docs/issues")
	}
}

// Rows come back sorted by ticket id for deterministic output.
func TestRowsSortedByID(t *testing.T) {
	rep := runFixture(t, "mixed")
	for i := 1; i < len(rep.Tickets); i++ {
		if rep.Tickets[i-1].ID > rep.Tickets[i].ID {
			t.Fatalf("rows not sorted: %s before %s", rep.Tickets[i-1].ID, rep.Tickets[i].ID)
		}
	}
}

// The default transcripts dir is derived from the repo's absolute path with
// path separators and dots flattened to '-', under ~/.claude/projects.
func TestDefaultTranscriptsDir(t *testing.T) {
	got, err := DefaultTranscriptsDir("/Users/x/Projects/github.com/spine")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, filepath.Join(".claude", "projects", "-Users-x-Projects-github-com-spine")) {
		t.Errorf("DefaultTranscriptsDir = %q", got)
	}
}
