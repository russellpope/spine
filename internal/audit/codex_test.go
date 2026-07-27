package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

// codexSessionMetaLineWithGit builds a session_meta line carrying a
// session_meta.payload.git.commit_hash (I009: payload.git carries only
// {commit_hash, branch}, no remote URL) — the D22/I043 commit-known repo
// scoping signal. Kept as a separate helper (rather than adding a param to
// codexSessionMetaLine) so every pre-I043 call site stays byte-untouched.
func codexSessionMetaLineWithGit(id, sessionID, parent, cwd, threadSource, sourceJSON, commitHash string) string {
	return fmt.Sprintf(
		`{"type":"session_meta","payload":{"id":%q,"session_id":%q,"parent_thread_id":%q,"cwd":%q,"thread_source":%q,"model":null,"source":%s,"git":{"commit_hash":%q,"branch":"main"}}}`,
		id, sessionID, parent, cwd, threadSource, sourceJSON, commitHash)
}

// makeTestGitRepo creates a real, minimal git repository at dir with one
// commit and returns its hash — D22's commit-known probe shells to git
// against a real object store, so the fixture must be a real repo, not a
// JSON stand-in (Testing Decisions: "tiny real git repos built in test
// temp dirs — precedented, spine already shells to git").
func makeTestGitRepo(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=spine-test", "GIT_AUTHOR_EMAIL=spine-test@example.com",
			"GIT_COMMITTER_NAME=spine-test", "GIT_COMMITTER_EMAIL=spine-test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q", ".")
	// Content includes dir so two fixture repos created in the same test
	// (and possibly the same wall-clock second) don't produce byte-identical
	// commit objects — a same-hash collision would silently defeat the
	// cross-repo test this helper exists for.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("fixture: "+dir+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-q", "-m", "fixture commit")
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
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

// codexUserMessageLine builds a response_item/message line with role "user"
// — the carrier for a session's opening dispatch brief (D21, I042). Content
// mirrors the Responses-API item shape (an array of typed text parts), the
// same undocumented-but-internally-consistent convention codexFunctionCallLine
// already uses for function_call items.
func codexUserMessageLine(text string) string {
	return fmt.Sprintf(`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":%q}]}}`, text)
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
			"cmd": "herdr agent start moo-clone-worker1 --kind codex --pane wM:p2 -- -m gpt-5.6-terra",
		}),
		// This leg deliberately keeps the pre-I048 "command" key (rather than
		// the real "cmd", I009 Verified 2026-07-27) to pin the tolerated
		// fallback: a mixed-shape fixture — one call real, one call the old
		// guess — must still dispatch correctly.
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

