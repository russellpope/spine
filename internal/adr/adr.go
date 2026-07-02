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
	// HasFrontMatter is true only when the file has a --- ... --- block as
	// its first two "---" lines. Pre-spine, hand-rolled ADRs (e.g. hbmview's)
	// have no such block; Title/Status are empty for them, not invalid.
	HasFrontMatter bool
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
		e.Title, e.Status, e.HasFrontMatter = parseFrontMatter(string(raw))
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// frontMatterBounds returns the line indices of the first "---" ... "---"
// block: start is the opening fence, end is the closing fence. Returns
// -1, -1 if no such block exists (e.g. pre-spine, hand-rolled ADRs that
// predate spine's front-matter convention).
func frontMatterBounds(lines []string) (start, end int) {
	start, end = -1, -1
	for i, line := range lines {
		if line != "---" {
			continue
		}
		if start == -1 {
			start = i
			continue
		}
		end = i
		break
	}
	return start, end
}

// parseFrontMatter reads title/status only from the front-matter block: the
// lines strictly between the first line that is exactly "---" and the next
// line that is exactly "---". Within that block the first matching
// "title: "/"status: " line wins (consistent with flippedContent). Anything
// outside the block — including a forged "---" fence or a "title: "/
// "status: " line in the body — is ignored, so it can't corrupt List output.
// hasFrontMatter is true only when the block exists at all; pre-spine ADRs
// with no such block get empty title/status and hasFrontMatter=false, which
// callers must treat as "not applicable" rather than "invalid".
func parseFrontMatter(content string) (title, status string, hasFrontMatter bool) {
	lines := strings.Split(content, "\n")
	start, end := frontMatterBounds(lines)
	if start == -1 || end == -1 {
		return "", "", false
	}
	for _, line := range lines[start+1 : end] {
		if title == "" {
			if t, ok := strings.CutPrefix(line, "title: "); ok {
				title = strings.TrimSpace(t)
			}
		}
		if status == "" {
			if s, ok := strings.CutPrefix(line, "status: "); ok {
				status = strings.TrimSpace(s)
			}
		}
	}
	return title, status, true
}

// New writes the next-numbered ADR; supersedes > 0 also flips that ADR's
// status line. Returns the new file's path.
func New(dir, title string, supersedes int) (string, error) {
	if strings.ContainsAny(title, "\n\r") {
		return "", fmt.Errorf("title %q contains a newline, which would inject fake front matter", title)
	}
	slug := slugify(title)
	if slug == "" {
		return "", fmt.Errorf("title %q produces an empty slug — use at least one ASCII letter or digit", title)
	}

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
				break
			}
		}
		if target == nil {
			return "", fmt.Errorf("supersedes target %04d not found", supersedes)
		}
	}

	// Compute the supersede flip before writing anything: if the target
	// can't be flipped (e.g. no status line), New must fail clean rather
	// than leave a new ADR claiming supersedes on an unflipped target.
	var flipped []byte
	if target != nil {
		flipped, err = flippedContent(target.Path, next)
		if err != nil {
			return "", err
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
	path := filepath.Join(dir, "docs", "adr", id+"-"+slug+".md")
	if err := fsutil.WriteFileAtomic(path, []byte(content)); err != nil {
		return "", err
	}
	if target != nil {
		// Residual window: if this second write fails physically, the new
		// ADR (already on disk) claims supersedes on a not-yet-flipped
		// target. Acceptable per review — validation now happens up front.
		if err := fsutil.WriteFileAtomic(target.Path, flipped); err != nil {
			return "", err
		}
	}
	return path, nil
}

// slugify lowercases s and keeps only ASCII letters/digits, collapsing every
// other rune into a single '-' separator (trimmed at both ends). Non-ASCII
// runes (e.g. CJK, accented letters) are not transliterated — they collapse
// to separators just like punctuation, so this is lossy by design. A title
// made entirely of non-ASCII or punctuation runes yields an empty string;
// callers must treat that as invalid rather than writing "NNNN-.md".
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

// flippedContent reads the ADR at path and returns its content with the
// front-matter status line rewritten to "Superseded by NNNN". It performs no
// writes, so New can validate a supersede target (and get its would-be new
// content) before writing anything at all.
//
// The search for "status: " is scoped to the same first "---" ... "---"
// block parseFrontMatter uses — never the body. Pre-spine ADRs with no such
// block (e.g. hbmview's hand-rolled files) cannot be flipped automatically;
// scanning the whole file for the first "status: " line risked rewriting an
// unrelated body line (e.g. inside a code sample), so that case is now a
// hard error instead of a silent mutation.
func flippedContent(path string, by int) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(raw), "\n")
	start, end := frontMatterBounds(lines)
	if start != -1 && end != -1 {
		for i := start + 1; i < end; i++ {
			if strings.HasPrefix(lines[i], "status: ") {
				lines[i] = fmt.Sprintf("status: Superseded by %04d", by)
				return []byte(strings.Join(lines, "\n")), nil
			}
		}
	}
	return nil, fmt.Errorf("target %s has no front-matter status line (pre-spine ADR) — supersede it manually", path)
}
