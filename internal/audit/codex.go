// Codex transcript reader (design D20/D23, ticket I041). Reads codex rollout
// JSONL — the codex harness's session format, verified dated 2026-07-25 in
// I009 and undocumented/version-drifting beyond that — into the SAME
// dispatch/subagent structures the claude reader produces, tagged with the
// "codex" flavor (I040's per-token seam), so Run's correlation and judgment
// logic needs no codex-specific branch: only the parsing here differs.
//
// Evidence sources (D20), mirrored here:
//  1. Dispatch records: spawn_agent function calls (explicit model field,
//     ticket token case-insensitive in task_name) and team spawn commands
//     (herdr/cmux `-m <model>` invocations) recorded as response_item/
//     function_call entries in the dispatching session.
//  2. Spawned-thread actuals: thread_spawn subagent files' per-turn
//     turn_context.payload.model, linked to their dispatching session by
//     shared root id (session_meta.payload.session_id) — no parent-walking,
//     per I009. These supersede the dispatch's declared model where
//     linkable, exactly as claude's linked subagent transcripts do.
//  3. A worker-session scan (D21, ticket I042) for builds that predate
//     explicit-model dispatch: a top-level session (thread_source "user",
//     no parent) that is not an orchestrator contributes its per-turn
//     models as ticket evidence, correlated by the ticket token's presence
//     in the FIRST LINE of the session's OPENING user message — the first
//     response_item/message with role "user" in file order (scanCodexLine),
//     matched on its title line only (I042 review fix C2: whole-message
//     matching let a context sentence naming a neighboring ticket attribute
//     to it). A session is an orchestrator iff it contains ANY spawn-shaped
//     record — a spawn_agent call or team-spawn start/prompt command, with
//     or without a usable model (I042 review fix C1: a model-less spawn is
//     still an orchestrator, generalizing D21's clause 3 beyond dispatches
//     that survived the explicit-model evidence gate) — whose own turns are
//     excluded, but whose dispatch records still count via the existing
//     description-match path above.
//
// Guardian auto-review threads (D23) are structurally excluded from every
// evidence path: their reported model is synthetic and must never be read.
// session_meta.payload.model is present but always null (D20) and is never
// read; model evidence is per-turn only, from turn_context.
//
// Repo scoping (D22, ticket I043): a session belongs to the audited repo iff
// its cwd resolves inside the repo (cwdInsideRepo) OR its
// session_meta.payload.git.commit_hash names a commit known to the repo
// (gitCommitProber, one git object-existence probe per distinct hash) —
// covering worktree cwds like /private/tmp team dirs and making cross-repo
// ticket-token collision impossible unless repos share history. Out-of-scope
// sessions are invisible to the audit, not "unattributed". A failing or
// absent git probe degrades to cwd-only plus a report warning naming the
// degradation, never an error.
package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// codexTeamSpawnStartRe matches a team spawn "start" command — herdr agent
// start <worker> … -- -m <model> (I009's verified example) or the cmux
// equivalent — capturing the worker name and the explicit model. Only
// commands structurally shaped like this may become dispatch evidence
// (fix, C1): an arbitrary exec_command carrying an unrelated `-m` flag
// (e.g. `git commit -m "..."`, which leads run routinely) must not.
var codexTeamSpawnStartRe = regexp.MustCompile(`^\s*(?:herdr|cmux)\s+agent\s+start\s+(\S+).*-m\s+(\S+)`)

// codexTeamSpawnPromptRe matches a team spawn "prompt" command — herdr
// agent prompt <worker> "…" or the cmux equivalent — capturing the worker
// name the prompt text (typically carrying the ticket token via a
// dispatch-task file path, I009) belongs to.
var codexTeamSpawnPromptRe = regexp.MustCompile(`^\s*(?:herdr|cmux)\s+agent\s+prompt\s+(\S+)`)

// codexTeamSpawnAnyRe matches ANY team-spawn start|prompt command, with or
// without an explicit -m model flag — the orchestrator-exclusion latch
// (D21 amended at I042 review, 0bd554a: "contains dispatch records" means
// any spawn-SHAPED record, not just ones that survived the stricter
// explicit-model evidence gate codexTeamSpawnStartRe enforces). A
// model-less "start" (the pre-I038 M4a class this ticket exists to make
// auditable) still marks the session as an orchestrator even though it
// contributes no dispatch-record evidence.
var codexTeamSpawnAnyRe = regexp.MustCompile(`^\s*(?:herdr|cmux)\s+agent\s+(?:start|prompt)\s+(\S+)`)

