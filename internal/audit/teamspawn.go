package audit

import (
	"path"
	"regexp"
	"strings"
)

// --- claude-team Bash dispatches (ticket I090) ---
//
// The claude-team transport (herdr / cmux) starts workers as shell commands
// rather than Task/Agent tool calls, so before I090 an entire team-built
// effort carried no dispatch record at all and every one of its tickets
// reported no-transcript — the routing contract's only enforcement was
// silent in the owner's default working mode. This file recognizes the two
// spawn shapes inside a Bash tool_use block's command text:
//
//	herdr agent start <name> --kind claude --pane <id> -- … --model <id> [--effort <e>]
//	cmux send --surface <id> '… claude … --model <id> [--effort <e>]'
//
// `--surface` is the flag cmux actually ships (`cmux send --surface
// surface:19 '…'`); `--pane`/`--target` are accepted alongside it.
//
// A second recognizer for the same commands lives in codex.go
// (codexTeamSpawnStartRe / codexTeamSpawnPromptRe) for codex rollout
// transcripts. The two pair a worker with its prompt differently — this one
// takes the FIRST prompt after a spawn, codex accumulates ALL of a worker's
// prompt text. Both readings are defensible; that they differ by flavor is
// not. Ticket I102 tracks sharing one worker-keyed pairing.
//
// and the follow-up prompt commands (`herdr agent prompt <name> …`,
// a `cmux send` carrying no claude invocation) that a spawn borrows its
// ticket attribution from when the spawn command names no ticket itself.
//
// Recognition is deliberately narrow: a command counts only when the spawn
// appears in COMMAND position (first word of the command or of one of its
// shell-separated segments) and outside any heredoc body, so `herdr pane
// list`, a `cmux send` running something other than claude, and a doc being
// written that merely quotes `herdr agent start` are all not dispatches.
// Everything downstream — repo qualification, ticket attribution, and the
// model-vs-tier judgement — is the existing Task/Agent logic unchanged.

// teamSpawn is one recognized worker spawn: the model the worker runs on,
// its effort when the command declares one, and the worker handle (herdr
// agent name / cmux pane) a following prompt command is paired by.
type teamSpawn struct {
	model  string
	effort string
	target string
}

// teamPrompt is one recognized follow-up prompt command: the worker handle
// it addresses and the command's own text, which is the attribution source
// for a spawn that named no ticket.
type teamPrompt struct {
	target    string
	text      string
	briefRef  string
	briefText string
	briefPath string
}

// ticketTokenRe matches a ticket-id-shaped token (I090, D3), the test for
// whether a spawn command carries its own attribution. It is deliberately
// shape-based rather than a lookup of the audited ids: parsing happens per
// transcript line, well before any ticket list is in reach, and a spawn that
// names some OTHER repo's ticket must still not silently borrow the next
// prompt's attribution.
//
// The shape is broader than any real id — an unrelated token such as ISO8601
// or SHA256 reads as "names a ticket" too. That direction is deliberate: a
// false positive only stops the spawn from borrowing the following prompt's
// attribution, so the ticket degrades to no-transcript (the pre-I090
// verdict, visibly unjudged) rather than being attributed to a ticket nobody
// dispatched. Misattribution is the failure this audit must never make;
// under-attribution it already reports honestly.
var ticketTokenRe = regexp.MustCompile(`(^|[^A-Za-z0-9])[A-Z]{1,4}[0-9]{2,5}([^A-Za-z0-9]|$)`)

// namesATicket reports whether text carries a ticket-id-shaped token.
func namesATicket(text string) bool {
	return ticketTokenRe.MatchString(text)
}

// parseTeamSpawn recognizes a claude-team worker spawn in a Bash command,
// returning the model (required — a spawn with no --model carries no
// routing evidence and is not a dispatch record), its effort, and the
// worker handle later prompt commands address.
func parseTeamSpawn(command string) (teamSpawn, bool) {
	m, ok := parseTeamSpawnSegment(command)
	return m.spawn, ok
}

