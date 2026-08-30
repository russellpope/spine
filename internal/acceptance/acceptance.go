// Package acceptance validates ticket-local APPROVED-UNTESTED records.
package acceptance

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
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

// ScanError is an explicit failure to open or fully read one ticket.
type ScanError struct {
	Path string
	Err  error
}

// Message formats a ticket scan failure for doctor and audit stages.
func (e ScanError) Message() string {
	return fmt.Sprintf("unable to read ticket: %v", e.Err)
}

// Message formats a problem for doctor and audit stages.
func (p Problem) Message() string {
	return fmt.Sprintf("line %d: invalid APPROVED-UNTESTED record: %s", p.Line, strings.Join(p.Failed, "; "))
}

// Summary contains every valid record and malformed candidate found.
type Summary struct {
	Records    []Record
	Problems   []Problem
	ScanErrors []ScanError
}

func (s Summary) ValidCount() int     { return len(s.Records) }
func (s Summary) InvalidCount() int   { return len(s.Problems) }
func (s Summary) CandidateCount() int { return s.ValidCount() + s.InvalidCount() }

var candidateRE = regexp.MustCompile(`^[ \t]*-[ \t]*\[[^]\r\n]*\].*APPROVED-UNTESTED`)

// ScanTicket scans one absolute or repository-relative ticket path.
func ScanTicket(repoRoot, ticketPath string) Summary {
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return Summary{ScanErrors: []ScanError{{Path: filepath.ToSlash(ticketPath), Err: err}}}
	}
	absTicket := ticketPath
	if !filepath.IsAbs(absTicket) {
		absTicket = filepath.Join(absRoot, filepath.FromSlash(ticketPath))
	}
	rel, err := filepath.Rel(absRoot, absTicket)
	if err != nil {
		return Summary{ScanErrors: []ScanError{{Path: filepath.ToSlash(ticketPath), Err: err}}}
	}
	rel = filepath.ToSlash(rel)
	f, err := os.Open(absTicket)
	if err != nil {
		return Summary{ScanErrors: []ScanError{{Path: rel, Err: err}}}
	}
	out := scanReader(absRoot, rel, f)
	if err := f.Close(); err != nil {
		out.ScanErrors = append(out.ScanErrors, ScanError{Path: rel, Err: err})
	}
	return out
}

func scanReader(repoRoot, ticketPath string, source io.Reader) Summary {
	out, _, _ := scanReaderWithIdentity(repoRoot, ticketPath, source)
	return out
}

func scanReaderWithIdentity(repoRoot, ticketPath string, source io.Reader) (Summary, string, bool) {
	var out Summary
	inAcceptance := false
	inFrontmatter := false
	var ticketID string
	identityKnown := false
	reader := bufio.NewReader(source)
	for lineNo := 1; ; lineNo++ {
		line, readErr := reader.ReadString('\n')
		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")
		trimmed := strings.TrimSpace(line)
		if lineNo == 1 && trimmed == "---" {
			inFrontmatter = true
		} else if inFrontmatter {
			if trimmed == "---" {
				inFrontmatter = false
			} else if strings.HasPrefix(line, "id:") {
				ticketID = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
				identityKnown = ticketID != ""
			}
		}
		if line != "" {
			inAcceptance = scanLine(repoRoot, ticketPath, lineNo, line, inAcceptance, &out)
		}
		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				out.ScanErrors = append(out.ScanErrors, ScanError{Path: ticketPath, Err: readErr})
			}
			break
		}
	}
	return out, ticketID, identityKnown
}

func scanLine(repoRoot, ticketPath string, lineNo int, line string, inAcceptance bool, out *Summary) bool {
	if line == "## Acceptance criteria" {
		inAcceptance = true
	} else if isH1OrH2Boundary(line) {
		inAcceptance = false
	}
	if !candidateRE.MatchString(line) {
		return inAcceptance
	}
	record, failed := parseLine(repoRoot, ticketPath, lineNo, line, inAcceptance)
	if len(failed) != 0 {
		out.Problems = append(out.Problems, Problem{Path: ticketPath, Line: lineNo, Failed: failed})
		return inAcceptance
	}
	out.Records = append(out.Records, record)
	return inAcceptance
}

