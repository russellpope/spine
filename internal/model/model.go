// Package model resolves the estate's model table: what model id and effort
// back a given (flavor, tier) in a given repo. The boundary is the pure
// function Resolve (repo dir + flavor + tier in, Entry out); the CLI and the
// routing audit are thin consumers wired in later tickets (ADR 0011).
//
// Two layers only (design D4): the embedded defaults in ../../models, then a
// per-repo override read from WORKFLOW.md's model_routing mirror. A repo
// with no override — including no repo at all — resolves to the embedded
// default. Flavors are open-ended data (the keys of the embedded table), not
// an enum; a third flavor is addable as data with no code change here.
//
// Design-latitude choices (the ticket leaves these open; pinned here):
//   - The override mirror this ticket reads is the dotted "<flavor>.<tier>"
//     syntax design D8 specifies for template generation 10, with an
//     optional " @ <effort>" suffix (D9). No repo renders this mirror yet —
//     nothing in this ticket writes it — so today an override is only ever
//     present in a hand-authored or test-fixture WORKFLOW.md. The existing
//     bare-tier `model_routing:` block that internal/audit and
//     internal/update read is a distinct, untouched format; a dotted key
//     never collides with a bare one, and this reader ignores bare keys.
//   - An on-disk value is Inherited when its (id, effective effort) pair
//     matches the entry's current default or any value it has ever shipped,
//     and Override otherwise (design D5/D6, corrected per task review
//     Important #1): matching on id alone let a deliberate effort override
//     on an otherwise-default id masquerade as Inherited, which would make
//     I036's refresh rule silently revert it — precisely the case user
//     stories 6/14 protect. "Effective effort" is each side's own effort if
//     set, else the tier default, so an override that repeats the default id
//     but omits effort where the default carries one (e.g. codex primary's
//     xhigh) reports Override, not Inherited, because the two sides'
//     effective efforts disagree.
//   - Effort inheritance (D3) applies uniformly to whichever entry is
//     authoritative — the override if one is present, else the shipped
//     default — so an override that sets an id but omits effort still
//     inherits the tier default, not the default entry's effort.
//   - A (flavor, tier) pair with no id in the table (an unshipped tier on an
//     otherwise-known flavor — the shape a partially-populated third flavor
//     would take) is a hard error, never a zero-value Entry: an empty model
//     id silently interpolated into a dispatch command is exactly the loud-
//     failure principle D8 itself is justified by. Guarded twice: load-time
//     validation in mustLoadDefaults fails a bad models/defaults.json at
//     test time, and Resolve itself refuses defensively.
package model

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/russellpope/spine/models"
)

// Tiers is the fixed set of semantic routing tiers (CONTEXT.md model tier).
// Unlike flavor, tiers are not open-ended: adding a fifth is a design change.
var Tiers = []string{"primary", "routine", "mechanical", "fallback"}

// Provenance classifies where a resolved entry's id/effort came from.
type Provenance string

// Provenance values.
const (
	// Default: no on-disk value for this (flavor, tier); the embedded
	// default answered the resolution.
	Default Provenance = "default"
	// Inherited: an on-disk value is present and matches the entry's
	// current default or a value it has ever shipped — a stale mirror,
	// not a deliberate choice.
	Inherited Provenance = "inherited"
	// Override: an on-disk value is present and matches no default this
	// entry has ever shipped — a deliberate per-repo choice.
	Override Provenance = "override"
)

// Entry is one resolved (flavor, tier) row.
type Entry struct {
	Flavor     string
	Tier       string
	ID         string
	Effort     string
	Aliases    []string
	Provenance Provenance
}

// tableEntry is one (flavor, tier) row as shipped in models/defaults.json.
type tableEntry struct {
	ID      string   `json:"id"`
	Effort  string   `json:"effort,omitempty"`
	Aliases []string `json:"aliases"`
	History []string `json:"history"`
}

type table struct {
	TierDefaultEffort map[string]string                `json:"tierDefaultEffort"`
	Flavors           map[string]map[string]tableEntry `json:"flavors"`
}

var defaults = mustLoadDefaults()

// mustLoadDefaults parses the embedded defaults.json and validates it's
// complete. Its presence, shape, and completeness are compile-time
// invariants of this repo (see models/embed.go), so any failure here is
// unreachable at runtime — same convention as internal/tmpl.Version's panic
// on a missing templates/VERSION.
func mustLoadDefaults() table {
	raw, err := models.FS.ReadFile("defaults.json")
	if err != nil {
		panic("models/defaults.json missing from embed: " + err.Error())
	}
	var t table
	if err := json.Unmarshal(raw, &t); err != nil {
		panic("models/defaults.json invalid: " + err.Error())
	}
	validateTable(t)
	return t
}

// validateTable enforces the completeness Resolve depends on: every flavor
// carries a non-empty id for all four Tiers, and tierDefaultEffort covers
// every tier. Without this, a data edit that ships a flavor with a partial
// tier table would resolve silently to an empty id at runtime instead of
// failing the build's own tests (task review Important #2 / Minor #6).
func validateTable(t table) {
	for _, tier := range Tiers {
		if t.TierDefaultEffort[tier] == "" {
			panic(fmt.Sprintf("models/defaults.json: tierDefaultEffort has no default for tier %q", tier))
		}
	}
	for flavor, tiers := range t.Flavors {
		for _, tier := range Tiers {
			if tiers[tier].ID == "" {
				panic(fmt.Sprintf("models/defaults.json: flavor %q has no id for tier %q", flavor, tier))
			}
		}
	}
}

