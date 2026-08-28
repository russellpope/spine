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

// isI114ContentDiffLine reports whether a unified-diff line carries I114's
// conscious current-generation template edit, or one of its blank separators.
func isI114ContentDiffLine(line string) bool {
	if len(line) == 0 || (line[0] != '+' && line[0] != '-') {
		return false
	}
	body := strings.TrimSpace(line[1:])
	return body == "" || i114ContentLines[body]
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
