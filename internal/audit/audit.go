// Package audit implements the core of `spine audit routing`: a
// deterministic diff of a scaffolded repo's declared per-ticket model-tier
// annotations against the models the harness transcripts say were actually
// used. The boundary is the pure function Run (repo dir + transcript dir in,
// Report out); the CLI in cmd/spine is a thin printer over it.
//
// Inputs, all read-only:
//   - docs/issues/*.md frontmatter: id plus the optional annotation fields
//     tier / execution-mode / effort / risk-triggers / review-tier.
//   - WORKFLOW.md `model_routing:` block: tier -> full model id for
//     primary / routine / mechanical / fallback, parsed tolerantly
//     (inline comments stripped, unknown keys ignored).
//   - .superpowers/sdd/progress.md: one-line ESCALATION / FALLBACK records.
//   - The harness transcript dir: <dir>/*.jsonl session records plus
//     <dir>/<session>/subagents/agent-*.jsonl (+ sibling .meta.json). This
//     format is undocumented and may shift: any parse failure, missing dir,
//     or unrecognized shape degrades to a Report warning, never an error.
//
// Design-latitude choices (the ticket leaves these open; pinned here):
//   - Every ticket file in docs/issues with a frontmatter id gets a row;
//     files without an id and files starting with "_" (templates) or named
//     README.md are ignored. Tickets whose tier annotation is not one of
//     the four known tiers are reported as unannotated (detail names the
//     unknown value), never judged.
//   - Model evidence per dispatch: the linked subagent transcript's
//     assistant model ids when one exists (linked via the meta.json
//     toolUseId, or its description's ticket token); otherwise the
//     dispatch's model alias; a dispatch with neither contributes nothing.
//     Main-session assistant models are never ticket evidence — inline
//     execution is out of the audit's scope by design.
//   - Alias/id -> tier: a token maps to a tier when it equals the mapped id
//     or the mapped id contains it (alias case, e.g. claude-sonnet-5 ~
//     "sonnet"). When a token maps to several ordered tiers, the reading
//     closest to a non-verdict wins: declared tier if present, else the
//     highest — degradation must not manufacture descent. A token mapping
//     only to fallback is lateral: covered by a FALLBACK record ->
//     escalated-with-reason; covered by a `tier: fallback` annotation ->
//     match; otherwise the warn-level unexplained-fallback. (A fallback id
//     shared with an ordered tier below the annotation resolves through the
//     ordered path and can therefore still be silent-descent.)
//   - Any model-tier ESCALATION record for a ticket counts as its recorded
//     reason, up or down — silent-descent is strictly deviation-below with
//     no record. Deviation above with no record is the warn-level
//     escalated-no-reason (not blocking: quality went up, but the contract
//     says escalations carry reasons). Effort ESCALATION records are
//     accepted grammar but are not model evidence.
//   - A ticket's verdict is its worst token's verdict: silent-descent >
//     unmapped-dispatch > unexplained-fallback > escalated-no-reason >
//     escalated-with-reason > match.
//   - A missing docs/issues dir is the one hard error (not a scaffolded
//     repo — CLI usage error); everything else degrades to warnings.
package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Verdict classifies one ticket's declared-vs-actual routing.
type Verdict string

// Verdict values, worst first.
const (
	VerdictSilentDescent       Verdict = "silent-descent"        // blocking
	VerdictUnmappedDispatch    Verdict = "unmapped-dispatch"     // warn
	VerdictUnexplainedFallback Verdict = "unexplained-fallback"  // warn
	VerdictEscalatedNoReason   Verdict = "escalated-no-reason"   // warn
	VerdictNoTranscript        Verdict = "no-transcript"         // warn
	VerdictEscalatedWithReason Verdict = "escalated-with-reason" // advisory
	VerdictMatch               Verdict = "match"
	VerdictUnannotated         Verdict = "unannotated" // informational
)

// severity orders verdicts for worst-token aggregation; higher is worse.
var severity = map[Verdict]int{
	VerdictMatch:               0,
	VerdictEscalatedWithReason: 1,
	VerdictEscalatedNoReason:   2,
	VerdictUnexplainedFallback: 3,
	VerdictUnmappedDispatch:    4,
	VerdictSilentDescent:       5,
}

