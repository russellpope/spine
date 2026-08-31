// Package doctor runs read-only workflow health checks (spec D1–D15).
package doctor

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/russellpope/spine/internal/adr"
	"github.com/russellpope/spine/internal/cursor"
	"github.com/russellpope/spine/internal/eval"
	"github.com/russellpope/spine/internal/handoff"
	"github.com/russellpope/spine/internal/hostconfig"
	"github.com/russellpope/spine/internal/model"
	"github.com/russellpope/spine/internal/stages"
	"github.com/russellpope/spine/internal/tmpl"
	"github.com/russellpope/spine/internal/update"
)

// Finding is one doctor result; Severity is error | warn | info.
type Finding struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Path     string `json:"path"`
	Message  string `json:"message"`
}

// Run executes all checks. It never writes.
func Run(dir string) ([]Finding, error) {
	return runWithHostPath(dir, "", nil)
}

// runWithHostPath is the race-safe test seam for the local, owner-managed
// capability file. The public command has no host path flag or environment
// override.
func runWithHostPath(dir, hostPath string, lookup func(string) (string, error)) ([]Finding, error) {
	required := []string{"WORKFLOW.md", "CLAUDE.md", "docs/harness-interface.md",
		"docs/specs", "docs/adr", "docs/issues", "docs/handoffs"}
	if raw, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md")); err == nil {
		if p := update.ExtractKeys(string(raw))["profile"]; p != "" {
			if _, _, err := tmpl.Defaults(p); err == nil {
				required = []string{"WORKFLOW.md", "CLAUDE.md"}
				required = append(required, tmpl.ProfileDirs(p)...)
				if tmpl.ProfileOwns(p, "docs/harness-interface.md") {
					required = append(required, "docs/harness-interface.md")
				}
			}
		}
	}

	findings := []Finding{}
	missingCore := false
	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			findings = append(findings, Finding{"D1", "error", rel, "missing — run spine init"})
			if rel == "WORKFLOW.md" {
				missingCore = true
			}
		}
	}
	if !missingCore {
		findings = append(findings, updateChecks(dir)...)
	}
	for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
		findings = append(findings, markerCheck(dir, name)...)
	}
	findings = append(findings, superpowersCheck(dir)...)
	findings = append(findings, adrCheck(dir)...)
	findings = append(findings, evalCheck(dir)...)
	findings = append(findings, handoffCheck(dir)...)
	findings = append(findings, stagesCheck(dir)...)
	findings = append(findings, gatePackCheck(dir)...)
	findings = append(findings, checkpointCheck(dir)...)
	findings = append(findings, slugCheck(dir)...)
	findings = append(findings, ticketCheck(dir)...)
	findings = append(findings, acceptanceCheck(dir)...)
	findings = append(findings, toolchainCheck()...)
	findings = append(findings, hostRoutingCheck(dir, hostPath, lookup)...)
	return findings, nil
}

