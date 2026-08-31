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

func TestDeclarationVerdictMatrixAndAggregation(t *testing.T) {
	base := DeclarationEvidence{ExpectedEffort: "high", DeclaredEffort: "high", ModelStatus: DeclarationModelConfirmed}
	for _, tc := range []struct {
		name       string
		evidence   DeclarationEvidence
		authorized bool
		want       Verdict
	}{
		{"confirmed exact observed effort", withObservedEffort(base, "high"), false, VerdictDeclarationConfirmed},
		{"confirmed model absent observed effort", base, false, VerdictDeclarationUnconfirmable},
		{"confirmed model different observed effort", withObservedEffort(base, "low"), false, VerdictDeclarationObservedMismatch},
		{"confirmed model absent declaration", DeclarationEvidence{ExpectedEffort: "high", ModelStatus: DeclarationModelConfirmed}, false, VerdictDeclarationUnconfirmable},
		{"confirmed model unauthorized declaration", DeclarationEvidence{ExpectedEffort: "high", DeclaredEffort: "low", ModelStatus: DeclarationModelConfirmed}, false, VerdictDeclarationEffortMismatch},
		{"confirmed model authorized retry", DeclarationEvidence{ExpectedEffort: "high", DeclaredEffort: "low", ObservedEffort: "low", ModelStatus: DeclarationModelConfirmed}, true, VerdictDeclarationConfirmed},
		{"model mismatch outranks effort", DeclarationEvidence{ExpectedEffort: "high", DeclaredEffort: "low", ModelStatus: DeclarationModelMismatch}, false, VerdictDeclarationObservedMismatch},
		{"unconfirmable model unauthorized declaration", DeclarationEvidence{ExpectedEffort: "high", DeclaredEffort: "low", ModelStatus: DeclarationModelUnconfirmable}, false, VerdictDeclarationEffortMismatch},
		{"unconfirmable model otherwise", DeclarationEvidence{ExpectedEffort: "high", DeclaredEffort: "high", ModelStatus: DeclarationModelUnconfirmable}, false, VerdictDeclarationUnconfirmable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := judgeDeclarationEvidence(tc.evidence, tc.authorized)
			if got.Verdict != tc.want {
				t.Fatalf("verdict = %q, want %q", got.Verdict, tc.want)
			}
		})
	}

	unconfirmable := judgeDeclarationEvidence(base, false)
	blocking := judgeDeclarationEvidence(DeclarationEvidence{ExpectedEffort: "high", DeclaredEffort: "low", ModelStatus: DeclarationModelUnconfirmable}, false)
	verdict, _, ok := aggregateDeclarationEvents([]DeclarationEvidence{unconfirmable, blocking})
	if !ok || verdict != VerdictDeclarationEffortMismatch {
		t.Fatalf("aggregate = %q present=%v, want declared-effort-mismatch", verdict, ok)
	}
	if !(Report{Tickets: []TicketRow{{Verdict: verdict}}}).Blocking() {
		t.Fatal("declared effort mismatch must block")
	}
}

