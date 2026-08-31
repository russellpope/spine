// Package yield reads explicit REVIEW records from a progress ledger. It does
// not inspect tickets, transcripts, model configuration, or Git history.
package yield

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const progressLedger = ".superpowers/sdd/progress.md"

type Scope string

const (
	ScopeTask  Scope = "task"
	ScopeFinal Scope = "final"
)

type Verdict string

const (
	VerdictAccepted   Verdict = "accepted"
	VerdictNeedsFixes Verdict = "needs-fixes"
)

type Record struct {
	Ticket     string
	Harness    string
	ModelID    string
	Tier       string
	Round      int
	Verdict    Verdict
	Scope      Scope
	Condition  string
	Line       int
	repository string
}

type Diagnostic struct {
	Repository string `json:"repository,omitempty"`
	Line       int    `json:"line,omitempty"`
	Message    string `json:"message"`
}

type Totals struct {
	ValidReviewLines              int `json:"valid_review_lines"`
	IgnoredIdentities             int `json:"ignored_identities"`
	Escalations                   int `json:"escalations"`
	Fallbacks                     int `json:"fallbacks"`
	FinalAccepted                 int `json:"final_accepted"`
	FinalNeedsFixes               int `json:"final_needs_fixes"`
	FinalUnattributableNeedsFixes int `json:"final_unattributable_needs_fixes"`
}

type Cell struct {
	Harness             string `json:"harness"`
	ModelID             string `json:"model_id"`
	Tier                string `json:"tier"`
	N                   int    `json:"n"`
	AcceptedFirstPass   int    `json:"accepted_first_pass"`
	NeedsFixesFirstPass int    `json:"needs_fixes_first_pass"`
	ReworkRounds        int    `json:"rework_rounds"`
	Rate                string `json:"accepted_first_pass_rate"`
	Confidence          string `json:"confidence"`
}

type RepositoryStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type Report struct {
	Scope        string             `json:"scope"`
	Totals       Totals             `json:"totals"`
	Cells        []Cell             `json:"cells"`
	Repositories []RepositoryStatus `json:"repositories,omitempty"`
	Diagnostics  []Diagnostic       `json:"diagnostics,omitempty"`
	childError   bool
}

// Invalid means REVIEW data was excluded because it was malformed, conflicted,
// or did not form a complete task sequence.
func (r Report) Invalid() bool { return r.Totals.IgnoredIdentities > 0 }

// Refused means at least one task-rate cell lacks the 20-sequence minimum. A
// report with no task sequence has zero evidence and is refused as well.
func (r Report) Refused() bool {
	if len(r.Cells) == 0 {
		return true
	}
	for _, cell := range r.Cells {
		if cell.N < 20 {
			return true
		}
	}
	return false
}

// ExitCode implements the public report contract: root errors are returned by
// Run, while a complete report exits 1 for refused or excluded data.
func (r Report) ExitCode() int {
	if r.Refused() || r.Invalid() || r.childError {
		return 1
	}
	return 0
}

type Options struct {
	Dir string
}

var ErrInvalidRoot = errors.New("invalid yield root")

func Run(opts Options) (Report, error) {
	if opts.Dir == "" {
		opts.Dir = "."
	}
	info, err := os.Stat(opts.Dir)
	if err != nil || !info.IsDir() {
		return Report{}, fmt.Errorf("%w: --dir", ErrInvalidRoot)
	}
	local, status := readRepository(opts.Dir, "")
	report := finalize(local, "repository")
	if status.Status == "error" {
		report.childError = true
		report.Diagnostics = append(report.Diagnostics, Diagnostic{Message: "progress ledger unreadable"})
	}
	return report, nil
}

type localResult struct {
	records     []Record
	totals      Totals
	diagnostics []Diagnostic
}

func (r *localResult) merge(other localResult) {
	r.records = append(r.records, other.records...)
	r.totals.ValidReviewLines += other.totals.ValidReviewLines
	r.totals.IgnoredIdentities += other.totals.IgnoredIdentities
	r.totals.Escalations += other.totals.Escalations
	r.totals.Fallbacks += other.totals.Fallbacks
	r.totals.FinalAccepted += other.totals.FinalAccepted
	r.totals.FinalNeedsFixes += other.totals.FinalNeedsFixes
	r.totals.FinalUnattributableNeedsFixes += other.totals.FinalUnattributableNeedsFixes
	r.diagnostics = append(r.diagnostics, other.diagnostics...)
}

func readRepository(dir, repository string) (localResult, RepositoryStatus) {
	path := filepath.Join(dir, filepath.FromSlash(progressLedger))
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return localResult{}, RepositoryStatus{Name: repository, Status: "missing-ledger"}
	}
	if err != nil {
		return localResult{}, RepositoryStatus{Name: repository, Status: "error"}
	}
	result := parseLedger(string(raw), repository)
	return result, RepositoryStatus{Name: repository, Status: "ok"}
}

