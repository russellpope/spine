package cursor_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/cursor"
)

// I109. The cursor block is delimited by line-anchored fences: the open and
// close tags count only when they are the entire line, starting at column 0.
// A tag mentioned mid-sentence, or indented as a worked example, is prose.
//
// These tests drive the package's public entry points (Load, ParseBlockResult,
// HasBlock, Save) rather than the scanner itself, so they survive the scanner
// being rewritten.

const (
	openFence  = "<!-- spine:cursor -->"
	closeFence = "<!-- /spine:cursor -->"
)

// body returns the four canonical key lines of a cursor block.
func body() string {
	return "effort: x\n" +
		"prd: docs/specs/x.md\n" +
		"tickets: I001\n" +
		"stages: grill[<] prd[ ] issues[ ] implement[ ]\n"
}

// block returns a complete canonical fenced region.
func block() string {
	return openFence + "\n" + body() + closeFence
}

func findingsText(res cursor.Result) string {
	return strings.Join(res.Findings, " | ")
}

// The reported failure: a handoff that quotes the open fence mid-sentence
// above the real block hijacked the parse, because the scan matched the tag
// anywhere. Mid-line occurrences are prose and must be skipped outright.
func TestProseFenceMentionIsSkipped(t *testing.T) {
	content := "# Handoff\n" +
		"Note the gate requires the literal `" + openFence + "` marker block — a\n" +
		"hand-written one will not do.\n\n" +
		block() + "\n"

	res := cursor.ParseBlockResult(content)

	if !res.HasCursor {
		t.Fatal("want HasCursor true — a real fenced region follows the prose mention")
	}
	if len(res.Findings) != 0 {
		t.Fatalf("want no findings — the prose mention is not a fence, got %#v", res.Findings)
	}
	if res.Cursor.Effort != "x" {
		t.Errorf("Effort = %q, want the real block's value", res.Cursor.Effort)
	}
}

// The escape hatch the design ships in place of code-fence awareness: an
// indented worked example is not a fence, so a document may show a complete
// block by indenting it.
func TestIndentedFenceExampleIsSkipped(t *testing.T) {
	content := "# Workflow\n\nGrammar reference:\n\n" +
		"    " + openFence + "\n" +
		"    effort: <kebab-name>\n" +
		"    " + closeFence + "\n\n" +
		block() + "\n"

	res := cursor.ParseBlockResult(content)

	if !res.HasCursor {
		t.Fatal("want HasCursor true — the real fenced region follows the indented example")
	}
	if len(res.Findings) != 0 {
		t.Fatalf("want no findings — an indented example is not a fence, got %#v", res.Findings)
	}
	if res.Cursor.Effort != "x" {
		t.Errorf("Effort = %q, want the real block's value", res.Cursor.Effort)
	}
}

// A trailing-whitespace fence line is still a fence — trailing space is
// invisible and cannot be a deliberate authoring gesture. It is drift from
// canonical form, so it must surface as NonCanonical rather than as a finding.
func TestTrailingWhitespaceFenceIsRecognizedButNonCanonical(t *testing.T) {
	content := openFence + "   \n" + body() + closeFence + "\n"

	res := cursor.ParseBlockResult(content)

	if !res.HasCursor || len(res.Findings) != 0 {
		t.Fatalf("want a clean parse, got HasCursor=%v findings=%#v", res.HasCursor, res.Findings)
	}
	if !res.NonCanonical {
		t.Error("want NonCanonical true — a padded fence line is not the canonical serialization")
	}
}

func TestTrailingSpacesOnClosingFenceAreNonCanonical(t *testing.T) {
	content := openFence + "\n" + body() + closeFence + "   \n"

	res := cursor.ParseBlockResult(content)

	if len(res.Findings) != 0 {
		t.Fatalf("want no findings, got %#v", res.Findings)
	}
	if !res.NonCanonical {
		t.Error("want NonCanonical true — spaces after the closing fence are not canonical")
	}
}

