package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// issuesReadmeFixture stages a repo whose docs/issues/README.md is a
// verbatim historical render of templates/current/issues-README.md, on top
// of a pristine gen-8 WORKFLOW.md/CLAUDE.md pair (the README is a simple
// machine-owned file — planSimple compares it against the current template
// regardless of the repo's generation).
func issuesReadmeFixture(t *testing.T, fixture string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"WORKFLOW.md", "CLAUDE.md"} {
		raw, err := os.ReadFile(filepath.Join("testdata", "ccq-gen8", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(filepath.Join("testdata", "issues-readme", fixture))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs", "issues"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "issues", "README.md"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// Every historical issues-README render must refresh cleanly: the bullets
// those generations emitted and the current template does not are in
// supersededLines, so they read as machine-emitted rather than local edits
// (I065). Without the backfill each fixture skips as locally modified.
func TestIssuesReadmeHistoricalRendersUpdateCleanly(t *testing.T) {
	fixtures := []struct {
		name, file string
	}{
		// gen 6 as first shipped (I003): short status, tier and
		// review-tier bullets.
		{"gen6-initial", "gen6-initial-ee0d0b3.md"},
		// after the gen-6 review fixes: short status and tier bullets.
		{"pre-i046-tier", "pre-i046-tier-3dacdde.md"},
		// after I046 retired the short tier bullet: short status bullet.
		{"pre-superseded-status", "pre-superseded-status-c55ffb3.md"},
	}
	for _, f := range fixtures {
		t.Run(f.name, func(t *testing.T) {
			reports, err := Run(Options{Dir: issuesReadmeFixture(t, f.file)})
			if err != nil {
				t.Fatal(err)
			}
			r := report(t, reports, "docs/issues/README.md")
			if len(r.Unrecognized) > 0 {
				t.Errorf("pristine %s lines misread as local edits: %v", f.name, r.Unrecognized)
			}
			if r.State != Pending {
				t.Errorf("want Pending, got %v", r.State)
			}
		})
	}
}

// A --write run replaces the historical bullets with the current wording,
// exactly once each — no duplicate left behind.
func TestIssuesReadmeMigrationCarriesCurrentBullets(t *testing.T) {
	dir := issuesReadmeFixture(t, "gen6-initial-ee0d0b3.md")
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "docs", "issues", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	for _, want := range []string{
		"- `status` — open | in-progress | fixed | wontfix | superseded",
		"- `tier` — primary | routine | mechanical | fallback; the model tier the work is dispatched at.",
		"- `review-tier` — the tier review runs at; never below `tier`. Inline tickets carry",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("migrated README missing current bullet %q", want)
		}
	}
	for _, stale := range []string{
		"- `status` — open | in-progress | fixed | wontfix\n",
		"- `tier` — primary | routine | mechanical | fallback; the model tier the work is dispatched at\n",
		"- `review-tier` — the tier review runs at; never below `tier`\n",
	} {
		if strings.Contains(got, stale) {
			t.Errorf("migrated README still contains the superseded bullet %q", stale)
		}
	}
}

// Negative control: a genuine local edit to the same bullet still reads as
// unrecognized and skips the file — the backfill recognizes the exact
// historical spellings, not arbitrary content in that position.
func TestIssuesReadmeHandEditedBulletStaysUnrecognized(t *testing.T) {
	dir := issuesReadmeFixture(t, "pre-superseded-status-c55ffb3.md")
	path := filepath.Join(dir, "docs", "issues", "README.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := strings.Replace(string(raw),
		"- `status` — open | in-progress | fixed | wontfix\n",
		"- `status` — open | in-progress | fixed | wontfix | deferred\n", 1)
	if content == string(raw) {
		t.Fatal("fixture status bullet not found to replace")
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, reports, "docs/issues/README.md")
	if r.State != SkippedUnrecognized {
		t.Fatalf("hand-edited status bullet must skip the file, got state=%v unrec=%v", r.State, r.Unrecognized)
	}
	named := false
	for _, u := range r.Unrecognized {
		if strings.Contains(u, "deferred") {
			named = true
		}
	}
	if !named {
		t.Errorf("unrecognized lines %v do not name the hand edit", r.Unrecognized)
	}
}