func parseLedger(raw, repository string) localResult {
	result := localResult{}
	for lineNo, line := range strings.Split(raw, "\n") {
		if looksLikeReview(line) {
			record, diagnostic, ok := parseReview(lineNo+1, line)
			if !ok {
				result.totals.IgnoredIdentities++
				result.diagnostics = append(result.diagnostics, Diagnostic{Repository: repository, Line: lineNo + 1, Message: diagnostic})
				continue
			}
			record.repository = repository
			result.records = append(result.records, record)
			result.totals.ValidReviewLines++
			continue
		}
		if validEscalation(line) {
			result.totals.Escalations++
		}
		if validFallback(line) {
			result.totals.Fallbacks++
		}
	}
	return result
}

func looksLikeReview(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	return strings.HasPrefix(trimmed, "REVIEW") && (len(trimmed) == len("REVIEW") || trimmed[len("REVIEW")] == ' ')
}

func parseReview(lineNo int, line string) (Record, string, bool) {
	invalid := func() (Record, string, bool) { return Record{}, fmt.Sprintf("REVIEW line %d malformed", lineNo), false }
	if !strings.HasPrefix(line, "REVIEW ") || strings.TrimSpace(line) != line {
		return invalid()
	}
	fields := strings.Split(line, " ")
	for _, field := range fields {
		if field == "" {
			return invalid()
		}
	}
	if len(fields) != 8 && len(fields) != 9 {
		return invalid()
	}
	if fields[0] != "REVIEW" {
		return invalid()
	}
	rec := Record{Line: lineNo}
	if fields[1] != "-" && !validTicket(fields[1]) {
		return invalid()
	}
	rec.Ticket = fields[1]
	expected := []string{"harness:", "model:", "tier:", "round:", "verdict:", "scope:"}
	values := make([]string, len(expected))
	for i, prefix := range expected {
		value, ok := strings.CutPrefix(fields[i+2], prefix)
		if !ok || value == "" || strings.ContainsAny(value, "\t\r\n\"") {
			return invalid()
		}
		values[i] = value
	}
	rec.Harness, rec.ModelID, rec.Tier = values[0], values[1], values[2]
	round, err := parseRound(values[3])
	if err != nil {
		return invalid()
	}
	rec.Round = round
	if values[4] != string(VerdictAccepted) && values[4] != string(VerdictNeedsFixes) {
		return invalid()
	}
	rec.Verdict = Verdict(values[4])
	if values[5] != string(ScopeTask) && values[5] != string(ScopeFinal) {
		return invalid()
	}
	rec.Scope = Scope(values[5])
	if rec.Scope == ScopeTask {
		if len(fields) != 8 || rec.Ticket == "-" || !validKnown(rec.Harness, []string{"claude", "codex", "pi", "openweights"}) || rec.ModelID == "-" || !validKnown(rec.Tier, []string{"mechanical", "routine", "primary", "fallback"}) {
			return invalid()
		}
		return rec, "", true
	}
	if rec.Ticket != "-" {
		if len(fields) != 8 || !validMaybeKnown(rec.Harness, []string{"claude", "codex", "pi", "openweights"}) || !validMaybeKnown(rec.Tier, []string{"mechanical", "routine", "primary", "fallback"}) {
			return invalid()
		}
		return rec, "", true
	}
	if len(fields) != 9 || rec.Harness != "-" || rec.ModelID != "-" || rec.Tier != "-" || rec.Verdict != VerdictNeedsFixes {
		return invalid()
	}
	condition, ok := strings.CutPrefix(fields[8], "condition:")
	if !ok || condition == "" || strings.ContainsAny(condition, "\t\r\n\"") {
		return invalid()
	}
	rec.Ticket = ""
	rec.Condition = condition
	return rec, "", true
}

