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
//     resolution is harness-scoped and goes through model.Resolve, honoring
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
//   - Harness of a dispatch comes from its observed model id when that id maps
//     to exactly one resolved harness (I111, extending design D15). An id
//     shared across harnesses, or absent from all of them, retains the
//     transcript-derived harness. Source and harness travel separately with
//     the evidence token: harness selects the model table, while source keeps
//     transcript-layout behavior such as D28 repo qualification and codex
//     source-file disclosure. The worker-session scan
//     (D21, ticket I042) and the git-commit-probe repo scoping (D22, ticket
//     I043) both shipped in codex.go: codex evidence covers dispatch
//     records, linkable spawned-thread actuals, and top-level orchestrator
//     sessions attributed via D21's opening-line rule, all repo-scoped by
//     D22's cwd/commit-hash checks.
//   - Token -> tier, within the dispatch's harness: a token maps to a tier
//     when it equals the resolved id, one of the table entry's explicitly
//     declared aliases, or an id the entry shipped as a default before the
//     current one (historical ids carry no aliases, so a pre-refresh
//     transcript — e.g. a claude-opus-4-8 dispatch — matches by full id
//     only). An Override entry matches by its exact on-disk id alone:
//     aliases and history describe the shipped defaults, and a dispatch on
//     the displaced default in a repo that pinned something else must
//     surface as unmapped, not read as a match (Default and Inherited
//     entries keep both). Substring containment is retired (design D13): it collides as
//     model names multiply across harnesses with unrelated naming schemes.
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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/russellpope/spine/internal/hostconfig"
	"github.com/russellpope/spine/internal/model"
	"github.com/russellpope/spine/internal/ticketref"
	"github.com/russellpope/spine/internal/tmpl"
	"github.com/russellpope/spine/internal/update"
)

// Verdict classifies one ticket's declared-vs-actual routing.
type Verdict string

// Verdict values, worst first.
const (
	VerdictSilentDescent               Verdict = "silent-descent"             // blocking
	VerdictDeclarationObservedMismatch Verdict = "declared-observed-mismatch" // blocking
	VerdictDeclarationEffortMismatch   Verdict = "declared-effort-mismatch"   // blocking
	VerdictDeclarationUnconfirmable    Verdict = "unconfirmable"              // advisory
	VerdictDeclarationConfirmed        Verdict = "confirmed"                  // informational
	VerdictUnmappedDispatch            Verdict = "unmapped-dispatch"          // warn
	VerdictUnexplainedFallback         Verdict = "unexplained-fallback"       // warn
	VerdictEscalatedNoReason           Verdict = "escalated-no-reason"        // warn
	VerdictNoTranscript                Verdict = "no-transcript"              // warn
	VerdictUnattributedTranscript      Verdict = "unattributed-transcript"    // warn (D24, ticket I044)
	VerdictEscalatedWithReason         Verdict = "escalated-with-reason"      // advisory
	VerdictDiscardedWithReason         Verdict = "discarded-with-reason"      // advisory
	VerdictMatch                       Verdict = "match"
	VerdictExempt                      Verdict = "exempt"      // informational (D27, ticket I046): tier: n/a opts out
	VerdictUnannotated                 Verdict = "unannotated" // informational
)

// severity orders verdicts for worst-token aggregation; higher is worse.
var severity = map[Verdict]int{
	VerdictMatch:                       0,
	VerdictDeclarationConfirmed:        0,
	VerdictEscalatedWithReason:         1,
	VerdictDiscardedWithReason:         1,
	VerdictEscalatedNoReason:           2,
	VerdictUnexplainedFallback:         3,
	VerdictUnmappedDispatch:            4,
	VerdictSilentDescent:               5,
	VerdictDeclarationUnconfirmable:    2,
	VerdictDeclarationObservedMismatch: 6,
	VerdictDeclarationEffortMismatch:   6,
}

// TicketRow is one ticket's audit outcome.
type TicketRow struct {
	ID                string
	Tier              string // declared tier annotation; "" if absent
	Actuals           []string
	Verdict           Verdict
	Detail            string
	ExpectedEffort    string
	DeclaredEffort    string
	DeclarationStatus string
	ObservedEffort    string
	DeclarationEvents []DeclarationEvidence
}

// DispatchInfo is an informational, never-judged dispatch record.
type DispatchInfo struct {
	Description string
	Harness     string
	Model       string
	// Effort is the worker effort a claude-team spawn declared (I090); ""
	// for every other dispatch shape. It is reported, never judged: the
	// routing contract's enforcement is model-vs-tier, and comparing effort
	// against a ticket's effort: frontmatter would be a second, separate
	// contract. Out of scope for I090 by design, not an oversight.
	Effort       string
	EffortSource string
	// TeamSpawn marks a claude-team worker spawn (I090). An unmatched one
	// is the residual blind spot the ticket's footer clause was written
	// for: the spawn was recognized but could not be attributed to any
	// ticket — no ticket token in the command or its prompt, a token
	// naming another repo, or a command that fails repo qualification.
	// The most common cause on real input is a brief delivered by file
	// reference (`"$(cat …dispatch-task-2.md)"`), whose expansion the
	// transcript never records; ticket I101 tracks that one. Counted in
	// the report footer so the gap stays visible rather than silent — the
	// footer states the fact, not a guessed cause.
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
		if t.Verdict == VerdictSilentDescent || t.Verdict == VerdictDeclarationObservedMismatch || t.Verdict == VerdictDeclarationEffortMismatch {
			return true
		}
	}
	return false
}

// tier order: mechanical < routine < primary; fallback is lateral (rank 0).
var tierRank = map[string]int{"mechanical": 1, "routine": 2, "primary": 3, "fallback": 0}

// evidenceToken is one observed model string paired with its model-derived
// harness and transcript source. judgeToken resolves value within harness's
// table alone; source controls transcript-layout behavior outside resolution.
type evidenceToken struct {
	value      string
	harness    string
	source     string
	sourceFile string // source transcript file (D24, codex only); "" for claude, keeping claude details byte-identical
	identity   evidenceIdentity
}

// evidenceIdentity is the immutable dispatch-event correlation key used by
// DISCARDED records. A partial key is deliberately unusable: source/session
// alone is too broad to distinguish a discarded prototype from later work.
type evidenceIdentity struct {
	source   string
	session  string
	dispatch string
}

func (i evidenceIdentity) usable() bool {
	return i.source != "" && i.session != "" && i.dispatch != ""
}

// DeclarationEvidence keeps the separate raw facts needed to explain an
// I074 declaration judgment. Empty observation fields mean no transcript
// evidence was available; they never stand for a transport default.
type DeclarationEvidence struct {
	Identity             evidenceIdentity
	Harness              string
	Model                string
	ExpectedModel        string
	ExpectedEffort       string
	DeclaredEffort       string
	ObservedModel        string
	ObservedEffort       string
	ModelStatus          DeclarationModelState
	DeclarationStatus    string
	ObservedEffortStatus string
	Correlation          string
	Verdict              Verdict
}

// DeclarationModelState keeps model evidence separate from the final verdict.
type DeclarationModelState string

const (
	DeclarationModelConfirmed     DeclarationModelState = "confirmed"
	DeclarationModelMismatch      DeclarationModelState = "mismatch"
	DeclarationModelUnconfirmable DeclarationModelState = "unconfirmable"
)

type observedRoute struct {
	harness string
	model   string
	effort  string
}

// observedRouteIndex is deliberately host-local. It is built exclusively
// from the validated I072 configuration and has no alias/history fallback.
type observedRouteIndex map[string]observedRoute

func newObservedRouteIndex(config hostconfig.Config) observedRouteIndex {
	routes := observedRouteIndex{}
	for harnessName, harness := range config.Harnesses {
		for modelID, route := range harness.Models {
			for _, observedID := range route.ObservedIDs {
				routes[observedID] = observedRoute{harness: harnessName, model: modelID}
			}
		}
	}
	return routes
}

type declarationObservation struct {
	identity     evidenceIdentity
	model        string
	effort       string
	linkedWorker bool
}

// judgeDeclarationModel permits confirmation only from one linked worker
// event with the complete controller dispatch identity. A root/session match
// is intentionally insufficient, even when its raw model happens to match.
func judgeDeclarationModel(declared DeclarationEvidence, observed declarationObservation, routes observedRouteIndex) DeclarationModelState {
	if !declared.Identity.usable() || !observed.identity.usable() || !observed.linkedWorker || declared.Identity != observed.identity || declared.Harness == "" || observed.model == "" {
		return DeclarationModelUnconfirmable
	}
	route, ok := routes[observed.model]
	if !ok || route.harness != declared.Harness {
		return DeclarationModelUnconfirmable
	}
	if declared.Model != declared.ExpectedModel || route.model != declared.ExpectedModel {
		return DeclarationModelMismatch
	}
	return DeclarationModelConfirmed
}