// hostRoutingCheck is D16. It evaluates repository preferences against each
// declared available harness without changing those preferences or inferring
// alternate routes. A host pin is trusted local authority, but a divergent
// ID remains non-launchable under controlled validation until I074.
func hostRoutingCheck(repoDir, hostPath string, lookup func(string) (string, error)) []Finding {
	normalizedRepo, err := filepath.Abs(repoDir)
	if err != nil {
		normalizedRepo = filepath.Clean(repoDir)
	}
	path := hostPath
	if path == "" {
		var err error
		path, err = hostconfig.DefaultPath()
		if err != nil {
			return []Finding{{"D16", "error", "routing-host.json", "cannot locate host routing configuration"}}
		}
	}
	if lookup == nil {
		lookup = exec.LookPath
	}
	config, err := hostconfig.Load(path, model.Harnesses(), lookup)
	if errors.Is(err, hostconfig.ErrNotConfigured) {
		return nil
	}
	if err != nil {
		return []Finding{{"D16", "error", path, err.Error()}}
	}
	names := make([]string, 0, len(config.Harnesses))
	for name, harness := range config.Harnesses {
		if harness.Available {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var findings []Finding
	for _, harnessName := range names {
		harness := config.Harnesses[harnessName]
		for _, tier := range model.Tiers {
			requested, err := model.Resolve(repoDir, harnessName, tier)
			if err != nil {
				findings = append(findings, Finding{"D16", "error", path, fmt.Sprintf("cannot resolve %s.%s preference", harnessName, tier)})
				return append(findings, pinEvidenceCheck(repoDir, config)...)
			}
			key := harnessName + "." + tier
			if pin, pinned := config.Pins[key]; pinned {
				if pin.Model != requested.ID {
					findings = append(findings, Finding{"D16", "warn", path, fmt.Sprintf("repository %s %s requested %s@%s has a divergent host pin and is not auditable until I074", normalizedRepo, key, requested.ID, requested.Effort)})
				}
				continue
			}
			route, reachable := harness.Models[requested.ID]
			if !reachable || !hostRouteContains(route.Efforts, requested.Effort) {
				findings = append(findings, Finding{"D16", "warn", path, fmt.Sprintf("repository %s %s requested %s@%s is not reachable on available harness", normalizedRepo, key, requested.ID, requested.Effort)})
			}
		}
	}
	return append(findings, pinEvidenceCheck(repoDir, config)...)
}

// pinEvidenceCheck formats only the typed, redacted result of eval's narrow
// selected-run reader. It is intentionally advisory: config loading,
// resolution, and every non-doctor command remain unaware of these findings.
func pinEvidenceCheck(repoDir string, config hostconfig.Config) []Finding {
	pins := make([]eval.PinEvidencePin, 0, len(config.Pins))
	for key, pin := range config.Pins {
		pins = append(pins, eval.PinEvidencePin{Key: key, Model: pin.Model, EvidenceRefs: pin.EvidenceRefs})
	}
	results := eval.CheckPinEvidence(repoDir, pins, time.Now().UTC())
	findings := make([]Finding, 0, len(results))
	for _, result := range results {
		message := ""
		switch result.Kind {
		case eval.PinEvidenceNoReference:
			message = fmt.Sprintf("pin %s has no eligible eval reference", result.PinKey)
		case eval.PinEvidenceBadReference:
			message = fmt.Sprintf("pin %s has a malformed eval reference", result.PinKey)
		case eval.PinEvidenceMissing:
			message = fmt.Sprintf("pin %s references missing eval evidence", result.PinKey)
		case eval.PinEvidenceMalformed:
			message = fmt.Sprintf("pin %s references malformed eval evidence", result.PinKey)
		case eval.PinEvidenceStale:
			message = fmt.Sprintf("pin %s references stale eval evidence", result.PinKey)
		case eval.PinEvidenceModelMismatch:
			message = fmt.Sprintf("pin %s eval model does not exactly match pinned model", result.PinKey)
		case eval.PinEvidenceNoBattery:
			message = fmt.Sprintf("pin %s eval evidence has no battery record", result.PinKey)
		case eval.PinEvidenceFailedBattery:
			message = fmt.Sprintf("pin %s eval battery verdict is fail", result.PinKey)
		}
		if message != "" {
			findings = append(findings, Finding{"D17", "warn", result.Path, message})
		}
	}
	return findings
}

func hostRouteContains(efforts []string, want string) bool {
	for _, effort := range efforts {
		if effort == want {
			return true
		}
	}
	return false
}

// stagesCheck is the I019 advisory: it reuses the internal/stages
// derivation engine (the same one spine audit stages blocks on) but only
// ever reports warn — a stage/artifact contradiction, a stale
// newest-handoff, or (I024) grammar findings on the cursor block itself is
// drift worth surfacing, never a doctor failure on its own beyond the
// existing warn-affects-exit rule every other check already follows. A
// dormant repo (no cursor at all) reports nothing: absence of an active
// effort is not unhealthy.
func stagesCheck(dir string) []Finding {
	rep, err := stages.Derive(dir)
	if err != nil {
		return []Finding{{"D9", "warn", ".superpowers/sdd/progress.md", "stage derivation failed: " + err.Error()}}
	}
	if !rep.HasCursor {
		return nil
	}
	var findings []Finding
	if len(rep.CursorFindings) > 0 {
		// I024: grammar-level findings on the cursor block itself (e.g. a
		// stages: line that parses to zero stage rows) previously never
		// surfaced through doctor at all — only spine audit stages caught
		// them, and only there as a block. D9 stays warn-only here, same as
		// every other check this function reports.
		findings = append(findings, Finding{"D9", "warn", ".superpowers/sdd/progress.md",
			"cursor block malformed — grammar findings: " + strings.Join(rep.CursorFindings, "; ")})
	}
	if rep.CursorNonCanonical {
		findings = append(findings, Finding{"D9", "warn", ".superpowers/sdd/progress.md", cursor.NonCanonicalRemediation})
	}
	for _, n := range rep.Notes {
		// F1 (final whole-branch review, I024-I027 batch): rep.Notes (e.g.
		// an unresolvable tickets: value, I026) previously never reached
		// doctor — same warn-only D9 treatment as every other check here.
		findings = append(findings, Finding{"D9", "warn", ".superpowers/sdd/progress.md", n})
	}
	for _, s := range rep.Stages {
		if s.Verdict == stages.VerdictTickedMissing || s.Verdict == stages.VerdictPresentUnticked {
			findings = append(findings, Finding{"D9", "warn", ".superpowers/sdd/progress.md",
				fmt.Sprintf("stage %q (%s): %s", s.Name, s.Verdict, s.Detail)})
		}
	}
	if rep.Handoff.Blocking() {
		findings = append(findings, Finding{"D9", "warn", "docs/handoffs", rep.Handoff.Detail})
	}
	return findings
}

// updateChecks maps a dry-run of update onto D2 (stale) and D4 (unrecognized).
// D4 is not warn-only: alongside its info/warn arms it carries an error arm for
// unrecognized content update refuses to render at all — today, an unknown
// gate_pack: value in WORKFLOW.md (I099).
func updateChecks(dir string) []Finding {
	var findings []Finding
	reports, err := update.Run(update.Options{Dir: dir})
	if err != nil {
		return []Finding{{"D2", "error", "WORKFLOW.md", "update cannot run: " + err.Error()}}
	}
	for _, r := range reports {
		if r.Path == update.MaipipeFile {
			// The gate pack's delivery region is D10's, whole: one finding
			// per problem, phrased for a region rather than a whole file.
			// The one exception (I099) is the unknown-pack report: update
			// files it against maipipe.toml because that is the file it
			// declined to render, but the defect and its remedy are both a
			// WORKFLOW.md gate_pack: value, so it is D4's, on WORKFLOW.md.
			if unknownGatePack(r) {
				findings = append(findings, Finding{"D4", "error", "WORKFLOW.md", r.Unrecognized[0]})
			}
			continue
		}
		if r.Preserved {
			findings = append(findings, Finding{"D4", "info", r.Path,
				"hand-authored file preserved — spine update --force regenerates from template"})
			continue
		}
		switch r.State {
		case update.Pending:
			msg := "behind template generation — run spine update"
			if r.Created {
				msg = "missing — spine update will create it"
			}
			findings = append(findings, Finding{"D2", "warn", r.Path, msg})
		case update.SkippedUnrecognized:
			msg := fmt.Sprintf("%d unrecognized local edit(s) in a machine-owned file — reconcile or spine update --force", len(r.Unrecognized))
			if (r.Path == "CLAUDE.md" || r.Path == "AGENTS.md") && len(r.Unrecognized) > 0 && strings.Contains(r.Unrecognized[0], "marker") {
				// --force deliberately cannot repair marker damage (unrecognized
				// with no newContent); the generic --force hint is actively wrong here.
				msg = "spine markers damaged — fix by hand (--force cannot repair)"
			}
			findings = append(findings, Finding{"D4", "warn", r.Path, msg})
		}
	}
	return findings
}

// unknownGatePack reports whether a maipipe.toml update report is the
// unknown-pack refusal — WORKFLOW.md pinned a gate_pack: this binary does
// not ship, so update rendered no region at all. It is recognised by the
// message prefix update stamps on it; no other unrecognized line for this
// file starts with a WORKFLOW.md key name.
func unknownGatePack(r update.FileReport) bool {
	return r.Path == update.MaipipeFile && len(r.Unrecognized) > 0 &&
		strings.HasPrefix(r.Unrecognized[0], "gate_pack:")
}

// gatePackCheck is D10: the integrity of spine's machine-managed gate-pack
// region in maipipe.toml (ADR 0017, superseding 0016) — markers present and
// well formed, and content canonical for the pinned pack version. It fires
// only when the repo sets gate_pack; a repo without a pack has no region and
// no maipipe.toml to answer for (the fleet negative control). A canonical
// region is silent.
// An unknown gate_pack: value is D4's (I099), but a region already on disk
// is stale executable state and remains D10's concern (I097).
func gatePackCheck(dir string) []Finding {
	wf, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		return nil // D1 already reported it
	}
	reports, err := update.Run(update.Options{Dir: dir})
	if err != nil {
		return nil // D2 already reported that update cannot run
	}
	if update.ExtractKeys(string(wf))["gate_pack"] == "" {
		return nil
	}
	var findings []Finding
	for _, r := range reports {
		if r.Path != update.MaipipeFile {
			continue
		}
		switch {
		case unknownGatePack(r):
			inspection := inspectGatePackRegion(dir)
			switch {
			case inspection.Err != nil:
				findings = append(findings, Finding{"D10", "error", update.MaipipeFile, inspection.Err.Error()})
			case inspection.Present:
				findings = append(findings, Finding{"D10", "warn", update.MaipipeFile,
					"gate_pack is not shipped but maipipe.toml still carries a stale gate-pack region — choose a shipped pack or clear its repo-owned composing stages and opt out"})
			}
		case len(r.Unrecognized) > 0:
			sev := "warn"
			msg := fmt.Sprintf("%d line(s) in the spine gate-pack region are not canonical for the pinned pack — reconcile or spine update --force", len(r.Unrecognized))
			if strings.Contains(r.Unrecognized[0], "markers") {
				// --force deliberately cannot repair marker damage.
				sev, msg = "error", r.Unrecognized[0]
			}
			findings = append(findings, Finding{"D10", sev, r.Path, msg})
		case (r.State == update.Pending || r.State == update.SkippedPreflight) && r.Created:
			findings = append(findings, Finding{"D10", "warn", r.Path,
				"gate_pack is set but the gate-pack region is missing — run spine update"})
		case r.State == update.Pending || r.State == update.SkippedPreflight:
			// The region differs from what this pack renders from the current
			// WORKFLOW.md. Two causes reach here — WORKFLOW.md changed and the
			// region has not been refreshed, or the region was edited into some
			// other shape this pack does not render — and the message does not
			// distinguish them because doctor cannot: under I095 reading (A) the
			// region is a pure projection of WORKFLOW.md with deliberately no
			// record of what spine last rendered (no fingerprint, no sidecar),
			// and telling the two apart needs exactly that record. The remedy is
			// the same either way, so the message names the remedy, not a cause.
			findings = append(findings, Finding{"D10", "warn", r.Path,
				"gate-pack region differs from what the pinned pack renders — run spine update"})
		}
	}
	return findings
}

func inspectGatePackRegion(dir string) update.GateRegionInspection {
	raw, err := os.ReadFile(filepath.Join(dir, update.MaipipeFile))
	if os.IsNotExist(err) {
		return update.GateRegionInspection{}
	}
	if err != nil {
		return update.GateRegionInspection{Err: err}
	}
	return update.InspectGateRegion(string(raw))
}

func markerCheck(dir, name string) []Finding {
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return nil // D1 already reported it
	}
	content := string(raw)
	beginMarker, endMarker := "<!-- spine:begin", "<!-- spine:end -->"
	begins := strings.Count(content, beginMarker)
	ends := strings.Count(content, endMarker)
	switch {
	case begins == 0 && ends == 0:
		return []Finding{{"D3", "info", name, "no spine markers (legacy file) — spine update will claim it"}}
	case begins == 1 && ends == 1:
		// counts alone don't catch a swapped pair — compare positions too.
		if strings.Index(content, endMarker) < strings.Index(content, beginMarker) {
			return []Finding{{"D3", "error", name, "spine markers out of order — fix by hand"}}
		}
		return nil
	default:
		return []Finding{{"D3", "error", name,
			fmt.Sprintf("unbalanced spine markers (%d begin / %d end) — fix by hand", begins, ends)}}
	}
}

