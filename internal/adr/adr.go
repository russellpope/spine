// Package adr manages the docs/adr/ ledger: immutable decisions, supersede
// status flips being the single permitted mutation.
package adr

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/russellpope/spine/internal/fsutil"
	"github.com/russellpope/spine/templates"
)

// Entry is one parsed ADR file.
type Entry struct {
	ID     int
	Title  string
	Status string
	Path   string
}

var fileRe = regexp.MustCompile(`^(\d{4})-.+\.md$`)

// List parses docs/adr/ under dir, sorted by ID. Files not matching
// NNNN-slug.md (e.g. README.md) are ignored.
func List(dir string) ([]Entry, error) {
	adrDir := filepath.Join(dir, "docs", "adr")
	des, err := os.ReadDir(adrDir)
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, de := range des {
		m := fileRe.FindStringSubmatch(de.Name())
		if m == nil {
			continue
		}
		id, _ := strconv.Atoi(m[1])
		e := Entry{ID: id, Path: filepath.Join(adrDir, de.Name())}
		raw, err := os.ReadFile(e.Path)
		if err != nil {
			return nil, err
		}
		e.Title, e.Status = parseFrontMatter(string(raw))
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func parseFrontMatter(content string) (title, status string) {
	for _, line := range strings.Split(content, "\n") {
		if t, ok := strings.CutPrefix(line, "title: "); ok {
			title = strings.TrimSpace(t)
		}
		if s, ok := strings.CutPrefix(line, "status: "); ok {
			status = strings.TrimSpace(s)
		}
	}
	return title, status
}

// New writes the next-numbered ADR; supersedes > 0 also flips that ADR's
// status line. Returns the new file's path.
func New(dir, title string, supersedes int) (string, error) {
	entries, err := List(dir)
	if err != nil {
		return "", err
	}
	next := 1
	for _, e := range entries {
		if e.ID >= next {
			next = e.ID + 1
		}
	}
	var target *Entry
	if supersedes > 0 {
		for i := range entries {
			if entries[i].ID == supersedes {
				target = &entries[i]
			}
		}
		if target == nil {
			return "", fmt.Errorf("supersedes target %04d not found", supersedes)
		}
	}
	raw, err := templates.FS.ReadFile("current/adr.tmpl.md")
	if err != nil {
		return "", err
	}
	sup := ""
	if supersedes > 0 {
		sup = fmt.Sprintf("\nsupersedes: %04d", supersedes)
	}
	id := fmt.Sprintf("%04d", next)
	content := strings.NewReplacer(
		"{{ADR_ID}}", id,
		"{{ADR_TITLE}}", title,
		"{{ADR_DATE}}", time.Now().Format("2006-01-02"),
		"{{ADR_SUPERSEDES}}", sup,
	).Replace(string(raw))
	path := filepath.Join(dir, "docs", "adr", id+"-"+slugify(title)+".md")
	if err := fsutil.WriteFileAtomic(path, []byte(content)); err != nil {
		return "", err
	}
	if target != nil {
		if err := flipStatus(target.Path, next); err != nil {
			return "", err
		}
	}
	return path, nil
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var b []rune
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b = append(b, r)
		default:
			if len(b) > 0 && b[len(b)-1] != '-' {
				b = append(b, '-')
			}
		}
	}
	return strings.Trim(string(b), "-")
}

func flipStatus(path string, by int) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(raw), "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, "status: ") {
			lines[i] = fmt.Sprintf("status: Superseded by %04d", by)
			return fsutil.WriteFileAtomic(path, []byte(strings.Join(lines, "\n")))
		}
	}
	return fmt.Errorf("no status line in %s", path)
}
