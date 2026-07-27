package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- codex fixture helpers ---
//
// Hand-written minimal JSONL encoding I009's verified 2026-07-25 format
// facts (line kinds, thread-tree ids, guardian markers, dispatch shapes).
// The exact byte-for-byte codex wire format is undocumented (I009/testing
// decisions); these helpers encode a defensible, internally-consistent
// reading of the prose facts, not a re-derivation from ~/.codex.

func writeCodexFile(t *testing.T, path string, lines ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// codexSessionMetaLine builds a session_meta line. sourceJSON is the raw
// "source" object: "{}" for a plain top-level user session, or one of the
// guardian/thread_spawn shapes I009 documents.
func codexSessionMetaLine(id, sessionID, parent, cwd, threadSource, sourceJSON string) string {
	return fmt.Sprintf(
		`{"type":"session_meta","payload":{"id":%q,"session_id":%q,"parent_thread_id":%q,"cwd":%q,"thread_source":%q,"model":null,"source":%s}}`,
		id, sessionID, parent, cwd, threadSource, sourceJSON)
}

func codexTurnContextLine(model string) string {
	return fmt.Sprintf(`{"type":"turn_context","payload":{"model":%q}}`, model)
}

// codexFunctionCallLine builds a response_item/function_call line with
// arguments JSON-string-encoded, the OpenAI-style function-calling
// convention I009's prose implies ("arguments carry the explicit model").
func codexFunctionCallLine(name string, argsObj map[string]string) string {
	argsJSON, err := json.Marshal(argsObj)
	if err != nil {
		panic(err)
	}
	encoded, err := json.Marshal(string(argsJSON))
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf(`{"type":"response_item","payload":{"type":"function_call","name":%q,"call_id":"call_1","arguments":%s}}`, name, encoded)
}

const guardianSource = `{"subagent":{"other":"guardian"}}`

func threadSpawnSource(parent string) string {
	return fmt.Sprintf(`{"subagent":{"thread_spawn":{"parent_thread_id":%q,"depth":1,"agent_path":"worker","agent_nickname":"worker"}}}`, parent)
}

// runCodexFixture audits a fresh repo (gen-9 default WORKFLOW.md, one ticket
// per (id, tier)) against the given codex sessions dir; no claude transcripts
// participate.
func runCodexFixture(t *testing.T, tickets map[string]string, codexDir string) (Report, string) {
	t.Helper()
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, tickets)
	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	return rep, dir
}

// Acceptance: a herdr-shaped fixture — a lead session recording a team spawn
// command with an explicit -m model flag, plus an unrelated top-level worker
// session sitting alongside it in the sessions dir — judges its routine
// ticket match from the dispatch record alone. The worker session (a
// top-level "user" thread, not a thread_spawn subagent — I009) is NOT
// codex-native subagent evidence and is deliberately given a different,
// wrong-looking model in its own turn_context: if the reader mistakenly
// treated it as ticket evidence (worker-session-scan, I042's job, explicitly
// out of scope here), the ticket would misjudge or show a spurious actual.
func TestCodexHerdrDispatchRecordJudgesMatch(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I041": "routine"})

	writeCodexFile(t, filepath.Join(codexDir, "lead.jsonl"),
		codexSessionMetaLine("root-1", "root-1", "", sessRepo, "user", "{}"),
		codexFunctionCallLine("exec_command", map[string]string{
			"command": "herdr agent start moo-clone-worker1 --kind codex --pane wM:p2 -- -m gpt-5.6-terra",
		}),
		codexFunctionCallLine("exec_command", map[string]string{
			"command": `herdr agent prompt moo-clone-worker1 "$(<.superpowers/sdd/2026-07-26-i041/dispatch-task-I041.md)"`,
		}),
	)
	writeCodexFile(t, filepath.Join(codexDir, "worker.jsonl"),
		codexSessionMetaLine("worker-1", "worker-1", "", sessRepo, "user", "{}"),
		codexTurnContextLine("gpt-5.6-sol"), // decoy: must never surface as I041 evidence
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)
	r := rows["I041"]
	if r.Verdict != VerdictMatch {
		t.Fatalf("I041 verdict = %s (%s), want match", r.Verdict, r.Detail)
	}
	if got := strings.Join(r.Actuals, ","); got != "gpt-5.6-terra" {
		t.Errorf("I041 actuals = %q, want gpt-5.6-terra only (worker session must not leak in)", got)
	}
}

