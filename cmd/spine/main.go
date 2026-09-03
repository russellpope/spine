// Command spine is the unified-workflow runtime companion.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/russellpope/spine/internal/adopt"
	"github.com/russellpope/spine/internal/adr"
	"github.com/russellpope/spine/internal/audit"
	"github.com/russellpope/spine/internal/checkpoint"
	"github.com/russellpope/spine/internal/cursor"
	"github.com/russellpope/spine/internal/doctor"
	"github.com/russellpope/spine/internal/eval"
	"github.com/russellpope/spine/internal/gate"
	"github.com/russellpope/spine/internal/handoff"
	"github.com/russellpope/spine/internal/hostconfig"
	"github.com/russellpope/spine/internal/model"
	"github.com/russellpope/spine/internal/scaffold"
	"github.com/russellpope/spine/internal/stages"
	"github.com/russellpope/spine/internal/tmpl"
	"github.com/russellpope/spine/internal/update"
	"github.com/russellpope/spine/internal/yield"
)

const usage = `usage: spine <command> [flags]

commands:
  init       scaffold the unified workflow into a repo
  adopt      retrofit a pre-spine repo (dry-run by default; --write applies)
  update     regenerate machine-owned workflow files (dry-run by default; --write applies)
  adr        manage architecture decision records (new, list)
  handoff    manage docs/handoffs (new, list, latest [--fleet DIR]); new embeds the
             cursor block and the newest checkpoint from the checkpoint working home
  eval       manage docs/evals (new, add-run, list)
  doctor     read-only workflow health checks
  audit      verify declared model routing (routing) or stage cursor derivation (stages) against on-disk artifacts
  gate       run a gate-pack check class (gate [--dir D] <pack>[@<v>] <check>)
  checkpoint write or replay a session checkpoint (new, latest, list)
  cursor     print or update the stage cursor (start | tick | here | set; --quiet for read hooks)
  model      resolve or validate the model table for a (harness, tier) pair (read-only)
  yield      report recorded routing yield (yield [--dir D] [--json] | yield --fleet P [--json])
  version    print the compiled template generation
`

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "init":
		return cmdInit(args[1:], stdout, stderr)
	case "update":
		return cmdUpdate(args[1:], stdout, stderr)
	case "adr":
		return cmdADR(args[1:], stdout, stderr)
	case "handoff":
		return cmdHandoff(args[1:], stdout, stderr)
	case "eval":
		return cmdEval(args[1:], stdout, stderr)
	case "doctor":
		return cmdDoctor(args[1:], stdout, stderr)
	case "adopt":
		return cmdAdopt(args[1:], stdout, stderr)
	case "audit":
		return cmdAudit(args[1:], stdout, stderr)
	case "gate":
		return cmdGate(args[1:], stdout, stderr)
	case "checkpoint":
		return cmdCheckpoint(args[1:], stdout, stderr)
	case "cursor":
		return cmdCursor(args[1:], stdout, stderr)
	case "model":
		return cmdModel(args[1:], stdout, stderr)
	case "yield":
		return cmdYield(args[1:], stdout, stderr)
	case "version":
		fmt.Fprintf(stdout, "spine template generation %d\n", tmpl.Version())
		bi, ok := debug.ReadBuildInfo()
		if !ok {
			bi = nil
		}
		fmt.Fprintln(stdout, buildLine(bi))
		return 0
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n%s", args[0], usage)
		return 2
	}
}

const yieldUsage = `usage: spine yield [--dir D] [--json]
       spine yield --fleet P [--json]`

func cmdYield(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("yield", flag.ContinueOnError)
	dir := fs.String("dir", ".", "repository root")
	fleet := fs.String("fleet", "", "scan immediate primary child repositories")
	asJSON := fs.Bool("json", false, "machine-readable output")
	if _, ok := parseArgs(fs, args, "yield", yieldUsage, 0, stderr); !ok {
		return 2
	}
	dirSet, fleetSet := false, false
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "dir":
			dirSet = true
		case "fleet":
			fleetSet = true
		}
	})
	if (dirSet && *dir == "") || (fleetSet && *fleet == "") {
		fmt.Fprintf(stderr, "yield: --dir or --fleet needs a directory value\n%s\n", yieldUsage)
		return 2
	}
	if *fleet != "" && dirSet {
		fmt.Fprintf(stderr, "yield: --dir and --fleet are mutually exclusive\n%s\n", yieldUsage)
		return 2
	}
	for _, f := range []struct{ name, value string }{{"dir", *dir}, {"fleet", *fleet}} {
		if strings.HasPrefix(f.value, "-") {
			fmt.Fprintf(stderr, "yield: --%s needs a directory value (did a following flag get consumed?)\n%s\n", f.name, yieldUsage)
			return 2
		}
	}
	report, err := yield.Run(yield.Options{Dir: *dir, Fleet: *fleet})
	if err != nil {
		fmt.Fprintln(stderr, "yield:", err)
		return 2
	}
	if *asJSON {
		if err := json.NewEncoder(stdout).Encode(report); err != nil {
			fmt.Fprintln(stderr, "yield:", err)
			return 2
		}
	} else {
		printYieldReport(stdout, report)
	}
	return report.ExitCode()
}

func printYieldReport(stdout io.Writer, report yield.Report) {
	if len(report.Cells) == 0 {
		fmt.Fprintf(stdout, "scope: %s rate=%s confidence=%s\n", report.Scope, report.Rate, report.Confidence)
	} else {
		fmt.Fprintf(stdout, "scope: %s\n", report.Scope)
	}
	totals := report.Totals
	fmt.Fprintf(stdout, "totals: valid_review_lines=%d ignored_identities=%d escalations=%d fallbacks=%d final_accepted=%d final_needs_fixes=%d final_unattributable_needs_fixes=%d\n",
		totals.ValidReviewLines, totals.IgnoredIdentities, totals.Escalations, totals.Fallbacks,
		totals.FinalAccepted, totals.FinalNeedsFixes, totals.FinalUnattributableNeedsFixes)
	for _, cell := range report.Cells {
		fmt.Fprintf(stdout, "cell: harness=%s model_id=%s tier=%s n=%d accepted_first_pass=%d needs_fixes_first_pass=%d rework_rounds=%d rate=%s confidence=%s\n",
			cell.Harness, cell.ModelID, cell.Tier, cell.N, cell.AcceptedFirstPass, cell.NeedsFixesFirstPass,
			cell.ReworkRounds, cell.Rate, cell.Confidence)
	}
	for _, repository := range report.Repositories {
		fmt.Fprintf(stdout, "repository: %s status=%s\n", repository.Name, repository.Status)
	}
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Repository != "" {
			fmt.Fprintf(stdout, "diagnostic: repository=%s", diagnostic.Repository)
		} else {
			fmt.Fprint(stdout, "diagnostic:")
		}
		if diagnostic.Line != 0 {
			fmt.Fprintf(stdout, " line=%d", diagnostic.Line)
		}
		fmt.Fprintf(stdout, " message=%s\n", diagnostic.Message)
	}
}

func cmdInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	profile := fs.String("profile", "", "profile: "+strings.Join(tmpl.Profiles(), " | "))
	dir := fs.String("dir", ".", "repo root")
	name := fs.String("name", "", "project name (default: basename of dir)")
	owner := fs.String("owner", "", "owner for maikanban.repositorySlug (default: parsed from origin)")
	if _, ok := parseArgs(fs, args, "init", `usage: spine init [--profile P] [--dir D] [--name N] [--owner O]`, 0, stderr); !ok {
		return 2
	}
	p := *profile
	if p == "" {
		detected, ok := scaffold.DetectProfile(*dir)
		if !ok {
			fmt.Fprintln(stderr, "cannot detect profile; pass --profile")
			return 2
		}
		p = detected
	}
	res, err := scaffold.Init(*dir, p, *name)
	if err != nil {
		fmt.Fprintln(stderr, "init:", err)
		return 2
	}
	// I094: fleet identity for maikanban, stamped once and never overwritten.
	if *owner != "" && !scaffold.ValidOwner(*owner) {
		// Phrased as a property of the value, not as a cause: origin may well
		// have named the slug, in which case the flag was never consulted.
		fmt.Fprintf(stderr, "init: --owner %q is not a valid slug component and cannot be used\n", *owner)
	}
	slug, noted := scaffold.StampSlug(*dir, *owner)
	if slug != "" {
		res.Created = append(res.Created, "git config "+scaffold.SlugKey+" "+slug)
	}
	for _, f := range res.Created {
		fmt.Fprintln(stdout, "create:", f)
	}
	for _, f := range res.Skipped {
		fmt.Fprintln(stdout, "skip (exists):", f)
	}
	if noted {
		fmt.Fprintln(stdout, scaffold.SlugNote)
	}
	fmt.Fprintf(stdout, "done: %s -> %s (template_version %d)\n", p, *dir, tmpl.Version())
	return 0
}