// judgeDeclarationEvidence is deliberately fact-only: callers provide any
// documented observation through the narrow seam, while production keeps
// ObservedEffort absent until a harness documents an exact extractor.
func judgeDeclarationEvidence(evidence DeclarationEvidence, authorized bool) DeclarationEvidence {
	if evidence.ObservedEffort == "" {
		evidence.ObservedEffort = "-"
	}
	evidence.ObservedEffortStatus = "unconfirmable"
	evidence.DeclarationStatus = "unconfirmable"
	evidence.Verdict = VerdictDeclarationUnconfirmable
	if evidence.DeclaredEffort == "" || evidence.ExpectedEffort == "" {
		return evidence
	}
	if evidence.DeclaredEffort == evidence.ExpectedEffort {
		evidence.DeclarationStatus = "target-match"
	} else if authorized {
		evidence.DeclarationStatus = "exact-authorized-deviation"
	} else {
		evidence.DeclarationStatus = "unauthorized-declaration"
	}
	if evidence.ModelStatus == DeclarationModelMismatch {
		evidence.Verdict = VerdictDeclarationObservedMismatch
		return evidence
	}
	if evidence.DeclarationStatus == "unauthorized-declaration" {
		evidence.Verdict = VerdictDeclarationEffortMismatch
		return evidence
	}
	if evidence.ModelStatus != DeclarationModelConfirmed || evidence.ObservedEffort == "-" {
		return evidence
	}
	if evidence.ObservedEffort != evidence.DeclaredEffort {
		evidence.ObservedEffortStatus = "mismatch"
		evidence.Verdict = VerdictDeclarationObservedMismatch
		return evidence
	}
	evidence.ObservedEffortStatus = "confirmed"
	evidence.Verdict = VerdictDeclarationConfirmed
	return evidence
}

func judgeHostDeclarations(repoDir, hostPath string, t ticket, dispatches []dispatch, agents []subagent, routes observedRouteIndex, l ledger) []DeclarationEvidence {
	if t.tier == "" || t.tier == "n/a" {
		return nil
	}
	var out []DeclarationEvidence
	for _, d := range dispatches {
		if d.harness == "" || d.model == "" || d.effort == "" || !d.identity.usable() {
			continue
		}
		evidence := DeclarationEvidence{
			Identity:       d.identity,
			Harness:        d.harness,
			Model:          d.model,
			DeclaredEffort: d.effort,
			ObservedEffort: "-",
			ModelStatus:    DeclarationModelUnconfirmable,
			Correlation:    formatEvidenceIdentity(d.identity),
		}
		resolution, err := model.ResolveForHost(repoDir, hostPath, d.harness, t.tier, nil)
		if err == nil {
			evidence.ExpectedModel = resolution.Entry.ID
			evidence.ExpectedEffort = resolution.Entry.Effort
			for _, a := range agents {
				if !a.identity.usable() || a.identity != d.identity {
					continue
				}
				for _, observedModel := range a.models {
					state := judgeDeclarationModel(evidence, declarationObservation{identity: a.identity, model: observedModel, linkedWorker: true}, routes)
					if state == DeclarationModelMismatch {
						evidence.ModelStatus = state
						evidence.ObservedModel = observedModel
						evidence.Verdict = VerdictDeclarationObservedMismatch
						break
					}
					if state == DeclarationModelConfirmed {
						evidence.ModelStatus = state
						evidence.ObservedModel = observedModel
					}
				}
				if evidence.ModelStatus == DeclarationModelMismatch {
					break
				}
			}
		}
		evidence = judgeDeclarationEvidence(evidence, effortAuthorized(l, t.id, evidence.ExpectedEffort, evidence.DeclaredEffort))
		out = append(out, evidence)
	}
	return out
}

func formatEvidenceIdentity(identity evidenceIdentity) string {
	if !identity.usable() {
		return "-"
	}
	return "source:" + identity.source + " session:" + identity.session + " dispatch:" + identity.dispatch
}

func aggregateDeclarationEvents(events []DeclarationEvidence) (Verdict, string, bool) {
	if len(events) == 0 {
		return "", "", false
	}
	verdict := VerdictDeclarationUnconfirmable
	for _, event := range events {
		if severity[event.Verdict] > severity[verdict] {
			verdict = event.Verdict
		}
	}
	return verdict, "host declaration evidence: " + string(verdict), true
}

func combineDeclarationVerdict(legacy, declaration Verdict) Verdict {
	if severity[declaration] >= severity[legacy] {
		return declaration
	}
	return legacy
}

func legacyTokensForDeclarationEvents(tokens []evidenceToken, events []DeclarationEvidence) []evidenceToken {
	if len(events) == 0 {
		return tokens
	}
	identities := make(map[evidenceIdentity]bool, len(events))
	for _, event := range events {
		identities[event.Identity] = true
	}
	filtered := make([]evidenceToken, 0, len(tokens))
	for _, token := range tokens {
		if token.identity.usable() && identities[token.identity] {
			continue
		}
		filtered = append(filtered, token)
	}
	return filtered
}

func summarizeDeclarationEvents(events []DeclarationEvidence) (expected, declared, status, observed string) {
	values := func(selectValue func(DeclarationEvidence) string) string {
		out := make([]string, 0, len(events))
		for _, event := range events {
			out = append(out, selectValue(event))
		}
		return strings.Join(out, ",")
	}
	return values(func(event DeclarationEvidence) string { return event.ExpectedEffort }),
		values(func(event DeclarationEvidence) string { return event.DeclaredEffort }),
		values(func(event DeclarationEvidence) string { return event.DeclarationStatus }),
		values(func(event DeclarationEvidence) string { return event.ObservedEffort })
}

// tokenValues extracts the raw model strings from a slice of evidence
// tokens, for the report's harness-agnostic TicketRow.Actuals display.
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
	// ClaudeTranscriptsDirs is the default-discovery union. When non-empty it
	// supersedes ClaudeTranscriptsDir; the singular field remains the explicit
	// --transcripts and backwards-compatible public Run seam.
	ClaudeTranscriptsDirs []string
	CodexSessionsDir      string // codex sessions dir (readCodexSessions); empty opts out of codex discovery entirely (I041)
	// Since scopes the transcript set to sessions active at/after a cutoff
	// (D28, ticket I047): an operator escape hatch, never an automatic
	// build-start anchor (rejected at grill — see parseSince). Accepts
	// RFC3339 or a bare YYYY-MM-DD date; empty means no filter. Applied to
	// both harnesses by comparing each transcript session's mtime against the
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
	return runWithHostPath(opts, "", nil)
}

