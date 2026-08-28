// Package cursor parses the spine stage cursor: a machine-readable block at
// the head of a repo's .superpowers/sdd/progress.md recording which
// WORKFLOW.md stage an effort is at.
//
// Grammar (defined once here; the gen 9 WORKFLOW.md template section reuses
// this text verbatim — see Grammar):
//
//	<!-- spine:cursor -->
//	effort: <kebab-name>
//	prd: docs/specs/<file>.md
//	tickets: I0NN | I0NN,I0MM[,...] | I0NN-I0MM | prefix I0
//	stages: grill[x] prd[x] issues[x] implement[<] functional-test[ ] review[ ] verify[ ] ship[ ] ...
//	<!-- /spine:cursor -->
//
// `[x]` marks a done stage, `[<]` marks YOU ARE HERE (exactly one, among the
// non-done stages), `[ ]` marks pending. Stage names must match the repo's
// WORKFLOW.md `stages:` list.
//
// tickets: resolves four forms (I026, I114; resolution itself lives in
// internal/stages' resolveTicketIDs, which anchors evidence against this
// grammar): a bare single-ticket id ("I0NN"), a comma-list of distinct bare
// ids that preserves input order ("I0NN,I0MM[,...]"), an inclusive numeric
// range of equal digit width ("I0NN-I0MM", where a same-endpoint range like
// "I001-I001" is a valid — if redundant — alias for the bare-id form), or
// every docs/issues ticket id sharing a literal prefix ("prefix <str>").
// Anything else is unresolvable: internal/stages degrades the issues and
// implement evidence rules to not-judged (absence of evidence never blocks)
// but surfaces a Notes entry naming the bad value, so the degradation is
// visible rather than silent.
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
//   - Zero YOU-ARE-HERE markers is only flagged when some stage is still
//     pending: an effort with every stage done legitimately has none, but a
//     cursor with a pending stage and no [<] violates the grammar's
//     "exactly one among non-done" rule just as multiple markers do.
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
	"strconv"
	"strings"

	"github.com/russellpope/spine/internal/fsutil"
	"github.com/russellpope/spine/internal/update"
)