// codexSessionMeta is the payload of a codex rollout's session_meta line —
// the one line kind carrying thread identity and the guardian marker
// (I009, verified 2026-07-25).
type codexSessionMeta struct {
	ID             string `json:"id"`         // this thread's own id
	SessionID      string `json:"session_id"` // the tree's ROOT thread id (shared tree-wide)
	ParentThreadID string `json:"parent_thread_id"`
	Cwd            string `json:"cwd"`
	ThreadSource   string `json:"thread_source"` // "user" (top-level) or "subagent"
	Source         struct {
		Subagent struct {
			Other       string          `json:"other"`        // "guardian" on auto-review threads (D23)
			ThreadSpawn json.RawMessage `json:"thread_spawn"` // present on real codex-native subagents
		} `json:"subagent"`
	} `json:"source"`
	Git struct {
		CommitHash string `json:"commit_hash"` // D22/I043 repo-scoping signal; no remote URL is ever present (I009)
	} `json:"git"`
}

// isGuardian reports whether this thread is a guardian auto-review subagent
// — structurally excluded from every evidence path (D23): its reported
// model is synthetic (codex-auto-review), never a routed one.
func (m codexSessionMeta) isGuardian() bool {
	return m.ThreadSource == "subagent" && m.Source.Subagent.Other == "guardian"
}

// isSubagent reports whether this thread is a real codex-native subagent
// (thread_spawn present) — a spawned-thread-actuals source (D20 clause 2),
// distinct from a guardian and from a plain top-level "user" session.
func (m codexSessionMeta) isSubagent() bool {
	return m.ThreadSource == "subagent" && len(m.Source.Subagent.ThreadSpawn) > 0
}

// isTopLevelUser reports whether this thread is a top-level, user-initiated
// codex session — D21's worker-session-scan candidate set (thread_source
// "user", no parent): a lead or a worker, as opposed to a codex-native
// subagent or guardian thread (both thread_source "subagent").
func (m codexSessionMeta) isTopLevelUser() bool {
	return m.ThreadSource == "user" && m.ParentThreadID == ""
}

// rootID is the thread tree's linking key (D20 clause 2): every file in a
// tree, the root included, carries the same session_id. Filtering on this
// one field finds every member; no parent-walking is needed (I009).
func (m codexSessionMeta) rootID() string {
	if m.SessionID != "" {
		return m.SessionID
	}
	return m.ID
}

// codexResponseItem is one response_item's payload, covering the two shapes
// this reader needs: a function_call (dispatch evidence) and a message
// (the opening-user-message carrier, D21/I042). Other response_item shapes
// (reasoning, …) decode harmlessly with empty fields and are skipped.
type codexResponseItem struct {
	Type      string          `json:"type"`
	Name      string          `json:"name"`      // function_call
	Arguments json.RawMessage `json:"arguments"` // function_call
	Role      string          `json:"role"`      // message
	Content   json.RawMessage `json:"content"`   // message
}

// codexMessageText extracts the human-readable text from a message
// response_item's content field: an array of typed parts, e.g.
// [{"type":"input_text","text":"..."}] (codex mirrors the OpenAI Responses
// API item shape here — undocumented but internally consistent, Testing
// Decisions). Parts are concatenated in file order; a bare string content is
// also accepted. No recognized shape yields "", never an error —
// degrade-never-fail.
func codexMessageText(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var parts []struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) != nil {
		return ""
	}
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(p.Text)
	}
	return b.String()
}

// codexFunctionArgs decodes a function_call's arguments into its string
// fields. Real codex (OpenAI function-calling convention) JSON-string-
// encodes arguments; this accepts that or a bare object. Non-string field
// values are skipped rather than failing the whole decode — an unrelated
// numeric field in a call's arguments must not lose the string fields this
// reader actually needs.
func codexFunctionArgs(raw json.RawMessage) map[string]string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		raw = json.RawMessage(s)
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return nil
	}
	out := map[string]string{}
	for k, v := range obj {
		var sv string
		if json.Unmarshal(v, &sv) == nil {
			out[k] = sv
		}
	}
	return out
}

