package update

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/russellpope/spine/internal/fsutil"
	"github.com/russellpope/spine/internal/tmpl"
)

// FileState classifies what update would do to one file.
type FileState int

const (
	UpToDate FileState = iota
	Pending
	SkippedUnrecognized
)

// FileReport is the per-file outcome. newContent stays unexported: only Run
// writes it, and only for Pending files.
type FileReport struct {
	Path         string
	State        FileState
	Diff         string
	Unrecognized []string
	// Created is true when the file did not exist on disk at plan time, so a
	// Pending state means "will be created" rather than "will be updated".
	Created bool
	// Preserved is true for a legacyPreserve file (docs/adr/README.md) whose
	// unrecognized hand-authored content was left as-is rather than flagged.
	// Only set when State == UpToDate. --force clears this and regenerates.
	Preserved  bool
	newContent string
}

// Options configures Run. Zero value = dry-run on ".". AdoptProfile switches
// on adopt mode: a missing WORKFLOW.md is synthesized from that profile's
// defaults (project name = AdoptName, else the dir basename) instead of
// being a hard error. Set only by spine adopt.
type Options struct {
	Dir          string
	Write        bool
	Force        bool
	AdoptProfile string
	AdoptName    string
}

const (
	markerBegin = "<!-- spine:begin"
	markerEnd   = "<!-- spine:end -->"
)

// simple machine-owned files: regenerate wholesale, no key extraction.
// inGen0 marks files whose gen0 content differs from current. legacyPreserve
// marks the one file (docs/adr/README.md) where unrecognized hand-authored
// content is a deliberate choice, not drift: ADR 0009.
var simpleFiles = []struct {
	tmplName, relPath string
	inGen0            bool
	legacyPreserve    bool
}{
	{"harness-interface.md", "docs/harness-interface.md", true, false},
	{"issues-README.md", "docs/issues/README.md", false, false},
	{"issue.tmpl.md", "docs/issues/_template.md", false, false},
	{"adr-README.md", "docs/adr/README.md", false, true},
}

// Run plans (and with opts.Write, applies) regeneration of every managed file.
func Run(opts Options) ([]FileReport, error) {
	if opts.Dir == "" {
		opts.Dir = "."
	}
	wf, vals, gen, err := planWorkflow(opts)
	if err != nil {
		return nil, err
	}
	reports := []FileReport{wf}
	cl, err := planClaude(opts.Dir, gen, vals)
	if err != nil {
		return nil, err
	}
	reports = append(reports, cl)
	ag, err := planAgents(opts.Dir, vals)
	if err != nil {
		return nil, err
	}
	reports = append(reports, ag)
	legacyPreserve := map[string]bool{}
	for _, f := range simpleFiles {
		if f.legacyPreserve {
			legacyPreserve[f.relPath] = true
		}
		if !tmpl.ProfileOwns(vals.Profile, f.relPath) {
			continue
		}
		r, err := planSimple(opts.Dir, gen, f.tmplName, f.relPath, f.inGen0, vals)
		if err != nil {
			return nil, err
		}
		reports = append(reports, r)
	}
	// docs/evals/README.md is opt-in machine-owned: managed only where the
	// convention is in use (the directory exists); never created by init/adopt.
	fi, err := os.Stat(filepath.Join(opts.Dir, "docs", "evals"))
	switch {
	case err == nil && fi.IsDir():
		r, err := planSimple(opts.Dir, gen, "evals-README.md", "docs/evals/README.md", false, vals)
		if err != nil {
			return nil, err
		}
		reports = append(reports, r)
	case err != nil && !os.IsNotExist(err):
		return nil, err
	}
	// policy: unrecognized edits skip the file unless --force; files with no
	// regenerable content (nil newContent) stay skipped regardless. The one
	// exception is legacyPreserve (docs/adr/README.md, ADR 0009): a
	// hand-authored index is a deliberate choice, not drift, so it's treated
	// as up-to-date rather than skipped/warned — --force is the explicit
	// opt-in to regenerate it from the template.
	for i := range reports {
		r := &reports[i]
		if len(r.Unrecognized) > 0 {
			if legacyPreserve[r.Path] && !opts.Force {
				r.State = UpToDate
				r.Preserved = true
				r.Diff = ""
				continue
			}
			if opts.Force && r.newContent != "" {
				r.State = Pending
			} else {
				r.State = SkippedUnrecognized
			}
		}
	}
	if opts.Write {
		for i := range reports {
			r := &reports[i]
			if r.State != Pending {
				continue
			}
			dst := filepath.Join(opts.Dir, r.Path)
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return reports, err
			}
			if err := fsutil.WriteFileAtomic(dst, []byte(r.newContent)); err != nil {
				return reports, err
			}
		}
	}
	return reports, nil
}