// TicketRow is one ticket's audit outcome.
type TicketRow struct {
	ID      string
	Tier    string // declared tier annotation; "" if absent
	Actuals []string
	Verdict Verdict
	Detail  string
}

// DispatchInfo is an informational, never-judged dispatch record.
type DispatchInfo struct {
	Description string
	Model       string
}

// Report is the audit result.
type Report struct {
	Tickets   []TicketRow
	Unmatched []DispatchInfo
	Warnings  []string
}

// Blocking reports whether any ticket carries a blocking verdict.
func (r Report) Blocking() bool {
	for _, t := range r.Tickets {
		if t.Verdict == VerdictSilentDescent {
			return true
		}
	}
	return false
}

// tier order: mechanical < routine < primary; fallback is lateral (rank 0).
var tierRank = map[string]int{"mechanical": 1, "routine": 2, "primary": 3, "fallback": 0}

// Run audits repoDir's declared routing against the transcript records in
// transcriptsDir. Transcript trouble of any kind degrades to Warnings; the
// only error is a repo without docs/issues.
func Run(repoDir, transcriptsDir string) (Report, error) {
	var rep Report
	tickets, err := readTickets(filepath.Join(repoDir, "docs", "issues"))
	if err != nil {
		return Report{}, err
	}
	mapping := readMapping(filepath.Join(repoDir, "WORKFLOW.md"), &rep.Warnings)
	ledger := readLedger(filepath.Join(repoDir, ".superpowers", "sdd", "progress.md"))
	dispatches, agents := readTranscripts(transcriptsDir, &rep.Warnings)

	evidence := map[string][]string{} // ticket id -> raw model tokens
	claimed := map[int]bool{}         // dispatch index -> matched a ticket
	linked := map[string]bool{}       // toolUseID -> a subagent transcript carries models
	for _, a := range agents {
		if a.toolUseID != "" && len(a.models) > 0 {
			linked[a.toolUseID] = true
		}
	}
	matches := func(d dispatch, id string) bool {
		return containsToken(d.description, id) || containsToken(firstLine(d.prompt), id)
	}
	for _, t := range tickets {
		for i, d := range dispatches {
			if !matches(d, t.id) {
				continue
			}
			claimed[i] = true
			if linked[d.toolUseID] {
				continue // the subagent transcript below is the actual
			}
			if d.model != "" {
				evidence[t.id] = append(evidence[t.id], d.model)
			}
		}
		for _, a := range agents {
			use := containsToken(a.description, t.id)
			for _, d := range dispatches {
				if use {
					break
				}
				use = d.toolUseID != "" && d.toolUseID == a.toolUseID && matches(d, t.id)
			}
			if use {
				evidence[t.id] = append(evidence[t.id], a.models...)
			}
		}
	}
	for i, d := range dispatches {
		if !claimed[i] {
			rep.Unmatched = appendUnmatched(rep.Unmatched, DispatchInfo{Description: d.description, Model: d.model})
		}
	}

	for _, t := range tickets {
		row := TicketRow{ID: t.id, Tier: t.tier, Actuals: dedupSorted(evidence[t.id])}
		row.Verdict, row.Detail = judge(t, row.Actuals, mapping, ledger)
		rep.Tickets = append(rep.Tickets, row)
	}
	sort.Slice(rep.Tickets, func(i, j int) bool { return rep.Tickets[i].ID < rep.Tickets[j].ID })
	return rep, nil
}

// judge decides one ticket's verdict from its declared tier, its observed
// model tokens, the tier mapping, and the ledger records.
func judge(t ticket, actuals []string, mapping map[string]string, l ledger) (Verdict, string) {
	if t.tier == "" {
		return VerdictUnannotated, "no tier annotation — not judged"
	}
	if _, known := tierRank[t.tier]; !known {
		return VerdictUnannotated, fmt.Sprintf("unknown tier %q — not judged", t.tier)
	}
	if len(actuals) == 0 {
		return VerdictNoTranscript, "no dispatch or transcript evidence found"
	}
	verdict, detail := VerdictMatch, ""
	worse := func(v Verdict, d string) {
		if severity[v] > severity[verdict] {
			verdict, detail = v, d
		}
	}
	for _, token := range actuals {
		v, d := judgeToken(token, t, mapping, l)
		worse(v, d)
	}
	return verdict, detail
}