// Flavors returns the known flavors, sorted. Data-driven: a third flavor
// becomes known by adding it to models/defaults.json, no code change.
func Flavors() []string {
	return flavorsOf(defaults)
}

func isKnownTier(tier string) bool {
	for _, t := range Tiers {
		if t == tier {
			return true
		}
	}
	return false
}

// Resolve answers what model id and effort back (flavor, tier) in repoDir.
// repoDir may be "", a nonexistent path, or a directory with no
// WORKFLOW.md — every one of those falls back to the embedded default (D11:
// outside a spine repo, resolution returns embedded defaults). Pure: no
// current-directory lookup, no mutation of package state.
func Resolve(repoDir, flavor, tier string) (Entry, error) {
	return resolveFrom(defaults, repoDir, flavor, tier)
}

// resolveFrom is Resolve against an explicit table rather than the package's
// loaded defaults, so tests can exercise a deliberately partial table (task
// review Important #2) without touching the real, always-complete
// models/defaults.json.
func resolveFrom(t table, repoDir, flavor, tier string) (Entry, error) {
	tiers, ok := t.Flavors[flavor]
	if !ok {
		return Entry{}, fmt.Errorf("unknown flavor %q (known: %s)", flavor, strings.Join(flavorsOf(t), ", "))
	}
	if !isKnownTier(tier) {
		return Entry{}, fmt.Errorf("unknown tier %q (known: %s)", tier, strings.Join(Tiers, ", "))
	}
	def := tiers[tier]
	if def.ID == "" {
		return Entry{}, fmt.Errorf("flavor %q has no %s entry", flavor, tier)
	}
	tierDefaultEffort := t.TierDefaultEffort[tier]

	entry := Entry{
		Flavor: flavor, Tier: tier,
		ID: def.ID, Effort: def.Effort, Aliases: def.Aliases,
		Provenance: Default,
	}

	if ov, found := readOverride(repoDir, flavor, tier); found {
		entry.ID = ov.id
		entry.Effort = ov.effort
		if everShipped(def, tierDefaultEffort, ov.id, ov.effort) {
			entry.Provenance = Inherited
		} else {
			entry.Provenance = Override
		}
	}

	if entry.Effort == "" {
		entry.Effort = tierDefaultEffort
	}
	return entry, nil
}

// everShipped reports whether (id, effort) was ever the shipped default for
// def: id must be the current id or in its history, AND the effective
// effort — the caller's effort if set, else the tier default — must match
// def's own effective effort the same way (task review Important #1). This
// keeps a deliberate effort override on an otherwise-default id from being
// misreported as Inherited, and keeps an entry that resolves to a different
// effort than what a matching id actually shipped with from being reported
// as Inherited either.
func everShipped(def tableEntry, tierDefaultEffort, id, effort string) bool {
	if id != def.ID && !containsStr(def.History, id) {
		return false
	}
	shippedEffort := def.Effort
	if shippedEffort == "" {
		shippedEffort = tierDefaultEffort
	}
	effective := effort
	if effective == "" {
		effective = tierDefaultEffort
	}
	return effective == shippedEffort
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func flavorsOf(t table) []string {
	out := make([]string, 0, len(t.Flavors))
	for f := range t.Flavors {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// override is one on-disk "<flavor>.<tier>" mirror value.
type override struct {
	id     string
	effort string
}

// readOverride looks up repoDir's WORKFLOW.md model_routing block for the
// dotted "<flavor>.<tier>" key design D8 defines for the gen-10 mirror.
// Absence of repoDir, the file, the block, or the key all report not-found,
// never an error — that is the "no override" and "outside a spine repo"
// cases, not failures.
func readOverride(repoDir, flavor, tier string) (override, bool) {
	if repoDir == "" {
		return override{}, false
	}
	raw, err := os.ReadFile(filepath.Join(repoDir, "WORKFLOW.md"))
	if err != nil {
		return override{}, false
	}
	want := flavor + "." + tier
	inBlock := false
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "model_routing:") {
			inBlock = true
			continue
		}
		if !inBlock {
			continue
		}
		if !strings.HasPrefix(line, "  ") || strings.TrimSpace(line) == "" {
			break
		}
		k, v, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok || strings.TrimSpace(k) != want {
			continue
		}
		return parseValue(v), true
	}
	return override{}, false
}

// parseValue splits one mirror value into id and optional effort per D9:
// "<id>" or "<id> @ <effort>", comment stripped first as the existing
// WORKFLOW.md parsers do.
func parseValue(v string) override {
	v = stripComment(v)
	id, effort, hasEffort := strings.Cut(v, "@")
	ov := override{id: strings.TrimSpace(id)}
	if hasEffort {
		ov.effort = strings.TrimSpace(effort)
	}
	return ov
}

func stripComment(v string) string {
	if i := strings.Index(v, "#"); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}
