// Package eval manages the docs/evals/ convention: spine owns the structure
// (dirs, eval.md, run records); the process driving the eval (/model-eval)
// owns every value. Stage and score are opaque strings here — no code in
// this package may branch on their contents (ADR 0007).
package eval

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/russellpope/spine/internal/fsutil"
	"github.com/russellpope/spine/internal/meta"
	"github.com/russellpope/spine/templates"
)

// Run is one parsed run record.
type Run struct {
	Name  string
	Stage string
	Score string
	Path  string
}

// Eval is one eval dir with its runs.
type Eval struct {
	Name string
	Path string
	Runs []Run
}

// Problem is a structural defect List found (doctor surfaces these as D7).
type Problem struct {
	Path    string
	Message string
}

var evalKeys = []string{"title", "created", "prompt", "rubric"}
var runKeys = []string{"name", "created", "model", "stage", "score"}

// New scaffolds docs/evals/<today>-<slug>/{eval.md,runs/}, plus the
// convention README on first use. It never overwrites.
func New(dir, title string) (string, error) {
	slug := meta.Slugify(title)
	if slug == "" {
		return "", fmt.Errorf("title %q produces an empty slug — use at least one ASCII letter or digit", title)
	}
	if strings.ContainsAny(title, "\n\r") {
		return "", fmt.Errorf("title %q contains a newline, which would inject fake front matter", title)
	}
	today := time.Now().Format("2006-01-02")
	root := filepath.Join(dir, "docs", "evals")
	evalDir := filepath.Join(root, today+"-"+slug)
	if _, err := os.Stat(evalDir); err == nil {
		return "", fmt.Errorf("%s already exists", evalDir)
	} else if !os.IsNotExist(err) {
		// Fast-path existence check: if evalDir exists, return the user-facing
		// "already exists" error up front. Non-NotExist Stat errors (EACCES,
		// ELOOP, ...) must still propagate to avoid silent failures.
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(evalDir, "runs"), 0o755); err != nil {
		return "", err
	}
	readme := filepath.Join(root, "README.md")
	rawReadme, rerr := templates.FS.ReadFile("current/evals-README.md")
	if rerr != nil {
		return "", rerr
	}
	if werr := fsutil.WriteFileExclusive(readme, rawReadme); werr != nil && !errors.Is(werr, fs.ErrExist) {
		return "", werr
	}
	raw, err := templates.FS.ReadFile("current/eval.tmpl.md")
	if err != nil {
		return "", err
	}
	content := strings.NewReplacer(
		"{{EVAL_TITLE_YAML}}", strconv.Quote(title),
		"{{EVAL_TITLE}}", title,
		"{{EVAL_DATE}}", today,
	).Replace(string(raw))
	if err := fsutil.WriteFileExclusive(filepath.Join(evalDir, "eval.md"), []byte(content)); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("%s already exists", evalDir)
		}
		return "", err
	}
	return evalDir, nil
}

// resolveEval matches ref against docs/evals/ children: exact dir name, or
// the name with its YYYY-MM-DD- prefix stripped. Ambiguity is an error.
func resolveEval(dir, ref string) (string, error) {
	root := filepath.Join(dir, "docs", "evals")
	des, err := os.ReadDir(root)
	if err != nil {
		return "", fmt.Errorf("no docs/evals/ in %s (run spine eval new first): %w", dir, err)
	}
	var matches []string
	for _, de := range des {
		if !de.IsDir() {
			continue
		}
		name := de.Name()
		stripped := name
		if len(name) > 11 && name[4] == '-' && name[7] == '-' && name[10] == '-' {
			stripped = name[11:]
		}
		if name == ref || stripped == ref {
			matches = append(matches, name)
		}
	}
	switch len(matches) {
	case 1:
		return filepath.Join(root, matches[0]), nil
	case 0:
		return "", fmt.Errorf("no eval matches %q under %s", ref, root)
	default:
		return "", fmt.Errorf("eval ref %q is ambiguous: %s", ref, strings.Join(matches, ", "))
	}
}

// AddRun scaffolds runs/<name>.md inside the resolved eval. Never overwrites.
func AddRun(dir, evalRef, name string) (string, error) {
	if name == "" || strings.ContainsAny(name, "/\\ \t\n\r") || strings.HasPrefix(name, ".") {
		return "", fmt.Errorf("run name %q must be a plain filename fragment (no separators, whitespace, or leading dot)", name)
	}
	evalDir, err := resolveEval(dir, evalRef)
	if err != nil {
		return "", err
	}
	runsDir := filepath.Join(evalDir, "runs")
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(runsDir, name+".md")
	raw, err := templates.FS.ReadFile("current/run.tmpl.md")
	if err != nil {
		return "", err
	}
	content := strings.NewReplacer(
		"{{RUN_NAME}}", name,
		"{{RUN_DATE}}", time.Now().Format("2006-01-02"),
	).Replace(string(raw))
	if err := fsutil.WriteFileExclusive(path, []byte(content)); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("%s already exists", path)
		}
		return "", err
	}
	return path, nil
}

// List parses docs/evals/. A missing tree is empty, not an error. Structural
// defects come back as Problems; values are returned verbatim.
func List(dir string) ([]Eval, []Problem, error) {
	root := filepath.Join(dir, "docs", "evals")
	des, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var evals []Eval
	var problems []Problem
	for _, de := range des {
		if !de.IsDir() {
			continue
		}
		e := Eval{Name: de.Name(), Path: filepath.Join(root, de.Name())}
		probs, err := checkDoc(filepath.Join(e.Path, "eval.md"), evalKeys)
		if err != nil {
			return nil, nil, err
		}
		problems = append(problems, probs...)
		runsDir := filepath.Join(e.Path, "runs")
		rdes, err := os.ReadDir(runsDir)
		if err != nil && !os.IsNotExist(err) {
			return nil, nil, err
		}
		if os.IsNotExist(err) {
			problems = append(problems, Problem{Path: runsDir, Message: "missing runs/ directory"})
		}
		for _, rde := range rdes {
			if rde.IsDir() || !strings.HasSuffix(rde.Name(), ".md") {
				continue
			}
			rpath := filepath.Join(runsDir, rde.Name())
			raw, err := os.ReadFile(rpath)
			if err != nil {
				return nil, nil, err
			}
			kv, has := meta.Parse(string(raw))
			if probs := checkKeys(rpath, kv, has, runKeys); len(probs) > 0 {
				problems = append(problems, probs...)
				continue
			}
			e.Runs = append(e.Runs, Run{Name: kv["name"], Stage: kv["stage"], Score: kv["score"], Path: rpath})
		}
		sort.Slice(e.Runs, func(i, j int) bool { return e.Runs[i].Name < e.Runs[j].Name })
		evals = append(evals, e)
	}
	sort.Slice(evals, func(i, j int) bool { return evals[i].Name < evals[j].Name })
	return evals, problems, nil
}

func checkDoc(path string, keys []string) ([]Problem, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []Problem{{Path: path, Message: "missing eval.md"}}, nil
	}
	if err != nil {
		return nil, err
	}
	kv, has := meta.Parse(string(raw))
	return checkKeys(path, kv, has, keys), nil
}

func checkKeys(path string, kv map[string]string, has bool, keys []string) []Problem {
	if !has {
		return []Problem{{Path: path, Message: "no front matter block"}}
	}
	var missing []string
	for _, k := range keys {
		if _, ok := kv[k]; !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return []Problem{{Path: path, Message: "front matter missing key(s): " + strings.Join(missing, ", ")}}
	}
	return nil
}
