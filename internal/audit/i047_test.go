package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- D28 fixture helpers (ticket I047) ---

// writeSingleDispatch writes one session file carrying exactly one Task
// dispatch, with an explicit "cwd" on its event line (D28's cwd-evidence
// clause) — the shape real ~/.claude session/subagent lines always carry
// (verified against ground truth in the package doc's readTranscripts
// comment).
func writeSingleDispatch(t *testing.T, path, cwd, ticketID, desc, model string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	line := fmt.Sprintf(
		`{"type":"assistant","cwd":%q,"message":{"model":"claude-fable-5","role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Task","input":{"description":%q,"model":%q,"prompt":"You are implementing ticket %s."}}]}}`+"\n",
		cwd, desc, model, ticketID)
	if err := os.WriteFile(path, []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writePathReferencingDispatch writes one session file whose dispatch
// PROMPT names the repo (absolute path or basename token) instead of
// carrying a cwd — D28's other qualifying clause. No cwd field at all, so
// a match here proves the path/basename reference alone is sufficient.
func writePathReferencingDispatch(t *testing.T, path, repoRef, ticketID, desc, model string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	line := fmt.Sprintf(
		`{"type":"assistant","message":{"model":"claude-fable-5","role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Task","input":{"description":%q,"model":%q,"prompt":"You are implementing ticket %s. Repo: %s"}}]}}`+"\n",
		desc, model, ticketID, repoRef)
	if err := os.WriteFile(path, []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
}

func chtime(t *testing.T, path string, when time.Time) {
	t.Helper()
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatal(err)
	}
}

// --- AC: I008-shaped fixture ---

// Acceptance (D28, ticket I047 — the I008 incident this design fixes): two
// repos share one controller transcript dir (I008's actual scenario:
// "controller sessions for different builds often share one project dir")
// and both carry an I003 ticket. Pre-D28, naive ticket-token matching would
// let each repo's audit see the OTHER repo's dispatch too — I008's reported
// consequence was a false blocking silent-descent. Here repoB's I003
// dispatch is a GENUINE descent (haiku vs its own declared primary) so
// this also proves D28 doesn't launder real descent away: repoB must still
// correctly block on its OWN evidence, just never on repoA's.
func TestI008ShapedCrossRepoSharedTranscriptDirNoContamination(t *testing.T) {
	sharedTranscripts := t.TempDir()
	repoA := t.TempDir()
	repoB := t.TempDir()
	writeAuditRepo(t, repoA, gen9DefaultWorkflow, map[string]string{"I003": "routine"})
	writeAuditRepo(t, repoB, gen9DefaultWorkflow, map[string]string{"I003": "primary"})

	// repoA-shaped: spine's I003, sonnet — correct for its own routine
	// declaration.
	writeSingleDispatch(t, filepath.Join(sharedTranscripts, "spine-build.jsonl"), repoA,
		"I003", "I003: gen-6 dispatch contract templates", "claude-sonnet-5")
	// repoB-shaped: praxis's UNRELATED I003, haiku — a real descent against
	// its own primary declaration.
	writeSingleDispatch(t, filepath.Join(sharedTranscripts, "praxis-build.jsonl"), repoB,
		"I003", "I003: unrelated praxis ticket", "claude-haiku-4-5")

	repA, err := Run(Options{RepoDir: repoA, ClaudeTranscriptsDir: sharedTranscripts})
	if err != nil {
		t.Fatal(err)
	}
	rowsA := rowsByID(t, repA)
	if r := rowsA["I003"]; r.Verdict != VerdictMatch {
		t.Errorf("repoA I003 verdict = %s (%s), want match — praxis's dispatch must not contaminate it", r.Verdict, r.Detail)
	}
	if got := strings.Join(rowsA["I003"].Actuals, ","); got != "claude-sonnet-5" {
		t.Errorf("repoA I003 actuals = %q, want only claude-sonnet-5 (its own evidence)", got)
	}
	if repA.Blocking() {
		t.Error("repoA must not block — its own I003 evidence matches cleanly")
	}
	foundPraxis := false
	for _, d := range repA.Unmatched {
		if strings.Contains(d.Description, "praxis") {
			foundPraxis = true
		}
	}
	if !foundPraxis {
		t.Errorf("repoA's Unmatched must list praxis's excluded dispatch, not silently drop it: %+v", repA.Unmatched)
	}

	repB, err := Run(Options{RepoDir: repoB, ClaudeTranscriptsDir: sharedTranscripts})
	if err != nil {
		t.Fatal(err)
	}
	rowsB := rowsByID(t, repB)
	if r := rowsB["I003"]; r.Verdict != VerdictSilentDescent {
		t.Errorf("repoB I003 verdict = %s (%s), want silent-descent — this is REAL descent from its own transcript, not false", r.Verdict, r.Detail)
	}
	if got := strings.Join(rowsB["I003"].Actuals, ","); got != "claude-haiku-4-5" {
		t.Errorf("repoB I003 actuals = %q, want only claude-haiku-4-5 (its own evidence, not spine's sonnet)", got)
	}
	if !repB.Blocking() {
		t.Error("repoB must block — its own I003 dispatch really did descend below primary with no ESCALATION record")
	}
	foundSpine := false
	for _, d := range repB.Unmatched {
		if strings.Contains(d.Description, "gen-6 dispatch contract templates") {
			foundSpine = true
		}
	}
	if !foundSpine {
		t.Errorf("repoB's Unmatched must list spine's excluded dispatch, not silently drop it: %+v", repB.Unmatched)
	}
}

// --- AC: repo path/basename claims normally; cwd-evidence path also claims ---

// Acceptance: a dispatch whose PROMPT names the audited repo's absolute
// path claims normally — no cwd field needed at all.
func TestD28DispatchClaimsViaAbsolutePathReference(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I960": "routine"})
	tdir := t.TempDir()
	writePathReferencingDispatch(t, filepath.Join(tdir, "s1.jsonl"), dir, "I960", "I960: fixture work", "claude-sonnet-5")

	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir})
	if err != nil {
		t.Fatal(err)
	}
	if r := rowsByID(t, rep)["I960"]; r.Verdict != VerdictMatch {
		t.Errorf("I960 verdict = %s (%s), want match — the prompt names the repo's absolute path", r.Verdict, r.Detail)
	}
}