// judgeToken classifies a single observed model token against the ticket's
// declared tier.
func judgeToken(token string, t ticket, mapping map[string]string, l ledger) (Verdict, string) {
	tiers := tiersOf(token, mapping)
	if len(tiers) == 0 {
		return VerdictUnmappedDispatch, fmt.Sprintf("%s maps to no tier in model_routing", token)
	}
	actual := pickTier(tiers, t.tier)
	if actual == t.tier {
		return VerdictMatch, ""
	}
	if actual == "fallback" { // lateral, never descent (see package doc)
		if reason, ok := l.fallback[t.id]; ok {
			return VerdictEscalatedWithReason, fmt.Sprintf("%s (fallback) — FALLBACK reason: %s", token, reason)
		}
		return VerdictUnexplainedFallback, fmt.Sprintf("%s (fallback) without a FALLBACK record or fallback annotation", token)
	}
	if reason, ok := l.escalation[t.id]; ok {
		return VerdictEscalatedWithReason, fmt.Sprintf("%s (%s) vs declared %s — ESCALATION reason: %s", token, actual, t.tier, reason)
	}
	if t.tier != "fallback" && tierRank[actual] < tierRank[t.tier] {
		return VerdictSilentDescent, fmt.Sprintf("%s (%s) below declared %s with no ESCALATION record", token, actual, t.tier)
	}
	return VerdictEscalatedNoReason, fmt.Sprintf("%s (%s) above declared %s with no ESCALATION record", token, actual, t.tier)
}

// tiersOf resolves a model token to every tier it could mean: exact id
// match, or the alias case where the mapped id contains the token.
func tiersOf(token string, mapping map[string]string) []string {
	var tiers []string
	for tier, id := range mapping {
		if id == token || strings.Contains(id, token) {
			tiers = append(tiers, tier)
		}
	}
	sort.Strings(tiers)
	return tiers
}

// pickTier chooses the reading of an ambiguous token: the declared tier if
// it is among the candidates, else the highest-ranked ordered candidate,
// else fallback. Ambiguity must not manufacture a verdict.
func pickTier(tiers []string, declared string) string {
	best := ""
	for _, tier := range tiers {
		if tier == declared {
			return tier
		}
		if tier == "fallback" {
			if best == "" {
				best = tier
			}
			continue
		}
		if best == "" || tierRank[tier] > tierRank[best] {
			best = tier
		}
	}
	return best
}

// --- repo inputs ---

type ticket struct {
	id   string
	tier string
}

// readTickets parses docs/issues frontmatter. Only the id is required for a
// row; README.md and _-prefixed files are not tickets.
func readTickets(dir string) ([]ticket, error) {
	des, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("docs/issues unreadable (not a scaffolded repo?): %w", err)
	}
	var tickets []ticket
	for _, de := range des {
		name := de.Name()
		if de.IsDir() || !strings.HasSuffix(name, ".md") || strings.HasPrefix(name, "_") || name == "README.md" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		fm := frontmatter(string(raw))
		if fm["id"] == "" {
			continue
		}
		tickets = append(tickets, ticket{id: fm["id"], tier: fm["tier"]})
	}
	return tickets, nil
}

// frontmatter parses the `key: value` lines between the leading --- fence
// pair. Values keep no inline comments; nested structure is out of scope.
func frontmatter(content string) map[string]string {
	fm := map[string]string{}
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return fm
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fm[strings.TrimSpace(k)] = stripComment(v)
	}
	return fm
}

