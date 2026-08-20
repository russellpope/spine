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
//	cmux send --pane <id> '… claude … --model <id> [--effort <e>]'
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
	target string
	text   string
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
	for _, seg := range commandSegments(command) {
		if s, ok := herdrStart(seg); ok {
			return s, true
		}
		if s, ok := cmuxClaudeSend(seg); ok {
			return s, true
		}
	}
	return teamSpawn{}, false
}

// parseTeamPrompt recognizes a follow-up prompt command addressed to an
// already-started worker: `herdr agent prompt <name> …`, or a `cmux send`
// that is not itself a spawn.
func parseTeamPrompt(command string) (teamPrompt, bool) {
	for _, seg := range commandSegments(command) {
		f := segmentFields(seg)
		switch {
		case isTool(f, "herdr", "agent", "prompt"):
			return teamPrompt{target: positional(f, 3), text: seg}, true
		case isTool(f, "cmux", "send"):
			if _, isSpawn := cmuxClaudeSend(seg); isSpawn {
				continue
			}
			return teamPrompt{target: flagValue(f, "--pane", "--target"), text: seg}, true
		}
	}
	return teamPrompt{}, false
}

// recognizeTeamSpawns gates I090's Bash-dispatch recognition. It is a var
// for exactly one reason: the package's negative control flips it off to
// prove the recognition is load-bearing — that without it the team-built
// positive-control fixture falls back to no-transcript, the very verdict
// I090 exists to remove. Nothing in production ever writes it.
var recognizeTeamSpawns = true

// attributeTeamPrompt gives a spawn that named no ticket the text of the
// following prompt command addressed to the same worker, which is where a
// claude-team lead names the ticket. The most recent unattributed spawn for
// that worker wins, and only the first prompt after it counts: a later
// prompt in the same worker's session is follow-up conversation, not the
// assignment. A spawn command that already names a ticket keeps its own
// attribution — borrowing would let one worker's second assignment reattach
// its first ticket's evidence.
func attributeTeamPrompt(dispatches []dispatch, p teamPrompt) {
	if p.target == "" {
		return
	}
	for i := len(dispatches) - 1; i >= 0; i-- {
		d := &dispatches[i]
		if d.teamTarget != p.target {
			continue
		}
		if d.prompt == "" && !namesATicket(d.description) {
			d.prompt = p.text
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
	return teamSpawn{model: model, effort: flagValue(f[i:], "--effort"), target: flagValue(f, "--pane", "--target")}, true
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
	var segs []string
	for _, line := range strings.Split(stripHeredocBodies(command), "\n") {
		start := 0
		for i := 0; i < len(line); i++ {
			switch line[i] {
			case ';', '|', '&', '(', ')', '{', '}':
				segs = append(segs, line[start:i])
				start = i + 1
			}
		}
		segs = append(segs, line[start:])
	}
	return segs
}

// heredocRe matches a heredoc redirection's delimiter word, quoted or not.
var heredocRe = regexp.MustCompile(`<<-?\s*['"]?([A-Za-z_][A-Za-z0-9_]*)['"]?`)

// stripHeredocBodies removes the body of every heredoc in a command — the
// shape a doc file being written takes inside a Bash block. Without this,
// a plan or handoff that documents `herdr agent start … --model …` would be
// read as having dispatched it.
func stripHeredocBodies(command string) string {
	lines := strings.Split(command, "\n")
	var out []string
	for i := 0; i < len(lines); i++ {
		out = append(out, lines[i])
		m := heredocRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		delim := m[1]
		for i+1 < len(lines) && strings.TrimSpace(lines[i+1]) != delim {
			i++
		}
		if i+1 < len(lines) {
			i++ // consume the terminator itself
		}
	}
	return strings.Join(out, "\n")
}
