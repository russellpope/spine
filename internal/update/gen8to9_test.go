package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// gen9ContentLines are the emitted-content changes gen 9 ships (I026), both
// sides of the diff: the gen-8 tickets: grammar line it drops/rewords
// (removed, "-") and the gen-9 line that replaces it (added, "+"). Diff
// lines are TrimSpace'd before lookup here (matching isGen9ContentDiffLine),
// so map keys carry no leading indent even though the rendered file (inside
// the Stage-cursor section's indented Grammar reference code block) does.
var gen9ContentLines = map[string]bool{
	// gen-8 tickets: grammar line, reworded in gen 9 to admit a bare
	// single-ticket id (I026) — same-endpoint ranges (I001-I001) already
	// resolved structurally and needed no code change, only documentation.
	"tickets: I0NN-I0MM | prefix I0":        true,
	"tickets: I0NN | I0NN-I0MM | prefix I0": true,

	// gen-8's **Handoff rule:** line, extended in place (M11, I027) to add
	// the doctor-advises half of the I014 backstop alongside the
	// already-stated audit-stages-blocks half. Rides I026's gen-9 bump —
	// both the dropped ("-") original wording and its ("+") replacement.
	"**Handoff rule:** `/handoff` and any resume/kickoff prompt MUST embed the verbatim output of `spine cursor` — a prose paraphrase of stage state is incomplete; the reader can't see which upstream stage was skipped from a summary alone.":                                                                                                                                     true,
	"**Handoff rule:** `/handoff` and any resume/kickoff prompt MUST embed the verbatim output of `spine cursor` — a prose paraphrase of stage state is incomplete; the reader can't see which upstream stage was skipped from a summary alone. Alongside `spine audit stages` blocking on a missing/stale cursor block in the newest handoff, `spine doctor` advises (warns) on the same condition.": true,
}

// isGen9ContentDiffLine reports whether a unified-diff line carries the
// gen-9 content change above, or is a bare added/removed blank line.
func isGen9ContentDiffLine(line string) bool {
	if len(line) == 0 || (line[0] != '+' && line[0] != '-') {
		return false
	}
	body := strings.TrimSpace(line[1:])
	return body == "" || gen9ContentLines[body]
}

// The ccq-gen8 fixture is a pristine gen-8 render (Version: 8, project
// "ccq", profile library-cli — same lineage as ccq-gen7) captured before the
// I026 template edit landed. It must update cleanly to gen 9: zero
// unrecognized lines, and the diff is exactly the stamp bump plus the
// declared gen-9 tickets: line change — proving a pristine gen-8 repo is
// recognized cleanly by a gen-9 binary rather than reporting unrecognized
// local edits (the ticket's explicit "gen8ContentLines-style" requirement).
func TestGen8To9PristineUpdatesCleanly(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"WORKFLOW.md", "CLAUDE.md"} {
		raw, err := os.ReadFile(filepath.Join("testdata", "ccq-gen8", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, r := range reports {
		switch r.Path {
		case "WORKFLOW.md", "CLAUDE.md":
			seen[r.Path] = true
			if len(r.Unrecognized) > 0 {
				t.Errorf("%s: pristine gen-8 lines misread as local edits: %v", r.Path, r.Unrecognized)
			}
			if r.State != Pending {
				t.Errorf("%s: want Pending, got %v", r.Path, r.State)
				continue
			}
			for _, line := range strings.Split(r.Diff, "\n") {
				if !strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "-") {
					continue
				}
				if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
					continue
				}
				if strings.Contains(line, "template_version") || strings.Contains(line, "spine:begin") {
					continue
				}
				if isGen9ContentDiffLine(line) {
					continue
				}
				t.Errorf("%s: unexpected changed line %q — 8→9 must be stamp plus declared gen-9 content only", r.Path, line)
			}
		}
	}
	for _, name := range []string{"WORKFLOW.md", "CLAUDE.md"} {
		if !seen[name] {
			t.Errorf("%s: never reported by Run — the lock did not exercise it", name)
		}
	}
}

// After a --write migration, the pristine gen-8 fixture's WORKFLOW.md must
// carry the gen-9 stamp and the new tickets: grammar wording, and must not
// still carry the gen-8 wording (the old line has exactly one replacement,
// not a duplicate).
func TestGen8To9MigrationCarriesNewGrammarLine(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"WORKFLOW.md", "CLAUDE.md"} {
		raw, err := os.ReadFile(filepath.Join("testdata", "ccq-gen8", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	gotStr := string(got)
	if !strings.Contains(gotStr, "template_version: 9") {
		t.Errorf("migrated WORKFLOW.md missing template_version: 9")
	}
	if !strings.Contains(gotStr, "tickets: I0NN | I0NN-I0MM | prefix I0") {
		t.Errorf("migrated WORKFLOW.md missing gen-9 tickets: grammar line")
	}
	if strings.Contains(gotStr, "tickets: I0NN-I0MM | prefix I0") {
		t.Errorf("migrated WORKFLOW.md still contains the superseded gen-8 tickets: grammar line")
	}
}

// A gen-8 repo that has hand-edited the tickets: grammar line to something
// else entirely (not the gen-8 wording, not the gen-9 wording) still reads
// as an unrecognized local edit — supersededLines only recognizes the exact
// gen-8 text, not arbitrary content in that position.
func TestGen8To9HandEditedTicketsLineStaysUnrecognized(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "ccq-gen8", "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := strings.Replace(string(raw),
		"    tickets: I0NN-I0MM | prefix I0\n",
		"    tickets: totally-custom-grammar\n", 1)
	if content == string(raw) {
		t.Fatal("fixture tickets: line not found to replace")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "WORKFLOW.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	claudeRaw, err := os.ReadFile(filepath.Join("testdata", "ccq-gen8", "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), claudeRaw, 0o644); err != nil {
		t.Fatal(err)
	}

	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != SkippedUnrecognized {
		t.Fatalf("hand-edited tickets: line must skip the file, got state=%v unrec=%v", wf.State, wf.Unrecognized)
	}
	named := false
	for _, u := range wf.Unrecognized {
		if strings.Contains(u, "totally-custom-grammar") {
			named = true
		}
	}
	if !named {
		t.Errorf("skip must name the hand-edited line, got %v", wf.Unrecognized)
	}
}
