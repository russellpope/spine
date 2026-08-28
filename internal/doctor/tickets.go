package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// batchIDRe is the `batch:` shape the board writes at batch claim:
// <YYYY-MM-DD>-<4 alnum>#<n>, e.g. 2026-08-27-dhyg#1 (see the ledger
// convention in docs/issues/README.md).
var batchIDRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-[0-9A-Za-z]{4}#\d+$`)

// closedStatuses are the ledger's three terminal statuses: a ticket in one
// of them has been through the close protocol.
var closedStatuses = map[string]bool{"fixed": true, "wontfix": true, "superseded": true}

// ticketCheck is D13 (I106): the first per-ticket check. maikanban and the
// claude-team skill write four coordination keys onto ledger tickets
// (`batch`, `workspace`, `commits`, `review`); spine owns their schema, so
// doctor advises where a value contradicts its documented lifecycle. All
// findings are warn, never error — a stale key is coordination drift, not a
// broken repo, and every other key stays silent as before.
//
// Three conditions, each scoped to the statuses where the key can still be
// meaningful:
//
//   - a `workspace:` path that does not exist on disk — checked on every
//     ticket, since a vanished worktree is wrong at any status.
//   - a `workspace:` on a closed ticket — the presence itself is the
//     finding; the close protocol clears the key.
//   - a malformed `batch:` on an open or in-progress ticket. Closed tickets
//     are never checked for shape: a historical malformation is history, and
//     warning on it forever would be noise no one can act on.
func ticketCheck(dir string) []Finding {
	issuesDir := filepath.Join(dir, "docs", "issues")
	des, err := os.ReadDir(issuesDir)
	if err != nil {
		return nil // D1 covers structural absence
	}
	findings := []Finding{}
	for _, de := range des {
		name := de.Name()
		if de.IsDir() || !strings.HasSuffix(name, ".md") || !strings.HasPrefix(name, "I") {
			continue // README.md, _template.md and non-ticket files are not tickets
		}
		raw, err := os.ReadFile(filepath.Join(issuesDir, name))
		if err != nil {
			continue
		}
		rel := filepath.ToSlash(filepath.Join("docs", "issues", name))
		fm := frontmatter(string(raw))
		status := fm["status"]
		if ws := fm["workspace"]; ws != "" {
			if !filepath.IsAbs(ws) {
				findings = append(findings, Finding{"D13", "warn", rel, fmt.Sprintf(
					"%s: workspace: %s must be absolute — relative paths are not portable worktree identities", name, ws)})
			} else if _, err := os.Stat(ws); err != nil {
				findings = append(findings, Finding{"D13", "warn", rel, fmt.Sprintf(
					"%s: workspace: %s does not exist — the worktree is gone; clear the key or restore it", name, ws)})
			}
			if closedStatuses[status] {
				findings = append(findings, Finding{"D13", "warn", rel, fmt.Sprintf(
					"%s: workspace: is set on a %s ticket — the close protocol clears it", name, status)})
			}
		}
		if b := fm["batch"]; b != "" && (status == "open" || status == "in-progress") {
			if !batchIDRe.MatchString(b) {
				findings = append(findings, Finding{"D13", "warn", rel, fmt.Sprintf(
					"%s: batch: %q is malformed — want <YYYY-MM-DD>-<4 alnum>#<n> (e.g. 2026-08-27-dhyg#1)", name, b)})
			}
		}
	}
	return findings
}

// frontmatter parses a ticket's leading --- fence into its key/value pairs.
// A file without a fence yields an empty map, so it produces no findings.
func frontmatter(content string) map[string]string {
	out := map[string]string{}
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return out
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value := strings.TrimSpace(v)
		// Do not strip inline comments: valid batch IDs contain `#`, so this
		// parser deliberately diverges from YAML-style comment handling.
		if len(value) >= 2 && ((value[0] == '\'' && value[len(value)-1] == '\'') || (value[0] == '"' && value[len(value)-1] == '"')) {
			value = value[1 : len(value)-1]
		}
		out[strings.TrimSpace(k)] = value
	}
	return out
}