// Acceptance: a dispatch whose prompt names only the repo's BASENAME (not
// the full path) also claims — D28's "absolute path OR basename token"
// wording.
func TestD28DispatchClaimsViaBasenameReference(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I961": "routine"})
	tdir := t.TempDir()
	writePathReferencingDispatch(t, filepath.Join(tdir, "s1.jsonl"), filepath.Base(dir), "I961", "I961: fixture work", "claude-sonnet-5")

	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir})
	if err != nil {
		t.Fatal(err)
	}
	if r := rowsByID(t, rep)["I961"]; r.Verdict != VerdictMatch {
		t.Errorf("I961 verdict = %s (%s), want match — the prompt names the repo's basename token", r.Verdict, r.Detail)
	}
}

// Acceptance: a dispatch whose prompt names NEITHER the repo's path nor its
// basename, but whose own session cwd resolves inside the repo, also
// claims — D28's other qualifying clause, independent of the text.
func TestD28DispatchClaimsViaCwdEvidence(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I962": "routine"})
	tdir := t.TempDir()
	writeSingleDispatch(t, filepath.Join(tdir, "s1.jsonl"), dir, "I962", "I962: fixture work", "claude-sonnet-5")

	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir})
	if err != nil {
		t.Fatal(err)
	}
	if r := rowsByID(t, rep)["I962"]; r.Verdict != VerdictMatch {
		t.Errorf("I962 verdict = %s (%s), want match — the session's own cwd resolves inside the repo", r.Verdict, r.Detail)
	}
}