func cmdUpdate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	dir := fs.String("dir", ".", "repo root")
	write := fs.Bool("write", false, "apply changes (default: dry-run diff)")
	force := fs.Bool("force", false, "regenerate files with unrecognized local edits (diff shows what gets dropped)")
	var forceFiles []string
	fs.Func("force-file", "regenerate only this managed file when it has unrecognized local edits (repeatable)", func(path string) error {
		forceFiles = append(forceFiles, path)
		return nil
	})
	var parseDiagnostic bytes.Buffer
	if _, ok := parseArgs(fs, args, "update", `usage: spine update [--dir D] [--write] [--force-file PATH]... [--force]`, 0, &parseDiagnostic); !ok {
		writeUpdateDiagnostic(stderr, parseDiagnostic.String())
		return 2
	}
	opts := update.Options{Dir: *dir, Write: *write, Force: *force, ForceFiles: forceFiles}
	if *write {
		opts.BeforeWrite = func(advisories []update.GateConfigAdvisory) {
			warnDirty(*dir, stderr)
			printGateConfigAdvisories(stdout, advisories)
		}
	}
	reports, err := update.Run(opts)
	if err != nil {
		writeUpdateDiagnostic(stderr, err.Error()+"\n")
		return 2
	}
	if !*write {
		for _, r := range reports {
			printGateConfigAdvisories(stdout, r.GateConfigAdvisories)
		}
	}
	outstanding := 0
	// itemized model-table results (design D6): each inherited refresh names
	// old and new value so a model change is never buried in template prose
	// churn; preserved overrides are reported so a pinned model is visible,
	// and overrides this migration mints from a retired effort: key say so —
	// "preserved" would misstate what the plan is about to create (I036
	// review Important 2). Skipped files print neither — nothing about them
	// is applied.
	modelNotes := func(r update.FileReport) {
		for _, m := range r.ModelRefreshes {
			kind := "inherited"
			if m.Retired {
				// I128: a deliberate override on a retired id migrates to
				// the successor keeping its effort — a refresh, not a
				// preserved override, and not the inherited kind either.
				kind = "retired override"
			}
			fmt.Fprintf(stdout, "model refresh (%s): %s: %s -> %s\n", kind, m.Key, m.Old, m.New)
		}
		for _, o := range r.ModelOverrides {
			if o.Migrated {
				fmt.Fprintf(stdout, "model override created (migrated from retired effort:): %s: %s\n", o.Key, o.Value)
			} else {
				fmt.Fprintf(stdout, "model override preserved: %s: %s\n", o.Key, o.Value)
			}
		}
	}
	// gate-pack stage churn (I098): an added stage is a new step in a gating
	// lane, and any stage change rewrites the region's bytes and so maipipe's
	// definition_hash — both are things to see before --write, not after.
	// Skipped files print neither: nothing about them is applied.
	stageNotes := func(r update.FileReport) {
		if len(r.StagesAdded) > 0 {
			fmt.Fprintf(stdout, "%s: this render adds %d stage(s) not previously present: %s\n",
				r.Path, len(r.StagesAdded), strings.Join(r.StagesAdded, ", "))
		}
		if len(r.StagesRemoved) > 0 {
			fmt.Fprintf(stdout, "%s: this render drops %d stage(s) present today: %s\n",
				r.Path, len(r.StagesRemoved), strings.Join(r.StagesRemoved, ", "))
		}
		if len(r.StagesChanged) > 0 {
			fmt.Fprintf(stdout, "%s: this render changes %d stage(s) not added or removed: %s\n",
				r.Path, len(r.StagesChanged), strings.Join(r.StagesChanged, ", "))
		}
		if len(r.StagesAdded) > 0 || len(r.StagesRemoved) > 0 || len(r.StagesChanged) > 0 {
			fmt.Fprintf(stdout, "%s: the region's bytes change, so maipipe's definition_hash does too — re-run `maipipe gate approve-definition`\n", r.Path)
		}
	}
	// The pre-flight's verdict belongs on the plan, not only on the write
	// (final review, Important 1): the diff is the review surface, so the one
	// thing that will stop --write is printed with it rather than sprung
	// afterwards. On a pass, state which authority preflight ran.
	preflightNotes := func(r update.FileReport) {
		if r.Refusal != "" {
			fmt.Fprintln(stdout, r.Refusal)
			fmt.Fprintf(stdout, "%s: --write would refuse this plan as a whole — no files would be written until this is fixed\n", r.Path)
			return
		}
		if r.Preflight != "" {
			fmt.Fprintf(stdout, "%s: pre-flight: %s\n", r.Path, r.Preflight)
		}
	}
	// Run owns normalization and validation. Once it returns reports, the raw
	// values are safe to clean only for matching presentation to the exact
	// planned path they authorized.
	scopedAuthorization := func(r update.FileReport) bool {
		return r.SelectedByForceFile && r.State == update.Pending && len(r.Unrecognized) > 0
	}
	for _, r := range reports {
		switch r.State {
		case update.UpToDate:
			if r.Preserved {
				fmt.Fprintf(stdout, "preserved (hand-authored): %s\n", r.Path)
			} else {
				fmt.Fprintf(stdout, "up-to-date: %s\n", r.Path)
			}
			modelNotes(r)
		case update.Pending:
			if *write {
				if scopedAuthorization(r) {
					fmt.Fprintf(stdout, "%s: local edits will be overwritten (authorized by --force-file %s)\n", r.Path, r.Path)
				}
				if r.Created {
					fmt.Fprintf(stdout, "created: %s\n", r.Path)
				} else {
					fmt.Fprintf(stdout, "updated: %s\n", r.Path)
				}
				modelNotes(r)
				stageNotes(r)
				preflightNotes(r)
			} else {
				outstanding++
				modelNotes(r)
				stageNotes(r)
				preflightNotes(r)
				if scopedAuthorization(r) {
					fmt.Fprintf(stdout, "%s: local edits will be overwritten (authorized by --force-file %s)\n", r.Path, r.Path)
				}
				fmt.Fprint(stdout, r.Diff)
			}
		case update.SkippedUnrecognized:
			outstanding++
			if r.SelectedByForceFile {
				fmt.Fprintf(stderr, "skipped %s — unrecognized local edits (manual repair required):\n", r.Path)
			} else if len(forceFiles) > 0 {
				fmt.Fprintf(stderr, "skipped %s — unrecognized local edits (use --force-file %s to drop only this file, or --force to drop all):\n", r.Path, r.Path)
			} else {
				fmt.Fprintf(stderr, "skipped %s — unrecognized local edits (use --force to drop):\n", r.Path)
			}
			for _, l := range r.Unrecognized {
				fmt.Fprintf(stderr, "  %s\n", l)
			}
		case update.SkippedPreflight:
			fmt.Fprintf(stdout, "skipped %s — %s\n", r.Path, r.Preflight)
		}
	}
	if outstanding > 0 {
		return 1
	}
	return 0
}

// writeUpdateDiagnostic gives only update a stable one-prefix contract.
// update.Run labels validation errors, while flag parsing does not.
func writeUpdateDiagnostic(stderr io.Writer, diagnostic string) {
	if strings.HasPrefix(diagnostic, "update:") {
		fmt.Fprint(stderr, diagnostic)
		return
	}
	fmt.Fprint(stderr, "update: ", diagnostic)
}

func printGateConfigAdvisories(stdout io.Writer, advisories []update.GateConfigAdvisory) {
	for _, advisory := range advisories {
		fmt.Fprintf(stdout, "advisory: enabled gate class %q lacks required gate_pack_config.%s; configure gate_pack_config.%s or add %q to gate_pack_disabled\n",
			advisory.Class, advisory.Key, advisory.Key, advisory.Class)
	}
}

// warnDirty nudges the user to review post-write diffs with git; git being
// absent or dir not being a repo is fine and silent.
func warnDirty(dir string, stderr io.Writer) {
	out, err := exec.Command("git", "-C", dir, "status", "--porcelain").Output()
	if err == nil && len(bytes.TrimSpace(out)) > 0 {
		fmt.Fprintln(stderr, "warning: repo has uncommitted changes — review the update with git diff afterwards")
	}
}

// parseADRID parses an ADR id as base-10 regardless of zero-padding.
func parseADRID(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("--supersedes needs a value (an ADR id like 0011)")
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("--supersedes %q: ADR ids are base-10 digits only", s)
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("--supersedes %q: ADR id out of range", s)
	}
	if n == 0 {
		return 0, fmt.Errorf("--supersedes %q: ADR ids start at 0001", s)
	}
	return n, nil
}

func cmdADR(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, `usage: spine adr <new|list> [flags]  (adr new [--dir D] [--supersedes NNNN] "Title")`)
		return 2
	}
	switch args[0] {
	case "new":
		fs := flag.NewFlagSet("adr new", flag.ContinueOnError)
		dir := fs.String("dir", ".", "repo root")
		supersedes := fs.String("supersedes", "", "ADR id this decision supersedes (base-10; zero-padding ok)")
		pos, ok := parseArgs(fs, args[1:], "adr new", `usage: spine adr new [--dir D] [--supersedes NNNN] "Title" (flags before title)`, 1, stderr)
		if !ok {
			return 2
		}
		// I120: this went through flag.Int, whose base-0 parse read the
		// conventional zero-padded ids as octal ("0011" -> 9) and silently
		// flipped the wrong ADR. Parse base-10 ourselves. Visit (not a ""
		// sentinel) so an explicitly empty value errors instead of silently
		// meaning "no supersede".
		supSet := false
		fs.Visit(func(f *flag.Flag) {
			if f.Name == "supersedes" {
				supSet = true
			}
		})
		supN := 0
		if supSet {
			n, err := parseADRID(*supersedes)
			if err != nil {
				fmt.Fprintln(stderr, "adr new:", err)
				return 2
			}
			supN = n
		}
		path, err := adr.New(*dir, pos[0], supN)
		if err != nil {
			fmt.Fprintln(stderr, "adr new:", err)
			return 2
		}
		fmt.Fprintln(stdout, path)
		if supN > 0 {
			fmt.Fprintf(stdout, "superseded: %04d\n", supN)
		}
		return 0
	case "list":
		fs := flag.NewFlagSet("adr list", flag.ContinueOnError)
		dir := fs.String("dir", ".", "repo root")
		asJSON := fs.Bool("json", false, "machine-readable output")
		if _, ok := parseArgs(fs, args[1:], "adr list", `usage: spine adr list [--dir D] [--json]`, 0, stderr); !ok {
			return 2
		}
		entries, err := adr.List(*dir)
		if err != nil {
			fmt.Fprintln(stderr, "adr list:", err)
			return 2
		}
		if *asJSON {
			type entryJSON struct {
				ID             int    `json:"id"`
				Title          string `json:"title"`
				Status         string `json:"status"`
				Path           string `json:"path"`
				HasFrontMatter bool   `json:"has_front_matter"`
			}
			out := make([]entryJSON, 0, len(entries))
			for _, e := range entries {
				out = append(out, entryJSON{e.ID, e.Title, e.Status, e.Path, e.HasFrontMatter})
			}
			if err := json.NewEncoder(stdout).Encode(out); err != nil {
				fmt.Fprintln(stderr, "adr list:", err)
				return 2
			}
			return 0
		}
		for _, e := range entries {
			fmt.Fprintf(stdout, "%04d  %-22s  %s\n", e.ID, e.Status, e.Title)
		}
		return 0
	default:
		fmt.Fprintf(stderr, "unknown adr subcommand %q\n", args[0])
		return 2
	}
}

type handoffJSON struct {
	Path  string `json:"path"`
	Date  string `json:"date"`
	Topic string `json:"topic"`
	Title string `json:"title"`
}

func handoffToJSON(e handoff.Entry) handoffJSON {
	return handoffJSON{Path: e.Path, Date: e.Date.Format("2006-01-02"), Topic: e.Topic, Title: e.Title}
}

func cmdHandoff(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, `usage: spine handoff <new|list|latest> [flags]  (handoff new [--dir D] "Topic")`)
		return 2
	}
	switch args[0] {
	case "new":
		fs := flag.NewFlagSet("handoff new", flag.ContinueOnError)
		dir := fs.String("dir", ".", "repo root")
		pos, ok := parseArgs(fs, args[1:], "handoff new",
			`usage: spine handoff new [--dir D] "Topic" (flags before topic)`+"\n"+
				`  embeds the cursor block, then the newest checkpoint: facts region verbatim, model region under "Prior narrative (model-authored, not evidence)"`,
			1, stderr)
		if !ok {
			return 2
		}
		path, embeddedCursor, err := handoff.NewWithCursor(*dir, pos[0])
		if err != nil {
			fmt.Fprintln(stderr, "handoff new:", err)
			return 2
		}
		fmt.Fprintln(stdout, path)
		if !embeddedCursor {
			fmt.Fprintln(stdout, "note: no spine cursor found; scaffolded handoff without a cursor block")
		}
		return 0
	case "list":
		fs := flag.NewFlagSet("handoff list", flag.ContinueOnError)
		dir := fs.String("dir", ".", "repo root")
		asJSON := fs.Bool("json", false, "machine-readable output")
		if _, ok := parseArgs(fs, args[1:], "handoff list", `usage: spine handoff list [--dir D] [--json]`, 0, stderr); !ok {
			return 2
		}
		entries, err := handoff.List(*dir)
		if err != nil {
			fmt.Fprintln(stderr, "handoff list:", err)
			return 2
		}
		if *asJSON {
			out := make([]handoffJSON, 0, len(entries))
			for _, e := range entries {
				out = append(out, handoffToJSON(e))
			}
			if err := json.NewEncoder(stdout).Encode(out); err != nil {
				fmt.Fprintln(stderr, "handoff list:", err)
				return 2
			}
			return 0
		}
		w := len("topic")
		for _, e := range entries {
			if len(e.Topic) > w {
				w = len(e.Topic)
			}
		}
		fmt.Fprintf(stdout, "%-10s  %-*s  %s\n", "date", w, "topic", "path")
		for _, e := range entries {
			fmt.Fprintf(stdout, "%-10s  %-*s  %s\n", e.Date.Format("2006-01-02"), w, e.Topic, e.Path)
		}
		return 0
	case "latest":
		return cmdHandoffLatest(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown handoff subcommand %q\n", args[0])
		return 2
	}
}

