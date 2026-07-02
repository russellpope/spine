// Package scaffold implements spine init: profile detection and first-time
// emission of the workflow file set.
package scaffold

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/russellpope/spine/internal/fsutil"
	"github.com/russellpope/spine/internal/tmpl"
)

// Result reports what Init did, as repo-relative paths.
type Result struct {
	Created []string
	Skipped []string
}

// Files is the scaffolded set, in emission order. Shared with update.
var Files = []struct{ TmplName, RelPath string }{
	{"CLAUDE.md.tmpl", "CLAUDE.md"},
	{"WORKFLOW.md.tmpl", "WORKFLOW.md"},
	{"harness-interface.md", "docs/harness-interface.md"},
	{"issues-README.md", "docs/issues/README.md"},
	{"issue.tmpl.md", "docs/issues/_template.md"},
	{"adr-README.md", "docs/adr/README.md"},
}

// DetectProfile inspects dir and returns a profile when signals are unambiguous.
func DetectProfile(dir string) (string, bool) {
	has := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}
	switch {
	case has("Cargo.toml"):
		return "rust", true
	case has("go.mod"):
		return "go-service", true
	case has("pyproject.toml"), has("setup.py"):
		return "py-tool", true
	}
	for _, pat := range []string{"*.pptx", "*.key"} {
		if m, _ := filepath.Glob(filepath.Join(dir, pat)); len(m) > 0 {
			return "presentation", true
		}
	}
	if raw, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil {
		s := string(raw)
		for _, fw := range []string{`"react"`, `"vue"`, `"svelte"`, `"next"`} {
			if strings.Contains(s, fw) {
				return "ui", true
			}
		}
	}
	return "", false
}

// Init scaffolds dir with the current-generation file set; existing files are
// skipped, never overwritten.
func Init(dir, profile, name string) (Result, error) {
	reviewers, harness, err := tmpl.Defaults(profile)
	if err != nil {
		return Result{}, err
	}
	if name == "" {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return Result{}, err
		}
		name = filepath.Base(abs)
	}
	for _, d := range []string{"docs/specs", "docs/adr", "docs/issues", "docs/handoffs"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			return Result{}, err
		}
	}
	v := tmpl.Values{Project: name, Profile: profile, Reviewers: reviewers, Harness: harness, Version: tmpl.Version()}
	var res Result
	for _, f := range Files {
		dst := filepath.Join(dir, f.RelPath)
		if _, err := os.Stat(dst); err == nil {
			res.Skipped = append(res.Skipped, f.RelPath)
			continue
		}
		content, err := tmpl.Render("current", f.TmplName, v)
		if err != nil {
			return res, err
		}
		if err := fsutil.WriteFileAtomic(dst, []byte(content)); err != nil {
			return res, err
		}
		res.Created = append(res.Created, f.RelPath)
	}
	return res, nil
}