// Acceptance: a spawn_agent fixture whose task_name carries the ticket token
// lowercase (codex convention, per D20) still claims the ticket, matched
// case-insensitively, with the declared model.
func TestCodexSpawnAgentLowercaseTaskNameClaimsTicket(t *testing.T) {
	codexDir := t.TempDir()
	rep, _ := runCodexFixtureWithDispatch(t, "I042", "mechanical", codexDir,
		codexFunctionCallLine("spawn_agent", map[string]string{
			"model":     "gpt-5.6-luna",
			"task_name": "i042 quick mechanical fix",
		}))
	r := rowsByID(t, rep)["I042"]
	if r.Verdict != VerdictMatch {
		t.Fatalf("I042 verdict = %s (%s), want match", r.Verdict, r.Detail)
	}
	if got := strings.Join(r.Actuals, ","); got != "gpt-5.6-luna" {
		t.Errorf("I042 actuals = %q, want gpt-5.6-luna", got)
	}
}

// runCodexFixtureWithDispatch writes a single lead session (root id "root")
// carrying the given function_call line(s), audits ticket id at tier, and
// returns the report plus the repo dir.
func runCodexFixtureWithDispatch(t *testing.T, id, tier, codexDir string, callLines ...string) (Report, string) {
	t.Helper()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{id: tier})
	lines := append([]string{codexSessionMetaLine("root", "root", "", sessRepo, "user", "{}")}, callLines...)
	writeCodexFile(t, filepath.Join(codexDir, "lead.jsonl"), lines...)
	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	return rep, sessRepo
}