// codexFileResult is what scanCodexFile extracts from one rollout file: its
// own thread identity, any dispatch records it carries, and its per-turn
// models (used only when the file turns out to be a genuine subagent
// thread — session_meta.payload.model is never read; every turn_context
// line contributes its own token, D20).
type codexFileResult struct {
	meta               codexSessionMeta
	dispatches         []dispatch
	dispatched         bool // I042 review fix (C1): ANY spawn-shaped record seen, model or not — see dispatches' doc
	turnModels         []string
	openingUserMessage string // D21/I042: first role="user" message in file order
	laterUserText      string // D24/I044: role="user" message text AFTER the opening one, concatenated — near-miss detection only, never attribution (D21's opening-line rule is unchanged)
}

// searchText is the near-miss detection surface (D24, ticket I044): every
// piece of this file's text that could name a ticket, regardless of whether
// it went on to become real evidence — the opening message (full, not just
// its first line), any later user messages, and every dispatch record's
// description (spawn_agent task_name or team-spawn command text, which
// typically carries the ticket token via a dispatch-task file path, I009).
// Used only to detect "material exists but failed attribution"; it plays no
// part in attribution itself.
func (r codexFileResult) searchText() string {
	var b strings.Builder
	b.WriteString(r.openingUserMessage)
	b.WriteByte('\n')
	b.WriteString(r.laterUserText)
	for _, d := range r.dispatches {
		b.WriteByte('\n')
		b.WriteString(d.description)
	}
	return b.String()
}

// codexNearMiss records repo-scoped, codex material that mentioned a ticket
// but failed attribution (D24): a guardian-excluded file, a worker session
// whose opening-message first line lacked the token, or an orchestrator
// session whose own turns are excluded. Matched against ticket tokens in
// readCodexSessions' caller exactly like dispatches/agents are, but
// contributes no evidence — only the unattributed-transcript verdict, and
// only when a ticket has no attributed evidence at all (Run never lets a
// near miss downgrade real evidence).
type codexNearMiss struct {
	text   string // searchable text (searchText()) to match ticket tokens against
	file   string // source transcript file
	reason string // human-readable why-excluded phrase
}

// codexExecWorker accumulates one team-spawned worker's evidence across its
// (possibly several) exec_command calls within one file: the "start" call
// binds the worker name to its explicit model, and the "prompt" call's text
// (which typically carries the ticket token, I009) attaches to that same
// worker (fix, C2). Keying per worker name — rather than one session-wide
// accumulator — is what stops two correctly-tiered spawns for two different
// tickets from colliding into a single, wrong dispatch record.
type codexExecWorker struct {
	model string
	text  strings.Builder
}

// codexScanState is scanCodexFile's mutable per-file accumulator, threaded
// through scanCodexLine one line at a time.
type codexScanState struct {
	res         codexFileResult
	haveMeta    bool
	haveOpening bool // D21/I042: opening user message already captured
	seenModel   map[string]bool
	execWorkers map[string]*codexExecWorker
}

// parseCodexBytes parses one rollout JSONL file's already-read bytes. ok is
// false when no session_meta line was found or parsed — without thread
// identity the file cannot be linked or repo-scoped, so it is unusable (a
// warning, not an evidence source). All other trouble degrades to a
// per-file warning, mirroring scanJSONL's posture for the claude reader.
//
// Takes data rather than reading path itself (I049): readCodexSessions
// already reads the file once for the discovery pre-filter's token scan
// and passes those SAME bytes on here — reading path a second time would
// double the I/O for every file that survives the pre-filter, which
// live-measured cost more than pruning saved (most codex stores mention
// audited ticket ids often enough in ordinary prose that few files actually
// prune, so avoiding the redundant read is where I049's real win comes
// from, not the prune count alone).
func parseCodexBytes(path string, data []byte, warnings *[]string) (codexFileResult, bool) {
	st := &codexScanState{seenModel: map[string]bool{}, execWorkers: map[string]*codexExecWorker{}}
	malformed := 0
	for _, line := range bytes.Split(data, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if !scanCodexLine(line, st) {
			malformed++
		}
	}
	if malformed > 0 {
		*warnings = append(*warnings, fmt.Sprintf("%s: %d malformed line(s) skipped", path, malformed))
	}
	// One dispatch per worker with an explicit declared model (D20: a
	// dispatch needs an explicit model to be evidence). Sorted by name for
	// deterministic output — map iteration order is not.
	names := make([]string, 0, len(st.execWorkers))
	for name := range st.execWorkers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		w := st.execWorkers[name]
		if w.model != "" {
			st.res.dispatches = append(st.res.dispatches, dispatch{model: w.model, description: w.text.String()})
		}
	}
	if !st.haveMeta {
		*warnings = append(*warnings, path+": no session_meta line — skipped")
		return codexFileResult{}, false
	}
	return st.res, true
}

