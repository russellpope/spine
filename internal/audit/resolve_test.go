package audit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// writeAuditRepo lays down a minimal auditable repo: WORKFLOW.md (skipped
// when content is empty — the missing-file case) plus one annotated ticket
// per (id, tier).
func writeAuditRepo(t *testing.T, dir, workflow string, tickets map[string]string) {
	t.Helper()
	if workflow != "" {
		if err := os.WriteFile(filepath.Join(dir, "WORKFLOW.md"), []byte(workflow), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs", "issues"), 0o755); err != nil {
		t.Fatal(err)
	}
	for id, tier := range tickets {
		gen6ProofTicket(t, dir, id, tier)
	}
}

// This catches the D36 union losing any of its three routes (primary path,
// live git worktree, or prefix-only removed worktree), double-reading a live
// worktree found by two routes, or weakening D28 while sweeping a prefix
// sibling. Each transcript is real JSONL consumed by Run.
func TestDefaultTranscriptsDirsUnionScansWorktreesWithoutOverAttribution(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	makeTestGitRepo(t, repo)
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{
		"I901": "routine",
		"I902": "routine",
		"I903": "routine",
	})

	worktree := filepath.Join(t.TempDir(), "live-worktree")
	if out, err := exec.Command("git", "-C", repo, "worktree", "add", "-q", worktree).CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v\n%s", err, out)
	}

	primaryDir, err := DefaultTranscriptsDir(repo)
	if err != nil {
		t.Fatal(err)
	}
	worktreeReal, err := filepath.EvalSymlinks(worktree)
	if err != nil {
		t.Fatal(err)
	}
	worktreeDir, err := DefaultTranscriptsDir(worktreeReal)
	if err != nil {
		t.Fatal(err)
	}
	removedDir := filepath.Join(filepath.Dir(primaryDir), filepath.Base(primaryDir)+"-removed-worktree")
	decoyDir := filepath.Join(filepath.Dir(primaryDir), filepath.Base(primaryDir)+"-sibling")

	writeSingleDispatch(t, filepath.Join(primaryDir, "primary.jsonl"), repo,
		"I901", "I901 primary transcript", "claude-sonnet-5")
	writePathReferencingDispatch(t, filepath.Join(worktreeDir, "live.jsonl"), repo,
		"I902", "I902 live worktree transcript", "claude-sonnet-5")
	writePathReferencingDispatch(t, filepath.Join(removedDir, "removed.jsonl"), repo,
		"I903", "I903 removed worktree transcript", "claude-sonnet-5")
	writeSingleDispatch(t, filepath.Join(decoyDir, "sibling.jsonl"), t.TempDir(),
		"I901", "I901 sibling transcript", "claude-haiku-4-5")

	dirs, err := DefaultTranscriptsDirs(repo)
	if err != nil {
		t.Fatal(err)
	}
	wantDirs := []string{primaryDir, worktreeDir, removedDir, decoyDir}
	sort.Strings(wantDirs)
	if !reflect.DeepEqual(dirs, wantDirs) {
		t.Fatalf("DefaultTranscriptsDirs() = %q, want %q", dirs, wantDirs)
	}

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDirs: dirs})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)
	for _, id := range []string{"I901", "I902", "I903"} {
		if r := rows[id]; r.Verdict != VerdictMatch {
			t.Errorf("%s verdict = %s (%s), want match", id, r.Verdict, r.Detail)
		}
	}
	if got := strings.Join(rows["I901"].Actuals, ","); got != "claude-sonnet-5" {
		t.Errorf("I901 actuals = %q, want only primary claude-sonnet-5; D28 must reject the decoy", got)
	}
	if len(rep.Unmatched) != 1 || !strings.Contains(rep.Unmatched[0].Description, "sibling transcript") {
		t.Errorf("decoy must stay visibly unmatched, got %+v", rep.Unmatched)
	}

	var scanned []string
	for _, warning := range rep.Warnings {
		if dir, ok := strings.CutPrefix(warning, "scanning transcript dir: "); ok {
			scanned = append(scanned, dir)
		}
	}
	if !reflect.DeepEqual(scanned, wantDirs) {
		t.Errorf("scanned-directory warnings = %q, want one stable entry per union directory %q", scanned, wantDirs)
	}
}

