// Package audit implements the core of `spine audit routing`: a
// deterministic diff of a scaffolded repo's declared per-ticket model-tier
// annotations against the models the harness transcripts say were actually
// used. The boundary is the pure function Run (repo dir + transcript dir in,
// Report out); the CLI in cmd/spine is a thin printer over it.
//
// Inputs, all read-only:
//   - docs/issues/*.md frontmatter: id plus the optional annotation fields
//     tier / execution-mode / effort / risk-triggers / review-tier.
//   - The shared model resolver (internal/model, design D13): tier -> id
//     resolution is flavor-scoped and goes through model.Resolve, honoring
//     the repo's WORKFLOW.md mirror overrides (dotted gen-10 keys; bare
//     gen ≤9 tier keys as claude values) and falling back to the embedded
//     defaults when the file, block, or key is absent — exactly what
//     dispatch-time resolution returns, so dispatched and verified cannot
//     disagree. The audit owns no WORKFLOW.md parser of its own.
//   - WORKFLOW.md's template_version stamp (via update.ExtractKeys, the one
//     top-level-key parser): a generation newer than this binary compiles
//     is refused (design D14), matching spine update's gate — an
//     un-upgraded binary must not emit confident verdicts from a misparse.
//   - .superpowers/sdd/progress.md: one-line ESCALATION / FALLBACK records.
//   - The claude harness transcript dir: <dir>/*.jsonl session records plus
//     <dir>/<session>/subagents/agent-*.jsonl (+ sibling .meta.json).
//   - The codex sessions dir (design D20/D23, I041, see codex.go): rollout
//     JSONL trees under CodexSessionsDir, read only when that option is
//     non-empty. Both transcript formats are undocumented and may shift:
//     any parse failure, missing dir, or unrecognized shape degrades to a
//     Report warning, never an error.
//
// Design-latitude choices (the ticket leaves these open; pinned here):
//   - Every ticket file in docs/issues with a frontmatter id gets a row;
//     files without an id and files starting with "_" (templates) or named
//     README.md are ignored. Tickets whose tier annotation is not one of
//     the four known tiers are reported as unannotated (detail names the
//     unknown value), never judged. `tier: n/a` (design D27, ticket I046)
//     is a declared decision to opt out, distinct from that: reported
//     exempt, never judged, mirroring the review-tier: n/a convention. An
//     absent tier stays unannotated — absence is a gap, n/a is a decision.
//   - Model evidence per dispatch: the linked subagent transcript's
//     assistant model ids when one exists (linked via the meta.json
//     toolUseId, or its description's ticket token); otherwise the
//     dispatch's model alias; a dispatch with neither contributes nothing.
//     Main-session assistant models are never ticket evidence — inline
//     execution is out of the audit's scope by design.
//   - Flavor of a dispatch is derived from its transcript source (design
//     D15; see transcriptFlavor and readCodexSessions) and travels with the
//     token itself (evidenceToken) all the way into judgeToken, which
//     resolves it within that flavor's table alone (I040). The claude
//     reader tags every token it produces "claude"; the codex reader (I041,
//     codex.go) tags every token it produces "codex" — judge/judgeToken
//     needed no change to pick up the second source. The worker-session scan
//     (D21, ticket I042) and the git-commit-probe repo scoping (D22, ticket
//     I043) both shipped in codex.go: codex evidence covers dispatch
//     records, linkable spawned-thread actuals, and top-level orchestrator
//     sessions attributed via D21's opening-line rule, all repo-scoped by
//     D22's cwd/commit-hash checks.
//   - Token -> tier, within the dispatch's flavor: a token maps to a tier
//     when it equals the resolved id, one of the table entry's explicitly
//     declared aliases, or an id the entry shipped as a default before the
//     current one (historical ids carry no aliases, so a pre-refresh
//     transcript — e.g. a claude-opus-4-8 dispatch — matches by full id
//     only). An Override entry matches by its exact on-disk id alone:
//     aliases and history describe the shipped defaults, and a dispatch on
//     the displaced default in a repo that pinned something else must
//     surface as unmapped, not read as a match (Default and Inherited
//     entries keep both). Substring containment is retired (design D13): it collides as
//     model names multiply across flavors with unrelated naming schemes.
//     When a token maps to several tiers — two tiers sharing an id is legal,
//     e.g. the shipped codex.routine/codex.fallback pair — the reading
//     closest to a non-verdict wins: declared tier if present; else, when
//     the ticket carries a FALLBACK record and fallback is among the
//     candidates, fallback (ADR 0012 / D25 — a ledger record decides what an
//     ambiguous token means, not just how the verdict reads, so a properly
//     recorded refusal-rerun can never stand as a false silent-descent
//     blocker); else the highest-ranked ordered candidate — degradation must
//     not manufacture descent. Without a FALLBACK record the ordered reading
//     stands, so real descent still judges silent-descent. A token mapping
//     only to fallback is lateral: covered by a FALLBACK record ->
//     escalated-with-reason; covered by a `tier: fallback` annotation ->
//     match; otherwise the warn-level unexplained-fallback.
//   - A model-tier ESCALATION record excuses exactly its recorded to-tier:
//     an off-tier token is escalated-with-reason iff some record on that
//     ticket has a to-tier equal to the token's resolved tier (a reasoned
//     DOWNWARD record — primary->routine — therefore keeps reasoned descent
//     advisory). A token matching no record's to-tier is judged against the
//     annotation: below it is silent-descent; above it is the warn-level
//     escalated-no-reason (not blocking: quality went up, but the contract
//     says escalations carry reasons). Effort ESCALATION records are
//     accepted grammar but are not model evidence; records that do not
//     parse as <from>-><to> excuse nothing.
//   - A ticket's verdict is its worst token's verdict: silent-descent >
//     unmapped-dispatch > unexplained-fallback > escalated-no-reason >
//     escalated-with-reason > match.
//   - Hard errors are a missing docs/issues dir (not a scaffolded repo —
//     CLI usage error), the D14 version-gate refusal above, and an
//     unparseable --since value (D28, ticket I047 review ruling 4 — an
//     operator-typed value has no valid fallback reading); everything else
//     degrades to warnings. A missing or unreadable WORKFLOW.md is a
//     warning, not an error: resolution falls back to embedded defaults,
//     and the report says so.
package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/russellpope/spine/internal/model"
	"github.com/russellpope/spine/internal/tmpl"
	"github.com/russellpope/spine/internal/update"
)

// Verdict classifies one ticket's declared-vs-actual routing.
type Verdict string

