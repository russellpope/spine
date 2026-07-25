package update

import (
	"github.com/russellpope/spine/internal/model"
)

// ModelRefresh is one itemized model-table refresh (design D6): the on-disk
// value matched a shipped default — current or historical — at its shipped
// effort, so it is an inherited stale mirror rather than a choice, and
// update moves it to the current default. Surfaced individually in the plan
// because on a fleet sweep an unitemized model change is invisible among
// template prose churn.
type ModelRefresh struct {
	Key string // dotted config key, e.g. "model_routing.fallback"
	Old string
	New string
}

// ModelOverride is one preserved deliberate per-repo model choice (D6): the
// on-disk value matches no default the entry ever shipped, so update leaves
// it untouched and reports it as an override.
type ModelOverride struct {
	Key   string
	Value string
}

// applyModelRouting sets every model_routing row of content from the shared
// model table instead of the choice-vs-default rule (design D6/D7, ADR
// 0011): an override keeps the repo's on-disk value verbatim; everything
// else — inherited historical defaults and the template's own rendered text
// alike — is set to the table's current default, with each actual value
// change itemized as a refresh. extracted is the on-disk key set from
// ExtractKeys (nil for a file being created), used to write an override back
// exactly as the repo spelled it. The rows written are the bare claude tier
// keys this generation renders; the dotted multi-flavor mirror is I036.
func applyModelRouting(repoDir, content string, extracted map[string]string) (string, []ModelRefresh, []ModelOverride, error) {
	var refreshes []ModelRefresh
	var overrides []ModelOverride
	for _, tier := range model.Tiers {
		key := "model_routing." + tier
		def, err := model.Resolve("", "claude", tier)
		if err != nil {
			return "", nil, nil, err
		}
		live, err := model.Resolve(repoDir, "claude", tier)
		if err != nil {
			return "", nil, nil, err
		}
		target := def.ID
		switch live.Provenance {
		case model.Override:
			target = live.ID
			if raw := extracted[key]; raw != "" {
				target = raw // the repo's own spelling, e.g. an effort suffix
			}
			overrides = append(overrides, ModelOverride{Key: key, Value: target})
		case model.Inherited:
			if live.ID != def.ID {
				refreshes = append(refreshes, ModelRefresh{Key: key, Old: live.ID, New: def.ID})
			}
		}
		content = setKey(content, key, target)
	}
	return content, refreshes, overrides, nil
}
