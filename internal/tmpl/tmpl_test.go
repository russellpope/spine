package tmpl_test

import (
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/cursor"
	"github.com/russellpope/spine/internal/tmpl"
)

func TestVersionMatchesCurrentGeneration(t *testing.T) {
	if got := tmpl.Version(); got != 9 {
		t.Fatalf("Version() = %d, want 9", got)
	}
}

// I026: the tickets: grammar reference is documented in exactly two places
// — cursor.Grammar (internal/cursor) and the WORKFLOW.md.tmpl Stage-cursor
// section's indented Grammar-reference code block — and the ticket requires
// them to stay verbatim-identical. This proves it directly: extract the
// rendered template's indented `<!-- spine:cursor -->` ... `<!--
// /spine:cursor -->` block, dedent it by the code block's 4-space markdown
// indent, and byte-compare against cursor.Grammar. A future edit to either
// side without the other fails this test.
func TestCursorGrammarVerbatimInTemplate(t *testing.T) {
	rendered, err := tmpl.Render("current", "WORKFLOW.md.tmpl", tmpl.Values{
		Project: "demo", Profile: "rust",
		Reviewers: "rust-reviewer, security-review", Harness: "cli", Version: tmpl.Version(),
	})
	if err != nil {
		t.Fatal(err)
	}
	// The indented (4-space) form specifically: "spine:cursor" also appears
	// earlier in unindented prose describing the block, which is not the
	// Grammar reference itself.
	const openTag = "    <!-- spine:cursor -->"
	const closeTag = "    <!-- /spine:cursor -->"
	start := strings.Index(rendered, openTag)
	if start == -1 {
		t.Fatal("rendered WORKFLOW.md.tmpl has no indented spine:cursor Grammar reference block")
	}
	end := strings.Index(rendered[start:], closeTag)
	if end == -1 {
		t.Fatal("rendered WORKFLOW.md.tmpl Grammar reference block has no closing tag")
	}
	block := rendered[start : start+end+len(closeTag)]
	var dedented []string
	for _, line := range strings.Split(block, "\n") {
		dedented = append(dedented, strings.TrimPrefix(line, "    "))
	}
	got := strings.Join(dedented, "\n") + "\n"
	want := cursor.Grammar
	if got != want {
		t.Errorf("template Grammar reference block is not verbatim-identical to cursor.Grammar:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderFillsAllPlaceholders(t *testing.T) {
	for _, gen := range []string{"current", "gen0"} {
		got, err := tmpl.Render(gen, "WORKFLOW.md.tmpl", tmpl.Values{
			Project: "demo", Profile: "rust",
			Reviewers: "rust-reviewer, security-review", Harness: "cli", Version: 1,
		})
		if err != nil {
			t.Fatalf("%s: %v", gen, err)
		}
		for _, want := range []string{"# Workflow — demo", "profile: rust", "functional_harness: cli"} {
			if !strings.Contains(got, want) {
				t.Errorf("%s: missing %q", gen, want)
			}
		}
		if strings.Contains(got, "{{") {
			t.Errorf("%s: unfilled placeholder:\n%s", gen, got)
		}
	}
}

func TestCurrentWorkflowIsStamped(t *testing.T) {
	got, err := tmpl.Render("current", "WORKFLOW.md.tmpl", tmpl.Values{Profile: "rust", Version: 2})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "template_version: 2") {
		t.Error("current WORKFLOW template lacks template_version stamp")
	}
	if !strings.Contains(got, "primary: claude-fable-5") {
		t.Error("current WORKFLOW template lacks model_routing")
	}
}

func TestCurrentClaudeHasMarkers(t *testing.T) {
	got, err := tmpl.Render("current", "CLAUDE.md.tmpl", tmpl.Values{Project: "p", Profile: "rust", Version: 2})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "<!-- spine:begin v2 -->") || !strings.Contains(got, "<!-- spine:end -->") {
		t.Errorf("markers missing:\n%s", got)
	}
}

func TestDefaults(t *testing.T) {
	rev, harness, err := tmpl.Defaults("rust")
	if err != nil || rev != "rust-reviewer, security-review" || harness != "cli" {
		t.Fatalf("rust defaults = %q %q %v", rev, harness, err)
	}
	if _, _, err := tmpl.Defaults("nope"); err == nil {
		t.Fatal("unknown profile should error")
	}
	if len(tmpl.Profiles()) != 9 {
		t.Fatalf("Profiles() = %v, want 9 entries", tmpl.Profiles())
	}
}

func TestNewProfileDefaults(t *testing.T) {
	cases := map[string][2]string{
		"swift":     {"swift-reviewer, security-review", "framebuffer"},
		"infra":     {"security-review", "none"},
		"knowledge": {"", "none"},
	}
	for p, want := range cases {
		rev, harness, err := tmpl.Defaults(p)
		if err != nil || rev != want[0] || harness != want[1] {
			t.Errorf("Defaults(%q) = %q,%q,%v", p, rev, harness, err)
		}
	}
}

func TestProfileManifest(t *testing.T) {
	if d := tmpl.ProfileDirs("knowledge"); len(d) != 2 || d[0] != "docs/adr" || d[1] != "docs/handoffs" {
		t.Fatalf("knowledge dirs=%v", d)
	}
	if d := tmpl.ProfileDirs("go-service"); len(d) != 4 {
		t.Fatalf("go-service dirs=%v", d)
	}
	for _, rel := range []string{"docs/harness-interface.md", "docs/issues/README.md", "docs/issues/_template.md"} {
		if tmpl.ProfileOwns("knowledge", rel) {
			t.Errorf("knowledge should not own %s", rel)
		}
		if !tmpl.ProfileOwns("swift", rel) {
			t.Errorf("swift should own %s", rel)
		}
	}
	if !tmpl.ProfileOwns("knowledge", "WORKFLOW.md") || !tmpl.ProfileOwns("knowledge", "CLAUDE.md") || !tmpl.ProfileOwns("knowledge", "docs/adr/README.md") {
		t.Error("knowledge must own WORKFLOW.md, CLAUDE.md, docs/adr/README.md")
	}
}
