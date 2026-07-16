// Package cursor parses the spine stage cursor: a machine-readable block at
// the head of a repo's .superpowers/sdd/progress.md recording which
// WORKFLOW.md stage an effort is at.
//
// Grammar (defined once here; the gen 8 WORKFLOW.md template section reuses
// this text verbatim — see Grammar):
//
//	<!-- spine:cursor -->
//	effort: <kebab-name>
//	prd: docs/specs/<file>.md
//	tickets: I0NN-I0MM | prefix I0
//	stages: grill[x] prd[x] issues[x] implement[<] functional-test[ ] review[ ] verify[ ] ship[ ] ...
//	<!-- /spine:cursor -->
//
// `[x]` marks a done stage, `[<]` marks YOU ARE HERE (at most one, among the
// non-done stages), `[ ]` marks pending. Stage names must match the repo's
// WORKFLOW.md `stages:` list.
//
// Load never panics on bad input: a missing repo, a missing ledger, or a
// missing cursor block is reported as HasCursor==false (nothing to derive
// against — the hook-friendly quiet case). A cursor block that fails to
// parse is reported as HasCursor==true with one or more Findings describing
// every problem found; err is reserved for genuine I/O failures (permission
// errors and the like), never for grammar violations.
//
// Design-latitude choices the ticket leaves open (pinned here):
//   - "Head" means the first `<!-- spine:cursor -->` block found in the
//     file, not necessarily line 1 — progress.md's own title/intro lines
//     precede it in the real ledger this parser must accept.
//   - Stage names are validated against WORKFLOW.md's stages: list only
//     when that list itself parses; an unparseable/missing stages: key
//     disables the unknown-stage-name check rather than blocking on it
//     (absence of evidence never blocks — matches the derivation engine's
//     stated philosophy in the design doc, even though derivation itself
//     is Task 2/I019).
//   - Zero YOU-ARE-HERE markers is not flagged: an effort with every stage
//     done legitimately has none. Only *multiple* HERE markers is a
//     contradiction the grammar itself rules out ("exactly one among
//     non-done").
//   - Required keys (effort, prd, tickets, stages) must each appear exactly
//     once; unknown keys, duplicate keys, and lines that aren't `key:
//     value` are each reported as their own finding rather than aborting
//     the whole parse — one malformed cursor should surface every problem
//     in a single pass, not just the first.
package cursor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/russellpope/spine/internal/update"
)

// Grammar is the canonical cursor block text, documented once here and
// reused verbatim by the gen 8 WORKFLOW.md template section (I020).
const Grammar = `<!-- spine:cursor -->
effort: <kebab-name>
prd: docs/specs/<file>.md
tickets: I0NN-I0MM | prefix I0
stages: grill[x] prd[x] issues[x] implement[<] functional-test[ ] review[ ] verify[ ] ship[ ] ...
<!-- /spine:cursor -->
`

// State is one stage's checklist marker.
type State int

const (
	Pending State = iota // [ ]
	Done                 // [x]
	Here                 // [<] YOU ARE HERE
)

// marker renders the bracket contents for the state ("x", "<", or " ").
func (s State) marker() string {
	switch s {
	case Done:
		return "x"
	case Here:
		return "<"
	default:
		return " "
	}
}

// Stage is one WORKFLOW.md stage name plus its checklist marker.
type Stage struct {
	Name  string
	State State
}

func (s Stage) String() string { return s.Name + "[" + s.State.marker() + "]" }

// Cursor is one parsed stage-cursor block.
type Cursor struct {
	Effort  string
	PRD     string
	Tickets string
	Stages  []Stage
}

// StagesLine re-renders Stages as the grammar's space-joined stages: value.
func (c Cursor) StagesLine() string {
	parts := make([]string, len(c.Stages))
	for i, s := range c.Stages {
		parts[i] = s.String()
	}
	return strings.Join(parts, " ")
}

// Result is what Load found.
type Result struct {
	Cursor Cursor
	// HasCursor is true iff an opening `<!-- spine:cursor -->` marker was
	// found. False means: not a spine repo, no progress.md, or a
	// progress.md with no cursor block at all — the quiet/hook-friendly
	// "nothing to show" case, never a Finding.
	HasCursor bool
	// Findings are grammar violations in a block that was found. Never a
	// panic; Cursor may be partially populated when Findings is non-empty.
	Findings []string
}

const (
	ledgerRel = ".superpowers/sdd/progress.md"
	openTag   = "<!-- spine:cursor -->"
	closeTag  = "<!-- /spine:cursor -->"
)

var requiredKeys = []string{"effort", "prd", "tickets", "stages"}

