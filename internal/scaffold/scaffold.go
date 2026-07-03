// Package scaffold implements spine init: profile detection and first-time
// emission of the workflow file set.
package scaffold

import (
	"fmt"
	"os"
	"os/exec"
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

// DetectProfile inspects dir and returns a profile when signals are
// unambiguous. Precedence: code signals, then infra, then knowledge — a repo
// with go.mod AND ansible/ is a go service that carries some automation.
func DetectProfile(dir string) (string, bool) {
	has := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}
	hasDir := func(name string) bool {
		fi, err := os.Stat(filepath.Join(dir, name))
		return err == nil && fi.IsDir()
	}
	switch {
	case has("Cargo.toml"):
		return "rust", true
	case has("go.mod"):
		return "go-service", true
	case has("pyproject.toml"), has("setup.py"):
		return "py-tool", true
	case has("Package.swift"):
		return "swift", true
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
	if m, _ := filepath.Glob(filepath.Join(dir, "*.xcodeproj")); len(m) > 0 {
		return "swift", true
	}
	// infra signals live one level below root (the home-lab-admin lesson).
	if has(filepath.Join("ansible", "ansible.cfg")) || hasDir(filepath.Join("ansible", "playbooks")) ||
		hasDir("helm") || hasDir("terraform") || hasDir("k8s") {
		return "infra", true
	}
	if hasDir(".obsidian") || mdMajority(dir) {
		return "knowledge", true
	}
	return "", false
}

// mdMajority reports whether ≥80% of git-tracked files are .md. Repos
// without git (or with nothing tracked) never qualify — .obsidian is then
// the only knowledge signal.
func mdMajority(dir string) bool {
	out, err := exec.Command("git", "-C", dir, "ls-files").Output()
	if err != nil {
		return false
	}
	var md, total int
	for _, f := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if f == "" {
			continue
		}
		total++
		if strings.HasSuffix(strings.ToLower(f), ".md") {
			md++
		}
	}
	return total > 0 && md*100 >= total*80
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
	for _, d := range tmpl.ProfileDirs(profile) {
		target := filepath.Join(dir, d)
		if err := os.MkdirAll(target, 0o755); err != nil {
			return Result{}, fmt.Errorf("mkdir %s: %w", target, err)
		}
	}
	v := tmpl.Values{Project: name, Profile: profile, Reviewers: reviewers, Harness: harness, Version: tmpl.Version()}
	var res Result
	for _, f := range Files {
		if !tmpl.ProfileOwns(profile, f.RelPath) {
			continue
		}
		dst := filepath.Join(dir, f.RelPath)
		if _, err := os.Stat(dst); err == nil {
			res.Skipped = append(res.Skipped, f.RelPath)
			continue
		} else if !os.IsNotExist(err) {
			return res, fmt.Errorf("stat %s: %w", dst, err)
		}
		content, err := tmpl.Render("current", f.TmplName, v)
		if err != nil {
			return res, fmt.Errorf("render %s: %w", f.TmplName, err)
		}
		if err := fsutil.WriteFileAtomic(dst, []byte(content)); err != nil {
			return res, fmt.Errorf("write %s: %w", dst, err)
		}
		res.Created = append(res.Created, f.RelPath)
	}
	return res, nil
}