func TestTrailingTabsOnClosingFenceAreNonCanonical(t *testing.T) {
	content := openFence + "\n" + body() + closeFence + "\t\t\n"

	res := cursor.ParseBlockResult(content)

	if len(res.Findings) != 0 {
		t.Fatalf("want no findings, got %#v", res.Findings)
	}
	if !res.NonCanonical {
		t.Error("want NonCanonical true — tabs after the closing fence are not canonical")
	}
}

func TestCanonicalBlockAtEOFWithoutTrailingNewlineIsCanonical(t *testing.T) {
	res := cursor.ParseBlockResult(block())

	if len(res.Findings) != 0 {
		t.Fatalf("want no findings, got %#v", res.Findings)
	}
	if res.NonCanonical {
		t.Error("want NonCanonical false — the canonical block ends at EOF without a trailing newline")
	}
}

// A CRLF document's fence line carries a trailing carriage return. It is a
// line ending, not authored content, so it must not stop a fence from being
// recognized — the substring scan this replaced found the tag in CRLF files,
// and losing that would be a regression rather than a tightening.
func TestCRLFDocumentFencesAreRecognized(t *testing.T) {
	content := strings.ReplaceAll("# Handoff\n\n"+block()+"\n", "\n", "\r\n")

	if !cursor.HasBlock(content) {
		t.Error("want HasBlock true — a CRLF fence line is still a fence")
	}
	res := cursor.ParseBlockResult(content)
	if !res.HasCursor {
		t.Fatal("want HasCursor true for a CRLF document carrying a real block")
	}
	if len(res.Findings) != 0 {
		t.Fatalf("want no findings, got %#v", res.Findings)
	}
	if res.Cursor.Effort != "x" {
		t.Errorf("Effort = %q", res.Cursor.Effort)
	}
}

// Two column-0 open fences is refused outright, naming both line numbers and
// the remedy. Picking one of two candidate blocks is the failure class this
// work exists to eliminate.
func TestTwoOpenFencesIsFindingNamingBothLines(t *testing.T) {
	content := block() + "\n\n" + block() + "\n"

	res := cursor.ParseBlockResult(content)

	if !res.HasCursor {
		t.Fatal("want HasCursor true — fences are present")
	}
	got := findingsText(res)
	if !strings.Contains(got, "2 opening") {
		t.Errorf("findings = %q, want it to name the structural cause (2 opening fences)", got)
	}
	for _, line := range []string{"1", "8"} {
		if !strings.Contains(got, line) {
			t.Errorf("findings = %q, want it to name line %s", got, line)
		}
	}
	if !strings.Contains(got, "indent the example") {
		t.Errorf("findings = %q, want it to carry the remedy", got)
	}
	if res.Cursor.Effort != "" {
		t.Errorf("Effort = %q, want no parse attempted against an ambiguous document", res.Cursor.Effort)
	}
}

// Symmetry: a duplicated close fence is as much a hand-edit signal as a
// duplicated open one.
func TestTwoCloseFencesIsFinding(t *testing.T) {
	content := block() + "\n" + closeFence + "\n"

	res := cursor.ParseBlockResult(content)

	got := findingsText(res)
	if !strings.Contains(got, "2 closing") {
		t.Errorf("findings = %q, want it to name 2 closing fences", got)
	}
}

// Symmetry: today the close is searched only after the open, so a stray close
// above the block is invisible.
func TestCloseFenceBeforeOpenIsFinding(t *testing.T) {
	content := closeFence + "\n\n" + openFence + "\n" + body() + closeFence + "\n"

	res := cursor.ParseBlockResult(content)

	got := findingsText(res)
	if len(res.Findings) == 0 {
		t.Fatal("want a finding — a closing fence precedes the opening fence")
	}
	if !strings.Contains(got, "2 closing") && !strings.Contains(got, "precedes") {
		t.Errorf("findings = %q, want it to describe the ordering or the duplicate", got)
	}
}

