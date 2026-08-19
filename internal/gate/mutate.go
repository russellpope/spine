package gate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// This file is the go pack's mutation battery: a Go port of the runner the
// /model-eval skill shipped as Python (ADR 0015 item 5, superseding ADR 0013
// items 2 and 4). The normative convention — the ten runnable classes, the
// record format, the two kill rates — stays in
// docs/mutation-battery-checklist.md; this file only executes it.
//
// The battery is the pack's advisory lane: it reports one row per probe and
// never fails a stage on survivors. The single failing condition is its own
// negative control — the unmutated tree must be green before any probe is
// believed, because a red baseline makes every KILLED row meaningless.
//
// The tree under --dir is never mutated. The tracked files are copied into a
// temp directory, every probe runs there, and the copy is removed afterwards.

// The mutation battery's environment-only knobs. None of them is a
// gate_pack_config key — the gate_pack_config key set is closed (ADR 0015),
// so spine update never renders these into the region; an operator sets them
// on the stage or in the shell, like SPINE_GATE_CLEANUP_FUNCS.
const (
	// MutateSpecVar names the per-tree mutation spec: a path relative to
	// --dir, or absolute. Unset means DefaultMutateSpec under --dir.
	MutateSpecVar = "SPINE_GATE_MUTATE_SPEC"
	// MutateVerifyVar overrides the verify command. It is run with sh -c in
	// the copied tree; exit 0 is green (the probe SURVIVED), non-zero is red
	// (KILLED). A custom command cannot distinguish a broken build from a
	// failing test, so it never reports BUILD-ERR.
	MutateVerifyVar = "SPINE_GATE_MUTATE_VERIFY"
	// MutateTimeoutVar overrides the per-command timeout (a Go duration).
	MutateTimeoutVar = "SPINE_GATE_MUTATE_TIMEOUT"

	// DefaultMutateSpec is the spec path used when MutateSpecVar is unset.
	DefaultMutateSpec = "docs/mutation-spec.json"

	defaultMutateBuild   = "go build ./..."
	defaultMutateTest    = "go test ./..."
	defaultMutateTimeout = 15 * time.Minute
)

// Probe results, verbatim from the checklist's record format.
const (
	resultKilled   = "KILLED"
	resultSurvived = "SURVIVED"
	resultNoSite   = "NO-SITE"
	resultBuildErr = "BUILD-ERR"
)

// mutation is one probe of the spec: the same JSON object shape the Python
// runner read, so a spec authored for either runner runs on both. find is a
// LITERAL string and its first occurrence is the site.
type mutation struct {
	ID         string `json:"id"`
	File       string `json:"file"`
	Find       string `json:"find"`
	Replace    string `json:"replace"`
	ReportOnly bool   `json:"report_only"`
	Desc       string `json:"desc"`
}

