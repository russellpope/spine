// Package tmpl renders the embedded workflow templates.
package tmpl

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/russellpope/spine/templates"
)

// Values fills the {{KEY}} placeholders in a template.
type Values struct {
	Project   string
	Profile   string
	Reviewers string
	Harness   string
	Version   int
}

type profileDefaults struct{ reviewers, harness string }

var profiles = map[string]profileDefaults{
	"go-service":   {"go-reviewer, security-review", "rest"},
	"py-tool":      {"python-reviewer, security-review", "cli"},
	"rust":         {"rust-reviewer, security-review", "cli"},
	"library-cli":  {"go-reviewer, python-reviewer", "cli"},
	"presentation": {"", "none"},
	"ui":           {"typescript-reviewer", "framebuffer"},
	"swift":        {"swift-reviewer, security-review", "framebuffer"},
	"infra":        {"security-review", "none"},
	"knowledge":    {"", "none"},
}

// Profiles lists the known profile names, sorted.
func Profiles() []string {
	out := make([]string, 0, len(profiles))
	for p := range profiles {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// Defaults returns the reviewers/harness pair the profile map assigns.
func Defaults(profile string) (reviewers, harness string, err error) {
	d, ok := profiles[profile]
	if !ok {
		return "", "", fmt.Errorf("unknown profile %q (known: %s)", profile, strings.Join(Profiles(), ", "))
	}
	return d.reviewers, d.harness, nil
}

// Version returns the compiled template generation from templates/VERSION.
func Version() int {
	raw, err := templates.FS.ReadFile("VERSION")
	if err != nil {
		// templates/VERSION is go:embed'd at compile time (see templates/embed.go);
		// its absence is a build-time invariant violation, unreachable at runtime.
		panic("templates/VERSION missing from embed: " + err.Error())
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || n < 1 {
		// Same compile-time invariant: the embedded VERSION file's contents
		// are controlled by this repo, not runtime input — unreachable at runtime.
		panic("templates/VERSION must be a positive integer")
	}
	return n
}

// ProfileDirs is the directory set init/adopt create for a profile.
// knowledge repos center on decisions + handoffs; specs/issues are opt-in.
func ProfileDirs(profile string) []string {
	if profile == "knowledge" {
		return []string{"docs/adr", "docs/handoffs"}
	}
	return []string{"docs/specs", "docs/adr", "docs/issues", "docs/handoffs"}
}

// ProfileOwns reports whether a machine-owned file belongs to the profile's
// manifest. knowledge has no build/test harness and no issue ledger.
func ProfileOwns(profile, relPath string) bool {
	if profile != "knowledge" {
		return true
	}
	switch relPath {
	case "docs/harness-interface.md", "docs/issues/README.md", "docs/issues/_template.md":
		return false
	}
	return true
}

// Render fills placeholders in templates/<gen>/<name>; gen is "current" or "gen0".
func Render(gen, name string, v Values) (string, error) {
	raw, err := templates.FS.ReadFile(gen + "/" + name)
	if err != nil {
		return "", fmt.Errorf("template %s/%s: %w", gen, name, err)
	}
	r := strings.NewReplacer(
		"{{PROJECT}}", v.Project,
		"{{PROFILE}}", v.Profile,
		"{{REVIEWERS}}", v.Reviewers,
		"{{HARNESS}}", v.Harness,
		"{{VERSION}}", strconv.Itoa(v.Version),
	)
	return r.Replace(string(raw)), nil
}
