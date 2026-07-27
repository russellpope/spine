package audit

import (
	"path/filepath"
	"strings"
	"testing"
)

// --- I049: codex discovery pruning — cheap pre-filter before JSONL parse ---
//
// Sound basis (ticket text): every codex evidence path requires an audited
// ticket token as a literal byte string in the file (spawn_agent task_name,
// team-spawn command text, opening user message) — EXCEPT D20 clause 2,
// spawned-thread actuals, which link purely by shared root id and never by
// ticket-token text in the subagent file's own bytes
// (TestCodexSpawnedThreadActualSupersedesDeclared's sub.jsonl is the
// existing regression fixture for that). codexMayContribute closes that gap
// by exempting every subagent-shaped file (thread_source "subagent", real
// codex-native subagent or guardian) via the "subagent" JSON marker, which
// only ever appears in that context (session_meta's thread_source value or
// its source.subagent key) — see codexSessionMetaLine/threadSpawnSource/
// guardianSource in codex_test.go for the exact shapes.

// TestCodexMayContributeMatchesTicketTokenCaseInsensitive pins the token
// half of the predicate: a token present anywhere in the raw bytes, matched
// case-insensitively (codex's task_name convention lowercases ticket ids,
// D20's "Flavor threading" closing paragraph) — and absent otherwise.
func TestCodexMayContributeMatchesTicketTokenCaseInsensitive(t *testing.T) {
	data := []byte(`{"type":"response_item","payload":{"type":"function_call","name":"spawn_agent","arguments":"{\"task_name\":\"i049 fix\"}"}}`)
	if !codexMayContribute(data, []string{"I049"}) {
		t.Fatal("expected match: lowercase i049 in raw bytes against uppercase ticket id I049")
	}
	if codexMayContribute(data, []string{"I050"}) {
		t.Fatal("expected no match: I050 not present in raw bytes")
	}
}

// TestCodexMayContributeExemptsSubagentShapedFiles pins the D20-clause-2
// carve-out: a file whose session_meta marks it thread_source "subagent"
// must never be pruned on token absence alone.
func TestCodexMayContributeExemptsSubagentShapedFiles(t *testing.T) {
	data := []byte(codexSessionMetaLine("sub-1", "root-1", "root-1", "/repo", "subagent", threadSpawnSource("root-1")))
	if !codexMayContribute(data, []string{"I999"}) {
		t.Fatal("expected subagent-shaped file to survive pruning even with no ticket token present")
	}
}

// TestCodexMayContributeSkipsPlainFileWithNoTokenOrMarker is the contrast
// case: a plain top-level session mentioning no audited ticket is prunable.
func TestCodexMayContributeSkipsPlainFileWithNoTokenOrMarker(t *testing.T) {
	data := []byte(codexSessionMetaLine("top-1", "root-1", "", "/repo", "user", "{}"))
	if codexMayContribute(data, []string{"I999"}) {
		t.Fatal("expected a plain top-level session with no ticket token to be pruned")
	}
}

// TestCodexPruneOmitsMalformedFileWarningWhenTokenAbsent pins the chosen
// warning-suppression semantics (I049 dispatch brief): a malformed junk file
// that never mentions any audited ticket token, and is not subagent-shaped,
// is pruned before scanCodexFile ever opens it — so its "no session_meta
// line — skipped" warning never fires. No existing fixture pins that
// warning firing for a token-less file (checked before choosing this
// semantics), so this is new, deliberate behavior, not a regression.
func TestCodexPruneOmitsMalformedFileWarningWhenTokenAbsent(t *testing.T) {
	codexDir := t.TempDir()
	rep, _ := runCodexFixtureWithDispatch(t, "I049", "mechanical", codexDir,
		codexFunctionCallLine("spawn_agent", map[string]string{
			"model":     "gpt-5.6-luna",
			"task_name": "i049 pruning fix",
		}))
	writeCodexFile(t, filepath.Join(codexDir, "junk.jsonl"), "not json at all {{{")

	r := rowsByID(t, rep)["I049"]
	if r.Verdict != VerdictMatch {
		t.Fatalf("I049 verdict = %s (%s), want match (an unrelated pruned junk file must not interfere)", r.Verdict, r.Detail)
	}
	for _, w := range rep.Warnings {
		if strings.Contains(w, "junk.jsonl") {
			t.Errorf("unexpected warning about pruned junk.jsonl: %q", w)
		}
	}
}

// TestCodexPruneKeepsMalformedFileWarningWhenTokenPresent is the contrast
// case: a malformed file whose garbage bytes DO mention an audited ticket
// token survives the pre-filter (it might carry real evidence; the
// pre-filter cannot prove otherwise) and its "no session_meta line —
// skipped" warning still fires exactly as it did before pruning existed.
func TestCodexPruneKeepsMalformedFileWarningWhenTokenPresent(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I049": "routine"})
	writeCodexFile(t, filepath.Join(codexDir, "junk.jsonl"), "not json at all {{{ I049")

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range rep.Warnings {
		if strings.Contains(w, "junk.jsonl") && strings.Contains(w, "no session_meta line") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'no session_meta line — skipped' warning for token-bearing junk.jsonl; warnings=%v", rep.Warnings)
	}
}

// TestCodexPruneDoesNotSuppressSessionMatchedWarningWhenTokenAbsent pins the
// I049 ordering decision: the token pre-filter is skipped whenever --session
// is set, so a --session id that matches a file mentioning no audited
// ticket token at all still counts as "matched" for M3's diagnostic (I047
// review) — exactly as before pruning existed. Pruning that file on token
// absence would silently flip the diagnostic and violate AC2 (byte-identical
// reports): --session queries are a narrow, manual query, not the
// whole-store sweep the perf case (~953 files) targets, so skipping the
// pre-filter there costs nothing that matters.
func TestCodexPruneDoesNotSuppressSessionMatchedWarningWhenTokenAbsent(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I991": "routine"})
	codexDir := t.TempDir()
	writeCodexFile(t, filepath.Join(codexDir, "quiet.jsonl"),
		codexSessionMetaLine("root-quiet", "root-quiet", "", dir, "user", "{}"),
		codexTurnContextLine("gpt-5.6-sol"), // no dispatch, no ticket mention anywhere in this file
	)

	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir, Session: "root-quiet"})
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range rep.Warnings {
		if strings.Contains(w, "matched no sessions") {
			t.Errorf("--session root-quiet exists in the store (token-less file); pruning must not make it look unmatched, got %q", rep.Warnings)
		}
	}
}
