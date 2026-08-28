package update

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/russellpope/spine/internal/fsutil"
	"github.com/russellpope/spine/internal/model"
	"github.com/russellpope/spine/internal/tmpl"
)

// FileState classifies what update would do to one file.
type FileState int

const (
	UpToDate FileState = iota
	Pending
	SkippedUnrecognized
	// SkippedPreflight is a planned file that cannot be safely touched until a
	// required external preflight is available. Unlike local-edit skips, it
	// does not make the rest of an update fail (I104).
	SkippedPreflight
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
	Preserved bool
	// ModelRefreshes itemizes each model_routing value whose on-disk value
	// matched a shipped historical default and was moved to the current
	// default (design D6). WORKFLOW.md only. The plan must surface these
	// individually, distinct from template prose churn.
	ModelRefreshes []ModelRefresh
	// ModelOverrides lists model_routing values preserved untouched as
	// deliberate per-repo choices (matched no shipped default). WORKFLOW.md
	// only.
	ModelOverrides []ModelOverride
	// StagesAdded, StagesRemoved, and StagesChanged name the gate-pack stages this render
	// adds to, or drops from, the region already in maipipe.toml. maipipe.toml
	// only, and only when the file already carries a region — a region being
	// written for the first time is wholly visible in the plan diff. The plan
	// prints them so the cost is visible before --write (I098): an added stage
	// is a new step in a gating lane, and either change rewrites the region's
	// bytes and so maipipe's definition_hash.
	StagesAdded   []string
	StagesRemoved []string
	StagesChanged []string
	// Refusal is why applying this file's pending content would be refused
	// (maipipe.toml only, I096). It is computed during the plan pass, not at
	// write time, because the plan diff is the review surface (ADR 0017): a
	// blocker the dry-run cannot show is a blocker the reader meets for the
	// first time when --write fails.
	Refusal string
	// Preflight records the pre-write check that ran, or the prerequisite that
	// caused this file to be skipped. maipipe.toml only (I104).
	Preflight  string
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
	{"remediation-README.md", "docs/remediation/README.md", false, false},
	{"hitlist.tmpl.md", "docs/remediation/_hitlist.template.md", false, false},
	{"remediation-round.tmpl.md", "docs/remediation/_round.template.md", false, false},
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
	// The gate pack's delivery region in maipipe.toml (ADR 0017, superseding
	// 0016) is planned from the WORKFLOW.md this run produces, so an owner
	// who sets gate_pack: and runs update once gets both the key and the
	// region.
	workflow := wf.newContent
	if workflow == "" {
		raw, err := os.ReadFile(filepath.Join(opts.Dir, "WORKFLOW.md"))
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		workflow = string(raw)
	}
	mp, ok, err := planMaipipe(opts.Dir, workflow)
	if err != nil {
		return nil, err
	}
	if ok {
		reports = append(reports, mp)
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
	// maipipe is the grammar authority for a gate-pack candidate (I104). It
	// runs in the plan pass, on every run — not only under --write — because
	// the plan is what a reader reviews before applying it (ADR 0017). When
	// maipipe is unavailable, this one file is a separately reported skip;
	// unrelated pending files are still safe to apply.
	for i := range reports {
		r := &reports[i]
		if r.Path != MaipipeFile || r.State != Pending {
			continue
		}
		// An I097 opt-out composition refusal is already the plan's verdict;
		// do not replace it by trying to validate an intentionally absent
		// candidate. --write below preserves whole-plan atomicity.
		if r.Refusal != "" {
			continue
		}
		bin, err := maipipeLookup("maipipe")
		if err != nil {
			r.State = SkippedPreflight
			r.Diff = ""
			r.newContent = ""
			r.Preflight = noMaipipePreflight
			continue
		}
		r.Preflight = maipipeValidatePreflight
		if err := checkMaipipeContent(bin, filepath.Join(opts.Dir, r.Path), r.newContent); err != nil {
			r.Refusal = err.Error()
		}
	}
	if opts.Write {
		// A refusal aborts the whole run: update presents one plan and
		// applies it as a whole, and a partial application would leave a
		// rendered region stale against a WORKFLOW.md that already moved —
		// a state the reader has to reason about instead of retrying. The
		// refusal says so, because an error naming only maipipe.toml would
		// read as if maipipe.toml were the only file skipped.
		for i := range reports {
			if r := &reports[i]; r.Refusal != "" {
				return reports, fmt.Errorf(
					"%s\nno files were written: spine update applies its plan as a whole, so every other pending file (WORKFLOW.md included) is unchanged too — fix %s and re-run",
					r.Refusal, MaipipeFile)
			}
		}
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
		// No WORKFLOW.md on disk, so every row resolves Default — this pins
		// the created file's model rows to the table's current defaults even
		// if the template text lags the table.
		newContent, _, _, rerr = applyModelRouting(opts.Dir, newContent, nil)
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
	// Model-routing rows resolve through the shared table, not the choice
	// rule above (D6/D7): inherited stale defaults refresh and are itemized,
	// deliberate overrides are written back verbatim.
	newContent, refreshes, modelOverrides, err := applyModelRouting(opts.Dir, newContent, keys)
	if err != nil {
		return report, tmpl.Values{}, "", err
	}
	report.ModelRefreshes = refreshes
	report.ModelOverrides = modelOverrides
	// Gen-10 retirement pass (D16): customized effort:/model_default: lines
	// are machine-owned retirement work — migrated or checked above — not
	// local edits; what remains surfaced is only a model_default value that
	// genuinely diverges from the resolved primary (controller ruling).
	report.Unrecognized = dropRetiredKeyLines(report.Unrecognized)
	divergence, err := modelDefaultDivergence(opts.Dir, gen, keys)
	if err != nil {
		return report, tmpl.Values{}, "", err
	}
	if divergence != "" {
		report.Unrecognized = append(report.Unrecognized, divergence)
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

// dropRetiredKeyLines removes customized top-level effort: and
// model_default: lines from the unrecognized set: both keys retire in gen 10
// (design D16 + controller ruling, I036) and their values are handled by
// applyModelRouting (a customized effort migrates into per-entry overrides)
// and modelDefaultDivergence (a deliberate divergent model_default is
// surfaced), so the raw lines are machine-owned retirement work, not local
// edits. Top-level only: the indented per-ticket/cursor-grammar effort:
// lines (design D17) never match the column-0 prefix.
func dropRetiredKeyLines(unrec []string) []string {
	var out []string
	for _, l := range unrec {
		if strings.HasPrefix(l, "effort:") || strings.HasPrefix(l, "model_default:") {
			continue
		}
		out = append(out, l)
	}
	return out
}

// supersededLines are lines a prior generation emitted that the current one
// no longer does. Unrecognized-detection renders only gen0 and current, so
// without this list a machine-emitted line changed by a content-bearing bump
// would read as a local edit and skip the file. Each generation that changes
// emitted content appends its predecessors' dropped lines here.
//
// This is binding, not advisory: a generation bump that changes emitted
// content is incomplete until the lines its predecessors emitted and it no
// longer does are appended here, in the same change. Skipping the append
// leaves every pristine repo of the older generation stuck — reported as
// locally modified and never refreshed (I065). The keys are the historical
// render verbatim, copied off the old template rather than retyped, with
// on-disk indentation intact (unrecognizedLines only right-trims).
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

	// gen 11's indented tickets grammar line, reworded in place by I114 to
	// document the comma-list form. This remains a content-bearing template
	// edit without a generation bump, so the exact predecessor must be
	// recognized here for an ordinary refresh to replace it.
	"    tickets: I0NN | I0NN-I0MM | prefix I0": true,

	// gen 8/early-gen-9's **Handoff rule:** line, extended in place (M11,
	// I027, rides I026's gen-9 bump — no further generation bump) to state
	// the doctor-advises half of the I014 backstop alongside the
	// already-stated audit-stages-blocks half.
	"**Handoff rule:** `/handoff` and any resume/kickoff prompt MUST embed the verbatim output of `spine cursor` — a prose paraphrase of stage state is incomplete; the reader can't see which upstream stage was skipped from a summary alone.": true,
	// gen 9's completed handoff rule is replaced in gen 10 (I060): handoff
	// creation now embeds the cursor automatically, and hand editing is a
	// workflow violation. Keep this exact full line in the supersession set
	// so ordinary gen-9 repos update rather than being skipped as locally
	// modified.
	"**Handoff rule:** `/handoff` and any resume/kickoff prompt MUST embed the verbatim output of `spine cursor` — a prose paraphrase of stage state is incomplete; the reader can't see which upstream stage was skipped from a summary alone. Alongside `spine audit stages` blocking on a missing/stale cursor block in the newest handoff, `spine doctor` advises (warns) on the same condition.": true,

	// gen 6–9 WORKFLOW.md model_routing block, superseded in gen 10 (I036,
	// design D8/D16) by the flavor-axis dotted mirror rendered from the model
	// table: the uncommented block header, the four bare claude tier rows
	// (both fallback values gen 9 ever emitted — pre- and post-I035), and the
	// retired top-level effort: and model_default: keys. The retired keys'
	// emitted spellings are listed so they read as machine-emitted, never as
	// local edits (I036 AC); customized values of the retired keys are
	// handled by planWorkflow's retirement pass, not by this set.
	"model_routing:": true,
	"  primary: claude-fable-5          # default thinker: design, judgment, orchestration, final review":  true,
	"  routine: claude-sonnet-5         # multi-step mechanical subagent roles":                            true,
	"  mechanical: claude-haiku-4-5     # verbatim plan-transcription + single-file mechanical fixes ONLY": true,
	"  fallback: claude-opus-4-8        # primary-refused or security-framed work":                         true,
	"  fallback: claude-opus-5          # primary-refused or security-framed work":                         true,
	"effort: high                       # tier default: primary=high, routine=medium, mechanical=low, fallback=high; xhigh reserved for final verification and security-critical passes; per-ticket effort: only on deviation": true,
	"model_default: claude-fable-5      # swappable; re-evaluate on major model/platform releases":                                                                                                                             true,
	"model_default: claude-opus-4-8     # swappable; re-evaluate on major model/platform releases":                                                                                                                             true,

	// docs/issues/README.md frontmatter bullets, backfilled by I065 from
	// the full history of templates/current/issues-README.md. All three
	// were reworded in place (no generation bump rode them), so a pristine
	// repo carrying any of them read as locally edited and was skipped.
	// The gen-0..10 status bullet, extended in d78f6ee to document the
	// superseded status value.
	"- `status` — open | in-progress | fixed | wontfix": true,
	// The gen-6..10 tier bullet, extended in c55ffb3 (I046) to document the
	// tier: n/a routing exemption.
	"- `tier` — primary | routine | mechanical | fallback; the model tier the work is dispatched at": true,
	// The gen-6 review-tier bullet as first shipped in ee0d0b3 (I003),
	// extended within the same generation by 3dacdde to document
	// review-tier: n/a for inline tickets.
	"- `review-tier` — the tier review runs at; never below `tier`": true,
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
		k, sig, ok := keyLineSignature(l)
		if !ok {
			return
		}
		// A bare tier key is carry-forwardable exactly when the current
		// generation renders its claude-flavored dotted successor: the
		// gen ≤9 bare rows are claude rows by definition (the transitional
		// affordance in internal/model), and applyModelRouting writes their
		// values into the dotted mirror rather than dropping them.
		if currentKeys[k] || (isRoutingKey(k) && currentKeys["claude."+k]) {
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
// comment" line — a top-level key, a two-space-indented model_routing
// sub-key, or a gen-10 dotted "<flavor>.<tier>" mirror key — with the value
// dropped and the comment kept verbatim, plus the bare key so callers can
// gate on which keys the current generation still renders. ok is false for
// anything that isn't a recognized key: value line (prose, headers, unknown
// keys), which keeps exact-text comparison for those.
func keyLineSignature(line string) (key, sig string, ok bool) {
	trimmed := strings.TrimSpace(line)
	for _, k := range topKeys {
		if _, has := cutKey(trimmed, k); has {
			return k, k + "\x00" + commentOf(trimmed, k), true
		}
	}
	for _, k := range model.Tiers {
		if _, has := cutKey(trimmed, k); has {
			return k, k + "\x00" + commentOf(trimmed, k), true
		}
	}
	for _, k := range gatePackConfigKeys {
		if _, has := cutKey(trimmed, k); has {
			dotted := "gate_pack_config." + k
			return dotted, dotted + "\x00" + commentOf(trimmed, k), true
		}
	}
	if dk, has := model.DottedRoutingKey(trimmed); has {
		return dk, dk + "\x00" + commentOf(trimmed, dk), true
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
