package update

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/russellpope/spine/internal/fsutil"
	"github.com/russellpope/spine/internal/tmpl"
)

// FileState classifies what update would do to one file.
type FileState int

const (
	UpToDate FileState = iota
	Pending
	SkippedUnrecognized
)

// FileReport is the per-file outcome. newContent stays unexported: only Run
// writes it, and only for Pending files.
type FileReport struct {
	Path         string
	State        FileState
	Diff         string
	Unrecognized []string
	// Created is true when the file did not exist on disk at plan time, so a
	// Pending state means "will be created" rather than "will be updated".
	Created bool
	// Preserved is true for a legacyPreserve file (docs/adr/README.md) whose
	// unrecognized hand-authored content was left as-is rather than flagged.
	// Only set when State == UpToDate. --force clears this and regenerates.
	Preserved  bool
	newContent string
}

// Options configures Run. Zero value = dry-run on ".". AdoptProfile switches
// on adopt mode: a missing WORKFLOW.md is synthesized from that profile's
// defaults (project name = AdoptName, else the dir basename) instead of
// being a hard error. Set only by spine adopt.
type Options struct {
	Dir          string
	Write        bool
	Force        bool
	AdoptProfile string
	AdoptName    string
}

const (
	markerBegin = "<!-- spine:begin"
	markerEnd   = "<!-- spine:end -->"
)

// simple machine-owned files: regenerate wholesale, no key extraction.
// inGen0 marks files whose gen0 content differs from current. legacyPreserve
// marks the one file (docs/adr/README.md) where unrecognized hand-authored
// content is a deliberate choice, not drift: ADR 0009.
var simpleFiles = []struct {
	tmplName, relPath string
	inGen0            bool
	legacyPreserve    bool
}{
	{"harness-interface.md", "docs/harness-interface.md", true, false},
	{"issues-README.md", "docs/issues/README.md", false, false},
	{"issue.tmpl.md", "docs/issues/_template.md", false, false},
	{"adr-README.md", "docs/adr/README.md", false, true},
}

// Run plans (and with opts.Write, applies) regeneration of every managed file.
func Run(opts Options) ([]FileReport, error) {
	if opts.Dir == "" {
		opts.Dir = "."
	}
	wf, vals, gen, err := planWorkflow(opts)
	if err != nil {
		return nil, err
	}
	reports := []FileReport{wf}
	cl, err := planClaude(opts.Dir, gen, vals)
	if err != nil {
		return nil, err
	}
	reports = append(reports, cl)
	legacyPreserve := map[string]bool{}
	for _, f := range simpleFiles {
		if f.legacyPreserve {
			legacyPreserve[f.relPath] = true
		}
		if !tmpl.ProfileOwns(vals.Profile, f.relPath) {
			continue
		}
		r, err := planSimple(opts.Dir, gen, f.tmplName, f.relPath, f.inGen0, vals)
		if err != nil {
			return nil, err
		}
		reports = append(reports, r)
	}
	// docs/evals/README.md is opt-in machine-owned: managed only where the
	// convention is in use (the directory exists); never created by init/adopt.
	if fi, err := os.Stat(filepath.Join(opts.Dir, "docs", "evals")); err == nil && fi.IsDir() {
		r, err := planSimple(opts.Dir, gen, "evals-README.md", "docs/evals/README.md", false, vals)
		if err != nil {
			return nil, err
		}
		reports = append(reports, r)
	}
	// policy: unrecognized edits skip the file unless --force; files with no
	// regenerable content (nil newContent) stay skipped regardless. The one
	// exception is legacyPreserve (docs/adr/README.md, ADR 0009): a
	// hand-authored index is a deliberate choice, not drift, so it's treated
	// as up-to-date rather than skipped/warned — --force is the explicit
	// opt-in to regenerate it from the template.
	for i := range reports {
		r := &reports[i]
		if len(r.Unrecognized) > 0 {
			if legacyPreserve[r.Path] && !opts.Force {
				r.State = UpToDate
				r.Preserved = true
				r.Diff = ""
				continue
			}
			if opts.Force && r.newContent != "" {
				r.State = Pending
			} else {
				r.State = SkippedUnrecognized
			}
		}
	}
	if opts.Write {
		for i := range reports {
			r := &reports[i]
			if r.State != Pending {
				continue
			}
			dst := filepath.Join(opts.Dir, r.Path)
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return reports, err
			}
			if err := fsutil.WriteFileAtomic(dst, []byte(r.newContent)); err != nil {
				return reports, err
			}
		}
	}
	return reports, nil
}

