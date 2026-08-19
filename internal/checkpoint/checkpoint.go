// Package checkpoint manages the checkpoint working home:
// .superpowers/sdd/checkpoints/NNN-<slug>.md, the ordinal-numbered,
// uncommitted location where a session's checkpoints accumulate.
//
// A checkpoint is the document a running session distils itself into just
// before a context reload. It has two structurally distinct regions:
//
//	---
//	ordinal: 1
//	created: 2026-08-18T17:04:05Z
//	effort: high
//	narrative: present
//	---
//	<!-- spine:checkpoint:model -->
//	## Task
//	...
//	<!-- /spine:checkpoint:model -->
//	<!-- spine:checkpoint:facts -->
//	touched:
//	- internal/checkpoint/checkpoint.go
//	gate: pass
//	sha: <git rev-parse HEAD>
//	effort_recommended: high
//	written: 2026-08-18T17:04:05Z
//	<!-- /spine:checkpoint:facts -->
//
// The model region holds the model's own prior claims — never evidence. The
// facts region is harness-written evidence and obeys the cursor's
// sole-writer and canonical-form rules: only New writes it, and its bytes
// are a pure function of its values (Facts.Block).
package checkpoint

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/russellpope/spine/internal/meta"
	"github.com/russellpope/spine/templates"
)

// Region markers. Both regions carry a closing marker, mirroring the cursor
// block's open/close tag convention.
const (
	ModelOpenTag  = "<!-- spine:checkpoint:model -->"
	ModelCloseTag = "<!-- /spine:checkpoint:model -->"
	FactsOpenTag  = "<!-- spine:checkpoint:facts -->"
	FactsCloseTag = "<!-- /spine:checkpoint:facts -->"
)

// NarrativePresent and NarrativeMissing are the frontmatter narrative values.
const (
	NarrativePresent = "present"
	NarrativeMissing = "missing"
)

// Home returns the checkpoint working home for the repo at dir.
func Home(dir string) string {
	return filepath.Join(dir, ".superpowers", "sdd", "checkpoints")
}

// Preamble returns the embedded reload preamble: static, spine-shipped text
// that precedes the checkpoint in the reload prompt. Byte-stable across a
// template generation so the reload prefix is cacheable.
func Preamble() (string, error) {
	raw, err := templates.FS.ReadFile("current/checkpoint-preamble.md")
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// Entry is one checkpoint file in the working home.
type Entry struct {
	Ordinal uint64
	Slug    string
	Created string // frontmatter created, RFC3339 UTC; empty when unreadable
	Path    string
}

var nameRe = regexp.MustCompile(`^(\d{3,})-(.+)\.md$`)

// List returns every checkpoint in the working home, in ascending ordinal
// order. A missing working home lists as empty, not an error.
func List(dir string) ([]Entry, error) {
	home := Home(dir)
	des, err := os.ReadDir(home)
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
		m := nameRe.FindStringSubmatch(de.Name())
		if m == nil {
			continue
		}
		ordinal, err := strconv.ParseUint(m[1], 10, 64)
		if err != nil || ordinal == 0 {
			continue
		}
		e := Entry{Ordinal: ordinal, Slug: m[2], Path: filepath.Join(home, de.Name())}
		raw, err := os.ReadFile(e.Path)
		if err != nil {
			return nil, err
		}
		if kv, has := meta.Parse(string(raw)); has {
			e.Created = kv["created"]
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ordinal < out[j].Ordinal })
	return out, nil
}

// Latest returns the newest checkpoint (highest ordinal); ok is false when
// the working home is empty or absent.
func Latest(dir string) (Entry, bool, error) {
	entries, err := List(dir)
	if err != nil || len(entries) == 0 {
		return Entry{}, false, err
	}
	return entries[len(entries)-1], true, nil
}

// Document is a checkpoint split into its three parts. Frontmatter is the
// body between the `---` fences; Model and Facts are the bodies between
// their region markers, both with the surrounding blank lines trimmed.
// Callers that need the facts values parse Facts with ParseFacts.
type Document struct {
	Frontmatter string
	Model       string
	Facts       string
}

// Split parses a checkpoint file's bytes into its three parts. A part that
// is absent comes back empty; Split reports no error for a hand-mangled
// document — reporting drift is the doctor advisory's job, not the
// parser's.
func Split(raw string) Document {
	var d Document
	if rest, ok := strings.CutPrefix(raw, "---\n"); ok {
		if fm, after, ok := strings.Cut(rest, "\n---\n"); ok {
			d.Frontmatter = fm
			raw = after
		}
	}
	d.Model = between(raw, ModelOpenTag, ModelCloseTag)
	// The facts region is located after the model region closes, so a
	// marker-like string inside a hand-edited model region cannot be
	// mistaken for harness evidence.
	if _, afterModel, ok := strings.Cut(raw, ModelCloseTag); ok {
		raw = afterModel
	}
	d.Facts = between(raw, FactsOpenTag, FactsCloseTag)
	return d
}

// between returns the text between the first open marker and the first
// close marker after it, trimmed of the newlines that hug the markers.
func between(raw, open, close string) string {
	_, after, ok := strings.Cut(raw, open)
	if !ok {
		return ""
	}
	body, _, ok := strings.Cut(after, close)
	if !ok {
		return ""
	}
	return strings.Trim(body, "\n")
}