// parseTeamSpawnSegment returns the exact command segment recognized as the
// spawn. Brief references must be read from this segment alone (D31).
type teamCandidate struct {
	text  string
	start int
	end   int
}

type teamSpawnMatch struct {
	spawn     teamSpawn
	candidate teamCandidate
}

func parseTeamSpawnSegment(command string) (teamSpawnMatch, bool) {
	for _, candidate := range allTeamCandidates(command) {
		if s, ok := herdrStart(candidate.text); ok {
			return teamSpawnMatch{spawn: s, candidate: candidate}, true
		}
		if s, ok := cmuxClaudeSend(candidate.text); ok {
			return teamSpawnMatch{spawn: s, candidate: candidate}, true
		}
	}
	return teamSpawnMatch{}, false
}

// parseTeamPrompt recognizes a follow-up prompt command addressed to an
// already-started worker: `herdr agent prompt <name> …`, or a `cmux send`
// that is not itself a spawn.
func parseTeamPrompt(command string) (teamPrompt, bool) {
	return parseTeamPromptAfter(command, -1)
}

// parseTeamPromptAfter finds the first prompt after a recognized spawn segment.
// A same-Bash `/exit` before a replacement start is cleanup for the old worker,
// never the replacement's assignment.
func parseTeamPromptAfter(command string, after int) (teamPrompt, bool) {
	for _, candidate := range allTeamCandidates(command) {
		if candidate.start <= after {
			continue
		}
		f := segmentFields(candidate.text)
		switch {
		case isTool(f, "herdr", "agent", "prompt"):
			return teamPrompt{target: positional(f, 3), text: candidate.text}, true
		case isTool(f, "cmux", "send"):
			if _, isSpawn := cmuxClaudeSend(candidate.text); isSpawn {
				continue
			}
			return teamPrompt{target: flagValue(f, "--surface", "--pane", "--target"), text: candidate.text}, true
		}
	}
	return teamPrompt{}, false
}

// teamCommandCandidates recognizes a command at the outer segment position or
// at the command position inside `$(...)`. The latter is a real shell command
// (for example `R=$(herdr agent prompt ...)`), unlike prose. Nested `$(cat)`
// stays inside the candidate and is resolved only by referencedBriefPath.
func teamCommandCandidates(seg string) []teamCandidate {
	candidates := []teamCandidate{{text: seg, end: len(seg)}}
	inSingleQuote := false
	for i := 0; i+1 < len(seg); i++ {
		if seg[i] == '\'' {
			inSingleQuote = !inSingleQuote
			continue
		}
		if inSingleQuote {
			continue
		}
		if seg[i] != '$' || seg[i+1] != '(' {
			continue
		}
		inner, end, ok := balancedSubstitution(seg, i)
		if !ok {
			continue // malformed substitutions under-attribute
		}
		trimmed := strings.TrimLeft(inner, " \t")
		start := i + 2 + len(inner) - len(trimmed)
		f := segmentFields(trimmed)
		if isTool(f, "herdr", "agent", "start") || isTool(f, "herdr", "agent", "prompt") || isTool(f, "cmux", "send") {
			candidates = append(candidates, teamCandidate{text: trimmed, start: start, end: i + 2 + end})
		}
		i = end
	}
	return candidates
}

func allTeamCandidates(command string) []teamCandidate {
	var out []teamCandidate
	for _, seg := range commandSegmentsWithOffsets(command) {
		for _, c := range teamCommandCandidates(seg.text) {
			c.start += seg.start
			c.end += seg.start
			if c.text == seg.text {
				out = append(out, c)
			} else {
				for _, inner := range commandSegmentsWithOffsets(c.text) {
					out = append(out, teamCandidate{text: inner.text, start: c.start + inner.start, end: c.start + inner.start + len(inner.text)})
				}
			}
		}
	}
	return out
}

