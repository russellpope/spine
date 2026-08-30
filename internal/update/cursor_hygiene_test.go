package update

import (
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/tmpl"
)

const (
	outgoingTicketsGrammar = "tickets: I0NN | I0NN-I0MM | prefix I0"
	currentTicketsGrammar  = "tickets: I0NN | I0NN,I0MM[,...] | I0NN-I0MM | prefix I0"
)

var i114ContentLines = map[string]bool{
	"## Template authoring": true,
	"Any content-changing template edit appends its predecessors' dropped lines to the superseded set in the same change.": true,
	outgoingTicketsGrammar: true,
	currentTicketsGrammar:  true,
}

// i075ContentLines are the additive current-template workflow declaration
// contract. They are allowed beside the historical generation migrations:
// I075 changes no template generation and removes no predecessor prose.
var i075ContentLines = map[string]bool{
	"## Raw dispatch effort declarations":                                                true,
	"Every controlled launch records the exact raw declaration before launch:":           true,
	"harness=<raw execution vehicle> model=<exact selected ID> effort=<exact raw token>": true,
	"`harness`, `model`, and `effort` are a dispatch declaration, not evidence of":       true,
	"what a provider, gateway, or agent runtime used. A supplied effort token is":        true,
	"validated byte-exactly against the selected harness vocabulary; no ordering or":     true,
	"cross-harness comparison is implied. A retry is a new declaration and may use":      true,
	"a different token only with its own exact authorization record.":                    true,
	"The exact effort authorization grammar is:":                                         true,
	"ESCALATION <ticket-id> effort <from>-><to> reason: <one line>":                      true,
	"The arrow is unspaced; `<from>` and `<to>` are non-empty raw tokens without":        true,
	"spaces; `reason:` appears once with non-empty one-line text. It authorizes":         true,
	"only that ticket's declaration when its selected target effort is exactly":          true,
	"`<from>` and its declared effort is exactly `<to>`. A malformed, reversed,":         true,
	"or different-ticket pair authorizes nothing and does not alter model-tier":          true,
	"judgment.": true,
	"`spine audit routing` preserves its leading `ticket tier actual verdict detail`": true,
	"layout and appends declared-only fields. `declared-effort=-` means the":          true,
	"transcript did not record a declaration. `observed-effort=-` is always":          true,
	"unavailable at this stage; it is not runtime evidence.":                          true,
}

func isI075ContentDiffLine(line string) bool {
	if len(line) == 0 || line[0] != '+' {
		return false
	}
	return i075ContentLines[strings.TrimSpace(line[1:])]
}

// isI114ContentDiffLine reports whether a unified-diff line carries I114's
// conscious current-generation template edit, or one of its blank separators.
func isI114ContentDiffLine(line string) bool {
	if len(line) == 0 || (line[0] != '+' && line[0] != '-') {
		return false
	}
	body := strings.TrimSpace(line[1:])
	return body == "" || i114ContentLines[body] || isI075ContentDiffLine(line)
}

// outgoingGrammarWorkflow is a pristine current WORKFLOW render frozen at the
// grammar emitted immediately before I114's comma-list documentation change.
func outgoingGrammarWorkflow(t *testing.T) string {
	t.Helper()
	workflow, err := tmpl.Render("current", "WORKFLOW.md.tmpl", tmpl.Values{
		Project:   "fixture",
		Profile:   "library-cli",
		Reviewers: "go-reviewer, python-reviewer",
		Harness:   "cli",
		Version:   tmpl.Version(),
	})
	if err != nil {
		t.Fatal(err)
	}
	workflow = strings.Replace(workflow, currentTicketsGrammar, outgoingTicketsGrammar, 1)
	if !strings.Contains(workflow, outgoingTicketsGrammar) {
		t.Fatal("fixture is missing the outgoing tickets grammar")
	}
	return workflow
}

// I114: an otherwise pristine WORKFLOW carrying the exact outgoing grammar
// line must refresh normally. The test fails if the template does not advance
// the grammar or if the predecessor is not recognized as machine-emitted.
func TestOutgoingTicketsGrammarRefreshesCleanly(t *testing.T) {
	dir := writeRepo(t, outgoingGrammarWorkflow(t), "")
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != Pending {
		t.Fatalf("outgoing grammar state=%v, want Pending; unrecognized=%v", wf.State, wf.Unrecognized)
	}
	if len(wf.Unrecognized) != 0 {
		t.Fatalf("outgoing grammar misread as a local edit: %v", wf.Unrecognized)
	}
	if !strings.Contains(wf.newContent, currentTicketsGrammar) {
		t.Errorf("refreshed WORKFLOW missing current grammar %q", currentTicketsGrammar)
	}
}

// I114 negative control: recognition is exact. A local replacement of the
// predecessor grammar must still stop the ordinary update rather than being
// treated as a machine-emitted superseded line.
func TestHandEditedTicketsGrammarStaysUnrecognized(t *testing.T) {
	workflow := strings.Replace(outgoingGrammarWorkflow(t), outgoingTicketsGrammar,
		"tickets: locally-customized-grammar", 1)
	if !strings.Contains(workflow, "tickets: locally-customized-grammar") {
		t.Fatal("fixture did not receive the local grammar edit")
	}

	reports, err := Run(Options{Dir: writeRepo(t, workflow, "")})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	t.Logf("local grammar edit state=%v unrecognized=%v", wf.State, wf.Unrecognized)
	if wf.State != SkippedUnrecognized {
		t.Fatalf("local grammar edit state=%v, want SkippedUnrecognized; unrecognized=%v", wf.State, wf.Unrecognized)
	}
	if len(wf.Unrecognized) != 1 || !strings.Contains(wf.Unrecognized[0], "locally-customized-grammar") {
		t.Errorf("local grammar edit must be named as unrecognized, got %v", wf.Unrecognized)
	}
}