func cmdHandoffLatest(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("handoff latest", flag.ContinueOnError)
	dir := fs.String("dir", ".", "repo root")
	asJSON := fs.Bool("json", false, "machine-readable output")
	fleet := fs.String("fleet", "", "scan every child repo of DIR instead of one repo")
	if _, ok := parseArgs(fs, args, "handoff latest", `usage: spine handoff latest [--dir D] [--json] [--fleet DIR]`, 0, stderr); !ok {
		return 2
	}
	for _, f := range []struct{ name, value string }{{"fleet", *fleet}, {"dir", *dir}} {
		if strings.HasPrefix(f.value, "-") {
			fmt.Fprintf(stderr, "handoff latest: --%s needs a directory value (did a following flag get consumed?)\n", f.name)
			return 2
		}
	}
	if *fleet != "" {
		return handoffFleet(*fleet, *asJSON, stdout, stderr)
	}
	e, ok, err := handoff.Latest(*dir)
	if err != nil {
		fmt.Fprintln(stderr, "handoff latest:", err)
		return 2
	}
	if !ok {
		fmt.Fprintln(stderr, "no handoffs found")
		return 1
	}
	if *asJSON {
		if err := json.NewEncoder(stdout).Encode(handoffToJSON(e)); err != nil {
			fmt.Fprintln(stderr, "handoff latest:", err)
			return 2
		}
		return 0
	}
	fmt.Fprintln(stdout, e.Path)
	return 0
}

func handoffFleet(parent string, asJSON bool, stdout, stderr io.Writer) int {
	rows, err := handoff.Fleet(parent)
	if err != nil {
		fmt.Fprintln(stderr, "handoff latest --fleet:", err)
		return 2
	}
	if asJSON {
		type row struct {
			Repo string `json:"repo"`
			handoffJSON
			AgeDays int `json:"age_days"`
		}
		out := make([]row, 0, len(rows))
		for _, r := range rows {
			out = append(out, row{Repo: r.Repo, handoffJSON: handoffToJSON(r.Entry), AgeDays: ageDays(r.Date)})
		}
		if err := json.NewEncoder(stdout).Encode(out); err != nil {
			fmt.Fprintln(stderr, "handoff latest --fleet:", err)
			return 2
		}
		return 0
	}
	for _, r := range rows {
		fmt.Fprintf(stdout, "%-24s %4dd  %s\n", r.Repo, ageDays(r.Date), r.Path)
	}
	return 0
}

// now is a seam for tests; production code always leaves it as time.Now.
var now = time.Now

// ageDays is a calendar-day difference. The handoff filename date is a plain
// local calendar date (handoff.New stamps time.Now().Format("2006-01-02"),
// handoff.go:52) that arrives parsed as UTC midnight; comparing instants
// against time.Now() made today's handoffs show "1d" west of UTC. Compare
// calendar dates instead.
func ageDays(d time.Time) int {
	n := now()
	today := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
	that := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	age := int(today.Sub(that).Hours() / 24)
	if age < 0 {
		return 0
	}
	return age
}

func cmdDoctor(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	dir := fs.String("dir", ".", "repo root")
	asJSON := fs.Bool("json", false, "machine-readable output")
	if _, ok := parseArgs(fs, args, "doctor", `usage: spine doctor [--dir D] [--json]`, 0, stderr); !ok {
		return 2
	}
	findings, err := doctor.Run(*dir)
	if err != nil {
		fmt.Fprintln(stderr, "doctor:", err)
		return 2
	}
	if *asJSON {
		payload := struct {
			Findings []doctor.Finding `json:"findings"`
		}{Findings: findings}
		if err := json.NewEncoder(stdout).Encode(payload); err != nil {
			fmt.Fprintln(stderr, "doctor:", err)
			return 2
		}
	} else if len(findings) == 0 {
		fmt.Fprintln(stdout, "ok — workflow healthy")
	} else {
		for _, f := range findings {
			fmt.Fprintf(stdout, "%s %-5s %s: %s\n", f.ID, f.Severity, f.Path, f.Message)
		}
	}
	// info findings do not affect exit code — only warn/error do.
	for _, f := range findings {
		if f.Severity == "warn" || f.Severity == "error" {
			return 1
		}
	}
	return 0
}

func cmdEval(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, `usage: spine eval <new|add-run|list> [flags]  (eval new [--dir D] "Title"; eval add-run --eval E --name N)`)
		return 2
	}
	switch args[0] {
	case "new":
		fs := flag.NewFlagSet("eval new", flag.ContinueOnError)
		dir := fs.String("dir", ".", "repo root")
		pos, ok := parseArgs(fs, args[1:], "eval new", `usage: spine eval new [--dir D] "Title" (flags before title)`, 1, stderr)
		if !ok {
			return 2
		}
		path, err := eval.New(*dir, pos[0])
		if err != nil {
			fmt.Fprintln(stderr, "eval new:", err)
			return 2
		}
		fmt.Fprintln(stdout, path)
		return 0
	case "add-run":
		fs := flag.NewFlagSet("eval add-run", flag.ContinueOnError)
		dir := fs.String("dir", ".", "repo root")
		evalRef := fs.String("eval", "", "eval dir name (date prefix optional)")
		name := fs.String("name", "", "run name (becomes runs/<name>.md)")
		if _, ok := parseArgs(fs, args[1:], "eval add-run", `usage: spine eval add-run [--dir D] --eval E --name N`, 0, stderr); !ok {
			return 2
		}
		if *evalRef == "" || *name == "" {
			fmt.Fprintln(stderr, "eval add-run: --eval and --name are required")
			return 2
		}
		path, err := eval.AddRun(*dir, *evalRef, *name)
		if err != nil {
			fmt.Fprintln(stderr, "eval add-run:", err)
			return 2
		}
		fmt.Fprintln(stdout, path)
		return 0
	case "list":
		fs := flag.NewFlagSet("eval list", flag.ContinueOnError)
		dir := fs.String("dir", ".", "repo root")
		asJSON := fs.Bool("json", false, "machine-readable output")
		if _, ok := parseArgs(fs, args[1:], "eval list", `usage: spine eval list [--dir D] [--json]`, 0, stderr); !ok {
			return 2
		}
		evals, problems, err := eval.List(*dir)
		if err != nil {
			fmt.Fprintln(stderr, "eval list:", err)
			return 2
		}
		for _, p := range problems {
			fmt.Fprintf(stderr, "warning: %s: %s\n", p.Path, p.Message)
		}
		if *asJSON {
			type runJSON struct {
				Name  string `json:"name"`
				Stage string `json:"stage"`
				Score string `json:"score"`
				Path  string `json:"path"`
			}
			type evalJSON struct {
				Name string    `json:"name"`
				Path string    `json:"path"`
				Runs []runJSON `json:"runs"`
			}
			out := make([]evalJSON, 0, len(evals))
			for _, e := range evals {
				ej := evalJSON{Name: e.Name, Path: e.Path, Runs: []runJSON{}}
				for _, r := range e.Runs {
					ej.Runs = append(ej.Runs, runJSON{Name: r.Name, Stage: r.Stage, Score: r.Score, Path: r.Path})
				}
				out = append(out, ej)
			}
			if err := json.NewEncoder(stdout).Encode(out); err != nil {
				fmt.Fprintln(stderr, "eval list:", err)
				return 2
			}
			return 0
		}
		fmt.Fprintf(stdout, "%-30s  %-20s  %-10s  %s\n", "eval", "run", "stage", "score")
		for _, e := range evals {
			if len(e.Runs) == 0 {
				fmt.Fprintf(stdout, "%-30s  %-20s  %-10s  %s\n", e.Name, "-", "-", "-")
			}
			for _, r := range e.Runs {
				fmt.Fprintf(stdout, "%-30s  %-20s  %-10s  %s\n", e.Name, r.Name, r.Stage, r.Score)
			}
		}
		return 0
	default:
		fmt.Fprintf(stderr, "unknown eval subcommand %q\n", args[0])
		return 2
	}
}

// cmdGate is a thin dispatcher over gate.Run: `spine gate [--dir D]
// <pack>[@<v>] <check>`. Flags precede the positionals like every other
// subcommand (I119 — the pre-parse positional read this replaced made
// `gate --dir X pack check` mis-read the pack as "--dir"). The pack owns
// the exit-code contract (0 pass, 1 findings, 2 misconfiguration), the
// results-contract emitter and the human table, so nothing but flag
// parsing lives here.
func cmdGate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gate", flag.ContinueOnError)
	dir := fs.String("dir", ".", "repo root to check")
	pos, ok := parseArgs(fs, args, "gate", gateUsage, 2, stderr)
	if !ok {
		return 2
	}
	return gate.Run(pos[0], pos[1], *dir, stdout, stderr, gate.EnvConfig())
}