// readMapping extracts tier -> model id from WORKFLOW.md's model_routing
// block. Absence of the file or the block degrades to a warning.
func readMapping(path string, warnings *[]string) map[string]string {
	raw, err := os.ReadFile(path)
	if err != nil {
		*warnings = append(*warnings, "WORKFLOW.md unreadable — every dispatch will report unmapped: "+err.Error())
		return nil
	}
	mapping := map[string]string{}
	inBlock := false
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "model_routing:") {
			inBlock = true
			continue
		}
		if !inBlock {
			continue
		}
		if !strings.HasPrefix(line, "  ") || strings.TrimSpace(line) == "" {
			break
		}
		k, v, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		tier := strings.TrimSpace(k)
		if _, known := tierRank[tier]; !known {
			continue // unknown keys ignored by contract
		}
		if id := stripComment(v); id != "" {
			mapping[tier] = id
		}
	}
	if len(mapping) == 0 {
		*warnings = append(*warnings, "no model_routing tier mapping found in WORKFLOW.md — every dispatch will report unmapped")
	}
	return mapping
}

// stripComment trims a value and drops any trailing "# comment".
func stripComment(v string) string {
	if i := strings.Index(v, "#"); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}

type ledger struct {
	escalation map[string]string // ticket id -> model-tier escalation reason
	fallback   map[string]string // ticket id -> fallback reason
}

// readLedger scans the build ledger for the pinned one-line grammar:
//
//	ESCALATION <ticket-id> <from>-><to> reason: <one line>
//	ESCALATION <ticket-id> effort <from>-><to> reason: <one line>
//	FALLBACK <ticket-id> reason: <one line>
//
// A missing ledger is normal (records are then simply absent). Effort
// escalations are parsed and deliberately unused: they justify effort, not
// model tier.
func readLedger(path string) ledger {
	l := ledger{escalation: map[string]string{}, fallback: map[string]string{}}
	raw, err := os.ReadFile(path)
	if err != nil {
		return l
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "-* "))
		kind, rest, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		id, rest, ok := strings.Cut(strings.TrimSpace(rest), " ")
		if !ok {
			continue
		}
		_, reason, hasReason := strings.Cut(line, "reason:")
		if !hasReason {
			continue
		}
		reason = strings.TrimSpace(reason)
		switch kind {
		case "ESCALATION":
			if strings.HasPrefix(strings.TrimSpace(rest), "effort ") {
				continue // effort record: not model evidence
			}
			l.escalation[id] = reason
		case "FALLBACK":
			l.fallback[id] = reason
		}
	}
	return l
}

// --- transcript inputs (undocumented harness format; degrade, never fail) ---

type dispatch struct {
	toolUseID   string
	description string
	prompt      string
	model       string
}

type subagent struct {
	toolUseID   string
	description string
	models      []string
}

// readTranscripts collects Task/Agent dispatch records from every session
// *.jsonl and actual models from <session>/subagents/agent-*.jsonl, linked
// by the sidecar meta.json. All trouble becomes warnings.
func readTranscripts(dir string, warnings *[]string) ([]dispatch, []subagent) {
	des, err := os.ReadDir(dir)
	if err != nil {
		*warnings = append(*warnings, "transcript dir unreadable — all tickets will report no-transcript: "+err.Error())
		return nil, nil
	}
	var dispatches []dispatch
	var agents []subagent
	for _, de := range des {
		name := de.Name()
		if de.IsDir() {
			subDir := filepath.Join(dir, name, "subagents")
			subs, _ := filepath.Glob(filepath.Join(subDir, "agent-*.jsonl"))
			sort.Strings(subs)
			for _, sub := range subs {
				a := subagent{}
				if metaRaw, err := os.ReadFile(strings.TrimSuffix(sub, ".jsonl") + ".meta.json"); err == nil {
					var meta struct {
						ToolUseID   string `json:"toolUseId"`
						Description string `json:"description"`
					}
					if json.Unmarshal(metaRaw, &meta) == nil {
						a.toolUseID, a.description = meta.ToolUseID, meta.Description
					}
				}
				more, models := scanJSONL(sub, warnings)
				a.models = models
				dispatches = append(dispatches, more...)
				agents = append(agents, a)
			}
			continue
		}
		if strings.HasSuffix(name, ".jsonl") {
			more, _ := scanJSONL(filepath.Join(dir, name), warnings)
			dispatches = append(dispatches, more...)
		}
	}
	return dispatches, agents
}