// A document carrying a close fence and no open one has lost its opening
// fence. It is not the quiet no-block case: cursor scaffolding is demonstrably
// present, so this reports rather than falling silent.
func TestCloseFenceWithNoOpenIsFinding(t *testing.T) {
	content := "# Handoff\n\n" + closeFence + "\n"

	res := cursor.ParseBlockResult(content)

	if !res.HasCursor {
		t.Fatal("want HasCursor true — a fence is present, so this is not the quiet case")
	}
	if len(res.Findings) == 0 {
		t.Fatal("want a finding for a closing fence with no opening fence")
	}
}

// The quiet, hook-friendly case must stay quiet: no fences at all is not a
// finding, it is simply nothing to derive against.
func TestNoFencesAtAllStaysQuiet(t *testing.T) {
	content := "# Handoff\n\nThe `spine:cursor` block records where an effort stands.\n"

	res := cursor.ParseBlockResult(content)

	if res.HasCursor {
		t.Error("want HasCursor false — a backticked bare word is not a fence")
	}
	if len(res.Findings) != 0 {
		t.Errorf("want no findings on a document with no fences, got %#v", res.Findings)
	}
}

// D2: the presence test and the parse must agree on what counts as a block.
// If HasBlock stayed a bare substring test while the parse anchored, a prose
// mention would report "a block exists" and "there is no block" for the same
// document — an empty Result with zero findings, which the derivation engine
// then misreports as a stale effort.
func TestHasBlockAgreesWithParseOnProseMention(t *testing.T) {
	content := "# Handoff\n\nNever hand-edit the " + openFence + " block.\n"

	if cursor.HasBlock(content) {
		t.Error("want HasBlock false — a mid-line mention is prose, not a fence")
	}
	if res := cursor.ParseBlockResult(content); res.HasCursor {
		t.Error("want HasCursor false — must agree with HasBlock")
	}
}

// An open fence with no close stays its own distinct finding, separate from
// the duplicate-fence case, and now names where the block was opened.
func TestUnterminatedBlockNamesTheOpeningLine(t *testing.T) {
	content := "# Handoff\n\n" + openFence + "\n" + body()

	res := cursor.ParseBlockResult(content)

	if !res.HasCursor {
		t.Fatal("want HasCursor true — an open fence was found")
	}
	got := findingsText(res)
	if !strings.Contains(got, "closing") {
		t.Fatalf("findings = %q, want it to name the missing closing fence", got)
	}
	if !strings.Contains(got, "3") {
		t.Errorf("findings = %q, want it to name the opening line (3)", got)
	}
}

// D4: findings carry whole-document line numbers, because the operator has
// the file open in an editor. A body-relative number restarts the hunt.
func TestBodyFindingsCarryWholeDocumentLineNumbers(t *testing.T) {
	content := "# Handoff\n\nSome prose.\n\n" +
		openFence + "\n" +
		"effort: x\n" +
		"prd: docs/specs/x.md\n" +
		"nonsense\n" +
		"tickets: I001\n" +
		"stages: grill[<] prd[ ] issues[ ] implement[ ]\n" +
		closeFence + "\n"

	res := cursor.ParseBlockResult(content)

	got := findingsText(res)
	if !strings.Contains(got, "nonsense") {
		t.Fatalf("findings = %q, want the offending line quoted", got)
	}
	if !strings.Contains(got, "line 8") {
		t.Errorf("findings = %q, want the whole-document line number (line 8)", got)
	}
}

// The negative control, taken from a real committed handoff
// (2026-07-24-flavor-model-table-i033-i039.md): a genuine block followed by a
// second complete literal mid-line further down. It parses correctly today
// only because first-match-wins happens to open at the real block. Strict
// column 0 must keep it clean — proving the rule is not over-broad.
func TestRealHandoffSpecimenWithMidLineLiteralParsesClean(t *testing.T) {
	content := "# Handoff\n\n" +
		block() + "\n\n" +
		"above, which clears it. Note the gate requires the literal `" + openFence + "` marker block — a\n" +
		"hand-written one will not do.\n"

	res := cursor.ParseBlockResult(content)

	if len(res.Findings) != 0 {
		t.Fatalf("want the real specimen to parse clean, got %#v", res.Findings)
	}
	if res.Cursor.Effort != "x" {
		t.Errorf("Effort = %q", res.Cursor.Effort)
	}
}

