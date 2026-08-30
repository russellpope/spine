// Package acceptance validates ticket-local APPROVED-UNTESTED records.
package acceptance

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Record is one valid approved-untested acceptance criterion.
type Record struct {
	Path      string
	Line      int
	Criterion string
	Date      string
	Approver  string
	Reference string
	Reason    string
}

// Problem is one malformed candidate line and all requirements it failed.
type Problem struct {
	Path   string
	Line   int
	Failed []string
}

// Message formats a problem for doctor and audit stages.
func (p Problem) Message() string {
	return fmt.Sprintf("line %d: invalid APPROVED-UNTESTED record: %s", p.Line, strings.Join(p.Failed, "; "))
}

// Summary contains every valid record and malformed candidate found.
type Summary struct {
	Records  []Record
	Problems []Problem
}

func (s Summary) ValidCount() int     { return len(s.Records) }
func (s Summary) InvalidCount() int   { return len(s.Problems) }
func (s Summary) CandidateCount() int { return s.ValidCount() + s.InvalidCount() }

var candidateRE = regexp.MustCompile(`^[ \t]*-[ \t]*\[[^]\r\n]*\].*APPROVED-UNTESTED`)
var canonicalRE = regexp.MustCompile(`^- \[ \] (.*?) -- APPROVED-UNTESTED ([^ ]*) by ([^[:space:]]*) ref: ([^[:space:]]*) reason: (.*)$`)

// ScanTicket scans one absolute or repository-relative ticket path.
func ScanTicket(repoRoot, ticketPath string) Summary {
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return Summary{}
	}
	absTicket := ticketPath
	if !filepath.IsAbs(absTicket) {
		absTicket = filepath.Join(absRoot, filepath.FromSlash(ticketPath))
	}
	rel, err := filepath.Rel(absRoot, absTicket)
	if err != nil {
		return Summary{}
	}
	rel = filepath.ToSlash(rel)
	f, err := os.Open(absTicket)
	if err != nil {
		return Summary{}
	}
	defer f.Close()

	var out Summary
	inAcceptance := false
	scanner := bufio.NewScanner(f)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := scanner.Text()
		if line == "## Acceptance criteria" {
			inAcceptance = true
		} else if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") {
			inAcceptance = false
		}
		if !candidateRE.MatchString(line) {
			continue
		}
		record, failed := parseLine(absRoot, rel, lineNo, line, inAcceptance)
		if len(failed) != 0 {
			out.Problems = append(out.Problems, Problem{Path: rel, Line: lineNo, Failed: failed})
			continue
		}
		out.Records = append(out.Records, record)
	}
	return out
}

// ScanAllTickets scans every eligible docs/issues/I*.md file.
func ScanAllTickets(repoRoot string) Summary {
	return scanTickets(repoRoot, nil)
}

// ScanTicketIDs scans eligible tickets whose frontmatter id exactly matches ids.
func ScanTicketIDs(repoRoot string, ids []string) Summary {
	wanted := make(map[string]bool, len(ids))
	for _, id := range ids {
		wanted[id] = true
	}
	return scanTickets(repoRoot, wanted)
}

func scanTickets(repoRoot string, wanted map[string]bool) Summary {
	issuesDir := filepath.Join(repoRoot, "docs", "issues")
	entries, err := os.ReadDir(issuesDir)
	if err != nil {
		return Summary{}
	}
	var paths []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "I") || !strings.HasSuffix(name, ".md") {
			continue
		}
		p := filepath.Join(issuesDir, name)
		if wanted != nil {
			raw, err := os.ReadFile(p)
			if err != nil || !wanted[frontmatterID(string(raw))] {
				continue
			}
		}
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var out Summary
	for _, p := range paths {
		part := ScanTicket(repoRoot, p)
		out.Records = append(out.Records, part.Records...)
		out.Problems = append(out.Problems, part.Problems...)
	}
	return out
}

func frontmatterID(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return ""
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		if strings.HasPrefix(line, "id:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "id:"))
		}
	}
	return ""
}

func parseLine(repoRoot, ticketPath string, lineNo int, line string, inAcceptance bool) (Record, []string) {
	record := Record{Path: ticketPath, Line: lineNo}
	var failed []string
	if !inAcceptance {
		failed = append(failed, "record must appear under ## Acceptance criteria")
	}
	parts := canonicalRE.FindStringSubmatch(line)
	if parts == nil {
		return record, append(failed, "record must match the canonical single-line grammar")
	}
	record.Criterion = strings.Trim(parts[1], " \t")
	record.Date = parts[2]
	record.Approver = parts[3]
	record.Reference = parts[4]
	record.Reason = strings.Trim(parts[5], " \t")
	if record.Criterion == "" {
		failed = append(failed, "criterion is required")
	}
	if !validDate(record.Date) {
		failed = append(failed, "date must be a real YYYY-MM-DD date")
	}
	if record.Approver == "" {
		failed = append(failed, "approver is required")
	}
	if record.Reference == "" {
		failed = append(failed, "reference is required")
	} else if reason := validateReference(repoRoot, record.Reference); reason != "" {
		failed = append(failed, reason)
	}
	if record.Reason == "" {
		failed = append(failed, "reason is required")
	}
	return record, failed
}

func validateReference(repoRoot, reference string) string {
	base, fragment, ok := strings.Cut(reference, "#")
	if !ok || base == "" || fragment == "" {
		return "reference must contain a nonempty path and fragment"
	}
	if strings.Contains(base, "\\") || path.IsAbs(base) || filepath.VolumeName(base) != "" || !strings.HasPrefix(base, "docs/") || path.Clean(base) != base {
		return "reference path must be a clean relative docs/ path using slash separators"
	}
	if !strings.HasSuffix(base, ".md") {
		return "reference path must end in .md"
	}
	name := path.Base(base)
	if len(name) < 11 || name[10] != '-' || !validDate(name[:10]) {
		return "reference basename must begin with a real YYYY-MM-DD date"
	}
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return "repository root cannot be resolved"
	}
	resolvedRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return "repository root cannot be resolved"
	}
	target := filepath.Join(absRoot, filepath.FromSlash(base))
	if !within(absRoot, target) {
		return "reference target escapes the repository root"
	}
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "reference target must exist"
	}
	if !within(resolvedRoot, resolved) {
		return "reference target escapes the resolved repository root"
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() {
		return "reference target must be a regular file"
	}
	return ""
}

func within(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func validDate(value string) bool {
	if len(value) != len("YYYY-MM-DD") {
		return false
	}
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}