// checkMutate is the mutate check class. It returns a Report rather than
// bare findings because the battery owns its own stage judgement: its rows
// are advisory, and its summary is the two kill rates, not a finding count.
func checkMutate(dir string, cfg Config) (Report, error) {
	spec, err := loadMutationSpec(dir, cfg)
	if err != nil {
		return Report{}, err
	}
	timeout, err := mutateTimeout(cfg)
	if err != nil {
		return Report{}, err
	}
	work, err := copyTree(dir)
	if err != nil {
		return Report{}, err
	}
	// The working copy is removed on the way out, except when the control
	// fails: that failure happened in the copy, and the copy is a
	// tracked-files-only tree that may not reproduce in the repo under --dir,
	// so it is left on disk for the operator and named in the finding.
	keep := false
	defer func() {
		if !keep {
			_ = os.RemoveAll(work)
		}
	}()

	verify := newVerifier(cfg, work, timeout)

	// The negative control: no probe is believed against a red baseline.
	green, out, err := verify.green()
	if err != nil {
		return Report{}, err
	}
	if !green {
		keep = true
		msg := fmt.Sprintf("control failed: unmutated tree is not green (%s) — no probes run; fix the tree or set %s. Working copy kept at %s",
			verify.describe(), MutateVerifyVar, work)
		if tail := outputTail(out); tail != "" {
			msg += "\nverify output (last " + fmt.Sprint(maxTailLines) + " lines):\n" + tail
		}
		return Report{
			Findings: []Finding{{
				Severity: SeverityError,
				Message:  msg,
				File:     ".",
				Line:     0,
				Code:     Code("mutate"),
			}},
			Summary: Code("mutate") + ": control failed: unmutated tree is not green — no probes run",
		}, nil
	}

	rep := Report{Advisory: true}
	tally := map[string]int{resultKilled: 0, resultSurvived: 0, resultNoSite: 0, resultBuildErr: 0}
	scorable := map[string]int{resultKilled: 0, resultSurvived: 0}
	reportOnlyValid := 0
	var survivors []string

	for _, m := range spec {
		target := filepath.Join(work, filepath.FromSlash(m.File))
		pristine, readErr := os.ReadFile(target)
		result, line := resultNoSite, 0
		switch {
		case readErr != nil:
			// The file is not in the tracked tree under --dir: spec drift,
			// not a probe result, and indistinguishable from a missing site.
		case !bytes.Contains(pristine, []byte(m.Find)):
		default:
			line = literalLine(pristine, m.Find)
			mutated := bytes.Replace(pristine, []byte(m.Find), []byte(m.Replace), 1)
			if err := os.WriteFile(target, mutated, 0o644); err != nil {
				return Report{}, fmt.Errorf("applying mutation %s to the working copy: %w", m.ID, err)
			}
			result, err = verify.probe()
			if wErr := os.WriteFile(target, pristine, 0o644); wErr != nil {
				return Report{}, fmt.Errorf("restoring %s in the working copy: %w", m.File, wErr)
			}
			if err != nil {
				return Report{}, err
			}
		}

		tally[result]++
		if _, valid := scorable[result]; valid {
			if m.ReportOnly {
				reportOnlyValid++
			} else {
				scorable[result]++
			}
		}
		note := m.Desc
		if note == "" {
			note = m.File
		}
		msg := fmt.Sprintf("%s %s %s", m.ID, result, note)
		if m.ReportOnly {
			msg += " [report-only]"
		}
		if result == resultSurvived {
			survivors = append(survivors, fmt.Sprintf("  - %s: %s", m.ID, note))
		}
		rep.Findings = append(rep.Findings, Finding{
			Severity: mutateSeverity(result),
			Message:  msg,
			File:     m.File,
			Line:     line,
			Code:     Code("mutate"),
		})
	}

	rep.Summary = Code("mutate") + ": " + strings.Join([]string{
		fmt.Sprintf("kill rate (raw): %s   (excluded: %d no-site, %d build-err)",
			rate(tally[resultKilled], tally[resultKilled]+tally[resultSurvived]),
			tally[resultNoSite], tally[resultBuildErr]),
		fmt.Sprintf("kill rate (scorable): %s   (excluded: %d report-only, %d no-site, %d build-err)",
			rate(scorable[resultKilled], scorable[resultKilled]+scorable[resultSurvived]),
			reportOnlyValid, tally[resultNoSite], tally[resultBuildErr]),
	}, "; ")
	if len(survivors) > 0 {
		rep.Detail = append([]string{"surviving mutations (behaviour the suite cannot see):"}, survivors...)
	}
	return rep, nil
}

// mutateSeverity maps a probe result to a results-contract severity so a
// consumer can filter rows without parsing the message: a blind spot is the
// only row that asks for attention, a kill is the good news, and the two
// invalid-probe results are neither.
func mutateSeverity(result string) string {
	if result == resultSurvived {
		return "warn"
	}
	return "info"
}

// rate formats one kill rate the way the Python runner did: "K/V = P%",
// with an empty denominator reported as 0%.
func rate(killed, valid int) string {
	pct := 0.0
	if valid > 0 {
		pct = 100.0 * float64(killed) / float64(valid)
	}
	return fmt.Sprintf("%d/%d = %.0f%%", killed, valid, pct)
}

// literalLine is the 1-based line of the first occurrence of find in src.
func literalLine(src []byte, find string) int {
	i := bytes.Index(src, []byte(find))
	if i < 0 {
		return 0
	}
	return 1 + bytes.Count(src[:i], []byte("\n"))
}

// loadMutationSpec reads the effective spec. A missing or unparseable spec
// is misconfiguration (exit 2) naming both the variable and the path,
// because a battery with no probes is a silently empty gate.
func loadMutationSpec(dir string, cfg Config) ([]mutation, error) {
	rel := strings.TrimSpace(cfg.env(MutateSpecVar))
	if rel == "" {
		rel = DefaultMutateSpec
	}
	path := rel
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, filepath.FromSlash(rel))
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("mutation spec %s: %v (set %s to a path relative to --dir or absolute; default %s)",
			path, err, MutateSpecVar, DefaultMutateSpec)
	}
	var spec []mutation
	if err := json.Unmarshal(raw, &spec); err != nil {
		return nil, fmt.Errorf("mutation spec %s (%s): %v", path, MutateSpecVar, err)
	}
	for i, m := range spec {
		if m.ID == "" || m.File == "" || m.Find == "" {
			return nil, fmt.Errorf("mutation spec %s (%s): entry %d needs id, file and find", path, MutateSpecVar, i)
		}
	}
	return spec, nil
}