// writeDispatchTranscript writes a single-session transcript into
// transcriptsDir with one Task dispatch per (ticket id -> model token),
// issued by a fable controller session (main-session models are never
// ticket evidence). Each event line carries "cwd": repoDir (D28, ticket
// I047: real transcript lines always carry their session's cwd) so these
// dispatches keep qualifying under the D28 repo-qualification gate — every
// caller already has its fixture repoDir on hand.
func writeDispatchTranscript(t *testing.T, repoDir, transcriptsDir string, dispatches map[string]string) {
	t.Helper()
	var b strings.Builder
	i := 0
	for id, token := range dispatches {
		i++
		fmt.Fprintf(&b,
			`{"type":"assistant","cwd":%q,"message":{"model":"claude-fable-5","role":"assistant","content":[{"type":"tool_use","id":"toolu_%d","name":"Task","input":{"description":"%s: fixture work","model":"%s","prompt":"You are implementing ticket %s."}}]}}`+"\n",
			repoDir, i, id, token, id)
	}
	if err := os.WriteFile(filepath.Join(transcriptsDir, "s1.jsonl"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

const gen9DefaultWorkflow = `# Workflow — proof

profile: go-service
template_version: 9
model_routing:
  primary: claude-fable-5          # default thinker
  routine: claude-sonnet-5         # multi-step mechanical subagent roles
  mechanical: claude-haiku-4-5     # verbatim plan transcription only
  fallback: claude-opus-4-8        # primary-refused or security-framed work
`

// Acceptance (D14): a WORKFLOW.md stamped with a generation newer than the
// binary compiles is refused with a clear message — the audit must not emit
// confident verdicts from a misparse. A non-integer stamp falls through,
// matching spine update's gate.
func TestAuditRefusesNewerGeneration(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, "profile: go-service\ntemplate_version: 99\n",
		map[string]string{"I801": "primary"})
	_, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: t.TempDir()})
	if err == nil {
		t.Fatal("want a refusal for template generation 99, got nil error")
	}
	for _, want := range []string{"generation 99", "upgrade spine"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal message %q should contain %q", err, want)
		}
	}

	tolerant := t.TempDir()
	writeAuditRepo(t, tolerant, "profile: go-service\ntemplate_version: someday\n",
		map[string]string{"I802": "primary"})
	if _, err := Run(Options{RepoDir: tolerant, ClaudeTranscriptsDir: t.TempDir()}); err != nil {
		t.Fatalf("non-integer stamp must fall through, got %v", err)
	}
}

// Acceptance (D13): explicit aliases replace substring containment. A
// declared alias still resolves; a proper substring of a model id — which
// the retired strings.Contains matching would have accepted — reports
// unmapped rather than guessing a tier.
func TestSubstringTokenNoLongerMaps(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow,
		map[string]string{"I811": "routine", "I812": "routine", "I813": "routine"})
	tdir := t.TempDir()
	writeDispatchTranscript(t, dir, tdir, map[string]string{
		"I811": "claude-sonnet",   // substring of claude-sonnet-5, not an alias
		"I812": "opus",            // current routine alias
		"I813": "claude-sonnet-5", // exact historical id
	})
	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)
	r := rows["I811"]
	if r.Verdict != VerdictUnmappedDispatch {
		t.Errorf("I811 verdict = %s (%s), want unmapped-dispatch — substring matching must be gone", r.Verdict, r.Detail)
	}
	if !strings.Contains(r.Detail, "claude") {
		t.Errorf("I811 detail should name the flavor the token failed to resolve in, got %q", r.Detail)
	}
	if r := rows["I812"]; r.Verdict != VerdictMatch {
		t.Errorf("I812 verdict = %s (%s), want match via the declared alias", r.Verdict, r.Detail)
	}
	if r := rows["I813"]; r.Verdict != VerdictMatch {
		t.Errorf("I813 verdict = %s (%s), want match via the exact historical id", r.Verdict, r.Detail)
	}
}

