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
	findings := []Finding{}
	missingCore := false
	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			findings = append(findings, Finding{"D1", "error", rel, "missing — run spine init"})
			if rel == "WORKFLOW.md" {
				missingCore = true
			}
		}
	}
	if !missingCore {
		findings = append(findings, updateChecks(dir)...)
	}
	findings = append(findings, markerCheck(dir)...)
	findings = append(findings, superpowersCheck(dir)...)
	findings = append(findings, adrCheck(dir)...)
	return findings, nil
}

// updateChecks maps a dry-run of update onto D2 (stale) and D4 (unrecognized).
func updateChecks(dir string) []Finding {
	var findings []Finding
	reports, err := update.Run(update.Options{Dir: dir})
	if err != nil {
		return []Finding{{"D2", "error", "WORKFLOW.md", "update cannot run: " + err.Error()}}
	}
	for _, r := range reports {
		switch r.State {
		case update.Pending:
			msg := "behind template generation — run spine update"
			if r.Created {
				msg = "missing — spine update will create it"
			}
			findings = append(findings, Finding{"D2", "warn", r.Path, msg})
		case update.SkippedUnrecognized:
			msg := fmt.Sprintf("%d unrecognized local edit(s) in a machine-owned file — reconcile or spine update --force", len(r.Unrecognized))
			if r.Path == "CLAUDE.md" && len(r.Unrecognized) > 0 && strings.Contains(r.Unrecognized[0], "marker") {
				// --force deliberately cannot repair marker damage (unrecognized
				// with no newContent); the generic --force hint is actively wrong here.
				msg = "spine markers damaged — fix by hand (--force cannot repair)"
			}
			findings = append(findings, Finding{"D4", "warn", r.Path, msg})
		}
	}
	return findings
}

func markerCheck(dir string) []Finding {
	raw, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		return nil // D1 already reported it
	}
	content := string(raw)
	beginMarker, endMarker := "<!-- spine:begin", "<!-- spine:end -->"
	begins := strings.Count(content, beginMarker)
	ends := strings.Count(content, endMarker)
	switch {
	case begins == 0 && ends == 0:
		return []Finding{{"D3", "info", "CLAUDE.md", "no spine markers (legacy file) — spine update will claim it"}}
	case begins == 1 && ends == 1:
		// counts alone don't catch a swapped pair — compare positions too.
		if strings.Index(content, endMarker) < strings.Index(content, beginMarker) {
			return []Finding{{"D3", "error", "CLAUDE.md", "spine markers out of order — fix by hand"}}
		}
		return nil
	default:
		return []Finding{{"D3", "error", "CLAUDE.md",
			fmt.Sprintf("unbalanced spine markers (%d begin / %d end) — fix by hand", begins, ends)}}
	}
}

func superpowersCheck(dir string) []Finding {
	var findings []Finding
	for _, sub := range []string{"specs", "plans"} {
		glob := filepath.Join(dir, "docs", "superpowers", sub, "*.md")
		if m, _ := filepath.Glob(glob); len(m) > 0 {
			findings = append(findings, Finding{"D5", "info", "docs/superpowers/" + sub,
				fmt.Sprintf("%d artifact(s) in legacy location — new work goes to docs/specs/", len(m))})
		}
	}
	return findings
}

func adrCheck(dir string) []Finding {
	entries, err := adr.List(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil // no docs/adr — D1 covers structural absence
		}
		return []Finding{{"D6", "error", "docs/adr", "adr ledger unreadable: " + err.Error()}}
	}
	var findings []Finding
	seen := map[int]bool{}
	for _, e := range entries {
		if seen[e.ID] {
			findings = append(findings, Finding{"D6", "error", e.Path, fmt.Sprintf("duplicate ADR number %04d", e.ID)})
		}
		seen[e.ID] = true
		if !e.HasFrontMatter {
			findings = append(findings, Finding{"D6", "info", e.Path,
				"pre-spine ADR (no front matter) — spine conventions apply to new ADRs"})
			continue
		}
		if e.Status != "Accepted" && !strings.HasPrefix(e.Status, "Superseded by ") {
			findings = append(findings, Finding{"D6", "warn", e.Path, fmt.Sprintf("invalid status %q", e.Status)})
		}
	}
	return findings
}
