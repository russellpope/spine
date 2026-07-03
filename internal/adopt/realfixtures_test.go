package adopt

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/doctor"
	"github.com/russellpope/spine/internal/update"
)

// copyTree copies testdata/<name> into a temp dir so --write can mutate it.
func copyTree(t *testing.T, name string) string {
	t.Helper()
	src := filepath.Join("testdata", name)
	dst := t.TempDir()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, raw, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func TestRealFixtureAdopts(t *testing.T) {
	cases := []struct {
		fixture, wantProfile string
	}{
		{"praxis", "go-service"},
		{"home-lab-admin", "infra"},
		{"obsidian-ep-vault", "knowledge"},
		{"moo-clone", "swift"},
	}
	for _, c := range cases {
		t.Run(c.fixture, func(t *testing.T) {
			dir := copyTree(t, c.fixture)
			res, err := Run(Options{Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			if res.Profile != c.wantProfile {
				t.Fatalf("profile=%q want %q", res.Profile, c.wantProfile)
			}
			if !res.Pending() {
				t.Fatal("fresh adopt must be pending")
			}
			if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
				t.Fatal(err)
			}
			// post-condition: doctor clean (info-only) and update a no-op
			findings, err := doctor.Run(dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range findings {
				if f.Severity == "warn" || f.Severity == "error" {
					t.Errorf("doctor %s %s %s: %s", f.ID, f.Severity, f.Path, f.Message)
				}
			}
			reports, err := update.Run(update.Options{Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			for _, r := range reports {
				if r.State != update.UpToDate {
					t.Errorf("update not no-op: %s", r.Path)
				}
			}
		})
	}
}

func TestPraxisClaimPreservesInvariants(t *testing.T) {
	dir := copyTree(t, "praxis")
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	if !strings.Contains(content, "spine:begin") {
		t.Error("marker block missing")
	}
	// the load-bearing praxis invariant must survive the claim verbatim
	if !strings.Contains(content, "github") || !strings.Contains(content, "NOT `origin`") {
		t.Error("praxis remote invariant lost in claim")
	}
}