// Acceptance: a dispatch naming the ticket token but referencing NEITHER
// the repo's path/basename nor carrying cwd evidence inside it fails to
// qualify — the false-negative surface D28's own ticket text flags. It
// must not vanish: it surfaces in Unmatched, and the ticket honestly
// degrades to no-transcript rather than manufacturing a match.
func TestD28UnqualifiedDispatchStaysUnmatched(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I963": "routine"})
	tdir := t.TempDir()
	// No cwd field, and the prompt text names neither the repo's absolute
	// path nor its basename — an unqualified dispatch shape.
	line := `{"type":"assistant","message":{"model":"claude-fable-5","role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Task","input":{"description":"I963: fixture work","model":"claude-sonnet-5","prompt":"You are implementing ticket I963."}}]}}` + "\n"
	if err := os.WriteFile(filepath.Join(tdir, "s1.jsonl"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}

	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir})
	if err != nil {
		t.Fatal(err)
	}
	if r := rowsByID(t, rep)["I963"]; r.Verdict != VerdictNoTranscript {
		t.Errorf("I963 verdict = %s (%s), want no-transcript — unqualified evidence must not manufacture a match", r.Verdict, r.Detail)
	}
	if len(rep.Unmatched) != 1 || !strings.Contains(rep.Unmatched[0].Description, "I963") {
		t.Errorf("want the disqualified dispatch visible in Unmatched, got %+v", rep.Unmatched)
	}
}

// --- AC: --since / --session / composition ---

// Acceptance: --since excludes an older session's evidence from the
// aggregate verdict; --session restricts to exactly one session; and the
// two compose (AND, not OR) rather than one silently overriding the other.
// Fixture: "old.jsonl" (2020-01-01, a genuine mechanical-tier dispatch —
// descent against the routine annotation) and "new.jsonl" (2020-01-05, a
// clean routine-tier dispatch), both for the same ticket, same repo.
func TestSinceAndSessionFiltersComposeWithDefaults(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I970": "routine"})
	tdir := t.TempDir()
	oldPath := filepath.Join(tdir, "old.jsonl")
	newPath := filepath.Join(tdir, "new.jsonl")
	writeSingleDispatch(t, oldPath, dir, "I970", "I970: old descent", "claude-haiku-4-5")
	writeSingleDispatch(t, newPath, dir, "I970", "I970: new clean run", "claude-sonnet-5")
	chtime(t, oldPath, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	chtime(t, newPath, time.Date(2020, 1, 5, 0, 0, 0, 0, time.UTC))

	t.Run("no filters: both sessions' evidence merges, real descent blocks", func(t *testing.T) {
		rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir})
		if err != nil {
			t.Fatal(err)
		}
		r := rowsByID(t, rep)["I970"]
		if r.Verdict != VerdictSilentDescent {
			t.Errorf("verdict = %s (%s), want silent-descent — old.jsonl's haiku evidence is real", r.Verdict, r.Detail)
		}
		if !rep.Blocking() {
			t.Error("want blocking with no filters applied")
		}
	})

	t.Run("--since excludes the older session", func(t *testing.T) {
		rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir, Since: "2020-01-03"})
		if err != nil {
			t.Fatal(err)
		}
		r := rowsByID(t, rep)["I970"]
		if r.Verdict != VerdictMatch {
			t.Errorf("verdict = %s (%s), want match — old.jsonl (2020-01-01) excluded by --since 2020-01-03, leaving only new.jsonl's clean routine dispatch", r.Verdict, r.Detail)
		}
		if got := strings.Join(r.Actuals, ","); got != "claude-sonnet-5" {
			t.Errorf("actuals = %q, want only claude-sonnet-5 — old.jsonl's evidence must not appear", got)
		}
		if rep.Blocking() {
			t.Error("must not block — the only in-scope evidence is a clean match")
		}
	})

	t.Run("--session restricts to exactly one session", func(t *testing.T) {
		rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir, Session: "new"})
		if err != nil {
			t.Fatal(err)
		}
		r := rowsByID(t, rep)["I970"]
		if r.Verdict != VerdictMatch {
			t.Errorf("verdict = %s (%s), want match — --session new isolates new.jsonl alone", r.Verdict, r.Detail)
		}
		if got := strings.Join(r.Actuals, ","); got != "claude-sonnet-5" {
			t.Errorf("actuals = %q, want only claude-sonnet-5", got)
		}
	})

	t.Run("--since and --session compose (AND, not OR)", func(t *testing.T) {
		// --session targets the OLD (bad) session specifically, but --since
		// 2020-01-03 excludes anything older than that — old.jsonl is
		// 2020-01-01. If the filters composed wrong (e.g. --session
		// overriding --since, or an OR instead of an AND), old's evidence
		// would leak back in as a match or a descent; the honest answer is
		// that nothing in scope survives both filters at once.
		rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir, Session: "old", Since: "2020-01-03"})
		if err != nil {
			t.Fatal(err)
		}
		r := rowsByID(t, rep)["I970"]
		if r.Verdict != VerdictNoTranscript {
			t.Errorf("verdict = %s (%s), want no-transcript — old.jsonl is selected by --session but excluded by --since; the two filters must compose, not the more permissive one winning", r.Verdict, r.Detail)
		}
	})
}