// Acceptance: spawned-thread actuals supersede the dispatch's declared
// model when both exist — exactly as a linked claude subagent transcript
// beats its dispatch's alias. The dispatch declares gpt-5.6-luna
// (mechanical); the linked thread_spawn subagent's turn_context reports
// gpt-5.6-terra (routine), the actual that must win.
func TestCodexSpawnedThreadActualSupersedesDeclared(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I043": "routine"})

	writeCodexFile(t, filepath.Join(codexDir, "lead.jsonl"),
		codexSessionMetaLine("root-3", "root-3", "", sessRepo, "user", "{}"),
		codexFunctionCallLine("spawn_agent", map[string]string{
			"model":     "gpt-5.6-luna", // declared, must be superseded
			"task_name": "i043 subagent dispatch",
		}),
	)
	writeCodexFile(t, filepath.Join(codexDir, "sub.jsonl"),
		codexSessionMetaLine("sub-3a", "root-3", "root-3", sessRepo, "subagent", threadSpawnSource("root-3")),
		codexTurnContextLine("gpt-5.6-terra"), // actual, must win
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I043"]
	if r.Verdict != VerdictMatch {
		t.Fatalf("I043 verdict = %s (%s), want match (spawned-thread actual, not the declared alias)", r.Verdict, r.Detail)
	}
	if got := strings.Join(r.Actuals, ","); got != "gpt-5.6-terra" {
		t.Errorf("I043 actuals = %q, want gpt-5.6-terra only — the declared gpt-5.6-luna must be superseded", got)
	}
}

// Acceptance: a guardian-only fixture contributes no evidence to any
// ticket. Guardian threads report a synthetic model (codex-auto-review) and
// are structurally excluded (D23) — even with no competing legitimate
// evidence, the ticket must land on no-transcript, never pick up the
// synthetic token.
func TestCodexGuardianContributesNoEvidence(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I044": "routine"})

	writeCodexFile(t, filepath.Join(codexDir, "guardian.jsonl"),
		codexSessionMetaLine("guard-4", "root-4", "root-4", sessRepo, "subagent", guardianSource),
		codexTurnContextLine("codex-auto-review"),
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I044"]
	if r.Verdict != VerdictNoTranscript {
		t.Fatalf("I044 verdict = %s (%s), want no-transcript — guardian threads must contribute nothing", r.Verdict, r.Detail)
	}
	if len(r.Actuals) != 0 {
		t.Errorf("I044 actuals = %v, want none", r.Actuals)
	}
}

// Acceptance: a model-switching session contributes each turn's model, not
// one per file. The linked subagent thread's two turn_context lines report
// different models; both must surface as evidence, and the worse one
// (mechanical, below the routine annotation) must drive the verdict — proof
// the second turn actually reached judgment, not just the first.
func TestCodexModelSwitchingSessionContributesEachTurn(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I045": "routine"})

	writeCodexFile(t, filepath.Join(codexDir, "lead.jsonl"),
		codexSessionMetaLine("root-5", "root-5", "", sessRepo, "user", "{}"),
		codexFunctionCallLine("spawn_agent", map[string]string{
			"model":     "gpt-5.6-terra",
			"task_name": "i045 switch dispatch",
		}),
	)
	writeCodexFile(t, filepath.Join(codexDir, "sub.jsonl"),
		codexSessionMetaLine("sub-5a", "root-5", "root-5", sessRepo, "subagent", threadSpawnSource("root-5")),
		codexTurnContextLine("gpt-5.6-terra"),
		codexTurnContextLine("gpt-5.6-luna"),
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I045"]
	if got := strings.Join(r.Actuals, ","); got != "gpt-5.6-luna,gpt-5.6-terra" {
		t.Fatalf("I045 actuals = %q, want both turn models (sorted)", got)
	}
	if r.Verdict != VerdictSilentDescent {
		t.Errorf("I045 verdict = %s (%s), want silent-descent — the mechanical turn must reach judgment", r.Verdict, r.Detail)
	}
}

// Acceptance: a missing/unreadable codex sessions dir degrades to a
// warning, never an error, and never changes verdicts driven by claude-side
// evidence.
func TestCodexMissingSessionsDirDegradesToWarning(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I046": "primary"})
	tdir := t.TempDir()
	writeDispatchTranscript(t, tdir, map[string]string{"I046": "fable"})

	rep, err := Run(Options{
		RepoDir:              dir,
		ClaudeTranscriptsDir: tdir,
		CodexSessionsDir:     filepath.Join(t.TempDir(), "no-such-codex-dir"),
	})
	if err != nil {
		t.Fatalf("missing codex sessions dir must not error: %v", err)
	}
	found := false
	for _, w := range rep.Warnings {
		if strings.Contains(w, "codex sessions dir unreadable") {
			found = true
		}
	}
	if !found {
		t.Errorf("want a warning about the missing codex sessions dir, got %q", rep.Warnings)
	}
	if r := rowsByID(t, rep)["I046"]; r.Verdict != VerdictMatch {
		t.Errorf("I046 verdict = %s (%s), want match — claude evidence unaffected by the missing codex dir", r.Verdict, r.Detail)
	}
	if rep.Blocking() {
		t.Error("a missing codex sessions dir must never block")
	}
}

// Acceptance: leaving Options.CodexSessionsDir empty — exactly what every
// pre-I041 caller and every existing fixture test does — must not even
// attempt codex discovery: a claude-only repo audits identically to before
// I041, with zero codex-related warnings.
func TestClaudeOnlyRepoAuditUnaffectedByAbsentCodexSessionsDir(t *testing.T) {
	rep := runFixture(t, "clean")
	if len(rep.Warnings) != 0 {
		t.Errorf("a claude-only repo (no --codex-sessions) must produce no warnings, got %q", rep.Warnings)
	}
}

// Acceptance: mixed claude+codex evidence for the same ticket judges each
// token within its own flavor's table — proving the flavor tag survives the
// real codex reader end to end (not just the synthetic-mappings unit proof
// in resolve_test.go).
func TestCodexMixedClaudeAndCodexEvidenceJudgedPerFlavor(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I048": "routine"})
	tdir := t.TempDir()
	writeDispatchTranscript(t, tdir, map[string]string{"I048": "sonnet"})
	codexDir := t.TempDir()
	writeCodexFile(t, filepath.Join(codexDir, "lead.jsonl"),
		codexSessionMetaLine("root-8", "root-8", "", dir, "user", "{}"),
		codexFunctionCallLine("spawn_agent", map[string]string{
			"model":     "gpt-5.6-terra",
			"task_name": "i048 codex leg",
		}),
	)

	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir, CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I048"]
	// writeDispatchTranscript issues its dispatch on the alias "sonnet"
	// (no linked subagent transcript to supersede it, unlike
	// TestSubagentTranscriptIsTheActual) — dedupSorted sorts alphabetically.
	if got := strings.Join(r.Actuals, ","); got != "gpt-5.6-terra,sonnet" {
		t.Fatalf("I048 actuals = %q, want both flavors' evidence", got)
	}
	if r.Verdict != VerdictMatch {
		t.Errorf("I048 verdict = %s (%s), want match — both tokens resolve within their own flavor's routine tier", r.Verdict, r.Detail)
	}
}
