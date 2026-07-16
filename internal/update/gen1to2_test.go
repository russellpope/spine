package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The ccq fixture is that repo's actual gen-1 WORKFLOW.md and CLAUDE.md.
// Updating 1→2 must be exactly the stamp + marker-version diff — v2 ships
// no content edits to existing templates, so anything else here is a bug.
func TestGen1To2IsStampOnly(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"WORKFLOW.md", "CLAUDE.md"} {
		raw, err := os.ReadFile(filepath.Join("testdata", "ccq", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		switch r.Path {
		case "WORKFLOW.md", "CLAUDE.md":
			if r.State != Pending {
				t.Errorf("%s: want Pending, got %v", r.Path, r.State)
				continue
			}
			for _, line := range strings.Split(r.Diff, "\n") {
				if !strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "-") {
					continue
				}
				if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
					continue
				}
				if strings.Contains(line, "template_version") || strings.Contains(line, "spine:begin") {
					continue
				}
				if isGen5ContentDiffLine(line) { // gen 5's conscious content edit; see gen4to5_test.go
					continue
				}
				if isGen6ContentDiffLine(line) { // gen 6's conscious content edit; see gen5to6_test.go
					continue
				}
				if isGen8ContentDiffLine(line) { // gen 8's conscious content edit; see gen7to8_test.go
					continue
				}
				if isGen9ContentDiffLine(line) { // gen 9's conscious content edit; see gen8to9_test.go
					continue
				}
				t.Errorf("%s: unexpected changed line %q — gen 1→2 must be stamp-only", r.Path, line)
			}
		}
	}
}