func validTicket(value string) bool {
	if len(value) < 2 || value[0] != 'I' {
		return false
	}
	for _, ch := range value[1:] {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func validKnown(value string, allowed []string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func validMaybeKnown(value string, allowed []string) bool {
	return value == "-" || validKnown(value, allowed)
}

func parseRound(value string) (int, error) {
	if value == "" || value == "0" || (len(value) > 1 && value[0] == '0') {
		return 0, errors.New("invalid round")
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return 0, errors.New("invalid round")
		}
	}
	round, err := strconv.Atoi(value)
	if err != nil || round < 1 {
		return 0, errors.New("invalid round")
	}
	return round, nil
}

func validEscalation(line string) bool {
	if strings.Count(line, "reason:") != 1 {
		return false
	}
	fields := strings.Split(line, " ")
	if len(fields) < 4 || fields[0] != "ESCALATION" || !validTicket(fields[1]) || fields[2] == "effort" {
		return false
	}
	from, to, ok := strings.Cut(fields[2], "->")
	if !ok || !validKnown(from, []string{"mechanical", "routine", "primary", "fallback"}) || !validKnown(to, []string{"mechanical", "routine", "primary", "fallback"}) {
		return false
	}
	return len(fields) > 4 && fields[3] == "reason:" && strings.TrimSpace(strings.Join(fields[4:], " ")) != ""
}

func validFallback(line string) bool {
	if strings.Count(line, "reason:") != 1 {
		return false
	}
	fields := strings.Split(line, " ")
	return len(fields) > 3 && fields[0] == "FALLBACK" && validTicket(fields[1]) && fields[2] == "reason:" && strings.TrimSpace(strings.Join(fields[3:], " ")) != ""
}

type identity struct {
	repository string
	scope      Scope
	ticket     string
	condition  string
	round      int
}

func finalize(local localResult, scope string) Report {
	report := Report{Scope: scope, Totals: local.totals, Diagnostics: local.diagnostics, Cells: []Cell{}}
	grouped := make(map[identity][]Record)
	for _, record := range local.records {
		key := identity{repository: record.repository, scope: record.Scope, ticket: record.Ticket, condition: record.Condition, round: record.Round}
		grouped[key] = append(grouped[key], record)
	}
	valid := make([]Record, 0, len(local.records))
	conflictedTasks := map[string]int{}
	for key, records := range grouped {
		if allEqual(records) {
			valid = append(valid, records[0])
			if len(records) > 1 {
				report.Diagnostics = append(report.Diagnostics, Diagnostic{Repository: key.repository, Line: records[1].Line, Message: "REVIEW duplicate identity counted once"})
			}
			continue
		}
		if key.scope == ScopeTask {
			conflictedTasks[key.repository+"\x00"+key.ticket] = records[0].Line
			continue
		}
		report.Totals.IgnoredIdentities++
		report.Diagnostics = append(report.Diagnostics, Diagnostic{Repository: key.repository, Line: records[0].Line, Message: "REVIEW conflicting identity excluded"})
	}
	taskSequences := map[string][]Record{}
	for _, record := range valid {
		if record.Scope == ScopeTask {
			key := record.repository + "\x00" + record.Ticket
			taskSequences[key] = append(taskSequences[key], record)
			continue
		}
		if record.Ticket == "" {
			report.Totals.FinalUnattributableNeedsFixes++
		} else if record.Verdict == VerdictAccepted {
			report.Totals.FinalAccepted++
		} else {
			report.Totals.FinalNeedsFixes++
		}
	}
	cellMap := map[string]*Cell{}
	for key, line := range conflictedTasks {
		report.Totals.IgnoredIdentities++
		report.Diagnostics = append(report.Diagnostics, Diagnostic{Line: line, Message: "REVIEW task sequence excluded"})
		delete(taskSequences, key)
	}
	for _, records := range taskSequences {
		if !validSequence(records) {
			report.Totals.IgnoredIdentities++
			report.Diagnostics = append(report.Diagnostics, Diagnostic{Line: firstLine(records), Message: "REVIEW task sequence excluded"})
			continue
		}
		sort.Slice(records, func(i, j int) bool { return records[i].Round < records[j].Round })
		first := records[0]
		cellKey := first.Harness + "\x00" + first.ModelID + "\x00" + first.Tier
		cell := cellMap[cellKey]
		if cell == nil {
			cell = &Cell{Harness: first.Harness, ModelID: first.ModelID, Tier: first.Tier}
			cellMap[cellKey] = cell
		}
		cell.N++
		if first.Verdict == VerdictAccepted {
			cell.AcceptedFirstPass++
		} else {
			cell.NeedsFixesFirstPass++
		}
		cell.ReworkRounds += records[len(records)-1].Round - 1
	}
	for _, cell := range cellMap {
		setConfidence(cell)
		report.Cells = append(report.Cells, *cell)
	}
	sort.Slice(report.Cells, func(i, j int) bool {
		if report.Cells[i].Harness != report.Cells[j].Harness {
			return report.Cells[i].Harness < report.Cells[j].Harness
		}
		if report.Cells[i].ModelID != report.Cells[j].ModelID {
			return report.Cells[i].ModelID < report.Cells[j].ModelID
		}
		return report.Cells[i].Tier < report.Cells[j].Tier
	})
	return report
}

func allEqual(records []Record) bool {
	for _, record := range records[1:] {
		if !sameRecord(record, records[0]) {
			return false
		}
	}
	return true
}

func sameRecord(left, right Record) bool {
	return left.Ticket == right.Ticket && left.Harness == right.Harness && left.ModelID == right.ModelID &&
		left.Tier == right.Tier && left.Round == right.Round && left.Verdict == right.Verdict &&
		left.Scope == right.Scope && left.Condition == right.Condition && left.repository == right.repository
}

func validSequence(records []Record) bool {
	if len(records) == 0 {
		return false
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Round < records[j].Round })
	for index, record := range records {
		if record.Round != index+1 {
			return false
		}
		if index > 0 && records[index-1].Verdict != VerdictNeedsFixes {
			return false
		}
		if index < len(records)-1 && record.Verdict == VerdictAccepted {
			return false
		}
	}
	return true
}

func firstLine(records []Record) int {
	if len(records) == 0 {
		return 0
	}
	return records[0].Line
}

func setConfidence(cell *Cell) {
	if cell.N < 20 {
		cell.Rate, cell.Confidence = "refused", "insufficient"
		return
	}
	cell.Rate = fmt.Sprintf("%.1f%%", float64(cell.AcceptedFirstPass)*100/float64(cell.N))
	if cell.N < 40 {
		cell.Confidence = "low-confidence"
		return
	}
	cell.Confidence = "stated"
}