func isH1OrH2Boundary(line string) bool {
	hashes := 0
	for hashes < len(line) && line[hashes] == '#' {
		hashes++
	}
	if hashes != 1 && hashes != 2 {
		return false
	}
	return len(line) == hashes || line[hashes] == ' ' || line[hashes] == '\t'
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
	return scanTicketsWithOpen(repoRoot, wanted, func(name string) (io.ReadCloser, error) {
		return os.Open(name)
	})
}

func scanTicketsWithOpen(repoRoot string, wanted map[string]bool, openFile func(string) (io.ReadCloser, error)) Summary {
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return Summary{ScanErrors: []ScanError{{Path: filepath.ToSlash(repoRoot), Err: err}}}
	}
	issuesDir := filepath.Join(absRoot, "docs", "issues")
	entries, err := os.ReadDir(issuesDir)
	if err != nil {
		if wanted != nil {
			return Summary{}
		}
		if !os.IsNotExist(err) {
			return Summary{ScanErrors: []ScanError{{Path: "docs/issues", Err: err}}}
		}
		return Summary{}
	}
	var paths []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "I") || !strings.HasSuffix(name, ".md") {
			continue
		}
		p := filepath.Join(issuesDir, name)
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var out Summary
	for _, p := range paths {
		rel, err := filepath.Rel(absRoot, p)
		if err != nil {
			if wanted == nil {
				out.ScanErrors = append(out.ScanErrors, ScanError{Path: filepath.ToSlash(p), Err: err})
			}
			continue
		}
		rel = filepath.ToSlash(rel)
		f, err := openFile(p)
		if err != nil {
			if wanted == nil {
				out.ScanErrors = append(out.ScanErrors, ScanError{Path: rel, Err: err})
			}
			continue
		}
		part, ticketID, identityKnown := scanReaderWithIdentity(absRoot, rel, f)
		if err := f.Close(); err != nil {
			part.ScanErrors = append(part.ScanErrors, ScanError{Path: rel, Err: err})
		}
		if wanted != nil && (!identityKnown || !wanted[ticketID]) {
			continue
		}
		appendSummary(&out, part)
	}
	return out
}

func appendSummary(dst *Summary, src Summary) {
	dst.Records = append(dst.Records, src.Records...)
	dst.Problems = append(dst.Problems, src.Problems...)
	dst.ScanErrors = append(dst.ScanErrors, src.ScanErrors...)
}