// gateUsage documents the gate pack in CONTEXT.md vocabulary: a gate pack
// is a versioned battery of check classes; one invocation runs one check
// class and every finding is attributable to <pack>@<version>/<check>.
var gateUsage = `usage: spine gate [--dir D] <pack>[@<v>] <check>

pack:    ` + gate.PackName + ` (version ` + gate.PackID() + `)
checks:  ` + strings.Join(gate.CheckNames(), ", ") + `

  tskip              any t.Skip/t.Skipf/t.SkipNow (and b.Skip*) call in a
                     _test.go file under --dir. Zero tolerance by default;
                     allowlist via ` + gateTskipAllowVar + `, a comma-separated
                     list of entries, each either a path relative to --dir
                     (slash-separated) or path:line. Unset means no allowlist.
  binary-hygiene     tracked files (git ls-files) that are executables or
                     archives by content, plus stray second module trees (a
                     tracked go.mod outside the repo root). --dir must be a
                     git repo.
  gitignore-control  two arms, reported distinctly. Arm 1: every declared
                     build output in ` + gateBuildOutputsVar + ` (a
                     comma-separated list of paths relative to --dir, e.g.
                     bin/spine,dist/) is ignored at that path. Arm 2: no
                     package main source file under --dir is ignored — the
                     hidden-entry-point control. --dir must be a git repo;
                     an unset or empty list is misconfiguration.
  fixture-manifest   the manifest at ` + gateFixtureManifestVar + ` exists
                     and is non-empty. A missing or empty manifest is a
                     finding; content judgment is the evaluator's and is
                     never done here. An unset variable, or a manifest that
                     exists but cannot be read, is misconfiguration.
  test-enum-vs-spec  typed string enums in code vs the values enumerated in
                     the spec file at ` + gateTestEnumSpecVar + `, reporting
                     each side's extras. Code side: a const with an explicit
                     named type and a string literal value (const Low
                     Severity = "low"), outside _test.go files. Spec side:
                     every ` + "`backticked`" + ` token inside a marked block —
                     <!-- spine:enum TypeName --> … <!-- /spine:enum -->.
                     Only types a marker names are compared; a spec with no
                     marker is misconfiguration.
  deferred-cleanup-errcheck
                     deferred calls to cleanup-class functions whose error
                     return is discarded (defer f.Close()). A call is a
                     finding only when go/types confirms the callee returns
                     an error; a deferred func literal that inspects the
                     error is not. Default name set: ` + strings.Join(gate.DefaultCleanupFuncs, ", ") + `.
                     Extra names via ` + gate.CleanupFuncsVar + `, comma-
                     separated — an env-only tuning knob, not a
                     gate_pack_config key, so spine update never renders it.
  dead-code-callgraph
                     functions and methods (exported and unexported)
                     unreachable from any root. Roots: every main in a
                     package main, every init, every Test/Benchmark/Example/
                     Fuzz function in a _test.go file, every package-level
                     reference, and every exported function and method of
                     an importable package — non-main, non-test, and with no
                     "internal" element in its import path. Exported
                     declarations under internal/ are candidates like any
                     other: no other module can import them. Interface calls reach
                     every concrete method of that name, and a method is
                     live if its name is a method of any interface declared
                     in the module or in a package it imports (so a String
                     called only by fmt is live). Residual limitation: a
                     method reached only through an interface from a package
                     the module does not directly import is still reported.
                     Only declarations outside _test.go files are reported.
  n-plus-one         a call to one of the client names in
                     ` + gateNPlusOneClientsVar + ` (comma-separated method or
                     function names, e.g. Get,Query,Fetch) lexically inside
                     a for or range loop body, at any depth, outside
                     _test.go files. The list is required configuration: an
                     unset or empty list is misconfiguration.
  mutate             the behavioural mutation battery (advisory lane): each
                     probe of the per-tree mutation spec is applied to a
                     copy of the tracked tree, the verify command re-run,
                     and the outcome reported as one row — KILLED (the
                     suite went red), SURVIVED (a blind spot), NO-SITE (the
                     literal is not in the file: spec drift), BUILD-ERR
                     (the mutation broke compilation: an invalid probe).
                     Severities let a consumer filter without parsing:
                     SURVIVED is warn, every other row is info, and a
                     failed control is error. The tree under --dir is never
                     mutated. The unmutated negative control runs first: a
                     tree that is not green yields one finding and exit 1
                     and no probes run; otherwise the exit code is 0
                     whatever the survivors, and the summary carries both
                     kill rates (raw, and scorable with report-only probes
                     excluded). Spec path via ` + gate.MutateSpecVar + `
                     (relative to --dir or absolute), default ` + gate.DefaultMutateSpec + `;
                     a missing or unparseable spec is misconfiguration.
                     Verify command via ` + gate.MutateVerifyVar + ` (run
                     with sh -c in the copy; exit 0 is green), default
                     go build ./... then go test ./..., which is what tells
                     BUILD-ERR from KILLED. Per-command timeout via
                     ` + gate.MutateTimeoutVar + ` (a Go duration), default
                     15m. All three are env-only tuning knobs, not
                     gate_pack_config keys, so spine update never renders
                     them.

The syntactic classes (tskip, n-plus-one, test-enum-vs-spec, gitignore-
control) walk the tree under --dir and skip what the Go toolchain itself
ignores: .git, vendor, node_modules, testdata, and any directory whose name
starts with "_" or ".". gitignore-control's entry-point arm still walks
testdata, since an ignored testdata entry point is hidden just the same, but
a testdata file that does not parse is skipped rather than reported as
misconfiguration. binary-hygiene's stray-module rule likewise exempts paths
with a testdata element — "go build ./..." never descends there — while its
content detection of committed binaries applies under testdata as everywhere
else.

deferred-cleanup-errcheck and dead-code-callgraph type-check the module
under --dir (go list + go/types, stdlib only). A --dir that does not compile
is misconfiguration, not a finding: a gate cannot judge code the compiler
has not agreed to.

exit:    0 pass, 1 findings, 2 misconfiguration (mutate is advisory: only a
         failed control exits 1)
results: with ` + gate.ResultsEnvVar + ` set, the results contract is written
         there as JSON (maipipe_results, status, summary, findings[] each
         with severity, message, file, line, code = "` + gate.PackID() + `/<check>");
         otherwise a human table on stdout and no file is written.
`

// gateTskipAllowVar is the tskip allowlist variable, named by the gate-pack
// convention: SPINE_GATE_ + the upper-snake of the WORKFLOW.md
// gate_pack_config key.
var gateTskipAllowVar = gate.EnvVar("tskip_allow")

// The other gate_pack_config keys the pack's check classes read, named by
// the same convention.
var (
	gateBuildOutputsVar    = gate.EnvVar("build_outputs")
	gateFixtureManifestVar = gate.EnvVar("fixture_manifest")
	gateTestEnumSpecVar    = gate.EnvVar("test_enum_spec")
	gateNPlusOneClientsVar = gate.EnvVar("n_plus_one_clients")
)

func cmdAudit(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, `usage: spine audit <routing|stages> [flags]  (audit routing [--dir D] [--transcripts DIR] [--codex-sessions DIR] [--since TIME] [--session ID]; audit stages [--dir D])`)
		return 2
	}
	switch args[0] {
	case "routing":
		return cmdAuditRouting(args[1:], stdout, stderr)
	case "stages":
		return cmdAuditStages(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown audit subcommand %q\n", args[0])
		return 2
	}
}

// cmdAuditRouting is a thin printer over audit.Run: table to stdout,
// warnings to stderr, exit 1 only on a blocking (silent-descent) verdict.
func cmdAuditRouting(args []string, stdout, stderr io.Writer) int {
	return cmdAuditRoutingWithHostPath(args, stdout, stderr, "", nil)
}

func cmdAuditRoutingWithHostPath(args []string, stdout, stderr io.Writer, hostPath string, lookup func(string) (string, error)) int {
	return cmdAuditRoutingWithHostPathAndDefaults(args, stdout, stderr, hostPath, lookup, audit.DefaultTranscriptsDirs, audit.DefaultCodexSessionsDir)
}

// cmdAuditRoutingWithHostPathAndDefaults makes default discovery injectable
// without mutating package globals. Host preflight intentionally runs before
// either default can inspect Claude or Codex session locations.
func cmdAuditRoutingWithHostPathAndDefaults(args []string, stdout, stderr io.Writer, hostPath string, lookup func(string) (string, error), defaultTranscriptsDirs func(string) ([]string, error), defaultCodexSessionsDir func() (string, error)) int {
	fs := flag.NewFlagSet("audit routing", flag.ContinueOnError)
	dir := fs.String("dir", ".", "repo root")
	transcripts := fs.String("transcripts", "", "harness transcript dir (default: repo, git-worktree, and matching project dirs under ~/.claude/projects)")
	codexSessions := fs.String("codex-sessions", "", "codex session dir (default: $CODEX_HOME/sessions, else ~/.codex/sessions)")
	since := fs.String("since", "", "scope to sessions at/after this cutoff (RFC3339, or YYYY-MM-DD for local midnight); operator escape hatch, default: unscoped")
	session := fs.String("session", "", "scope to one session id (claude: the session's file/dir base name; codex: session_meta's root session_id); default: unscoped")
	if _, ok := parseArgs(fs, args, "audit routing", `usage: spine audit routing [--dir D] [--transcripts DIR] [--codex-sessions DIR] [--since TIME] [--session ID]`, 0, stderr); !ok {
		return 2
	}
	if err := preflightHostConfig(hostPath, lookup); err != nil {
		fmt.Fprintln(stderr, "audit routing:", err)
		return 2
	}
	auditOpts := audit.Options{RepoDir: *dir, Since: *since, Session: *session}
	if *transcripts == "" {
		derived, err := defaultTranscriptsDirs(*dir)
		if err != nil {
			fmt.Fprintln(stderr, "audit routing:", err)
			return 2
		}
		auditOpts.ClaudeTranscriptsDirs = derived
	} else {
		auditOpts.ClaudeTranscriptsDir = *transcripts
	}
	// Warning rule (ratified at I041 review, design D-doc "Harness
	// threading"): a missing EXPLICITLY-requested sessions dir warns; a
	// missing un-overridden default is a silent skip — a codex-less machine
	// is normal, and a standing warning on every audit there is exactly the
	// permanent-noise failure the design's problem statement decries. So
	// only the derived default is stat-checked here; an explicit
	// --codex-sessions always reaches Run and gets its warning from
	// readCodexSessions same as before.
	cdir := *codexSessions
	if cdir == "" {
		derived, err := defaultCodexSessionsDir()
		if err != nil {
			fmt.Fprintln(stderr, "audit routing:", err)
			return 2
		}
		if _, statErr := os.Stat(derived); statErr == nil {
			cdir = derived
		}
	}
	auditOpts.CodexSessionsDir = cdir
	rep, err := audit.Run(auditOpts)
	if err != nil {
		fmt.Fprintln(stderr, "audit routing:", err)
		return 2
	}
	for _, w := range rep.Warnings {
		fmt.Fprintln(stderr, "warning:", w)
	}
	return printAuditRoutingReport(stdout, rep)
}

func printAuditRoutingReport(stdout io.Writer, rep audit.Report) int {
	dash := func(s string) string {
		if s == "" {
			return "-"
		}
		return s
	}
	wID, wTier, wActual, wVerdict := len("ticket"), len("tier"), len("actual"), len("verdict")
	for _, t := range rep.Tickets {
		wID = max(wID, len(t.ID))
		wTier = max(wTier, len(dash(t.Tier)))
		wActual = max(wActual, len(dash(strings.Join(t.Actuals, ","))))
		wVerdict = max(wVerdict, len(t.Verdict))
	}
	hasDeclarationEvents := false
	for _, t := range rep.Tickets {
		hasDeclarationEvents = hasDeclarationEvents || len(t.DeclarationEvents) > 0
	}
	heading := "expected-effort  declared-effort  declaration-status  observed-effort"
	if hasDeclarationEvents {
		heading += "  expected-pair  declared  model-confirmation  observed-effort-status  correlation"
	}
	fmt.Fprintf(stdout, "%-*s  %-*s  %-*s  %-*s  %s  %s\n", wID, "ticket", wTier, "tier", wActual, "actual", wVerdict, "verdict", "detail", heading)
	for _, t := range rep.Tickets {
		suffix := declarationEventOutput(t.DeclarationEvents)
		fmt.Fprintf(stdout, "%-*s  %-*s  %-*s  %-*s  %s  expected-effort=%s declared-effort=%s declaration-status=%s observed-effort=%s%s\n",
			wID, t.ID, wTier, dash(t.Tier), wActual, dash(strings.Join(t.Actuals, ",")), wVerdict, string(t.Verdict), t.Detail,
			dash(t.ExpectedEffort), dash(t.DeclaredEffort), dash(t.DeclarationStatus), dash(t.ObservedEffort), suffix)
	}
	if len(rep.Unmatched) > 0 {
		fmt.Fprintln(stdout, "unmatched dispatches (no ticket id or not repo-qualified; not judged):")
		teamSpawns := 0
		for _, d := range rep.Unmatched {
			model := dash(d.Model)
			if d.Effort != "" {
				model += " @ " + d.Effort
			}
			if d.TeamSpawn {
				teamSpawns++
			}
			// firstLine keeps a multi-line command (a lead writing the
			// brief and starting the worker in one call) from printing its
			// continuations at column 0, out of the two-space indent.
			desc := d.Description
			if i := strings.IndexByte(desc, '\n'); i >= 0 {
				desc = desc[:i] + " …"
			}
			fmt.Fprintf(stdout, "  %s  [%s]\n", desc, model)
		}
		// I090's residual blind spot, stated rather than left silent. The
		// wording names only what this code knows — that these spawns were
		// recognized and not attributed — not WHY. A record lands here for
		// several reasons (no ticket token in the command or its prompt,
		// the token naming another repo, the command failing repo
		// qualification), and the brief-delivered-by-file case that
		// motivated ticket I101 is only the most common one on real input.
		// Guessing a single cause in the footer was wrong for the others.
		if teamSpawns > 0 {
			fmt.Fprintf(stdout,
				"  note: %d team spawn(s) recognised but not attributed (no ticket token in the command or its prompt, or not repo-qualified)\n",
				teamSpawns)
		}
	}
	if rep.Blocking() {
		return 1
	}
	return 0
}