// stageTokenRe matches one `name[x]` / `name[<]` / `name[_]` token, where
// "_" stands in for the literal internal space of a pending marker "[ ]"
// (substituted before tokenizing — see parseStages).
var stageTokenRe = regexp.MustCompile(`^([a-z][a-z0-9-]*)\[([x<_])\]$`)

// Load reads dir's WORKFLOW.md (for the stages: validation list) and
// .superpowers/sdd/progress.md (for the cursor block itself) and parses the
// cursor at the head of the ledger. It never returns a non-nil error for
// grammar problems — only for I/O failures other than the file simply not
// existing.
func Load(dir string) (Result, error) {
	wfRaw, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		if os.IsNotExist(err) {
			return Result{}, nil // not a spine repo
		}
		return Result{}, err
	}
	validStages := parseStagesList(update.ExtractKeys(string(wfRaw))["stages"])

	raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(ledgerRel)))
	if err != nil {
		if os.IsNotExist(err) {
			return Result{}, nil // no ledger yet — nothing to derive against
		}
		return Result{}, err
	}
	return parse(string(raw), validStages), nil
}

// parse extracts and parses the first cursor block found in content.
func parse(content string, validStages []string) Result {
	start := strings.Index(content, openTag)
	if start == -1 {
		return Result{} // no cursor block present
	}
	rest := content[start+len(openTag):]
	endRel := strings.Index(rest, closeTag)
	if endRel == -1 {
		return Result{HasCursor: true,
			Findings: []string{"cursor block missing its closing `" + closeTag + "` marker"}}
	}
	c, findings := parseBody(rest[:endRel], validStages)
	return Result{Cursor: c, HasCursor: true, Findings: findings}
}

// parseBody parses the key: value lines between the cursor markers.
func parseBody(body string, validStages []string) (Cursor, []string) {
	var findings []string
	values := map[string]string{}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			findings = append(findings, fmt.Sprintf("unrecognized line in cursor block: %q", line))
			continue
		}
		if !isRequiredKey(key) {
			findings = append(findings, fmt.Sprintf("unknown key %q in cursor block", key))
			continue
		}
		if _, dup := values[key]; dup {
			findings = append(findings, fmt.Sprintf("duplicate key %q in cursor block", key))
			continue
		}
		values[key] = strings.TrimSpace(val)
	}
	for _, k := range requiredKeys {
		if _, ok := values[k]; !ok {
			findings = append(findings, fmt.Sprintf("cursor block missing required key %q", k))
		}
	}

	c := Cursor{Effort: values["effort"], PRD: values["prd"], Tickets: values["tickets"]}
	if raw, ok := values["stages"]; ok {
		stages, stageFindings := parseStages(raw, validStages)
		c.Stages = stages
		findings = append(findings, stageFindings...)
	}
	return c, findings
}

func isRequiredKey(key string) bool {
	for _, k := range requiredKeys {
		if k == key {
			return true
		}
	}
	return false
}

// parseStages tokenizes a stages: value into Stage entries, flagging
// malformed tokens, unknown stage names (when validStages is non-empty),
// and more than one YOU-ARE-HERE marker.
func parseStages(raw string, validStages []string) ([]Stage, []string) {
	var findings []string
	// "[ ]" (pending) contains a literal space between its brackets, which
	// would otherwise split under Fields; mask it before tokenizing.
	placeholder := strings.ReplaceAll(raw, "[ ]", "[_]")
	tokens := strings.Fields(placeholder)

	validSet := map[string]bool{}
	for _, s := range validStages {
		validSet[s] = true
	}

	var stages []Stage
	hereCount := 0
	for _, tok := range tokens {
		m := stageTokenRe.FindStringSubmatch(tok)
		if m == nil {
			findings = append(findings, fmt.Sprintf("malformed stage token %q", tok))
			continue
		}
		name := m[1]
		var state State
		switch m[2] {
		case "x":
			state = Done
		case "<":
			state = Here
			hereCount++
		default:
			state = Pending
		}
		if len(validSet) > 0 && !validSet[name] {
			findings = append(findings, fmt.Sprintf("unknown stage name %q (not in WORKFLOW.md stages:)", name))
		}
		stages = append(stages, Stage{Name: name, State: state})
	}
	if hereCount > 1 {
		findings = append(findings, fmt.Sprintf("multiple YOU-ARE-HERE ([<]) markers: found %d, want at most 1", hereCount))
	}
	return stages, findings
}

// parseStagesList parses WORKFLOW.md's "stages" value (from
// update.ExtractKeys, e.g. "[grill, prd, issues]") into a name list. Returns
// nil when raw is empty or unparseable, which disables unknown-stage-name
// validation rather than blocking on it.
func parseStagesList(raw string) []string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "[")
	raw = strings.TrimSuffix(raw, "]")
	if raw == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
