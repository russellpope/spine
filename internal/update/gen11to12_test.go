package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var gen12ManagedFiles = []string{
	"WORKFLOW.md",
	"CLAUDE.md",
	"AGENTS.md",
	"docs/issues/README.md",
	"docs/issues/_template.md",
}

var gen12ContentLines = map[string]bool{
	"## Acceptance exceptions": true,
	"An applicable acceptance criterion that was consciously approved without a test stays unchecked and records the decision on one physical line under the ticket's exact column-0 `## Acceptance criteria` heading:": true,
	"- [ ] <criterion> -- APPROVED-UNTESTED <YYYY-MM-DD> by <approver> ref: <docs/YYYY-MM-DD-artifact.md#anchor> reason: <one-line reason>":                                                                             true,
	"The dated Markdown reference must be a clean repository-relative `docs/` path to a regular file. Spine verifies durable local provenance but does not authenticate the approver or resolve the fragment. A complete record is silent in `spine doctor`; an incomplete or invalid record is a D15 warning. `spine audit stages` scans only cursor-resolved tickets, keeps acceptance warnings nonblocking, and prints `acceptance: approved-untested=<valid-count> invalid=<invalid-count>` when the scoped tickets contain candidates. Ordinary unchecked criteria and tickets with no uppercase candidate keep their existing behavior.": true,
}

func isGen12ContentDiffLine(line string) bool {
	if len(line) == 0 || (line[0] != '+' && line[0] != '-') || strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
		return false
	}
	if line[0] == '-' {
		return isGen13ContentDiffLine(line)
	}
	body := strings.TrimSpace(line[1:])
	return body == "" || gen12ContentLines[body] || isGen13ContentDiffLine(line)
}

func TestGen11To12PristineUpdatesCleanly(t *testing.T) {
	dir := stageGen11Repo(t)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range gen12ManagedFiles {
		r := report(t, reports, name)
		if r.State != Pending || len(r.Unrecognized) != 0 {
			t.Errorf("%s: state=%v unrecognized=%v, want clean pending migration", name, r.State, r.Unrecognized)
		}
	}
}

func TestGen11To12WritesConventionAndIsIdempotent(t *testing.T) {
	dir := stageGen11Repo(t)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	assertFileContains(t, dir, "WORKFLOW.md", "template_version: 14", "## Acceptance exceptions", "APPROVED-UNTESTED", "provenance", "does not authenticate")
	assertFileContains(t, dir, "docs/issues/_template.md", "## Acceptance criteria", "WORKFLOW.md")
	assertFileContains(t, dir, "docs/issues/README.md", "approved without a test", "WORKFLOW.md")
	assertFileContains(t, dir, "AGENTS.md", "<!-- spine:begin v14 -->")
	assertFileContains(t, dir, "CLAUDE.md", "<!-- spine:begin v14 -->")
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.State != UpToDate {
			t.Errorf("second pass %s state=%v diff=%s", r.Path, r.State, r.Diff)
		}
	}
}

func TestGen11To12PreservesLocalEditRefusals(t *testing.T) {
	cases := map[string][2]string{
		"WORKFLOW.md":              {"## Template authoring", "## Local template authoring"},
		"CLAUDE.md":                {"**Mandatory gates:**", "**Local mandatory gates:**"},
		"AGENTS.md":                {"This file is read by **Codex**", "This file is locally edited for **Codex**"},
		"docs/issues/README.md":    {"# Issue / Bug Ledger — convention", "# Local ledger convention"},
		"docs/issues/_template.md": {"## Fix", "## Local fix"},
	}
	for name, replacement := range cases {
		t.Run(name, func(t *testing.T) {
			dir := stageGen11Repo(t)
			p := filepath.Join(dir, filepath.FromSlash(name))
			raw, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			changed := strings.Replace(string(raw), replacement[0], replacement[1], 1)
			if changed == string(raw) {
				t.Fatalf("fixture lacks %q", replacement[0])
			}
			if err := os.WriteFile(p, []byte(changed), 0o644); err != nil {
				t.Fatal(err)
			}
			reports, err := Run(Options{Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			r := report(t, reports, name)
			if r.State != SkippedUnrecognized || len(r.Unrecognized) == 0 {
				t.Fatalf("state=%v unrecognized=%v, want local-edit refusal", r.State, r.Unrecognized)
			}
		})
	}
}

func TestGen12ChangesAreAdditive(t *testing.T) {
	dir := stageGen11Repo(t)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range gen12ManagedFiles {
		r := report(t, reports, name)
		for _, line := range strings.Split(r.Diff, "\n") {
			if !strings.HasPrefix(line, "-") || strings.HasPrefix(line, "---") {
				continue
			}
			if isGen13ContentDiffLine(line) {
				continue
			}
			body := strings.TrimSpace(strings.TrimPrefix(line, "-"))
			if body == "template_version: 11" || body == "<!-- spine:begin v11 -->" {
				continue
			}
			t.Errorf("%s removes predecessor prose in additive generation 12: %q", name, line)
		}
	}
}

func stageGen11Repo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range gen12ManagedFiles {
		raw, err := os.ReadFile(filepath.Join("testdata", "spine-gen11", filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func assertFileContains(t *testing.T, dir, name string, wants ...string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range wants {
		if !strings.Contains(string(raw), want) {
			t.Errorf("%s missing %q", name, want)
		}
	}
}