func declarationEventOutput(events []audit.DeclarationEvidence) string {
	if len(events) == 0 {
		return ""
	}
	dash := func(value string) string {
		if value == "" {
			return "-"
		}
		return value
	}
	parts := make([]string, 0, len(events))
	for _, event := range events {
		parts = append(parts, fmt.Sprintf(
			"expected-pair=%s@%s declared=%s,%s,%s model-confirmation=%s observed-effort-status=%s correlation=%s",
			dash(event.ExpectedModel), dash(event.ExpectedEffort), dash(event.Harness), dash(event.Model), dash(event.DeclaredEffort),
			dash(string(event.ModelStatus)), dash(event.ObservedEffortStatus), dash(event.Correlation)))
	}
	return " " + strings.Join(parts, ";")
}

// cmdAuditStages is a thin printer over stages.Derive: a table of every
// cursor stage's derivation verdict (like audit routing's ticket table),
// plus the newest-handoff backstop line. Exit 1 when Report.Blocking() — a
// ticked-but-missing or present-but-unticked stage, or a missing/stale
// newest-handoff cursor block — OR when the cursor block itself is malformed
// or valid-but-non-canonical. The latter is the sole-writer gate: parse then
// canonical reserialize comparison detects a hand edit without treating it
// as a grammar failure. Audit stages is the ONLY caller where either cursor
// condition blocks — `spine cursor` stays exit-0-always (read-only printer)
// and doctor's D9 stays warn-only; neither is changed by this. No cursor at
// all is a warning, never a failure (exit 0) — see internal/stages' package
// doc for the three quiet cases this collapses.
func cmdAuditStages(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("audit stages", flag.ContinueOnError)
	dir := fs.String("dir", ".", "repo root")
	if _, ok := parseArgs(fs, args, "audit stages", `usage: spine audit stages [--dir D]`, 0, stderr); !ok {
		return 2
	}
	rep, err := stages.Derive(*dir)
	if err != nil {
		fmt.Fprintln(stderr, "audit stages:", err)
		return 2
	}
	for _, f := range rep.CursorFindings {
		fmt.Fprintln(stderr, "warning: cursor finding:", f)
	}
	for _, n := range rep.Notes {
		fmt.Fprintln(stderr, "warning:", n)
	}
	// The remediation round-budget advisory (I087): printed here only, and
	// never consulted by Blocking() — a round beyond budget advises, it does
	// not gate. Kept out of rep.Notes so doctor's D9 pass-through (a warn,
	// which does set doctor's exit code) cannot turn it into a gate.
	for _, n := range rep.RoundBudget {
		fmt.Fprintln(stderr, "warning:", n)
	}
	for _, problem := range rep.Acceptance.Problems {
		fmt.Fprintln(stderr, "warning:", problem.Path+":", problem.Message())
	}
	for _, scanErr := range rep.Acceptance.ScanErrors {
		fmt.Fprintln(stderr, "warning:", scanErr.Path+":", scanErr.Message())
	}
	if !rep.HasCursor {
		fmt.Fprintln(stdout, "no spine cursor — nothing to audit")
		return 0
	}
	cursorMalformed := len(rep.CursorFindings) > 0
	cursorNonCanonical := rep.CursorNonCanonical
	const cursorName, cursorState, cursorVerdict = "cursor", "n/a", "blocking"
	wName, wState, wVerdict := len("stage"), len("state"), len("verdict")
	for _, s := range rep.Stages {
		wName = max(wName, len(s.Name))
		wState = max(wState, len(s.StateLabel()))
		wVerdict = max(wVerdict, len(string(s.Verdict)))
	}
	if cursorMalformed || cursorNonCanonical {
		wName = max(wName, len(cursorName))
		wState = max(wState, len(cursorState))
		wVerdict = max(wVerdict, len(cursorVerdict))
	}
	fmt.Fprintf(stdout, "%-*s  %-*s  %-*s  %s\n", wName, "stage", wState, "state", wVerdict, "verdict", "detail")
	if cursorMalformed {
		fmt.Fprintf(stdout, "%-*s  %-*s  %-*s  %s\n", wName, cursorName, wState, cursorState, wVerdict, cursorVerdict,
			"malformed cursor block — grammar findings: "+strings.Join(rep.CursorFindings, "; "))
	}
	if cursorNonCanonical {
		fmt.Fprintf(stdout, "%-*s  %-*s  %-*s  %s\n", wName, cursorName, wState, cursorState, wVerdict, cursorVerdict, cursor.NonCanonicalRemediation)
	}
	for _, s := range rep.Stages {
		fmt.Fprintf(stdout, "%-*s  %-*s  %-*s  %s\n", wName, s.Name, wState, s.StateLabel(), wVerdict, string(s.Verdict), s.Detail)
	}
	if rep.Acceptance.CandidateCount() > 0 {
		fmt.Fprintf(stdout, "acceptance: approved-untested=%d invalid=%d\n", rep.Acceptance.ValidCount(), rep.Acceptance.InvalidCount())
	}
	fmt.Fprintf(stdout, "handoff: applicable=%v blocking=%v — %s\n", rep.Handoff.Applicable, rep.Handoff.Blocking(), rep.Handoff.Detail)
	if rep.Blocking() || cursorMalformed || cursorNonCanonical {
		return 1
	}
	return 0
}

// checkpointUsage documents the verbs in CONTEXT.md vocabulary: a
// checkpoint is what a running session distils itself into before a context
// reload; it holds a model region (the model's own prior claims) and a
// facts region (harness-written evidence), and accumulates in the
// checkpoint working home.
const checkpointUsage = `usage: spine checkpoint <new|latest|list> [flags]

  new     write a checkpoint into the working home
          (.superpowers/sdd/checkpoints/NNN-<slug>.md)
          spine checkpoint new [--dir D] --from <narrative.md> --touched <csv>
            --gate <pass|fail|none> --effort <level> [--slug s] [--facts-only]
          The narrative becomes the model region and must carry non-empty
          "## Task", "## Conclusions" and "## Next moves" sections; a missing
          or empty section is refused (exit 2, naming the section).
          --facts-only skips the narrative and writes narrative: missing.
  latest  print the reload preamble followed by the newest checkpoint
          (exit 1 when the working home is empty)
  list    list the working home in ordinal order
`

func cmdCheckpoint(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, checkpointUsage)
		return 2
	}
	switch args[0] {
	case "new":
		return cmdCheckpointNew(args[1:], stdout, stderr)
	case "latest":
		return cmdCheckpointLatest(args[1:], stdout, stderr)
	case "list":
		return cmdCheckpointList(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown checkpoint subcommand %q\n%s", args[0], checkpointUsage)
		return 2
	}
}

func cmdCheckpointNew(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("checkpoint new", flag.ContinueOnError)
	dir := fs.String("dir", ".", "repo root")
	from := fs.String("from", "", "narrative file becoming the model region")
	touched := fs.String("touched", "", "comma-separated files touched (order preserved)")
	gate := fs.String("gate", "", "gate status: pass | fail | none")
	effort := fs.String("effort", "", "recommended per-leg effort")
	slug := fs.String("slug", "", "filename slug (default: derived from the ## Task line)")
	factsOnly := fs.Bool("facts-only", false, "write facts with narrative: missing and an empty model region")
	if _, ok := parseArgs(fs, args, "checkpoint new", `usage: spine checkpoint new [--dir D] --from <narrative.md> --touched <csv> --gate <pass|fail|none> --effort <level> [--slug s] [--facts-only]`, 0, stderr); !ok {
		return 2
	}
	path, err := checkpoint.New(checkpoint.Options{
		Dir:       *dir,
		From:      *from,
		Touched:   splitTouched(*touched),
		Gate:      *gate,
		Effort:    *effort,
		Slug:      *slug,
		FactsOnly: *factsOnly,
	})
	if err != nil {
		fmt.Fprintln(stderr, "checkpoint new:", err)
		return 2
	}
	fmt.Fprintln(stdout, path)
	return 0
}