func runWithHostPath(opts Options, hostPath string, lookup func(string) (string, error)) (Report, error) {
	hostConfig, hostConfigured, err := loadHostConfig(hostPath, lookup)
	if err != nil {
		return Report{}, err
	}
	repoDir := opts.RepoDir
	transcriptsDirs := opts.ClaudeTranscriptsDirs
	discloseTranscriptDirs := len(transcriptsDirs) > 0
	if !discloseTranscriptDirs {
		transcriptsDirs = []string{opts.ClaudeTranscriptsDir}
	}
	transcriptsDirs = dedupSorted(transcriptsDirs)
	if len(transcriptsDirs) == 0 {
		transcriptsDirs = []string{""} // preserve the legacy missing-dir warning
	}
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
	sourceHarness := transcriptHarness(transcriptsDirs[0])
	// mappings is keyed by harness so judgeToken can resolve each token
	// within its own harness's table (I040). Every shipped harness resolves
	// unconditionally so an observed model id can select its unique table;
	// transcript source remains the tiebreaker for ambiguous or unknown ids.
	mappings := map[string]map[string]resolvedTier{}
	for _, harness := range model.Harnesses() {
		mapping, err := resolveHarnessTiers(repoDir, harness)
		if err != nil {
			return Report{}, err
		}
		mappings[harness] = mapping
	}
	ledger := readLedger(filepath.Join(repoDir, ".superpowers", "sdd", "progress.md"))
	rep.Warnings = append(rep.Warnings, ledger.warnings...)
	var dispatches []dispatch
	var agents []subagent
	ticketTokens := make([]string, len(tickets))
	for i, ticket := range tickets {
		ticketTokens[i] = ticket.id
	}
	sessionMatched := false
	for _, transcriptsDir := range transcriptsDirs {
		if discloseTranscriptDirs {
			rep.Warnings = append(rep.Warnings, "scanning transcript dir: "+transcriptsDir)
		}
		moreDispatches, moreAgents, matched := readTranscripts(transcriptsDir, sourceHarness, since, opts.Session, ticketTokens, &rep.Warnings)
		dispatches = append(dispatches, moreDispatches...)
		agents = append(agents, moreAgents...)
		sessionMatched = sessionMatched || matched
	}
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
		codexDispatches, codexAgents, codexNM, codexSessionMatched := readCodexSessions(opts.CodexSessionsDir, repoDir, since, opts.Session, ticketTokens, &rep.Warnings)
		dispatches = append(dispatches, codexDispatches...)
		agents = append(agents, codexAgents...)
		codexNearMisses = codexNM
		sessionMatched = sessionMatched || codexSessionMatched
	}
	for i := range dispatches {
		dispatches[i].observedHarness = deriveHarness(dispatches[i].model, dispatches[i].source, mappings)
	}
	// M3 (I047 review): a non-empty --session that matched nothing anywhere
	// (claude or codex) is silently misleading otherwise — an operator
	// typo'd id produces an unexplained all-no-transcript audit, and codex
	// root ids are invisible in filenames (rollout-<ts>-<uuid>.jsonl) so
	// there's no other way to notice. Fires once, combined across both
	// harnesses, not per-reader.
	if opts.Session != "" && !sessionMatched {
		rep.Warnings = append(rep.Warnings, fmt.Sprintf("--session %q matched no sessions", opts.Session))
	}

	evidence := map[string][]evidenceToken{}    // ticket id -> harness-tagged model tokens
	declarations := map[string][]dispatch{}     // ticket id -> raw controller declarations, one per retry
	briefSources := map[string][]string{}       // ticket id -> resolved recorded brief paths (I101 D35)
	claimed := map[int]bool{}                   // dispatch index -> matched a ticket
	linked := map[string]bool{}                 // coarse-linkage disclosure only (I044)
	linkedClaude := map[evidenceIdentity]bool{} // complete Claude dispatch identity -> a subagent transcript carries models
	linkedCodex := map[string]bool{}            // Codex root-thread linkage is source-local (D20)
	for _, a := range agents {
		if a.toolUseID != "" && len(a.models) > 0 {
			linked[a.toolUseID] = true
		}
		if a.source == "claude" && a.identity.usable() && len(a.models) > 0 {
			linkedClaude[a.identity] = true
		}
		if a.source == "codex" && a.toolUseID != "" && len(a.models) > 0 {
			linkedCodex[a.toolUseID] = true
		}
	}
	// codex ticket-token matching is case-insensitive (D20's "Harness
	// threading" closing paragraph); the claude reader's matching is
	// untouched. codex dispatch descriptions/prompts are folded to upper
	// case here rather than at parse time, so Unmatched's display text
	// keeps its natural case.
	matches := func(d dispatch, id string) bool {
		desc, prompt := d.description, firstLine(d.prompt)
		qualifyingText := d.description + "\n" + d.prompt
		if d.briefText != "" && !namesATicket(d.description) {
			desc, prompt = firstLine(d.briefText), ""
			qualifyingText = d.briefText
		}
		if d.source == "codex" {
			desc, prompt = strings.ToUpper(desc), strings.ToUpper(prompt)
		}
		claimsTicket := ticketref.Contains(desc, id) || ticketref.Contains(prompt, id)
		if d.source == "codex" {
			// Team-spawn commands conventionally carry the assignment through
			// a dispatch-task-I###.md file reference. This is a structured
			// artifact name, not a hyphen-delimited ticket token, so retain it
			// as the one Codex-specific compatibility path while ordinary text
			// uses ticketref's strict standalone grammar.
			claimsTicket = claimsTicket || containsCodexDispatchTaskReference(desc, id) || containsCodexDispatchTaskReference(prompt, id)
		}
		if !claimsTicket {
			return false
		}
		// D28 (ticket I047): a claude dispatch claims the ticket only if it
		// ALSO references the audited repo or its own session shows cwd
		// evidence inside it — see repoQualifies. Codex evidence is already
		// hard-scoped to the repo before it reaches Run (D22,
		// readCodexSessions' cwdInsideRepo/gitCommitProber gate), so gating
		// it again here would be redundant, not stricter.
		if d.source == "claude" && !repoQualifies(qualifyingText, d.cwd, absRepoDir, repoBase) {
			return false
		}
		return true
	}
	// rootTickets tracks, per dispatch root (toolUseID), every distinct
	// ticket claimed under it — the coarse-linkage disclosure input (I041
	// review referred-Q3, ticket I044): thread_spawn actuals link by ROOT
	// session id only, so two dispatches for two different tickets sharing
	// one root also share whatever actual evidence that root's subagent(s)
	// contribute. Populated for every harness: a claude toolUseID IS the
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
			declarations[t.id] = append(declarations[t.id], d)
			if d.briefPath != "" {
				briefSources[t.id] = append(briefSources[t.id], d.briefPath)
			}
			if d.toolUseID != "" {
				if rootTickets[d.toolUseID] == nil {
					rootTickets[d.toolUseID] = map[string]bool{}
				}
				rootTickets[d.toolUseID][t.id] = true
			}
			if (d.source == "claude" && linkedClaude[d.identity]) || (d.source == "codex" && linkedCodex[d.toolUseID]) {
				continue // the subagent transcript below is the actual
			}
			if d.model != "" {
				evidence[t.id] = append(evidence[t.id], evidenceToken{value: d.model, harness: d.observedHarness, source: d.source, sourceFile: d.sourceFile, identity: d.identity})
			}
		}
		for _, a := range agents {
			desc := a.description
			if a.source == "codex" {
				desc = strings.ToUpper(desc)
			}
			use := ticketref.Contains(desc, t.id)
			// D28: the same repo qualification a dispatch needs applies to a
			// subagent's own description carrying the ticket token directly
			// (meta.json's description typically mirrors its parent
			// dispatch's) — otherwise this path would readmit exactly the
			// cross-repo collision the dispatch-side gate above closes, on a
			// shared transcript dir where both repos' subagent files carry
			// the same ticket id in their descriptions.
			if use && a.source == "claude" && !repoQualifies(a.description, a.cwd, absRepoDir, repoBase) {
				use = false
			}
			for _, d := range dispatches {
				if use {
					break
				}
				if d.source == "claude" {
					use = a.source == "claude" && d.identity.usable() && d.identity == a.identity && matches(d, t.id)
				} else if d.source == "codex" {
					use = a.source == "codex" && d.toolUseID != "" && d.toolUseID == a.toolUseID && matches(d, t.id)
				} else {
					use = d.toolUseID != "" && d.toolUseID == a.toolUseID && matches(d, t.id)
				}
			}
			if use {
				for _, m := range a.models {
					evidence[t.id] = append(evidence[t.id], evidenceToken{
						value: m, harness: deriveHarness(m, a.source, mappings), source: a.source, sourceFile: a.sourceFile, identity: a.identity,
					})
				}
			}
		}
	}
	rep.Warnings = append(rep.Warnings, validateDiscarded(ledger, tickets, evidence, mappings)...)
	for i, d := range dispatches {
		if !claimed[i] {
			rep.Unmatched = appendUnmatched(rep.Unmatched, DispatchInfo{
				Description: d.description, Harness: d.harness, Model: d.model, Effort: d.effort, EffortSource: d.effortSource, TeamSpawn: d.teamSpawn})
		}
	}

	coarseNotes := coarseLinkageNotes(rootTickets, dispatches, linked)
	var observedRoutes observedRouteIndex
	if hostConfigured {
		observedRoutes = newObservedRouteIndex(hostConfig)
	}
	for _, t := range tickets {
		tokens := evidence[t.id]
		row := TicketRow{ID: t.id, Tier: t.tier, Actuals: dedupSorted(tokenValues(tokens))}
		row.ExpectedEffort, row.DeclaredEffort, row.DeclarationStatus, row.ObservedEffort = summarizeEffortDeclarations(repoDir, t, declarations[t.id], ledger)
		if hostConfigured {
			row.DeclarationEvents = judgeHostDeclarations(repoDir, hostPath, t, declarations[t.id], agents, observedRoutes, ledger)
		}
		row.Verdict, row.Detail = judge(t, legacyTokensForDeclarationEvents(tokens, row.DeclarationEvents), mappings, ledger)
		if verdict, detail, ok := aggregateDeclarationEvents(row.DeclarationEvents); ok {
			if combineDeclarationVerdict(row.Verdict, verdict) == verdict {
				row.Verdict, row.Detail = verdict, detail
			}
			row.ExpectedEffort, row.DeclaredEffort, row.DeclarationStatus, row.ObservedEffort = summarizeDeclarationEvents(row.DeclarationEvents)
		}
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
		if sources := dedupSorted(briefSources[t.id]); len(sources) > 0 {
			note := "source: " + strings.Join(sources, ", ")
			if row.Detail == "" {
				row.Detail = note
			} else {
				row.Detail += "; " + note
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

// preflightHostConfig validates only local configuration structure, declared
// executable availability, and exact pins before any transcript work. It
// intentionally does not infer reachability for unpinned preferences or
// alter any preference-only audit mapping; those are I074 concerns.
func preflightHostConfig(path string, lookup func(string) (string, error)) error {
	_, _, err := loadHostConfig(path, lookup)
	return err
}

func loadHostConfig(path string, lookup func(string) (string, error)) (hostconfig.Config, bool, error) {
	if path == "" {
		var err error
		path, err = hostconfig.DefaultPath()
		if err != nil {
			return hostconfig.Config{}, false, err
		}
	}
	if lookup == nil {
		lookup = exec.LookPath
	}
	config, err := hostconfig.Load(path, model.Harnesses(), lookup)
	if errors.Is(err, hostconfig.ErrNotConfigured) {
		return hostconfig.Config{}, false, nil
	}
	if err != nil {
		return hostconfig.Config{}, false, err
	}
	return config, true, nil
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
		if !ticketref.Contains(strings.ToUpper(nm.text), id) {
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
// harness, not just codex, see its doc — would fire this codex-worded note on
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
// own harness's table (I040): judge itself never picks a harness.
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
	var discardedDetails []string
	for _, tok := range tokens {
		v, d := judgeToken(tok, t, mappings, l)
		worse(v, d)
		if v == VerdictDiscardedWithReason && d != "" {
			discardedDetails = append(discardedDetails, d)
		}
		if v == VerdictMatch && tok.source == "codex" && tok.sourceFile != "" {
			matchSources = append(matchSources, tok.sourceFile)
		}
	}
	if verdict == VerdictMatch && len(matchSources) > 0 {
		detail = "source: " + strings.Join(dedupSorted(matchSources), ", ")
	}
	if verdict != VerdictDiscardedWithReason && len(discardedDetails) > 0 {
		note := "discarded event: " + strings.Join(dedupSorted(discardedDetails), "; ")
		if detail == "" {
			detail = note
		} else {
			detail += "; " + note
		}
	}
	return verdict, detail
}

// withSource appends a codex-source evidence token's transcript file to a
// judged detail line (D24: every judged codex verdict names its source, the
// I008 silent-descent requirement satisfied as a special case). Claude-source
// tokens never carry a sourceFile (readTranscripts/scanJSONL/parseLine never
// set one), so this is a no-op for every claude-layout call — the guarantee
// that claude verdict details stay byte-identical.
func withSource(detail string, tok evidenceToken) string {
	if tok.source != "codex" || tok.sourceFile == "" {
		return detail
	}
	return detail + " (source: " + tok.sourceFile + ")"
}

// judgeToken classifies a single observed evidence token against the
// ticket's declared tier, resolving the token within its own harness's table
// — the per-token seam design D15 names and I040 makes real. A token whose
// harness has no resolved table (unreachable while only the claude reader is
// wired) is treated the same as one that maps to no entry: unmapped, never a
// crash or a silent match.
func judgeToken(tok evidenceToken, t ticket, mappings map[string]map[string]resolvedTier, l ledger) (Verdict, string) {
	mapping := mappings[tok.harness]
	tiers := tiersOf(tok.value, mapping)
	if len(tiers) == 0 {
		return VerdictUnmappedDispatch, withSource(fmt.Sprintf("%s maps to no %s entry in the model table", tok.value, tok.harness), tok)
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
		key := discardedKey{ticket: t.id, identity: tok.identity, tier: actual}
		if rec, ok := l.discarded[key]; ok && tok.identity.usable() {
			return VerdictDiscardedWithReason, withSource(fmt.Sprintf("%s (%s) vs declared %s — DISCARDED reason: %s", tok.value, actual, t.tier, rec.reason), tok)
		}
		return VerdictSilentDescent, withSource(fmt.Sprintf("%s (%s) below declared %s with no ESCALATION record", tok.value, actual, t.tier), tok)
	}
	return VerdictEscalatedNoReason, withSource(fmt.Sprintf("%s (%s) above declared %s with no ESCALATION record", tok.value, actual, t.tier), tok)
}

// tiersOf resolves a model token to every tier it could mean within one
// harness's resolved table: exact match on the resolved id, on an explicitly
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
// design D15's harness scoping decides between harnesses, this rule decides
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
	id     string
	tier   string
	effort string
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
		tickets = append(tickets, ticket{id: fm["id"], tier: fm["tier"], effort: fm["effort"]})
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

// transcriptHarness names the harness associated with one transcript source.
// I111 narrows its authority: deriveHarness uses it only when an observed id
// is ambiguous across resolved harnesses or unknown to all of them. It covers
// the claude harness's ~/.claude/projects layout; readCodexSessions tags the
// codex source directly.
func transcriptHarness(transcriptsDir string) string {
	return "claude"
}

// deriveHarness uses the observed model id when it identifies exactly one
// resolved harness. The transcript source retains D15's authority for an id
// shared across harnesses and preserves the existing behavior for unknown ids.
func deriveHarness(token, sourceHarness string, mappings map[string]map[string]resolvedTier) string {
	match := ""
	for harness, mapping := range mappings {
		if len(tiersOf(token, mapping)) == 0 {
			continue
		}
		if match != "" {
			return sourceHarness
		}
		match = harness
	}
	if match == "" {
		return sourceHarness
	}
	return match
}

// resolvedTier is the audit's view of one (harness, tier) row, obtained
// through the strict launch validator so the audit's active leg judges exactly
// what controlled dispatch-time validation returns. The audit owns no
// WORKFLOW.md routing parser of its own.
type resolvedTier struct {
	id      string   // resolved model id: the repo's mirror value if present, else the embedded default
	aliases []string // the table entry's explicitly declared aliases
	history []string // ids this entry shipped as defaults before the current one (exact-match only)
}

func (rt resolvedTier) matches(token string) bool {
	if model.ActiveIDMatches(rt.id, token) {
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

// resolveHarnessTiers builds one harness's tier -> resolvedTier table for
// repoDir via model.ResolveStrictActive. Strict repository configuration errors
// refuse the audit rather than letting compatibility resolution select a
// different active ID. Audit-only aliases and history are layered below after
// the shared active result.
func resolveHarnessTiers(repoDir, harness string) (map[string]resolvedTier, error) {
	mapping := map[string]resolvedTier{}
	for _, tier := range model.Tiers {
		e, err := model.ResolveStrictActive(repoDir, harness, tier)
		if err != nil {
			return nil, fmt.Errorf("model launch policy resolution failed for %s.%s: %w", harness, tier, err)
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
			rt.history = model.HistoricalIDs(harness, tier)
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

// effortEscalation is a raw effort-deviation authorization. Its tokens carry
// no cross-harness ordering; it authorizes only its exact ticket/from/to
// tuple and never participates in model-tier judgement.
type effortEscalation struct {
	ticket string
	from   string
	to     string
	reason string
	line   int
}

type ledger struct {
	escalation        map[string][]escRecord // ticket id -> model-tier escalation records
	effortEscalations map[string][]effortEscalation
	fallback          map[string]string // ticket id -> fallback reason
	discarded         map[discardedKey]discardedRecord
	pending           []discardedRecord
	warnings          []string
}

type discardedKey struct {
	ticket   string
	identity evidenceIdentity
	tier     string
}

type discardedRecord struct {
	ticket   string
	identity evidenceIdentity
	tier     string
	reason   string
	line     int
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
	l := ledger{escalation: map[string][]escRecord{}, effortEscalations: map[string][]effortEscalation{}, fallback: map[string]string{}, discarded: map[discardedKey]discardedRecord{}}
	raw, err := os.ReadFile(path)
	if err != nil {
		return l
	}
	var effortFence markdownFence
	for lineNo, line := range strings.Split(string(raw), "\n") {
		effortFence.advance(line)
		if !effortFence.active() {
			if rec, ok := parseEffortEscalation(line, lineNo+1); ok {
				l.effortEscalations[rec.ticket] = append(l.effortEscalations[rec.ticket], rec)
				continue // effort records never become model evidence
			}
		}
		if strings.HasPrefix(strings.TrimSpace(line), "DISCARDED") {
			rec, ok := parseDiscarded(strings.TrimSpace(line), lineNo+1)
			if !ok {
				l.warnings = append(l.warnings, fmt.Sprintf("DISCARDED line %d malformed — ignored", lineNo+1))
				continue
			}
			l.pending = append(l.pending, rec)
			continue
		}
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
				continue // effort records never become model evidence
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
	counts := map[discardedKey]int{}
	for _, rec := range l.pending {
		counts[discardedKey{ticket: rec.ticket, identity: rec.identity, tier: rec.tier}]++
	}
	duplicates := map[discardedKey]bool{}
	for _, rec := range l.pending {
		key := discardedKey{ticket: rec.ticket, identity: rec.identity, tier: rec.tier}
		if counts[key] > 1 {
			if !duplicates[key] {
				l.warnings = append(l.warnings, fmt.Sprintf("DISCARDED line %d duplicate identity — ignored", rec.line))
				duplicates[key] = true
			}
			continue
		}
		l.discarded[key] = rec
	}
	return l
}

// markdownFence tracks Markdown code fences only for the I075 effort-record
// boundary. Other ledger grammars intentionally retain their existing reader.
type markdownFence struct {
	marker byte
	width  int
}

func (f *markdownFence) active() bool {
	return f.marker != 0
}

func (f *markdownFence) advance(line string) {
	trimmed := strings.TrimLeft(line, " \t")
	if len(trimmed) < 3 || (trimmed[0] != '`' && trimmed[0] != '~') {
		return
	}
	width := 0
	for width < len(trimmed) && trimmed[width] == trimmed[0] {
		width++
	}
	if width < 3 {
		return
	}
	if !f.active() {
		f.marker, f.width = trimmed[0], width
		return
	}
	if trimmed[0] == f.marker && width >= f.width && strings.TrimSpace(trimmed[width:]) == "" {
		f.marker, f.width = 0, 0
	}
}

// parseEffortEscalation accepts exactly the ordered I075 authorization
// grammar. Raw endpoints are retained byte-for-byte; malformed records are
// ignored and authorize nothing.
func parseEffortEscalation(line string, lineNo int) (effortEscalation, bool) {
	if line == "" || strings.TrimSpace(line) != line {
		return effortEscalation{}, false
	}
	if strings.Count(line, "reason:") != 1 {
		return effortEscalation{}, false
	}
	fields := strings.SplitN(line, " reason: ", 2)
	if len(fields) != 2 || fields[1] == "" || strings.TrimSpace(fields[1]) == "" {
		return effortEscalation{}, false
	}
	parts := strings.Split(fields[0], " ")
	if len(parts) != 4 || parts[0] != "ESCALATION" || parts[2] != "effort" {
		return effortEscalation{}, false
	}
	if strings.Count(parts[3], "->") != 1 {
		return effortEscalation{}, false
	}
	from, to, _ := strings.Cut(parts[3], "->")
	if from == "" || to == "" || strings.ContainsAny(from, " \t") || strings.ContainsAny(to, " \t") {
		return effortEscalation{}, false
	}
	return effortEscalation{ticket: parts[1], from: from, to: to, reason: fields[1], line: lineNo}, true
}

func effortAuthorized(l ledger, ticketID, expected, declared string) bool {
	for _, rec := range l.effortEscalations[ticketID] {
		if rec.from == expected && rec.to == declared {
			return true
		}
	}
	return false
}

// parseDiscarded accepts only the published, ordered one-line grammar. The
// literal spacing is intentional: reordering or adding a field must not turn
// a broad operator typo into a routing exception.
func parseDiscarded(line string, lineNo int) (discardedRecord, bool) {
	const prefix = "DISCARDED "
	if !strings.HasPrefix(line, prefix) {
		return discardedRecord{}, false
	}
	head, reason, ok := strings.Cut(strings.TrimPrefix(line, prefix), " reason: ")
	if !ok || strings.TrimSpace(reason) == "" {
		return discardedRecord{}, false
	}
	parts := strings.Split(head, " ")
	if len(parts) != 5 || parts[0] == "" {
		return discardedRecord{}, false
	}
	get := func(part, name string) (string, bool) {
		value, ok := strings.CutPrefix(part, name)
		return value, ok && value != "" && !strings.ContainsAny(value, " \t\"'")
	}
	source, ok := get(parts[1], "source:")
	if !ok || (source != "claude" && source != "codex") {
		return discardedRecord{}, false
	}
	session, ok := get(parts[2], "session:")
	if !ok {
		return discardedRecord{}, false
	}
	dispatch, ok := get(parts[3], "dispatch:")
	if !ok {
		return discardedRecord{}, false
	}
	tier, ok := get(parts[4], "tier:")
	if !ok {
		return discardedRecord{}, false
	}
	if _, ok := tierRank[tier]; !ok {
		return discardedRecord{}, false
	}
	return discardedRecord{ticket: parts[0], identity: evidenceIdentity{source: source, session: session, dispatch: dispatch}, tier: tier, reason: strings.TrimSpace(reason), line: lineNo}, true
}

// validateDiscarded restricts each parsed record to exactly one otherwise
// lower-tier token. Records that match zero or several candidates remain
// visible as warnings and never enter the judge lookup.
func validateDiscarded(l ledger, tickets []ticket, evidence map[string][]evidenceToken, mappings map[string]map[string]resolvedTier) []string {
	byID := map[string]ticket{}
	for _, t := range tickets {
		byID[t.id] = t
	}
	var warnings []string
	for key, rec := range l.discarded {
		t, ok := byID[rec.ticket]
		matches := 0
		if ok {
			for _, tok := range evidence[rec.ticket] {
				if tok.identity != rec.identity || !discardEligible(tok, t, mappings, l) {
					continue
				}
				actual := pickTier(tiersOf(tok.value, mappings[tok.harness]), t.tier, l.fallback[t.id] != "")
				if actual == rec.tier {
					matches++
				}
			}
		}
		if matches != 1 {
			delete(l.discarded, key)
			warnings = append(warnings, fmt.Sprintf("DISCARDED line %d matches %d eligible evidence token(s) — ignored", rec.line, matches))
		}
	}
	return warnings
}

func discardEligible(tok evidenceToken, t ticket, mappings map[string]map[string]resolvedTier, l ledger) bool {
	if !tok.identity.usable() || t.tier == "" || t.tier == "n/a" {
		return false
	}
	tiers := tiersOf(tok.value, mappings[tok.harness])
	if len(tiers) == 0 {
		return false
	}
	actual := pickTier(tiers, t.tier, l.fallback[t.id] != "")
	if actual == "fallback" || actual == t.tier || t.tier == "fallback" || tierRank[actual] >= tierRank[t.tier] {
		return false
	}
	for _, rec := range l.escalation[t.id] {
		if rec.to == actual {
			return false
		}
	}
	return true
}

// --- transcript inputs (undocumented harness format; degrade, never fail) ---

type dispatch struct {
	toolUseID       string
	description     string
	prompt          string
	briefText       string // I101: body recorded in the lead transcript, never read from disk
	briefPath       string // normalized transcript path; disclosure is added in Task 3
	briefCutoff     int    // I101 D32: evidence available when this spawn occurred
	model           string
	harness         string // raw controller declaration; never inferred from source
	effort          string // declared worker effort, claude-team spawns only (I090); reported, never judged — see DispatchInfo.Effort
	effortSource    string
	observedHarness string // observed-model harness, with source as D15 tiebreaker (I111)
	source          string // transcript layout/source; distinct from model-derived harness (I111)
	sourceFile      string // source transcript file (D24, codex only); "" for claude
	cwd             string // D28 (I047): the event line's own cwd, claude only; "" for codex (D22 scopes it separately)
	identity        evidenceIdentity

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
	source      string // transcript layout/source; each model derives its own harness (I111)
	sourceFile  string // source transcript file (D24, codex only); "" for claude
	cwd         string // D28 (I047): the subagent transcript's own session cwd, claude only
	identity    evidenceIdentity
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

// containsCodexDispatchTaskReference recognizes the historical Codex team
// dispatch artifact name, dispatch-task-I###.md, only as a full path
// component. It deliberately does not relax ticketref's standalone grammar
// for ordinary hyphenated text.
func containsCodexDispatchTaskReference(text, id string) bool {
	const prefix = "DISPATCH-TASK-"
	const suffix = ".MD"
	text = strings.ToUpper(text)
	name := prefix + strings.ToUpper(id) + suffix
	for start := 0; ; {
		i := strings.Index(text[start:], name)
		if i < 0 {
			return false
		}
		i += start
		before := i == 0 || text[i-1] == '/' || text[i-1] == '\\'
		afterIndex := i + len(name)
		after := afterIndex >= len(text) || !isPathWordChar(text[afterIndex])
		if before && after {
			return true
		}
		start = i + 1
	}
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
// by the sidecar meta.json. Every record it emits is tagged with its source;
// Run derives the model harness after all resolved mappings are available.
// since and sessionID implement D28's
// --since/--session filters (ticket I047), applied once per session id
// (I2 fix — see sessionFiles/sessionInScope): a zero since and empty
// sessionID (every pre-I047 caller) filter nothing, keeping every existing
// behavior byte-identical. matchedSession reports whether sessionID (when
// non-empty) equaled at least one discovered session id — independent of
// whether --since then excluded it — the diagnostic input for M3's
// "matched no sessions" warning; always true when sessionID is empty (no
// filter to fail to match). All trouble becomes warnings.
func readTranscripts(dir, source string, since time.Time, sessionID string, ticketTokens []string, warnings *[]string) ([]dispatch, []subagent, bool) {
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
				more[i].observedHarness = source
				more[i].source = source
				more[i].identity = evidenceIdentity{source: source, session: id, dispatch: more[i].toolUseID}
			}
			dispatches = append(dispatches, more...)
		}
		if sf.dirPath != "" {
			subDir := filepath.Join(sf.dirPath, "subagents")
			subs, _ := filepath.Glob(filepath.Join(subDir, "agent-*.jsonl"))
			sort.Strings(subs)
			for _, sub := range subs {
				a := subagent{source: source}
				if metaRaw, err := os.ReadFile(strings.TrimSuffix(sub, ".jsonl") + ".meta.json"); err == nil {
					var meta struct {
						ToolUseID   string `json:"toolUseId"`
						Description string `json:"description"`
					}
					if json.Unmarshal(metaRaw, &meta) == nil {
						a.toolUseID, a.description = meta.ToolUseID, meta.Description
						a.identity = evidenceIdentity{source: source, session: id, dispatch: meta.ToolUseID}
					}
				}
				more, models, cwd := scanJSONL(sub, warnings)
				for i := range more {
					more[i].observedHarness = source
					more[i].source = source
					more[i].identity = evidenceIdentity{source: source, session: id, dispatch: more[i].toolUseID}
				}
				a.models = models
				a.cwd = cwd
				dispatches = append(dispatches, more...)
				agents = append(agents, a)
			}

			workflowSubs, _ := filepath.Glob(filepath.Join(subDir, "workflows", "*", "agent-*.jsonl"))
			sort.Strings(workflowSubs)
			for _, sub := range workflowSubs {
				workflowID := filepath.Base(filepath.Dir(sub))
				agentID := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(sub), "agent-"), ".jsonl")
				meta, ok := loadWorkflowMeta(dir, id, workflowID, agentID, warnings)
				if !ok || meta.AgentType != "workflow-subagent" || meta.SpawnDepth != 1 {
					continue
				}
				file, err := openWorkflowFile(dir, []string{id, "subagents", "workflows", workflowID}, filepath.Base(sub), workflowTranscriptBeforeOpen)
				if err != nil {
					if readErr, ok := err.(*workflowMetadataReadError); ok && readErr.kind == workflowMetadataUnsafe {
						*warnings = append(*warnings, sub+": workflow transcript unsafe — transcript skipped")
					} else {
						*warnings = append(*warnings, sub+": workflow transcript unreadable — transcript skipped: "+err.Error())
					}
					continue
				}
				openingLine, more, models, cwd := scanWorkflowJSONL(sub, file, warnings)
				// A valid nested tool-use dispatch is independent evidence. Its
				// prompt/description and complete workflow-scoped identity decide
				// attribution, so retain it after admission and safe parsing even
				// when the parent opening cannot be attributed to one ticket.
				workflowSession := id + "/" + filepath.Base(filepath.Dir(sub)) + "/" + strings.TrimSuffix(filepath.Base(sub), ".jsonl")
				for i := range more {
					more[i].observedHarness = source
					more[i].source = source
					more[i].identity = evidenceIdentity{source: source, session: workflowSession, dispatch: more[i].toolUseID}
				}
				dispatches = append(dispatches, more...)
				referenceCount := ticketref.ReferenceCount(openingLine, ticketTokens)
				if referenceCount == 0 {
					continue
				}
				if referenceCount > 1 {
					*warnings = append(*warnings, sub+": workflow opening line names multiple tickets — skipped")
					continue
				}
				a := subagent{source: source, description: openingLine}
				if len(models) == 0 {
					if agentID == "" {
						*warnings = append(*warnings, sub+": workflow agent filename has no agent id — model fallback skipped")
					} else if model, found := workflowRunModel(
						dir, id, workflowID, agentID, warnings,
					); found {
						models = []string{model}
					}
				}
				a.models = models
				a.cwd = cwd
				agents = append(agents, a)
			}
		}
	}
	return dispatches, agents, matchedSession
}

type workflowMeta struct {
	AgentType  string `json:"agentType"`
	SpawnDepth int    `json:"spawnDepth"`
}

// Workflow run metadata is a separate, sibling artifact from the per-agent
// sidecar. The sidecar admits only the precise workflow-subagent/depth shape;
// when the transcript itself carries no model, workflowProgress is the only
// accepted fallback source. In particular, defaultModel and entries for any
// other worker must never become evidence for this worker.
const workflowMetadataMaxBytes = 1 << 20

// workflowMetadataBeforeOpen is a deterministic replacement-attack seam for
// tests. Production leaves it nil. Every selected file is checked both before
// and after the open, so an atomic replacement cannot substitute another
// session or workflow's metadata.
var workflowMetadataBeforeOpen func(string)

// workflowTranscriptBeforeOpen is the equivalent deterministic replacement
// seam for workflow JSONL tests. Production leaves it nil.
var workflowTranscriptBeforeOpen func(string)

type workflowMetadataReadKind uint8

const (
	workflowMetadataUnavailable workflowMetadataReadKind = iota
	workflowMetadataUnsafe
	workflowMetadataOversized
)

type workflowMetadataReadError struct {
	kind workflowMetadataReadKind
	err  error
}

func (e *workflowMetadataReadError) Error() string {
	if e.err == nil {
		return "workflow metadata read failed"
	}
	return e.err.Error()
}

// openWorkflowFile resolves each named component under transcriptDir with
// descriptor-relative opens. Lstat plus SameFile checks reject symlinks,
// non-regular targets, and path replacement between inspection and open.
func openWorkflowFile(transcriptDir string, components []string, name string, beforeOpen func(string)) (*os.File, error) {
	root, err := os.OpenRoot(transcriptDir)
	if err != nil {
		return nil, &workflowMetadataReadError{kind: workflowMetadataUnavailable, err: err}
	}
	defer func() { _ = root.Close() }()
	for _, component := range components {
		if !workflowMetadataComponent(component) {
			return nil, &workflowMetadataReadError{kind: workflowMetadataUnsafe}
		}
		info, err := root.Lstat(component)
		if err != nil {
			return nil, &workflowMetadataReadError{kind: workflowMetadataUnavailable, err: err}
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, &workflowMetadataReadError{kind: workflowMetadataUnsafe}
		}
		child, err := root.OpenRoot(component)
		if err != nil {
			return nil, &workflowMetadataReadError{kind: workflowMetadataUnsafe, err: err}
		}
		opened, statErr := child.Stat(".")
		current, currentErr := root.Lstat(component)
		if statErr != nil || currentErr != nil || current.Mode()&os.ModeSymlink != 0 || !opened.IsDir() || !os.SameFile(info, opened) || !os.SameFile(info, current) {
			_ = child.Close()
			return nil, &workflowMetadataReadError{kind: workflowMetadataUnsafe}
		}
		_ = root.Close()
		root = child
	}

	if !workflowMetadataComponent(name) {
		return nil, &workflowMetadataReadError{kind: workflowMetadataUnsafe}
	}
	info, err := root.Lstat(name)
	if err != nil {
		return nil, &workflowMetadataReadError{kind: workflowMetadataUnavailable, err: err}
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, &workflowMetadataReadError{kind: workflowMetadataUnsafe}
	}
	if beforeOpen != nil {
		beforeOpen(filepath.Join(transcriptDir, filepath.Join(append(append([]string(nil), components...), name)...)))
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, &workflowMetadataReadError{kind: workflowMetadataUnsafe, err: err}
	}
	opened, statErr := file.Stat()
	current, currentErr := root.Lstat(name)
	if statErr != nil || currentErr != nil || current.Mode()&os.ModeSymlink != 0 || !opened.Mode().IsRegular() || !os.SameFile(info, opened) || !os.SameFile(info, current) {
		_ = file.Close()
		return nil, &workflowMetadataReadError{kind: workflowMetadataUnsafe}
	}
	return file, nil
}

func readWorkflowMetadataFile(transcriptDir string, components []string, name string) ([]byte, error) {
	file, err := openWorkflowFile(transcriptDir, components, name, workflowMetadataBeforeOpen)
	if err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(io.LimitReader(file, workflowMetadataMaxBytes+1))
	if err != nil {
		_ = file.Close()
		return nil, &workflowMetadataReadError{kind: workflowMetadataUnavailable, err: err}
	}
	if err := file.Close(); err != nil {
		return nil, &workflowMetadataReadError{kind: workflowMetadataUnavailable, err: err}
	}
	if len(raw) > workflowMetadataMaxBytes {
		return nil, &workflowMetadataReadError{kind: workflowMetadataOversized}
	}
	return raw, nil
}

func workflowMetadataComponent(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, `/\\`) && filepath.Base(name) == name
}

func loadWorkflowMeta(transcriptDir, session, workflow, agentID string, warnings *[]string) (workflowMeta, bool) {
	path := filepath.Join(transcriptDir, session, "subagents", "workflows", workflow, "agent-"+agentID+".meta.json")
	raw, err := readWorkflowMetadataFile(transcriptDir, []string{session, "subagents", "workflows", workflow}, "agent-"+agentID+".meta.json")
	if err != nil {
		if readErr, ok := err.(*workflowMetadataReadError); ok {
			switch readErr.kind {
			case workflowMetadataUnsafe:
				*warnings = append(*warnings, path+": workflow metadata unsafe — transcript skipped")
			case workflowMetadataOversized:
				*warnings = append(*warnings, path+": workflow metadata exceeds 1048576 bytes — transcript skipped")
			default:
				*warnings = append(*warnings, path+": workflow metadata unreadable — transcript skipped: "+err.Error())
			}
		} else {
			*warnings = append(*warnings, path+": workflow metadata unreadable — transcript skipped: "+err.Error())
		}
		return workflowMeta{}, false
	}
	var meta workflowMeta
	duplicate, err := unmarshalWorkflowMetadata(raw, &meta)
	if duplicate {
		*warnings = append(*warnings, path+": ambiguous workflow metadata — transcript skipped")
		return workflowMeta{}, false
	}
	if err != nil {
		*warnings = append(*warnings, path+": malformed workflow metadata — transcript skipped")
		return workflowMeta{}, false
	}
	return meta, true
}

type workflowRunProgress struct {
	AgentID string `json:"agentId"`
	Model   string `json:"model"`
	Type    string `json:"type"`
}

func workflowRunModel(transcriptDir, session, workflow, agentID string, warnings *[]string) (string, bool) {
	path := filepath.Join(transcriptDir, session, "workflows", workflow+".json")
	data, err := readWorkflowMetadataFile(transcriptDir, []string{session, "workflows"}, workflow+".json")
	if err != nil {
		if readErr, ok := err.(*workflowMetadataReadError); ok {
			switch readErr.kind {
			case workflowMetadataUnsafe:
				*warnings = append(*warnings, path+": workflow run metadata unsafe — model fallback skipped")
			case workflowMetadataOversized:
				*warnings = append(*warnings, path+": workflow run metadata exceeds 1048576 bytes — model fallback skipped")
			default:
				*warnings = append(*warnings, fmt.Sprintf("%s: workflow run metadata unavailable for session %q workflow %q — model fallback skipped", path, session, workflow))
			}
		} else {
			*warnings = append(*warnings, fmt.Sprintf("%s: workflow run metadata unavailable for session %q workflow %q — model fallback skipped", path, session, workflow))
		}
		return "", false
	}

	var envelope struct {
		WorkflowProgress json.RawMessage `json:"workflowProgress"`
	}
	duplicate, err := unmarshalWorkflowRunMetadata(data, &envelope)
	if duplicate {
		*warnings = append(*warnings, path+": ambiguous workflow run metadata — model fallback skipped")
		return "", false
	}
	if err != nil {
		*warnings = append(*warnings, path+": malformed workflow run metadata — model fallback skipped")
		return "", false
	}
	if len(envelope.WorkflowProgress) == 0 || string(envelope.WorkflowProgress) == "null" {
		*warnings = append(*warnings, path+": workflow run metadata has no workflowProgress — model fallback skipped")
		return "", false
	}

	var progress []json.RawMessage
	duplicate, err = unmarshalWorkflowRunMetadata(envelope.WorkflowProgress, &progress)
	if duplicate {
		*warnings = append(*warnings, path+": ambiguous workflow run metadata — model fallback skipped")
		return "", false
	}
	if err != nil {
		*warnings = append(*warnings, path+": malformed workflow run metadata — model fallback skipped")
		return "", false
	}
	var matches []workflowRunProgress
	for _, raw := range progress {
		entry, phase, valid := parseWorkflowRunProgress(raw)
		if !valid {
			*warnings = append(*warnings, path+": malformed workflow run metadata — model fallback skipped")
			return "", false
		}
		if phase {
			continue
		}
		if entry.AgentID == agentID {
			matches = append(matches, entry)
		}
	}
	switch len(matches) {
	case 0:
		*warnings = append(*warnings, fmt.Sprintf("%s: workflow run metadata has no exact entry for agent %q — model fallback skipped", path, agentID))
		return "", false
	case 1:
		if matches[0].Model == "" {
			*warnings = append(*warnings, fmt.Sprintf("%s: workflow run metadata entry for agent %q has no model — model fallback skipped", path, agentID))
			return "", false
		}
		return matches[0].Model, true
	default:
		*warnings = append(*warnings, fmt.Sprintf("%s: workflow run metadata has multiple entries for agent %q — model fallback skipped", path, agentID))
		return "", false
	}
}

// parseWorkflowRunProgress admits exactly two workflowProgress shapes:
// workflow_phase entries without routing evidence, and agent entries with an
// exact identity/model pair. Historical synthetic metadata omitted type, so
// that canonical agent shape remains compatible. Everything agent-like but
// incomplete is malformed rather than an excuse to use defaultModel.
func parseWorkflowRunProgress(raw json.RawMessage) (workflowRunProgress, bool, bool) {
	var fields map[string]json.RawMessage
	duplicate, err := unmarshalWorkflowRunMetadata(raw, &fields)
	if duplicate || err != nil || fields == nil {
		return workflowRunProgress{}, false, false
	}
	var entry workflowRunProgress
	if err := json.Unmarshal(raw, &entry); err != nil {
		return workflowRunProgress{}, false, false
	}
	_, hasAgentID := fields["agentId"]
	_, hasModel := fields["model"]
	_, hasType := fields["type"]
	switch entry.Type {
	case "workflow_phase":
		return workflowRunProgress{}, true, !hasAgentID && !hasModel
	case "workflow_agent":
		return entry, false, hasAgentID && hasModel && entry.AgentID != "" && entry.Model != ""
	case "":
		return entry, false, !hasType && hasAgentID && hasModel && entry.AgentID != "" && entry.Model != ""
	default:
		return workflowRunProgress{}, false, false
	}
}

// unmarshalUniqueJSON rejects duplicate member names in every object before
// decoding, so untrusted metadata never inherits encoding/json's last-value-
// wins behavior.
func unmarshalUniqueJSON(data []byte, dst any) (bool, error) {
	return unmarshalUniqueJSONWithMemberValidator(data, dst, nil)
}

// unmarshalWorkflowMetadata additionally rejects case-variant spellings of
// the sidecar fields that admit a workflow transcript. encoding/json matches
// struct fields case-insensitively, so accepting aliases here would reintroduce
// last-value-wins admission despite the duplicate-member guard.
func unmarshalWorkflowMetadata(data []byte, dst any) (bool, error) {
	return unmarshalUniqueJSONWithMemberValidator(data, dst, func(path []string, name string) bool {
		return len(path) == 0 && caseVariantJSONMember(name, "agentType", "spawnDepth")
	})
}

// unmarshalWorkflowRunMetadata preserves unrelated run metadata while
// requiring exact spellings for the fields used as routing evidence: the
// top-level workflowProgress member and agentId/model in each of its entries.
func unmarshalWorkflowRunMetadata(data []byte, dst any) (bool, error) {
	return unmarshalUniqueJSONWithMemberValidator(data, dst, func(path []string, name string) bool {
		if len(path) == 0 {
			return caseVariantJSONMember(name, "workflowProgress")
		}
		return ((len(path) == 2 && path[0] == "workflowProgress" && strings.HasPrefix(path[1], "[")) || len(path) == 0) &&
			caseVariantJSONMember(name, "agentId", "model", "type")
	})
}

func caseVariantJSONMember(name string, admitted ...string) bool {
	for _, exact := range admitted {
		if name != exact && strings.EqualFold(name, exact) {
			return true
		}
	}
	return false
}

func unmarshalUniqueJSONWithMemberValidator(data []byte, dst any, invalidMember func([]string, string) bool) (bool, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	duplicate, err := jsonHasDuplicateMembers(decoder, nil, invalidMember)
	if err != nil || duplicate {
		return duplicate, err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return false, errors.New("multiple JSON values")
		}
		return false, err
	}
	return false, json.Unmarshal(data, dst)
}

func jsonHasDuplicateMembers(decoder *json.Decoder, path []string, invalidMember func([]string, string) bool) (bool, error) {
	token, err := decoder.Token()
	if err != nil {
		return false, err
	}
	switch token := token.(type) {
	case json.Delim:
		switch token {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				key, err := decoder.Token()
				if err != nil {
					return false, err
				}
				name, ok := key.(string)
				if !ok {
					return false, errors.New("object member is not a string")
				}
				if seen[name] {
					return true, nil
				}
				if invalidMember != nil && invalidMember(path, name) {
					return true, nil
				}
				seen[name] = true
				duplicate, err := jsonHasDuplicateMembers(decoder, append(path, name), invalidMember)
				if duplicate || err != nil {
					return duplicate, err
				}
			}
			_, err := decoder.Token()
			return false, err
		case '[':
			for index := 0; decoder.More(); index++ {
				duplicate, err := jsonHasDuplicateMembers(decoder, append(path, fmt.Sprintf("[%d]", index)), invalidMember)
				if duplicate || err != nil {
					return duplicate, err
				}
			}
			_, err := decoder.Token()
			return false, err
		}
	}
	return false, nil
}

// scanWorkflowJSONL applies the workflow event contract before letting the
// common scanner read models or dispatches. A top-level event type and its
// nested message role are one carrier, not alternate aliases: a mixed event
// must not open worker attribution or leak an assistant model into it.
func scanWorkflowJSONL(path string, f *os.File, warnings *[]string) (string, []dispatch, []string, string) {
	defer func() {
		if err := f.Close(); err != nil {
			*warnings = append(*warnings, path+": close: "+err.Error())
		}
	}()

	var opening string
	haveOpening := false
	var dispatches []dispatch
	var models []string
	var cwd string
	briefs := newBriefTable()
	position := 0
	seen := map[string]bool{}
	malformed := 0
	r := bufio.NewReader(f)
	for {
		line, readErr := r.ReadBytes('\n')
		if len(strings.TrimSpace(string(line))) > 0 {
			event, ok := parseWorkflowEvent(line)
			if !ok {
				malformed++
			} else {
				if event.Type == "user" && !haveOpening {
					text, _ := workflowUserMessageText(event.Message.Content)
					opening = firstLine(text)
					haveOpening = true
				}
				d, prompts, model, lineCwd, parsed := parseLine(line, briefs, &position)
				if !parsed {
					malformed++
				} else {
					dispatches = append(dispatches, d...)
					for _, prompt := range prompts {
						attributeTeamPromptWithBriefs(dispatches, prompt, briefs)
					}
					if model != "" && !seen[model] {
						seen[model] = true
						models = append(models, model)
					}
					if cwd == "" && lineCwd != "" {
						cwd = lineCwd
					}
				}
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			*warnings = append(*warnings, path+": read error: "+readErr.Error())
			break
		}
	}
	if malformed > 0 {
		*warnings = append(*warnings, fmt.Sprintf("%s: %d malformed line(s) skipped", path, malformed))
	}
	return opening, dispatches, models, cwd
}

type workflowEvent struct {
	Type    string                `json:"type"`
	Cwd     string                `json:"cwd"`
	Message *workflowEventMessage `json:"message"`
}

type workflowEventMessage struct {
	Role    string          `json:"role"`
	Model   string          `json:"model"`
	Content json.RawMessage `json:"content"`
}

// parseWorkflowEvent accepts only the two observed carrier shapes. Exact
// JSON member spellings matter because encoding/json otherwise accepts
// case-insensitive aliases, which would let a drifted or ambiguous event
// manufacture routing evidence.
func parseWorkflowEvent(line []byte) (workflowEvent, bool) {
	var event workflowEvent
	duplicate, err := unmarshalWorkflowEvent(line, &event)
	if duplicate || err != nil || event.Message == nil || event.Type == "" || event.Message.Role == "" || event.Type != event.Message.Role {
		return workflowEvent{}, false
	}
	switch event.Type {
	case "user":
		_, ok := workflowUserMessageText(event.Message.Content)
		return event, ok
	case "assistant":
		return event, workflowAssistantMessageContent(event.Message.Content)
	default:
		return workflowEvent{}, false
	}
}

func unmarshalWorkflowEvent(data []byte, dst any) (bool, error) {
	duplicate, err := unmarshalUniqueJSONWithMemberValidator(data, dst, func(path []string, name string) bool {
		switch {
		case len(path) == 0:
			return caseVariantJSONMember(name, "type", "cwd", "message")
		case len(path) == 1 && path[0] == "message":
			return caseVariantJSONMember(name, "role", "model", "content")
		case len(path) == 3 && path[0] == "message" && path[1] == "content" && strings.HasPrefix(path[2], "["):
			return caseVariantJSONMember(name, "type", "text")
		default:
			return false
		}
	})
	if duplicate || err != nil {
		return duplicate, err
	}

	// parseLine decodes only tool_use blocks, but encoding/json case-folds
	// their struct fields. Validate precisely that subset before parseLine sees
	// it, keeping unrelated content-block and tool-input fields compatible.
	var carrier struct {
		Type    string `json:"type"`
		Message *struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(data, &carrier); err != nil {
		return false, err
	}
	if carrier.Type != "assistant" || carrier.Message == nil {
		return false, nil
	}
	return validateWorkflowToolUseMembers(carrier.Message.Content)
}

// validateWorkflowToolUseMembers rejects aliases for the fields parseLine
// consumes as routing evidence. It deliberately inspects only blocks whose
// exact type is tool_use, and only the immediate input members parseLine uses.
func validateWorkflowToolUseMembers(content json.RawMessage) (bool, error) {
	var blocks []json.RawMessage
	if json.Unmarshal(content, &blocks) != nil {
		return false, nil // workflowAssistantMessageContent reports the malformed shape
	}
	for _, block := range blocks {
		var fields map[string]json.RawMessage
		if json.Unmarshal(block, &fields) != nil {
			return false, nil // workflowAssistantMessageContent reports the malformed shape
		}
		var blockType string
		if json.Unmarshal(fields["type"], &blockType) != nil || blockType != "tool_use" {
			continue
		}
		duplicate, err := unmarshalUniqueJSONWithMemberValidator(block, &struct{}{}, func(path []string, name string) bool {
			switch {
			case len(path) == 0:
				return caseVariantJSONMember(name, "type", "id", "name", "input")
			case len(path) == 1 && path[0] == "input":
				return caseVariantJSONMember(name, "description", "prompt", "model", "command")
			default:
				return false
			}
		})
		if duplicate || err != nil {
			return duplicate, err
		}
	}
	return false, nil
}

// workflowUserMessageText preserves the string and text-block forms observed
// in user messages. A structurally malformed content value does not consume
// the opening-user latch, so a later valid user event can still be the brief.
func workflowUserMessageText(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", false
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text, true
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &blocks) != nil || len(blocks) == 0 {
		return "", false
	}
	for _, block := range blocks {
		if block.Type != "text" {
			return "", false
		}
		if block.Text != "" {
			return block.Text, true
		}
	}
	return "", true
}

// workflowAssistantMessageContent keeps the broader assistant block shape
// that parseLine needs for tool-use dispatches while rejecting scalar, null,
// and malformed nested containers before their model can become evidence.
func workflowAssistantMessageContent(raw json.RawMessage) bool {
	if len(raw) == 0 || string(raw) == "null" {
		return false
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return true
	}
	var blocks []json.RawMessage
	if json.Unmarshal(raw, &blocks) != nil || len(blocks) == 0 {
		return false
	}
	for _, block := range blocks {
		var object map[string]json.RawMessage
		if json.Unmarshal(block, &object) != nil || object == nil {
			return false
		}
	}
	return true
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
	briefs := newBriefTable()
	position := 0
	seen := map[string]bool{}
	malformed := 0
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadBytes('\n')
		if len(strings.TrimSpace(string(line))) > 0 {
			d, prompts, m, lineCwd, ok := parseLine(line, briefs, &position)
			if !ok {
				malformed++
			} else {
				dispatches = append(dispatches, d...)
				for _, p := range prompts {
					attributeTeamPromptWithBriefs(dispatches, p, briefs)
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
func parseLine(line []byte, briefs *briefTable, position *int) (dispatches []dispatch, prompts []teamPrompt, model string, cwd string, ok bool) {
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
		case b.Name == "Bash":
			stripped := stripHeredocBodies(b.Input.Command)
			base := *position
			// The cursor is absolute, not a fixed-width block number: a dispatch
			// brief can be larger than any arbitrary stride. Every later Bash
			// block therefore sorts after every byte-position in this one.
			*position += len(stripped) + 1
			briefs.recordCommandOrdered(b.Input.Command, cwd, base)
			if !recognizeTeamSpawns {
				continue
			}
			if match, isSpawn := parseTeamSpawnSegment(b.Input.Command); isSpawn {
				s := match.spawn
				segment := match.candidate.text
				d := dispatch{
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
					description: segment,
					model:       s.model,
					harness:     s.harness,
					effort:      s.effort,
					effortSource: func() string {
						if s.effort != "" {
							return "--effort"
						}
						return ""
					}(),
					cwd:         cwd,
					teamSpawn:   true,
					teamTarget:  s.target,
					briefCutoff: base + match.candidate.start,
				}
				if recognizeBriefFiles && !namesATicket(d.description) {
					if ref, hasRef := referencedBriefPath(segment); hasRef {
						if resolved, found := briefs.resolve(ref, cwd, d.briefCutoff); found {
							d.briefText = resolved.body
							d.briefPath = resolved.path
						}
					}
				}
				dispatches = append(dispatches, d)
				if p, isPrompt := parseTeamPromptAfter(b.Input.Command, match.candidate.start); isPrompt {
					if recognizeBriefFiles {
						p.briefRef, _ = referencedBriefPath(p.text)
					}
					prompts = append(prompts, p)
				}
				continue
			}
			if p, isPrompt := parseTeamPrompt(b.Input.Command); isPrompt {
				if recognizeBriefFiles {
					p.briefRef, _ = referencedBriefPath(p.text)
				}
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

// summarizeEffortDeclarations reports controller-declared effort alongside
// the existing model judgement. It never reads worker actuals or changes a
// Verdict: observed effort is intentionally unavailable in I075.
func summarizeEffortDeclarations(repoDir string, t ticket, dispatches []dispatch, l ledger) (expected, declared, status, observed string) {
	if len(dispatches) == 0 {
		return "-", "-", "unconfirmable", "-"
	}
	expectedValues := make([]string, 0, len(dispatches))
	declaredValues := make([]string, 0, len(dispatches))
	statuses := make([]string, 0, len(dispatches))
	for _, d := range dispatches {
		expectedEffort := "-"
		if d.harness != "" && t.tier != "" {
			entry, err := model.ResolveDispatchTarget(model.DispatchTargetRequest{
				RepoDir: repoDir, Harness: d.harness, Tier: t.tier, RequestedEffort: t.effort,
			})
			if err == nil {
				expectedEffort = entry.Effort
			}
		}
		declaredEffort := d.effort
		if declaredEffort == "" {
			declaredEffort = "-"
		}
		declarationStatus := "unconfirmable"
		if d.harness != "" && d.model != "" && expectedEffort != "-" && declaredEffort != "-" {
			switch {
			case declaredEffort == expectedEffort:
				declarationStatus = "target-match"
			case effortAuthorized(l, t.id, expectedEffort, declaredEffort):
				declarationStatus = "exact-authorized-deviation"
			default:
				declarationStatus = "unauthorized-declaration"
			}
		}
		expectedValues = append(expectedValues, expectedEffort)
		declaredValues = append(declaredValues, declaredEffort)
		statuses = append(statuses, declarationStatus)
	}
	return strings.Join(expectedValues, ","), strings.Join(declaredValues, ","), strings.Join(statuses, ","), "-"
}

// DefaultTranscriptsDir derives the harness's exact-repo transcript dir
// for a repo: ~/.claude/projects/<slug>, slug being the absolute repo path
// with '/' and '.' flattened to '-'. Best-effort — the harness convention
// is undocumented; DefaultTranscriptsDirs expands it for default discovery.
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

// DefaultTranscriptsDirs returns D36's stable, de-duplicated default scope:
// the exact repo slug, every currently listed git worktree slug, and existing
// project directories sharing the exact repo-slug prefix. Git failure and a
// non-git repo deliberately degrade to the slug scan; no transcript text is
// ever executed or used to choose a path.
func DefaultTranscriptsDirs(repoDir string) ([]string, error) {
	primary, err := DefaultTranscriptsDir(repoDir)
	if err != nil {
		return nil, err
	}
	dirs := []string{primary}
	repoReal, _ := filepath.EvalSymlinks(repoDir)
	if out, err := exec.Command("git", "-C", repoDir, "worktree", "list", "--porcelain").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			path, ok := strings.CutPrefix(line, "worktree ")
			if !ok || path == "" {
				continue
			}
			if worktreeReal, err := filepath.EvalSymlinks(path); err == nil && repoReal != "" && worktreeReal == repoReal {
				continue // git can spell the primary checkout through a different symlink path
			}
			if dir, err := DefaultTranscriptsDir(path); err == nil {
				dirs = append(dirs, dir)
			}
		}
	}
	projectsDir, prefix := filepath.Dir(primary), filepath.Base(primary)+"-"
	if entries, err := os.ReadDir(projectsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
				dirs = append(dirs, filepath.Join(projectsDir, entry.Name()))
			}
		}
	}
	return dedupSorted(dirs), nil
}
