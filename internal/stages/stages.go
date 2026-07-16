// Package stages implements the shared derivation engine for I019: it
// judges a repo's stage cursor (internal/cursor) bidirectionally against
// on-disk artifacts and exposes a per-stage verdict plus a newest-handoff
// backstop check. Two callers reuse the same engine: `spine audit stages`
// (blocking) and doctor's advisory D9 check (never blocking on its own —
// doctor only surfaces).
//
// Conservative derivation is a binding design rule (pinned in the design
// doc and repeated here since this package is where it is enforced):
// absence of evidence never blocks; only presence-contradiction does. A
// stage marked done ([x]) whose expected artifact(s) are missing blocks
// (VerdictTickedMissing) — the cursor is lying about completion. A stage
// still pending ([ ]) whose artifact(s) already exist on disk blocks
// (VerdictPresentUnticked) — the cursor is stale, reality moved on without
// it. The current stage ([<], YOU ARE HERE) is exempt from both directions:
// partial evidence while actively working a stage is expected, not a
// contradiction.
//
// Evidence rules exist for exactly three stage names, per the ticket:
//   - "prd": the cursor's prd: path exists under the repo root.
//   - "issues": every ticket id in the cursor's tickets: set has a
//     docs/issues/*.md file with a matching id: frontmatter field.
//   - "implement": a heuristic scan of .superpowers/sdd/progress.md's
//     dispatch/escalation lines for a "<ticket-id>: ... done|complete"
//     record (case-insensitive), matched via the word-boundary regexp
//     \b(done|completed?)\b so substrings like "abandoned" or "incomplete"
//     do not manufacture false evidence — the same ledger convention audit
//     routing already reads (ESCALATION/FALLBACK lines live in the same
//     file). Documented here as a heuristic because, unlike prd/issues,
//     there is no authoritative on-disk artifact for "implemented" —
//     commit/branch inspection was ruled out (design's Testing Decisions:
//     fixture-repo trees, not real git state) in favor of the ledger's own
//     dispatch record, which every effort already maintains under the
//     audit-routing contract. One accepted residual: the word-boundary
//     match still fires on a phrase like "not complete" (it contains the
//     whole word "complete"), so a negated-but-word-boundary-matching
//     record can still manufacture false evidence — narrower than the
//     substring bug it replaces, but not eliminated.
//
// Every other stage name (grill, functional-test, review, verify, ship,
// deploy, docs, handoff, and anything not in this list) has no rule: it
// derives no evidence and is reported VerdictNotJudged regardless of its
// marker — it can never block.
//
// Design-latitude choices (pinned here):
//   - tickets: grammar has three forms (docs/issues/README.md, the cursor
//     grammar comment; I026 added the bare-id form): "I0NN" (a bare
//     single-ticket id), "I0NN-I0MM" (an inclusive numeric range, both ends
//     the same digit width — a same-endpoint range like "I001-I001" is a
//     valid, if redundant, alias for the bare-id form), or "prefix <str>"
//     (every docs/issues ticket id sharing that literal prefix). Anything
//     else fails to resolve. An unresolved-but-non-empty ticket set degrades
//     to zero evidence for both the issues and implement stages (never a
//     block, same as any other absent evidence) but is surfaced as a
//     Report.Notes entry naming the bad value (I026) — conservative and
//     non-blocking, but visible, rather than a silent degradation
//     indistinguishable from a legitimately empty "prefix" match.
//   - Evidence over an anchored *set* (issues' ticket ids; implement's same
//     set) uses an asymmetric bar to keep both directions conservative:
//     the done-direction check requires ALL items present to count as
//     "not missing" (one missing ticket file is still a lie about a done
//     stage); the pending-direction check requires ANY item present to
//     flag staleness (even one artifact appearing early is a real signal
//     the cursor didn't advance). A resolved-but-empty set (e.g. a "prefix"
//     that matches nothing) is vacuously safe under both directions —
//     nothing to contradict.
//   - The newest-handoff check (I014's backstop) applies whenever a cursor
//     exists, independent of any stage's marker: no handoff at all, an
//     unreadable newest handoff, or one whose content lacks the literal
//     `<!-- spine:cursor -->` marker all count as missing and block. This
//     is the one check that is NOT "absence never blocks" — the ticket
//     names it as an explicit blocking backstop (fixture matrix entry
//     "handoff-missing-block"), independent of the general philosophy
//     above, because an effort with no handoff carrying its cursor is
//     exactly the failure I014 exists to catch.
//   - I025 tightens the same check from presence-only to effort-matched: a
//     newest handoff whose cursor block parses but names a different
//     effort (fixture "handoff-stale-effort") is treated identically to a
//     missing block — HasBlock false, blocking — because a stale block from
//     a previous effort satisfies the letter of I014 while defeating its
//     intent. One accepted consequence, not re-litigated here (see the
//     design doc's handoff-absent-blocks exception, already shipped): a
//     freshly opened effort has no handoff of its own yet, so the newest
//     handoff on disk necessarily belongs to the previous effort and this
//     check blocks until the new effort's first handoff exists — the same
//     shape as the missing-block case it already accepted.
//   - HasCursor==false (cursor.Load's three quiet cases: no WORKFLOW.md, no
//     progress.md, or a progress.md with no cursor block) never blocks and
//     never even runs the stage/handoff checks — there is nothing anchored
//     to judge against. A Notes entry explains which of the three applies,
//     matching the ticket's explicit "no progress.md ⇒ warn, exit 0"
//     acceptance criterion (the sibling two quiet cases get the same
//     non-blocking treatment by the same reasoning, worded accordingly).
//   - Grammar-level cursor.Result.Findings (Task 1's concern — malformed
//     blocks, unknown stage names, and so on) pass through verbatim as
//     Report.CursorFindings for callers to surface, but never affect
//     Blocking(): a parse problem is not itself a stage/artifact
//     contradiction, and Task 1 already decided findings are advisory.
package stages

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/russellpope/spine/internal/cursor"
	"github.com/russellpope/spine/internal/handoff"
)

