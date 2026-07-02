// Package update regenerates machine-owned workflow files from the compiled
// templates, preserving deliberate per-repo choices (spec: ownership split +
// config-preserving regeneration + choice-vs-default rule).
package update

import (
	"fmt"
	"strings"

	"github.com/russellpope/spine/internal/tmpl"
)

var topKeys = []string{
	"profile", "template_version", "reviewers", "functional_harness", "gates",
	"effort", "model_default", "security_routing", "stages",
}

var routingKeys = []string{"primary", "fallback", "routine"}

func splitLines(s string) []string { return strings.Split(s, "\n") }

// cutKey returns the value of "key: value  # comment" with comment stripped.
func cutKey(line, key string) (string, bool) {
	rest, ok := strings.CutPrefix(line, key+":")
	if !ok {
		return "", false
	}
	if i := strings.Index(rest, "#"); i >= 0 {
		rest = rest[:i]
	}
	return strings.TrimSpace(rest), true
}

// ExtractKeys pulls known config keys out of WORKFLOW.md content. Sub-keys of
// model_routing come back dotted: "model_routing.primary".
func ExtractKeys(content string) map[string]string {
	keys := map[string]string{}
	inRouting := false
	for _, line := range splitLines(content) {
		if strings.HasPrefix(line, "model_routing:") {
			inRouting = true
			continue
		}
		if inRouting {
			if strings.HasPrefix(line, "  ") {
				trimmed := strings.TrimSpace(line)
				for _, k := range routingKeys {
					if v, ok := cutKey(trimmed, k); ok {
						keys["model_routing."+k] = v
					}
				}
				continue
			}
			inRouting = false
		}
		for _, k := range topKeys {
			if v, ok := cutKey(line, k); ok {
				keys[k] = v
			}
		}
	}
	return keys
}

// ProjectFromWorkflow reads the project name from the "# Workflow — X" title.
func ProjectFromWorkflow(content, fallback string) string {
	for _, line := range splitLines(content) {
		if rest, ok := strings.CutPrefix(line, "# Workflow — "); ok {
			return strings.TrimSpace(rest)
		}
	}
	return fallback
}

// Choices filters extracted keys down to deliberate user choices: values that
// differ from what the file's own generation would have rendered by default.
// profile is always a choice; template_version never is.
func Choices(extracted map[string]string, gen, project string) (map[string]string, error) {
	profile := extracted["profile"]
	if profile == "" {
		return nil, fmt.Errorf("no profile: key found in WORKFLOW.md")
	}
	reviewers, harness, err := tmpl.Defaults(profile)
	if err != nil {
		return nil, err
	}
	rendered, err := tmpl.Render(gen, "WORKFLOW.md.tmpl", tmpl.Values{
		Project: project, Profile: profile, Reviewers: reviewers, Harness: harness, Version: tmpl.Version(),
	})
	if err != nil {
		return nil, err
	}
	defaults := ExtractKeys(rendered)
	choices := map[string]string{"profile": profile}
	for k, v := range extracted {
		if k == "profile" || k == "template_version" {
			continue
		}
		if defaults[k] != v {
			choices[k] = v
		}
	}
	return choices, nil
}

// setKey rewrites the value of a top-level or model_routing.* key in rendered
// WORKFLOW.md content, keeping the template's trailing comment.
func setKey(content, dotted, val string) string {
	top, sub, isSub := strings.Cut(dotted, ".")
	lines := splitLines(content)
	inBlock := false
	for i, line := range lines {
		if isSub {
			if strings.HasPrefix(line, top+":") {
				inBlock = true
				continue
			}
			if !inBlock {
				continue
			}
			if !strings.HasPrefix(line, "  ") {
				inBlock = false
				continue
			}
			if strings.HasPrefix(strings.TrimSpace(line), sub+":") {
				lines[i] = replaceValue(line, sub, val)
				return strings.Join(lines, "\n")
			}
			continue
		}
		if strings.HasPrefix(line, top+":") {
			lines[i] = replaceValue(line, top, val)
			return strings.Join(lines, "\n")
		}
	}
	return strings.Join(lines, "\n")
}

func replaceValue(line, key, val string) string {
	indent := line[:strings.Index(line, key)]
	comment := ""
	if i := strings.Index(line, "#"); i >= 0 {
		comment = "    " + strings.TrimRight(line[i:], " ")
	}
	return indent + key + ": " + val + comment
}