// scanJSONL extracts dispatch tool_use records and distinct assistant model
// ids from one transcript file. Malformed lines are counted into a single
// per-file warning; they never fail the audit.
func scanJSONL(path string, warnings *[]string) ([]dispatch, []string) {
	f, err := os.Open(path)
	if err != nil {
		*warnings = append(*warnings, path+": unreadable: "+err.Error())
		return nil, nil
	}
	defer f.Close()
	var dispatches []dispatch
	var models []string
	seen := map[string]bool{}
	malformed := 0
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadBytes('\n')
		if len(strings.TrimSpace(string(line))) > 0 {
			d, m, ok := parseLine(line)
			if !ok {
				malformed++
			} else {
				dispatches = append(dispatches, d...)
				if m != "" && !seen[m] {
					seen[m] = true
					models = append(models, m)
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			*warnings = append(*warnings, path+": read error: "+err.Error())
			break
		}
	}
	if malformed > 0 {
		*warnings = append(*warnings, fmt.Sprintf("%s: %d malformed line(s) skipped", path, malformed))
	}
	return dispatches, models
}

// parseLine reads one transcript event. Only assistant events matter: their
// message.model is the actual, and Task/Agent tool_use blocks are dispatch
// records. Unrecognized JSON shapes report as malformed (ok=false).
func parseLine(line []byte) (dispatches []dispatch, model string, ok bool) {
	var ev struct {
		Type    string `json:"type"`
		Message struct {
			Model   string          `json:"model"`
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(line, &ev) != nil {
		return nil, "", false
	}
	if ev.Type != "assistant" {
		return nil, "", true
	}
	var blocks []struct {
		Type  string `json:"type"`
		ID    string `json:"id"`
		Name  string `json:"name"`
		Input struct {
			Description string `json:"description"`
			Prompt      string `json:"prompt"`
			Model       string `json:"model"`
		} `json:"input"`
	}
	if len(ev.Message.Content) > 0 && json.Unmarshal(ev.Message.Content, &blocks) != nil {
		return nil, ev.Message.Model, false // assistant event of unrecognized shape
	}
	for _, b := range blocks {
		if b.Type == "tool_use" && (b.Name == "Task" || b.Name == "Agent") {
			dispatches = append(dispatches, dispatch{
				toolUseID:   b.ID,
				description: b.Input.Description,
				prompt:      b.Input.Prompt,
				model:       b.Input.Model,
			})
		}
	}
	return dispatches, ev.Message.Model, true
}

// --- helpers ---

// containsToken reports whether text contains id as a whole token (so I20
// never matches a dispatch for I201).
func containsToken(text, id string) bool {
	if id == "" {
		return false
	}
	for start := 0; ; {
		i := strings.Index(text[start:], id)
		if i < 0 {
			return false
		}
		i += start
		before := i == 0 || !isAlnum(text[i-1])
		afterIdx := i + len(id)
		after := afterIdx >= len(text) || !isAlnum(text[afterIdx])
		if before && after {
			return true
		}
		start = i + 1
	}
}

func isAlnum(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func dedupSorted(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

func appendUnmatched(list []DispatchInfo, d DispatchInfo) []DispatchInfo {
	for _, have := range list {
		if have == d {
			return list // listed once, informationally
		}
	}
	return append(list, d)
}

// DefaultTranscriptsDir derives the harness's per-project transcript dir
// for a repo: ~/.claude/projects/<slug>, slug being the absolute repo path
// with '/' and '.' flattened to '-'. Best-effort — the harness convention
// is undocumented; `--transcripts` overrides it.
func DefaultTranscriptsDir(repoDir string) (string, error) {
	abs, err := filepath.Abs(repoDir)
	if err != nil {
		return "", err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	slug := strings.NewReplacer("/", "-", ".", "-").Replace(abs)
	return filepath.Join(home, ".claude", "projects", slug), nil
}