func planWorkflow(opts Options) (FileReport, tmpl.Values, string, error) {
	report := FileReport{Path: "WORKFLOW.md"}
	path := filepath.Join(opts.Dir, "WORKFLOW.md")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) && opts.AdoptProfile != "" {
		project := opts.AdoptName
		if project == "" {
			abs, aerr := filepath.Abs(opts.Dir)
			if aerr != nil {
				return report, tmpl.Values{}, "", aerr
			}
			project = filepath.Base(abs)
		}
		defRev, defHarness, derr := tmpl.Defaults(opts.AdoptProfile)
		if derr != nil {
			return report, tmpl.Values{}, "", derr
		}
		vals := tmpl.Values{Project: project, Profile: opts.AdoptProfile,
			Reviewers: defRev, Harness: defHarness, Version: tmpl.Version()}
		newContent, rerr := tmpl.Render("current", "WORKFLOW.md.tmpl", vals)
		if rerr != nil {
			return report, tmpl.Values{}, "", rerr
		}
		report.State = Pending
		report.Created = true
		report.Diff = Diff(report.Path, "", newContent)
		report.newContent = newContent
		return report, vals, "current", nil
	}
	if err != nil {
		return report, tmpl.Values{}, "", fmt.Errorf("read %s (run spine init first?): %w", path, err)
	}
	old := string(raw)
	keys := ExtractKeys(old)
	gen := "gen0"
	if tv := keys["template_version"]; tv != "" {
		// A stamped generation newer than what this binary compiles is never
		// "current" — that would silently downgrade the file. Non-integer
		// stamps fall through to the existing current-gen treatment.
		if n, err := strconv.Atoi(tv); err == nil && n > tmpl.Version() {
			return report, tmpl.Values{}, "", fmt.Errorf(
				"WORKFLOW.md is template generation %d but this spine binary compiles generation %d — upgrade spine (make install in ~/Projects/github.com/spine)",
				n, tmpl.Version())
		}
		gen = "current"
	}
	abs, err := filepath.Abs(opts.Dir)
	if err != nil {
		return report, tmpl.Values{}, "", err
	}
	project := ProjectFromWorkflow(old, filepath.Base(abs))
	profile := keys["profile"]
	if profile == "" {
		return report, tmpl.Values{}, "", fmt.Errorf("%s has no profile: line", path)
	}
	defRev, defHarness, err := tmpl.Defaults(profile)
	if err != nil {
		return report, tmpl.Values{}, "", err
	}
	vals := tmpl.Values{Project: project, Profile: profile, Reviewers: defRev, Harness: defHarness, Version: tmpl.Version()}

	// unrecognized detection: what the old generation would look like with
	// every extracted key applied — anything beyond that is a local edit.
	expectedOld, err := tmpl.Render(gen, "WORKFLOW.md.tmpl", vals)
	if err != nil {
		return report, tmpl.Values{}, "", err
	}
	for k, v := range keys {
		expectedOld = setKey(expectedOld, k, v)
	}
	newContent, err := tmpl.Render("current", "WORKFLOW.md.tmpl", vals)
	if err != nil {
		return report, tmpl.Values{}, "", err
	}
	report.Unrecognized = unrecognizedLines(old, expectedOld, newContent)

	choices, err := Choices(keys, gen, project)
	if err != nil {
		return report, tmpl.Values{}, "", err
	}
	for k, v := range choices {
		if k == "profile" {
			continue
		}
		newContent = setKey(newContent, k, v)
	}
	if d := Diff(report.Path, old, newContent); d != "" {
		report.State = Pending
		report.Diff = d
		report.newContent = newContent
	}
	return report, vals, gen, nil
}

