// Package handoff manages docs/handoffs/: date-named session handoff notes.
// spine owns the naming and skeleton; the /handoff skill owns the content.
package handoff

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/russellpope/spine/internal/fsutil"
	"github.com/russellpope/spine/internal/meta"
	"github.com/russellpope/spine/templates"
)

// Entry is one handoff file. Title comes from front matter when present
// (spine-scaffolded files); legacy handoffs fall back to the filename topic.
type Entry struct {
	Date  time.Time
	Topic string
	Title string
	Path  string
}

var nameRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-(.+)\.md$`)

// ParseName validates a handoff filename: date-prefixed, .md, real date.
func ParseName(filename string) (date time.Time, topic string, ok bool) {
	m := nameRe.FindStringSubmatch(filename)
	if m == nil {
		return time.Time{}, "", false
	}
	d, err := time.Parse("2006-01-02", m[1])
	if err != nil {
		return time.Time{}, "", false
	}
	return d, m[2], true
}

// New scaffolds docs/handoffs/<today>-<slug>.md. It never overwrites.
func New(dir, topic string) (string, error) {
	slug := meta.Slugify(topic)
	if slug == "" {
		return "", fmt.Errorf("topic %q produces an empty slug — use at least one ASCII letter or digit", topic)
	}
	if strings.ContainsAny(topic, "\n\r") {
		return "", fmt.Errorf("topic %q contains a newline, which would inject fake front matter", topic)
	}
	today := time.Now().Format("2006-01-02")
	hdir := filepath.Join(dir, "docs", "handoffs")
	if err := os.MkdirAll(hdir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(hdir, today+"-"+slug+".md")
	raw, err := templates.FS.ReadFile("current/handoff.tmpl.md")
	if err != nil {
		return "", err
	}
	content := strings.NewReplacer(
		"{{HANDOFF_TITLE_YAML}}", strconv.Quote(topic),
		"{{HANDOFF_TITLE}}", topic,
		"{{HANDOFF_DATE}}", today,
	).Replace(string(raw))
	if err := fsutil.WriteFileExclusive(path, []byte(content)); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("%s already exists — pick a more specific topic", path)
		}
		return "", err
	}
	return path, nil
}

// List returns entries newest-first (date desc, filename desc as tiebreak).
// A missing docs/handoffs dir lists as empty, not an error.
func List(dir string) ([]Entry, error) {
	hdir := filepath.Join(dir, "docs", "handoffs")
	des, err := os.ReadDir(hdir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, de := range des {
		if de.IsDir() {
			continue
		}
		d, topic, ok := ParseName(de.Name())
		if !ok {
			continue
		}
		e := Entry{Date: d, Topic: topic, Title: topic, Path: filepath.Join(hdir, de.Name())}
		raw, err := os.ReadFile(e.Path)
		if err != nil {
			return nil, err
		}
		if kv, has := meta.Parse(string(raw)); has && kv["title"] != "" {
			// Gen-4 templates YAML-quote the title (strconv.Quote in New).
			// UnquoteScalar unquotes for display; unquoted pre-gen-4 titles
			// pass through verbatim.
			e.Title = meta.UnquoteScalar(kv["title"])
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Date.Equal(out[j].Date) {
			return out[i].Date.After(out[j].Date)
		}
		return out[i].Path > out[j].Path
	})
	return out, nil
}

// Latest returns the newest entry; ok is false when there are none.
func Latest(dir string) (Entry, bool, error) {
	entries, err := List(dir)
	if err != nil || len(entries) == 0 {
		return Entry{}, false, err
	}
	return entries[0], true, nil
}

// FleetEntry is one repo's latest handoff in a --fleet scan.
type FleetEntry struct {
	Repo string
	Entry
}

// Fleet scans every immediate child dir of parent for docs/handoffs and
// returns each repo's latest handoff, newest first (repo name as tiebreak).
// Children without handoffs are silently skipped; a missing parent errors.
func Fleet(parent string) ([]FleetEntry, error) {
	des, err := os.ReadDir(parent)
	if err != nil {
		return nil, err
	}
	var out []FleetEntry
	for _, de := range des {
		if !de.IsDir() || strings.HasPrefix(de.Name(), ".") {
			continue
		}
		e, ok, err := Latest(filepath.Join(parent, de.Name()))
		if err != nil || !ok {
			continue
		}
		out = append(out, FleetEntry{Repo: de.Name(), Entry: e})
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Date.Equal(out[j].Date) {
			return out[i].Date.After(out[j].Date)
		}
		return out[i].Repo < out[j].Repo
	})
	return out, nil
}