// splitTouched parses the --touched CSV; an empty value is an empty list,
// and caller order is preserved.
func splitTouched(csv string) []string {
	var out []string
	for _, part := range strings.Split(csv, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func cmdCheckpointLatest(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("checkpoint latest", flag.ContinueOnError)
	dir := fs.String("dir", ".", "repo root")
	if _, pok := parseArgs(fs, args, "checkpoint latest", `usage: spine checkpoint latest [--dir D]`, 0, stderr); !pok {
		return 2
	}
	e, ok, err := checkpoint.Latest(*dir)
	if err != nil {
		fmt.Fprintln(stderr, "checkpoint latest:", err)
		return 2
	}
	if !ok {
		fmt.Fprintln(stderr, "no checkpoints found")
		return 1
	}
	preamble, err := checkpoint.Preamble()
	if err != nil {
		fmt.Fprintln(stderr, "checkpoint latest:", err)
		return 2
	}
	raw, err := os.ReadFile(e.Path)
	if err != nil {
		fmt.Fprintln(stderr, "checkpoint latest:", err)
		return 2
	}
	fmt.Fprint(stdout, preamble)
	fmt.Fprint(stdout, string(raw))
	return 0
}

func cmdCheckpointList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("checkpoint list", flag.ContinueOnError)
	dir := fs.String("dir", ".", "repo root")
	if _, ok := parseArgs(fs, args, "checkpoint list", `usage: spine checkpoint list [--dir D]`, 0, stderr); !ok {
		return 2
	}
	entries, err := checkpoint.List(*dir)
	if err != nil {
		fmt.Fprintln(stderr, "checkpoint list:", err)
		return 2
	}
	for _, e := range entries {
		fmt.Fprintf(stdout, "%03d  %-30s  %s\n", e.Ordinal, e.Slug, e.Created)
	}
	return 0
}

// cmdCursor is a thin, read-only printer over cursor.Load: it prints the
// parsed stage cursor plus the live derivation verdict (internal/stages,
// I019). Flag-only invocations always exit 0: a cursor mismatch or parse
// finding is surfaced, never gated, here — spine audit stages is where
// that becomes a blocking check. Positionals are a usage error (I119):
// hooks never pass them, and the old behavior — an unknown sub-subcommand
// like `show` silently swallowed, its trailing flags dropped, and the CWD
// repo answered for with exit 0 — was a clean exit over wrong data.
// --quiet is for hook use: it silences the "nothing to report" case (no
// spine repo, no ledger, no cursor block) but never silences a cursor that
// was actually found, malformed or not — the SessionStart hook (I021)
// needs real output precisely when a cursor exists.
//
// The derivation line is one of three: "clean" (no contradictions), "n/a
// (cursor malformed)" (I024 — grammar findings on the cursor block itself,
// e.g. a stages: line that parses to zero stage rows, printed instead of a
// misleading "clean"), or "blocking" (a stage/artifact contradiction or a
// stale newest-handoff). Grammar findings win priority over "blocking":
// with zero parsed stage rows there is nothing coherent to call clean or
// blocking about the stages themselves — see internal/stages' package doc
// (CursorFindings never affects Report.Blocking()).
func cmdCursor(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "start":
			return cmdCursorStart(args[1:], stdout, stderr)
		case "tick":
			return cmdCursorTick(args[1:], stdout, stderr)
		case "here":
			return cmdCursorHere(args[1:], stdout, stderr)
		case "set":
			return cmdCursorSet(args[1:], stdout, stderr)
		default:
			// A non-flag first token is an unknown sub-subcommand, reported
			// like the other dispatchers (I119) — never silently swallowed
			// into the bare printer.
			if !strings.HasPrefix(args[0], "-") {
				fmt.Fprintf(stderr, "unknown cursor subcommand %q\n%s\n", args[0], cursorUsage)
				return 2
			}
		}
	}
	fs := flag.NewFlagSet("cursor", flag.ContinueOnError)
	dir := fs.String("dir", ".", "repo root")
	quiet := fs.Bool("quiet", false, "print nothing and exit 0 when there is no spine cursor (hook-friendly)")
	if _, ok := parseArgs(fs, args, "cursor", cursorUsage, 0, stderr); !ok {
		return 2
	}
	res, err := cursor.Load(*dir)
	if err != nil {
		// Genuine I/O failure (not "file doesn't exist") — still advisory,
		// per the "exit 0 always" contract, but worth surfacing.
		fmt.Fprintln(stderr, "cursor:", err)
		return 0
	}
	if !res.HasCursor {
		if !*quiet {
			fmt.Fprintf(stdout, "no spine cursor found in %s\n", *dir)
		}
		return 0
	}
	for _, f := range res.Findings {
		fmt.Fprintln(stdout, "finding:", f)
	}
	if res.Cursor.Effort != "" {
		fmt.Fprintf(stdout, "effort: %s\n", res.Cursor.Effort)
	}
	if res.Cursor.PRD != "" {
		fmt.Fprintf(stdout, "prd: %s\n", res.Cursor.PRD)
	}
	if res.Cursor.Tickets != "" {
		fmt.Fprintf(stdout, "tickets: %s\n", res.Cursor.Tickets)
	}
	if len(res.Cursor.Stages) > 0 {
		fmt.Fprintf(stdout, "stages: %s\n", res.Cursor.StagesLine())
	}
	rep := stages.FromResult(*dir, res)
	// F1 (final whole-branch review, I024-I027 batch): rep.Notes (e.g. an
	// unresolvable tickets: value, I026) was computed but never printed
	// here — a hook consuming this command's stdout never saw it, leaving
	// an ambient "derivation: clean" with no visible reason to distrust
	// the tickets: line. Match how audit stages already surfaces the same
	// Notes entries (as "warning: <note>").
	for _, n := range rep.Notes {
		fmt.Fprintln(stdout, "warning:", n)
	}
	switch {
	case len(rep.CursorFindings) > 0:
		// I024: a cursor block with grammar findings (e.g. a stages: line
		// that parses to zero stage rows) must not be reported "clean" —
		// that read was incoherent with `spine audit stages` blocking on
		// the same fixture. The grammar problems themselves were already
		// printed above as "finding:" lines; this just names the verdict
		// honestly. Still exit 0 — spine cursor stays a read-only printer.
		fmt.Fprintln(stdout, "derivation: n/a (cursor malformed)")
		// F1(b): a malformed stages: grammar says nothing about the
		// handoff backstop (I014/I025) — the two checks are independent.
		// Print the blocking handoff detail here too, or a stale/missing
		// newest handoff silently drops off the hook surface the moment
		// the cursor also happens to be malformed (the info-loss corner
		// the I024 review found).
		if rep.Handoff.Blocking() {
			fmt.Fprintf(stdout, "  handoff: %s\n", rep.Handoff.Detail)
		}
	case !rep.Blocking():
		fmt.Fprintln(stdout, "derivation: clean")
	default:
		fmt.Fprintln(stdout, "derivation: blocking")
		for _, s := range rep.Stages {
			if s.Verdict == stages.VerdictTickedMissing || s.Verdict == stages.VerdictPresentUnticked {
				fmt.Fprintf(stdout, "  %s (%s): %s\n", s.Name, s.Verdict, s.Detail)
			}
		}
		if rep.Handoff.Blocking() {
			fmt.Fprintf(stdout, "  handoff: %s\n", rep.Handoff.Detail)
		}
	}
	return 0
}

// cursorUsage names the read form and the write verbs; the unknown-
// subcommand error above leans on it to list the real verbs.
const cursorUsage = `usage: spine cursor [--dir D] [--quiet]  (or: spine cursor <start|tick|here|set> [flags])`

// cursorForWrite loads the working-home cursor and refuses mutations of a
// malformed block. A writer must never turn a diagnostic parse into a
// partially informed rewrite; `set` can normalize formatting once the block
// is valid.
func cursorForWrite(dir string) (cursor.Cursor, error) {
	res, err := cursor.Load(dir)
	if err != nil {
		return cursor.Cursor{}, err
	}
	if !res.HasCursor {
		return cursor.Cursor{}, fmt.Errorf("no spine cursor found; use `spine cursor start`")
	}
	if len(res.Findings) > 0 {
		return cursor.Cursor{}, fmt.Errorf("cursor block is malformed: %s", strings.Join(res.Findings, "; "))
	}
	return res.Cursor, nil
}

func cursorStageIndex(c cursor.Cursor, name string) int {
	for i, s := range c.Stages {
		if s.Name == name {
			return i
		}
	}
	return -1
}

// takeForce accepts --force in the conventional trailing position as well as
// before a positional stage name, where the standard flag package otherwise
// stops parsing. The remaining flags retain flag.FlagSet's usual validation.
func takeForce(args []string) ([]string, bool) {
	kept := make([]string, 0, len(args))
	force := false
	for _, arg := range args {
		if arg == "--force" || arg == "-force" {
			force = true
			continue
		}
		kept = append(kept, arg)
	}
	return kept, force
}

func cmdCursorStart(args []string, stdout, stderr io.Writer) int {
	args, force := takeForce(args)
	fs := flag.NewFlagSet("cursor start", flag.ContinueOnError)
	dir := fs.String("dir", ".", "repo root")
	effort := fs.String("effort", "", "effort name")
	prd := fs.String("prd", "", "PRD path")
	tickets := fs.String("tickets", "", "ticket range")
	const startUsage = "usage: spine cursor start --effort <name> [--prd <path>] [--tickets <range>] [--force] [--dir D]"
	if _, ok := parseArgs(fs, args, "cursor start", startUsage, 0, stderr); !ok {
		return 2
	}
	if strings.TrimSpace(*effort) == "" {
		fmt.Fprintln(stderr, startUsage)
		return 2
	}
	res, err := cursor.Load(*dir)
	if err != nil {
		fmt.Fprintln(stderr, "cursor start:", err)
		return 2
	}
	if res.HasCursor {
		if len(res.Findings) > 0 {
			fmt.Fprintln(stderr, "cursor start: cursor block is malformed:", strings.Join(res.Findings, "; "))
			return 1
		}
		for _, s := range res.Cursor.Stages {
			if s.State != cursor.Done && !force {
				fmt.Fprintln(stderr, "cursor start: existing cursor is mid-flight; pass --force to supersede it")
				return 1
			}
		}
	}
	c, err := cursor.New(*dir, *effort, *prd, *tickets)
	if err != nil {
		fmt.Fprintln(stderr, "cursor start:", err)
		return 2
	}
	if err := cursor.Save(*dir, c); err != nil {
		fmt.Fprintln(stderr, "cursor start:", err)
		return 2
	}
	fmt.Fprintf(stdout, "cursor started: %s\n", c.Effort)
	return 0
}

func cmdCursorTick(args []string, stdout, stderr io.Writer) int {
	args, force := takeForce(args)
	fs := flag.NewFlagSet("cursor tick", flag.ContinueOnError)
	dir := fs.String("dir", ".", "repo root")
	pos, pok := parseArgs(fs, args, "cursor tick", "usage: spine cursor tick [--dir D] <stage> [--force]", 1, stderr)
	if !pok {
		return 2
	}
	c, err := cursorForWrite(*dir)
	if err != nil {
		fmt.Fprintln(stderr, "cursor tick:", err)
		return 1
	}
	idx := cursorStageIndex(c, pos[0])
	if idx < 0 {
		fmt.Fprintf(stderr, "cursor tick: stage %q is not in the cursor\n", pos[0])
		return 2
	}
	candidate := c
	candidate.Stages = append([]cursor.Stage(nil), c.Stages...)
	wasHere := candidate.Stages[idx].State == cursor.Here
	candidate.Stages[idx].State = cursor.Done
	if wasHere {
		// Search cyclically so a marker moved forward with `here` cannot strand
		// earlier pending stages. Drop the marker only when every other stage is
		// already done.
		for offset := 1; offset < len(candidate.Stages); offset++ {
			next := (idx + offset) % len(candidate.Stages)
			if candidate.Stages[next].State != cursor.Done {
				candidate.Stages[next].State = cursor.Here
				break
			}
		}
	}
	// FromResult is the audit derivation engine applied to the proposed
	// cursor state. Its Detail is printed verbatim below, so this early
	// refusal and `spine audit stages` share one finding vocabulary.
	rep := stages.FromResult(*dir, cursor.Result{Cursor: candidate, HasCursor: true})
	row := rep.Stages[idx]
	if row.Verdict == stages.VerdictTickedMissing && !force {
		fmt.Fprintln(stderr, row.Detail)
		return 1
	}
	if err := cursor.Save(*dir, candidate); err != nil {
		fmt.Fprintln(stderr, "cursor tick:", err)
		return 2
	}
	fmt.Fprintf(stdout, "cursor ticked: %s\n", pos[0])
	return 0
}