func planClaude(dir, gen string, vals tmpl.Values) (FileReport, error) {
	report := FileReport{Path: "CLAUDE.md"}
	block, err := tmpl.Render("current", "CLAUDE.md.tmpl", vals)
	if err != nil {
		return report, err
	}
	path := filepath.Join(dir, "CLAUDE.md")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		report.State = Pending
		report.Created = true
		report.Diff = Diff(report.Path, "", block)
		report.newContent = block
		return report, nil
	}
	if err != nil {
		return report, err
	}
	old := string(raw)
	var newContent string
	if strings.Contains(old, markerBegin) {
		replaced, err := replaceMarkerBlock(report.Path, old, block)
		if err != nil {
			// unbalanced markers: never force-droppable, no newContent.
			report.Unrecognized = []string{err.Error()}
			return report, nil
		}
		newContent = replaced
	} else {
		gen0, err := tmpl.Render("gen0", "CLAUDE.md.tmpl", vals)
		if err != nil {
			return report, err
		}
		if strings.TrimSpace(old) == strings.TrimSpace(gen0) {
			newContent = block // pristine legacy file: clean claim
		} else {
			newContent = block + "\n" + old // claim on top, preserve everything
		}
	}
	if d := Diff(report.Path, old, newContent); d != "" {
		report.State = Pending
		report.Diff = d
		report.newContent = newContent
	}
	return report, nil
}

func planAgents(dir string, vals tmpl.Values) (FileReport, error) {
	report := FileReport{Path: "AGENTS.md"}
	block, err := tmpl.Render("current", "AGENTS.md.tmpl", vals)
	if err != nil {
		return report, err
	}
	path := filepath.Join(dir, "AGENTS.md")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		report.State = Pending
		report.Created = true
		report.Diff = Diff(report.Path, "", block)
		report.newContent = block
		return report, nil
	}
	if err != nil {
		return report, err
	}
	old := string(raw)
	var newContent string
	if strings.Contains(old, markerBegin) {
		replaced, err := replaceMarkerBlock(report.Path, old, block)
		if err != nil {
			// unbalanced markers: never force-droppable, no newContent.
			report.Unrecognized = []string{err.Error()}
			return report, nil
		}
		newContent = replaced
	} else {
		// No spine-owned block yet: claim on top, preserve everything below.
		// (AGENTS.md has no gen0 template, so there is no pristine-legacy
		// clean-claim case as CLAUDE.md has.)
		newContent = block + "\n" + old
	}
	if d := Diff(report.Path, old, newContent); d != "" {
		report.State = Pending
		report.Diff = d
		report.newContent = newContent
	}
	return report, nil
}

func replaceMarkerBlock(path, old, block string) (string, error) {
	if strings.Count(old, markerBegin) != 1 || strings.Count(old, markerEnd) != 1 {
		return "", fmt.Errorf("%s spine markers unbalanced; fix by hand", path)
	}
	begin := strings.Index(old, markerBegin)
	end := strings.Index(old, markerEnd)
	if end < begin {
		return "", fmt.Errorf("%s spine markers out of order; fix by hand", path)
	}
	return old[:begin] + strings.TrimSuffix(block, "\n") + old[end+len(markerEnd):], nil
}

