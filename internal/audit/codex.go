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
//     no parent) with no dispatch records of its own contributes its
//     per-turn models as ticket evidence, correlated by the ticket token's
//     presence in the session's OPENING user message — the first
//     response_item/message with role "user" in file order (scanCodexLine).
//     A session that dispatches is an orchestrator (clause 3): its own
//     turns are excluded, but its dispatch records still count via the
//     existing description-match path above.
//
// Guardian auto-review threads (D23) are structurally excluded from every
// evidence path: their reported model is synthetic and must never be read.
// session_meta.payload.model is present but always null (D20) and is never
// read; model evidence is per-turn only, from turn_context.
package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
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
	turnModels         []string
	openingUserMessage string // D21/I042: first role="user" message in file order
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

// scanCodexFile reads one rollout JSONL file. ok is false when no
// session_meta line was found or parsed — without thread identity the file
// cannot be linked or repo-scoped, so it is unusable (a warning, not an
// evidence source). All other trouble degrades to a per-file warning,
// mirroring scanJSONL's posture for the claude reader.
func scanCodexFile(path string, warnings *[]string) (codexFileResult, bool) {
	f, err := os.Open(path)
	if err != nil {
		*warnings = append(*warnings, path+": unreadable: "+err.Error())
		return codexFileResult{}, false
	}
	defer f.Close()

	st := &codexScanState{seenModel: map[string]bool{}, execWorkers: map[string]*codexExecWorker{}}
	malformed := 0
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadBytes('\n')
		if strings.TrimSpace(string(line)) != "" {
			if !scanCodexLine(line, st) {
				malformed++
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
			// in file order (D21). Only the first is captured — a later
			// message naming a different ticket (neighboring-ticket bleed,
			// I009) must never overwrite it, and non-user messages (e.g. an
			// assistant turn preceding the first real user line, if any) are
			// skipped without latching haveOpening.
			if !st.haveOpening && item.Role == "user" {
				st.res.openingUserMessage = codexMessageText(item.Content)
				st.haveOpening = true
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

// cwdInsideRepo reports whether cwd resolves inside absRepo. This is the
// simplest correct repo scoping available to I041 — full D22 scoping (a
// git-commit-hash probe for worktree cwds like /private/tmp team dirs) is
// I043's job; this cwd-only check is the seam it extends. A missing or
// unresolvable cwd cannot be proven to belong to the audited repo and is
// excluded rather than leniently included: a false negative here costs one
// missed session (no-transcript, at worst); a false positive would be
// exactly the cross-repo attribution I008 exists to prevent.
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

// readCodexSessions walks a codex sessions dir — real layout
// <dir>/YYYY/MM/DD/rollout-<ts>-<uuid>.jsonl, though any nesting is
// accepted — collecting dispatch records and spawned-thread actuals (D20)
// as the same dispatch/subagent structures readTranscripts produces,
// tagged with the codex flavor. Guardian threads (D23) and out-of-repo
// sessions (cwdInsideRepo) are structurally excluded before their content
// is ever folded into evidence. All trouble degrades to a warning, never an
// error — an undocumented, drifting format must never break the verify
// stage (design D-doc, "spine maintainer" story).
func readCodexSessions(dir, repoDir string, warnings *[]string) ([]dispatch, []subagent) {
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
		if !de.IsDir() && strings.HasSuffix(de.Name(), ".jsonl") {
			files = append(files, path)
		}
		return nil
	})
	if walkErr != nil {
		*warnings = append(*warnings, "codex sessions dir unreadable — codex tickets will report no-transcript: "+walkErr.Error())
		return nil, nil
	}
	sort.Strings(files)

	var dispatches []dispatch
	var agents []subagent
	for _, path := range files {
		res, ok := scanCodexFile(path, warnings)
		if !ok {
			continue
		}
		if res.meta.isGuardian() {
			continue // D23: never evidence, in any path
		}
		if !cwdInsideRepo(res.meta.Cwd, absRepo) {
			continue
		}
		root := res.meta.rootID()
		for i := range res.dispatches {
			res.dispatches[i].toolUseID = "codex:" + root
			res.dispatches[i].flavor = "codex"
		}
		dispatches = append(dispatches, res.dispatches...)
		if res.meta.isSubagent() && len(res.turnModels) > 0 {
			agents = append(agents, subagent{toolUseID: "codex:" + root, models: res.turnModels, flavor: "codex"})
		}
		// D21 worker-session scan (I042): a top-level session with no
		// dispatch records of its own (clause 3, orchestrator exclusion)
		// contributes its per-turn models as evidence, correlated in Run by
		// the ticket token's presence in its opening user message (clauses
		// 1-2) — the same description+containsToken seam claude subagents
		// already use, so Run needs no codex-specific worker-scan logic.
		if res.meta.isTopLevelUser() && len(res.dispatches) == 0 && len(res.turnModels) > 0 {
			agents = append(agents, subagent{
				description: res.openingUserMessage,
				models:      res.turnModels,
				flavor:      "codex",
			})
		}
	}
	return dispatches, agents
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
