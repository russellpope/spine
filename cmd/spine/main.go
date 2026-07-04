// Command spine is the unified-workflow runtime companion.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/russellpope/spine/internal/adopt"
	"github.com/russellpope/spine/internal/adr"
	"github.com/russellpope/spine/internal/doctor"
	"github.com/russellpope/spine/internal/eval"
	"github.com/russellpope/spine/internal/handoff"
	"github.com/russellpope/spine/internal/scaffold"
	"github.com/russellpope/spine/internal/tmpl"
	"github.com/russellpope/spine/internal/update"
)

const usage = `usage: spine <command> [flags]

commands:
  init     scaffold the unified workflow into a repo
  adopt    retrofit a pre-spine repo (dry-run by default; --write applies)
  update   regenerate machine-owned workflow files (dry-run by default; --write applies)
  adr      manage architecture decision records (new, list)
  handoff  manage docs/handoffs (new, list, latest [--fleet DIR])
  eval     manage docs/evals (new, add-run, list)
  doctor   read-only workflow health checks
  version  print the compiled template generation
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
	case "version":
		fmt.Fprintf(stdout, "spine template generation %d\n", tmpl.Version())
		return 0
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n%s", args[0], usage)
		return 2
	}
}

func cmdInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	profile := fs.String("profile", "", "profile: "+strings.Join(tmpl.Profiles(), " | "))
	dir := fs.String("dir", ".", "repo root")
	name := fs.String("name", "", "project name (default: basename of dir)")
	if err := fs.Parse(args); err != nil {
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
	for _, f := range res.Created {
		fmt.Fprintln(stdout, "create:", f)
	}
	for _, f := range res.Skipped {
		fmt.Fprintln(stdout, "skip (exists):", f)
	}
	fmt.Fprintf(stdout, "done: %s -> %s (template_version %d)\n", p, *dir, tmpl.Version())
	return 0
}

func cmdUpdate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "repo root")
	write := fs.Bool("write", false, "apply changes (default: dry-run diff)")
	force := fs.Bool("force", false, "regenerate files with unrecognized local edits (diff shows what gets dropped)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *write {
		warnDirty(*dir, stderr)
	}
	reports, err := update.Run(update.Options{Dir: *dir, Write: *write, Force: *force})
	if err != nil {
		fmt.Fprintln(stderr, "update:", err)
		return 2
	}
	outstanding := 0
	for _, r := range reports {
		switch r.State {
		case update.UpToDate:
			if r.Preserved {
				fmt.Fprintf(stdout, "preserved (hand-authored): %s\n", r.Path)
			} else {
				fmt.Fprintf(stdout, "up-to-date: %s\n", r.Path)
			}
		case update.Pending:
			if *write {
				if r.Created {
					fmt.Fprintf(stdout, "created: %s\n", r.Path)
				} else {
					fmt.Fprintf(stdout, "updated: %s\n", r.Path)
				}
			} else {
				outstanding++
				fmt.Fprint(stdout, r.Diff)
			}
		case update.SkippedUnrecognized:
			outstanding++
			fmt.Fprintf(stderr, "skipped %s — unrecognized local edits (use --force to drop):\n", r.Path)
			for _, l := range r.Unrecognized {
				fmt.Fprintf(stderr, "  %s\n", l)
			}
		}
	}
	if outstanding > 0 {
		return 1
	}
	return 0
}

// warnDirty nudges the user to review post-write diffs with git; git being
// absent or dir not being a repo is fine and silent.
func warnDirty(dir string, stderr io.Writer) {
	out, err := exec.Command("git", "-C", dir, "status", "--porcelain").Output()
	if err == nil && len(bytes.TrimSpace(out)) > 0 {
		fmt.Fprintln(stderr, "warning: repo has uncommitted changes — review the update with git diff afterwards")
	}
}

func cmdADR(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, `usage: spine adr <new|list> [flags]  (adr new [--dir D] [--supersedes N] "Title")`)
		return 2
	}
	switch args[0] {
	case "new":
		fs := flag.NewFlagSet("adr new", flag.ContinueOnError)
		fs.SetOutput(stderr)
		dir := fs.String("dir", ".", "repo root")
		supersedes := fs.Int("supersedes", 0, "ADR number this decision supersedes")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if fs.NArg() != 1 {
			fmt.Fprintln(stderr, `usage: spine adr new [--dir D] [--supersedes N] "Title" (flags before title)`)
			return 2
		}
		path, err := adr.New(*dir, fs.Arg(0), *supersedes)
		if err != nil {
			fmt.Fprintln(stderr, "adr new:", err)
			return 2
		}
		fmt.Fprintln(stdout, path)
		return 0
	case "list":
		fs := flag.NewFlagSet("adr list", flag.ContinueOnError)
		fs.SetOutput(stderr)
		dir := fs.String("dir", ".", "repo root")
		asJSON := fs.Bool("json", false, "machine-readable output")
		if err := fs.Parse(args[1:]); err != nil {
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
		fs.SetOutput(stderr)
		dir := fs.String("dir", ".", "repo root")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if fs.NArg() != 1 {
			fmt.Fprintln(stderr, `usage: spine handoff new [--dir D] "Topic" (flags before topic)`)
			return 2
		}
		path, err := handoff.New(*dir, fs.Arg(0))
		if err != nil {
			fmt.Fprintln(stderr, "handoff new:", err)
			return 2
		}
		fmt.Fprintln(stdout, path)
		return 0
	case "list":
		fs := flag.NewFlagSet("handoff list", flag.ContinueOnError)
		fs.SetOutput(stderr)
		dir := fs.String("dir", ".", "repo root")
		asJSON := fs.Bool("json", false, "machine-readable output")
		if err := fs.Parse(args[1:]); err != nil {
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
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "repo root")
	asJSON := fs.Bool("json", false, "machine-readable output")
	fleet := fs.String("fleet", "", "scan every child repo of DIR instead of one repo")
	if err := fs.Parse(args); err != nil {
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
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "repo root")
	asJSON := fs.Bool("json", false, "machine-readable output")
	if err := fs.Parse(args); err != nil {
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
		fs.SetOutput(stderr)
		dir := fs.String("dir", ".", "repo root")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if fs.NArg() != 1 {
			fmt.Fprintln(stderr, `usage: spine eval new [--dir D] "Title" (flags before title)`)
			return 2
		}
		path, err := eval.New(*dir, fs.Arg(0))
		if err != nil {
			fmt.Fprintln(stderr, "eval new:", err)
			return 2
		}
		fmt.Fprintln(stdout, path)
		return 0
	case "add-run":
		fs := flag.NewFlagSet("eval add-run", flag.ContinueOnError)
		fs.SetOutput(stderr)
		dir := fs.String("dir", ".", "repo root")
		evalRef := fs.String("eval", "", "eval dir name (date prefix optional)")
		name := fs.String("name", "", "run name (becomes runs/<name>.md)")
		if err := fs.Parse(args[1:]); err != nil {
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
		fs.SetOutput(stderr)
		dir := fs.String("dir", ".", "repo root")
		asJSON := fs.Bool("json", false, "machine-readable output")
		if err := fs.Parse(args[1:]); err != nil {
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

func cmdAdopt(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("adopt", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "repo root")
	profile := fs.String("profile", "", "override profile detection")
	name := fs.String("name", "", "project name (default: basename of dir)")
	write := fs.Bool("write", false, "apply the plan (default: dry-run)")
	force := fs.Bool("force", false, "regenerate files with unrecognized local edits")
	asJSON := fs.Bool("json", false, "machine-readable plan output")
	if err := fs.Parse(args); err != nil {
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
