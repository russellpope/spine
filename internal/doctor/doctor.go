// Package doctor runs read-only workflow health checks (spec D1–D6).
package doctor

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/russellpope/spine/internal/adr"
	"github.com/russellpope/spine/internal/update"
)

// Finding is one doctor result; Severity is error | warn | info.
type Finding struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Path     string `json:"path"`
	Message  string `json:"message"`
}

var required = []string{
	"WORKFLOW.md", "CLAUDE.md", "docs/harness-interface.md",
	"docs/specs", "docs/adr", "docs/issues", "docs/handoffs",
}

// Run executes all checks. It never writes.
func Run(dir string) ([]Finding, error) {
	fs := []Finding{}
	missingCore := false
	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			fs = append(fs, Finding{"D1", "error", rel, "missing — run spine init"})
			if rel == "WORKFLOW.md" {
				missingCore = true
			}
		}
	}
	if !missingCore {
		fs = append(fs, updateChecks(dir)...)
	}
	fs = append(fs, markerCheck(dir)...)
	fs = append(fs, superpowersCheck(dir)...)
	fs = append(fs, adrCheck(dir)...)
	return fs, nil
}

// updateChecks maps a dry-run of update onto D2 (stale) and D4 (unrecognized).
func updateChecks(dir string) []Finding {
	var fs []Finding
	reports, err := update.Run(update.Options{Dir: dir})
	if err != nil {
		return []Finding{{"D2", "error", "WORKFLOW.md", "update cannot run: " + err.Error()}}
	}
	for _, r := range reports {
		switch r.State {
		case update.Pending:
			fs = append(fs, Finding{"D2", "warn", r.Path, "behind template generation — run spine update"})
		case update.SkippedUnrecognized:
			fs = append(fs, Finding{"D4", "warn", r.Path,
				fmt.Sprintf("%d unrecognized local edit(s) in a machine-owned file — reconcile or spine update --force", len(r.Unrecognized))})
		}
	}
	return fs
}

func markerCheck(dir string) []Finding {
	raw, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		return nil // D1 already reported it
	}
	begins := strings.Count(string(raw), "<!-- spine:begin")
	ends := strings.Count(string(raw), "<!-- spine:end -->")
	switch {
	case begins == 1 && ends == 1:
		return nil
	case begins == 0 && ends == 0:
		return []Finding{{"D3", "info", "CLAUDE.md", "no spine markers (legacy file) — spine update will claim it"}}
	default:
		return []Finding{{"D3", "error", "CLAUDE.md",
			fmt.Sprintf("unbalanced spine markers (%d begin / %d end) — fix by hand", begins, ends)}}
	}
}

func superpowersCheck(dir string) []Finding {
	var fs []Finding
	for _, sub := range []string{"specs", "plans"} {
		glob := filepath.Join(dir, "docs", "superpowers", sub, "*.md")
		if m, _ := filepath.Glob(glob); len(m) > 0 {
			fs = append(fs, Finding{"D5", "info", "docs/superpowers/" + sub,
				fmt.Sprintf("%d artifact(s) in legacy location — new work goes to docs/specs/", len(m))})
		}
	}
	return fs
}

func adrCheck(dir string) []Finding {
	entries, err := adr.List(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil // no docs/adr — D1 covers structural absence
		}
		return []Finding{{"D6", "error", "docs/adr", "adr ledger unreadable: " + err.Error()}}
	}
	var fs []Finding
	seen := map[int]bool{}
	for _, e := range entries {
		if seen[e.ID] {
			fs = append(fs, Finding{"D6", "error", e.Path, fmt.Sprintf("duplicate ADR number %04d", e.ID)})
		}
		seen[e.ID] = true
		if !e.HasFrontMatter {
			fs = append(fs, Finding{"D6", "info", e.Path,
				"pre-spine ADR (no front matter) — spine conventions apply to new ADRs"})
			continue
		}
		if e.Status != "Accepted" && !strings.HasPrefix(e.Status, "Superseded by ") {
			fs = append(fs, Finding{"D6", "warn", e.Path, fmt.Sprintf("invalid status %q", e.Status)})
		}
	}
	return fs
}