func planWorkflow(opts Options) (FileReport, tmpl.Values, string, error) {
	report := FileReport{Path: "WORKFLOW.md"}
	path := filepath.Join(opts.Dir, "WORKFLOW.md")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) && opts.AdoptProfile != "" {
		project := opts.AdoptName
		if project == "" {
			abs, aerr := filepath.Abs(opts.Dir)
			if aerr != nil {
				return report, tmpl.Values{}, "", aerr
			}
			project = filepath.Base(abs)
		}
		defRev, defHarness, derr := tmpl.Defaults(opts.AdoptProfile)
		if derr != nil {
			return report, tmpl.Values{}, "", derr
		}
		vals := tmpl.Values{Project: project, Profile: opts.AdoptProfile,
			Reviewers: defRev, Harness: defHarness, Version: tmpl.Version()}
		newContent, rerr := tmpl.Render("current", "WORKFLOW.md.tmpl", vals)
		if rerr != nil {
			return report, tmpl.Values{}, "", rerr
		}
		report.State = Pending
		report.Created = true
		report.Diff = Diff(report.Path, "", newContent)
		report.newContent = newContent
		return report, vals, "current", nil
	}
	if err != nil {
		return report, tmpl.Values{}, "", fmt.Errorf("read %s (run spine init first?): %w", path, err)
	}
	old := string(raw)
	keys := ExtractKeys(old)
	gen := "gen0"
	if tv := keys["template_version"]; tv != "" {
		// A stamped generation newer than what this binary compiles is never
		// "current" — that would silently downgrade the file. Non-integer
		// stamps fall through to the existing current-gen treatment.
		if n, err := strconv.Atoi(tv); err == nil && n > tmpl.Version() {
			return report, tmpl.Values{}, "", fmt.Errorf(
				"WORKFLOW.md is template generation %d but this spine binary compiles generation %d — upgrade spine (make install in ~/Projects/github.com/spine)",
				n, tmpl.Version())
		}
		gen = "current"
	}
	abs, err := filepath.Abs(opts.Dir)
	if err != nil {
		return report, tmpl.Values{}, "", err
	}
	project := ProjectFromWorkflow(old, filepath.Base(abs))
	profile := keys["profile"]
	if profile == "" {
		return report, tmpl.Values{}, "", fmt.Errorf("%s has no profile: line", path)
	}
	defRev, defHarness, err := tmpl.Defaults(profile)
	if err != nil {
		return report, tmpl.Values{}, "", err
	}
	vals := tmpl.Values{Project: project, Profile: profile, Reviewers: defRev, Harness: defHarness, Version: tmpl.Version()}

	// unrecognized detection: what the old generation would look like with
	// every extracted key applied — anything beyond that is a local edit.
	expectedOld, err := tmpl.Render(gen, "WORKFLOW.md.tmpl", vals)
	if err != nil {
		return report, tmpl.Values{}, "", err
	}
	for k, v := range keys {
		expectedOld = setKey(expectedOld, k, v)
	}
	report.Unrecognized = unrecognizedLines(old, expectedOld)

	choices, err := Choices(keys, gen, project)
	if err != nil {
		return report, tmpl.Values{}, "", err
	}
	newContent, err := tmpl.Render("current", "WORKFLOW.md.tmpl", vals)
	if err != nil {
		return report, tmpl.Values{}, "", err
	}
	for k, v := range choices {
		if k == "profile" {
			continue
		}
		newContent = setKey(newContent, k, v)
	}
	if d := Diff(report.Path, old, newContent); d != "" {
		report.State = Pending
		report.Diff = d
		report.newContent = newContent
	}
	return report, vals, gen, nil
}