func balancedSubstitution(s string, start int) (string, int, bool) {
	depth := 1
	quote := byte(0)
	for i := start + 2; i < len(s); i++ {
		if s[i] == '\'' || s[i] == '"' {
			if quote == 0 {
				quote = s[i]
			} else if quote == s[i] {
				quote = 0
			}
			continue
		}
		if quote != 0 {
			continue
		}
		if i+1 < len(s) && s[i] == '$' && s[i+1] == '(' {
			depth++
			i++
			continue
		}
		if s[i] == ')' {
			depth--
			if depth == 0 {
				return s[start+2 : i], i, true
			}
		}
	}
	return "", 0, false
}

// recognizeTeamSpawns gates I090's Bash-dispatch recognition. It is a var
// for exactly one reason: the package's negative control flips it off to
// prove the recognition is load-bearing — that without it the team-built
// positive-control fixture falls back to no-transcript, the very verdict
// I090 exists to remove. Nothing in production ever writes it.
var recognizeTeamSpawns = true

// recognizeBriefFiles gates I101's transcript-backed brief recovery. Its
// negative control proves this new evidence path, rather than incidental raw
// command text, is what attributes the brief fixture. Production never writes
// it.
var recognizeBriefFiles = true

// briefTable is one claude session's record of the brief bodies its lead wrote.
// It deliberately holds transcript evidence only: callers supply command text,
// cwd, and transcript position, so resolving a brief cannot open a path or run
// a shell command.
type briefTable struct {
	assignments []briefAssignment
	writes      []brief
}

type briefAssignment struct {
	name     string
	value    string
	position int
}

type brief struct {
	path     string
	body     string
	position int
}

func newBriefTable() *briefTable { return &briefTable{} }

// recordCommand saves assignments and heredoc writes found in one transcript
// command. Assignments are recorded first so a command that sets WS immediately
// before writing $WS/brief.md is resolved in the same way Bash sees it.
func (t *briefTable) recordCommand(command, cwd string, position int) {
	for _, a := range commandAssignments(command) {
		t.assignments = append(t.assignments, briefAssignment{
			name:     a.name,
			value:    a.value,
			position: position,
		})
	}
	_, writes := scanHeredocs(command)
	for _, w := range writes {
		t.recordWrite(w, cwd, position)
	}
}

// recordCommandOrdered assigns every assignment and heredoc write an offset
// within one Bash block. A spawn cutoff can therefore exclude text that
// appears later in that same block (D32), rather than merely later JSONL
// events.
func (t *briefTable) recordCommandOrdered(command, cwd string, base int) {
	stripped, writes := scanHeredocs(command)
	for _, m := range assignmentRe.FindAllStringSubmatchIndex(stripped, -1) {
		t.assignments = append(t.assignments, briefAssignment{
			name:     stripped[m[2]:m[3]],
			value:    strings.Trim(stripped[m[4]:m[5]], `"'`),
			position: base + m[0],
		})
	}
	search := 0
	for _, w := range writes {
		offset := strings.Index(stripped[search:], w.header)
		if offset < 0 {
			continue
		}
		search += offset + len(w.header)
		t.recordWrite(w, cwd, base+search-len(w.header))
	}
}

func (t *briefTable) recordWrite(w heredocWrite, cwd string, position int) {
	p, ok := t.normalize(w.target, cwd, position)
	if !ok {
		return
	}
	body := w.body
	if w.append {
		for i := len(t.writes) - 1; i >= 0; i-- {
			previous := t.writes[i]
			if previous.path == p && previous.position <= position {
				body = previous.body + body
				break
			}
		}
	}
	t.writes = append(t.writes, brief{path: p, body: body, position: position})
}

// resolve returns the last recorded write of ref available at position. A path
// retaining an unresolved variable has no entry by design; guessing would turn
// missing evidence into an attribution.
func (t *briefTable) resolve(ref, cwd string, position int) (brief, bool) {
	p, ok := t.normalize(ref, cwd, position)
	if !ok {
		return brief{}, false
	}
	for i := len(t.writes) - 1; i >= 0; i-- {
		w := t.writes[i]
		if w.path == p && w.position <= position {
			return w, true
		}
	}
	return brief{}, false
}