// Grammar is the canonical cursor block text, documented once here and
// reused verbatim by the gen 9 WORKFLOW.md template section (I020, I026,
// I114).
// It is illustrative documentation, never a live cursor: the trailing "..."
// on the stages: line is a "more stages may follow" marker, not a real
// stage token, and would not itself parse.
const Grammar = `<!-- spine:cursor -->
effort: <kebab-name>
prd: docs/specs/<file>.md
tickets: I0NN | I0NN,I0MM[,...] | I0NN-I0MM | prefix I0
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

// Block renders c as the canonical stage-cursor block. It deliberately does
// not include a trailing newline: callers replacing an existing block retain
// the surrounding ledger bytes, while callers creating a ledger choose its
// file-level newline convention.
func (c Cursor) Block() string {
	return openTag + "\n" +
		"effort: " + strings.TrimSpace(c.Effort) + "\n" +
		"prd: " + strings.TrimSpace(c.PRD) + "\n" +
		"tickets: " + strings.TrimSpace(c.Tickets) + "\n" +
		"stages: " + c.StagesLine() + "\n" +
		closeTag
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
	// NonCanonical reports a grammatically valid cursor block whose original
	// bytes differ from Cursor.Block(). It is deliberately separate from
	// Findings: formatting drift is a sole-writer violation, not a grammar
	// violation, so audit and doctor can report the two conditions distinctly.
	NonCanonical bool
}

const (
	ledgerRel = ".superpowers/sdd/progress.md"
	openTag   = "<!-- spine:cursor -->"
	closeTag  = "<!-- /spine:cursor -->"
	// NonCanonicalRemediation is shared by the audit gate and doctor advisory.
	// Every cursor write verb rewrites the block, while a flagless `cursor set`
	// is the explicit no-op recovery path.
	NonCanonicalRemediation = "cursor block is non-canonical — run any `spine cursor` write verb (or no-op `spine cursor set`) to rewrite it canonically"
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

// New returns an all-pending cursor using the stage order declared by dir's
// WORKFLOW.md. Its first stage is current. Empty prd and tickets values are
// intentional: they are legal early-effort cursor values.
func New(dir, effort, prd, tickets string) (Cursor, error) {
	wfRaw, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		return Cursor{}, err
	}
	names := parseStagesList(update.ExtractKeys(string(wfRaw))["stages"])
	if len(names) == 0 {
		return Cursor{}, fmt.Errorf("WORKFLOW.md has no usable stages: list")
	}
	c := Cursor{Effort: effort, PRD: prd, Tickets: tickets, Stages: make([]Stage, len(names))}
	for i, name := range names {
		state := Pending
		if i == 0 {
			state = Here
		}
		c.Stages[i] = Stage{Name: name, State: state}
	}
	return c, nil
}

// Save replaces the first cursor block in dir's working ledger with c's
// canonical serialization. When no block exists, it creates the ledger (or
// prepends the block to an existing non-cursor ledger), which is the start
// verb's seed behavior. It writes only after the complete replacement text is
// ready, so a partially serialized block is never emitted.
func Save(dir string, c Cursor) error {
	for name, value := range map[string]string{
		"effort":  c.Effort,
		"prd":     c.PRD,
		"tickets": c.Tickets,
	} {
		if strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("cursor %s value contains a newline", name)
		}
	}
	path := filepath.Join(dir, filepath.FromSlash(ledgerRel))
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	content := string(raw)
	block := c.Block()
	open, closeF, findings, ok := locate(content)
	switch {
	case ok:
		content = content[:open.start] + block + content[closeF.end:]
	case len(findings) > 0:
		// The replacement spans open-through-close, so against an ambiguous
		// ledger it would destroy everything between two candidate fences.
		// Refuse instead (I109 D8). The ledger is machine-owned, so this
		// should be unreachable — which is exactly when a loud failure is
		// wanted: it means a hand edit or corruption, and a destructive
		// rewrite is the worst available response.
		return fmt.Errorf("refusing to rewrite %s: %s", ledgerRel, strings.Join(findings, "; "))
	case content == "":
		content = block + "\n"
	default:
		content = block + "\n\n" + content
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, []byte(content))
}

// fence is one line-anchored cursor delimiter found in a document.
type fence struct {
	line    int // 1-based line number, for findings the operator can act on
	start   int // byte offset of the line's first byte
	end     int // byte offset just past the tag text
	lineEnd int // byte offset just past line content, excluding its terminator
}

// fenceRemedy is appended to duplicate-fence findings. The escape hatch ships
// as a clause in the error rather than as code-block tracking (I109 D6): an
// indented example is not a fence, so a document can show a whole block.
const fenceRemedy = "; to show a fence in a document, indent the example instead of fencing it"

// scanFences returns every line-anchored open and close fence in content, in
// document order.
//
// A tag counts only when it is the entire line, starting at column 0; trailing
// whitespace is tolerated because it is invisible and cannot be a deliberate
// authoring gesture, while leading whitespace is one and means "this is an
// example". A tag occurring mid-sentence or indented is prose and is skipped.
//
// This is what lets a document explain the cursor convention without hijacking
// its own parse (I109). The previous scan matched the tag anywhere, so a
// handoff quoting the opening tag in prose opened the block at the quote and
// ran to the real closing tag, swallowing the genuine opening tag as a body
// line.
func scanFences(content string) (opens, closes []fence) {
	for off, line := 0, 1; ; line++ {
		nl := strings.IndexByte(content[off:], '\n')
		text := content[off:]
		if nl >= 0 {
			text = content[off : off+nl]
		}
		lineEnd := off + len(text)
		if nl >= 0 && strings.HasSuffix(text, "\r") {
			lineEnd--
		}
		// "\r" is trimmed as a line ending, not as authored content: on a CRLF
		// document the fence line is "<tag>\r", and the substring scan this
		// replaced matched it. Dropping that would be a regression rather than
		// a tightening.
		switch strings.TrimRight(text, " \t\r") {
		case openTag:
			opens = append(opens, fence{line: line, start: off, end: off + len(openTag), lineEnd: lineEnd})
		case closeTag:
			closes = append(closes, fence{line: line, start: off, end: off + len(closeTag), lineEnd: lineEnd})
		}
		if nl < 0 {
			return opens, closes
		}
		off += nl + 1
	}
}

// fenceLines renders fence line numbers as a findings-ready list.
func fenceLines(fs []fence) string {
	parts := make([]string, len(fs))
	for i, f := range fs {
		parts[i] = strconv.Itoa(f.line)
	}
	return strings.Join(parts, ", ")
}

// locate resolves content's single fenced region. ok is false when there is
// nothing to parse: either no fences at all (the quiet case — findings nil) or
// a fence problem that makes any parse a guess (findings non-empty).
//
// Fence rules are symmetric: exactly one open, exactly one close, close after
// open. Refusing to parse an ambiguous document is deliberate — silently
// choosing one of two candidate blocks is the failure this replaces.
func locate(content string) (open, closeF fence, findings []string, ok bool) {
	opens, closes := scanFences(content)
	if len(opens) == 0 && len(closes) == 0 {
		return fence{}, fence{}, nil, false // nothing to derive against
	}
	if len(opens) > 1 {
		findings = append(findings, fmt.Sprintf(
			"%d opening `%s` fences (lines %s) — a document must contain exactly one%s",
			len(opens), openTag, fenceLines(opens), fenceRemedy))
	}
	if len(closes) > 1 {
		findings = append(findings, fmt.Sprintf(
			"%d closing `%s` fences (lines %s) — a document must contain exactly one%s",
			len(closes), closeTag, fenceLines(closes), fenceRemedy))
	}
	switch {
	case len(findings) > 0:
		return fence{}, fence{}, findings, false
	case len(opens) == 0:
		return fence{}, fence{}, []string{fmt.Sprintf(
			"closing `%s` fence at line %d with no opening fence", closeTag, closes[0].line)}, false
	case len(closes) == 0:
		return fence{}, fence{}, []string{fmt.Sprintf(
			"cursor block opened at line %d is missing its closing `%s` fence",
			opens[0].line, closeTag)}, false
	case closes[0].line < opens[0].line:
		return fence{}, fence{}, []string{fmt.Sprintf(
			"closing `%s` fence at line %d precedes the opening fence at line %d",
			closeTag, closes[0].line, opens[0].line)}, false
	}
	return opens[0], closes[0], nil, true
}

// parse extracts and parses the fenced cursor region in content.
func parse(content string, validStages []string) Result {
	open, closeF, findings, ok := locate(content)
	if !ok {
		if len(findings) == 0 {
			return Result{} // no cursor block present
		}
		return Result{HasCursor: true, Findings: findings}
	}
	bodyStart := open.end
	if nl := strings.IndexByte(content[bodyStart:], '\n'); nl >= 0 {
		bodyStart += nl + 1
	}
	c, findings := parseBody(content[bodyStart:closeF.start], validStages, open.line+1)
	return Result{
		Cursor:       c,
		HasCursor:    true,
		Findings:     findings,
		NonCanonical: len(findings) == 0 && content[open.start:closeF.lineEnd] != c.Block(),
	}
}

// parseBody parses the key: value lines between the cursor fences. baseLine is
// the whole-document 1-based line number of the body's first line, so findings
// that quote an offending line can name where it is (I109 D4) — the operator
// has the file open in an editor, and a body-relative number restarts the hunt.
//
// Findings about the block as a whole (a missing required key) carry no line
// number: there is no offending line to point at, and the key names the fix.
func parseBody(body string, validStages []string, baseLine int) (Cursor, []string) {
	var findings []string
	values := map[string]string{}
	keyLines := map[string]int{}
	for i, line := range strings.Split(body, "\n") {
		lineNo := baseLine + i
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			findings = append(findings, fmt.Sprintf("line %d: unrecognized line in cursor block: %q", lineNo, line))
			continue
		}
		if !isRequiredKey(key) {
			findings = append(findings, fmt.Sprintf("line %d: unknown key %q in cursor block", lineNo, key))
			continue
		}
		if _, dup := values[key]; dup {
			findings = append(findings, fmt.Sprintf("line %d: duplicate key %q in cursor block", lineNo, key))
			continue
		}
		values[key] = strings.TrimSpace(val)
		keyLines[key] = lineNo
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
		for _, f := range stageFindings {
			findings = append(findings, fmt.Sprintf("line %d: %s", keyLines["stages"], f))
		}
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
// more than one YOU-ARE-HERE marker, and zero HERE markers while a stage is
// still pending.
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
	pendingCount := 0
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
			pendingCount++
		}
		if len(validSet) > 0 && !validSet[name] {
			findings = append(findings, fmt.Sprintf("unknown stage name %q (not in WORKFLOW.md stages:)", name))
		}
		stages = append(stages, Stage{Name: name, State: state})
	}
	if hereCount > 1 {
		findings = append(findings, fmt.Sprintf("multiple YOU-ARE-HERE ([<]) markers: found %d, want exactly 1 among non-done stages", hereCount))
	}
	if hereCount == 0 && pendingCount > 0 {
		findings = append(findings, "missing YOU-ARE-HERE ([<]) marker: a non-done stage is pending but no stage is marked [<]")
	}
	return stages, findings
}

// HasBlock reports whether content carries any line-anchored cursor fence.
// Exported so callers outside this package (the derivation engine's
// newest-handoff check, I019) can test for an embedded cursor block without
// duplicating the tag literal.
//
// It shares scanFences with parse deliberately (I109 D2). As a bare substring
// test it disagreed with an anchored parse: a prose mention made it report a
// block while the parse found none, yielding an empty Result with no findings
// — which the derivation engine then misreported as a stale effort. A stray
// close fence counts too, so "a block exists" and "there is something to
// report" stay the same question.
func HasBlock(content string) bool {
	opens, closes := scanFences(content)
	return len(opens) > 0 || len(closes) > 0
}

// ParseBlockResult parses the first spine:cursor block in a handoff document
// and retains its grammar findings. A handoff has no WORKFLOW.md stages list,
// so stage-name validation does not apply.
func ParseBlockResult(content string) Result {
	return parse(content, nil)
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