func cmdCursorHere(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("cursor here", flag.ContinueOnError)
	dir := fs.String("dir", ".", "repo root")
	pos, pok := parseArgs(fs, args, "cursor here", "usage: spine cursor here [--dir D] <stage>", 1, stderr)
	if !pok {
		return 2
	}
	c, err := cursorForWrite(*dir)
	if err != nil {
		fmt.Fprintln(stderr, "cursor here:", err)
		return 1
	}
	idx := cursorStageIndex(c, pos[0])
	if idx < 0 {
		fmt.Fprintf(stderr, "cursor here: stage %q is not in the cursor\n", pos[0])
		return 2
	}
	for i := range c.Stages {
		if i != idx && c.Stages[i].State == cursor.Here {
			c.Stages[i].State = cursor.Pending
		}
	}
	// Assigning Here deliberately turns a completed stage back into current:
	// this is the sole regression path and there is no separate untick verb.
	c.Stages[idx].State = cursor.Here
	if err := cursor.Save(*dir, c); err != nil {
		fmt.Fprintln(stderr, "cursor here:", err)
		return 2
	}
	fmt.Fprintf(stdout, "cursor here: %s\n", pos[0])
	return 0
}

func cmdCursorSet(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("cursor set", flag.ContinueOnError)
	dir := fs.String("dir", ".", "repo root")
	prd := fs.String("prd", "", "PRD path")
	tickets := fs.String("tickets", "", "ticket range")
	if _, ok := parseArgs(fs, args, "cursor set", "usage: spine cursor set [--prd <path>] [--tickets <range>] [--dir D]", 0, stderr); !ok {
		return 2
	}
	c, err := cursorForWrite(*dir)
	if err != nil {
		fmt.Fprintln(stderr, "cursor set:", err)
		return 1
	}
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "prd":
			c.PRD = *prd
		case "tickets":
			c.Tickets = *tickets
		}
	})
	if err := cursor.Save(*dir, c); err != nil {
		fmt.Fprintln(stderr, "cursor set:", err)
		return 2
	}
	fmt.Fprintln(stdout, "cursor updated")
	return 0
}

// buildLine formats spine version's provenance line from
// debug.ReadBuildInfo (I118): module version, vcs revision (12 chars),
// vcs time, and a dirty flag — enough to compare two devices' builds with
// one command. No ldflags: `go install` stamps the module version and
// `go build` in a checkout stamps the vcs fields. Absent fields are
// omitted; no build info at all degrades to a fixed placeholder rather
// than an error.
func buildLine(bi *debug.BuildInfo) string {
	if bi == nil {
		return "build: (no build info)"
	}
	var rev, vcsTime string
	dirty := false
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.time":
			vcsTime = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if len(rev) > 12 {
		rev = rev[:12]
	}
	var parts []string
	for _, p := range []string{bi.Main.Version, rev, vcsTime} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if dirty {
		parts = append(parts, "dirty")
	}
	if len(parts) == 0 {
		return "build: (no build info)"
	}
	return "build: " + strings.Join(parts, " ")
}

// parseArgs is the shared strict parse for every subcommand (I119,
// generalizing I116): it wires the FlagSet to stderr, parses, then rejects
// a flag-like token among the positionals (the ordering rule) and any
// positional count other than wantN — a stray positional is discarded
// input, the same defect as a dropped flag. wantN < 0 skips the arity
// check. On violation the command-prefixed rule plus usage goes to stderr
// and the caller returns 2. Ordering is judged before arity so the
// correct-arity shape (I116's `model claude --json`) still names the rule.
// Callers that pre-strip args (takeForce) pass the stripped slice, keeping
// the guard on what the FlagSet actually saw.
func parseArgs(fs *flag.FlagSet, args []string, name, usage string, wantN int, stderr io.Writer) ([]string, bool) {
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return nil, false
	}
	pos := fs.Args()
	if tok, prev := flagAmongPositionals(pos); tok != "" {
		fmt.Fprintf(stderr, "%s: flags must precede positionals (saw %q after %q)\n%s\n", name, tok, prev, strings.TrimRight(usage, "\n"))
		return nil, false
	}
	if wantN >= 0 && len(pos) > wantN {
		fmt.Fprintf(stderr, "%s: unexpected argument %q\n%s\n", name, pos[wantN], strings.TrimRight(usage, "\n"))
		return nil, false
	}
	if wantN >= 0 && len(pos) < wantN {
		fmt.Fprintln(stderr, strings.TrimRight(usage, "\n"))
		return nil, false
	}
	return pos, true
}

// flagAmongPositionals returns the first flag-like token (leading "-")
// left among the positionals after a successful parse, and the positional
// preceding it. A first-position hit is only reachable via an explicit
// "--" terminator — a deliberate positional, not an ordering mistake — so
// it is not reported.
func flagAmongPositionals(args []string) (tok, prev string) {
	for i, a := range args {
		if i > 0 && strings.HasPrefix(a, "-") {
			return a, args[i-1]
		}
	}
	return "", ""
}

// cmdModel is a thin printer over model.Resolve (design D12): the CLI does
// no resolution of its own, just flag parsing and formatting. Harness and
// tier are both required positional arguments — never inferred or
// defaulted, per the ticket's invisible-resolution concern — so a missing
// or unknown one is reported via model.Resolve's own error rather than a
// second validation path here.
func cmdModel(args []string, stdout, stderr io.Writer) int {
	return cmdModelWithHostPath(args, stdout, stderr, "", nil)
}

const modelValidateUsage = `usage: spine model [--dir D] validate [--expect MODEL_ID] <harness> <tier>`

// cmdModelWithHostPath keeps host-config location and executable discovery as
// argument seams. Production supplies neither, so there is no CLI path or
// environment override for the local host authority.
func cmdModelWithHostPath(args []string, stdout, stderr io.Writer, hostPath string, lookup func(string) (string, error)) int {
	if isModelValidateInvocation(args) {
		return cmdModelValidateInvocation(args, stdout, stderr, hostPath, lookup)
	}
	fs := flag.NewFlagSet("model", flag.ContinueOnError)
	dir := fs.String("dir", ".", "repo root")
	effort := fs.Bool("effort", false, "print the resolved effort instead of the bare id")
	asJSON := fs.Bool("json", false, "print the whole resolved entry as JSON")
	dispatchEffort := fs.String("dispatch-effort", "", "request a raw dispatch effort (JSON output only)")
	alternate := fs.Bool("alternate", false, "print the cell's alternate instead of its primary id/effort")
	// I116's ordering guard now lives in parseArgs (I119 generalized it to
	// every subcommand); this command's error text is the shape the helper
	// standardizes on.
	const modelUsage = `usage: spine model [--dir D] [--alternate] [--effort|--json] <harness> <tier>`
	pos, ok := parseArgs(fs, args, "model", modelUsage, 2, stderr)
	if !ok {
		return 2
	}
	dispatchEffortSet := false
	fs.Visit(func(f *flag.Flag) {
		dispatchEffortSet = f.Name == "dispatch-effort" || dispatchEffortSet
	})
	if dispatchEffortSet && !*asJSON {
		fmt.Fprintln(stderr, "model: --dispatch-effort requires --json")
		return 2
	}
	if dispatchEffortSet && *alternate {
		fmt.Fprintln(stderr, "model: --dispatch-effort cannot be combined with --alternate")
		return 2
	}
	var entry model.Entry
	var resolution model.Resolution
	var err error
	if *alternate {
		// I072 validates a present local file structurally, but host constraints
		// deliberately do not apply to the legacy cell alternate. In particular,
		// a valid config must not require this harness, its executable, or its
		// primary route to be reachable before returning the critic pair.
		if err := preflightHostConfig(hostPath, lookup); err != nil {
			fmt.Fprintln(stderr, "model:", err)
			return 2
		}
		entry, err = model.Resolve(*dir, pos[0], pos[1])
	} else {
		resolution, err = model.ResolveForHost(*dir, hostPath, pos[0], pos[1], lookup)
		entry = resolution.Entry
	}
	if err != nil {
		fmt.Fprintln(stderr, "model:", err)
		return 2
	}
	if dispatchEffortSet {
		entry, err = model.ApplyDispatchEffort(entry, *dispatchEffort)
		if err != nil {
			fmt.Fprintln(stderr, "model:", err)
			return 2
		}
	}
	// --alternate answers from the cell's alternate half (I079). A cell that
	// ships none is an error rather than a silent fall-back to the primary:
	// an evaluator that asked for the critic and got the author would run a
	// model against itself and report agreement.
	if *alternate && entry.Alternate == nil && !*asJSON {
		fmt.Fprintf(stderr, "model: %s.%s has no alternate\n", entry.Harness, entry.Tier)
		return 2
	}
	switch {
	case *asJSON:
		type altJSON struct {
			ID     string `json:"id"`
			Effort string `json:"effort"`
		}
		type requestedJSON struct {
			ID         string `json:"id"`
			Effort     string `json:"effort"`
			Provenance string `json:"provenance"`
		}
		type entryJSON struct {
			Harness string `json:"harness"`
			// Flavor is retained as a byte-equal deprecated compatibility field
			// throughout the generation-14 fleet window.
			Flavor     string                `json:"flavor"`
			Tier       string                `json:"tier"`
			ID         string                `json:"id"`
			Effort     string                `json:"effort"`
			Aliases    []string              `json:"aliases"`
			Alternate  *altJSON              `json:"alternate,omitempty"`
			Provenance string                `json:"provenance"`
			Requested  *requestedJSON        `json:"requested,omitempty"`
			Host       *model.HostResolution `json:"host,omitempty"`
			Pin        *model.PinResolution  `json:"pin,omitempty"`
		}
		out := entryJSON{
			Harness: entry.Harness, Flavor: entry.Harness, Tier: entry.Tier, ID: entry.ID, Effort: entry.Effort,
			Aliases: entry.Aliases, Provenance: string(entry.Provenance),
		}
		if entry.Alternate != nil {
			out.Alternate = &altJSON{ID: entry.Alternate.ID, Effort: entry.Alternate.Effort}
		}
		if !*alternate && resolution.Host.Status != model.HostUnconfigured {
			requested := resolution.Requested
			out.Requested = &requestedJSON{ID: requested.ID, Effort: requested.Effort, Provenance: string(requested.Provenance)}
			out.Host = &resolution.Host
			out.Pin = resolution.Pin
		}
		if out.Aliases == nil {
			out.Aliases = []string{}
		}
		if err := json.NewEncoder(stdout).Encode(out); err != nil {
			fmt.Fprintln(stderr, "model:", err)
			return 2
		}
	case *alternate && *effort:
		fmt.Fprintln(stdout, entry.Alternate.Effort)
	case *alternate:
		fmt.Fprintln(stdout, entry.Alternate.ID)
	case *effort:
		fmt.Fprintln(stdout, entry.Effort)
	default:
		fmt.Fprintln(stdout, entry.ID)
	}
	return 0
}

// isModelValidateInvocation recognizes validate only while scanning the
// leading flag region. A later "validate" used as a regular model harness or
// tier remains owned by the legacy model command and keeps its diagnostics.
func isModelValidateInvocation(args []string) bool {
	for i := 0; i < len(args); {
		switch arg := args[i]; {
		case arg == "validate":
			return true
		case arg == "--dir" || arg == "-dir" || arg == "--expect" || arg == "-expect":
			i += 2
		case strings.HasPrefix(arg, "-"):
			i++
		default:
			return false
		}
	}
	return false
}