// The other arm of the control: move that same second literal to column 0 and
// it must block — proving the rule is load-bearing rather than vacuous.
func TestSameSpecimenWithSecondLiteralAtColumnZeroBlocks(t *testing.T) {
	content := "# Handoff\n\n" +
		block() + "\n\n" +
		openFence + "\n"

	res := cursor.ParseBlockResult(content)

	if len(res.Findings) == 0 {
		t.Fatal("want a finding — two column-0 opening fences")
	}
	if !strings.Contains(findingsText(res), "2 opening") {
		t.Errorf("findings = %q, want the duplicate-open finding", findingsText(res))
	}
}

// Load is the highest seam in the package and must inherit the same anchoring.
func TestLoadSkipsProseFenceMentionInLedger(t *testing.T) {
	dir := t.TempDir()
	writeFixtureFiles(t, dir, map[string]string{
		"WORKFLOW.md": "stages: [grill, prd, issues, implement]\n",
		".superpowers/sdd/progress.md": "# Progress\n\nNever hand-edit the `" + openFence + "` block.\n\n" +
			block() + "\n",
	})

	res, err := cursor.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !res.HasCursor {
		t.Fatal("want HasCursor true")
	}
	if len(res.Findings) != 0 {
		t.Fatalf("want no findings, got %#v", res.Findings)
	}
	if res.Cursor.Effort != "x" {
		t.Errorf("Effort = %q", res.Cursor.Effort)
	}
}

// D8: the write path replaces open-through-close, so against an ambiguous
// document it would destroy everything between two candidate fences. It must
// refuse, name both lines, and leave the file byte-identical.
func TestSaveRefusesDuplicateOpenFences(t *testing.T) {
	dir := t.TempDir()
	ledger := block() + "\n\n" + block() + "\n"
	writeFixtureFiles(t, dir, map[string]string{
		"WORKFLOW.md":                  "stages: [grill, prd, issues, implement]\n",
		".superpowers/sdd/progress.md": ledger,
	})

	c, err := cursor.New(dir, "new-effort", "docs/specs/new.md", "I002")
	if err != nil {
		t.Fatal(err)
	}
	err = cursor.Save(dir, c)

	if err == nil {
		t.Fatal("want an error — Save must refuse to rewrite an ambiguous ledger")
	}
	for _, want := range []string{"1", "8"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to name line %s", err.Error(), want)
		}
	}
	after, readErr := os.ReadFile(filepath.Join(dir, filepath.FromSlash(".superpowers/sdd/progress.md")))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != ledger {
		t.Error("want the ledger left byte-identical after a refused write")
	}
}

// Save must anchor the same way it parses: a prose mention above the real
// block must not become the replacement target, which would delete the
// intervening text.
func TestSaveReplacesTheRealBlockNotAProseMention(t *testing.T) {
	dir := t.TempDir()
	preserve := "Never hand-edit the `" + openFence + "` block."
	writeFixtureFiles(t, dir, map[string]string{
		"WORKFLOW.md":                  "stages: [grill, prd, issues, implement]\n",
		".superpowers/sdd/progress.md": "# Progress\n\n" + preserve + "\n\n" + block() + "\n",
	})

	c, err := cursor.New(dir, "new-effort", "docs/specs/new.md", "I002")
	if err != nil {
		t.Fatal(err)
	}
	if err := cursor.Save(dir, c); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(".superpowers/sdd/progress.md")))
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	if !strings.Contains(got, preserve) {
		t.Error("want the prose line preserved — Save replaced from the prose mention onward")
	}
	if !strings.Contains(got, "effort: new-effort") {
		t.Error("want the real block replaced with the new cursor")
	}
}