func (t *briefTable) normalize(raw, cwd string, position int) (string, bool) {
	expanded, ok := expandBriefVariables(raw, t.assignments, position)
	if !ok || expanded == "" {
		return "", false
	}
	if !path.IsAbs(expanded) {
		expanded = path.Join(cwd, expanded)
	}
	return path.Clean(expanded), true
}

type commandAssignment struct{ name, value string }

var assignmentRe = regexp.MustCompile(`(?:^|[\n;]\s*)([A-Za-z_][A-Za-z0-9_]*)=([^\s;]+)`)

func commandAssignments(command string) []commandAssignment {
	var assignments []commandAssignment
	for _, m := range assignmentRe.FindAllStringSubmatch(command, -1) {
		assignments = append(assignments, commandAssignment{
			name:  m[1],
			value: strings.Trim(m[2], `"'`),
		})
	}
	return assignments
}

var briefVariableRe = regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)|\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

func expandBriefVariables(raw string, assignments []briefAssignment, position int) (string, bool) {
	ok := true
	expanded := briefVariableRe.ReplaceAllStringFunc(raw, func(match string) string {
		m := briefVariableRe.FindStringSubmatch(match)
		name := m[1]
		if name == "" {
			name = m[2]
		}
		for i := len(assignments) - 1; i >= 0; i-- {
			a := assignments[i]
			if a.name == name && a.position <= position {
				return a.value
			}
		}
		ok = false
		return match
	})
	return expanded, ok
}

// referencedBriefPath recognizes the three narrow brief delivery forms. It
// returns a textual path only; resolving it remains the table's job.
func referencedBriefPath(command string) (string, bool) {
	stripped := stripHeredocBodies(command)
	if m := catReferenceRe.FindStringSubmatch(stripped); m != nil {
		return strings.Trim(m[1], `"'`), true
	}
	fields := segmentFields(stripped)
	if p := flagValue(fields, "--brief"); p != "" {
		return p, true
	}
	for _, field := range fields {
		field = strings.Trim(field, `"'`)
		if strings.HasSuffix(field, ".md") {
			return field, true
		}
	}
	return "", false
}

var catReferenceRe = regexp.MustCompile(`\$\(cat\s+([^\s)]+)\)`)

// attributeTeamPrompt gives a spawn that named no ticket the text of the
// following prompt command addressed to the same worker, which is where a
// claude-team lead names the ticket. The most recent unattributed spawn for
// that worker wins, and only the first prompt after it counts: a later
// prompt in the same worker's session is follow-up conversation, not the
// assignment. A spawn command that already names a ticket keeps its own
// attribution — borrowing would let one worker's second assignment reattach
// its first ticket's evidence.
func attributeTeamPrompt(dispatches []dispatch, p teamPrompt) {
	attributeTeamPromptWithBriefs(dispatches, p, nil)
}

// attributeTeamPromptWithBriefs resolves a paired prompt's brief against the
// matched spawn's cutoff, never the later prompt position (D32).
func attributeTeamPromptWithBriefs(dispatches []dispatch, p teamPrompt, briefs *briefTable) {
	if p.target == "" {
		return
	}
	for i := len(dispatches) - 1; i >= 0; i-- {
		d := &dispatches[i]
		if d.teamTarget != p.target {
			continue
		}
		if d.prompt == "" && d.briefText == "" && !namesATicket(d.description) {
			d.prompt = p.text
			if briefs != nil && p.briefRef != "" {
				if resolved, found := briefs.resolve(p.briefRef, d.cwd, d.briefCutoff); found {
					p.briefText = resolved.body
					p.briefPath = resolved.path
				}
			}
			d.briefText = p.briefText
			d.briefPath = p.briefPath
		}
		return
	}
}

// herdrStart recognizes `herdr agent start <name> … --model <id>`.
func herdrStart(seg string) (teamSpawn, bool) {
	f := segmentFields(seg)
	if !isTool(f, "herdr", "agent", "start") {
		return teamSpawn{}, false
	}
	model := flagValue(f, "--model")
	if model == "" {
		return teamSpawn{}, false
	}
	return teamSpawn{model: model, effort: flagValue(f, "--effort"), target: positional(f, 3)}, true
}

