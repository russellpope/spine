package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runFixture(t *testing.T, name string) Report {
	t.Helper()
	rep, err := Run(Options{
		RepoDir:              filepath.Join("testdata", name, "repo"),
		ClaudeTranscriptsDir: filepath.Join("testdata", name, "transcripts"),
	})
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

// A discarded declaration is scoped to the immutable Claude dispatch event,
// not merely to a ticket or tier. Removing that correlation must make this
// regression fail: a routine prototype on a primary ticket would otherwise
// remain a blocking silent descent.
func TestDiscardedClaudeIdentityIsPerDispatch(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I078": "primary"})
	if err := os.MkdirAll(filepath.Join(repo, ".superpowers", "sdd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".superpowers", "sdd", "progress.md"), []byte(
		"DISCARDED I078 source:claude session:prototype dispatch:toolu_1 tier:routine reason: prototype was discarded\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	transcripts := t.TempDir()
	writeSingleDispatch(t, filepath.Join(transcripts, "prototype.jsonl"), repo, "I078", "I078 prototype", "claude-sonnet-5")

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	row := rowsByID(t, rep)["I078"]
	if got, want := string(row.Verdict), "discarded-with-reason"; got != want {
		t.Fatalf("I078 verdict = %q (%s), want %q", got, row.Detail, want)
	}
	if !strings.Contains(row.Detail, "prototype was discarded") {
		t.Fatalf("I078 detail = %q, want discarded reason", row.Detail)
	}
	if rep.Blocking() {
		t.Fatal("one exact discarded prototype must be advisory, not blocking")
	}
}

// A discarded declaration covers one exact event. A later routine dispatch
// for the same primary ticket remains a real silent descent and must win the
// ticket aggregation, while retaining the discarded prototype's reason.
func TestDiscardedDoesNotExcuseLandedSibling(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I078": "primary"})
	if err := os.MkdirAll(filepath.Join(repo, ".superpowers", "sdd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".superpowers", "sdd", "progress.md"), []byte(
		"DISCARDED I078 source:claude session:prototype dispatch:toolu_1 tier:routine reason: prototype was discarded\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	transcripts := t.TempDir()
	writeSingleDispatch(t, filepath.Join(transcripts, "prototype.jsonl"), repo, "I078", "I078 prototype", "claude-sonnet-5")
	writeSingleDispatch(t, filepath.Join(transcripts, "landed.jsonl"), repo, "I078", "I078 landed work", "claude-sonnet-5")

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	row := rowsByID(t, rep)["I078"]
	if row.Verdict != VerdictSilentDescent || !rep.Blocking() {
		t.Fatalf("I078 = %s (%s), blocking=%v; want blocking silent descent", row.Verdict, row.Detail, rep.Blocking())
	}
	if !strings.Contains(row.Detail, "prototype was discarded") {
		t.Fatalf("I078 detail = %q, want preserved discarded reason", row.Detail)
	}
}

func TestDiscardedAbsentKeepsSilentDescent(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I078": "primary"})
	transcripts := t.TempDir()
	writeSingleDispatch(t, filepath.Join(transcripts, "prototype.jsonl"), repo, "I078", "I078 prototype", "claude-sonnet-5")
	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	if row := rowsByID(t, rep)["I078"]; row.Verdict != VerdictSilentDescent || !rep.Blocking() {
		t.Fatalf("I078 = %s (%s), blocking=%v; want blocking silent descent", row.Verdict, row.Detail, rep.Blocking())
	}
}

func TestDiscardedWrongIdentityOrTierDoesNotExcuse(t *testing.T) {
	for _, record := range []string{
		"DISCARDED I078 source:codex session:prototype dispatch:toolu_1 tier:routine reason: wrong source",
		"DISCARDED I078 source:claude session:other dispatch:toolu_1 tier:routine reason: wrong session",
		"DISCARDED I078 source:claude session:prototype dispatch:other tier:routine reason: wrong dispatch",
		"DISCARDED I078 source:claude session:prototype dispatch:toolu_1 tier:mechanical reason: wrong tier",
	} {
		t.Run(record, func(t *testing.T) {
			repo := t.TempDir()
			writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I078": "primary"})
			if err := os.MkdirAll(filepath.Join(repo, ".superpowers", "sdd"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(repo, ".superpowers", "sdd", "progress.md"), []byte(record+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			transcripts := t.TempDir()
			writeSingleDispatch(t, filepath.Join(transcripts, "prototype.jsonl"), repo, "I078", "I078 prototype", "claude-sonnet-5")
			rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
			if err != nil {
				t.Fatal(err)
			}
			if row := rowsByID(t, rep)["I078"]; row.Verdict != VerdictSilentDescent || !rep.Blocking() {
				t.Fatalf("I078 = %s (%s), blocking=%v; want blocking silent descent", row.Verdict, row.Detail, rep.Blocking())
			}
		})
	}
}

func TestDiscardedMalformedDuplicateAndAmbiguousRecordsDoNotExcuse(t *testing.T) {
	for _, tc := range []struct {
		name, ledger string
		ambiguous    bool
	}{
		{"missing dispatch", "DISCARDED I078 source:claude session:prototype tier:routine reason: missing dispatch", false},
		{"reordered fields", "DISCARDED I078 session:prototype source:claude dispatch:toolu_1 tier:routine reason: reordered", false},
		{"empty reason", "DISCARDED I078 source:claude session:prototype dispatch:toolu_1 tier:routine reason: ", false},
		{"duplicate identity", "DISCARDED I078 source:claude session:prototype dispatch:toolu_1 tier:routine reason: first\nDISCARDED I078 source:claude session:prototype dispatch:toolu_1 tier:routine reason: second", false},
		{"ambiguous evidence", "DISCARDED I078 source:claude session:prototype dispatch:toolu_1 tier:routine reason: too coarse", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I078": "primary"})
			if err := os.MkdirAll(filepath.Join(repo, ".superpowers", "sdd"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(repo, ".superpowers", "sdd", "progress.md"), []byte(tc.ledger+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			transcripts := t.TempDir()
			path := filepath.Join(transcripts, "prototype.jsonl")
			writeSingleDispatch(t, path, repo, "I078", "I078 prototype", "claude-sonnet-5")
			if tc.ambiguous {
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, append(raw, raw...), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
			if err != nil {
				t.Fatal(err)
			}
			if row := rowsByID(t, rep)["I078"]; row.Verdict != VerdictSilentDescent || !rep.Blocking() {
				t.Fatalf("I078 = %s (%s), blocking=%v; want blocking silent descent", row.Verdict, row.Detail, rep.Blocking())
			}
			if got := strings.Join(rep.Warnings, "\n"); !strings.Contains(got, "DISCARDED") {
				t.Fatalf("warnings = %q, want discarded grammar diagnostic", rep.Warnings)
			}
		})
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

// Acceptance (D27, ticket I046): a ticket declaring tier: n/a opts out of
// routing judgment entirely — reported exempt, distinct from unannotated,
// even when its dispatch evidence would otherwise be a silent-descent.
func TestExemptTierNeverJudged(t *testing.T) {
	rows := rowsByID(t, runFixture(t, "mixed"))
	r := rows["I212"]
	if r.Verdict != VerdictExempt {
		t.Errorf("I212 verdict = %s (%s), want exempt", r.Verdict, r.Detail)
	}
	if r.Verdict == VerdictUnannotated {
		t.Error("I212 must not be reported as unannotated — n/a is a decision, not an absence")
	}
	if got := strings.Join(r.Actuals, ","); got != "haiku" {
		t.Errorf("I212 actuals = %q, want the evidence still listed even though unjudged", got)
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
	want := []string{"I201", "I202", "I203", "I204", "I205", "I206", "I207", "I208", "I209", "I210", "I211", "I212"}
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
	rep, err := Run(Options{
		RepoDir:              filepath.Join("testdata", "clean", "repo"),
		ClaudeTranscriptsDir: filepath.Join("testdata", "clean", "no-such-dir"),
	})
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
	if _, err := Run(Options{RepoDir: t.TempDir(), ClaudeTranscriptsDir: filepath.Join("testdata", "clean", "transcripts")}); err == nil {
		t.Fatal("want an error for a repo with no docs/issues")
	}
}

// A repo where zero docs/issues tickets carry a tier: annotation audits
// vacuously — every row is unannotated, nothing is ever judged, and the
// report exits 0. That "clean" pass is indistinguishable from a real one
// unless the report itself says so: a repo-wide warning must call out that
// nothing was actually audited.
func TestVacuousAuditWarns(t *testing.T) {
	rep := runFixture(t, "vacuous")
	for _, r := range rep.Tickets {
		if r.Verdict != VerdictUnannotated {
			t.Errorf("%s: verdict = %s, want unannotated (fixture has no tier: annotations)", r.ID, r.Verdict)
		}
	}
	found := false
	for _, w := range rep.Warnings {
		if strings.Contains(w, "no annotated tickets") {
			found = true
		}
	}
	if !found {
		t.Errorf("vacuous audit (zero tickets carry a tier: annotation) must warn, got %q", rep.Warnings)
	}
	if rep.Blocking() {
		t.Error("a vacuous audit must never block")
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