// A historical default id still maps to its tier by exact id — the fleet has
// run claude-opus-4-8 dispatches, and their transcripts must stay auditable
// after the mirror refreshes to claude-opus-5. Historical ids carry no
// aliases, so the match is by full id only; the tier it maps to is fallback,
// so the ordinary lateral-fallback rules then apply.
func TestHistoricalIDMatchesByExactID(t *testing.T) {
	dir := t.TempDir()
	// No model_routing block at all: every tier resolves to the embedded
	// defaults, so fallback is claude-opus-5 and the old id is history-only.
	writeAuditRepo(t, dir, "# Workflow — proof\n\nprofile: go-service\ntemplate_version: 9\n",
		map[string]string{"I821": "fallback", "I822": "primary"})
	tdir := t.TempDir()
	writeDispatchTranscript(t, dir, tdir, map[string]string{
		"I821": "claude-opus-4-8",
		"I822": "claude-opus-4-8",
	})
	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)
	if r := rows["I821"]; r.Verdict != VerdictMatch {
		t.Errorf("I821 verdict = %s (%s), want match — historical id on a fallback-annotated ticket", r.Verdict, r.Detail)
	}
	if r := rows["I822"]; r.Verdict != VerdictUnexplainedFallback {
		t.Errorf("I822 verdict = %s (%s), want unexplained-fallback — lateral, never descent, never unmapped", r.Verdict, r.Detail)
	}
	if rep.Blocking() {
		t.Error("historical-id evidence must not block")
	}
}

// Acceptance (D15, flavor scoping): tier resolution is scoped to the
// dispatch's transcript-derived flavor — claude for everything audited
// today — so an id declared only under codex reports unmapped rather than
// resolving through another flavor's table.
func TestCodexIDInvisibleWithinClaudeFlavor(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I831": "primary"})
	tdir := t.TempDir()
	writeDispatchTranscript(t, dir, tdir, map[string]string{"I831": "gpt-5.6-sol"})
	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir})
	if err != nil {
		t.Fatal(err)
	}
	if r := rowsByID(t, rep)["I831"]; r.Verdict != VerdictUnmappedDispatch {
		t.Errorf("I831 verdict = %s (%s), want unmapped-dispatch — codex ids must not resolve within the claude flavor", r.Verdict, r.Detail)
	}
}

// I040: proves the per-token flavor seam directly. judgeToken must resolve
// each evidenceToken within the table named by its own flavor field, not a
// single table shared across every token judged for a ticket. No fixture
// can yet exercise this end to end through Run (only the claude reader is
// wired, so every token Run produces carries "claude"); this pins the
// plumbing the deferred codex reader will rely on without churning judge or
// judgeToken again.
func TestJudgeTokenResolvesWithinItsOwnFlavor(t *testing.T) {
	mappings := map[string]map[string]resolvedTier{
		"claude": {"primary": {id: "claude-model-x"}},
		"codex":  {"primary": {id: "codex-model-x"}},
	}
	tk := ticket{id: "I900", tier: "primary"}

	// Same declared tier, disjoint tables: a codex-flavored token matches
	// the codex entry...
	if v, d := judgeToken(evidenceToken{value: "codex-model-x", flavor: "codex"}, tk, mappings, ledger{}); v != VerdictMatch {
		t.Errorf("codex-flavored token: verdict = %s (%s), want match against the codex table", v, d)
	}
	// ...the identical string tagged claude instead must not: the flavor on
	// the token, not the value, decides which table resolves it.
	if v, d := judgeToken(evidenceToken{value: "codex-model-x", flavor: "claude"}, tk, mappings, ledger{}); v != VerdictUnmappedDispatch {
		t.Errorf("claude-flavored token carrying a codex-only id: verdict = %s (%s), want unmapped-dispatch", v, d)
	}
	// A claude-flavored token still matches the claude table normally.
	if v, d := judgeToken(evidenceToken{value: "claude-model-x", flavor: "claude"}, tk, mappings, ledger{}); v != VerdictMatch {
		t.Errorf("claude-flavored token: verdict = %s (%s), want match against the claude table", v, d)
	}
	// A flavor with no resolved table at all degrades to unmapped, never a
	// panic or a silent match.
	if v, d := judgeToken(evidenceToken{value: "anything", flavor: "unknown-flavor"}, tk, mappings, ledger{}); v != VerdictUnmappedDispatch {
		t.Errorf("unresolved-flavor token: verdict = %s (%s), want unmapped-dispatch", v, d)
	}
}

