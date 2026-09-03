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
	// Retired marks a retired-override migration (I128): the on-disk value
	// was a deliberate override whose id is a historical id of its harness.
	// Launch validation refuses historical ids byte-exactly, so the value
	// could never launch; update replaces only the id with its successor and
	// keeps the effort and alternate the repo chose. Reported as a refresh,
	// never as a preserved override.
	Retired bool
}

// ModelOverride is one deliberate per-repo model choice (D6): a value
// matching no default the entry ever shipped, preserved untouched — or, on
// the gen-10 migration run, a per-entry effort override newly minted from a
// customized top-level effort: key (D16). Migrated distinguishes the two so
// the plan never announces a just-created override as "preserved": the
// itemized lines are the net a sweep reviewer trusts, and a net that
// mislabels what it reports is worse than no net.
type ModelOverride struct {
	Key      string
	Value    string
	Migrated bool
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
// change itemized as a refresh. Rows cover every (harness, tier) the table
// ships (design D8); a gen ≤9 repo's bare tier keys reach here as
// claude-harnessed values via the shared resolver's transitional affordance
// and land in the dotted claude rows.
//
// extracted is the on-disk key set from ExtractKeys (nil for a file being
// created), used to write an override back exactly as the repo spelled it
// and to migrate a customized top-level effort: value into per-entry
// overrides on the repo's claude entries — the only harness any generation
// that rendered that key ever dispatched (D16). Migrated entries are
// reported as overrides so the plan surfaces them.
func applyModelRouting(repoDir, content string, extracted map[string]string) (string, []ModelRefresh, []ModelOverride, error) {
	effortMigration := extracted["effort"]
	if effortMigration == effortDefault {
		effortMigration = ""
	}
	var refreshes []ModelRefresh
	var overrides []ModelOverride
	for _, harness := range model.Harnesses() {
		for _, tier := range model.Tiers {
			key := "model_routing." + harness + "." + tier
			def, err := model.Resolve("", harness, tier)
			if err != nil {
				return "", nil, nil, err
			}
			live, err := model.Resolve(repoDir, harness, tier)
			if err != nil {
				return "", nil, nil, err
			}
			target := model.MirrorValue(def)
			retired := false
			switch live.Provenance {
			case model.Override:
				target = model.MirrorValue(live)
				if raw := rawOverride(extracted, harness, tier); raw != "" {
					target = raw // the repo's own spelling, e.g. an effort suffix
				}
				// Retired override (I128): the id is historical for this
				// harness, so no launch can ever use it. Replace only the
				// id token with its successor; the rest of the repo's
				// spelling (effort, alternate) is the deliberate half and
				// survives. Itemized as a refresh so a sweep reviewer sees
				// it, distinct from the inherited kind.
				if successor, ok := model.SuccessorID(harness, tier, live.ID); ok {
					migratedValue := model.MirrorValue(model.Entry{Harness: harness, Tier: tier, ID: successor, Effort: live.Effort, Alternate: live.Alternate})
					if strings.HasPrefix(target, live.ID) {
						migratedValue = successor + strings.TrimPrefix(target, live.ID)
					}
					refreshes = append(refreshes, ModelRefresh{Key: key, Old: target, New: migratedValue, Retired: true})
					target = migratedValue
					retired = true
				}
			case model.Inherited:
				// Pair-aware (D11): an inherited value can be stale in id,
				// effort, or both; any of those is a refresh and every
				// refresh is itemized (D6).
				if live.ID != def.ID || live.Effort != def.Effort {
					refreshes = append(refreshes, ModelRefresh{Key: key, Old: model.MirrorValue(live), New: model.MirrorValue(def)})
				}
			}
			// A customized legacy effort overrides the table-rendered effort
			// for inherited Claude rows. Compare the fully rendered candidate
			// with the current default rather than just the tier default: the
			// routine tier's customized "medium" is a real override now that
			// its shipped pair is claude-opus-5 @ low, even though medium is
			// also the tier default. Existing explicit per-entry effort
			// overrides keep winning, as before.
			migrated := false
			if effortMigration != "" && harness == "claude" &&
				(live.Provenance != model.Override || !strings.Contains(target, "@")) {
				id, _, _ := strings.Cut(target, " @ ")
				candidate := model.MirrorValue(model.Entry{
					Harness: harness,
					Tier:    tier,
					ID:      id,
					Effort:  effortMigration,
				})
				if candidate != target {
					target = candidate
					migrated = true
				}
			}
			if (live.Provenance == model.Override && !retired) || migrated {
				overrides = append(overrides, ModelOverride{Key: key, Value: target, Migrated: migrated})
			}
			content = setKey(content, key, target)
		}
	}
	return content, refreshes, overrides, nil
}

// rawOverride is the repo's on-disk spelling for (harness, tier): the gen-10
// dotted key if present, else — for claude only — the bare gen ≤9 tier key
// (the same precedence the shared resolver applies).
func rawOverride(extracted map[string]string, harness, tier string) string {
	if raw := extracted["model_routing."+harness+"."+tier]; raw != "" {
		return raw
	}
	if harness == "claude" {
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
	// Any id the primary row ever shipped was a default of the lineage, not
	// a deliberate divergence from it (I128): it retires quietly whatever
	// generation rendered it.
	for _, historical := range model.HistoricalIDs("claude", "primary") {
		if md == historical {
			return "", nil
		}
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