// scanCodexLine handles one JSONL line, folding its evidence into st.res or
// st.execWorkers. Returns false only for a line whose recognized shape
// failed to parse.
func scanCodexLine(line []byte, st *codexScanState) bool {
	var ev struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if json.Unmarshal(line, &ev) != nil {
		return false
	}
	switch ev.Type {
	case "session_meta":
		if json.Unmarshal(ev.Payload, &st.res.meta) != nil {
			return false
		}
		st.haveMeta = true
	case "turn_context":
		var tc struct {
			Model string `json:"model"`
		}
		if json.Unmarshal(ev.Payload, &tc) != nil {
			return false
		}
		if tc.Model != "" && !st.seenModel[tc.Model] {
			st.seenModel[tc.Model] = true
			st.res.turnModels = append(st.res.turnModels, tc.Model)
		}
	case "response_item":
		var item codexResponseItem
		if json.Unmarshal(ev.Payload, &item) != nil {
			return false
		}
		switch item.Type {
		case "function_call":
			args := codexFunctionArgs(item.Arguments)
			if item.Name == "spawn_agent" {
				// I042 review fix (C1): ANY spawn_agent call marks the
				// session as an orchestrator, model or not — the D21
				// amendment's broad "dispatch record" reading. A model-less
				// spawn_agent contributes no dispatch-record evidence below
				// but must still exclude this session's own turns.
				st.res.dispatched = true
				if m := args["model"]; m != "" {
					st.res.dispatches = append(st.res.dispatches, dispatch{
						model:       m,
						description: args["task_name"],
					})
				}
				return true
			}
			cmd := args["command"]
			if cmd == "" {
				return true
			}
			// I042 review fix (C1): a team-spawn command shape (start or
			// prompt) marks the session as an orchestrator regardless of
			// whether it carries an explicit -m model — codexTeamSpawnAnyRe
			// is intentionally looser than the two evidence-producing
			// regexes below.
			if codexTeamSpawnAnyRe.MatchString(cmd) {
				st.res.dispatched = true
			}
			// Only commands structurally shaped like a team spawn (herdr/cmux
			// agent start|prompt) may become dispatch evidence (fix, C1) — an
			// arbitrary exec_command carrying an unrelated -m flag (a lead
			// committing routinely, `git commit -m "..."`) is not a dispatch
			// record and must contribute nothing.
			if match := codexTeamSpawnStartRe.FindStringSubmatch(cmd); match != nil {
				w := st.execWorker(match[1])
				w.model = match[2]
				w.text.WriteString(cmd)
				w.text.WriteByte(' ')
			} else if match := codexTeamSpawnPromptRe.FindStringSubmatch(cmd); match != nil {
				w := st.execWorker(match[1])
				w.text.WriteString(cmd)
				w.text.WriteByte(' ')
			}
		case "message":
			// Opening message = the first role="user" response_item/message
			// in file order (D21). Only the first is captured for
			// attribution — a later message naming a different ticket
			// (neighboring-ticket bleed, I009) must never overwrite it or
			// attribute, and non-user messages (e.g. an assistant turn
			// preceding the first real user line, if any) are skipped
			// without latching haveOpening. Later user messages are still
			// captured into laterUserText (D24/I044) — near-miss detection
			// only, so a later mention reports unattributed-transcript
			// rather than vanishing as no-transcript; attribution itself is
			// unchanged.
			if item.Role == "user" {
				text := codexMessageText(item.Content)
				if !st.haveOpening {
					st.res.openingUserMessage = text
					st.haveOpening = true
				} else {
					st.res.laterUserText += "\n" + text
				}
			}
		}
	}
	// event_msg / world_state and any other recognized-but-irrelevant line
	// kinds carry no dispatch evidence and are silently skipped, not
	// malformed — mirroring parseLine's treatment of non-assistant events.
	return true
}

// execWorker returns the accumulator for a team-spawned worker name,
// creating it on first reference. Order of start/prompt calls does not
// matter — both branches key into the same map.
func (st *codexScanState) execWorker(name string) *codexExecWorker {
	w, ok := st.execWorkers[name]
	if !ok {
		w = &codexExecWorker{}
		st.execWorkers[name] = w
	}
	return w
}

