// Package adopt retrofits a pre-spine repo: compose init's dir creation and
// update's claim/regenerate machinery under one dry-runnable plan. It maps
// nothing and migrates nothing: legacy trees stay put and are reported as
// info (ADR 0008).
package adopt

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/russellpope/spine/internal/adr"
	"github.com/russellpope/spine/internal/scaffold"
	"github.com/russellpope/spine/internal/tmpl"
	"github.com/russellpope/spine/internal/update"
)

// Options configures Run. Zero value = dry-run detection on ".".
type Options struct {
	Dir     string
	Profile string
	Name    string
	Write   bool
	Force   bool
}

// Info is a transparency note in the plan (never affects exit codes).
type Info struct {
	Path    string
	Message string
}

// Result is the adopt plan (and with Write, what was applied).
type Result struct {
	Profile      string
	DirsToCreate []string
	Reports      []update.FileReport
	Infos        []Info
}

// Pending reports whether applying would change anything.
func (r Result) Pending() bool {
	if len(r.DirsToCreate) > 0 {
		return true
	}
	for _, rep := range r.Reports {
		if rep.State != update.UpToDate {
			return true
		}
	}
	return false
}

// dirsKnown is every docs/ entry spine has a concept of; anything else in
// docs/ is reported as "not spine's" for plan transparency.
var dirsKnown = map[string]bool{
	"specs": true, "adr": true, "issues": true, "handoffs": true,
	"evals": true, "superpowers": true, "harness-interface.md": true,
}

// Run plans (and with opts.Write, applies) the retrofit.
func Run(opts Options) (Result, error) {
	if opts.Dir == "" {
		opts.Dir = "."
	}
	profile := opts.Profile
	if profile == "" {
		detected, ok := scaffold.DetectProfile(opts.Dir)
		if !ok {
			return Result{}, fmt.Errorf("cannot detect profile for %s; pass --profile", opts.Dir)
		}
		profile = detected
	}
	if _, _, err := tmpl.Defaults(profile); err != nil {
		return Result{}, err
	}
	res := Result{Profile: profile}
	for _, d := range tmpl.ProfileDirs(profile) {
		if fi, err := os.Stat(filepath.Join(opts.Dir, d)); err != nil || !fi.IsDir() {
			res.DirsToCreate = append(res.DirsToCreate, d)
		}
	}
	if opts.Write {
		for _, d := range res.DirsToCreate {
			if err := os.MkdirAll(filepath.Join(opts.Dir, d), 0o755); err != nil {
				return res, err
			}
		}
	}
	reports, err := update.Run(update.Options{
		Dir: opts.Dir, Write: opts.Write, Force: opts.Force,
		AdoptProfile: profile, AdoptName: opts.Name,
	})
	if err != nil {
		return res, err
	}
	res.Reports = reports
	res.Infos = gatherInfos(opts.Dir)
	return res, nil
}

func gatherInfos(dir string) []Info {
	var infos []Info
	for _, sub := range []string{"specs", "plans"} {
		glob := filepath.Join(dir, "docs", "superpowers", sub, "*.md")
		if m, _ := filepath.Glob(glob); len(m) > 0 {
			infos = append(infos, Info{Path: "docs/superpowers/" + sub,
				Message: fmt.Sprintf("%d artifact(s) in legacy location — left alone; new work goes to docs/specs/", len(m))})
		}
	}
	if entries, err := adr.List(dir); err == nil {
		preSpine := 0
		for _, e := range entries {
			if !e.HasFrontMatter {
				preSpine++
			}
		}
		if preSpine > 0 {
			infos = append(infos, Info{Path: "docs/adr",
				Message: fmt.Sprintf("%d pre-spine ADR(s) (no front matter) — left alone; spine conventions apply to new ADRs", preSpine)})
		}
	}
	if des, err := os.ReadDir(filepath.Join(dir, "docs")); err == nil {
		var unknown []string
		for _, de := range des {
			if !dirsKnown[de.Name()] {
				unknown = append(unknown, "docs/"+de.Name())
			}
		}
		sort.Strings(unknown)
		if len(unknown) > 0 {
			infos = append(infos, Info{Path: strings.Join(unknown, ", "),
				Message: "not spine's — left alone"})
		}
	}
	return infos
}