// Two tiers sharing one id is legal (the shipped codex routine/fallback pair
// is the live case); within a flavor the ambiguity resolves by the rule the
// audit has always applied: the reading closest to a non-verdict wins —
// declared tier if among the candidates, else the highest ordered tier.
// Ambiguity must not manufacture descent, and it must not hide real descent.
func TestSharedIDAmbiguityResolvesTowardDeclaredTier(t *testing.T) {
	dir := t.TempDir()
	wf := `# Workflow — proof

profile: go-service
template_version: 9
model_routing:
  primary: claude-fable-5
  routine: claude-sonnet-5
  mechanical: claude-sonnet-5      # deliberate: same id on two ordered tiers
  fallback: claude-opus-5
`
	writeAuditRepo(t, dir, wf, map[string]string{"I841": "routine", "I842": "primary"})
	tdir := t.TempDir()
	writeDispatchTranscript(t, dir, tdir, map[string]string{
		"I841": "claude-sonnet-5",
		"I842": "claude-sonnet-5",
	})
	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)
	if r := rows["I841"]; r.Verdict != VerdictMatch {
		t.Errorf("I841 verdict = %s (%s), want match — the declared tier is among the candidates", r.Verdict, r.Detail)
	}
	if r := rows["I842"]; r.Verdict != VerdictSilentDescent {
		t.Errorf("I842 verdict = %s (%s), want silent-descent — the highest ordered candidate (routine) is below primary", r.Verdict, r.Detail)
	}
	if !rep.Blocking() {
		t.Error("I842's descent must block — ambiguity must not hide real descent")
	}
}

// Fix round 1, I-1: a gen 6+ WORKFLOW.md that exists but has lost its
// model_routing block must not audit indistinguishably from a healthy repo.
// Verdicts stay faithful to dispatch-time resolution (embedded defaults, per
// D13), but the report says the spine-managed mirror is gone. A pre-gen-6
// stamp carries no such promise, so no warning fires there.
func TestMissingRoutingBlockWarnsOnGen6Plus(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, "# Workflow — proof\n\nprofile: go-service\ntemplate_version: 10\n",
		map[string]string{"I861": "primary"})
	tdir := t.TempDir()
	writeDispatchTranscript(t, dir, tdir, map[string]string{"I861": "fable"})
	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir})
	if err != nil {
		t.Fatal(err)
	}
	if r := rowsByID(t, rep)["I861"]; r.Verdict != VerdictMatch {
		t.Errorf("I861 verdict = %s (%s), want match — verdicts stay faithful to dispatch", r.Verdict, r.Detail)
	}
	found := false
	for _, w := range rep.Warnings {
		if strings.Contains(w, "no model_routing block") {
			found = true
		}
	}
	if !found {
		t.Errorf("gen-10 stamp over a missing block must warn, got %q", rep.Warnings)
	}

	old := t.TempDir()
	writeAuditRepo(t, old, "# Workflow — proof\n\nprofile: go-service\ntemplate_version: 5\n",
		map[string]string{"I862": "primary"})
	rep, err = Run(Options{RepoDir: old, ClaudeTranscriptsDir: tdir})
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range rep.Warnings {
		if strings.Contains(w, "no model_routing block") {
			t.Errorf("pre-gen-6 stamp must not warn about a missing block, got %q", rep.Warnings)
		}
	}
}

