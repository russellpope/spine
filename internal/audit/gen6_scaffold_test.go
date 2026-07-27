package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/scaffold"
)

// Proof (I003, format frozen at gen 9; reworked in I037 to run through the
// audit's real boundary, since the audit's own block reader is gone): the
// bare-tier model_routing block every gen 6–9 repo carries on disk — the
// format the fleet still runs until the I039 sweep — reaches the audit
// through the shared resolver's transitional bare-key affordance. All four
// tiers resolve; a bespoke per-repo override matches (embedded defaults
// alone could never produce it, so this pins that the on-disk block was
// actually read); a prior shipped default still reads as the fallback id.
func TestGen9BareTierModelRoutingParses(t *testing.T) {
	dir := t.TempDir()
	wf := `# Workflow — proof

profile: go-service
template_version: 9
model_routing:
  primary: claude-fable-5          # default thinker: design, judgment, orchestration, final review
  routine: my-team-tuned-model     # deliberate per-repo override
  mechanical: claude-haiku-4-5     # verbatim plan-transcription + single-file mechanical fixes ONLY
  fallback: claude-opus-4-8        # primary-refused or security-framed work
`
	writeAuditRepo(t, dir, wf, map[string]string{
		"I501": "primary", "I502": "routine", "I503": "mechanical", "I504": "fallback",
	})
	tdir := t.TempDir()
	writeDispatchTranscript(t, dir, tdir, map[string]string{
		"I501": "claude-fable-5",
		"I502": "my-team-tuned-model",
		"I503": "claude-haiku-4-5",
		"I504": "claude-opus-4-8",
	})
	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Warnings) != 0 {
		t.Errorf("unexpected warnings auditing the gen-9 block: %v", rep.Warnings)
	}
	rows := rowsByID(t, rep)
	for _, id := range []string{"I501", "I502", "I503", "I504"} {
		if r := rows[id]; r.Verdict != VerdictMatch {
			t.Errorf("%s: verdict = %s (%s), want match", id, r.Verdict, r.Detail)
		}
	}
	if rep.Blocking() {
		t.Error("a clean gen-9 audit must not block")
	}
}

// I036 rendered the gen-10 dotted flavor mirror, which the audit's pre-I037
// bare-tier reader could not parse; TestGen10ScaffoldMirrorFailsLoudlyUntil-
// AuditConsolidation pinned that loud failure. Its premise died with I037:
// the audit now consumes the shared resolver, so a gen-10 scaffold resolves
// cleanly end to end — including a dotted per-repo override, which embedded
// defaults alone could never match, pinning that the dotted mirror is
// actually read (the nested-block silent misparse D8 warns about would fail
// this: the override lives under a claude. prefix a bare-tier reader would
// strip or skip).
func TestGen10ScaffoldMirrorResolvesThroughAudit(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "go-service", "proof"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "WORKFLOW.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(raw), "\n")
	replaced := false
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "claude.routine:") {
			lines[i] = "  claude.routine: my-pinned-routine-model"
			replaced = true
		}
	}
	if !replaced {
		t.Fatal("gen-10 scaffold renders no claude.routine row — mirror shape changed?")
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	for id, tier := range map[string]string{
		"I601": "primary", "I602": "routine", "I603": "mechanical", "I604": "fallback",
	} {
		gen6ProofTicket(t, dir, id, tier)
	}
	tdir := t.TempDir()
	writeDispatchTranscript(t, dir, tdir, map[string]string{
		"I601": "fable",
		"I602": "my-pinned-routine-model",
		"I603": "claude-haiku-4-5",
		"I604": "opus",
	})
	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: tdir})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Warnings) != 0 {
		t.Errorf("unexpected warnings auditing the gen-10 mirror: %v", rep.Warnings)
	}
	rows := rowsByID(t, rep)
	for _, id := range []string{"I601", "I602", "I603", "I604"} {
		if r := rows[id]; r.Verdict != VerdictMatch {
			t.Errorf("%s: verdict = %s (%s), want match", id, r.Verdict, r.Detail)
		}
	}
	if rep.Blocking() {
		t.Error("a clean gen-10 audit must not block")
	}
}

// gen6ProofTicket writes a minimal docs/issues ticket carrying id and the
// gen-6 annotation fields the template ships, tier set to the given value.
func gen6ProofTicket(t *testing.T, dir, id, tier string) {
	t.Helper()
	body := fmt.Sprintf(
		"---\nid: %s\ntitle: proof ticket\nseverity: med\nstatus: open\naffects: []\nblocked-by: []\nexecution-mode: subagent-driven\ntier: %s\neffort: medium\nrisk-triggers: []\nreview-tier: primary\n---\n\n## Problem\n\n## Fix\n",
		id, tier)
	if err := os.WriteFile(filepath.Join(dir, "docs", "issues", id+".md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// End-to-end proof: a gen-6 scaffolded repo with a ticket annotated at each
// tier runs cleanly through Run — every tier resolves (no unmapped-dispatch,
// no unannotated), confirming the rendered WORKFLOW.md and the ticket
// frontmatter fields the template ships both plug into the audit's real
// entry point, not just the internal mapping reader.
func TestGen6ScaffoldTicketsAuditCleanly(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "go-service", "proof"); err != nil {
		t.Fatal(err)
	}
	tiers := map[string]string{"I101": "primary", "I102": "routine", "I103": "mechanical", "I104": "fallback"}
	for id, tier := range tiers {
		gen6ProofTicket(t, dir, id, tier)
	}
	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: filepath.Join(dir, "no-such-transcripts-dir")})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Warnings) == 0 {
		t.Error("want a warning about the missing transcripts dir (expected — no harness in this proof)")
	}
	rows := rowsByID(t, rep)
	for id, tier := range tiers {
		r, ok := rows[id]
		if !ok {
			t.Fatalf("no row for %s", id)
		}
		if r.Tier != tier {
			t.Errorf("%s: Tier = %q, want %q (tier annotation not recognized)", id, r.Tier, tier)
		}
		// No transcript dir, so the only legitimate verdict is no-transcript
		// — anything else (unannotated, unmapped-dispatch) would mean the
		// tier value or mapping failed to parse.
		if r.Verdict != VerdictNoTranscript {
			t.Errorf("%s: Verdict = %s (%s), want no-transcript", id, r.Verdict, r.Detail)
		}
	}
	if rep.Blocking() {
		t.Error("a no-transcript-only report must never block")
	}
}

// Acceptance (I003): the gen-6 ticket template's annotation fields are
// optional — a plain bug issue written without any of them (the pre-gen-6
// field set only) is still a valid ticket row: reported as unannotated,
// never judged.
func TestGen6PlainBugIssueWithoutAnnotationsStaysValid(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "go-service", "proof"); err != nil {
		t.Fatal(err)
	}
	plain := "---\nid: I999\ntitle: plain bug, no routing annotations\nseverity: med\nstatus: open\naffects: []\nblocked-by: []\n---\n\n## Problem\n\n## Fix\n"
	if err := os.WriteFile(filepath.Join(dir, "docs", "issues", "I999.md"), []byte(plain), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: filepath.Join(dir, "no-such-transcripts-dir")})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)
	r, ok := rows["I999"]
	if !ok {
		t.Fatal("no row for I999 — a plain bug issue must still get a ticket row")
	}
	if r.Verdict != VerdictUnannotated {
		t.Errorf("I999: Verdict = %s (%s), want unannotated", r.Verdict, r.Detail)
	}
	if r.Tier != "" {
		t.Errorf("I999: Tier = %q, want empty (no annotation present)", r.Tier)
	}
}
