package gate

import (
	"bytes"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"
)

// buildOutputsKey is the gate_pack_config key holding the declared build
// outputs; it reaches the stage as SPINE_GATE_BUILD_OUTPUTS.
const buildOutputsKey = "build_outputs"

// checkGitignoreControl reports the two arms of the ignore contract:
//
//	arm 1 (declared build output not ignored) — every path in
//	SPINE_GATE_BUILD_OUTPUTS must be ignored at that path, so a build
//	output written where the repo actually writes it never becomes a
//	tracked file.
//	arm 2 (entry point ignored) — no `package main` source file under
//	--dir may be ignored, the hidden-entry-point control: an ignored
//	main.go is invisible to review even in a repo that correctly ignores
//	its binaries.
//
// The arms report distinctly by message; both carry the same check class
// code. An unset or empty SPINE_GATE_BUILD_OUTPUTS is misconfiguration:
// this class has nothing to check without the declared list.
func checkGitignoreControl(dir string, cfg Config) ([]Finding, error) {
	raw, _ := cfg.Get(buildOutputsKey)
	outputs := splitList(raw)
	if len(outputs) == 0 {
		return nil, fmt.Errorf("%s is unset or empty: list the declared build outputs as comma-separated paths relative to --dir (e.g. bin/spine,dist/)", EnvVar(buildOutputsKey))
	}
	var findings []Finding
	for _, out := range outputs {
		// --no-index: arm 1 asks whether the path *would* be ignored, which
		// is a question about the ignore rules alone. A declared build
		// output usually does not exist in a clean checkout.
		ignored, err := gitCheckIgnore(dir, out, true)
		if err != nil {
			return nil, err
		}
		if !ignored {
			findings = append(findings, Finding{
				Severity: SeverityError,
				Message:  fmt.Sprintf("declared build output not ignored: %s is not matched by any ignore rule at that path (%s)", out, EnvVar(buildOutputsKey)),
				File:     out,
				Line:     0,
				Code:     cfg.Code("gitignore-control"),
			})
		}
	}
	mains, err := mainPackageFiles(dir)
	if err != nil {
		return nil, err
	}
	for _, rel := range mains {
		// No --no-index here: a tracked file is not ignored no matter what
		// the rules say, and only an untracked-because-ignored entry point
		// is the bug this arm exists to catch.
		ignored, err := gitCheckIgnore(dir, rel, false)
		if err != nil {
			return nil, err
		}
		if ignored {
			findings = append(findings, Finding{
				Severity: SeverityError,
				Message:  "ignored entry point: a package main source file is matched by an ignore rule and so is invisible to review",
				File:     rel,
				Line:     1,
				Code:     cfg.Code("gitignore-control"),
			})
		}
	}
	return findings, nil
}

// mainPackageFiles returns the working-tree .go files under dir whose
// package clause is `package main`, relative to dir in slash form. Only
// The toolchain's ignore set is skipped except testdata, which is walked on
// purpose, since an ignored testdata entry point is hidden just the same; a
// testdata file that does not parse is skipped rather than failing the
// walk, because a template or deliberately broken source under testdata is
// a normal thing for a Go repo to carry.
func mainPackageFiles(dir string) ([]string, error) {
	var mains []string
	fset := token.NewFileSet()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDirExceptTestdata(d.Name()) && path != dir {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		rel := relSlash(dir, path)
		file, perr := parser.ParseFile(fset, path, nil, parser.PackageClauseOnly)
		if perr != nil {
			if underTestdata(rel) {
				return nil
			}
			return fmt.Errorf("parsing %s: %w", rel, perr)
		}
		if file.Name != nil && file.Name.Name == "main" {
			mains = append(mains, rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return mains, nil
}

// gitCheckIgnore reports whether path (relative to dir) is ignored, asking
// git so the answer accounts for every ignore source git itself honours.
// A dir that is not a git repo is misconfiguration, not a finding.
func gitCheckIgnore(dir, path string, noIndex bool) (bool, error) {
	args := []string{"check-ignore", "-q"}
	if noIndex {
		args = append(args, "--no-index")
	}
	args = append(args, "--", path)
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exit *exec.ExitError
	// git check-ignore: 0 ignored, 1 not ignored, 128 error.
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return false, nil
	}
	msg := strings.TrimSpace(stderr.String())
	if msg == "" {
		msg = err.Error()
	}
	return false, fmt.Errorf("--dir %s: asking git whether %s is ignored: %s", dir, path, msg)
}

// splitList parses the comma-separated list form shared by the
// configuration values that carry several paths. Blank entries are ignored.
func splitList(raw string) []string {
	var out []string
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry != "" {
			out = append(out, entry)
		}
	}
	return out
}
