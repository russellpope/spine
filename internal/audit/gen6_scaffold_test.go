package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/russellpope/spine/internal/scaffold"
)

// Proof (I003): the gen-6 model_routing block that scaffold.Init actually
// renders parses with this package's WORKFLOW.md reader exactly as the
// supplement's "block shape" note requires — model_routing: at column 0
// plus two-space-indented `key: value  # comment` lines — and resolves to
// all four tiers at their pinned ids, with no warnings.
func TestGen6ScaffoldModelRoutingParses(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "go-service", "proof"); err != nil {
		t.Fatal(err)
	}
	var warnings []string
	mapping := readMapping(filepath.Join(dir, "WORKFLOW.md"), &warnings)
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings reading the gen-6 model_routing block: %v", warnings)
	}
	want := map[string]string{
		"primary":    "claude-fable-5",
		"routine":    "claude-sonnet-5",
		"mechanical": "claude-haiku-4-5",
		"fallback":   "claude-opus-4-8",
	}
	if len(mapping) != len(want) {
		t.Fatalf("mapping = %v, want exactly the four tiers %v", mapping, want)
	}
	for tier, id := range want {
		if mapping[tier] != id {
			t.Errorf("mapping[%q] = %q, want %q", tier, mapping[tier], id)
		}
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
	rep, err := Run(dir, filepath.Join(dir, "no-such-transcripts-dir"))
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
	rep, err := Run(dir, filepath.Join(dir, "no-such-transcripts-dir"))
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