func planSimple(dir, gen, tmplName, relPath string, inGen0 bool, vals tmpl.Values) (FileReport, error) {
	report := FileReport{Path: relPath}
	newContent, err := tmpl.Render("current", tmplName, vals)
	if err != nil {
		return report, err
	}
	path := filepath.Join(dir, relPath)
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		report.State = Pending
		report.Created = true
		report.Diff = Diff(relPath, "", newContent)
		report.newContent = newContent
		return report, nil
	}
	if err != nil {
		return report, err
	}
	old := string(raw)
	expectedGen := "current"
	if gen == "gen0" && inGen0 {
		expectedGen = "gen0"
	}
	expectedOld, err := tmpl.Render(expectedGen, tmplName, vals)
	if err != nil {
		return report, err
	}
	report.Unrecognized = unrecognizedLines(old, expectedOld, newContent)
	if d := Diff(relPath, old, newContent); d != "" {
		report.State = Pending
		report.Diff = d
		report.newContent = newContent
	}
	return report, nil
}

// supersededLines are lines a prior generation emitted that the current one
// no longer does. Unrecognized-detection renders only gen0 and current, so
// without this list a machine-emitted line changed by a content-bearing bump
// would read as a local edit and skip the file. Each generation that changes
// emitted content appends its predecessors' dropped lines here.
var supersededLines = map[string]bool{
	// gen0–4 WORKFLOW.md gates line, reworded in gen 5 (to-spec, spec-review).
	"Mandatory gates: a PRD up front (grill-with-docs -> to-prd) and verification before completion.": true,
	// gen5 WORKFLOW.md model_routing lines, rewritten in gen 6 as the full
	// dispatch contract: reworded comments, the mechanical tier added, the
	// standalone security_routing key folded into fallback semantics, and
	// the old one-line execution-mode rule replaced by the Execution modes
	// section (I003).
	"  primary: claude-fable-5          # long-horizon, ambiguous, or first-shot-complex work (design, plan, implement, orchestrate)":           true,
	"  fallback: claude-opus-4-8        # auto on stop_reason: refusal (cyber/bio/reasoning-extraction); also context/usage exhaustion":         true,
	"  routine: claude-sonnet-5         # mechanical subagent roles: doc edits, plan-transcription implementers, build fixers, simple reviews":  true,
	"effort: high                       # default; xhigh for security-critical analysis + final verification; medium/low for routine subagents": true,
	"security_routing: quality-framing-opus-4-8": true,
	"Execution mode per plan: live-system mutation, secrets, or interactive steps -> inline with the human; otherwise subagent-driven.": true,

	// ultima-dci-edition's hand-written "## Stage cursor (consistency rule)"
	// section (gen 7, real repo, captured verbatim 2026-07-15 — see
	// internal/update/testdata/ultima/WORKFLOW.md), superseded wholesale by
	// gen 8's spine-owned section of the same name (I020). This section
	// predates the I018 cursor grammar: it describes a prose checklist +
	// "← YOU ARE HERE" marker, not the `<!-- spine:cursor -->` block.
	"## Stage cursor (consistency rule)": true,
	"Stages run **in order**; none may be silently skipped (the miss mode is a handoff that names an":           true,
	"abbreviated path — e.g. \"grill -> to-spec -> build\" quietly dropping `issues`/`to-tickets`). To prevent": true,
	"it, every SDD effort's `.superpowers/sdd/progress.md` **opens with a WORKFLOW stage checklist** — one":     true,
	"line per stage above, ticked as each completes, with a `← YOU ARE HERE` marker on the active stage. The":   true,
	"cursor is the single source of truth for \"where are we\"; check it at session start before acting.":       true,
	"**Handoff rule:** `/handoff` and any resume/kickoff prompt MUST carry the stage cursor **verbatim** (not":  true,
	"a prose paraphrase of \"what's next\"). A handoff that names the next action without the full cursor is":   true,
	"incomplete — the reader can't see which upstream stage was skipped. When in doubt, re-derive the cursor":   true,
	"from the artifacts on disk: PRD in `docs/specs/` ⇒ `prd` done; build tickets in":                           true,
	"`docs/issues/` ⇒ `issues` done; per-task commits ⇒ `implement` in progress.":                               true,

	// gen 8's indented "tickets: I0NN-I0MM | prefix I0" Grammar-reference
	// line, reworded in gen 9 to admit a bare single-ticket id (I026). Note
	// the leading 4-space indent: this line lives inside the Stage-cursor
	// section's indented code block, unlike the other superseded lines
	// above, which are unindented prose — unrecognizedLines only
	// right-trims, so the key must carry the on-disk indentation verbatim.
	"    tickets: I0NN-I0MM | prefix I0": true,

	// gen 8/early-gen-9's **Handoff rule:** line, extended in place (M11,
	// I027, rides I026's gen-9 bump — no further generation bump) to state
	// the doctor-advises half of the I014 backstop alongside the
	// already-stated audit-stages-blocks half.
	"**Handoff rule:** `/handoff` and any resume/kickoff prompt MUST embed the verbatim output of `spine cursor` — a prose paraphrase of stage state is incomplete; the reader can't see which upstream stage was skipped from a summary alone.": true,
}