func parseLine(repoRoot, ticketPath string, lineNo int, line string, inAcceptance bool) (Record, []string) {
	record := Record{Path: ticketPath, Line: lineNo}
	var failed []string
	const (
		marker           = "APPROVED-UNTESTED"
		structuralMarker = " -- APPROVED-UNTESTED "
	)
	markerAt := strings.Index(line, marker)
	structuralCount := strings.Count(line, structuralMarker)
	if structuralAt := strings.Index(line, structuralMarker); structuralAt >= 0 {
		markerAt = structuralAt + len(" -- ")
	}
	beforeMarker := line[:markerAt]
	afterMarker := line[markerAt+len(marker):]

	if !strings.HasPrefix(line, "- [ ] ") {
		failed = append(failed, "record must start at column 0 with the exact - [ ] prefix")
	}
	if structuralCount > 1 {
		failed = append(failed, "record must contain exactly one ` -- APPROVED-UNTESTED ` structural marker")
	}
	criterionText := beforeMarker
	if checkboxEnd := strings.Index(criterionText, "]"); checkboxEnd >= 0 {
		criterionText = criterionText[checkboxEnd+1:]
	}
	hasSeparator := strings.HasSuffix(criterionText, " -- ")
	if hasSeparator {
		criterionText = strings.TrimSuffix(criterionText, " -- ")
	} else {
		criterionText = strings.Trim(criterionText, " \t-")
	}
	record.Criterion = strings.Trim(criterionText, " \t")
	if record.Criterion == "" {
		failed = append(failed, "criterion is required")
	}
	if !hasSeparator {
		failed = append(failed, "criterion and marker must be separated by exact ` -- ` bytes")
	}

	hasMarkerSpace := strings.HasPrefix(afterMarker, " ")
	fields := strings.TrimPrefix(afterMarker, " ")
	beforeReason, reason, hasReasonDelimiter := strings.Cut(fields, " reason: ")
	beforeReference, reference, hasReferenceDelimiter := strings.Cut(beforeReason, " ref: ")
	date, approver, hasByDelimiter := strings.Cut(beforeReference, " by ")
	record.Date = date
	record.Approver = approver
	record.Reference = reference
	record.Reason = strings.Trim(reason, " \t")

	if !validDate(record.Date) {
		failed = append(failed, "date must be a real YYYY-MM-DD date")
	}
	if !hasMarkerSpace {
		failed = append(failed, "marker must be followed by one ASCII space")
	}
	if !hasByDelimiter {
		failed = append(failed, "date and approver must be separated by exact ` by ` bytes")
	}
	if !hasReferenceDelimiter {
		failed = append(failed, "approver and reference must be separated by exact ` ref: ` bytes")
	}
	if !hasReasonDelimiter {
		failed = append(failed, "reference and reason must be separated by exact ` reason: ` bytes")
	}
	if record.Approver == "" {
		failed = append(failed, "approver is required")
	} else if strings.IndexFunc(record.Approver, unicode.IsSpace) >= 0 {
		failed = append(failed, "approver must be a whitespace-free token")
	}
	if record.Reference == "" {
		failed = append(failed, "reference is required")
	} else {
		if strings.IndexFunc(record.Reference, unicode.IsSpace) >= 0 {
			failed = append(failed, "reference must be a whitespace-free token")
		}
		failed = append(failed, validateReference(repoRoot, record.Reference)...)
	}
	if record.Reason == "" {
		failed = append(failed, "reason is required")
	}
	if !inAcceptance {
		failed = append(failed, "record must appear under ## Acceptance criteria")
	}
	return record, failed
}

func validateReference(repoRoot, reference string) []string {
	var failed []string
	base, fragment, ok := strings.Cut(reference, "#")
	if !ok || base == "" || fragment == "" {
		failed = append(failed, "reference must contain a nonempty path and fragment")
	}
	pathSafe := base != "" && !strings.Contains(base, "\\") && !path.IsAbs(base) && filepath.VolumeName(base) == "" && path.Clean(base) == base
	if base != "" && (!pathSafe || !strings.HasPrefix(base, "docs/")) {
		failed = append(failed, "reference path must be a clean relative docs/ path using slash separators")
	}
	if base != "" && !strings.HasSuffix(base, ".md") {
		failed = append(failed, "reference path must end in .md")
	}
	name := path.Base(base)
	if base != "" && (len(name) < 11 || name[10] != '-' || !validDate(name[:10])) {
		failed = append(failed, "reference basename must begin with a real YYYY-MM-DD date")
	}
	if !pathSafe {
		return failed
	}
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return append(failed, "repository root cannot be resolved")
	}
	resolvedRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return append(failed, "repository root cannot be resolved")
	}
	target := filepath.Join(absRoot, filepath.FromSlash(base))
	if !within(absRoot, target) {
		return append(failed, "reference target escapes the repository root")
	}
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		return append(failed, "reference target must exist")
	}
	if !within(resolvedRoot, resolved) {
		return append(failed, "reference target escapes the resolved repository root")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() {
		return append(failed, "reference target must be a regular file")
	}
	return failed
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
