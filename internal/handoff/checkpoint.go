package handoff

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/russellpope/spine/internal/checkpoint"
)

// Fixed headings for the embedded checkpoint. Both are byte-stable: a
// handoff reader (human or model) can find the facts region and the prior
// narrative by exact string.
const (
	checkpointHeading = "## Checkpoint (newest)"
	narrativeHeading  = "### Prior narrative (model-authored, not evidence)"
)

// checkpointEmbed renders the newest checkpoint in the checkpoint working
// home for appending to a new handoff, or "" when the working home is empty
// or absent — in which case the handoff is byte-identical to a pre-embed
// one.
//
// Shape, after the cursor block:
//
//	## Checkpoint (newest): NNN-<slug>.md
//	<!-- spine:checkpoint:facts -->   (facts region verbatim, markers included)
//	...
//	<!-- /spine:checkpoint:facts -->
//	### Prior narrative (model-authored, not evidence)
//	<model region content>
//
// The model region's own markers are deliberately NOT carried over: the
// fixed heading already labels the narrative as model-authored, and leaving
// the markers out keeps the embedded copy from reading as a second
// checkpoint document. The facts region keeps its markers so the harness
// evidence stays structurally separate from narrative, exactly as in the
// checkpoint itself.
//
// A checkpoint with no facts region (hand-mangled) does not fail the
// handoff: whatever regions exist are embedded and the gap is noted in
// place. The doctor advisory (D11) is where that drift is reported.
func checkpointEmbed(dir string) (string, error) {
	latest, ok, err := checkpoint.Latest(dir)
	if err != nil || !ok {
		return "", err
	}
	raw, err := os.ReadFile(latest.Path)
	if err != nil {
		return "", err
	}
	doc := checkpoint.Split(string(raw))

	var b strings.Builder
	b.WriteString("\n" + checkpointHeading + ": " + filepath.Base(latest.Path) + "\n\n")
	if doc.Facts == "" {
		b.WriteString("facts region: missing or malformed — run spine doctor\n\n")
	} else {
		b.WriteString(checkpoint.FactsOpenTag + "\n" + doc.Facts + "\n" + checkpoint.FactsCloseTag + "\n\n")
	}
	b.WriteString(narrativeHeading + "\n\n")
	if doc.Model == "" {
		b.WriteString("narrative: missing — reconstruct intent from facts\n")
	} else {
		b.WriteString(doc.Model + "\n")
	}
	return b.String(), nil
}
