package adopt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/update"
)

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAdoptPraxisShape(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"go.mod":                          "module praxis\n",
		"CLAUDE.md":                       "## Repo invariants\n\n- remote is github, not origin\n",
		"docs/adr/0001-legacy.md":         "# 0001: legacy decision\n\nno front matter\n",
		"docs/superpowers/specs/a.md":     "old spec\n",
		"docs/decisions/2026-Q3-nonce.md": "quarterly recheck\n",
	})
	res, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if res.Profile != "go-service" {
		t.Fatalf("profile=%q", res.Profile)
	}
	if !res.Pending() {
		t.Fatal("fresh adopt must be pending")
	}
	joined := strings.Join(res.DirsToCreate, " ")
	for _, d := range []string{"docs/specs", "docs/issues", "docs/handoffs"} {
		if !strings.Contains(joined, d) {
			t.Errorf("missing dir %s in %q", d, joined)
		}
	}
	if strings.Contains(joined, "docs/adr") {
		t.Error("docs/adr exists; must not be in DirsToCreate")
	}
	var infoText string
	for _, i := range res.Infos {
		infoText += i.Path + ": " + i.Message + "\n"
	}
	for _, want := range []string{"docs/superpowers/specs", "pre-spine", "docs/decisions"} {
		if !strings.Contains(infoText, want) {
			t.Errorf("infos missing %q:\n%s", want, infoText)
		}
	}

	// apply, then idempotency: second adopt is a clean no-op
	res, err = Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "spine:begin") || !strings.Contains(string(raw), "Repo invariants") {
		t.Fatalf("claim failed: %q", raw)
	}
	res, err = Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if res.Pending() {
		for _, r := range res.Reports {
			t.Logf("%s state=%v", r.Path, r.State)
		}
		t.Fatal("adopted repo must be a no-op")
	}
	// post-condition: update agrees
	reports, err := update.Run(update.Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.State != update.UpToDate {
			t.Errorf("update not no-op: %s state=%v", r.Path, r.State)
		}
	}
}

// C1: a hand-authored docs/adr/README.md (praxis-style index) is preserved,
// not flagged — the plan must say so via an Info line, and must not count
// the file as pending.
func TestAdoptPreservesHandAuthoredADRReadme(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"go.mod": "module demo\n",
		"docs/adr/README.md": "# Architecture Decision Records\n\n" +
			"See the index below.\n\n| # | Decision |\n|---|---|\n| 0001 | Something |\n",
	})
	res, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range res.Reports {
		if r.Path == "docs/adr/README.md" && (r.State != update.UpToDate || !r.Preserved) {
			t.Fatalf("docs/adr/README.md state=%v preserved=%v", r.State, r.Preserved)
		}
	}
	var infoText string
	for _, i := range res.Infos {
		infoText += i.Path + ": " + i.Message + "\n"
	}
	if !strings.Contains(infoText, "docs/adr/README.md") || !strings.Contains(infoText, "preserved") {
		t.Errorf("infos missing preserved ADR README note:\n%s", infoText)
	}
}

// I1: multiple unrecognized docs/ entries must each get their own Info
// (path + message), not one comma-joined path — comma-joining defeats any
// consumer that keys on Info.Path for a single entry.
func TestAdoptUnknownDocsDirsEachGetOwnInfo(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"go.mod":                     "module demo\n",
		"docs/decisions/nonce.md":    "x\n",
		"docs/legacy-notes/nonce.md": "x\n",
	})
	res, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	var gotDecisions, gotLegacy bool
	for _, i := range res.Infos {
		if i.Path == "docs/decisions" {
			gotDecisions = true
			if !strings.Contains(i.Message, "not spine's") {
				t.Errorf("docs/decisions message = %q", i.Message)
			}
		}
		if i.Path == "docs/legacy-notes" {
			gotLegacy = true
			if !strings.Contains(i.Message, "not spine's") {
				t.Errorf("docs/legacy-notes message = %q", i.Message)
			}
		}
		if strings.Contains(i.Path, ",") {
			t.Errorf("Info.Path must be a single entry, got comma-joined: %q", i.Path)
		}
	}
	if !gotDecisions || !gotLegacy {
		t.Fatalf("want separate infos for both unknown dirs, got %#v", res.Infos)
	}
}

func TestAdoptUndetectableErrors(t *testing.T) {
	if _, err := Run(Options{Dir: t.TempDir()}); err == nil {
		t.Fatal("want detection error")
	}
}