// Verdict classifies one stage row's derivation outcome.
type Verdict string

const (
	// VerdictMatch: evidence (or its absence) is consistent with the
	// stage's marker.
	VerdictMatch Verdict = "match"
	// VerdictTickedMissing: the stage is marked done but its expected
	// artifact(s) are missing — blocking.
	VerdictTickedMissing Verdict = "ticked-missing"
	// VerdictPresentUnticked: the stage is marked pending but its
	// artifact(s) already exist — blocking.
	VerdictPresentUnticked Verdict = "present-unticked"
	// VerdictNotJudged: no derivation rule for this stage name, the
	// anchored evidence set is empty/unresolved, or the stage is the
	// current ([<]) one — never blocks.
	VerdictNotJudged Verdict = "not-judged"
)

// StageRow is one cursor stage's derivation outcome.
type StageRow struct {
	Name    string
	State   cursor.State
	Verdict Verdict
	Detail  string
}

// StateLabel renders State for display ("done" | "here" | "pending").
func (r StageRow) StateLabel() string {
	switch r.State {
	case cursor.Done:
		return "done"
	case cursor.Here:
		return "here"
	default:
		return "pending"
	}
}

// HandoffCheck is the newest-handoff-carries-the-cursor-block backstop
// (I014). Applicable is true whenever a cursor exists — there is then
// always something to check, even when no handoff exists at all.
type HandoffCheck struct {
	Applicable bool
	Path       string // newest docs/handoffs/* path; "" if none exist
	HasBlock   bool
	Detail     string
}

// Blocking reports whether the handoff check fails: applicable and the
// newest handoff (or its absence) does not carry the cursor block.
func (h HandoffCheck) Blocking() bool { return h.Applicable && !h.HasBlock }

