package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// gen6ContentLines are the emitted-content changes gen 6 ships (I003), both
// sides of the diff: the gen-5 lines gen 6 drops or rewords (removed, "-")
// and the gen-6 lines that replace them (added, "+"). Diff lines are
// TrimSpace'd before lookup here (matching isGen6ContentDiffLine), so map
// keys carry no leading indent even though the rendered file does.
var gen6ContentLines = map[string]bool{
	// gen-5 model_routing lines dropped/reworded.
	"primary: claude-fable-5          # long-horizon, ambiguous, or first-shot-complex work (design, plan, implement, orchestrate)":             true,
	"fallback: claude-opus-4-8        # auto on stop_reason: refusal (cyber/bio/reasoning-extraction); also context/usage exhaustion":           true,
	"routine: claude-sonnet-5         # mechanical subagent roles: doc edits, plan-transcription implementers, build fixers, simple reviews":    true,
	"effort: high                       # default; xhigh for security-critical analysis + final verification; medium/low for routine subagents": true,
	"security_routing: quality-framing-opus-4-8": true,
	"Execution mode per plan: live-system mutation, secrets, or interactive steps -> inline with the human; otherwise subagent-driven.": true,
	"**Model:** see `WORKFLOW.md` `model_routing` (primary / fallback-on-refusal / routine; swappable).":                                true,

	// gen-6 model_routing lines and effort comment.
	"primary: claude-fable-5          # default thinker: design, judgment, orchestration, final review":  true,
	"routine: claude-sonnet-5         # multi-step mechanical subagent roles":                            true,
	"mechanical: claude-haiku-4-5     # verbatim plan-transcription + single-file mechanical fixes ONLY": true,
	"fallback: claude-opus-4-8        # primary-refused or security-framed work":                         true,
	"effort: high                       # tier default: primary=high, routine=medium, mechanical=low, fallback=high; xhigh reserved for final verification and security-critical passes; per-ticket effort: only on deviation": true,
	"**Model:** see `WORKFLOW.md` `model_routing` (primary / routine / mechanical / fallback; swappable).":                                                                                                                     true,

	// gen-6 "## Model routing" section.
	"## Model routing": true,
	"Artifacts (plans, tickets) reference tiers, never model ids — the mapping above is per-repo remappable (new model families, local models, other providers).":                                                                                                                                                                               true,
	"Escalation: dispatch may exceed a ticket's annotated tier or effort freely, WITH a recorded reason; dispatching below the annotation without a matching record is silent descent and fails the verify gate. Record grammar (exact — arrow is unspaced `->`; spaced arrows do not parse), one line each in `.superpowers/sdd/progress.md`:": true,
	"ESCALATION <ticket-id> <from-tier>-><to-tier> reason: <one line>": true,
	"ESCALATION <ticket-id> effort <from>-><to> reason: <one line>":    true,
	"FALLBACK <ticket-id> reason: <one line>":                          true,
	"A record excuses exactly its to-tier, nothing else. Any record not matching the grammar exactly excuses nothing — spaced arrows, missing `reason:`, missing tokens, all of it.":                                                                                                                                                                                                                                                                  true,
	"Reviewer floor: review-tier is never below tier; inline tickets carry `review-tier: n/a` — no per-task review cycle exists, verify-stage gates still apply. Risk triggers force primary-tier review — cross-task-integration, concurrency-subtle-state, security-surface, plan-flagged-ambiguity. The final whole-branch review and acceptance simulation always run primary. Reviewers re-run claims and demand raw transcripts at every tier.": true,
	"Fallback routing: proactive — security-framed work (attacker/exploit framing) routes to fallback from the first dispatch; security-touching but quality-framed work stays on its natural tier with the security-surface trigger. Reactive — on a primary refusal the orchestrator re-dispatches on fallback with quality framing, writes a FALLBACK record, and push-notifies the owner.":                                                        true,
	"Dispatch conventions the audit depends on: every subagent dispatch carries an explicit model (never inherit), and its description contains the ticket id token (the correlation contract). Verify stage: run `spine audit routing` (add `--transcripts <dir>` when the controller session runs in a different repo than the audited one) — reasoned escalations are advisory, silent descent blocks.":                                            true,

	// gen-6 "## Execution modes" section.
	"## Execution modes": true,
	"subagent-driven is the default for planned build work. ultracode is for work whose shape demands parallel orchestration (unknown-size discovery, cross-cutting audits, N-perspective verification); opt-in is granted by the owner's ticket approval, mid-build escalation is recommend-only. inline is the rare justified exception — tightly-coupled sequential chains, verbatim pre-specified diffs, live-system/secret/interactive steps — and requires a one-line justification in the ticket.": true,
}

// isGen6ContentDiffLine reports whether a unified-diff line carries one of
// the gen-6 content changes above, or is a bare added/removed blank line
// from the new sections' spacing.
func isGen6ContentDiffLine(line string) bool {
	if len(line) == 0 || (line[0] != '+' && line[0] != '-') {
		return false
	}
	body := strings.TrimSpace(line[1:])
	return body == "" || gen6ContentLines[body]
}