// cwdInsideRepo reports whether cwd resolves inside absRepo — clause 1 of
// D22's repo-scoping rule (I041 seam, extended by clause 2 below). A missing
// or unresolvable cwd cannot be proven to belong to the audited repo and is
// excluded rather than leniently included: a false negative here costs one
// missed session (no-transcript, at worst, unless clause 2 admits it via a
// known commit); a false positive would be exactly the cross-repo
// attribution I008 exists to prevent.
func cwdInsideRepo(cwd, absRepo string) bool {
	if cwd == "" {
		return false
	}
	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRepo, absCwd)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// gitCommitProber implements D22 clause 2: a codex session whose cwd falls
// outside the repo is still in scope if its session_meta.payload.git.
// commit_hash names a commit KNOWN to the audited repo's object store — the
// signal that makes worktree cwds (/private/tmp team dirs) visible. The
// probe mechanism itself (does `git` work at all against this repo dir) is
// checked lazily, on first use, and cached: most audits are fully cwd-scoped
// and never need a probe at all (readCodexSessions short-circuits on
// cwdInsideRepo before calling knows), so a repo that happens not to be a
// git checkout must not warn when nothing ever needed the probe. Once
// checked, per-distinct-hash results are cached too — a 951-file session
// store must not fork 951 git processes, only one per distinct hash it
// actually needs.
type gitCommitProber struct {
	repoDir   string
	warnings  *[]string
	checked   bool
	available bool
	cache     map[string]bool
}

func newGitCommitProber(repoDir string, warnings *[]string) *gitCommitProber {
	return &gitCommitProber{repoDir: repoDir, warnings: warnings, cache: map[string]bool{}}
}

// gitObjectIDRe matches a plausible raw git object id: lowercase hex,
// abbreviated (4 chars) up to full-length (40 chars, sha1) or longer
// (sha256 repos use 64). Guards knows against ref-ish inputs (I043 review
// finding I1): `git cat-file -e <rev>^{commit}` resolves ANY valid revision
// expression, not just a raw object id — "main^{commit}" or "HEAD^{commit}"
// both exit 0 in a repo with commits, so a non-SHA commit_hash (branch
// name, HEAD, a future format's ref-ish field) would false-positive into
// nearly every repo, exactly the cross-repo false-positive class D22 exists
// to prevent. Today's verified wire format always sends a real SHA (I009);
// this guards against format drift, not today's known shape.
var gitObjectIDRe = regexp.MustCompile(`^[0-9a-f]{4,64}$`)

// knows reports whether hash names a commit in p.repoDir's object store. A
// missing/empty hash, or one that doesn't look like a raw object id
// (gitObjectIDRe), trivially contributes no probe (D22) — treated
// identically to the empty-hash case, never shelled out to git. Degrade-
// never-fail: if the probe mechanism itself doesn't work — repoDir is not a
// git checkout, or the git binary is unavailable — knows reports false
// (cwd-only behavior) and records exactly one report warning naming the
// degradation, no matter how many hashes are asked about.
func (p *gitCommitProber) knows(hash string) bool {
	if hash == "" || !gitObjectIDRe.MatchString(hash) {
		return false
	}
	if !p.checked {
		p.checked = true
		p.available = exec.Command("git", "-C", p.repoDir, "rev-parse", "--git-dir").Run() == nil
		if !p.available {
			*p.warnings = append(*p.warnings, fmt.Sprintf(
				"codex repo scoping: git commit-hash probe unavailable for %s (not a git repository, or git is not installed) — degrading to cwd-only scoping",
				p.repoDir))
		}
	}
	if !p.available {
		return false
	}
	if known, ok := p.cache[hash]; ok {
		return known
	}
	known := exec.Command("git", "-C", p.repoDir, "cat-file", "-e", hash+"^{commit}").Run() == nil
	p.cache[hash] = known
	return known
}