// cmdModelValidateInvocation parses only the outer validation grammar. Its
// FlagSet intentionally knows only --dir, so legacy model flags and a misplaced
// --expect fail as validation usage errors without changing other model calls.
func cmdModelValidateInvocation(args []string, stdout, stderr io.Writer, hostPath string, lookup func(string) (string, error)) int {
	fs := flag.NewFlagSet("model validate", flag.ContinueOnError)
	dir := fs.String("dir", ".", "repo root")
	var diagnostics bytes.Buffer
	fs.SetOutput(&diagnostics)
	fs.Usage = func() {}
	if err := fs.Parse(args); err != nil {
		writeModelValidateDiagnostic(stderr, diagnostics.String())
		return 2
	}
	pos := fs.Args()
	if len(pos) == 0 || pos[0] != "validate" {
		fmt.Fprintf(stderr, "model validate: expected validate after outer flags\n%s\n", modelValidateUsage)
		return 2
	}
	return cmdModelValidateWithHostPath(pos[1:], *dir, stdout, stderr, hostPath, lookup)
}

func parseModelValidateArgs(fs *flag.FlagSet, args []string, stderr io.Writer) ([]string, bool) {
	var diagnostics bytes.Buffer
	fs.Usage = func() {}
	pos, ok := parseArgs(fs, args, "model validate", modelValidateUsage, 2, &diagnostics)
	if !ok {
		writeModelValidateDiagnostic(stderr, diagnostics.String())
	}
	return pos, ok
}

func writeModelValidateDiagnostic(stderr io.Writer, diagnostic string) {
	if strings.HasPrefix(diagnostic, "model validate:") {
		fmt.Fprint(stderr, diagnostic)
		return
	}
	diagnostic = strings.TrimSuffix(diagnostic, "\n")
	if diagnostic == modelValidateUsage {
		fmt.Fprintf(stderr, "model validate: %s\n", diagnostic)
		return
	}
	fmt.Fprintf(stderr, "model validate: %s\n%s\n", escapeModelValidateControlBytes(diagnostic), modelValidateUsage)
}

// preflightHostConfig performs only the complete ratified hostconfig.Load
// boundary. It leaves selection to model resolution and preserves an absent
// file as the legacy host-blind path.
func preflightHostConfig(path string, lookup func(string) (string, error)) error {
	if path == "" {
		var err error
		path, err = hostconfig.DefaultPath()
		if err != nil {
			return err
		}
	}
	if lookup == nil {
		lookup = exec.LookPath
	}
	_, err := hostconfig.Load(path, model.Harnesses(), lookup)
	if errors.Is(err, hostconfig.ErrNotConfigured) {
		return nil
	}
	return err
}

func cmdModelValidateWithHostPath(args []string, repoDir string, stdout, stderr io.Writer, hostPath string, lookup func(string) (string, error)) int {
	fs := flag.NewFlagSet("model validate", flag.ContinueOnError)
	var expected string
	var expectSet bool
	fs.Func("expect", "exact model id the launcher will use", func(value string) error {
		if expectSet {
			return fmt.Errorf("--expect may be supplied only once")
		}
		expectSet = true
		expected = value
		return nil
	})
	pos, ok := parseModelValidateArgs(fs, args, stderr)
	if !ok {
		return 2
	}
	if expectSet && expected == "" {
		fmt.Fprintf(stderr, "model validate: --expect must not be empty\n%s\n", modelValidateUsage)
		return 2
	}
	resolution, err := model.ValidateLaunchForHost(model.LaunchRequest{
		RepoDir:            repoDir,
		Harness:            pos[0],
		Tier:               pos[1],
		Expected:           expected,
		MaxTemplateVersion: tmpl.Version(),
	}, hostPath, lookup)
	if err != nil {
		var refusal *model.LaunchRefusal
		if errors.As(err, &refusal) {
			printModelLaunchRefusal(stderr, refusal, repoDir, expectSet)
			return 1
		}
		writeModelValidateConfigurationDiagnostic(stderr, pos[0], pos[1], err)
		return 2
	}
	fmt.Fprintln(stdout, resolution.Entry.ID)
	return 0
}

func writeModelValidateConfigurationDiagnostic(stderr io.Writer, harness, tier string, err error) {
	detail := escapeModelValidateControlBytes(err.Error())
	key, known := knownModelRouteKey(harness, tier)
	if !known {
		fmt.Fprintf(stderr, "model validate: %q: %s\n", harness+"."+tier, detail)
		return
	}

	// Some resolver errors already lead with the selected key. Keep one
	// canonical route label and escape any other detail occurrence so an
	// untrusted path cannot duplicate or impersonate it.
	detail = strings.TrimPrefix(detail, key+": ")
	detail = strings.ReplaceAll(detail, key, harness+`\x2e`+tier)
	fmt.Fprintf(stderr, "model validate: %s: %s\n", key, detail)
}

func knownModelRouteKey(harness, tier string) (string, bool) {
	knownHarness := false
	for _, candidate := range model.Harnesses() {
		if harness == candidate {
			knownHarness = true
			break
		}
	}
	if !knownHarness {
		return "", false
	}
	for _, candidate := range model.Tiers {
		if tier == candidate {
			return harness + "." + tier, true
		}
	}
	return "", false
}

func escapeModelValidateControlBytes(value string) string {
	var escaped strings.Builder
	for _, r := range value {
		switch {
		case r == '\n':
			escaped.WriteString(`\n`)
		case r == '\r':
			escaped.WriteString(`\r`)
		case r == '\t':
			escaped.WriteString(`\t`)
		case r < ' ' || (r >= 0x7f && r <= 0x9f):
			fmt.Fprintf(&escaped, `\u%04x`, r)
		default:
			escaped.WriteRune(r)
		}
	}
	return escaped.String()
}

func printModelLaunchRefusal(stderr io.Writer, refusal *model.LaunchRefusal, repoDir string, expected bool) {
	subject := "resolves"
	if expected {
		subject = "candidate"
	}
	switch refusal.Reason {
	case model.ReasonForbiddenModel:
		fmt.Fprintf(stderr, "model validate: %s: %s %s %q (rule: %s)\n", refusal.Reason, refusal.Key, subject, refusal.Value, refusal.Rule)
	case model.ReasonInvalidModelID:
		fmt.Fprintf(stderr, "model validate: %s: %s %s %q (allowed: ASCII letters, digits, '.', '_', '/', ':', '+', '-')\n", refusal.Reason, refusal.Key, subject, refusal.Value)
	case model.ReasonRetiredModel:
		fmt.Fprintf(stderr, "model validate: %s: %s resolves historical id %q; refresh WORKFLOW.md with 'spine update --dir %q --write'\n", refusal.Reason, refusal.Key, refusal.Value, repoDir)
	case model.ReasonRouteMismatch:
		fmt.Fprintf(stderr, "model validate: %s: %s candidate %q is active for %s\n", refusal.Reason, refusal.Key, refusal.Value, refusal.Detail)
	case model.ReasonUnmappedDispatch:
		fmt.Fprintf(stderr, "model validate: %s: %s does not map candidate %q\n", refusal.Reason, refusal.Key, refusal.Value)
	default:
		fmt.Fprintf(stderr, "model validate: %s: %s %s %q\n", refusal.Reason, refusal.Key, subject, refusal.Value)
	}
}

func cmdAdopt(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("adopt", flag.ContinueOnError)
	dir := fs.String("dir", ".", "repo root")
	profile := fs.String("profile", "", "override profile detection")
	name := fs.String("name", "", "project name (default: basename of dir)")
	write := fs.Bool("write", false, "apply the plan (default: dry-run)")
	force := fs.Bool("force", false, "regenerate files with unrecognized local edits")
	asJSON := fs.Bool("json", false, "machine-readable plan output")
	if _, ok := parseArgs(fs, args, "adopt", `usage: spine adopt [--dir D] [--profile P] [--name N] [--write] [--force] [--json]`, 0, stderr); !ok {
		return 2
	}
	if *write {
		warnDirty(*dir, stderr)
	}
	res, err := adopt.Run(adopt.Options{Dir: *dir, Profile: *profile, Name: *name, Write: *write, Force: *force})
	if err != nil {
		fmt.Fprintln(stderr, "adopt:", err)
		return 2
	}
	action := func(r update.FileReport) string {
		if r.Preserved {
			return "preserve"
		}
		switch r.State {
		case update.UpToDate:
			return "up-to-date"
		case update.SkippedUnrecognized:
			return "skip"
		default:
			if r.Created {
				return "create"
			}
			return "update"
		}
	}
	if *asJSON {
		type fileJSON struct {
			Path   string `json:"path"`
			Action string `json:"action"`
		}
		type infoJSON struct {
			Path    string `json:"path"`
			Message string `json:"message"`
		}
		payload := struct {
			Profile string     `json:"profile"`
			Dirs    []string   `json:"dirs"`
			Files   []fileJSON `json:"files"`
			Infos   []infoJSON `json:"infos"`
			Pending bool       `json:"pending"`
		}{Profile: res.Profile, Dirs: res.DirsToCreate, Files: []fileJSON{}, Infos: []infoJSON{}, Pending: res.Pending()}
		if payload.Dirs == nil {
			payload.Dirs = []string{}
		}
		for _, r := range res.Reports {
			payload.Files = append(payload.Files, fileJSON{Path: r.Path, Action: action(r)})
		}
		for _, i := range res.Infos {
			payload.Infos = append(payload.Infos, infoJSON{Path: i.Path, Message: i.Message})
		}
		if err := json.NewEncoder(stdout).Encode(payload); err != nil {
			fmt.Fprintln(stderr, "adopt:", err)
			return 2
		}
	} else {
		fmt.Fprintf(stdout, "profile: %s\n", res.Profile)
		fmt.Fprintln(stdout, "plan:")
		for _, d := range res.DirsToCreate {
			fmt.Fprintf(stdout, "  create dir  %s\n", d)
		}
		for _, r := range res.Reports {
			fmt.Fprintf(stdout, "  %-11s %s\n", action(r), r.Path)
			// dry-run only: the T15 human review gate needs to see what
			// would actually land, not just a one-line create/update label.
			if !*write && r.State == update.Pending {
				fmt.Fprint(stdout, r.Diff)
			}
			if r.State == update.SkippedUnrecognized {
				for _, l := range r.Unrecognized {
					fmt.Fprintf(stderr, "    unrecognized: %s\n", l)
				}
			}
		}
		if len(res.Infos) > 0 {
			fmt.Fprintln(stdout, "info:")
			for _, i := range res.Infos {
				fmt.Fprintf(stdout, "  %s: %s\n", i.Path, i.Message)
			}
		}
	}
	if !*write && res.Pending() {
		if !*asJSON {
			fmt.Fprintln(stdout, "rerun with --write to apply")
		}
		return 1
	}
	skipped := false
	for _, r := range res.Reports {
		if r.State == update.SkippedUnrecognized {
			skipped = true
		}
	}
	if skipped {
		return 1
	}
	return 0
}