// The ccq-gen5 fixture is the gen-5 output of the ccq-gen4 fixture,
// generated BY GEN-5 CODE before any gen-6 template edit (I003). Gen 6
// rewrites the model_routing block, drops security_routing, and replaces
// the old execution-mode line with the full dispatch contract — a
// content-bearing bump like gen 5 was — so this lock pins the 5→6 diff to
// exactly the stamp plus the declared gen-6 content above, and requires
// that every superseded gen-5 line (registered in supersededLines) reads as
// machine-owned (Pending), never as a local edit (SkippedUnrecognized).
func TestGen5To6IsStampPlusDeclaredContent(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"WORKFLOW.md", "CLAUDE.md"} {
		raw, err := os.ReadFile(filepath.Join("testdata", "ccq-gen5", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, r := range reports {
		switch r.Path {
		case "WORKFLOW.md", "CLAUDE.md":
			seen[r.Path] = true
			if len(r.Unrecognized) > 0 {
				t.Errorf("%s: superseded machine lines misread as local edits: %v", r.Path, r.Unrecognized)
			}
			if r.State != Pending {
				t.Errorf("%s: want Pending, got %v", r.Path, r.State)
				continue
			}
			for _, line := range strings.Split(r.Diff, "\n") {
				if !strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "-") {
					continue
				}
				if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
					continue
				}
				if strings.Contains(line, "template_version") || strings.Contains(line, "spine:begin") {
					continue
				}
				if isGen6ContentDiffLine(line) {
					continue
				}
				t.Errorf("%s: unexpected changed line %q — 5→6 must be stamp plus declared gen-6 content only", r.Path, line)
			}
		}
	}
	for _, name := range []string{"WORKFLOW.md", "CLAUDE.md"} {
		if !seen[name] {
			t.Errorf("%s: never reported by Run — the lock did not exercise it", name)
		}
	}
}

// The gen-5 fixture's model_routing mapping and effort defaults must carry
// forward correctly: the mechanical tier appears, the mapping still
// contains all four tiers, security_routing is gone, and the word "auto"
// does not survive the migration.
func TestGen5To6MigrationCarriesFixtureForward(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"WORKFLOW.md", "CLAUDE.md"} {
		raw, err := os.ReadFile(filepath.Join("testdata", "ccq-gen5", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	wf, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"template_version: 6",
		"primary: claude-fable-5",
		"routine: claude-sonnet-5",
		"mechanical: claude-haiku-4-5",
		"fallback: claude-opus-4-8",
		"spine audit routing",
	} {
		if !strings.Contains(string(wf), want) {
			t.Errorf("migrated WORKFLOW.md missing %q", want)
		}
	}
	for _, unwanted := range []string{"security_routing", "auto"} {
		if strings.Contains(string(wf), unwanted) {
			t.Errorf("migrated WORKFLOW.md still contains %q", unwanted)
		}
	}
	cl, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(cl), "<!-- spine:begin v6 -->") {
		t.Error("migrated CLAUDE.md missing v6 marker")
	}
	if !strings.Contains(string(cl), "primary / routine / mechanical / fallback") {
		t.Error("migrated CLAUDE.md Model pointer missing the four-tier parenthetical")
	}
}

// A user-remapped model_routing tier id survives gen5->6 migration even
// when the hand edit used non-4-space padding before the trailing comment
// (keys.go's replaceValue always normalizes new writes to 4 spaces, but a
// human editing the file by hand has no reason to match that exactly). The
// remap is a sanctioned choice, not a local edit to flag or lose.
func TestGen5To6RemapWithNonstandardPaddingRecognized(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "ccq-gen5", "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	old := "  routine: claude-sonnet-5         # mechanical subagent roles: doc edits, plan-transcription implementers, build fixers, simple reviews\n"
	// 3 spaces before the comment, not the template's 4.
	remap := "  routine: local-llama-70b        # mechanical subagent roles: doc edits, plan-transcription implementers, build fixers, simple reviews\n"
	content := string(raw)
	if !strings.Contains(content, old) {
		t.Fatal("fixture line to remap not found")
	}
	content = strings.Replace(content, old, remap, 1)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "WORKFLOW.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	claudeRaw, err := os.ReadFile(filepath.Join("testdata", "ccq-gen5", "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), claudeRaw, 0o644); err != nil {
		t.Fatal(err)
	}

	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State == SkippedUnrecognized {
		t.Fatalf("3-space-padded remap misread as an unrecognized local edit: %v", wf.Unrecognized)
	}
	for _, u := range wf.Unrecognized {
		if strings.Contains(u, "local-llama-70b") {
			t.Errorf("remapped routine value flagged as unrecognized: %q", u)
		}
	}

	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "local-llama-70b") {
		t.Error("remapped routine value did not survive migration")
	}
}

// A CUSTOMIZED value of a key that gen 6 REMOVES (security_routing) is NOT
// a sanctioned remap: the key exists nowhere in the current template, so
// the value cannot be carried forward by Choices/setKey — accepting it as
// recognized would let a plain --write silently destroy it. It must stay a
// named unrecognized local edit (SkippedUnrecognized), and a plain update
// must leave the file untouched. Lock for the NEW-1 data-loss regression.
func TestGen5To6CustomizedRemovedKeyStaysSkipped(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "ccq-gen5", "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	if !strings.Contains(content, "security_routing: quality-framing-opus-4-8\n") {
		t.Fatal("fixture security_routing default line not found")
	}
	content = strings.Replace(content,
		"security_routing: quality-framing-opus-4-8\n",
		"security_routing: my-custom-value\n", 1)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "WORKFLOW.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	claudeRaw, err := os.ReadFile(filepath.Join("testdata", "ccq-gen5", "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), claudeRaw, 0o644); err != nil {
		t.Fatal(err)
	}

	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != SkippedUnrecognized {
		t.Fatalf("customized removed key must skip the file, got state=%v unrec=%v", wf.State, wf.Unrecognized)
	}
	named := false
	for _, u := range wf.Unrecognized {
		if strings.Contains(u, "security_routing: my-custom-value") {
			named = true
		}
	}
	if !named {
		t.Errorf("skip must name the customized line, got %v", wf.Unrecognized)
	}

	// A plain --write (no --force) must not migrate the file: the custom
	// value has nowhere to go in gen 6 and would be silently destroyed.
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "my-custom-value") {
		t.Error("plain update silently destroyed the customized removed-key value")
	}
}