func withObservedEffort(evidence DeclarationEvidence, effort string) DeclarationEvidence {
	evidence.ObservedEffort = effort
	return evidence
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

// A Claude tool_use id is only unique within its session. A sidecar from a
// different session that reuses the id must neither suppress the prototype
// dispatch nor inherit its ticket attribution.
func TestDiscardedClaudeLinkedDispatchIdentityIncludesSession(t *testing.T) {
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
	// The other session's sidecar names no ticket. It can reach I078 only if
	// the audit cross-links it to the prototype by tool_use.id alone.
	writeOrphanSubagent(t, transcripts, "other", "same-id", "toolu_1", repo, "unrelated worker", "claude-sonnet-5")

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	row := rowsByID(t, rep)["I078"]
	if row.Verdict != VerdictDiscardedWithReason || rep.Blocking() {
		t.Fatalf("I078 = %s (%s), blocking=%v; cross-session same tool_use.id must not link", row.Verdict, row.Detail, rep.Blocking())
	}
}

// A Claude dispatch cannot be linked to a Codex worker by a colliding coarse
// tool id. The source is part of the immutable identity as well as session
// and dispatch event.
func TestDiscardedClaudeLinkedDispatchIdentityIncludesSource(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I078": "primary"})
	if err := os.MkdirAll(filepath.Join(repo, ".superpowers", "sdd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".superpowers", "sdd", "progress.md"), []byte(
		"DISCARDED I078 source:claude session:prototype dispatch:codex:root-1 tier:routine reason: prototype was discarded\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	transcripts := t.TempDir()
	prototype := filepath.Join(transcripts, "prototype.jsonl")
	writeSingleDispatch(t, prototype, repo, "I078", "I078 prototype", "claude-sonnet-5")
	raw, err := os.ReadFile(prototype)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prototype, []byte(strings.Replace(string(raw), "toolu_1", "codex:root-1", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	codexDir := t.TempDir()
	writeCodexFile(t, filepath.Join(codexDir, "worker.jsonl"),
		codexSessionMetaLine("worker", "root-1", "root-1", repo, "subagent", threadSpawnSource("root-1")),
		codexUserMessageLine("I078 unrelated Codex worker"),
		codexTurnContextLine("gpt-5.6-terra"),
	)

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts, CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	row := rowsByID(t, rep)["I078"]
	if row.Verdict != VerdictDiscardedWithReason || rep.Blocking() {
		t.Fatalf("I078 = %s (%s), blocking=%v; cross-source same coarse id must not link", row.Verdict, row.Detail, rep.Blocking())
	}
}

// Direct Codex dispatches retain their documented root-thread linkage only to
// Codex spawned-thread actuals. A Claude sidecar with the same coarse id must
// not suppress or replace that direct dispatch evidence.
func TestDiscardedCodexLinkedDispatchIdentityIncludesSource(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I078": "primary"})
	if err := os.MkdirAll(filepath.Join(repo, ".superpowers", "sdd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".superpowers", "sdd", "progress.md"), []byte(
		"DISCARDED I078 source:codex session:root-1 dispatch:call_discard tier:routine reason: prototype was discarded\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	codexDir := t.TempDir()
	writeCodexFile(t, filepath.Join(codexDir, "lead.jsonl"),
		codexSessionMetaLine("root-1", "root-1", "", repo, "user", topLevelSource),
		codexFunctionCallLineWithID("spawn_agent", "call_discard", map[string]string{
			"task_name": "I078 prototype", "model": "gpt-5.6-terra",
		}),
	)
	transcripts := t.TempDir()
	writeOrphanSubagent(t, transcripts, "other", "same-id", "codex:root-1", repo, "unrelated worker", "claude-sonnet-5")

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts, CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	row := rowsByID(t, rep)["I078"]
	if row.Verdict != VerdictDiscardedWithReason || rep.Blocking() {
		t.Fatalf("I078 = %s (%s), blocking=%v; cross-source Codex root must not link", row.Verdict, row.Detail, rep.Blocking())
	}
}

func TestDiscardedQuotedIdentityFieldsAreMalformed(t *testing.T) {
	for _, record := range []string{
		`DISCARDED I078 source:claude session:"prototype" dispatch:toolu_1 tier:routine reason: quoted session`,
		`DISCARDED I078 source:claude session:prototype dispatch:"toolu_1" tier:routine reason: quoted dispatch`,
		`DISCARDED I078 source:claude session:'prototype' dispatch:toolu_1 tier:routine reason: single-quoted session`,
		`DISCARDED I078 source:claude session:prototype dispatch:'toolu_1' tier:routine reason: single-quoted dispatch`,
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
				t.Fatalf("I078 = %s (%s), blocking=%v; quoted identity must not excuse", row.Verdict, row.Detail, rep.Blocking())
			}
			if got, want := strings.Join(rep.Warnings, "\n"), "DISCARDED line 1 malformed — ignored"; !strings.Contains(got, want) {
				t.Fatalf("warnings = %q, want %q", got, want)
			}
		})
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

func TestReadLedgerParsesOnlyExactEffortAuthorizationPairs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.md")
	content := strings.Join([]string{
		"ESCALATION I075 effort low->xhigh reason: retry budget",
		"ESCALATION I075 effort `medium`->low=>xhigh reason: arbitrary raw endpoint bytes",
		"ESCALATION I075 effort low→medium->[xhigh] reason: arbitrary unicode endpoint bytes",
		"ESCALATION I075 effort xhigh ->low reason: spaced arrow",
		"ESCALATION I075 effort medium->low->xhigh reason: repeated arrow endpoint",
		"ESCALATION I075 effort low->xhigh reason: one reason: duplicate",
		"ESCALATION I076 effort low->xhigh reason: other ticket",
		"- ESCALATION I075 effort low->xhigh reason: Markdown list",
		"* ESCALATION I075 effort low->xhigh reason: Markdown list",
		"+ ESCALATION I075 effort low->xhigh reason: Markdown list",
		"  ESCALATION I075 effort low->xhigh reason: indentation",
		" ESCALATION I075 effort low->xhigh reason: leading whitespace",
		"ESCALATION I075 effort low->xhigh reason: trailing whitespace ",
		"> ESCALATION I075 effort low->xhigh reason: quote decoration",
		"`ESCALATION I075 effort low->xhigh reason: inline code`",
		"```text",
		"ESCALATION I075 effort low->xhigh reason: fenced code",
		"```",
		"ESCALATION I075 effort xhigh->max reason: distinct post-fence retry",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	l := readLedger(path)
	got := l.effortEscalations["I075"]
	if len(got) != 4 || got[0].from != "low" || got[0].to != "xhigh" || got[0].reason != "retry budget" || got[1].from != "`medium`" || got[1].to != "low=>xhigh" || got[2].from != "low→medium" || got[2].to != "[xhigh]" || got[3].from != "xhigh" || got[3].to != "max" || got[3].reason != "distinct post-fence retry" {
		t.Fatalf("effort ledger = %+v, want only the four literal I075 records", got)
	}
	if !effortAuthorized(l, "I075", "`medium`", "low=>xhigh") || !effortAuthorized(l, "I075", "low→medium", "[xhigh]") {
		t.Fatal("arbitrary non-space raw endpoint bytes were not retained exactly")
	}
	if !effortAuthorized(l, "I075", "xhigh", "max") {
		t.Fatal("distinct literal pair after a fenced block was not authorized")
	}
	if effortAuthorized(l, "I075", "xhigh", "low") {
		t.Fatal("reversed pair was authorized")
	}
	if effortAuthorized(l, "I077", "low", "xhigh") {
		t.Fatal("other ticket pair was authorized")
	}
}

func TestEffortDeclarationsKeepRetriesAndNeverChangeObservedEffort(t *testing.T) {
	ticket := ticket{id: "I075", tier: "primary"}
	dispatches := []dispatch{
		{harness: "claude", model: "claude-fable-5", effort: "high", effortSource: "--effort"},
		{harness: "claude", model: "claude-fable-5", effort: "low", effortSource: "--effort"},
	}
	l := ledger{effortEscalations: map[string][]effortEscalation{
		"I075": {{from: "high", to: "low", reason: "retry budget"}},
	}}
	expected, declared, status, observed := summarizeEffortDeclarations(t.TempDir(), ticket, dispatches, l)
	if expected != "high,high" || declared != "high,low" {
		t.Fatalf("expected/declaration = %q/%q, want high,high/high,low", expected, declared)
	}
	if status != "target-match,exact-authorized-deviation" {
		t.Fatalf("status = %q", status)
	}
	if observed != "-" {
		t.Fatalf("observed effort = %q, want declared-only '-'", observed)
	}
}
