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

	"github.com/russellpope/spine/internal/adr"
	"github.com/russellpope/spine/internal/doctor"
	"github.com/russellpope/spine/internal/scaffold"
	"github.com/russellpope/spine/internal/tmpl"
	"github.com/russellpope/spine/internal/update"
)

const usage = `usage: spine <command> [flags]

commands:
  init     scaffold the unified workflow into a repo
  update   regenerate machine-owned workflow files (dry-run by default; --write applies)
  adr      manage architecture decision records (new, list)
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
	case "doctor":
		return cmdDoctor(args[1:], stdout, stderr)
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
			fmt.Fprintf(stdout, "up-to-date: %s\n", r.Path)
		case update.Pending:
			if *write {
				fmt.Fprintf(stdout, "updated: %s\n", r.Path)
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
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		entries, err := adr.List(*dir)
		if err != nil {
			fmt.Fprintln(stderr, "adr list:", err)
			return 2
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
	if len(findings) > 0 {
		return 1
	}
	return 0
}