// Verdict values, worst first.
const (
	VerdictSilentDescent          Verdict = "silent-descent"          // blocking
	VerdictUnmappedDispatch       Verdict = "unmapped-dispatch"       // warn
	VerdictUnexplainedFallback    Verdict = "unexplained-fallback"    // warn
	VerdictEscalatedNoReason      Verdict = "escalated-no-reason"     // warn
	VerdictNoTranscript           Verdict = "no-transcript"           // warn
	VerdictUnattributedTranscript Verdict = "unattributed-transcript" // warn (D24, ticket I044)
	VerdictEscalatedWithReason    Verdict = "escalated-with-reason"   // advisory
	VerdictMatch                  Verdict = "match"
	VerdictExempt                 Verdict = "exempt"      // informational (D27, ticket I046): tier: n/a opts out
	VerdictUnannotated            Verdict = "unannotated" // informational
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
	// Effort is the worker effort a claude-team spawn declared (I090); ""
	// for every other dispatch shape. It is reported, never judged: the
	// routing contract's enforcement is model-vs-tier, and comparing effort
	// against a ticket's effort: frontmatter would be a second, separate
	// contract. Out of scope for I090 by design, not an oversight.
	Effort string
	// TeamSpawn marks a claude-team worker spawn (I090). An unmatched one
	// is the residual blind spot the ticket's footer clause was written
	// for: the lead's real dispatch flow delivers the brief by file
	// reference (`"$(cat …dispatch-task-2.md)"`), so neither the spawn
	// command nor its prompt carries a ticket token and the worker cannot
	// be attributed. Counted in the report footer so the gap stays
	// visible rather than silent. Ticket I101 tracks closing it.
	TeamSpawn bool
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

// evidenceToken is one observed model string paired with the flavor of the
// transcript source it came from (design D15's per-token seam, made real by
// I040). judgeToken resolves value within flavor's table alone; nothing
// upstream of it needs to know which flavor a token carries.
type evidenceToken struct {
	value      string
	flavor     string
	sourceFile string // source transcript file (D24, codex only); "" for claude, keeping claude details byte-identical
}

// tokenValues extracts the raw model strings from a slice of evidence
// tokens, for the report's flavor-agnostic TicketRow.Actuals display.
func tokenValues(tokens []evidenceToken) []string {
	values := make([]string, len(tokens))
	for i, tok := range tokens {
		values[i] = tok.value
	}
	return values
}

// Options configures one Run.
type Options struct {
	RepoDir              string
	ClaudeTranscriptsDir string
	CodexSessionsDir     string // codex sessions dir (readCodexSessions); empty opts out of codex discovery entirely (I041)
	// Since scopes the transcript set to sessions active at/after a cutoff
	// (D28, ticket I047): an operator escape hatch, never an automatic
	// build-start anchor (rejected at grill — see parseSince). Accepts
	// RFC3339 or a bare YYYY-MM-DD date; empty means no filter. Applied to
	// both flavors by comparing each transcript session's mtime against the
	// cutoff (see parseSince's doc for why mtime, not an in-JSONL
	// timestamp). An unparseable value is a usage error: Run returns it
	// directly (never a Report), the same hard-error class as a missing
	// docs/issues dir. Ratified at I047 review: an operator-typed value has
	// no valid fallback reading, and warn-and-proceed would silently
	// readmit exactly the sessions the operator was trying to exclude —
	// the wrong degrade direction for a flag whose entire purpose is
	// exclusion.
	Since string
	// Session restricts the transcript set to one session (D28, ticket
	// I047): an operator escape hatch. Empty means no filter. For claude,
	// matches the top-level session file's base name (<id>.jsonl) and its
	// sibling <id>/subagents dir — a session's own spawned subagents stay
	// in scope, exactly as they do unfiltered. For codex, matches
	// session_meta.payload.session_id (the thread tree's ROOT id, shared by
	// every file in the tree; see codexSessionMeta.rootID) — restricting to
	// one codex "session" means one thread tree, the same granularity a
	// claude session-plus-its-subagents represents.
	Session string
}

// Run audits opts.RepoDir's declared routing against the transcript records
// in opts.ClaudeTranscriptsDir. Transcript trouble of any kind degrades to
// Warnings; the only errors are a repo without docs/issues and the D14
// version-gate refusal (see the package comment).
func Run(opts Options) (Report, error) {
	repoDir, transcriptsDir := opts.RepoDir, opts.ClaudeTranscriptsDir
	var rep Report
	tickets, err := readTickets(filepath.Join(repoDir, "docs", "issues"))
	if err != nil {
		return Report{}, err
	}
	// D28 (ticket I047) repo-qualification inputs: the audited repo's
	// absolute path (for a dispatch's literal-path self-reference and for
	// cwd-inside-repo evidence) and its basename (for a shorter token
	// reference). Best-effort on Abs failure — repoQualifies then falls
	// back to comparing the raw repoDir, never a hard error over a
	// qualification input.
	absRepoDir, err := filepath.Abs(repoDir)
	if err != nil {
		absRepoDir = repoDir
	}
	repoBase := filepath.Base(absRepoDir)
	// D28/I047 review ruling 4: an unparseable --since is a usage error
	// (this Run returns it directly, exit 2 at the CLI boundary via the
	// existing err != nil handling in cmdAuditRouting — no CLI change
	// needed), not a degrade-to-warning path. An operator-typed value has
	// no valid fallback reading, and warn-and-proceed would silently
	// readmit exactly the sessions --since exists to exclude.
	var since time.Time
	if opts.Since != "" {
		t, err := parseSince(opts.Since)
		if err != nil {
			return Report{}, fmt.Errorf(
				"--since %q: %w (accepted formats: RFC3339 e.g. 2026-07-20T00:00:00Z, or a bare date e.g. 2026-07-20)",
				opts.Since, err)
		}
		since = t
	}
	if err := gateTemplateVersion(repoDir, &rep.Warnings); err != nil {
		return Report{}, err
	}
	if !anyAnnotated(tickets) {
		rep.Warnings = append(rep.Warnings, "nothing audited — no annotated tickets found (zero docs/issues tickets carry a tier: annotation); an exit-0 run judged nothing")
	}
	flavor := transcriptFlavor(transcriptsDir)
	mapping, err := resolveFlavorTiers(repoDir, flavor)
	if err != nil {
		return Report{}, err
	}
	codexMapping, err := resolveFlavorTiers(repoDir, "codex")
	if err != nil {
		return Report{}, err
	}
	// mappings is keyed by flavor so judgeToken can resolve each token
	// within its own flavor's table (I040). The codex entry resolves
	// unconditionally (cheap, always defined via embedded defaults) even
	// when no codex sessions dir is configured — only the reader below is
	// gated on that.
	mappings := map[string]map[string]resolvedTier{flavor: mapping, "codex": codexMapping}
	ledger := readLedger(filepath.Join(repoDir, ".superpowers", "sdd", "progress.md"))
	dispatches, agents, sessionMatched := readTranscripts(transcriptsDir, flavor, since, opts.Session, &rep.Warnings)
	// Codex discovery only runs when CodexSessionsDir is set (the CLI always
	// sets it — design D-doc, "discovery is always on"; leaving it empty is
	// how every pre-I041 caller and every existing test opts out, and must
	// audit exactly as before — no attempt, no warning, I041). codexNearMisses
	// stays nil in that case, so the D24 near-miss override below never fires
	// for a claude-only caller — one more guarantee claude paths stay
	// byte-identical.
	var codexNearMisses []codexNearMiss
	if opts.CodexSessionsDir != "" {
		// I049: the discovery pre-filter's token set is every audited
		// ticket's id, already read above (line 238) — well before this
		// call, so no extra pass over docs/issues is needed.
		ticketTokens := make([]string, len(tickets))
		for i, t := range tickets {
			ticketTokens[i] = t.id
		}
		codexDispatches, codexAgents, codexNM, codexSessionMatched := readCodexSessions(opts.CodexSessionsDir, repoDir, since, opts.Session, ticketTokens, &rep.Warnings)
		dispatches = append(dispatches, codexDispatches...)
		agents = append(agents, codexAgents...)
		codexNearMisses = codexNM
		sessionMatched = sessionMatched || codexSessionMatched
	}
	// M3 (I047 review): a non-empty --session that matched nothing anywhere
	// (claude or codex) is silently misleading otherwise — an operator
	// typo'd id produces an unexplained all-no-transcript audit, and codex
	// root ids are invisible in filenames (rollout-<ts>-<uuid>.jsonl) so
	// there's no other way to notice. Fires once, combined across both
	// flavors, not per-reader.
	if opts.Session != "" && !sessionMatched {
		rep.Warnings = append(rep.Warnings, fmt.Sprintf("--session %q matched no sessions", opts.Session))
	}

	evidence := map[string][]evidenceToken{} // ticket id -> flavor-tagged model tokens
	claimed := map[int]bool{}                // dispatch index -> matched a ticket
	linked := map[string]bool{}              // toolUseID -> a subagent transcript carries models
	for _, a := range agents {
		if a.toolUseID != "" && len(a.models) > 0 {
			linked[a.toolUseID] = true
		}
	}
	// codex ticket-token matching is case-insensitive (D20's "Flavor
	// threading" closing paragraph); the claude reader's matching is
	// untouched. codex dispatch descriptions/prompts are folded to upper
	// case here rather than at parse time, so Unmatched's display text
	// keeps its natural case.
	matches := func(d dispatch, id string) bool {
		desc, prompt := d.description, firstLine(d.prompt)
		if d.flavor == "codex" {
			desc, prompt = strings.ToUpper(desc), strings.ToUpper(prompt)
		}
		if !(containsToken(desc, id) || containsToken(prompt, id)) {
			return false
		}
		// D28 (ticket I047): a claude dispatch claims the ticket only if it
		// ALSO references the audited repo or its own session shows cwd
		// evidence inside it — see repoQualifies. Codex evidence is already
		// hard-scoped to the repo before it reaches Run (D22,
		// readCodexSessions' cwdInsideRepo/gitCommitProber gate), so gating
		// it again here would be redundant, not stricter.
		if d.flavor == "claude" && !repoQualifies(d.description+"\n"+d.prompt, d.cwd, absRepoDir, repoBase) {
			return false
		}
		return true
	}
	// rootTickets tracks, per dispatch root (toolUseID), every distinct
	// ticket claimed under it — the coarse-linkage disclosure input (I041
	// review referred-Q3, ticket I044): thread_spawn actuals link by ROOT
	// session id only, so two dispatches for two different tickets sharing
	// one root also share whatever actual evidence that root's subagent(s)
	// contribute. Populated for every flavor: a claude toolUseID IS the
	// tool_use block's own id, unique per dispatch call, but that only means
	// it can't be shared by two SEPARATE dispatch calls — a single Task
	// dispatch whose own description names two ticket ids still claims both
	// under that one toolUseID (final-review fix round, Important-2). The
	// disclosure text itself stays codex-specific — see coarseLinkageNotes.
	rootTickets := map[string]map[string]bool{}
	for _, t := range tickets {
		for i, d := range dispatches {
			if !matches(d, t.id) {
				continue
			}
			claimed[i] = true
			if d.toolUseID != "" {
				if rootTickets[d.toolUseID] == nil {
					rootTickets[d.toolUseID] = map[string]bool{}
				}
				rootTickets[d.toolUseID][t.id] = true
			}
			if linked[d.toolUseID] {
				continue // the subagent transcript below is the actual
			}
			if d.model != "" {
				evidence[t.id] = append(evidence[t.id], evidenceToken{value: d.model, flavor: d.flavor, sourceFile: d.sourceFile})
			}
		}
		for _, a := range agents {
			desc := a.description
			if a.flavor == "codex" {
				desc = strings.ToUpper(desc)
			}
			use := containsToken(desc, t.id)
			// D28: the same repo qualification a dispatch needs applies to a
			// subagent's own description carrying the ticket token directly
			// (meta.json's description typically mirrors its parent
			// dispatch's) — otherwise this path would readmit exactly the
			// cross-repo collision the dispatch-side gate above closes, on a
			// shared transcript dir where both repos' subagent files carry
			// the same ticket id in their descriptions.
			if use && a.flavor == "claude" && !repoQualifies(a.description, a.cwd, absRepoDir, repoBase) {
				use = false
			}
			for _, d := range dispatches {
				if use {
					break
				}
				use = d.toolUseID != "" && d.toolUseID == a.toolUseID && matches(d, t.id)
			}
			if use {
				for _, m := range a.models {
					evidence[t.id] = append(evidence[t.id], evidenceToken{value: m, flavor: a.flavor, sourceFile: a.sourceFile})
				}
			}
		}
	}
	for i, d := range dispatches {
		if !claimed[i] {
			rep.Unmatched = appendUnmatched(rep.Unmatched, DispatchInfo{
				Description: d.description, Model: d.model, Effort: d.effort, TeamSpawn: d.teamSpawn})
		}
	}

	coarseNotes := coarseLinkageNotes(rootTickets, dispatches, linked)
	for _, t := range tickets {
		tokens := evidence[t.id]
		row := TicketRow{ID: t.id, Tier: t.tier, Actuals: dedupSorted(tokenValues(tokens))}
		row.Verdict, row.Detail = judge(t, tokens, mappings, ledger)
		// D24 (ticket I044): a ticket that landed on no-transcript — zero
		// attributed evidence — upgrades to unattributed-transcript when
		// repo-scoped codex material mentioned it but failed attribution
		// (guardian-only, mid-transcript-only, orchestrator-only). Any
		// ticket with real evidence already judged above never consults
		// this: found-but-unusable never downgrades found-and-usable.
		if row.Verdict == VerdictNoTranscript {
			if detail, ok := nearMissDetail(codexNearMisses, t.id); ok {
				row.Verdict, row.Detail = VerdictUnattributedTranscript, detail
			}
		}
		if note, ok := coarseNotes[t.id]; ok {
			if row.Detail == "" {
				row.Detail = note
			} else {
				row.Detail += "; " + note
			}
		}
		rep.Tickets = append(rep.Tickets, row)
	}
	sort.Slice(rep.Tickets, func(i, j int) bool { return rep.Tickets[i].ID < rep.Tickets[j].ID })
	return rep, nil
}

// nearMissDetail matches a ticket's token against every accumulated codex
// near miss (D24, ticket I044), reporting every distinct (reason, file)
// combination found — what was found, why it was excluded, and its source
// file, so a surprising unattributed-transcript verdict is diagnosable at a
// glance. Matching is case-insensitive (every near miss is codex-sourced,
// and codex ticket-token matching is case-insensitive per D20).
func nearMissDetail(nms []codexNearMiss, id string) (string, bool) {
	var parts []string
	seen := map[string]bool{}
	for _, nm := range nms {
		if !containsToken(strings.ToUpper(nm.text), id) {
			continue
		}
		key := nm.reason + "|" + nm.file
		if seen[key] {
			continue
		}
		seen[key] = true
		parts = append(parts, fmt.Sprintf("%s (source: %s)", nm.reason, nm.file))
	}
	if len(parts) == 0 {
		return "", false
	}
	return "found but unattributed: " + strings.Join(parts, "; "), true
}

// coarseLinkageNotes builds the I041-review-referred-Q3 disclosure (ticket
// I044): for every codex dispatch root shared by two or more distinct
// tickets where that root's declared aliases were superseded by real linked
// actuals (linked[root]), each of those tickets' detail should disclose that
// its actuals are only root-linked (D20 clause 2), not per-dispatch-linked,
// and so may be shared with the other ticket(s) named. A root claimed by
// only one ticket, or one with no linked actual evidence at all (nothing to
// merge), gets no note — this is a diagnostic aid for a real ambiguity, not
// standing noise.
//
// The disclosure's wording ("codex session root", "(D20)") is codex-specific
// by construction, so only roots carrying the "codex:" toolUseID prefix
// (readCodexSessions' own tagging) are eligible (final-review fix round,
// Important-2). Without this, a single claude Task dispatch whose own
// description names two ticket ids — rootTickets is populated for every
// flavor, not just codex, see its doc — would fire this codex-worded note on
// pure-claude evidence, breaking the I040 claude-only byte-identity promise
// with misleading, factually wrong text.
func coarseLinkageNotes(rootTickets map[string]map[string]bool, dispatches []dispatch, linked map[string]bool) map[string]string {
	notes := map[string]string{}
	for root, ids := range rootTickets {
		if len(ids) < 2 || !linked[root] || !strings.HasPrefix(root, "codex:") {
			continue
		}
		idList := make([]string, 0, len(ids))
		for id := range ids {
			idList = append(idList, id)
		}
		sort.Strings(idList)
		file := ""
		for _, d := range dispatches {
			if d.toolUseID == root && d.sourceFile != "" {
				file = d.sourceFile
				break
			}
		}
		for _, id := range idList {
			var others []string
			for _, o := range idList {
				if o != id {
					others = append(others, o)
				}
			}
			note := fmt.Sprintf("coarse linkage: codex session root also dispatches %s — thread actuals link by root id only (D20) and may be shared across these tickets", strings.Join(others, ", "))
			if file != "" {
				note += " (source: " + file + ")"
			}
			notes[id] = note
		}
	}
	return notes
}

// judge decides one ticket's verdict from its declared tier, its observed
// evidence tokens, and the ledger records. Each token resolves within its
// own flavor's table (I040): judge itself never picks a flavor.
func judge(t ticket, tokens []evidenceToken, mappings map[string]map[string]resolvedTier, l ledger) (Verdict, string) {
	if t.tier == "" {
		return VerdictUnannotated, "no tier annotation — not judged"
	}
	// D27 (ticket I046): tier: n/a is a declared decision to opt out of
	// routing judgment, mirroring the review-tier: n/a convention — distinct
	// from an absent annotation, which stays unannotated (a gap, not a
	// decision).
	if t.tier == "n/a" {
		return VerdictExempt, "tier: n/a — exempt from routing judgment"
	}
	if _, known := tierRank[t.tier]; !known {
		return VerdictUnannotated, fmt.Sprintf("unknown tier %q — not judged", t.tier)
	}
	if len(tokens) == 0 {
		return VerdictNoTranscript, "no dispatch or transcript evidence found"
	}
	verdict, detail := VerdictMatch, ""
	worse := func(v Verdict, d string) {
		if severity[v] > severity[verdict] {
			verdict, detail = v, d
		}
	}
	// matchSources collects the source files of codex tokens that
	// individually judged match (D24 AC: every judged codex verdict,
	// including match, names its source file). judge()'s worst-token
	// aggregation has no detail to carry for a plain match — a matching
	// token's own judgeToken detail is "" — so matched codex sources are
	// gathered separately and only surface here when the ticket's final,
	// aggregate verdict is itself Match (i.e. every token matched: any
	// worse per-token verdict would already have won via worse() above).
	var matchSources []string
	for _, tok := range tokens {
		v, d := judgeToken(tok, t, mappings, l)
		worse(v, d)
		if v == VerdictMatch && tok.flavor == "codex" && tok.sourceFile != "" {
			matchSources = append(matchSources, tok.sourceFile)
		}
	}
	if verdict == VerdictMatch && len(matchSources) > 0 {
		detail = "source: " + strings.Join(dedupSorted(matchSources), ", ")
	}
	return verdict, detail
}

// withSource appends a codex evidence token's source transcript file to a
// judged detail line (D24: every judged codex verdict names its source, the
// I008 silent-descent requirement satisfied as a special case). Claude
// tokens never carry a sourceFile (readTranscripts/scanJSONL/parseLine never
// set one), so this is a no-op for every claude-flavor call — the guarantee
// that claude verdict details stay byte-identical.
func withSource(detail string, tok evidenceToken) string {
	if tok.flavor != "codex" || tok.sourceFile == "" {
		return detail
	}
	return detail + " (source: " + tok.sourceFile + ")"
}

// judgeToken classifies a single observed evidence token against the
// ticket's declared tier, resolving the token within its own flavor's table
// — the per-token seam design D15 names and I040 makes real. A token whose
// flavor has no resolved table (unreachable while only the claude reader is
// wired) is treated the same as one that maps to no entry: unmapped, never a
// crash or a silent match.
func judgeToken(tok evidenceToken, t ticket, mappings map[string]map[string]resolvedTier, l ledger) (Verdict, string) {
	mapping := mappings[tok.flavor]
	tiers := tiersOf(tok.value, mapping)
	if len(tiers) == 0 {
		return VerdictUnmappedDispatch, withSource(fmt.Sprintf("%s maps to no %s entry in the model table", tok.value, tok.flavor), tok)
	}
	_, recordedFallback := l.fallback[t.id]
	actual := pickTier(tiers, t.tier, recordedFallback)
	if actual == t.tier {
		return VerdictMatch, ""
	}
	if actual == "fallback" { // lateral, never descent (see package doc)
		if reason, ok := l.fallback[t.id]; ok {
			return VerdictEscalatedWithReason, withSource(fmt.Sprintf("%s (fallback) — FALLBACK reason: %s", tok.value, reason), tok)
		}
		return VerdictUnexplainedFallback, withSource(fmt.Sprintf("%s (fallback) without a FALLBACK record or fallback annotation", tok.value), tok)
	}
	for _, rec := range l.escalation[t.id] {
		if rec.to == actual { // a record excuses exactly its to-tier
			return VerdictEscalatedWithReason, withSource(fmt.Sprintf("%s (%s) vs declared %s — ESCALATION reason: %s", tok.value, actual, t.tier, rec.reason), tok)
		}
	}
	if t.tier != "fallback" && tierRank[actual] < tierRank[t.tier] {
		return VerdictSilentDescent, withSource(fmt.Sprintf("%s (%s) below declared %s with no ESCALATION record", tok.value, actual, t.tier), tok)
	}
	return VerdictEscalatedNoReason, withSource(fmt.Sprintf("%s (%s) above declared %s with no ESCALATION record", tok.value, actual, t.tier), tok)
}

// tiersOf resolves a model token to every tier it could mean within one
// flavor's resolved table: exact match on the resolved id, on an explicitly
// declared alias, or on an id the entry's default ever shipped (historical
// ids carry no aliases — a pre-refresh transcript matches by full id only).
// Substring containment is retired (design D13).
func tiersOf(token string, mapping map[string]resolvedTier) []string {
	var tiers []string
	for tier, rt := range mapping {
		if rt.matches(token) {
			tiers = append(tiers, tier)
		}
	}
	sort.Strings(tiers)
	return tiers
}

// pickTier chooses the reading of an ambiguous token — one that maps to
// several tiers because they share an id or alias, which the table permits
// (the shipped codex.routine/codex.fallback "terra" pair is the live case;
// design D15's flavor scoping decides between flavors, this rule decides
// within one): the declared tier if it is among the candidates; else, when
// the ticket carries a FALLBACK record and fallback is among the
// candidates, fallback (ADR 0012 / D25 — a recorded refusal-rerun wins the
// ambiguity before the ordered reading gets a look, so it never stands as a
// false silent-descent blocker); else the highest-ranked ordered candidate;
// else fallback. Ambiguity must not manufacture a verdict.
func pickTier(tiers []string, declared string, recordedFallback bool) string {
	for _, tier := range tiers {
		if tier == declared {
			return tier
		}
	}
	if recordedFallback {
		for _, tier := range tiers {
			if tier == "fallback" {
				return tier
			}
		}
	}
	best := ""
	for _, tier := range tiers {
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

// anyAnnotated reports whether at least one ticket carries a tier:
// annotation. A repo where none do audits vacuously: every row comes back
// unannotated and unjudged, and the CLI still exits 0 — indistinguishable
// from a real clean pass unless the report says so.
func anyAnnotated(tickets []ticket) bool {
	for _, t := range tickets {
		if t.tier != "" {
			return true
		}
	}
	return false
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

// transcriptFlavor derives the flavor of one transcript source — THE
// flavor-derivation seam (design D15): flavor comes from the transcript
// source, never from the ticket or the table. readTranscripts tags every
// dispatch and subagent record it reads from that source with this value,
// and it rides along inside evidenceToken all the way into judgeToken
// (I040), so mixed builds judge each token within its own source's flavor.
// It covers the claude harness's ~/.claude/projects layout; the codex-audit
// effort (I041, codex.go) tags its own records "codex" directly in
// readCodexSessions rather than calling this — judge and judgeToken needed
// no further change to pick up that second source.
func transcriptFlavor(transcriptsDir string) string {
	return "claude"
}

// resolvedTier is the audit's view of one (flavor, tier) row, obtained
// through the shared resolver (design D13) so the audit judges exactly what
// dispatch-time resolution returns — the audit owns no WORKFLOW.md routing
// parser of its own.
type resolvedTier struct {
	id      string   // resolved model id: the repo's mirror value if present, else the embedded default
	aliases []string // the table entry's explicitly declared aliases
	history []string // ids this entry shipped as defaults before the current one (exact-match only)
}

func (rt resolvedTier) matches(token string) bool {
	if token == rt.id {
		return true
	}
	for _, a := range rt.aliases {
		if token == a {
			return true
		}
	}
	for _, h := range rt.history {
		if token == h {
			return true
		}
	}
	return false
}

// resolveFlavorTiers builds one flavor's tier -> resolvedTier table for
// repoDir via model.Resolve. An error is a broken embedded table or an
// unknown flavor — never a repo state — and refuses the audit outright:
// judging a fleet against a half-resolved table is exactly the confident
// misparse D13/D14 exist to prevent. (Unreachable today: the embedded table
// is load-time validated and transcriptFlavor only names shipped flavors.)
func resolveFlavorTiers(repoDir, flavor string) (map[string]resolvedTier, error) {
	mapping := map[string]resolvedTier{}
	for _, tier := range model.Tiers {
		e, err := model.Resolve(repoDir, flavor, tier)
		if err != nil {
			return nil, fmt.Errorf("model table resolution failed for %s.%s: %w", flavor, tier, err)
		}
		rt := resolvedTier{id: e.ID, aliases: e.Aliases}
		// Provenance-scoped matching (I037 fix round 1): a deliberate
		// Override matches by its exact on-disk id only — the resolver
		// already withholds the shipped aliases (e.Aliases is nil), and the
		// shipped historical ids are withheld here for the same reason: in a
		// repo that pinned bespoke-x, a dispatch on the displaced default is
		// the drift the override makes visible, and matching it through the
		// default's lineage would judge it a clean pass. Default and
		// Inherited entries keep both — an inherited claude-opus-4-8 must
		// keep matching its current default's aliases and history.
		if e.Provenance != model.Override {
			rt.history = model.HistoricalIDs(flavor, tier)
		}
		mapping[tier] = rt
	}
	return mapping, nil
}

// gateTemplateVersion refuses a WORKFLOW.md stamped with a template
// generation newer than this binary compiles (design D14) — the same gate
// spine update applies, for the same reason: an un-upgraded binary reading
// a newer format must fail loudly, not emit confident verdicts from a
// misparse. Non-integer stamps fall through, matching update. An unreadable
// WORKFLOW.md is a warning, not an error: the resolver falls back to the
// embedded defaults and the report says so — as it also does when the file
// is readable but a gen 6+ stamp sits over a missing model_routing block.
func gateTemplateVersion(repoDir string, warnings *[]string) error {
	raw, err := os.ReadFile(filepath.Join(repoDir, "WORKFLOW.md"))
	if err != nil {
		*warnings = append(*warnings, "WORKFLOW.md unreadable — routing resolved from embedded defaults: "+err.Error())
		return nil
	}
	content := string(raw)
	if tv := update.ExtractKeys(content)["template_version"]; tv != "" {
		if n, err := strconv.Atoi(tv); err == nil {
			if n > tmpl.Version() {
				return fmt.Errorf(
					"WORKFLOW.md is template generation %d but this spine binary compiles generation %d — refusing to audit a newer format; upgrade spine (make install in ~/Projects/github.com/spine)",
					n, tmpl.Version())
			}
			// Every gen 6+ template renders the model_routing mirror, so a
			// gen 6+ stamp over an empty block means the spine-managed
			// mirror was lost (bad merge, hand edit). Verdicts stay faithful
			// to dispatch-time resolution either way (D13) — this warning is
			// the diagnostics residue of the retired loud-failure mode: the
			// gate must not report a clean bill indistinguishable from a
			// healthy repo's (I037 fix round 1, finding I-1).
			if n >= 6 && len(model.RoutingKeys(content)) == 0 {
				*warnings = append(*warnings, "WORKFLOW.md carries no model_routing block — routing resolved from embedded defaults")
			}
		}
	}
	return nil
}

// stripComment trims a value and drops any trailing "# comment". It serves
// ticket FRONTMATTER values only — deliberately not the WORKFLOW.md comment
// rule (model.CommentIndex): frontmatter is a different surface with its own
// historical naive rule, and no routing value ever passes through here.
func stripComment(v string) string {
	if i := strings.Index(v, "#"); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}

// escRecord is one parsed model-tier ESCALATION ledger record; it excuses
// dispatches on exactly its to-tier.
type escRecord struct {
	to     string
	reason string
}

type ledger struct {
	escalation map[string][]escRecord // ticket id -> model-tier escalation records
	fallback   map[string]string      // ticket id -> fallback reason
}

// readLedger scans the build ledger for the pinned one-line grammar:
//
//	ESCALATION <ticket-id> <from>-><to> reason: <one line>
//	ESCALATION <ticket-id> effort <from>-><to> reason: <one line>
//	FALLBACK <ticket-id> reason: <one line>
//
// A missing ledger is normal (records are then simply absent). Effort
// escalations are parsed and deliberately unused: they justify effort, not
// model tier. A model-tier record keeps its to-tier — it excuses dispatches
// on that tier only.
func readLedger(path string) ledger {
	l := ledger{escalation: map[string][]escRecord{}, fallback: map[string]string{}}
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
			rest = strings.TrimSpace(rest)
			if strings.HasPrefix(rest, "effort ") {
				continue // effort record: not model evidence
			}
			fromTo, _, _ := strings.Cut(rest, " ")
			_, to, ok := strings.Cut(fromTo, "->")
			if !ok {
				continue // no <from>-><to> tiers: excuses nothing
			}
			l.escalation[id] = append(l.escalation[id], escRecord{to: strings.TrimSpace(to), reason: reason})
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
	effort      string // declared worker effort, claude-team spawns only (I090); reported, never judged — see DispatchInfo.Effort
	flavor      string // the transcript source's flavor (I040 per-token seam)
	sourceFile  string // source transcript file (D24, codex only); "" for claude
	cwd         string // D28 (I047): the event line's own cwd, claude only; "" for codex (D22 scopes it separately)

	// teamSpawn marks this record as a claude-team worker spawn (I090) —
	// see DispatchInfo.TeamSpawn for what an unmatched one means.
	teamSpawn bool

	// teamTarget is the worker handle a claude-team spawn addressed (I090):
	// the herdr agent name or the cmux pane. It is how a spawn that named
	// no ticket is paired with the following prompt command that did.
	teamTarget string
}

type subagent struct {
	toolUseID   string
	description string
	models      []string
	flavor      string // the transcript source's flavor (I040 per-token seam)
	sourceFile  string // source transcript file (D24, codex only); "" for claude
	cwd         string // D28 (I047): the subagent transcript's own session cwd, claude only
}

// repoQualifies implements D28's claude-side repo-qualification rule
// (ticket I047, the I008 incident's fix): text — a dispatch's own
// description+prompt, or a subagent's own description — claims the audited
// repo iff it names the repo's absolute path, names the repo's basename as
// a whole token, or cwd (the claiming record's own session cwd) resolves
// inside the repo. Any one of the three suffices; none of them is
// privileged over the others, matching the ticket's "OR" phrasing.
// Basename matching is skipped when repoBase is empty (Base of a failed
// Abs can't be trusted). A dispatch/subagent that fails all three does not
// vanish — its caller (matches, or the agent direct-match branch) simply
// declines to attribute it to this ticket, and it surfaces in the report's
// existing unmatched informational list rather than being silently
// dropped.
func repoQualifies(text, cwd, absRepoDir, repoBase string) bool {
	if absRepoDir != "" && containsPathToken(text, absRepoDir) {
		return true
	}
	if repoBase != "" && containsPathToken(text, repoBase) {
		return true
	}
	return cwdInsideRepo(cwd, absRepoDir)
}

// isPathWordChar extends isAlnum with the characters that routinely appear
// WITHIN a single path component or repo name — '-', '_', '.' — so a
// boundary check built on it treats "praxis-web" as one unbroken word
// rather than "praxis" + a boundary + "-web". Used only by repoQualifies'
// two text clauses (I047 review C1); containsToken's ticket-id matching
// stays alnum-only and untouched — ticket ids never carry these
// characters, and loosening that boundary was never in scope.
func isPathWordChar(c byte) bool {
	return isAlnum(c) || c == '-' || c == '_' || c == '.'
}

// containsPathToken implements repoQualifies' two boundary-aware text
// clauses (I047 review C1, amended D28): text references path as a whole
// path-word — the absolute-path clause and the basename clause both use
// this, since a repo's basename is exactly a one-component path. Without
// this, `strings.Contains`/`containsToken`'s alnum-only boundary let
// "praxis" match inside "praxis-web" (a real sibling-repo directory name,
// hyphen not being alnum) — readmitting the exact I008 cross-repo
// collision class for any two repos sharing a name prefix. Boundary is
// checked on BOTH sides: the amended design text only mandates an
// after-boundary for the path clause, but requiring both is strictly
// safer (a legitimate reference is never itself preceded by a path-word
// character in practice — that would mean matching a longer, unrelated
// path) and keeps one mechanism for both clauses instead of two.
func containsPathToken(text, path string) bool {
	return containsTokenWith(text, path, isPathWordChar)
}

// containsTokenWith generalizes containsToken's word-boundary matching
// over a caller-supplied word-character predicate — containsToken itself
// stays pinned to isAlnum (ticket ids), while containsPathToken supplies
// isPathWordChar (repo path/basename references, I047 review C1).
func containsTokenWith(text, id string, isWordChar func(byte) bool) bool {
	if id == "" {
		return false
	}
	for start := 0; ; {
		i := strings.Index(text[start:], id)
		if i < 0 {
			return false
		}
		i += start
		before := i == 0 || !isWordChar(text[i-1])
		afterIdx := i + len(id)
		after := afterIdx >= len(text) || !isWordChar(text[afterIdx])
		if before && after {
			return true
		}
		start = i + 1
	}
}

// parseSince implements the --since operator escape hatch (D28, ticket
// I047): a fixed cutoff, never a relative duration or an automatic
// build-start anchor (design D-doc: "No started-date anchoring... the
// wrong default for the estate"). Two formats, both unambiguous and
// deterministic to parse — no reliance on the audit's own wall-clock at
// run time, so the same --since string always scopes the same way: full
// RFC3339 (2026-07-20T15:04:05Z, for an operator with a precise cutoff —
// e.g. copied from a session's own timestamp) or a bare YYYY-MM-DD date,
// read as local midnight (for an operator who thinks in calendar days —
// "since Monday"). Filtering compares this cutoff against each transcript
// session's own mtime (readTranscripts/readCodexSessions), not an in-JSONL
// timestamp: both harness formats are undocumented and drift (package
// doc), but a file's mtime needs no format knowledge at all, and it is
// exactly the signal the I008 workaround approximated by hand (copying a
// build's session files to a scratch dir). A value that matches neither
// format is a usage error at the Run boundary (I047 review ruling 4) —
// this function only parses, it never decides how its error is handled.
func parseSince(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("not RFC3339 or YYYY-MM-DD: %q", s)
}

// sessionFiles is one claude session's two possible on-disk pieces — its
// top-level "<id>.jsonl" file and its "<id>/subagents" dir — grouped under
// one session id so --since/--session scope the WHOLE session as a unit
// (I047 review I2), never the two pieces independently.
type sessionFiles struct {
	filePath string // "<id>.jsonl"; "" if absent
	dirPath  string // "<id>"; "" if absent
}

// sessionInScope applies the D28 --since/--session filters (ticket I047,
// unified per-session at I047 review finding I2) to one claude session id.
// --session is exact id equality. --since compares the cutoff against the
// LATER of the file's and dir's mtimes (whichever exist) — deciding scope
// once per session, not once per on-disk piece: a cutoff falling between
// the dir's mtime (often stamped near session start, when subagents/ is
// first created) and the file's mtime (kept moving as the top-level
// session is appended to) must not silently keep the session's declared
// dispatch aliases while dropping its subagent actuals — exactly the
// evidence that catches a worker running a lower model than declared. A
// session with no readable mtime on either piece is never excluded by a
// since filter it can't evaluate — degrade toward inclusion, matching the
// package's everything-is-a-warning-not-a-loss posture elsewhere.
func sessionInScope(sf sessionFiles, mtime func(string) (time.Time, bool), since time.Time, id, wantID string) bool {
	if wantID != "" && id != wantID {
		return false
	}
	if since.IsZero() {
		return true
	}
	var latest time.Time
	have := false
	for _, p := range []string{sf.filePath, sf.dirPath} {
		if p == "" {
			continue
		}
		if t, ok := mtime(p); ok {
			if !have || t.After(latest) {
				latest = t
			}
			have = true
		}
	}
	return !have || !latest.Before(since)
}

// readTranscripts collects Task/Agent dispatch records from every session
// *.jsonl and actual models from <session>/subagents/agent-*.jsonl, linked
// by the sidecar meta.json. Every record it emits is tagged with flavor —
// the claude source's flavor for every call today, and the seam later
// readers (codex) tag with their own. since and sessionID implement D28's
// --since/--session filters (ticket I047), applied once per session id
// (I2 fix — see sessionFiles/sessionInScope): a zero since and empty
// sessionID (every pre-I047 caller) filter nothing, keeping every existing
// behavior byte-identical. matchedSession reports whether sessionID (when
// non-empty) equaled at least one discovered session id — independent of
// whether --since then excluded it — the diagnostic input for M3's
// "matched no sessions" warning; always true when sessionID is empty (no
// filter to fail to match). All trouble becomes warnings.
func readTranscripts(dir, flavor string, since time.Time, sessionID string, warnings *[]string) ([]dispatch, []subagent, bool) {
	des, err := os.ReadDir(dir)
	if err != nil {
		*warnings = append(*warnings, "transcript dir unreadable — all tickets will report no-transcript: "+err.Error())
		return nil, nil, sessionID == ""
	}
	sessions := map[string]*sessionFiles{}
	var ids []string
	get := func(id string) *sessionFiles {
		sf, ok := sessions[id]
		if !ok {
			sf = &sessionFiles{}
			sessions[id] = sf
			ids = append(ids, id)
		}
		return sf
	}
	for _, de := range des {
		name := de.Name()
		if de.IsDir() {
			get(name).dirPath = filepath.Join(dir, name)
			continue
		}
		if strings.HasSuffix(name, ".jsonl") {
			get(strings.TrimSuffix(name, ".jsonl")).filePath = filepath.Join(dir, name)
		}
	}
	sort.Strings(ids)

	matchedSession := sessionID == ""
	var dispatches []dispatch
	var agents []subagent
	for _, id := range ids {
		sf := *sessions[id]
		if sessionID != "" && id == sessionID {
			matchedSession = true
		}
		if !sessionInScope(sf, fileMTime, since, id, sessionID) {
			continue
		}
		if sf.filePath != "" {
			more, _, _ := scanJSONL(sf.filePath, warnings)
			for i := range more {
				more[i].flavor = flavor
			}
			dispatches = append(dispatches, more...)
		}
		if sf.dirPath != "" {
			subDir := filepath.Join(sf.dirPath, "subagents")
			subs, _ := filepath.Glob(filepath.Join(subDir, "agent-*.jsonl"))
			sort.Strings(subs)
			for _, sub := range subs {
				a := subagent{flavor: flavor}
				if metaRaw, err := os.ReadFile(strings.TrimSuffix(sub, ".jsonl") + ".meta.json"); err == nil {
					var meta struct {
						ToolUseID   string `json:"toolUseId"`
						Description string `json:"description"`
					}
					if json.Unmarshal(metaRaw, &meta) == nil {
						a.toolUseID, a.description = meta.ToolUseID, meta.Description
					}
				}
				more, models, cwd := scanJSONL(sub, warnings)
				for i := range more {
					more[i].flavor = flavor
				}
				a.models = models
				a.cwd = cwd
				dispatches = append(dispatches, more...)
				agents = append(agents, a)
			}
		}
	}
	return dispatches, agents, matchedSession
}

// fileMTime is sessionInScope's --since probe (D28, ticket I047): a file or
// dir whose mtime can't be read (permission trouble, a race with deletion)
// is reported as unknown, never a false cutoff match — sessionInScope then
// includes it, matching the package's degrade-toward-inclusion posture for
// operator-filter mechanics that fail closed would silently under-report.
func fileMTime(path string) (time.Time, bool) {
	fi, err := os.Stat(path)
	if err != nil {
		return time.Time{}, false
	}
	return fi.ModTime(), true
}

// scanJSONL extracts dispatch tool_use records, distinct assistant model
// ids, and the session's own cwd (D28, ticket I047 — the first non-empty
// "cwd" seen on any line; real transcripts carry it on every line, so the
// first is as good as any) from one transcript file. Malformed lines are
// counted into a single per-file warning; they never fail the audit.
func scanJSONL(path string, warnings *[]string) ([]dispatch, []string, string) {
	f, err := os.Open(path)
	if err != nil {
		*warnings = append(*warnings, path+": unreadable: "+err.Error())
		return nil, nil, ""
	}
	defer func() {
		if err := f.Close(); err != nil {
			*warnings = append(*warnings, path+": close: "+err.Error())
		}
	}()
	var dispatches []dispatch
	var models []string
	var cwd string
	seen := map[string]bool{}
	malformed := 0
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadBytes('\n')
		if len(strings.TrimSpace(string(line))) > 0 {
			d, prompts, m, lineCwd, ok := parseLine(line)
			if !ok {
				malformed++
			} else {
				dispatches = append(dispatches, d...)
				for _, p := range prompts {
					attributeTeamPrompt(dispatches, p)
				}
				if m != "" && !seen[m] {
					seen[m] = true
					models = append(models, m)
				}
				if cwd == "" && lineCwd != "" {
					cwd = lineCwd
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
	return dispatches, models, cwd
}

// parseLine reads one transcript event. Only assistant events matter: their
// message.model is the actual, and Task/Agent tool_use blocks are dispatch
// records — each stamped with this line's own top-level "cwd" (D28, ticket
// I047: verified present on both user and assistant lines in real
// ~/.claude session and subagent transcripts). Bash tool_use blocks running
// a claude-team worker spawn are dispatch records too (I090, teamspawn.go),
// and the prompt commands that follow them are returned alongside for the
// caller to pair. Unrecognized JSON shapes report as malformed (ok=false).
func parseLine(line []byte) (dispatches []dispatch, prompts []teamPrompt, model string, cwd string, ok bool) {
	var ev struct {
		Type    string `json:"type"`
		Cwd     string `json:"cwd"`
		Message struct {
			Model   string          `json:"model"`
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(line, &ev) != nil {
		return nil, nil, "", "", false
	}
	cwd = ev.Cwd
	if ev.Type != "assistant" {
		return nil, nil, "", cwd, true
	}
	var blocks []struct {
		Type  string `json:"type"`
		ID    string `json:"id"`
		Name  string `json:"name"`
		Input struct {
			Description string `json:"description"`
			Prompt      string `json:"prompt"`
			Model       string `json:"model"`
			Command     string `json:"command"`
		} `json:"input"`
	}
	if len(ev.Message.Content) > 0 && json.Unmarshal(ev.Message.Content, &blocks) != nil {
		return nil, nil, ev.Message.Model, cwd, false // assistant event of unrecognized shape
	}
	for _, b := range blocks {
		if b.Type != "tool_use" {
			continue
		}
		switch {
		case b.Name == "Task" || b.Name == "Agent":
			dispatches = append(dispatches, dispatch{
				toolUseID:   b.ID,
				description: b.Input.Description,
				prompt:      b.Input.Prompt,
				model:       b.Input.Model,
				cwd:         cwd,
			})
		case recognizeTeamSpawns && b.Name == "Bash":
			if s, isSpawn := parseTeamSpawn(b.Input.Command); isSpawn {
				dispatches = append(dispatches, dispatch{
					toolUseID: b.ID,
					// The record carries the HEREDOC-STRIPPED command
					// (final review C1). A lead routinely writes the
					// worker's brief and starts the worker in one Bash
					// call; the brief names the ticket under work and
					// often several more "for context". Attributing on
					// the raw command text made every one of those a
					// match — certifying work nobody dispatched as
					// routed. matches() and namesATicket must see exactly
					// the text the recognizer accepted as a command, so
					// the two can never disagree about what was run.
					description: stripHeredocBodies(b.Input.Command),
					model:       s.model,
					effort:      s.effort,
					cwd:         cwd,
					teamSpawn:   true,
					teamTarget:  s.target,
				})
				continue
			}
			if p, isPrompt := parseTeamPrompt(b.Input.Command); isPrompt {
				prompts = append(prompts, p)
			}
		}
	}
	return dispatches, prompts, ev.Message.Model, cwd, true
}

// --- helpers ---

// containsToken reports whether text contains id as a whole token (so I20
// never matches a dispatch for I201).
func containsToken(text, id string) bool {
	return containsTokenWith(text, id, isAlnum)
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
