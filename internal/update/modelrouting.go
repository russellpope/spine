package update

import (
	"fmt"
	"strings"

	"github.com/russellpope/spine/internal/model"
)

// ModelRefresh is one itemized model-table refresh (design D6): the on-disk
// value matched a shipped default — current or historical — at its shipped
// effort, so it is an inherited stale mirror rather than a choice, and
// update moves it to the current default. Surfaced individually in the plan
// because on a fleet sweep an unitemized model change is invisible among
// template prose churn. Old and New are mirror values (design D9), so an
// effort-only refresh itemizes too — itemization is pair-aware, matching
// D11's (id, effort) definition of a value (I035 carry-forward).
type ModelRefresh struct {
	Key string // dotted config key, e.g. "model_routing.claude.fallback"
	Old string
	New string
}

// ModelOverride is one deliberate per-repo model choice (D6): a value
// matching no default the entry ever shipped, preserved untouched — or, on
// the gen-10 migration run, a per-entry effort override newly minted from a
// customized top-level effort: key (D16).
type ModelOverride struct {
	Key   string
	Value string
}

// effortDefault is the only value the retired top-level effort: key ever
// shipped as its default, in every generation that rendered it (gen 5-9); a
// repo carrying anything else customized it, and that customization migrates
// into per-entry overrides (design D16) rather than being discarded.
const effortDefault = "high"

// applyModelRouting sets every model_routing mirror row of content from the
// shared model table instead of the choice-vs-default rule (design D6/D7,
// ADR 0011): an override keeps the repo's on-disk value verbatim; everything
// else — inherited historical defaults and the table-rendered template rows
// alike — is set to the table's current default, with each actual value
// change itemized as a refresh. Rows cover every (flavor, tier) the table
// ships (design D8); a gen ≤9 repo's bare tier keys reach here as
// claude-flavored values via the shared resolver's transitional affordance
// and land in the dotted claude rows.
//
// extracted is the on-disk key set from ExtractKeys (nil for a file being
// created), used to write an override back exactly as the repo spelled it
// and to migrate a customized top-level effort: value into per-entry
// overrides on the repo's claude entries — the only flavor any generation
// that rendered that key ever dispatched (D16). Migrated entries are
// reported as overrides so the plan surfaces them.
func applyModelRouting(repoDir, content string, extracted map[string]string) (string, []ModelRefresh, []ModelOverride, error) {
	effortMigration := extracted["effort"]
	if effortMigration == effortDefault {
		effortMigration = ""
	}
	var refreshes []ModelRefresh
	var overrides []ModelOverride
	for _, flavor := range model.Flavors() {
		for _, tier := range model.Tiers {
			key := "model_routing." + flavor + "." + tier
			def, err := model.Resolve("", flavor, tier)
			if err != nil {
				return "", nil, nil, err
			}
			live, err := model.Resolve(repoDir, flavor, tier)
			if err != nil {
				return "", nil, nil, err
			}
			target := model.MirrorValue(def)
			switch live.Provenance {
			case model.Override:
				target = model.MirrorValue(live)
				if raw := rawOverride(extracted, flavor, tier); raw != "" {
					target = raw // the repo's own spelling, e.g. an effort suffix
				}
			case model.Inherited:
				// Pair-aware (D11): an inherited value can be stale in id,
				// effort, or both; any of those is a refresh and every
				// refresh is itemized (D6).
				if live.ID != def.ID || live.Effort != def.Effort {
					refreshes = append(refreshes, ModelRefresh{Key: key, Old: model.MirrorValue(live), New: model.MirrorValue(def)})
				}
			}
			migrated := false
			if effortMigration != "" && flavor == "claude" && !strings.Contains(target, "@") {
				target += " @ " + effortMigration
				migrated = true
			}
			if live.Provenance == model.Override || migrated {
				overrides = append(overrides, ModelOverride{Key: key, Value: target})
			}
			content = setKey(content, key, target)
		}
	}
	return content, refreshes, overrides, nil
}

// rawOverride is the repo's on-disk spelling for (flavor, tier): the gen-10
// dotted key if present, else — for claude only — the bare gen ≤9 tier key
// (the same precedence the shared resolver applies).
func rawOverride(extracted map[string]string, flavor, tier string) string {
	if raw := extracted["model_routing."+flavor+"."+tier]; raw != "" {
		return raw
	}
	if flavor == "claude" {
		return extracted["model_routing."+tier]
	}
	return ""
}

// modelDefaultDivergence implements the retired model_default: key's careful
// case (I036 controller ruling): the key duplicates model_routing's claude
// primary row and has zero consumers, so it retires — but a value the repo
// deliberately customized that disagrees with the resolved primary is a
// genuine divergence, surfaced for a human decision rather than silently
// dropped or silently promoted. A value equal to its own generation's
// shipped default was never a deliberate choice (the same choice-vs-default
// convention the rest of update applies, ADR 0002) and retires quietly, as
// does one equal to the resolved primary (nothing is lost — the primary row
// carries it). Returns "" when there is nothing to surface.
func modelDefaultDivergence(repoDir, gen string, extracted map[string]string) (string, error) {
	md := extracted["model_default"]
	if md == "" {
		return "", nil
	}
	shipped := "claude-fable-5" // gen 1-9 rendered default
	if gen == "gen0" {
		shipped = "claude-opus-4-8"
	}
	if md == shipped {
		return "", nil
	}
	def, err := model.Resolve("", "claude", "primary")
	if err != nil {
		return "", err
	}
	live, err := model.Resolve(repoDir, "claude", "primary")
	if err != nil {
		return "", err
	}
	resolved := def.ID
	if live.Provenance == model.Override {
		resolved = live.ID
	}
	if md == resolved {
		return "", nil
	}
	return fmt.Sprintf(
		"model_default: %s diverges from model_routing claude.primary (%s) — the retired key holds a deliberate value; adopt it as a claude.primary override or delete the line, then re-run",
		md, resolved), nil
}