// codexMayContribute is the I049 pre-filter predicate: a cheap raw-bytes
// check of whether data COULD contribute evidence to any of tokens (audited
// ticket ids), run before the full JSONL parse. Two ways to survive it:
//
//  1. the JSON string-value shape `:"subagent"` (compact JSON, no space
//     after ':' — guaranteed by the JSONL format itself: a rollout line is
//     one JSON object per line, so codex cannot be emitting embedded
//     newlines inside a value either, meaning every standard encoder's
//     compact form applies here). D20 clause 2 (spawned-thread actuals)
//     links a codex-native subagent's turn_context evidence to its
//     dispatching session PURELY by shared root id
//     (session_meta.payload.session_id) — never by ticket-token text in the
//     subagent file's own bytes (TestCodexSpawnedThreadActualSupersedesDeclared's
//     sub.jsonl carries no ticket token at all). A subagent-shaped file
//     (thread_source "subagent" — real subagent or guardian, codexSessionMeta.
//     Source.Subagent) can therefore never be soundly proven irrelevant by
//     token absence alone, so it is always let through — codex's own
//     session_meta encodes it as `"thread_source":"subagent"` (codexSessionMetaLine/
//     threadSpawnSource/guardianSource in codex_test.go show the exact
//     shapes; a plain top-level user session's source is "{}" and never
//     contains this string), which `:"subagent"` matches without assuming
//     the specific key name "thread_source" — a placement this package
//     elsewhere calls "undocumented, version-drifting" (I009). Bare
//     "subagent" (no colon/quote, case-folded) was measured live against
//     this repo's own ~/.codex/sessions store to match ~97% of it: this
//     project's subject matter IS subagent auditing, so plain-English
//     prose mentions of the word are common here specifically, and a
//     marker that can't tell code from prose prunes almost nothing on this
//     particular repo.
//  2. tokens: a ticket id appears anywhere in data as a literal byte
//     string, either upper (a doc/ticket reference) or lower (D20: codex's
//     task_name convention lowercases ids) — the only two case forms a
//     ticket id (one uppercase letter, digits only otherwise,
//     docs/issues/README.md convention) ever appears in. This covers every
//     evidence path except clause 1 above: spawn_agent task_name, a
//     team-spawn command's text (which carries the token via a
//     dispatch-task file path, I009), and the opening-user-message carrier
//     (D21) all place the token in the file's own bytes.
//
// Both checks are over-inclusive by construction (a false "yes" only costs
// a wasted full parse); under-inclusion is what would break soundness, so
// neither check ever narrows past what's proven safe. Neither case-folds
// its ENTIRE input: bytes.ToUpper(data) (an O(n) allocation and copy per
// file) was measured live to cost more than pruning saved, on top of
// catching even more incidental prose.
func codexMayContribute(data []byte, tokens []string) bool {
	if bytes.Contains(data, []byte(`:"subagent"`)) {
		return true
	}
	for _, tok := range tokens {
		if tok == "" {
			continue
		}
		upper := strings.ToUpper(tok)
		lower := strings.ToLower(tok)
		if bytes.Contains(data, []byte(upper)) || bytes.Contains(data, []byte(lower)) {
			return true
		}
	}
	return false
}