// Fix round 1, I-2: a deliberate Override matches by its exact on-disk id
// only. In a repo that pinned bespoke ids, a dispatch that ran the displaced
// default anyway — by full id, by alias, or by the displaced entry's
// historical id — is the drift the override exists to make visible: it
// reports unmapped, never a match through the shipped entry's lineage.
func TestOverriddenTierMatchesExactIDOnly(t *testing.T) {
	dir := t.TempDir()
	wf := `# Workflow — proof

profile: go-service
template_version: 9
model_routing:
  primary: bespoke-x
  fallback: bespoke-y
`
	writeAuditRepo(t, dir, wf, map[string]string{
		"I871": "primary", "I872": "primary", "I873": "primary", "I874": "fallback",
	})
	tdir := t.TempDir()
	writeDispatchTranscript(t, dir, tdir, map[string]string{
		"I871": "bespoke-x",       // the pinned id itself
		"I872": "fable",           // displaced default's alias
		"I873": "claude-fable-5",  // displaced default's full id
		"I874": "claude-opus-4-8", // displaced fallback's historical id
	})
	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)
	if r := rows["I871"]; r.Verdict != VerdictMatch {
		t.Errorf("I871 verdict = %s (%s), want match on the pinned id", r.Verdict, r.Detail)
	}
	for _, id := range []string{"I872", "I873", "I874"} {
		if r := rows[id]; r.Verdict != VerdictUnmappedDispatch {
			t.Errorf("%s verdict = %s (%s), want unmapped-dispatch — the displaced default must not match through the override's entry", id, r.Verdict, r.Detail)
		}
	}
	if rep.Blocking() {
		t.Error("unmapped drift is warn-level, never blocking")
	}
}

// Fix round 1, I-3: the ordered-beats-fallback leg of pickTier, the exact
// leg the shipped codex terra pair (routine + fallback sharing one id)
// exercises. Declared-tier readings stay charitable in both directions;
// on an above-tier ticket the ordered reading wins over the lateral one, so
// real descent cannot launder itself as an unexplained fallback.
func TestOrderedTierBeatsFallbackInSharedIDAmbiguity(t *testing.T) {
	dir := t.TempDir()
	wf := `# Workflow — proof

profile: go-service
template_version: 9
model_routing:
  primary: claude-fable-5
  routine: shared-tier-model
  mechanical: claude-haiku-4-5
  fallback: shared-tier-model
`
	writeAuditRepo(t, dir, wf, map[string]string{
		"I881": "routine", "I882": "fallback", "I883": "primary",
	})
	tdir := t.TempDir()
	writeDispatchTranscript(t, dir, tdir, map[string]string{
		"I881": "shared-tier-model",
		"I882": "shared-tier-model",
		"I883": "shared-tier-model",
	})
	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)
	if r := rows["I881"]; r.Verdict != VerdictMatch {
		t.Errorf("I881 verdict = %s (%s), want match — declared routine is among the candidates", r.Verdict, r.Detail)
	}
	if r := rows["I882"]; r.Verdict != VerdictMatch {
		t.Errorf("I882 verdict = %s (%s), want match — declared fallback is among the candidates", r.Verdict, r.Detail)
	}
	if r := rows["I883"]; r.Verdict != VerdictSilentDescent {
		t.Errorf("I883 verdict = %s (%s), want silent-descent — routine (ordered) beats fallback (lateral) and sits below primary", r.Verdict, r.Detail)
	}
	if !rep.Blocking() {
		t.Error("I883's descent must block")
	}
}