func planClaude(dir, gen string, vals tmpl.Values) (FileReport, error) {
	report := FileReport{Path: "CLAUDE.md"}
	block, err := tmpl.Render("current", "CLAUDE.md.tmpl", vals)
	if err != nil {
		return report, err
	}
	path := filepath.Join(dir, "CLAUDE.md")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		report.State = Pending
		report.Created = true
		report.Diff = Diff(report.Path, "", block)
		report.newContent = block
		return report, nil
	}
	if err != nil {
		return report, err
	}
	old := string(raw)
	var newContent string
	if strings.Contains(old, markerBegin) {
		replaced, err := replaceMarkerBlock(old, block)
		if err != nil {
			// unbalanced markers: never force-droppable, no newContent.
			report.Unrecognized = []string{err.Error()}
			return report, nil
		}
		newContent = replaced
	} else {
		gen0, err := tmpl.Render("gen0", "CLAUDE.md.tmpl", vals)
		if err != nil {
			return report, err
		}
		if strings.TrimSpace(old) == strings.TrimSpace(gen0) {
			newContent = block // pristine legacy file: clean claim
		} else {
			newContent = block + "\n" + old // claim on top, preserve everything
		}
	}
	if d := Diff(report.Path, old, newContent); d != "" {
		report.State = Pending
		report.Diff = d
		report.newContent = newContent
	}
	return report, nil
}

func replaceMarkerBlock(old, block string) (string, error) {
	if strings.Count(old, markerBegin) != 1 || strings.Count(old, markerEnd) != 1 {
		return "", fmt.Errorf("CLAUDE.md spine markers unbalanced; fix by hand")
	}
	begin := strings.Index(old, markerBegin)
	end := strings.Index(old, markerEnd)
	if end < begin {
		return "", fmt.Errorf("CLAUDE.md spine markers out of order; fix by hand")
	}
	return old[:begin] + strings.TrimSuffix(block, "\n") + old[end+len(markerEnd):], nil
}

func planSimple(dir, gen, tmplName, relPath string, inGen0 bool, vals tmpl.Values) (FileReport, error) {
	report := FileReport{Path: relPath}
	newContent, err := tmpl.Render("current", tmplName, vals)
	if err != nil {
		return report, err
	}
	path := filepath.Join(dir, relPath)
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		report.State = Pending
		report.Created = true
		report.Diff = Diff(relPath, "", newContent)
		report.newContent = newContent
		return report, nil
	}
	if err != nil {
		return report, err
	}
	old := string(raw)
	expectedGen := "current"
	if gen == "gen0" && inGen0 {
		expectedGen = "gen0"
	}
	expectedOld, err := tmpl.Render(expectedGen, tmplName, vals)
	if err != nil {
		return report, err
	}
	report.Unrecognized = unrecognizedLines(old, expectedOld)
	if d := Diff(relPath, old, newContent); d != "" {
		report.State = Pending
		report.Diff = d
		report.newContent = newContent
	}
	return report, nil
}

// unrecognizedLines returns non-blank lines of got that expected does not
// contain anywhere (order-insensitive, trailing-space-insensitive).
func unrecognizedLines(got, expected string) []string {
	want := map[string]bool{}
	for _, l := range splitLines(expected) {
		want[strings.TrimRight(l, " ")] = true
	}
	var extra []string
	for _, l := range splitLines(got) {
		t := strings.TrimRight(l, " ")
		if t == "" || want[t] {
			continue
		}
		extra = append(extra, t)
	}
	return extra
}