// Acceptance: an unparseable --since value degrades to a Report warning
// and the filter is ignored (not applied) — consistent with every other
// operator-input-trouble path in this package.
func TestSinceUnparseableWarnsAndIsIgnored(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I971": "routine"})
	tdir := t.TempDir()
	writeSingleDispatch(t, filepath.Join(tdir, "s1.jsonl"), dir, "I971", "I971: fixture work", "claude-sonnet-5")

	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir, Since: "not-a-time"})
	if err != nil {
		t.Fatal(err)
	}
	if r := rowsByID(t, rep)["I971"]; r.Verdict != VerdictMatch {
		t.Errorf("verdict = %s (%s), want match — an unparseable --since must not silently exclude everything", r.Verdict, r.Detail)
	}
	found := false
	for _, w := range rep.Warnings {
		if strings.Contains(w, "--since") && strings.Contains(w, "not-a-time") {
			found = true
		}
	}
	if !found {
		t.Errorf("want a warning naming the unparseable --since value, got %q", rep.Warnings)
	}
}

// --- AC: --since / --session apply to codex discovery too ---

// Acceptance: D28's --since/--session filters scope codex discovery
// exactly as they scope claude discovery (ticket text: "both claude and
// codex sides"). Two codex rollout files for the same ticket, different
// root session ids and mtimes; --since excludes the older, --session
// restricts to one root id (session_meta.payload.session_id).
func TestCodexSinceAndSessionFilters(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I972": "routine"})
	codexDir := t.TempDir()
	oldPath := filepath.Join(codexDir, "old.jsonl")
	newPath := filepath.Join(codexDir, "new.jsonl")
	writeCodexFile(t, oldPath,
		codexSessionMetaLine("root-old", "root-old", "", dir, "user", "{}"),
		codexFunctionCallLine("spawn_agent", map[string]string{"model": "gpt-5.6-luna", "task_name": "i972 old leg"}),
	)
	writeCodexFile(t, newPath,
		codexSessionMetaLine("root-new", "root-new", "", dir, "user", "{}"),
		codexFunctionCallLine("spawn_agent", map[string]string{"model": "gpt-5.6-terra", "task_name": "i972 new leg"}),
	)
	chtime(t, oldPath, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	chtime(t, newPath, time.Date(2020, 1, 5, 0, 0, 0, 0, time.UTC))

	t.Run("no filters: both roots contribute", func(t *testing.T) {
		rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
		if err != nil {
			t.Fatal(err)
		}
		got := strings.Join(rowsByID(t, rep)["I972"].Actuals, ",")
		if got != "gpt-5.6-luna,gpt-5.6-terra" {
			t.Errorf("actuals = %q, want both roots' models", got)
		}
	})

	t.Run("--since excludes the older codex session", func(t *testing.T) {
		rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir, Since: "2020-01-03"})
		if err != nil {
			t.Fatal(err)
		}
		got := strings.Join(rowsByID(t, rep)["I972"].Actuals, ",")
		if got != "gpt-5.6-terra" {
			t.Errorf("actuals = %q, want only the newer root's model", got)
		}
	})

	t.Run("--session restricts to one codex root", func(t *testing.T) {
		rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir, Session: "root-old"})
		if err != nil {
			t.Fatal(err)
		}
		got := strings.Join(rowsByID(t, rep)["I972"].Actuals, ",")
		if got != "gpt-5.6-luna" {
			t.Errorf("actuals = %q, want only root-old's model", got)
		}
	})
}