// ADR 0012 / D25: a FALLBACK record on the audited ticket moves fallback
// consultation before tier resolution. Same shared-id fixture as
// TestOrderedTierBeatsFallbackInSharedIDAmbiguity, but I883 now carries a
// FALLBACK record — the ordered (routine) reading must lose to the recorded
// lateral one, so the properly recorded refusal-rerun escalates with its
// quoted reason instead of standing as a false blocker. I881 also carries a
// FALLBACK record, even though its declared tier (routine) is itself among
// the candidates: an exact declared-tier match is never reinterpreted by a
// record — there is no ambiguity to resolve — so I881 must stay a plain
// match, not get bumped to escalated-with-reason noise.
func TestFallbackRecordWinsSharedIDAmbiguity(t *testing.T) {
	dir := t.TempDir()
	wf := `# Workflow — proof

profile: go-service
template_version: 9
model_routing:
  primary: claude-fable-5
  routine: shared-tier-model
  mechanical: claude-haiku-4-5
  fallback: shared-tier-model
`
	writeAuditRepo(t, dir, wf, map[string]string{
		"I881": "routine", "I882": "fallback", "I883": "primary",
	})
	if err := os.MkdirAll(filepath.Join(dir, ".superpowers", "sdd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".superpowers", "sdd", "progress.md"), []byte(
		"FALLBACK I881 reason: declared tier is itself the candidate, record must not override it\n"+
			"FALLBACK I883 reason: refused on primary, rerun on fallback\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tdir := t.TempDir()
	writeDispatchTranscript(t, dir, tdir, map[string]string{
		"I881": "shared-tier-model",
		"I882": "shared-tier-model",
		"I883": "shared-tier-model",
	})
	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)
	if r := rows["I881"]; r.Verdict != VerdictMatch {
		t.Errorf("I881 verdict = %s (%s), want match — declared routine is among the candidates and a FALLBACK record must not reinterpret an exact match", r.Verdict, r.Detail)
	}
	if r := rows["I882"]; r.Verdict != VerdictMatch {
		t.Errorf("I882 verdict = %s (%s), want match — declared fallback is among the candidates", r.Verdict, r.Detail)
	}
	r := rows["I883"]
	if r.Verdict != VerdictEscalatedWithReason {
		t.Errorf("I883 verdict = %s (%s), want escalated-with-reason — the FALLBACK record wins the shared-id ambiguity", r.Verdict, r.Detail)
	}
	if !strings.Contains(r.Detail, "refused on primary, rerun on fallback") {
		t.Errorf("I883 detail = %q, want it to quote the FALLBACK record's reason", r.Detail)
	}
	if rep.Blocking() {
		t.Error("I883 must not block — the record excuses the shared-id reading as a lateral fallback")
	}
}

// A repo with no WORKFLOW.md still audits: resolution falls back to the
// embedded defaults — the same answer dispatch-time resolution gives — and
// the report says so instead of failing or reporting everything unmapped.
func TestMissingWorkflowResolvesEmbeddedDefaultsWithWarning(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, "", map[string]string{"I851": "primary"})
	tdir := t.TempDir()
	writeDispatchTranscript(t, dir, tdir, map[string]string{"I851": "fable"})
	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir})
	if err != nil {
		t.Fatal(err)
	}
	if r := rowsByID(t, rep)["I851"]; r.Verdict != VerdictMatch {
		t.Errorf("I851 verdict = %s (%s), want match against embedded defaults", r.Verdict, r.Detail)
	}
	found := false
	for _, w := range rep.Warnings {
		if strings.Contains(w, "embedded defaults") {
			found = true
		}
	}
	if !found {
		t.Errorf("want a warning that routing resolved from embedded defaults, got %q", rep.Warnings)
	}
}