// Report is the full derivation result for one repo.
type Report struct {
	HasCursor bool
	Cursor    cursor.Cursor
	// CursorFindings passes through cursor.Result.Findings verbatim —
	// grammar problems, never blocking here (Task 1's concern).
	CursorFindings []string
	// Notes carries advisory explanations, never gating (Blocking() does
	// not consult it). When HasCursor is false: which of the three quiet
	// cases applies. When HasCursor is true: non-blocking warnings such as
	// an unresolvable tickets: value (I026) — empty when there is nothing
	// to warn about.
	Notes  []string
	Stages []StageRow
	// Handoff is the zero value (Applicable=false) when HasCursor is
	// false — nothing to check a handoff against.
	Handoff HandoffCheck
}

// Blocking reports whether any stage row or the handoff check is blocking.
func (r Report) Blocking() bool {
	for _, s := range r.Stages {
		if s.Verdict == VerdictTickedMissing || s.Verdict == VerdictPresentUnticked {
			return true
		}
	}
	return r.Handoff.Blocking()
}

// Derive loads dir's cursor and derives its full stage report. The only
// error is a genuine I/O failure from cursor.Load (permission errors and
// the like) — never a grammar or derivation problem.
func Derive(dir string) (Report, error) {
	res, err := cursor.Load(dir)
	if err != nil {
		return Report{}, err
	}
	return FromResult(dir, res), nil
}

// FromResult derives a Report from an already-loaded cursor.Result, so a
// caller that has already called cursor.Load (cmd/spine's cursor command)
// need not read the repo twice.
func FromResult(dir string, res cursor.Result) Report {
	rep := Report{HasCursor: res.HasCursor, Cursor: res.Cursor, CursorFindings: res.Findings}
	if !res.HasCursor {
		rep.Notes = []string{noCursorNote(dir)}
		return rep
	}
	var notes []string
	rep.Stages, notes = deriveStages(dir, res.Cursor)
	rep.Notes = notes
	rep.Handoff = deriveHandoff(dir, res.Cursor.Effort)
	return rep
}

// noCursorNote explains which of cursor.Load's three quiet cases applies,
// so callers can tell "dormant repo" from "not a spine repo at all" apart
// even though cursor.Result collapses them to the same HasCursor==false.
func noCursorNote(dir string) string {
	if _, err := os.Stat(filepath.Join(dir, "WORKFLOW.md")); err != nil {
		return "no WORKFLOW.md — not a spine repo, nothing to derive"
	}
	if _, err := os.Stat(filepath.Join(dir, ".superpowers", "sdd", "progress.md")); err != nil {
		return "no .superpowers/sdd/progress.md — dormant repo (not mid-effort), nothing to derive"
	}
	return "progress.md has no spine:cursor block — nothing to derive"
}

// deriveStages judges every stage in c.Stages in cursor order. The second
// return is Notes: currently just a single entry (if any) when c.Tickets is
// non-empty but unresolvable against the grammar — see
// unresolvableTicketsNote.
func deriveStages(dir string, c cursor.Cursor) ([]StageRow, []string) {
	var prdPresent []bool
	if c.PRD != "" {
		_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(c.PRD)))
		prdPresent = []bool{err == nil}
	}

	var issuesPresent, implPresent []bool
	var ids []string
	var notes []string
	if resolved, ok := resolveTicketIDs(dir, c.Tickets); ok {
		ids = resolved
		if len(ids) > 0 {
			have := issueIDs(filepath.Join(dir, "docs", "issues"))
			evidenced := implementEvidence(readLedgerRaw(dir), ids)
			for _, id := range ids {
				issuesPresent = append(issuesPresent, have[id])
				implPresent = append(implPresent, evidenced[id])
			}
		}
	} else if strings.TrimSpace(c.Tickets) != "" {
		notes = append(notes, unresolvableTicketsNote(c.Tickets))
	}

	rows := make([]StageRow, 0, len(c.Stages))
	for _, s := range c.Stages {
		var verdict Verdict
		var detail string
		switch s.Name {
		case "prd":
			// prd's evidence is a single path, not an anchored ticket-id
			// set, so no ids/tickets value to name on a miss.
			verdict, detail = judgeSet(s.State, prdPresent, nil, "", "PRD file "+dash(c.PRD))
		case "issues":
			verdict, detail = judgeSet(s.State, issuesPresent, ids, c.Tickets, "ticket file(s)")
		case "implement":
			verdict, detail = judgeSet(s.State, implPresent, ids, c.Tickets, "ledger implement evidence")
		default:
			verdict, detail = VerdictNotJudged, "no derivation rule for stage \""+s.Name+"\""
		}
		rows = append(rows, StageRow{Name: s.Name, State: s.State, Verdict: verdict, Detail: detail})
	}
	return rows, notes
}