// mutateTimeout is the per-command timeout, 15 minutes by default (the
// Python runner's 900 s).
func mutateTimeout(cfg Config) (time.Duration, error) {
	v := strings.TrimSpace(cfg.env(MutateTimeoutVar))
	if v == "" {
		return defaultMutateTimeout, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("%s=%q is not a positive Go duration (e.g. 15m)", MutateTimeoutVar, v)
	}
	return d, nil
}

// verifier runs the verify command in the working copy. The default is two
// phases so a mutation that breaks compilation is reported as BUILD-ERR (an
// invalid probe, per the checklist) rather than counted as a kill; a custom
// command is one phase and so has no BUILD-ERR outcome.
type verifier struct {
	dir     string
	timeout time.Duration
	custom  string
}

func newVerifier(cfg Config, dir string, timeout time.Duration) verifier {
	return verifier{dir: dir, timeout: timeout, custom: strings.TrimSpace(cfg.env(MutateVerifyVar))}
}

func (v verifier) describe() string {
	if v.custom != "" {
		return MutateVerifyVar + "=" + v.custom
	}
	return defaultMutateBuild + " && " + defaultMutateTest
}

// green reports whether the tree in the working copy passes verification —
// the negative control when the tree is unmutated — and returns the output of
// the phase that decided it.
func (v verifier) green() (bool, string, error) {
	if v.custom != "" {
		return v.run(v.custom)
	}
	built, out, err := v.run(defaultMutateBuild)
	if err != nil || !built {
		return false, out, err
	}
	return v.run(defaultMutateTest)
}

// probe verifies a mutated working copy and names the outcome.
func (v verifier) probe() (string, error) {
	if v.custom != "" {
		ok, _, err := v.run(v.custom)
		if err != nil {
			return "", err
		}
		if ok {
			return resultSurvived, nil
		}
		return resultKilled, nil
	}
	built, _, err := v.run(defaultMutateBuild)
	if err != nil {
		return "", err
	}
	if !built {
		return resultBuildErr, nil
	}
	passed, _, err := v.run(defaultMutateTest)
	if err != nil {
		return "", err
	}
	if passed {
		return resultSurvived, nil
	}
	return resultKilled, nil
}

// run executes one shell command in the working copy and reports whether it
// exited 0, along with its combined output — the only diagnostic an operator
// gets when the control fails. Only a failure to start the command (or a
// timeout) is an error; a non-zero exit is the answer, not a problem.
func (v verifier) run(command string) (bool, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), v.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = v.dir
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	if ctx.Err() != nil {
		return false, out.String(), fmt.Errorf("verify command %q timed out after %s (raise %s)", command, v.timeout, MutateTimeoutVar)
	}
	if err == nil {
		return true, out.String(), nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return false, out.String(), nil
	}
	return false, out.String(), fmt.Errorf("running verify command %q: %v", command, err)
}

// outputTail is the last maxTailLines lines of a verify command's output,
// trimmed — enough to name the failing test without pasting a whole build log
// into a finding message.
const maxTailLines = 20

func outputTail(out string) string {
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) > maxTailLines {
		lines = lines[len(lines)-maxTailLines:]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// copyTree copies the tree under dir into a fresh temp directory: the
// tracked files when dir is a git repo, everything but .git otherwise. The
// battery mutates only this copy, never the repo under --dir.
func copyTree(dir string) (string, error) {
	work, err := os.MkdirTemp("", "spine-mutate-")
	if err != nil {
		return "", fmt.Errorf("creating the mutation working copy: %w", err)
	}
	names, err := trackedFiles(dir)
	if err != nil {
		_ = os.RemoveAll(work)
		return "", err
	}
	for _, rel := range names {
		if err := copyOne(filepath.Join(dir, filepath.FromSlash(rel)), filepath.Join(work, filepath.FromSlash(rel))); err != nil {
			_ = os.RemoveAll(work)
			return "", err
		}
	}
	return work, nil
}

// trackedFiles lists the tree's files as slash-separated relative paths.
func trackedFiles(dir string) ([]string, error) {
	out, err := gitLsFiles(dir)
	if err == nil {
		return out, nil
	}
	var names []string
	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		names = append(names, filepath.ToSlash(rel))
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("reading the tree under --dir %s: %w", dir, walkErr)
	}
	return names, nil
}

// copyOne copies one regular file, preserving its mode. A tracked path that
// no longer exists in the working tree is skipped rather than fatal.
func copyOne(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil || !info.Mode().IsRegular() {
		return nil
	}
	raw, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("copying %s into the mutation working copy: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("copying %s into the mutation working copy: %w", src, err)
	}
	if err := os.WriteFile(dst, raw, info.Mode().Perm()); err != nil {
		return fmt.Errorf("copying %s into the mutation working copy: %w", src, err)
	}
	return nil
}