// Acceptance: a guardian-only fixture contributes no EVIDENCE to any
// ticket — guardian threads report a synthetic model (codex-auto-review) and
// are structurally excluded (D23), so the ticket must never pick up the
// synthetic token or judge match/descent from it.
//
// Verdict updated at I044 (D24): this is the textbook guardian-only-match
// near miss the design names explicitly ("guardian-only matches" is one of
// D24's three unattributed-transcript populations) — the guardian's quoted
// spawn_agent task_name names I044 case-insensitively, so once repo-scoped
// codex material mentioning the ticket exists but fails attribution, the
// honest verdict is unattributed-transcript, not the "nothing at all"
// no-transcript reading this test asserted pre-I044. Found-but-unusable is
// not nothing-found.
//
// Review finding I1 (I041): a fixture whose only content is the
// turn_context model has zero discriminating power — the isSubagent() gate
// (no thread_spawn on a guardian) already drops that content regardless of
// isGuardian(), so mutating isGuardian() to unconditionally return false
// left this test green. Per I009, guardian threads carry quoted transcript
// history that can include a replayed spawn_agent call; a dispatch-shaped
// record is added here so the D23 exclusion is what the test actually
// exercises — gutting isGuardian() must still turn this red (it would then
// judge match/gpt-5.6-terra instead of unattributed-transcript).
func TestCodexGuardianContributesNoEvidence(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I044": "routine"})

	writeCodexFile(t, filepath.Join(codexDir, "guardian.jsonl"),
		codexSessionMetaLine("guard-4", "root-4", "root-4", sessRepo, "subagent", guardianSource),
		codexTurnContextLine("codex-auto-review"),
		// Quoted/replayed transcript history inside the guardian's own
		// review, structurally identical to a real dispatch record. If the
		// D23 exclusion did not gate the whole file, this alone would claim
		// I044 with a matching model.
		codexFunctionCallLine("spawn_agent", map[string]string{
			"model":     "gpt-5.6-terra",
			"task_name": "i044 quoted replay inside guardian review",
		}),
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I044"]
	if r.Verdict != VerdictUnattributedTranscript {
		t.Fatalf("I044 verdict = %s (%s), want unattributed-transcript — a guardian-only match (D24) is found-but-unusable, not nothing-found", r.Verdict, r.Detail)
	}
	if len(r.Actuals) != 0 {
		t.Errorf("I044 actuals = %v, want none", r.Actuals)
	}
}

// Regression (I044 fix round 1, review finding Important-1): the D22 scope
// check must run BEFORE the guardian check, not after — an out-of-scope
// guardian file must stay invisible to the audit entirely, never surface as
// a D24 near miss just because its quoted content happens to mention a
// ticket ("Sessions outside scope are invisible to attribution — they are
// not 'unattributed', they simply do not exist for this audit," I043's
// ticket text, D22). Every guardian fixture before this one used an
// in-scope cwd, so the reorder had no forcing test: reverting it left the
// full suite green. This fixture's guardian session has a cwd OUTSIDE the
// audited repo and no commit_hash — out of scope by D22 — with content that
// would otherwise be a textbook guardian-only match (I044's own
// TestCodexGuardianContributesNoEvidence fixture, restaged out of scope).
// It must stay no-transcript, not unattributed-transcript, and produce no
// warning (an out-of-scope session isn't a degradation, it's normal).
func TestCodexOutOfScopeGuardianProducesNoNearMiss(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	outsideCwd := t.TempDir() // deliberately NOT inside sessRepo, no commit_hash given
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I955": "routine"})

	writeCodexFile(t, filepath.Join(codexDir, "guardian.jsonl"),
		codexSessionMetaLine("guard-955", "root-955", "root-955", outsideCwd, "subagent", guardianSource),
		codexTurnContextLine("codex-auto-review"),
		codexFunctionCallLine("spawn_agent", map[string]string{
			"model":     "gpt-5.6-terra",
			"task_name": "i955 quoted replay inside an out-of-scope guardian review",
		}),
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I955"]
	if r.Verdict != VerdictNoTranscript {
		t.Fatalf("I955 verdict = %s (%s), want no-transcript — an out-of-scope guardian's content must never surface as a near miss", r.Verdict, r.Detail)
	}
	if len(r.Actuals) != 0 {
		t.Errorf("I955 actuals = %v, want none", r.Actuals)
	}
	if len(rep.Warnings) != 0 {
		t.Errorf("want no warnings for an out-of-scope guardian (out of scope is normal, not a degradation), got %q", rep.Warnings)
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

// Regression (review finding C1, Critical): an exec_command carrying an
// unrelated -m flag must never become dispatch evidence. Leads commit
// routinely; before the fix, ANY -m flag in ANY exec command was
// accumulated (codexModelFlagRe matched indiscriminately), so
// `git commit -m "feat: work done"` judged the ticket unmapped-dispatch
// with actuals `["feat:` — the commit message fragment became the "model".
// Only commands structurally shaped like a team spawn (herdr/cmux agent
// start|prompt) may contribute.
func TestCodexGitCommitNoiseNotMistakenForDispatch(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I903": "routine"})

	writeCodexFile(t, filepath.Join(codexDir, "lead.jsonl"),
		codexSessionMetaLine("root-903", "root-903", "", sessRepo, "user", "{}"),
		codexFunctionCallLine("exec_command", map[string]string{
			"cmd": "herdr agent start w1 --kind codex --pane wA:p1 -- -m gpt-5.6-terra",
		}),
		codexFunctionCallLine("exec_command", map[string]string{
			"cmd": `herdr agent prompt w1 "$(<.superpowers/sdd/2026-07-26-i903/dispatch-task-I903.md)"`,
		}),
		codexFunctionCallLine("exec_command", map[string]string{
			"cmd": `git commit -m "feat: work done"`,
		}),
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I903"]
	if r.Verdict != VerdictMatch {
		t.Fatalf("I903 verdict = %s (%s), want match — the commit-message -m must not poison the dispatch", r.Verdict, r.Detail)
	}
	if got := strings.Join(r.Actuals, ","); got != "gpt-5.6-terra" {
		t.Errorf(`I903 actuals = %q, want gpt-5.6-terra only (not a "feat:" fragment from the commit)`, got)
	}
}

// Regression (review finding C2, Critical): two team spawns in one lead
// session must not collapse into a single last-wins dispatch record. Before
// the fix, the accumulator was session-scoped (not per worker), so a
// second, unrelated spawn's -m flag silently overwrote the first — two
// correctly-tiered dispatches (w1 terra -> I903 routine, w2 luna -> I904
// mechanical) judged I903 silent-descent (BLOCKING) with actuals
// [gpt-5.6-luna]: a manufactured false blocking verdict from a fully
// correct build, exactly the confident-wrong-answer class the design
// forbids. I009's verified example shows the worker name in both the start
// and prompt commands — the accumulator now keys on it.
func TestCodexTwoTeamSpawnsKeyedPerWorker(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I903": "routine", "I904": "mechanical"})

	writeCodexFile(t, filepath.Join(codexDir, "lead.jsonl"),
		codexSessionMetaLine("root-904", "root-904", "", sessRepo, "user", "{}"),
		codexFunctionCallLine("exec_command", map[string]string{
			"cmd": "herdr agent start w1 --kind codex --pane wA:p1 -- -m gpt-5.6-terra",
		}),
		codexFunctionCallLine("exec_command", map[string]string{
			"cmd": `herdr agent prompt w1 "$(<.superpowers/sdd/2026-07-26-i903/dispatch-task-I903.md)"`,
		}),
		codexFunctionCallLine("exec_command", map[string]string{
			"cmd": "herdr agent start w2 --kind codex --pane wB:p1 -- -m gpt-5.6-luna",
		}),
		codexFunctionCallLine("exec_command", map[string]string{
			"cmd": `herdr agent prompt w2 "$(<.superpowers/sdd/2026-07-26-i904/dispatch-task-I904.md)"`,
		}),
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)
	if r := rows["I903"]; r.Verdict != VerdictMatch || strings.Join(r.Actuals, ",") != "gpt-5.6-terra" {
		t.Errorf("I903 = %s actuals=%v, want match/gpt-5.6-terra — w2's luna must not bleed into w1's ticket", r.Verdict, r.Actuals)
	}
	if r := rows["I904"]; r.Verdict != VerdictMatch || strings.Join(r.Actuals, ",") != "gpt-5.6-luna" {
		t.Errorf("I904 = %s actuals=%v, want match/gpt-5.6-luna — w1's terra must not bleed into w2's ticket", r.Verdict, r.Actuals)
	}
	if rep.Blocking() {
		t.Error("two correctly-tiered spawns must never manufacture a blocking verdict")
	}
}

// Acceptance (Testing Decisions' CLI clause; review finding I3):
// DefaultCodexSessionsDir's default-derivation branches — $CODEX_HOME set,
// and the ~/.codex/sessions fallback when it is not.
func TestDefaultCodexSessionsDir(t *testing.T) {
	t.Run("CODEX_HOME set", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("CODEX_HOME", home)
		got, err := DefaultCodexSessionsDir()
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(home, "sessions"); got != want {
			t.Errorf("DefaultCodexSessionsDir() = %q, want %q", got, want)
		}
	})
	t.Run("CODEX_HOME unset falls back to home dir", func(t *testing.T) {
		t.Setenv("CODEX_HOME", "")
		got, err := DefaultCodexSessionsDir()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(got, filepath.Join(".codex", "sessions")) {
			t.Errorf("DefaultCodexSessionsDir() = %q, want suffix .codex/sessions", got)
		}
	})
}

// Acceptance: a missing/unreadable codex sessions dir degrades to a
// warning, never an error, and never changes verdicts driven by claude-side
// evidence.
func TestCodexMissingSessionsDirDegradesToWarning(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I046": "primary"})
	tdir := t.TempDir()
	writeDispatchTranscript(t, dir, tdir, map[string]string{"I046": "fable"})

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
	writeDispatchTranscript(t, dir, tdir, map[string]string{"I048": "sonnet"})
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

// --- D21 worker-session scan (I042) ---

// Acceptance: a worker fixture — opening user message carries the ticket
// token (the dispatch brief), no dispatch records of its own — attributes
// its per-turn models as ticket evidence. This is the case I041 deliberately
// left inert (AC1's ratified inert-worker note); D21 turns it on.
func TestCodexWorkerOpeningMessageAttributesOwnTurns(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I900": "routine"})

	writeCodexFile(t, filepath.Join(codexDir, "worker.jsonl"),
		codexSessionMetaLine("worker-900", "worker-900", "", sessRepo, "user", "{}"),
		codexUserMessageLine("# Task I900 — implementer dispatch\n\nBuild the thing."),
		codexTurnContextLine("gpt-5.6-terra"),
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I900"]
	if r.Verdict != VerdictMatch {
		t.Fatalf("I900 verdict = %s (%s), want match — the opening message names I900 and the session dispatches nothing of its own", r.Verdict, r.Detail)
	}
	if got := strings.Join(r.Actuals, ","); got != "gpt-5.6-terra" {
		t.Errorf("I900 actuals = %q, want gpt-5.6-terra", got)
	}
}

// Acceptance (D21 clause 2): a ticket token appearing only in a LATER user
// message — never the opening one — must not attribute. This is the
// neighboring-ticket bleed the design calls out: opening message dispatches
// I900; a later message mentions I901 in passing. I900 must still attribute
// correctly from the true opening message.
//
// Verdict updated at I044 (D24, pre-justified at I042 review): I901's
// mid-transcript-only mention is exactly the "token absent from the opening
// message" near miss D24 names — found (a real session mentions it) but not
// attributable (not the opening line), so the honest verdict is
// unattributed-transcript, not no-transcript. Recorded at I042 review
// (0bd554a) as a genuine premise change this ticket exists to make.
func TestCodexLaterMessageTokenDoesNotAttribute(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I900": "routine", "I901": "routine"})

	writeCodexFile(t, filepath.Join(codexDir, "worker.jsonl"),
		codexSessionMetaLine("worker-901", "worker-901", "", sessRepo, "user", "{}"),
		codexUserMessageLine("# Task I900 — implementer dispatch\n\nBuild the thing."),
		codexTurnContextLine("gpt-5.6-terra"),
		codexUserMessageLine("while you're at it, take a look at I901 too"),
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)
	if r := rows["I900"]; r.Verdict != VerdictMatch {
		t.Errorf("I900 verdict = %s (%s), want match from the true opening message", r.Verdict, r.Detail)
	}
	if r := rows["I901"]; r.Verdict != VerdictUnattributedTranscript {
		t.Errorf("I901 verdict = %s (%s), want unattributed-transcript — a later-message mention must not attribute (neighboring-ticket bleed), but D24 reports it honestly rather than as no-transcript", r.Verdict, r.Detail)
	}
}

// Acceptance (D21 amended, I009 Verified 2026-07-27 live acceptance): the
// literal first role="user" message in a real codex session is a
// harness-injected preamble, not the operator's brief — an "# AGENTS.md
// instructions" block, then a "<recommended_plugins>" block, THEN the real
// dispatch brief. Both injected-shaped messages must be skipped for the
// opening-message latch; the brief (the first non-injected user message)
// is what attributes.
func TestCodexInjectedPreamblesSkippedForOpeningMessage(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I909": "routine"})

	writeCodexFile(t, filepath.Join(codexDir, "worker.jsonl"),
		codexSessionMetaLine("worker-909", "worker-909", "", sessRepo, "user", "{}"),
		codexUserMessageLine("# AGENTS.md instructions for /Users/x/project\n\nFollow repo conventions."),
		codexUserMessageLine("<recommended_plugins>\nsome-plugin\n</recommended_plugins>"),
		codexUserMessageLine("# Task I909 — implementer dispatch\n\nBuild the thing."),
		codexTurnContextLine("gpt-5.6-terra"),
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I909"]
	if r.Verdict != VerdictMatch {
		t.Fatalf("I909 verdict = %s (%s), want match — the injected AGENTS.md/recommended_plugins preambles must be skipped, leaving the real brief as the opening message", r.Verdict, r.Detail)
	}
	if got := strings.Join(r.Actuals, ","); got != "gpt-5.6-terra" {
		t.Errorf("I909 actuals = %q, want gpt-5.6-terra", got)
	}
}

// Acceptance (D21 amended, I009 Verified 2026-07-27): if EVERY role="user"
// message in a file is injected-shaped (no real brief ever arrives), the
// opening-message latch never fires — the existing "no opening" degrade
// applies: this session contributes nothing, not a mistaken attribution to
// injected preamble text.
func TestCodexAllInjectedMessagesContributeNoOpening(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I910": "routine"})

	writeCodexFile(t, filepath.Join(codexDir, "worker.jsonl"),
		codexSessionMetaLine("worker-910", "worker-910", "", sessRepo, "user", "{}"),
		codexUserMessageLine("# AGENTS.md instructions for /Users/x/project\n\nFollow repo conventions."),
		codexUserMessageLine("<recommended_plugins>\nsome-plugin\n</recommended_plugins>"),
		codexTurnContextLine("gpt-5.6-terra"),
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I910"]
	if r.Verdict != VerdictNoTranscript {
		t.Fatalf("I910 verdict = %s (%s), want no-transcript — an all-injected-preamble session names no ticket anywhere and must contribute nothing", r.Verdict, r.Detail)
	}
	if len(r.Actuals) != 0 {
		t.Errorf("I910 actuals = %v, want none — injected-preamble turn models must never attribute", r.Actuals)
	}
}

// Acceptance (D21 clause 3): an orchestrator fixture whose opening message
// names the ticket but which itself carries a dispatch record (spawn_agent,
// unrelated task) contributes no own-turn evidence. Any session that
// dispatches is an orchestrator; its own models are never ticket evidence —
// proof the orchestrator's own turn_context model (a decoy, wrong-looking on
// purpose) never leaked in as I902's actual.
//
// Verdict updated at I044 (D24): this is exactly the "orchestrator-only
// mentions" near miss D24 names by name — the opening message's title line
// names I902 in a session that turns out to be an orchestrator, so the
// mention is found but structurally unusable, not simply absent. The honest
// verdict is unattributed-transcript, not no-transcript; the point this test
// exists to prove (own-turn evidence never leaks in) is unchanged.
func TestCodexOrchestratorOpeningMessageContributesNoOwnTurnEvidence(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I902": "routine"})

	writeCodexFile(t, filepath.Join(codexDir, "lead.jsonl"),
		codexSessionMetaLine("lead-902", "lead-902", "", sessRepo, "user", "{}"),
		codexUserMessageLine("# Task I902 — orchestrator dispatch\n\nCoordinate the build."),
		codexTurnContextLine("gpt-5.6-sol"), // decoy: must never surface as I902 evidence
		codexFunctionCallLine("spawn_agent", map[string]string{
			"model":     "gpt-5.6-terra",
			"task_name": "unrelated review pass",
		}),
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I902"]
	if r.Verdict != VerdictUnattributedTranscript {
		t.Fatalf("I902 verdict = %s (%s), want unattributed-transcript — the orchestrator's own turn model must not attribute, but D24 reports the opening-message mention honestly", r.Verdict, r.Detail)
	}
	if len(r.Actuals) != 0 {
		t.Errorf("I902 actuals = %v, want none (the decoy gpt-5.6-sol own-turn model must not leak in)", r.Actuals)
	}
}

// Acceptance (D21 clause 3, composition): a worker-that-spawns fixture —
// opening message names the ticket, own turn_context carries a decoy model,
// AND it spawns its own subagent (e.g. a spec-review pass) whose dispatch
// record also names the ticket — keeps ticket evidence via the dispatch
// record while its own turns are excluded. Proves the rules compose without
// loss, per D21's validation note.
func TestCodexWorkerThatSpawnsKeepsDispatchRecordEvidenceOnly(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I903": "routine"})

	writeCodexFile(t, filepath.Join(codexDir, "worker.jsonl"),
		codexSessionMetaLine("worker-903", "worker-903", "", sessRepo, "user", "{}"),
		codexUserMessageLine("# Task I903 — implementer dispatch\n\nBuild the thing, then spawn review."),
		codexTurnContextLine("gpt-5.6-sol"), // decoy: must never surface (own-turn, orchestrator now)
		codexFunctionCallLine("spawn_agent", map[string]string{
			"model":     "gpt-5.6-terra",
			"task_name": "i903 spec-review pass",
		}),
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I903"]
	if r.Verdict != VerdictMatch {
		t.Fatalf("I903 verdict = %s (%s), want match — the spawn_agent dispatch record still carries evidence", r.Verdict, r.Detail)
	}
	if got := strings.Join(r.Actuals, ","); got != "gpt-5.6-terra" {
		t.Errorf("I903 actuals = %q, want gpt-5.6-terra only — the own-turn decoy gpt-5.6-sol must be excluded", got)
	}
}

// Acceptance: an M4a-shaped fixture — a worker attributed by D21 whose
// declared model was never a shipped id (gpt-5.5, the design doc's own
// example of a pre-explicit-model-dispatch build's honest history) — judges
// unmapped-dispatch, not match (it isn't) and not silence (no-transcript
// would hide real, repo-scoped, attributed evidence).
func TestCodexM4aUndeclaredModelJudgesUnmappedDispatch(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I904": "routine"})

	writeCodexFile(t, filepath.Join(codexDir, "worker.jsonl"),
		codexSessionMetaLine("worker-904", "worker-904", "", sessRepo, "user", "{}"),
		codexUserMessageLine("# Task I904 — implementer dispatch\n\nBuild the thing."),
		codexTurnContextLine("gpt-5.5"),
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I904"]
	if r.Verdict != VerdictUnmappedDispatch {
		t.Fatalf("I904 verdict = %s (%s), want unmapped-dispatch — gpt-5.5 was never a shipped id", r.Verdict, r.Detail)
	}
}

// --- I042 review fix round 1 ---

// Regression (review finding C1, Critical; probe P1). A model-less
// dispatcher — a team-spawn "start" command with no explicit -m flag,
// definitionally the pre-I038 M4a class this ticket exists to make
// auditable — is still a spawn-SHAPED record and must still trigger the
// orchestrator exclusion (D21 amended at review, 0bd554a): "contains
// dispatch records" means ANY spawn-shaped record, with or without a
// usable model, not just ones that survived the explicit-model evidence
// gate. Before the fix, len(res.dispatches)==0 saw zero dispatch-record
// *evidence* (the model-less spawn produces none) and treated this lead as
// a worker: its own decoy turn model attributed and manufactured a
// blocking silent-descent (probe P1). After the fix, the lead contributes
// nothing at all — no own-turn evidence, no dispatch evidence (there was
// none to give) — and the ticket lands honestly, never blocking.
//
// Verdict updated at I044 (D24): the lead's opening message names I905 in
// its title line, and the lead turns out to be an orchestrator — another
// instance of the "orchestrator-only mentions" near miss D24 names. The
// regression's real point (own-turn evidence never leaks in, never blocks)
// is unaffected; only the honest label for "found but excluded" changes
// from no-transcript to unattributed-transcript.
func TestCodexModelLessSpawnStillExcludesOwnTurns(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I905": "routine"})

	writeCodexFile(t, filepath.Join(codexDir, "lead.jsonl"),
		codexSessionMetaLine("lead-905", "lead-905", "", sessRepo, "user", "{}"),
		codexUserMessageLine("# Task I905 — orchestrator dispatch\n\nCoordinate the build."),
		codexTurnContextLine("gpt-5.6-luna"), // decoy: must never surface as I905 evidence
		codexFunctionCallLine("exec_command", map[string]string{
			// no -m flag: the M4a-class spawn shape that carries no explicit
			// model. codexTeamSpawnStartRe (evidence path) will not match
			// this — it must still mark the session as dispatched.
			"cmd": "herdr agent start w1 --kind codex --pane wA:p1 --",
		}),
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I905"]
	if r.Verdict != VerdictUnattributedTranscript {
		t.Fatalf("I905 verdict = %s (%s), want unattributed-transcript — a model-less spawn is still an orchestrator; its own turn must not attribute, but D24 reports the opening-message mention honestly", r.Verdict, r.Detail)
	}
	if len(r.Actuals) != 0 {
		t.Errorf("I905 actuals = %v, want none (the decoy gpt-5.6-luna own-turn model must not leak in)", r.Actuals)
	}
	if rep.Blocking() {
		t.Error("a model-less dispatcher's own decoy turn must never manufacture a blocking verdict")
	}
}

// Regression (review finding C2, Critical; probe P2). Token matching is
// against the FIRST LINE of the opening user message only (D21 amended at
// review, 0bd554a) — a context sentence naming a neighboring, higher-tier
// ticket must not attribute to it. Before the fix, the whole opening
// message was folded into the worker's description and matched per
// ticket, so a brief titled for I906 that merely mentions "this stacks on
// I907's reader work" gave I907 (primary) a terra actual and manufactured
// a blocking silent-descent from a fully correct build (probe P2). After
// the fix: I906 (named in the title line) attributes; I907 (named only in
// a later line of the SAME opening message) gets no evidence from this
// session at all, and nothing blocks.
//
// Verdict updated at I044 (D24): I907's context-sentence mention is exactly
// the "token absent from the opening message['s first line]" near miss D24
// names — found (this same session's fuller text names it) but not
// attributable, so the honest verdict is unattributed-transcript, not
// no-transcript. The regression's real point (no evidence, no blocking) is
// unaffected.
func TestCodexOpeningMessageContextSentenceDoesNotAttributeToNeighbor(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I906": "routine", "I907": "primary"})

	writeCodexFile(t, filepath.Join(codexDir, "worker.jsonl"),
		codexSessionMetaLine("worker-906", "worker-906", "", sessRepo, "user", "{}"),
		codexUserMessageLine("# Task I906 — implementer dispatch\n\nContext: this stacks on I907's reader work."),
		codexTurnContextLine("gpt-5.6-terra"),
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)
	if r := rows["I906"]; r.Verdict != VerdictMatch {
		t.Errorf("I906 verdict = %s (%s), want match — its token is in the opening message's title line", r.Verdict, r.Detail)
	}
	if r := rows["I907"]; r.Verdict != VerdictUnattributedTranscript || len(r.Actuals) != 0 {
		t.Errorf("I907 verdict = %s actuals=%v, want unattributed-transcript/none — a context-sentence mention (not the title line) must not attribute, but D24 reports it honestly", r.Verdict, r.Actuals)
	}
	if rep.Blocking() {
		t.Error("a brief's context-sentence mention of a higher-tier neighbor must never manufacture a blocking verdict")
	}
}

// Regression (review finding I1, Important). The audit.go codex case-fold
// (Run's agent-correlation loop, ToUpper for a.flavor=="codex") was
// previously untested — every prior fixture's opening-message title
// happened to carry the ticket token uppercase already, so deleting the
// fold left the suite green. A lowercase title line (plausible per D20's
// own "lowercase by convention" note for task_name) must still attribute —
// this is the Run-boundary, flavor-scoped seam the I040 Testing Decisions
// clause explicitly permits testing.
func TestCodexLowercaseOpeningMessageTitleAttributesCaseInsensitively(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I908": "routine"})

	writeCodexFile(t, filepath.Join(codexDir, "worker.jsonl"),
		codexSessionMetaLine("worker-908", "worker-908", "", sessRepo, "user", "{}"),
		codexUserMessageLine("task i908 — implementer dispatch\n\nBuild the thing."),
		codexTurnContextLine("gpt-5.6-terra"),
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I908"]
	if r.Verdict != VerdictMatch {
		t.Fatalf("I908 verdict = %s (%s), want match — codex ticket-token matching is case-insensitive (D20)", r.Verdict, r.Detail)
	}
	if got := strings.Join(r.Actuals, ","); got != "gpt-5.6-terra" {
		t.Errorf("I908 actuals = %q, want gpt-5.6-terra", got)
	}
}

// --- D22 repo scoping — cwd or known commit (I043) ---

// Acceptance: a session whose cwd resolves inside the audited repo attributes
// normally; a sibling repo's session carrying the SAME ticket token (I024
// restarts per repo across the estate, per D22's rationale) is out of scope
// and contributes nothing — cwd-only scoping's existing guarantee, made an
// explicit D22 fixture.
func TestCodexSiblingRepoCwdSameTicketTokenOutOfScope(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	siblingRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I910": "routine"})

	writeCodexFile(t, filepath.Join(codexDir, "in-scope.jsonl"),
		codexSessionMetaLine("in-910", "in-910", "", sessRepo, "user", "{}"),
		codexUserMessageLine("# Task I910 — implementer dispatch\n\nBuild the thing."),
		codexTurnContextLine("gpt-5.6-terra"),
	)
	writeCodexFile(t, filepath.Join(codexDir, "sibling.jsonl"),
		codexSessionMetaLine("sib-910", "sib-910", "", siblingRepo, "user", "{}"),
		codexUserMessageLine("# Task I910 — implementer dispatch\n\nBuild the thing (sibling repo's own I910)."),
		codexTurnContextLine("gpt-5.6-luna"), // decoy: sibling repo, must never surface here
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I910"]
	if r.Verdict != VerdictMatch {
		t.Fatalf("I910 verdict = %s (%s), want match", r.Verdict, r.Detail)
	}
	if got := strings.Join(r.Actuals, ","); got != "gpt-5.6-terra" {
		t.Errorf("I910 actuals = %q, want gpt-5.6-terra only — the sibling repo's session must not attribute", got)
	}
}

// Acceptance (D22 clause 2, the ticket's headline scenario): a worktree
// fixture — cwd OUTSIDE the audited repo (a /private/tmp team dir stand-in),
// but session_meta.payload.git.commit_hash names a commit that IS in the
// audited repo's real git history — is in scope. This is what makes
// worktree-cwd codex teams visible at all.
func TestCodexWorktreeCwdKnownCommitInScope(t *testing.T) {
	codexDir := t.TempDir()
	repoDir := t.TempDir()
	commitHash := makeTestGitRepo(t, repoDir)
	writeAuditRepo(t, repoDir, gen9DefaultWorkflow, map[string]string{"I911": "routine"})
	worktreeCwd := t.TempDir() // deliberately NOT inside repoDir

	writeCodexFile(t, filepath.Join(codexDir, "worker.jsonl"),
		codexSessionMetaLineWithGit("worker-911", "worker-911", "", worktreeCwd, "user", "{}", commitHash),
		codexUserMessageLine("# Task I911 — implementer dispatch\n\nBuild the thing."),
		codexTurnContextLine("gpt-5.6-terra"),
	)

	rep, err := Run(Options{RepoDir: repoDir, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I911"]
	if r.Verdict != VerdictMatch {
		t.Fatalf("I911 verdict = %s (%s), want match — a commit known to the repo puts the worktree cwd in scope", r.Verdict, r.Detail)
	}
	if got := strings.Join(r.Actuals, ","); got != "gpt-5.6-terra" {
		t.Errorf("I911 actuals = %q, want gpt-5.6-terra", got)
	}
}

// Acceptance (D22 clause 2, negative): cwd outside the repo AND a
// well-formed but unknown commit hash — neither scoping signal fires, so the
// session is invisible to this audit (not "unattributed": D22 says out-of-
// scope sessions do not exist for the audit) and the ticket stays
// no-transcript.
func TestCodexUnknownCommitOutsideCwdOutOfScope(t *testing.T) {
	codexDir := t.TempDir()
	repoDir := t.TempDir()
	makeTestGitRepo(t, repoDir) // a real repo, but the hash below is not one of its objects
	writeAuditRepo(t, repoDir, gen9DefaultWorkflow, map[string]string{"I912": "routine"})
	worktreeCwd := t.TempDir()

	writeCodexFile(t, filepath.Join(codexDir, "worker.jsonl"),
		codexSessionMetaLineWithGit("worker-912", "worker-912", "", worktreeCwd, "user", "{}",
			"deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
		codexUserMessageLine("# Task I912 — implementer dispatch\n\nBuild the thing."),
		codexTurnContextLine("gpt-5.6-sol"), // decoy: out of scope, must never surface
	)

	rep, err := Run(Options{RepoDir: repoDir, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I912"]
	if r.Verdict != VerdictNoTranscript {
		t.Fatalf("I912 verdict = %s (%s), want no-transcript — an unknown commit hash and an outside cwd must not admit the session", r.Verdict, r.Detail)
	}
	if len(r.Actuals) != 0 {
		t.Errorf("I912 actuals = %v, want none", r.Actuals)
	}
}

// Acceptance (D22 clause 3, degrade-never-fail): when the audited repo dir
// is not a git repository — a hermetic stand-in for "git probe failure"
// (the same failure mode covers a missing git binary, since both make the
// probe mechanism itself unusable) — commit-hash scoping degrades to
// cwd-only, with a report warning naming the degradation, never an error.
// A same-run cwd-inside-repo session proves cwd-only scoping keeps working
// through the degradation, not that scoping breaks wholesale.
func TestCodexGitProbeFailureDegradesToCwdOnlyWithWarning(t *testing.T) {
	codexDir := t.TempDir()
	repoDir := t.TempDir() // deliberately never git-initialized
	writeAuditRepo(t, repoDir, gen9DefaultWorkflow, map[string]string{"I913": "routine", "I914": "routine"})

	writeCodexFile(t, filepath.Join(codexDir, "cwd-worker.jsonl"),
		codexSessionMetaLine("cwd-913", "cwd-913", "", repoDir, "user", "{}"),
		codexUserMessageLine("# Task I913 — implementer dispatch\n\nBuild the thing."),
		codexTurnContextLine("gpt-5.6-terra"),
	)
	writeCodexFile(t, filepath.Join(codexDir, "worktree-worker.jsonl"),
		codexSessionMetaLineWithGit("wt-914", "wt-914", "", t.TempDir(), "user", "{}",
			"0123456789abcdef0123456789abcdef01234567"),
		codexUserMessageLine("# Task I914 — implementer dispatch\n\nBuild the thing."),
		codexTurnContextLine("gpt-5.6-luna"), // decoy: the probe can't run, must never surface
	)

	rep, err := Run(Options{RepoDir: repoDir, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)
	if r := rows["I913"]; r.Verdict != VerdictMatch {
		t.Errorf("I913 verdict = %s (%s), want match — cwd-only scoping must keep working when the git probe degrades", r.Verdict, r.Detail)
	}
	if r := rows["I914"]; r.Verdict != VerdictNoTranscript {
		t.Errorf("I914 verdict = %s (%s), want no-transcript — a degraded git probe must not admit a worktree-cwd session", r.Verdict, r.Detail)
	}
	found := false
	for _, w := range rep.Warnings {
		if strings.Contains(w, "git") {
			found = true
		}
	}
	if !found {
		t.Errorf("want a warning naming the git-probe degradation, got %q", rep.Warnings)
	}
	if rep.Blocking() {
		t.Error("a degraded git probe must never manufacture a blocking verdict")
	}
}

// Acceptance (D22, the ticket's cross-repo guarantee): two distinct real git
// repos each carry a worktree-cwd session for the SAME ticket id (I024
// restarts per repo across the estate). Each repo's audit must see only its
// own session's evidence — a shared codex sessions dir must never let the
// commit-hash probe cross repo boundaries.
func TestCodexCrossRepoCollisionSameTicketEachAuditsOwnEvidence(t *testing.T) {
	codexDir := t.TempDir()
	repoA := t.TempDir()
	hashA := makeTestGitRepo(t, repoA)
	writeAuditRepo(t, repoA, gen9DefaultWorkflow, map[string]string{"I024": "routine"})
	repoB := t.TempDir()
	hashB := makeTestGitRepo(t, repoB)
	writeAuditRepo(t, repoB, gen9DefaultWorkflow, map[string]string{"I024": "routine"})

	writeCodexFile(t, filepath.Join(codexDir, "worker-a.jsonl"),
		codexSessionMetaLineWithGit("wa-024", "wa-024", "", t.TempDir(), "user", "{}", hashA),
		codexUserMessageLine("# Task I024 — implementer dispatch\n\nBuild repo A's thing."),
		codexTurnContextLine("gpt-5.6-terra"),
	)
	writeCodexFile(t, filepath.Join(codexDir, "worker-b.jsonl"),
		codexSessionMetaLineWithGit("wb-024", "wb-024", "", t.TempDir(), "user", "{}", hashB),
		codexUserMessageLine("# Task I024 — implementer dispatch\n\nBuild repo B's thing."),
		codexTurnContextLine("gpt-5.6-luna"),
	)

	repA, err := Run(Options{RepoDir: repoA, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	repB, err := Run(Options{RepoDir: repoB, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}

	rA := rowsByID(t, repA)["I024"]
	if got := strings.Join(rA.Actuals, ","); got != "gpt-5.6-terra" {
		t.Errorf("repo A I024 actuals = %q, want gpt-5.6-terra only — repo B's session must not leak in", got)
	}
	rB := rowsByID(t, repB)["I024"]
	if got := strings.Join(rB.Actuals, ","); got != "gpt-5.6-luna" {
		t.Errorf("repo B I024 actuals = %q, want gpt-5.6-luna only — repo A's session must not leak in", got)
	}
}

// Regression (I043 review finding I1, Important). `git cat-file -e
// <rev>^{commit}` resolves ANY valid revision expression, not just a raw
// object id — confirmed directly: in a repo with a `main` branch, both
// "main^{commit}" and "HEAD^{commit}" exit 0. Before the fix, a ref-ish
// commit_hash value (branch name, HEAD, a future format's drifted field)
// would false-positive knows() into nearly every repo that happens to have
// a same-named ref — exactly the cross-repo false-positive class D22 exists
// to prevent, worse than a missed session by the design's own ranking. Both
// sessions here have cwd outside the repo and a ref-like (not SHA-like)
// commit_hash; both must stay out of scope, same as an empty/unknown hash.
func TestCodexRefLikeCommitHashNotTreatedAsObjectID(t *testing.T) {
	codexDir := t.TempDir()
	repoDir := t.TempDir()
	makeTestGitRepo(t, repoDir)
	writeAuditRepo(t, repoDir, gen9DefaultWorkflow, map[string]string{"I915": "routine", "I916": "routine"})
	// Guarantee a ref literally named "main" exists at HEAD regardless of
	// this host's git init.defaultBranch setting (checkout -B creates or
	// resets it, so it succeeds whether or not "main" already exists).
	if out, err := exec.Command("git", "-C", repoDir, "checkout", "-q", "-B", "main").CombinedOutput(); err != nil {
		t.Fatalf("git checkout -B main: %v\n%s", err, out)
	}
	worktreeCwd := t.TempDir()

	writeCodexFile(t, filepath.Join(codexDir, "main-ref.jsonl"),
		codexSessionMetaLineWithGit("worker-915", "worker-915", "", worktreeCwd, "user", "{}", "main"),
		codexUserMessageLine("# Task I915 — implementer dispatch\n\nBuild the thing."),
		codexTurnContextLine("gpt-5.6-sol"), // decoy: must never surface — "main" is a ref, not an object id
	)
	writeCodexFile(t, filepath.Join(codexDir, "head-ref.jsonl"),
		codexSessionMetaLineWithGit("worker-916", "worker-916", "", worktreeCwd, "user", "{}", "HEAD"),
		codexUserMessageLine("# Task I916 — implementer dispatch\n\nBuild the thing."),
		codexTurnContextLine("gpt-5.6-luna"), // decoy: must never surface — "HEAD" is a ref, not an object id
	)

	rep, err := Run(Options{RepoDir: repoDir, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)
	if r := rows["I915"]; r.Verdict != VerdictNoTranscript || len(r.Actuals) != 0 {
		t.Errorf(`I915 verdict = %s actuals=%v, want no-transcript/none — "main" must not resolve via cat-file`, r.Verdict, r.Actuals)
	}
	if r := rows["I916"]; r.Verdict != VerdictNoTranscript || len(r.Actuals) != 0 {
		t.Errorf(`I916 verdict = %s actuals=%v, want no-transcript/none — "HEAD" must not resolve via cat-file`, r.Verdict, r.Actuals)
	}
	if rep.Blocking() {
		t.Error("a ref-like commit_hash value must never manufacture a blocking verdict via cross-repo false-positive")
	}
}

// --- D24 unattributed-transcript verdict + source-file naming (I044) ---

// Acceptance (D24 AC): a ticket with truly ZERO scoped codex material —
// never mentioned anywhere in the sessions dir, not even a near miss — must
// still yield no-transcript. Proves the D24 near-miss override in Run only
// fires when repo-scoped material actually named the ticket; a codex
// sessions dir being configured at all must not, by itself, change a
// genuinely-nothing-found verdict.
func TestCodexZeroScopedMaterialStaysNoTranscript(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I960": "routine", "I961": "routine"})

	// I960 is attributed normally; I961 is never named anywhere in the
	// codex sessions dir — not the opening line, not later text, not a
	// guardian, not an orchestrator mention. Nothing at all.
	writeCodexFile(t, filepath.Join(codexDir, "worker.jsonl"),
		codexSessionMetaLine("worker-960", "worker-960", "", sessRepo, "user", "{}"),
		codexUserMessageLine("# Task I960 — implementer dispatch\n\nBuild the thing."),
		codexTurnContextLine("gpt-5.6-terra"),
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)
	if r := rows["I960"]; r.Verdict != VerdictMatch {
		t.Errorf("I960 verdict = %s (%s), want match", r.Verdict, r.Detail)
	}
	if r := rows["I961"]; r.Verdict != VerdictNoTranscript {
		t.Errorf("I961 verdict = %s (%s), want no-transcript — zero scoped material anywhere, not unattributed-transcript", r.Verdict, r.Detail)
	}
}

// Acceptance (D24 AC/I044): every JUDGED codex verdict — match, descent,
// escalation (with and without a ledger reason), and unmapped — names its
// source transcript file in the detail line, the I008 silent-descent
// requirement (name the source) satisfied here as a special case of the
// broader D24 rule. Each ticket's dispatch lives in its own lead file (a
// distinct codex session root) so the D24 coarse-linkage disclosure — a
// separate feature, tested below — never fires here.
func TestCodexJudgedVerdictsNameSourceFile(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{
		"I950": "routine", "I951": "mechanical", "I952": "primary", "I953": "routine", "I954": "mechanical",
	})
	if err := os.MkdirAll(filepath.Join(sessRepo, ".superpowers", "sdd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sessRepo, ".superpowers", "sdd", "progress.md"),
		[]byte("ESCALATION I954 mechanical->routine reason: deliberate up-tier for reader work\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	matchFile := filepath.Join(codexDir, "i950.jsonl")
	writeCodexFile(t, matchFile,
		codexSessionMetaLine("root-950", "root-950", "", sessRepo, "user", "{}"),
		codexFunctionCallLine("spawn_agent", map[string]string{"model": "gpt-5.6-terra", "task_name": "i950 match dispatch"}),
	)
	escNoReasonFile := filepath.Join(codexDir, "i951.jsonl")
	writeCodexFile(t, escNoReasonFile,
		codexSessionMetaLine("root-951", "root-951", "", sessRepo, "user", "{}"),
		codexFunctionCallLine("spawn_agent", map[string]string{"model": "gpt-5.6-terra", "task_name": "i951 escalate dispatch"}),
	)
	descentFile := filepath.Join(codexDir, "i952.jsonl")
	writeCodexFile(t, descentFile,
		codexSessionMetaLine("root-952", "root-952", "", sessRepo, "user", "{}"),
		codexFunctionCallLine("spawn_agent", map[string]string{"model": "gpt-5.6-luna", "task_name": "i952 descent dispatch"}),
	)
	unmappedFile := filepath.Join(codexDir, "i953.jsonl")
	writeCodexFile(t, unmappedFile,
		codexSessionMetaLine("root-953", "root-953", "", sessRepo, "user", "{}"),
		codexFunctionCallLine("spawn_agent", map[string]string{"model": "totally-unknown-model", "task_name": "i953 unmapped dispatch"}),
	)
	escWithReasonFile := filepath.Join(codexDir, "i954.jsonl")
	writeCodexFile(t, escWithReasonFile,
		codexSessionMetaLine("root-954", "root-954", "", sessRepo, "user", "{}"),
		codexFunctionCallLine("spawn_agent", map[string]string{"model": "gpt-5.6-terra", "task_name": "i954 escalated with reason dispatch"}),
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)

	if r := rows["I950"]; r.Verdict != VerdictMatch || !strings.Contains(r.Detail, "source: "+matchFile) {
		t.Errorf("I950 = %s (%q), want match naming source %s", r.Verdict, r.Detail, matchFile)
	}
	if r := rows["I951"]; r.Verdict != VerdictEscalatedNoReason || !strings.Contains(r.Detail, "source: "+escNoReasonFile) {
		t.Errorf("I951 = %s (%q), want escalated-no-reason naming source %s", r.Verdict, r.Detail, escNoReasonFile)
	}
	if r := rows["I952"]; r.Verdict != VerdictSilentDescent || !strings.Contains(r.Detail, "source: "+descentFile) {
		t.Errorf("I952 = %s (%q), want silent-descent naming source %s", r.Verdict, r.Detail, descentFile)
	}
	if r := rows["I953"]; r.Verdict != VerdictUnmappedDispatch || !strings.Contains(r.Detail, "source: "+unmappedFile) {
		t.Errorf("I953 = %s (%q), want unmapped-dispatch naming source %s", r.Verdict, r.Detail, unmappedFile)
	}
	if r := rows["I954"]; r.Verdict != VerdictEscalatedWithReason || !strings.Contains(r.Detail, "source: "+escWithReasonFile) || !strings.Contains(r.Detail, "deliberate up-tier") {
		t.Errorf("I954 = %s (%q), want escalated-with-reason naming source %s and carrying the ledger reason", r.Verdict, r.Detail, escWithReasonFile)
	}
	if !rep.Blocking() {
		t.Error("I952's silent-descent must still block — the new source-naming must not soften an existing blocking verdict")
	}
}

// Acceptance (I041-review-referred-Q3, ticket I044's coarse-linkage
// disclosure note): thread_spawn actuals link by ROOT session id only (D20
// clause 2) — that granularity is all I009's facts support. When a single
// root dispatches two DISTINCT tickets and a linked subagent's actual
// supersedes both dispatches' declared aliases, the shared merged actual is
// coarse — it cannot be proven to belong to one dispatch over the other —
// so both tickets' details must disclose the coarse linkage, naming the
// other ticket sharing the root and the root's source file.
//
// I972 (mechanical) and I973 (primary) are deliberately chosen so the SAME
// merged actual (gpt-5.6-terra, routine) drives a different verdict for
// each — above I972's declared tier, below I973's — making the disclosure
// something an operator would actually need: without it, two surprising,
// unrelated-looking verdicts; with it, a one-line pointer to the shared
// root that produced them both.
func TestCodexCoarseLinkageDisclosedWhenRootSharesDistinctTickets(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{"I972": "mechanical", "I973": "primary"})

	leadFile := filepath.Join(codexDir, "lead.jsonl")
	writeCodexFile(t, leadFile,
		codexSessionMetaLine("root-lc", "root-lc", "", sessRepo, "user", "{}"),
		codexFunctionCallLine("spawn_agent", map[string]string{"model": "gpt-5.6-luna", "task_name": "i972 coarse dispatch"}),
		codexFunctionCallLine("spawn_agent", map[string]string{"model": "gpt-5.6-sol", "task_name": "i973 coarse dispatch"}),
	)
	subFile := filepath.Join(codexDir, "sub.jsonl")
	writeCodexFile(t, subFile,
		codexSessionMetaLine("sub-lc", "root-lc", "root-lc", sessRepo, "subagent", threadSpawnSource("root-lc")),
		codexTurnContextLine("gpt-5.6-terra"),
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)

	r972 := rows["I972"]
	if r972.Verdict != VerdictEscalatedNoReason {
		t.Errorf("I972 verdict = %s (%s), want escalated-no-reason (the shared routine actual is above its mechanical declaration)", r972.Verdict, r972.Detail)
	}
	if !strings.Contains(r972.Detail, "coarse linkage") || !strings.Contains(r972.Detail, "I973") || !strings.Contains(r972.Detail, leadFile) {
		t.Errorf("I972 detail = %q, want a coarse-linkage note naming I973 and the shared root's file %s", r972.Detail, leadFile)
	}

	r973 := rows["I973"]
	if r973.Verdict != VerdictSilentDescent {
		t.Errorf("I973 verdict = %s (%s), want silent-descent (the shared routine actual is below its primary declaration)", r973.Verdict, r973.Detail)
	}
	if !strings.Contains(r973.Detail, "coarse linkage") || !strings.Contains(r973.Detail, "I972") || !strings.Contains(r973.Detail, leadFile) {
		t.Errorf("I973 detail = %q, want a coarse-linkage note naming I972 and the shared root's file %s", r973.Detail, leadFile)
	}
}

// Acceptance (D24 AC): unattributed-transcript never blocks and never
// changes exit-code-driving Blocking() — proven across every near-miss
// scenario at once (guardian-only, mid-transcript-only, orchestrator-only)
// alongside a genuine silent-descent, so Blocking() reflects only the real
// descent.
func TestCodexUnattributedTranscriptNeverBlocks(t *testing.T) {
	codexDir := t.TempDir()
	sessRepo := t.TempDir()
	writeAuditRepo(t, sessRepo, gen9DefaultWorkflow, map[string]string{
		"I980": "routine", // guardian-only match
		"I981": "routine", // mid-transcript-only match
		"I982": "routine", // orchestrator-only mention
		"I983": "primary", // genuine silent-descent, must still block
	})

	writeCodexFile(t, filepath.Join(codexDir, "guardian.jsonl"),
		codexSessionMetaLine("guard-980", "root-980", "root-980", sessRepo, "subagent", guardianSource),
		codexFunctionCallLine("spawn_agent", map[string]string{"model": "gpt-5.6-terra", "task_name": "i980 quoted replay"}),
	)
	writeCodexFile(t, filepath.Join(codexDir, "worker981.jsonl"),
		codexSessionMetaLine("worker-981", "worker-981", "", sessRepo, "user", "{}"),
		codexUserMessageLine("# Task I900-placeholder — implementer dispatch\n\nBuild the thing."),
		codexTurnContextLine("gpt-5.6-terra"),
		codexUserMessageLine("also take a look at I981 while you're there"),
	)
	writeCodexFile(t, filepath.Join(codexDir, "lead982.jsonl"),
		codexSessionMetaLine("lead-982", "lead-982", "", sessRepo, "user", "{}"),
		codexUserMessageLine("# Task I982 — orchestrator dispatch\n\nCoordinate the build."),
		codexTurnContextLine("gpt-5.6-sol"),
		codexFunctionCallLine("spawn_agent", map[string]string{"model": "gpt-5.6-terra", "task_name": "unrelated review pass"}),
	)
	writeCodexFile(t, filepath.Join(codexDir, "worker983.jsonl"),
		codexSessionMetaLine("worker-983", "worker-983", "", sessRepo, "user", "{}"),
		codexUserMessageLine("# Task I983 — implementer dispatch\n\nBuild the thing."),
		codexTurnContextLine("gpt-5.6-luna"), // mechanical, below primary — real descent
	)

	rep, err := Run(Options{RepoDir: sessRepo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)
	for _, id := range []string{"I980", "I981", "I982"} {
		if r := rows[id]; r.Verdict != VerdictUnattributedTranscript {
			t.Errorf("%s verdict = %s (%s), want unattributed-transcript", id, r.Verdict, r.Detail)
		}
	}
	if r := rows["I983"]; r.Verdict != VerdictSilentDescent {
		t.Errorf("I983 verdict = %s (%s), want silent-descent", r.Verdict, r.Detail)
	}
	if !rep.Blocking() {
		t.Error("I983's genuine silent-descent must still block")
	}
	// Blocking() only inspects VerdictSilentDescent (audit.go); this asserts
	// the behavioral guarantee end to end: three unattributed-transcript
	// tickets sitting alongside the real descent must not suppress or add to
	// what blocks.
}