func dash(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

// judgeSet applies the bidirectional check to one stage given its state and
// the presence facts for its anchored evidence set (one bool per anchored
// item — a single-element set for prd, one per ticket id for issues and
// implement). An empty set (nothing resolvable) is always VerdictNotJudged.
//
// ids is the parallel ticket-id list for present (nil for prd, which has no
// per-item ids to name — its single-element set is already fully named by
// label). ticketsRaw is the cursor's live tickets: value, used only on a
// VerdictTickedMissing verdict; it may be "" wherever ids is nil. I029: a
// ticked-missing detail names the missing ids (see missingIDs/namedIDs) and,
// when every resolved id is missing, also surfaces ticketsRaw — an
// all-missing set is exactly the shape a resolvable-but-wrong tickets:
// value (a typo'd range/prefix) produces, so the reader is pointed at the
// likely cause rather than left with a bare count.
func judgeSet(state cursor.State, present []bool, ids []string, ticketsRaw string, label string) (Verdict, string) {
	if len(present) == 0 {
		return VerdictNotJudged, "no evidence to derive (n/a)"
	}
	if state == cursor.Here {
		return VerdictNotJudged, "current stage — not judged"
	}
	existing := 0
	for _, p := range present {
		if p {
			existing++
		}
	}
	all := existing == len(present)
	any := existing > 0
	switch state {
	case cursor.Done:
		if !all {
			detail := fmt.Sprintf("marked done but %d/%d %s missing", len(present)-existing, len(present), label)
			if missing := missingIDs(present, ids); len(missing) > 0 {
				detail += ": " + namedIDs(missing)
			}
			if existing == 0 && ticketsRaw != "" {
				detail += fmt.Sprintf(" — tickets: %q resolved but every id is missing; check it for a typo", ticketsRaw)
			}
			return VerdictTickedMissing, detail
		}
		return VerdictMatch, fmt.Sprintf("%d/%d %s present", existing, len(present), label)
	default: // cursor.Pending
		if any {
			return VerdictPresentUnticked, fmt.Sprintf("%d/%d %s already present but stage not marked done", existing, len(present), label)
		}
		return VerdictMatch, fmt.Sprintf("no %s yet, consistent with pending", label)
	}
}

// maxNamedMissingIDs caps how many missing ticket ids namedIDs lists
// verbatim before folding the rest into a "+N more" tail, so a large
// missing range doesn't dump every id onto one detail line.
const maxNamedMissingIDs = 5

// missingIDs returns the subset of ids whose parallel present entry is
// false, preserving ids' order. ids and present must be parallel (same
// length); a mismatch (including ids==nil, prd's case) yields nil — nothing
// to name.
func missingIDs(present []bool, ids []string) []string {
	if len(ids) != len(present) {
		return nil
	}
	var out []string
	for i, p := range present {
		if !p {
			out = append(out, ids[i])
		}
	}
	return out
}

// namedIDs renders missing as a comma-joined list, capped at
// maxNamedMissingIDs with a "+N more" count for longer sets.
func namedIDs(missing []string) string {
	if len(missing) <= maxNamedMissingIDs {
		return strings.Join(missing, ", ")
	}
	return strings.Join(missing[:maxNamedMissingIDs], ", ") + fmt.Sprintf(" +%d more", len(missing)-maxNamedMissingIDs)
}

// deriveHandoff applies the I014 newest-handoff backstop, plus I025's
// effort-match requirement: presence of a `<!-- spine:cursor -->` block is
// not enough — the block's effort: must match the live cursor's effort
// (liveEffort, i.e. the effort anchoring the report this check is part of).
// A stale-effort block (well-formed, but carried over from a previous
// effort) is treated identically to an absent block — HasBlock false,
// Blocking() true — because a stale block defeats I014's intent exactly as
// much as a missing one does; only the Detail differs, naming both efforts
// so the finding is actionable. Only called when a cursor exists
// (Applicable is therefore always true on the way in).
//
// M4 (I027): a genuine I/O error reading docs/handoffs (handoff.Latest's
// err) and docs/handoffs legitimately having zero entries (ok false, err
// nil) both block the same way, but the Detail wording is kept distinct —
// "unreadable" vs "no ... entries found" — so a transient read failure
// doesn't masquerade as "you never wrote a handoff."
func deriveHandoff(dir string, liveEffort string) HandoffCheck {
	entry, ok, err := handoff.Latest(dir)
	if err != nil {
		return HandoffCheck{Applicable: true,
			Detail: "docs/handoffs unreadable: " + err.Error()}
	}
	if !ok {
		return HandoffCheck{Applicable: true,
			Detail: "no docs/handoffs entries found — the newest handoff must carry the spine:cursor block once a cursor exists"}
	}
	raw, err := os.ReadFile(entry.Path)
	if err != nil {
		return HandoffCheck{Applicable: true, Path: entry.Path,
			Detail: "newest handoff unreadable: " + err.Error()}
	}
	content := string(raw)
	if !cursor.HasBlock(content) {
		return HandoffCheck{Applicable: true, Path: entry.Path, HasBlock: false,
			Detail: "newest handoff " + entry.Path + " is missing the spine:cursor block"}
	}
	blockCursor, _ := cursor.ParseBlock(content)
	if blockCursor.Effort != liveEffort {
		return HandoffCheck{Applicable: true, Path: entry.Path, HasBlock: false,
			Detail: fmt.Sprintf("newest handoff %s carries a stale effort cursor block: block effort %q, live effort %q", entry.Path, blockCursor.Effort, liveEffort)}
	}
	return HandoffCheck{Applicable: true, Path: entry.Path, HasBlock: true,
		Detail: "newest handoff " + entry.Path + " carries the spine:cursor block"}
}

// readLedgerRaw returns progress.md's content, or "" if it can't be read
// (Derive already established HasCursor==true, so this should not fail in
// practice, but a failure here degrades to zero implement evidence rather
// than an error — absence of evidence never blocks).
func readLedgerRaw(dir string) string {
	raw, err := os.ReadFile(filepath.Join(dir, ".superpowers", "sdd", "progress.md"))
	if err != nil {
		return ""
	}
	return string(raw)
}

// implementDoneWordRe matches "done" or "complete"/"completed" as whole
// words (not substrings), so it does not fire on "abandoned" (contains
// "done") or "incomplete" (contains "complete"). Compiled once at package
// init rather than per call.
var implementDoneWordRe = regexp.MustCompile(`\b(done|completed?)\b`)

// implementEvidence scans ledgerRaw's dispatch/escalation lines (the same
// file audit routing reads ESCALATION/FALLBACK records from) for a
// "<ticket-id>: ... done|complete" record per id, case-insensitive, matching
// done/complete as whole words via implementDoneWordRe so that negations
// like "abandoned" or "incomplete" do not manufacture false evidence. This
// is the documented heuristic for "implement" evidence: there is no
// authoritative on-disk artifact for "implemented", so the ledger's own
// dispatch record — which every effort already maintains — stands in. One
// residual the word-boundary match cannot cure: a done-word about a
// different stage on a ticket-prefixed line (e.g. "I019: grill done")
// still counts as implement evidence — under-detection's mirror image,
// accepted because the line format is ledger-convention-bound rather than
// stage-structured.
func implementEvidence(ledgerRaw string, ids []string) map[string]bool {
	evidenced := map[string]bool{}
	for _, line := range strings.Split(ledgerRaw, "\n") {
		trimmed := strings.TrimLeft(strings.TrimSpace(line), "-* ")
		lower := strings.ToLower(trimmed)
		if !implementDoneWordRe.MatchString(lower) {
			continue
		}
		for _, id := range ids {
			if evidenced[id] {
				continue
			}
			if strings.HasPrefix(trimmed, id+":") {
				evidenced[id] = true
			}
		}
	}
	return evidenced
}

// issueIDs returns the set of docs/issues ticket ids present on disk (files
// with a parseable id: frontmatter field). A missing/unreadable dir yields
// an empty set — never an error, matching the conservative philosophy.
func issueIDs(issuesDir string) map[string]bool {
	ids := map[string]bool{}
	des, err := os.ReadDir(issuesDir)
	if err != nil {
		return ids
	}
	for _, de := range des {
		name := de.Name()
		if de.IsDir() || !strings.HasSuffix(name, ".md") || strings.HasPrefix(name, "_") || name == "README.md" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(issuesDir, name))
		if err != nil {
			continue
		}
		if id := frontmatterID(string(raw)); id != "" {
			ids[id] = true
		}
	}
	return ids
}

