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

func TestAdoptUndetectableErrors(t *testing.T) {
	if _, err := Run(Options{Dir: t.TempDir()}); err == nil {
		t.Fatal("want detection error")
	}
}