// unrecognizedLines returns non-blank lines of got that expected does not
// contain anywhere (order-insensitive, trailing-space-insensitive) and that
// no prior generation emitted, and that is not a sanctioned remap of a
// known key (see keyLineSignature): a got line whose key+comment match a
// want or supersededLines line is recognized regardless of its value or
// comment padding — the value is exactly what a remap changes, and a
// hand-typed comment column width was never meaningful.
//
// Signature recognition is limited to keys the CURRENT generation still
// renders (current): only those values are carry-forwardable via
// Choices/setKey. A customized value of a key the current generation
// REMOVED (e.g. a gen-5 security_routing local value under gen 6) has
// nowhere to go — accepting it as a remap would let a plain --write
// silently destroy it — so such lines stay literal-match-only and read as
// unrecognized local edits.
func unrecognizedLines(got, expected, current string) []string {
	currentKeys := map[string]bool{}
	for _, l := range splitLines(current) {
		if k, _, ok := keyLineSignature(l); ok {
			currentKeys[k] = true
		}
	}
	want := map[string]bool{}
	sigs := map[string]bool{}
	addSig := func(l string) {
		if k, sig, ok := keyLineSignature(l); ok && currentKeys[k] {
			sigs[sig] = true
		}
	}
	for _, l := range splitLines(expected) {
		t := strings.TrimRight(l, " ")
		want[t] = true
		addSig(t)
	}
	for l := range supersededLines {
		addSig(l)
	}
	var extra []string
	for _, l := range splitLines(got) {
		t := strings.TrimRight(l, " ")
		if t == "" || want[t] || supersededLines[t] {
			continue
		}
		if _, sig, ok := keyLineSignature(t); ok && sigs[sig] {
			continue // sanctioned remap: same key/comment; value and padding may differ
		}
		extra = append(extra, t)
	}
	return extra
}

// keyLineSignature is the identifying signature of a "key: value  #
// comment" line — a top-level key or a two-space-indented model_routing
// sub-key — with the value dropped and the comment kept verbatim, plus the
// bare key so callers can gate on which keys the current generation still
// renders. ok is false for anything that isn't a recognized key: value
// line (prose, headers, unknown keys), which keeps exact-text comparison
// for those.
func keyLineSignature(line string) (key, sig string, ok bool) {
	trimmed := strings.TrimSpace(line)
	for _, k := range topKeys {
		if _, has := cutKey(trimmed, k); has {
			return k, k + "\x00" + commentOf(trimmed, k), true
		}
	}
	for _, k := range routingKeys {
		if _, has := cutKey(trimmed, k); has {
			return k, k + "\x00" + commentOf(trimmed, k), true
		}
	}
	return "", "", false
}

// commentOf returns the trailing "# comment" of a "key: value # comment"
// line (comment padding stripped, comment text verbatim), or "" if the
// line carries none.
func commentOf(trimmed, key string) string {
	rest, _ := strings.CutPrefix(trimmed, key+":")
	if i := commentIndex(rest); i >= 0 {
		return strings.TrimSpace(rest[i:])
	}
	return ""
}
