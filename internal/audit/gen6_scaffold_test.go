package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/scaffold"
)

// Proof (I003, format frozen at gen 9): the bare-tier model_routing block
// every gen 6–9 repo carries on disk — the format the fleet still runs until
// the I039 sweep — parses with this package's WORKFLOW.md reader exactly as
// the supplement's "block shape" note requires: model_routing: at column 0
// plus two-space-indented `key: value  # comment` lines, all four tiers, no
// warnings. Inline fixture rather than scaffold.Init because gen 10's
// scaffold no longer renders this format (see the gen-10 test below).
func TestGen9BareTierModelRoutingParses(t *testing.T) {
	dir := t.TempDir()
	wf := `# Workflow — proof

profile: go-service
template_version: 9
model_routing:
  primary: claude-fable-5          # default thinker: design, judgment, orchestration, final review
  routine: claude-sonnet-5         # multi-step mechanical subagent roles
  mechanical: claude-haiku-4-5     # verbatim plan-transcription + single-file mechanical fixes ONLY
  fallback: claude-opus-4-8        # primary-refused or security-framed work
`
	if err := os.WriteFile(filepath.Join(dir, "WORKFLOW.md"), []byte(wf), 0o644); err != nil {
		t.Fatal(err)
	}
	var warnings []string
	mapping := readMapping(filepath.Join(dir, "WORKFLOW.md"), &warnings)
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings reading the gen-9 model_routing block: %v", warnings)
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

// I036 transition state (design D8, resolved by I037): the gen-10 scaffold
// renders the dotted flavor-axis mirror, which this package's pre-I037
// bare-tier reader deliberately cannot parse — the mapping comes back empty
// and the existing "no tier mapping found" warning fires. This is the loud,
// obviously-broken failure mode the dotted syntax was chosen for (spec user
// story 18); the silent alternative (nested blocks misparsed as bare tiers,
// last flavor winning) is the regression this test guards against. I037
// replaces readMapping with the shared resolver and rewrites this
// expectation.
func TestGen10ScaffoldMirrorFailsLoudlyUntilAuditConsolidation(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "go-service", "proof"); err != nil {
		t.Fatal(err)
	}
	var warnings []string
	mapping := readMapping(filepath.Join(dir, "WORKFLOW.md"), &warnings)
	if len(mapping) != 0 {
		t.Errorf("mapping = %v, want empty — a bare-tier reader must not half-parse the dotted mirror", mapping)
	}
	loud := false
	for _, w := range warnings {
		if strings.Contains(w, "no model_routing tier mapping found") {
			loud = true
		}
	}
	if !loud {
		t.Errorf("warnings = %v, want the loud no-mapping warning", warnings)
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
