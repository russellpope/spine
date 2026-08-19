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

	"github.com/russellpope/spine/internal/cursor"
	"github.com/russellpope/spine/internal/fsutil"
	"github.com/russellpope/spine/internal/meta"
	"github.com/russellpope/spine/templates"
)

// Entry is one handoff file. Title comes from front matter when present
// (spine-scaffolded files); legacy handoffs fall back to the filename topic.
type Entry struct {
	Date    time.Time
	Topic   string
	Title   string
	Path    string
	Ordinal uint64 // persisted handoff_ordinal; zero is the legacy fallback
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
	path, _, err := NewWithCursor(dir, topic)
	return path, err
}

// NewWithCursor scaffolds a handoff and captures the current parsed cursor
// block when one exists. The bool reports whether that snapshot was embedded.
func NewWithCursor(dir, topic string) (string, bool, error) {
	slug := meta.Slugify(topic)
	if slug == "" {
		return "", false, fmt.Errorf("topic %q produces an empty slug — use at least one ASCII letter or digit", topic)
	}
	if strings.ContainsAny(topic, "\n\r") {
		return "", false, fmt.Errorf("topic %q contains a newline, which would inject fake front matter", topic)
	}
	cursorResult, err := cursor.Load(dir)
	if err != nil {
		return "", false, err
	}
	if cursorResult.HasCursor && len(cursorResult.Findings) > 0 {
		return "", false, fmt.Errorf("cursor block is malformed: %s", strings.Join(cursorResult.Findings, "; "))
	}
	today := time.Now().Format("2006-01-02")
	hdir := filepath.Join(dir, "docs", "handoffs")
	if err := os.MkdirAll(hdir, 0o755); err != nil {
		return "", false, err
	}
	path := filepath.Join(hdir, today+"-"+slug+".md")
	raw, err := templates.FS.ReadFile("current/handoff.tmpl.md")
	if err != nil {
		return "", false, err
	}
	ordinal, releaseOrdinal, err := reserveNextOrdinal(dir, hdir)
	if err != nil {
		return "", false, err
	}
	defer releaseOrdinal()
	content := strings.NewReplacer(
		"{{HANDOFF_TITLE_YAML}}", strconv.Quote(topic),
		"{{HANDOFF_TITLE}}", topic,
		"{{HANDOFF_DATE}}", today,
		"{{HANDOFF_ORDINAL}}", strconv.FormatUint(ordinal, 10),
	).Replace(string(raw))
	if cursorResult.HasCursor {
		content += "\n" + cursorResult.Cursor.Block() + "\n"
	}
	// The newest checkpoint, when the checkpoint working home is non-empty:
	// forward intent survives the session end. Empty working home => the
	// handoff is byte-identical to a pre-embed one.
	embed, err := checkpointEmbed(dir)
	if err != nil {
		return "", false, err
	}
	content += embed
	if err := fsutil.WriteFileExclusive(path, []byte(content)); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", false, fmt.Errorf("%s already exists — pick a more specific topic", path)
		}
		return "", false, err
	}
	return path, cursorResult.HasCursor, nil
}

const ordinalReservationDir = ".spine-handoff-ordinal-reservations"

// reserveNextOrdinal reserves a repository-wide handoff ordinal with the
// shared exclusive-marker primitive. Legacy handoffs without
// handoff_ordinal have ordinal zero, so the first handoff created with this
// mechanism receives one.
func reserveNextOrdinal(dir, hdir string) (uint64, func(), error) {
	return fsutil.ReserveOrdinal(
		filepath.Join(hdir, ordinalReservationDir),
		func() (uint64, error) {
			entries, err := List(dir)
			if err != nil {
				return 0, err
			}
			var max uint64
			for _, e := range entries {
				if e.Ordinal > max {
					max = e.Ordinal
				}
			}
			return max, nil
		},
		func(ordinal uint64) (bool, error) {
			entries, err := List(dir)
			if err != nil {
				return false, err
			}
			for _, entry := range entries {
				if entry.Ordinal == ordinal {
					return true, nil
				}
			}
			return false, nil
		},
	)
}

// List returns entries newest-first by date, then persisted creation ordinal,
// then filename. Missing or invalid legacy ordinals are zero, preserving a
// deterministic filename fallback for existing handoffs and duplicate values.
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
		if kv, has := meta.Parse(string(raw)); has {
			if kv["title"] != "" {
				// Gen-4 templates YAML-quote the title (strconv.Quote in New).
				// UnquoteScalar unquotes for display; unquoted pre-gen-4 titles
				// pass through verbatim.
				e.Title = meta.UnquoteScalar(kv["title"])
			}
			e.Ordinal = parseOrdinal(kv["handoff_ordinal"])
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Date.Equal(out[j].Date) {
			return out[i].Date.After(out[j].Date)
		}
		if out[i].Ordinal != out[j].Ordinal {
			return out[i].Ordinal > out[j].Ordinal
		}
		return out[i].Path > out[j].Path
	})
	return out, nil
}

func parseOrdinal(raw string) uint64 {
	ordinal, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || ordinal == 0 {
		return 0
	}
	return ordinal
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