// cmuxClaudeSend recognizes a `cmux send` whose payload invokes claude with
// a --model. A send running anything else is not a spawn.
func cmuxClaudeSend(seg string) (teamSpawn, bool) {
	f := segmentFields(seg)
	if !isTool(f, "cmux", "send") {
		return teamSpawn{}, false
	}
	i := indexOfCommand(f, "claude")
	if i < 0 {
		return teamSpawn{}, false
	}
	model := flagValue(f[i:], "--model")
	if model == "" {
		return teamSpawn{}, false
	}
	return teamSpawn{model: model, effort: flagValue(f[i:], "--effort"), target: flagValue(f, "--surface", "--pane", "--target")}, true
}

// isTool reports whether the segment's first fields are the named tool
// (matched on the path's base, so /opt/homebrew/bin/herdr counts) followed
// by the given subcommand words in order. This is what confines recognition
// to command position: a mention anywhere later in the segment is prose.
func isTool(f []string, tool string, sub ...string) bool {
	if len(f) < 1+len(sub) || path.Base(f[0]) != tool {
		return false
	}
	for i, s := range sub {
		if f[1+i] != s {
			return false
		}
	}
	return true
}

// indexOfCommand finds the field at which name is invoked as a command —
// the first field of the segment, or the field right after a `--` payload
// separator or an opening quote. Used for the claude invocation nested
// inside a `cmux send` payload, which takes both forms in practice.
func indexOfCommand(f []string, name string) int {
	for i, w := range f {
		if path.Base(w) != name {
			continue
		}
		if i == 0 || f[i-1] == "--" || isQuote(f[i-1]) {
			return i
		}
	}
	return -1
}

// positional returns field i when it exists and is not a flag.
func positional(f []string, i int) string {
	if i >= len(f) || strings.HasPrefix(f[i], "-") || isQuote(f[i]) {
		return ""
	}
	return f[i]
}

// flagValue returns the value of the first of names present, in either
// `--flag value` or `--flag=value` form. A quote field between flag and
// value (`--model 'x'`) is stepped over — quotes survive tokenization only
// to mark where a nested payload starts, never as values.
func flagValue(f []string, names ...string) string {
	for i, w := range f {
		for _, n := range names {
			if w == n {
				if j := i + 1; j < len(f) && isQuote(f[j]) {
					j++
					if j < len(f) && !isQuote(f[j]) && !strings.HasPrefix(f[j], "-") {
						return f[j]
					}
				} else if j < len(f) && !strings.HasPrefix(f[j], "-") {
					return f[j]
				}
			}
			if v, ok := strings.CutPrefix(w, n+"="); ok && v != "" {
				return strings.Trim(v, `"'`)
			}
		}
	}
	return ""
}

func isQuote(w string) bool { return w == "'" || w == `"` }

// segmentFields splits a command segment into shell-ish words. Quote
// characters become fields of their own rather than being stripped: the
// opening quote of a `cmux send` payload is exactly the marker that the
// claude invocation inside it stands in command position.
func segmentFields(seg string) []string {
	seg = strings.NewReplacer("'", " ' ", `"`, ` " `).Replace(seg)
	return strings.Fields(seg)
}

// commandSegments splits a Bash command into the segments that each start
// with a command word, after removing heredoc bodies. Both steps exist to
// keep prose out: text written into a file through a heredoc, and a command
// name quoted mid-sentence, must not read as a dispatch.
func commandSegments(command string) []string {
	withOffsets := commandSegmentsWithOffsets(command)
	segs := make([]string, len(withOffsets))
	for i, seg := range withOffsets {
		segs[i] = seg.text
	}
	return segs
}

type commandSegment struct {
	text  string
	start int
}

