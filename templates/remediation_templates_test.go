package templates_test

import (
	"strings"
	"testing"

	"github.com/russellpope/spine/templates"
)

// AC (I087): both remediation templates are embedded and reachable through
// the templates FS, and carry the header/frontmatter fields the spec names.
func TestHitlistTemplateEmbedded(t *testing.T) {
	raw, err := templates.FS.ReadFile("current/hitlist.tmpl.md")
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	for _, want := range []string{
		"effort:", "round:", "dose: findings-only", "source run id:",
		"findings-only", "prescriptive", "raw-review",
		"`go@1/tskip`", "`go@1/mutate`",
		"file:line", "why it matters", "Do not regress",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("hitlist template missing %q", want)
		}
	}
	// The default dose is findings-without-fixes: the template must not
	// carry a fix section for the author to fill in.
	for _, forbidden := range []string{"## Fix", "### Fix", "fix:", "remedy:"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("hitlist template carries fix text %q — the findings-only dose has none", forbidden)
		}
	}
	// The no-fix-text rule is dose-scoped; the template says so.
	if !strings.Contains(got, "dose-scoped") {
		t.Error("hitlist template does not state that the no-fix-text rule is dose-scoped")
	}
}

// AC (I087): the round record's frontmatter carries the round-budget keys and
// its per-finding table keys on the results-contract code, with go@1/<check>
// example ids.
func TestRemediationRoundTemplateEmbedded(t *testing.T) {
	raw, err := templates.FS.ReadFile("current/remediation-round.tmpl.md")
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	if !strings.HasPrefix(got, "---\n") {
		t.Fatal("round template must open with a frontmatter block")
	}
	for _, want := range []string{
		"round: 1", "dose: findings-only", "hitlist:", "run_id:", "verdict:",
		"extension-ratified-by:",
		"| code | status | note |",
		"`go@1/tskip`", "`go@1/errsink`", "`go@1/mutate`",
		"open", "fixed", "regressed",
		"`prescriptive`", "`raw-review`",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("round template missing %q", want)
		}
	}
}