// frontmatterID extracts the id: field from a docs/issues file's leading
// --- frontmatter fence, or "" if absent/malformed.
func frontmatterID(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return ""
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if strings.TrimSpace(k) == "id" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

var ticketIDRe = regexp.MustCompile(`^I\d+$`)
var ticketRangeRe = regexp.MustCompile(`^I(\d+)-I(\d+)$`)

// resolveTicketIDs parses the cursor's tickets: value into the concrete set
// of ticket ids it anchors. Three grammar forms resolve (see package doc
// and cursor.Grammar, I026): a bare single-ticket id "I0NN" (resolves to
// that one id, unconditionally — unlike "prefix", a bare id names a
// specific ticket rather than a repo-resolved set, so it never needs a
// docs/issues lookup to resolve); "I0NN-I0MM" (an inclusive numeric range,
// equal digit width — a same-endpoint range like "I001-I001" resolves to
// the same single-element set as the bare-id form, structurally, though the
// bare form is the documented idiom); and "prefix <str>" (every docs/issues
// ticket id sharing that prefix, resolved against the repo — so it can
// legitimately resolve to an empty set). Anything else returns ok=false:
// unresolvable, never a block — the caller surfaces this as a Notes entry
// naming the bad value (see unresolvableTicketsNote) rather than silently
// treating it like a resolved-but-empty set.
func resolveTicketIDs(dir, raw string) ([]string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, false
	}
	if rest, ok := strings.CutPrefix(raw, "prefix "); ok {
		prefix := strings.TrimSpace(rest)
		if prefix == "" {
			return nil, false
		}
		have := issueIDs(filepath.Join(dir, "docs", "issues"))
		var out []string
		for id := range have {
			if strings.HasPrefix(id, prefix) {
				out = append(out, id)
			}
		}
		sort.Strings(out)
		return out, true
	}
	if ticketIDRe.MatchString(raw) {
		return []string{raw}, true
	}
	m := ticketRangeRe.FindStringSubmatch(raw)
	if m == nil || len(m[1]) != len(m[2]) {
		return nil, false
	}
	start, err1 := strconv.Atoi(m[1])
	end, err2 := strconv.Atoi(m[2])
	if err1 != nil || err2 != nil || start > end {
		return nil, false
	}
	width := len(m[1])
	out := make([]string, 0, end-start+1)
	for n := start; n <= end; n++ {
		out = append(out, fmt.Sprintf("I%0*d", width, n))
	}
	return out, true
}

// unresolvableTicketsNote explains a non-empty tickets: value that failed
// to resolve against the grammar (I026): conservative and non-blocking —
// Report.Blocking() never consults Notes, so this can never gate anything —
// but visible, naming the exact bad value, following the same explanatory
// pattern as noCursorNote. Without this, an unresolvable tickets: value
// degrades the issues and implement evidence rules to VerdictNotJudged
// exactly like a well-formed-but-empty "prefix" match, with no
// operator-visible signal that the degradation happened at all.
func unresolvableTicketsNote(raw string) string {
	return fmt.Sprintf("tickets: %q does not resolve (grammar: I0NN | I0NN-I0MM | prefix <str>) — issues/implement evidence not judged", raw)
}