func commandSegmentsWithOffsets(command string) []commandSegment {
	stripped := stripHeredocBodies(command)
	var segs []commandSegment
	lineOffset := 0
	for _, line := range strings.Split(stripped, "\n") {
		// A segment that only exists because a shell metacharacter appeared
		// AFTER a '#' is prose inside a trailing comment, not a command:
		// `cat notes # (herdr agent start … --model …)` would otherwise
		// yield a segment beginning with `herdr`. The comment itself is not
		// stripped — a spawn command's trailing `# I402` is real ticket
		// attribution, and it lives in the line's FIRST segment, which
		// starts at 0 and is always kept. A '#' inside a quoted payload
		// likewise suppresses only later segments on that line: a false
		// negative (the ticket degrades to no-transcript), never a false
		// dispatch.
		comment := -1
		inSingleQuote := false
		inDoubleQuote := false
		substitutionDepth := 0
		escaped := false
		start := 0
		keep := func(seg string, at int) {
			if comment < 0 || at <= comment {
				segs = append(segs, commandSegment{text: seg, start: lineOffset + at})
			}
		}
		for i := 0; i < len(line); i++ {
			if escaped {
				escaped = false
				continue
			}
			if line[i] == '\\' && !inSingleQuote {
				escaped = true
				continue
			}
			if line[i] == '\'' && !inDoubleQuote {
				inSingleQuote = !inSingleQuote
				continue
			}
			if line[i] == '"' && !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
				continue
			}
			if inSingleQuote {
				continue
			}
			if i+1 < len(line) && line[i] == '$' && line[i+1] == '(' {
				substitutionDepth++
				i++
				continue
			}
			if line[i] == ')' && substitutionDepth > 0 {
				substitutionDepth--
				continue
			}
			if substitutionDepth > 0 || inDoubleQuote {
				continue
			}
			if line[i] == '#' {
				comment = i
				break
			}
			switch line[i] {
			case ';', '|', '&', '{', '}':
				keep(line[start:i], start)
				start = i + 1
			case '(', ')':
				keep(line[start:i], start)
				start = i + 1
			}
		}
		keep(line[start:], start)
		lineOffset += len(line) + 1
	}
	return segs
}

// heredocRe matches a heredoc redirection's delimiter word, quoted or not.
var heredocRe = regexp.MustCompile(`<<-?\s*['"]?([A-Za-z_][A-Za-z0-9_]*)['"]?`)

type heredocWrite struct {
	target string
	body   string
	append bool
	header string
}

var catHeredocRe = regexp.MustCompile(`(?:^|[;&|]\s*)cat\s+(>>|>)\s+([^\s]+)\s+<<-?\s*['"]?[A-Za-z_][A-Za-z0-9_]*['"]?`)

// stripHeredocBodies removes the body of every heredoc in a command — the
// shape a doc file being written takes inside a Bash block. Without this,
// a plan or handoff that documents `herdr agent start … --model …` would be
// read as having dispatched it.
func stripHeredocBodies(command string) string {
	stripped, _ := scanHeredocs(command)
	return stripped
}

// scanHeredocs keeps the old stripped-command contract while exposing the
// target and body of transcript-recorded cat writes for brief attribution.
func scanHeredocs(command string) (string, []heredocWrite) {
	lines := strings.Split(command, "\n")
	var out []string
	var writes []heredocWrite
	for i := 0; i < len(lines); i++ {
		out = append(out, lines[i])
		m := heredocRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		delim := m[1]
		bodyStart := i + 1
		for i+1 < len(lines) && strings.TrimSpace(lines[i+1]) != delim {
			i++
		}
		if i+1 < len(lines) {
			if write := catHeredocRe.FindStringSubmatch(lines[bodyStart-1]); write != nil {
				body := strings.Join(lines[bodyStart:i+1], "\n")
				if body != "" {
					body += "\n"
				}
				writes = append(writes, heredocWrite{
					target: write[2],
					body:   body,
					append: write[1] == ">>",
					header: lines[bodyStart-1],
				})
			}
			i++ // consume the terminator itself
		}
	}
	return strings.Join(out, "\n"), writes
}
