package tmpl_test

import (
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/tmpl"
)

func TestVersionIsOne(t *testing.T) {
	if got := tmpl.Version(); got != 2 {
		t.Fatalf("Version() = %d, want 2", got)
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
	if len(tmpl.Profiles()) != 6 {
		t.Fatalf("Profiles() = %v, want 6 entries", tmpl.Profiles())
	}
}