// readCodexSessions walks a codex sessions dir — real layout
// <dir>/YYYY/MM/DD/rollout-<ts>-<uuid>.jsonl, though any nesting is
// accepted — collecting dispatch records and spawned-thread actuals (D20)
// as the same dispatch/subagent structures readTranscripts produces,
// tagged with the codex flavor, plus near-miss records (D24, ticket I044)
// for repo-scoped material that mentioned a ticket but failed attribution.
// Out-of-repo sessions (D22: neither cwdInsideRepo nor a known commit hash,
// gitCommitProber) are excluded before anything else — including near-miss
// detection — so out-of-scope material stays invisible to this audit, never
// "unattributed" ("Sessions outside scope are invisible to attribution —
// they are not 'unattributed', they simply do not exist for this audit,"
// I043's ticket text, D22). Guardian threads (D23)
// are structurally excluded from evidence but, once in scope, DO produce a
// near miss: their content still gets searched for ticket mentions so a
// guardian-only match reports honestly rather than as no-transcript. All
// trouble degrades to a warning, never an error — an undocumented, drifting
// format must never break the verify stage (design D-doc, "spine
// maintainer" story).
//
// since and sessionID implement D28's --since/--session operator filters
// (ticket I047), layered on top of D22's hard repo scoping rather than
// replacing it: --since compares each rollout FILE's own mtime against the
// cutoff (parseSince's doc — the format is undocumented, a file's mtime
// needs no parsing of it at all), skipped before the file is even opened;
// --session matches session_meta.payload.session_id, the thread tree's
// ROOT id (codexSessionMeta.rootID) — the same id every file in one lead+
// workers tree shares, so restricting to one session id keeps that whole
// tree together, mirroring what --session does for a claude session and
// its subagents. A zero since and empty sessionID (every pre-I047 caller)
// filter nothing.
//
// tokens implements I049's discovery pre-filter: after the mtime skip above
// (cheapest — no file open) each surviving file is read ONCE into memory;
// codexMayContribute checks those bytes for the pre-filter (cheap — a
// handful of substring scans) before parseCodexBytes decodes the same bytes
// line by line (expensive — ~12-14s live over 953 real files, the ticket's
// own measurement, dominated by per-line JSON unmarshaling). A file proven
// unable to contribute (see codexMayContribute's doc) is dropped without
// ever reaching parseCodexBytes. The pre-filter itself is skipped whenever
// sessionID is set: --session's matchedSession diagnostic (M3, I047 review)
// needs every candidate file's rootID parsed to know whether the requested
// id exists in the store at all, even for a file that turns out to carry no
// ticket token — pruning it first would silently flip "matched" to "matched
// no sessions" and violate AC2's byte-identical-Reports requirement, for a
// query that isn't the whole-store sweep the pre-filter targets anyway. A
// read failure degrades exactly as it always has — a per-file "unreadable"
// warning, never an error — and is never treated as pre-filter proof of
// anything.
func readCodexSessions(dir, repoDir string, since time.Time, sessionID string, tokens []string, warnings *[]string) ([]dispatch, []subagent, []codexNearMiss, bool) {
	absRepo, err := filepath.Abs(repoDir)
	if err != nil {
		absRepo = repoDir
	}
	var files []string
	walkErr := filepath.WalkDir(dir, func(path string, de os.DirEntry, err error) error {
		if err != nil {
			if path == dir {
				return err // the sessions dir itself is missing/unreadable
			}
			return nil // a deeper entry failed to stat; skip it, keep walking
		}
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".jsonl") {
			return nil
		}
		if !since.IsZero() {
			if info, err := de.Info(); err == nil && info.ModTime().Before(since) {
				return nil // --since: older than the cutoff, skipped before it's even opened
			}
		}
		files = append(files, path)
		return nil
	})
	if walkErr != nil {
		*warnings = append(*warnings, "codex sessions dir unreadable — codex tickets will report no-transcript: "+walkErr.Error())
		return nil, nil, nil, sessionID == ""
	}
	sort.Strings(files)

	prober := newGitCommitProber(absRepo, warnings)
	var dispatches []dispatch
	var agents []subagent
	var nearMisses []codexNearMiss
	// matchedSession is M3's diagnostic input (I047 review): whether
	// sessionID (non-empty) equaled at least one root id among files that
	// SURVIVED the --since walk-time skip above. Unlike claude's
	// sessionInScope (id lives in the filename, checkable before any
	// parsing), a codex session's id lives inside the file
	// (session_meta.payload.session_id; the filename is
	// rollout-<ts>-<uuid>.jsonl, uuid unrelated to the thread id) — so a
	// --since-excluded file's id is never discovered at all, and can't
	// contribute to this check. An operator combining --since with a
	// codex --session id that only exists in an excluded file gets "matched
	// no sessions" too; that is still an honest answer (nothing in scope
	// has that id), just not "the id exists nowhere in the store".
	matchedSession := sessionID == ""
	for _, path := range files {
		// I049: read once, reuse the same bytes for both the pre-filter and
		// the parse below — reading path a second time inside the parse
		// step would double the I/O for every file that survives, which
		// live-measured cost more than pruning saved. A read failure here
		// degrades exactly as it always has (a warning, never an error) and
		// is never treated as pre-filter proof of anything.
		data, err := os.ReadFile(path)
		if err != nil {
			*warnings = append(*warnings, path+": unreadable: "+err.Error())
			continue
		}
		if sessionID == "" && !codexMayContribute(data, tokens) {
			continue // no ticket token, not subagent-shaped; cannot contribute
		}
		res, ok := parseCodexBytes(path, data, warnings)
		if !ok {
			continue
		}
		if sessionID != "" {
			if res.meta.rootID() != sessionID {
				continue // --session: restrict to one thread tree
			}
			matchedSession = true
		}
		// D22: in scope iff cwd resolves inside the repo (clause 1) or the
		// session's git commit hash is known to the repo (clause 2). The
		// short-circuit means a fully cwd-scoped session never asks the
		// prober anything, keeping probe count and probe-failure warnings
		// bounded to builds that actually use worktree cwds. Checked BEFORE
		// the guardian exclusion below (D24 fix note: an out-of-scope
		// guardian file must stay invisible, not surface as a near miss —
		// regression-guarded by TestCodexOutOfScopeGuardianProducesNoNearMiss,
		// I044 fix round 1).
		if !cwdInsideRepo(res.meta.Cwd, absRepo) && !prober.knows(res.meta.Git.CommitHash) {
			continue
		}
		root := res.meta.rootID()
		for i := range res.dispatches {
			res.dispatches[i].toolUseID = "codex:" + root
			res.dispatches[i].flavor = "codex"
			res.dispatches[i].sourceFile = path
		}
		if res.meta.isGuardian() {
			// D23: never evidence, in any path — but its content (e.g. a
			// quoted/replayed spawn_agent call, D23's own regression
			// fixture) can still name a ticket. That's a guardian-only
			// match (D24): found, but structurally unusable.
			if text := res.searchText(); strings.TrimSpace(text) != "" {
				nearMisses = append(nearMisses, codexNearMiss{
					text:   text,
					file:   path,
					reason: "guardian auto-review thread — structurally excluded from evidence (D23)",
				})
			}
			continue
		}
		dispatches = append(dispatches, res.dispatches...)
		if res.meta.isSubagent() && len(res.turnModels) > 0 {
			agents = append(agents, subagent{toolUseID: "codex:" + root, models: res.turnModels, flavor: "codex", sourceFile: path})
		}
		// D21 worker-session scan (I042): a top-level session with no
		// dispatch records of its own (clause 3, orchestrator exclusion —
		// gated on res.dispatched, not len(res.dispatches): I042 review fix
		// C1, any spawn-shaped record excludes, model or not) contributes
		// its per-turn models as evidence, correlated in Run by the ticket
		// token's presence in the FIRST LINE of its opening user message
		// (clauses 1-2; I042 review fix C2 — matching the whole message let
		// a context sentence naming a neighboring ticket attribute to it,
		// mirroring the fix Run's dispatch matching already applies via
		// firstLine(d.prompt)) — the same description+containsToken seam
		// claude subagents already use, so Run needs no codex-specific
		// worker-scan logic.
		if res.meta.isTopLevelUser() {
			if res.dispatched {
				// D24: an orchestrator's own turns never attribute (D21
				// clause 3), but its opening message can still name a
				// ticket — an orchestrator-only mention, found but
				// structurally unusable as own-turn evidence. Its dispatch
				// records, if any, are separate legitimate evidence handled
				// via the normal dispatches path above.
				if text := res.searchText(); strings.TrimSpace(text) != "" {
					nearMisses = append(nearMisses, codexNearMiss{
						text:   text,
						file:   path,
						reason: "orchestrator session — its own turns are excluded from ticket evidence (D21); only its dispatch records count separately",
					})
				}
			} else if len(res.turnModels) > 0 {
				agents = append(agents, subagent{
					description: firstLine(res.openingUserMessage),
					models:      res.turnModels,
					flavor:      "codex",
					sourceFile:  path,
				})
				// D24: the same file's FULLER text (opening message beyond
				// its first line, plus any later user messages) is a
				// near-miss surface for any OTHER ticket mentioned there —
				// the mid-transcript-only-match case. A ticket that IS
				// named in the first line already has real evidence above
				// and never consults this.
				if text := res.searchText(); strings.TrimSpace(text) != "" {
					nearMisses = append(nearMisses, codexNearMiss{
						text:   text,
						file:   path,
						reason: "ticket token absent from the opening message's first line — a later mention does not attribute (D21)",
					})
				}
			}
		}
	}
	return dispatches, agents, nearMisses, matchedSession
}

// DefaultCodexSessionsDir derives the discovery default for codex's session
// store (design D-doc, "Flavor threading" closing paragraph): $CODEX_HOME/
// sessions if CODEX_HOME is set, else ~/.codex/sessions. Mirrors
// DefaultTranscriptsDir for the CLI's --codex-sessions flag.
func DefaultCodexSessionsDir() (string, error) {
	if home := os.Getenv("CODEX_HOME"); home != "" {
		return filepath.Join(home, "sessions"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "sessions"), nil
}