func superpowersCheck(dir string) []Finding {
	var findings []Finding
	for _, sub := range []string{"specs", "plans"} {
		glob := filepath.Join(dir, "docs", "superpowers", sub, "*.md")
		if m, _ := filepath.Glob(glob); len(m) > 0 {
			findings = append(findings, Finding{"D5", "info", "docs/superpowers/" + sub,
				fmt.Sprintf("%d artifact(s) in legacy location — new work goes to docs/specs/", len(m))})
		}
	}
	return findings
}

func adrCheck(dir string) []Finding {
	entries, err := adr.List(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil // no docs/adr — D1 covers structural absence
		}
		return []Finding{{"D6", "error", "docs/adr", "adr ledger unreadable: " + err.Error()}}
	}
	var findings []Finding
	seen := map[int]bool{}
	for _, e := range entries {
		if seen[e.ID] {
			findings = append(findings, Finding{"D6", "error", e.Path, fmt.Sprintf("duplicate ADR number %04d", e.ID)})
		}
		seen[e.ID] = true
		if !e.HasFrontMatter {
			findings = append(findings, Finding{"D6", "info", e.Path,
				"pre-spine ADR (no front matter) — spine conventions apply to new ADRs"})
			continue
		}
		if e.Status != "Accepted" && !strings.HasPrefix(e.Status, "Superseded by ") {
			findings = append(findings, Finding{"D6", "warn", e.Path, fmt.Sprintf("invalid status %q", e.Status)})
		}
	}
	return findings
}

// evalCheck maps eval.List structural problems onto D7. Values (stage,
// score) are never validated — structure only (ADR 0007).
func evalCheck(dir string) []Finding {
	_, problems, err := eval.List(dir)
	if err != nil {
		return []Finding{{"D7", "error", "docs/evals", "evals tree unreadable: " + err.Error()}}
	}
	var findings []Finding
	for _, p := range problems {
		findings = append(findings, Finding{"D7", "warn", p.Path, p.Message})
	}
	return findings
}

// handoffCheck flags files in docs/handoffs that don't follow the
// YYYY-MM-DD-<topic>.md convention. Info only — legacy is legal.
func handoffCheck(dir string) []Finding {
	des, err := os.ReadDir(filepath.Join(dir, "docs", "handoffs"))
	if err != nil {
		return nil // D1 covers structural absence
	}
	var findings []Finding
	for _, de := range des {
		if de.IsDir() {
			continue
		}
		if _, _, ok := handoff.ParseName(de.Name()); !ok {
			findings = append(findings, Finding{"D8", "info", "docs/handoffs/" + de.Name(),
				"does not match YYYY-MM-DD-<topic>.md — spine handoff new produces conforming names"})
		}
	}
	return findings
}
