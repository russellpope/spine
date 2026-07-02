// Command spine is the unified-workflow runtime companion.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/russellpope/spine/internal/scaffold"
	"github.com/russellpope/spine/internal/tmpl"
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
	fmt.Fprintln(stderr, "update: not implemented yet")
	return 2
}

func cmdADR(args []string, stdout, stderr io.Writer) int {
	fmt.Fprintln(stderr, "adr: not implemented yet")
	return 2
}

func cmdDoctor(args []string, stdout, stderr io.Writer) int {
	fmt.Fprintln(stderr, "doctor: not implemented yet")
	return 2
}
