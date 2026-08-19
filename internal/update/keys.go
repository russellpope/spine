// Package update regenerates machine-owned workflow files from the compiled
// templates, preserving deliberate per-repo choices (spec: ownership split +
// config-preserving regeneration + choice-vs-default rule).
package update

import (
	"fmt"
	"strings"

	"github.com/russellpope/spine/internal/model"
	"github.com/russellpope/spine/internal/tmpl"
)

var topKeys = []string{
	"profile", "template_version", "reviewers", "functional_harness", "gates",
	"effort", "model_default", "security_routing", "stages",
	"gate_pack", "gate_pack_disabled", "gate_pack_config",
}

// isRoutingKey reports whether k names a bare routing tier. The tier list is
// the resolver's — one definition estate-wide.
func isRoutingKey(k string) bool {
	for _, r := range model.Tiers {
		if r == k {
			return true
		}
	}
	return false
}

func splitLines(s string) []string { return strings.Split(s, "\n") }

// cutKey returns the value of "key: value  # comment" with comment stripped.
func cutKey(line, key string) (string, bool) {
	rest, ok := strings.CutPrefix(line, key+":")
	if !ok {
		return "", false
	}
	if i := commentIndex(rest); i >= 0 {
		rest = rest[:i]
	}
	return strings.TrimSpace(rest), true
}

// commentIndex delegates to the consolidated comment rule in internal/model
// (I037): one definition of where a trailing comment starts, shared by every
// WORKFLOW.md reader.
func commentIndex(s string) int { return model.CommentIndex(s) }

// ExtractKeys pulls known config keys out of WORKFLOW.md content. Sub-keys of
// model_routing come back dotted — "model_routing.primary" for a bare gen ≤9
// tier key, "model_routing.claude.primary" for a gen-10 mirror key — read
// through the shared model_routing parser (model.RoutingKeys, I037): one
// block grammar for dispatch resolution, update, and audit alike. Top-level
// keys are scanned line-wise; routing-block lines are indented and can never
// match a top-level key prefix.
func ExtractKeys(content string) map[string]string {
	keys := map[string]string{}
	for k, v := range model.RoutingKeys(content) {
		keys["model_routing."+k] = v
	}
	inGateConfig := false
	for _, line := range splitLines(content) {
		// gate_pack_config sub-keys come back dotted too, the same shape
		// model_routing rows use: an indented row under the block header.
		if strings.HasPrefix(line, "gate_pack_config:") {
			inGateConfig = true
		} else if inGateConfig {
			if sub := strings.TrimSpace(line); strings.HasPrefix(line, "  ") && sub != "" {
				for _, k := range gatePackConfigKeys {
					if v, ok := cutKey(sub, k); ok {
						keys["gate_pack_config."+k] = v
					}
				}
			} else {
				inGateConfig = false
			}
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
		if strings.HasPrefix(k, "model_routing.") {
			// Model-routing values never enter the choice-vs-default rule
			// (design D7): the shared model table resolves them via
			// applyModelRouting. Left here, this rule would misread any
			// stale inherited default as a deliberate choice — the
			// propagation trap that made a template model-id change reach
			// zero repos — and fight the resolver over the same keys.
			continue
		}
		if k == "effort" || k == "model_default" {
			// Both retire in gen 10 (design D16 + controller ruling, I036):
			// effort migrates into per-entry overrides and model_default is
			// checked against the resolved primary — in applyModelRouting,
			// outside this rule for the same D7 reason as the routing keys.
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
	keyIdx := strings.Index(line, key)
	indent := line[:keyIdx]
	rest := line[keyIdx+len(key)+2:] // skip "key: "

	// Extract old value (before comment or end of line)
	oldVal := rest
	if i := commentIndex(rest); i >= 0 {
		oldVal = rest[:i]
	}

	// If value hasn't changed, preserve the line exactly as-is — including
	// any alignment padding between key and value, which the gen-10 mirror
	// rows carry (design D8) and which is never itself a value change.
	if strings.TrimSpace(oldVal) == val {
		return line
	}

	// Value changed: normalize spacing to 4 spaces
	comment := ""
	if i := commentIndex(rest); i >= 0 {
		comment = "    " + strings.TrimRight(rest[i:], " ")
	}
	return indent + key + ": " + val + comment
}
