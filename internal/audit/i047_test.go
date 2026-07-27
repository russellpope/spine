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

// writeOrphanSubagent writes a subagent transcript + meta.json UNLINKED to
// any dispatch (its own toolUseId matches no dispatch's own id) under
// transcriptsDir/sessionID/subagents/agent-<name>.jsonl — isolating the
// agent-direct-description evidence path (Run's second attribution loop,
// the audit.go:369-area `use := containsToken(desc, t.id)` branch, gated
// by repoQualifies since I047 review finding I1) from the toolUseID-linked
// fallback, so a ticket's evidence can be proven to come from THIS path
// alone, not smuggled in via dispatch linkage.
func writeOrphanSubagent(t *testing.T, transcriptsDir, sessionID, name, toolUseID, cwd, description, model string) {
	t.Helper()
	subDir := filepath.Join(transcriptsDir, sessionID, "subagents")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(subDir, "agent-"+name)
	line := fmt.Sprintf(`{"type":"assistant","cwd":%q,"message":{"model":%q,"role":"assistant","content":[{"type":"text","text":"done"}]}}`+"\n", cwd, model)
	if err := os.WriteFile(base+".jsonl", []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	meta := fmt.Sprintf(`{"toolUseId":%q,"description":%q}`, toolUseID, description)
	if err := os.WriteFile(base+".meta.json", []byte(meta), 0o644); err != nil {
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
	writeAuditRepo(t, repoA, gen9DefaultWorkflow, map[string]string{"I003": "routine", "I005": "routine"})
	writeAuditRepo(t, repoB, gen9DefaultWorkflow, map[string]string{"I003": "primary"})

	// repoA-shaped: spine's I003, sonnet — correct for its own routine
	// declaration.
	writeSingleDispatch(t, filepath.Join(sharedTranscripts, "spine-build.jsonl"), repoA,
		"I003", "I003: gen-6 dispatch contract templates", "claude-sonnet-5")
	// repoB-shaped: praxis's UNRELATED I003, haiku — a real descent against
	// its own primary declaration.
	writeSingleDispatch(t, filepath.Join(sharedTranscripts, "praxis-build.jsonl"), repoB,
		"I003", "I003: unrelated praxis ticket", "claude-haiku-4-5")

	// I1 (I047 review fix round): the side door specifically. I005 has NO
	// dispatch at all — its only possible evidence is an orphan subagent
	// (unlinked to any dispatch's toolUseID) whose meta.json description
	// carries the ticket token directly. repoA's own orphan agent is
	// cwd-qualified and must be admitted; repoB's colliding orphan agent
	// (same shared dir, same "I005" token in its description, but cwd
	// outside repoA and a genuinely bad model) must be excluded — proving
	// the side door both admits its own repo's evidence and excludes the
	// other's, not just one or the other.
	writeOrphanSubagent(t, sharedTranscripts, "spine-agents", "a5", "orphan-a-i005", repoA,
		"I005: repoA-only agent evidence", "claude-sonnet-5")
	writeOrphanSubagent(t, sharedTranscripts, "praxis-agents", "b5", "orphan-b-i005", repoB,
		"I005: colliding agent mention from praxis", "claude-haiku-4-5")

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
	// I1: I005 has zero dispatch evidence — a match here can ONLY come from
	// the agent-direct-description side door, proving it admits repoA's own
	// orphan agent. If it instead pulled in repoB's colliding orphan
	// agent's haiku evidence too, this would be silent-descent, not match.
	if r := rowsA["I005"]; r.Verdict != VerdictMatch {
		t.Errorf("repoA I005 verdict = %s (%s), want match — evidence exists ONLY via the agent-direct-description side door", r.Verdict, r.Detail)
	}
	if got := strings.Join(rowsA["I005"].Actuals, ","); got != "claude-sonnet-5" {
		t.Errorf("repoA I005 actuals = %q, want only claude-sonnet-5 — praxis's colliding orphan agent (haiku) must not leak in via the side door", got)
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

// --- C1 (I047 review fix round): prefix-sharing sibling repos ---

// Acceptance (C1, I047 review): the I008 class survives naive substring
// matching for sibling repos whose names share a prefix — "praxis" and
// "praxis-web" is the review's own repro shape. A dispatch naming
// "praxis-web" in its prompt must NOT qualify an audit of "praxis": before
// the boundary fix, `strings.Contains`/the alnum-only `containsToken`
// treated "praxis-web" as containing "praxis" (hyphen isn't alnum, so it
// read as a word boundary) — exactly readmitting cross-repo collision for
// the commonest sibling-naming shape in the estate (base repo + "-docs"/
// "-web"/"-api" variants).
func TestD28SiblingRepoPrefixDoesNotQualify(t *testing.T) {
	base := t.TempDir()
	praxis := filepath.Join(base, "praxis")
	praxisWeb := filepath.Join(base, "praxis-web")
	if err := os.MkdirAll(praxis, 0o755); err != nil {
		t.Fatal(err)
	}
	writeAuditRepo(t, praxis, gen9DefaultWorkflow, map[string]string{"I003": "routine"})
	transcripts := t.TempDir()

	// praxis's own legitimate dispatch.
	writePathReferencingDispatch(t, filepath.Join(transcripts, "praxis-build.jsonl"), praxis,
		"I003", "I003: praxis own ticket", "claude-sonnet-5")
	// praxis-web's dispatch — a DIFFERENT repo that merely shares praxis's
	// name as a prefix. Its haiku evidence would be a real descent if it
	// were (wrongly) attributed to praxis's routine-declared I003.
	writePathReferencingDispatch(t, filepath.Join(transcripts, "praxis-web-build.jsonl"), praxisWeb,
		"I003", "I003: praxis-web unrelated ticket", "claude-haiku-4-5")

	rep, err := Run(Options{RepoDir: praxis, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I003"]
	if r.Verdict != VerdictMatch {
		t.Errorf("I003 verdict = %s (%s), want match — praxis-web's dispatch (a different, prefix-sharing repo) must not contaminate it", r.Verdict, r.Detail)
	}
	if got := strings.Join(r.Actuals, ","); got != "claude-sonnet-5" {
		t.Errorf("I003 actuals = %q, want only claude-sonnet-5 (praxis-web's haiku must not leak in)", got)
	}
	if rep.Blocking() {
		t.Error("must not block — praxis-web's dispatch is not evidence for praxis")
	}
	foundWeb := false
	for _, d := range rep.Unmatched {
		if strings.Contains(d.Description, "praxis-web") {
			foundWeb = true
		}
	}
	if !foundWeb {
		t.Errorf("want praxis-web's excluded dispatch visible in Unmatched, got %+v", rep.Unmatched)
	}
}

// Acceptance (C1 companion): the same boundary rule applies to the
// basename clause specifically (no absolute path in the prompt at all,
// only the bare sibling-repo name) — proving the fix isn't limited to the
// full-path clause.
func TestD28SiblingRepoBasenamePrefixDoesNotQualify(t *testing.T) {
	base := t.TempDir()
	praxis := filepath.Join(base, "praxis")
	if err := os.MkdirAll(praxis, 0o755); err != nil {
		t.Fatal(err)
	}
	writeAuditRepo(t, praxis, gen9DefaultWorkflow, map[string]string{"I004": "routine"})
	transcripts := t.TempDir()

	// Names only the bare sibling basename "praxis-web", never the audited
	// repo's own basename "praxis" as a whole token.
	writePathReferencingDispatch(t, filepath.Join(transcripts, "s1.jsonl"), "praxis-web",
		"I004", "I004: praxis-web basename only", "claude-haiku-4-5")

	rep, err := Run(Options{RepoDir: praxis, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	if r := rowsByID(t, rep)["I004"]; r.Verdict != VerdictNoTranscript {
		t.Errorf("I004 verdict = %s (%s), want no-transcript — \"praxis-web\" must not satisfy the \"praxis\" basename clause", r.Verdict, r.Detail)
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

// Acceptance (I047 review ruling 4): an unparseable --since value is a
// usage error — Run returns it directly (never a Report), the same class
// as a missing docs/issues dir. The CLI's existing err != nil handling in
// cmdAuditRouting already maps any Run error to exit 2, so this is the
// full behavior with no separate CLI-level test needed. Ruled AGAINST the
// original warn-and-ignore posture: a malformed operator-typed value has
// no valid fallback reading, and warning-then-proceeding would have
// silently readmitted exactly the sessions --since was told to exclude.
func TestSinceUnparseableIsUsageError(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I971": "routine"})
	tdir := t.TempDir()
	writeSingleDispatch(t, filepath.Join(tdir, "s1.jsonl"), dir, "I971", "I971: fixture work", "claude-sonnet-5")

	_, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir, Since: "not-a-time"})
	if err == nil {
		t.Fatal("want an error for an unparseable --since value, got nil")
	}
	for _, want := range []string{"--since", "not-a-time"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should contain %q", err, want)
		}
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

// --- Important-1 (final-review fix round): --since must scope a codex
// thread TREE as one unit, not per file ---

// Acceptance (Important-1, final whole-branch review — the codex
// counterpart of the claude-side I2 fix above): a codex thread tree's lead
// file and its linked thread_spawn subagent file must be scoped by --since
// TOGETHER, not independently. Fixture mirrors the review's own probe: the
// lead file (spawn_agent dispatch declaring the CORRECT routine-tier model)
// is newer; the linked subagent file (the REAL actual — a genuine descent
// to mechanical tier) is older. Before the fix, --since skipped the
// subagent file at walk time by its own mtime, before its root id was ever
// discovered — leaving the lead's clean declared alias standing alone and
// manufacturing a false `match`, hiding real descent behind the declared
// alias. The fix must keep the whole tree in scope whenever ANY member's
// mtime is at/after the cutoff (max-mtime-governs), exactly like claude's
// session-unit rule.
func TestCodexSinceScopesThreadTreeAsOneUnit(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I982": "routine"})
	codexDir := t.TempDir()
	leadPath := filepath.Join(codexDir, "lead.jsonl")
	subPath := filepath.Join(codexDir, "sub.jsonl")

	// Lead: declares gpt-5.6-terra — routine's OWN model, clean on its own.
	writeCodexFile(t, leadPath,
		codexSessionMetaLine("root-982", "root-982", "", dir, "user", "{}"),
		codexFunctionCallLine("spawn_agent", map[string]string{
			"model":     "gpt-5.6-terra",
			"task_name": "i982 dispatch",
		}),
	)
	// Linked subagent: the REAL actual, gpt-5.6-luna (mechanical) — a
	// genuine descent against the routine declaration once it supersedes
	// the lead's declared alias (D20 clause 2).
	writeCodexFile(t, subPath,
		codexSessionMetaLine("sub-982a", "root-982", "root-982", dir, "subagent", threadSpawnSource("root-982")),
		codexTurnContextLine("gpt-5.6-luna"),
	)

	// Straddle mtimes: lead NEW, subagent OLD — the typical ordering (a
	// spawned worker finishes, and stops touching its file, before its lead
	// keeps being appended to).
	chtime(t, leadPath, time.Date(2020, 1, 5, 0, 0, 0, 0, time.UTC))
	chtime(t, subPath, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))

	t.Run("no filter: linked subagent actual supersedes the alias, real descent shows", func(t *testing.T) {
		rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
		if err != nil {
			t.Fatal(err)
		}
		r := rowsByID(t, rep)["I982"]
		if r.Verdict != VerdictSilentDescent {
			t.Errorf("verdict = %s (%s), want silent-descent — the linked subagent's gpt-5.6-luna actual is real", r.Verdict, r.Detail)
		}
	})

	t.Run("--since between the subagent's old mtime and the lead's new mtime keeps the whole tree in scope", func(t *testing.T) {
		rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir, Since: "2020-01-03"})
		if err != nil {
			t.Fatal(err)
		}
		r := rowsByID(t, rep)["I982"]
		if r.Verdict != VerdictSilentDescent {
			t.Errorf("verdict = %s (%s), want silent-descent — tree-unit scoping must keep lead+subagent together (max mtime), not drop the subagent alone and fall back to the clean declared alias", r.Verdict, r.Detail)
		}
		if got := strings.Join(r.Actuals, ","); got != "gpt-5.6-luna" {
			t.Errorf("actuals = %q, want gpt-5.6-luna (the linked subagent actual) — a false match here (actuals gpt-5.6-terra) means the per-file mtime-skip bug is back", got)
		}
	})

	t.Run("--since after both mtimes excludes the whole tree", func(t *testing.T) {
		rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir, Since: "2020-01-10"})
		if err != nil {
			t.Fatal(err)
		}
		if r := rowsByID(t, rep)["I982"]; r.Verdict != VerdictNoTranscript {
			t.Errorf("verdict = %s (%s), want no-transcript — a cutoff after both mtimes must exclude the whole tree", r.Verdict, r.Detail)
		}
	})
}

// --- I2 (I047 review fix round): --since must scope a session as one unit ---

// Acceptance (I2, I047 review): a claude session's top-level "<id>.jsonl"
// file and its "<id>/subagents" dir must be scoped by --since TOGETHER, not
// independently. Fixture mirrors the review's own motivating scenario: the
// subagents dir's mtime is old (stamped when the subagent was spawned near
// session start) while the top-level file's mtime is newer (kept moving as
// the session is appended to). A cutoff falling between them must not keep
// the session's declared dispatch ALIAS ("sonnet", which resolves clean)
// while dropping the subagent's real ACTUAL (haiku — a genuine descent) —
// that combination is exactly the false-clean verdict the split would
// produce, hiding the evidence that catches an under-model worker.
func TestSinceScopesSessionFileAndSubagentsDirAsOneUnit(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I980": "routine"})
	tdir := t.TempDir()

	// The dispatch: declared alias "sonnet" (resolves to the routine tier
	// cleanly on its own).
	dispatchLine := fmt.Sprintf(
		`{"type":"assistant","cwd":%q,"message":{"model":"claude-fable-5","role":"assistant","content":[{"type":"tool_use","id":"tool-1","name":"Task","input":{"description":"I980: fixture work","model":"sonnet","prompt":"You are implementing ticket I980."}}]}}`+"\n",
		dir)
	if err := os.WriteFile(filepath.Join(tdir, "s1.jsonl"), []byte(dispatchLine), 0o644); err != nil {
		t.Fatal(err)
	}
	// The linked subagent: the REAL actual, haiku — a genuine descent
	// against the routine declaration once it supersedes the alias.
	subDir := filepath.Join(tdir, "s1", "subagents")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	agentLine := fmt.Sprintf(`{"type":"assistant","cwd":%q,"message":{"model":"claude-haiku-4-5","role":"assistant","content":[{"type":"text","text":"done"}]}}`+"\n", dir)
	if err := os.WriteFile(filepath.Join(subDir, "agent-x.jsonl"), []byte(agentLine), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "agent-x.meta.json"),
		[]byte(`{"toolUseId":"tool-1","description":"I980: fixture work"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Straddle mtimes: dir OLD, file NEW.
	chtime(t, filepath.Join(tdir, "s1"), time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	chtime(t, filepath.Join(tdir, "s1.jsonl"), time.Date(2020, 1, 5, 0, 0, 0, 0, time.UTC))

	t.Run("no filter: linked subagent evidence supersedes the alias, real descent shows", func(t *testing.T) {
		rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir})
		if err != nil {
			t.Fatal(err)
		}
		r := rowsByID(t, rep)["I980"]
		if r.Verdict != VerdictSilentDescent {
			t.Errorf("verdict = %s (%s), want silent-descent — the linked subagent's haiku actual is real", r.Verdict, r.Detail)
		}
	})

	t.Run("--since between dir's old mtime and file's new mtime keeps the whole session in scope", func(t *testing.T) {
		rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir, Since: "2020-01-03"})
		if err != nil {
			t.Fatal(err)
		}
		r := rowsByID(t, rep)["I980"]
		if r.Verdict != VerdictSilentDescent {
			t.Errorf("verdict = %s (%s), want silent-descent — session-unit scoping must keep file+dir together (max mtime), not drop the subagents dir alone and fall back to the clean alias", r.Verdict, r.Detail)
		}
		if got := strings.Join(r.Actuals, ","); got != "claude-haiku-4-5" {
			t.Errorf("actuals = %q, want claude-haiku-4-5 (the linked subagent actual) — a false match here means the split-scoping bug is back", got)
		}
	})

	t.Run("--since after both mtimes excludes the whole session", func(t *testing.T) {
		rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir, Since: "2020-01-10"})
		if err != nil {
			t.Fatal(err)
		}
		if r := rowsByID(t, rep)["I980"]; r.Verdict != VerdictNoTranscript {
			t.Errorf("verdict = %s (%s), want no-transcript — a cutoff after both mtimes must exclude the whole session", r.Verdict, r.Detail)
		}
	})
}

// Acceptance (I2 companion): the mtime ordering reversed from the test
// above — the subagents DIR is newer, the top-level FILE is older — so a
// "just use the file's own mtime" implementation (which would still pass
// the test above, since the file happened to be the newer piece there)
// gets caught here: it would wrongly exclude the whole session (file's
// mtime alone is before the cutoff) even though the dir is fresh. The fix
// must use the LATER of the two, in either direction, not favor one piece.
func TestSinceScopesSessionByMaxOfFileAndDirMtimeEitherDirection(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I981": "routine"})
	tdir := t.TempDir()

	dispatchLine := fmt.Sprintf(
		`{"type":"assistant","cwd":%q,"message":{"model":"claude-fable-5","role":"assistant","content":[{"type":"tool_use","id":"tool-2","name":"Task","input":{"description":"I981: fixture work","model":"sonnet","prompt":"You are implementing ticket I981."}}]}}`+"\n",
		dir)
	if err := os.WriteFile(filepath.Join(tdir, "s2.jsonl"), []byte(dispatchLine), 0o644); err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(tdir, "s2", "subagents")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	agentLine := fmt.Sprintf(`{"type":"assistant","cwd":%q,"message":{"model":"claude-haiku-4-5","role":"assistant","content":[{"type":"text","text":"done"}]}}`+"\n", dir)
	if err := os.WriteFile(filepath.Join(subDir, "agent-y.jsonl"), []byte(agentLine), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "agent-y.meta.json"),
		[]byte(`{"toolUseId":"tool-2","description":"I981: fixture work"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Reversed from the companion test: dir NEW, file OLD.
	chtime(t, filepath.Join(tdir, "s2.jsonl"), time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	chtime(t, filepath.Join(tdir, "s2"), time.Date(2020, 1, 5, 0, 0, 0, 0, time.UTC))

	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir, Since: "2020-01-03"})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I981"]
	if r.Verdict != VerdictSilentDescent {
		t.Errorf("verdict = %s (%s), want silent-descent — the dir's newer mtime must be able to keep the session in scope even though the file alone is older than the cutoff", r.Verdict, r.Detail)
	}
}

// --- M3 (I047 review fix round): --session matching nothing warns ---

// Acceptance (M3, I047 review): a non-empty --session that matches no
// session anywhere (claude or codex) warns, rather than producing an
// unexplained all-no-transcript audit an operator has no way to diagnose
// (especially for codex, whose root ids never appear in filenames).
func TestSessionMatchingNothingWarns(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I990": "routine"})
	tdir := t.TempDir()
	writeSingleDispatch(t, filepath.Join(tdir, "s1.jsonl"), dir, "I990", "I990: fixture work", "claude-sonnet-5")

	t.Run("no match anywhere warns", func(t *testing.T) {
		rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir, Session: "typo-d-session-id"})
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, w := range rep.Warnings {
			if strings.Contains(w, `--session "typo-d-session-id" matched no sessions`) {
				found = true
			}
		}
		if !found {
			t.Errorf("want a warning naming the unmatched --session value, got %q", rep.Warnings)
		}
	})

	t.Run("a real match does not warn", func(t *testing.T) {
		rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir, Session: "s1"})
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range rep.Warnings {
			if strings.Contains(w, "matched no sessions") {
				t.Errorf("a real --session match must not warn, got %q", rep.Warnings)
			}
		}
	})
}

// --- Important-2 (final-review fix round): coarse-linkage note must not
// fire on claude-only evidence ---

// Acceptance (Important-2, final whole-branch review): coarseLinkageNotes
// gates on rootTickets, which is populated for EVERY flavor (see its own
// doc) — not just codex, despite the codex-specific wording of the
// disclosure text it drives. A single claude Task dispatch whose own
// description names two ticket ids claims both under that one toolUseID,
// so a linked subagent transcript superseding the declared alias produces
// exactly the "root claimed by >=2 distinct tickets with a linked actual"
// shape coarseLinkageNotes looks for — on pure claude evidence, with no
// codex sessions dir configured at all. Before the fix, this fired the
// codex-worded note ("codex session root", "(D20)") on both tickets,
// breaking the I040 claude-only byte-identity promise. The fix scopes the
// note to roots carrying codex's own "codex:" toolUseID prefix.
func TestCoarseLinkageNoteDoesNotFireOnClaudeOnlyEvidence(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I201": "mechanical", "I202": "primary"})
	tdir := t.TempDir()

	// One dispatch, one toolUseID, description naming BOTH tickets.
	dispatchLine := fmt.Sprintf(
		`{"type":"assistant","cwd":%q,"message":{"model":"claude-fable-5","role":"assistant","content":[{"type":"tool_use","id":"tool-combined","name":"Task","input":{"description":"I201 and I202: combined fix","model":"claude-haiku-4-5","prompt":"You are implementing tickets I201 and I202."}}]}}`+"\n",
		dir)
	if err := os.WriteFile(filepath.Join(tdir, "s1.jsonl"), []byte(dispatchLine), 0o644); err != nil {
		t.Fatal(err)
	}
	// Linked subagent: its actual (claude-sonnet-5, routine) supersedes the
	// dispatch's declared alias for BOTH tickets once linked — the shared,
	// root-coarse actual coarseLinkageNotes exists to disclose for codex.
	subDir := filepath.Join(tdir, "s1", "subagents")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	agentLine := fmt.Sprintf(`{"type":"assistant","cwd":%q,"message":{"model":"claude-sonnet-5","role":"assistant","content":[{"type":"text","text":"done"}]}}`+"\n", dir)
	if err := os.WriteFile(filepath.Join(subDir, "agent-x.jsonl"), []byte(agentLine), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "agent-x.meta.json"),
		[]byte(`{"toolUseId":"tool-combined","description":"I201 and I202: combined fix"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Claude-only repo: no CodexSessionsDir at all.
	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)

	r201 := rows["I201"]
	if r201.Verdict != VerdictEscalatedNoReason {
		t.Errorf("I201 verdict = %s (%s), want escalated-no-reason (the shared routine actual is above its mechanical declaration)", r201.Verdict, r201.Detail)
	}
	if strings.Contains(r201.Detail, "coarse linkage") {
		t.Errorf("I201 detail = %q, must not carry the codex-worded coarse-linkage note on claude-only evidence", r201.Detail)
	}
	if strings.Contains(strings.ToLower(r201.Detail), "codex") {
		t.Errorf("I201 detail = %q, must not mention codex on a claude-only audit", r201.Detail)
	}

	r202 := rows["I202"]
	if r202.Verdict != VerdictSilentDescent {
		t.Errorf("I202 verdict = %s (%s), want silent-descent (the shared routine actual is below its primary declaration)", r202.Verdict, r202.Detail)
	}
	if strings.Contains(r202.Detail, "coarse linkage") {
		t.Errorf("I202 detail = %q, must not carry the codex-worded coarse-linkage note on claude-only evidence", r202.Detail)
	}
	if strings.Contains(strings.ToLower(r202.Detail), "codex") {
		t.Errorf("I202 detail = %q, must not mention codex on a claude-only audit", r202.Detail)
	}
}
