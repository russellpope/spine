package checkpoint

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/russellpope/spine/internal/fsutil"
	"github.com/russellpope/spine/internal/meta"
)

// Options are the inputs to New: what the caller knows (the narrative file
// and the files touched, gate status and recommended effort) plus the slug
// choice. Ordinal, git sha and timestamps are computed here — each fact
// comes from the party that actually knows it.
type Options struct {
	Dir       string   // repo root
	From      string   // path to the narrative file; ignored when FactsOnly
	Touched   []string // caller order preserved
	Gate      string   // pass | fail | none
	Effort    string   // recommended per-leg effort
	Slug      string   // optional; derived from the narrative's ## Task when empty
	FactsOnly bool     // write narrative: missing with an empty model region
}

// sections are the model region's required headed sections, in order.
var sections = []string{"## Task", "## Conclusions", "## Next moves"}

// maxSlug bounds a derived slug so a long task line cannot produce an
// unwieldy filename.
const maxSlug = 60

// New writes a checkpoint into the working home and returns its path. The
// ordinal is reserved exclusively (like handoffs) so concurrent writers
// never collide and an ordinal is never reused.
func New(opt Options) (string, error) {
	if !ValidGate(opt.Gate) {
		return "", fmt.Errorf("--gate %q must be pass, fail, or none", opt.Gate)
	}
	if strings.TrimSpace(opt.Effort) == "" {
		return "", errors.New("--effort is required")
	}
	narrative := ""
	if !opt.FactsOnly {
		if opt.From == "" {
			return "", errors.New("--from <narrative.md> is required (or pass --facts-only)")
		}
		raw, err := os.ReadFile(opt.From)
		if err != nil {
			return "", err
		}
		narrative = strings.Trim(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
		if err := rejectMarkers(narrative); err != nil {
			return "", err
		}
		if err := validateModel(narrative); err != nil {
			return "", err
		}
	}
	slug, err := resolveSlug(opt, narrative)
	if err != nil {
		return "", err
	}
	sha, err := headSHA(opt.Dir)
	if err != nil {
		return "", err
	}
	home := Home(opt.Dir)
	if err := os.MkdirAll(home, 0o755); err != nil {
		return "", err
	}
	ordinal, release, err := reserveNextOrdinal(opt.Dir, home)
	if err != nil {
		return "", err
	}
	defer release()

	stamp := time.Now().UTC().Format(time.RFC3339)
	facts := Facts{
		Touched:           opt.Touched,
		Gate:              opt.Gate,
		SHA:               sha,
		EffortRecommended: opt.Effort,
		Written:           stamp,
	}
	narrativeState := NarrativePresent
	if opt.FactsOnly {
		narrativeState = NarrativeMissing
	}
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("ordinal: " + strconv.FormatUint(ordinal, 10) + "\n")
	b.WriteString("created: " + stamp + "\n")
	b.WriteString("effort: " + strings.TrimSpace(opt.Effort) + "\n")
	b.WriteString("narrative: " + narrativeState + "\n")
	b.WriteString("---\n")
	b.WriteString(ModelOpenTag + "\n")
	if narrative != "" {
		b.WriteString(narrative + "\n")
	}
	b.WriteString(ModelCloseTag + "\n")
	b.WriteString(facts.Block() + "\n")

	path := filepath.Join(home, fmt.Sprintf("%03d-%s.md", ordinal, slug))
	if err := fsutil.WriteFileExclusive(path, []byte(b.String())); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("%s already exists", path)
		}
		return "", err
	}
	return path, nil
}

// rejectMarkers keeps the two regions structurally separate at the writer:
// a narrative carrying any region marker could otherwise close the model
// region early or smuggle a fake facts region, letting model-authored text
// masquerade as harness evidence.
func rejectMarkers(narrative string) error {
	for _, marker := range []string{ModelOpenTag, ModelCloseTag, FactsOpenTag, FactsCloseTag} {
		if strings.Contains(narrative, marker) {
			return fmt.Errorf("narrative contains the region marker %s — the model region may never carry checkpoint markers", marker)
		}
	}
	return nil
}

// validateModel enforces the strict model-region contract: each of the three
// headed sections must be present with non-whitespace content before the
// next `## ` heading or EOF. A hollow checkpoint is never mistaken for a
// good one, and the message names the offending section.
func validateModel(narrative string) error {
	lines := strings.Split(narrative, "\n")
	for _, section := range sections {
		start := -1
		for i, line := range lines {
			if strings.TrimRight(line, " \t") == section {
				start = i
				break
			}
		}
		if start < 0 {
			return fmt.Errorf("model region is missing the %q section", section)
		}
		body := ""
		for _, line := range lines[start+1:] {
			if strings.HasPrefix(line, "## ") {
				break
			}
			body += line
		}
		if strings.TrimSpace(body) == "" {
			return fmt.Errorf("model region section %q is empty", section)
		}
	}
	return nil
}

// resolveSlug takes --slug when given, else derives one from the first
// non-empty line under the narrative's `## Task` heading. A facts-only
// checkpoint with no --slug is named facts-only.
func resolveSlug(opt Options, narrative string) (string, error) {
	if opt.Slug != "" {
		slug := meta.Slugify(opt.Slug)
		if slug == "" {
			return "", fmt.Errorf("--slug %q produces an empty slug — use at least one ASCII letter or digit", opt.Slug)
		}
		return bound(slug), nil
	}
	if opt.FactsOnly {
		return "facts-only", nil
	}
	slug := meta.Slugify(taskLine(narrative))
	if slug == "" {
		return "", errors.New(`could not derive a slug from the "## Task" section — pass --slug`)
	}
	return bound(slug), nil
}

// taskLine returns the first non-empty line under the `## Task` heading.
func taskLine(narrative string) string {
	lines := strings.Split(narrative, "\n")
	for i, line := range lines {
		if strings.TrimRight(line, " \t") != sections[0] {
			continue
		}
		for _, candidate := range lines[i+1:] {
			if strings.TrimSpace(candidate) != "" {
				return candidate
			}
		}
	}
	return ""
}

// bound trims a slug to maxSlug runes without leaving a trailing separator.
func bound(slug string) string {
	if len(slug) <= maxSlug {
		return slug
	}
	return strings.TrimRight(slug[:maxSlug], "-")
}

// headSHA is the repo's current commit, computed by spine rather than
// supplied by the caller.
func headSHA(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD in %s: %w", dir, err)
	}
	return strings.TrimSpace(string(out)), nil
}

const ordinalReservationDir = ".spine-checkpoint-ordinal-reservations"

// reserveNextOrdinal reserves a working-home-wide checkpoint ordinal with
// the shared exclusive-marker primitive (same technique as handoffs): a
// crash-left reservation consumes its ordinal so it is never reused.
func reserveNextOrdinal(dir, home string) (uint64, func(), error) {
	return fsutil.ReserveOrdinal(
		filepath.Join(home, ordinalReservationDir),
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
			for _, e := range entries {
				if e.Ordinal == ordinal {
					return true, nil
				}
			}
			return false, nil
		},
	)
}
