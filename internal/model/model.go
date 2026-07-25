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
//   - The override mirror this reader prefers is the dotted "<flavor>.<tier>"
//     syntax design D8 specifies for template generation 10, with an
//     optional " @ <effort>" suffix (D9). No repo renders this mirror yet.
//     TRANSITIONAL (I035, superseded by I036): a bare tier key
//     (`fallback: claude-opus-4-8`) — the format every gen ≤9 mirror
//     actually carries — is also read, as a claude-flavored override, since
//     claude is the only flavor those generations ever rendered. Without
//     this, the I035 refresh rule would see zero overrides in every real
//     repo while passing dotted-key-only tests. A dotted key wins over a
//     bare one for the same tier; bare keys are invisible to every other
//     flavor. I036's gen-10 mirror retires the bare format.
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
	ID      string         `json:"id"`
	Effort  string         `json:"effort,omitempty"`
	Aliases []string       `json:"aliases"`
	History []historyEntry `json:"history"`
}

// historyEntry is one previously shipped default as the (id, effort) pair it
// actually shipped as (D11: inherited means matching "a shipped historical
// pair", not a bare id). Effort "" means the pair shipped at the tier's
// default effort, resolved at comparison time exactly like a current entry's
// omitted effort (D3). Without the pair, a historical id would be compared
// against the CURRENT default's effort — harmless until the first time a
// default's effort changes across a history entry, then wrong.
type historyEntry struct {
	ID     string `json:"id"`
	Effort string `json:"effort,omitempty"`
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
			for _, h := range tiers[tier].History {
				if h.ID == "" {
					panic(fmt.Sprintf("models/defaults.json: flavor %q tier %q has a history entry with no id", flavor, tier))
				}
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

// everShipped reports whether (id, effort) was ever a shipped default pair
// for def: the current (id, effort) or any pair in its history. Each side's
// effective effort is its own effort if set, else the tier default (task
// review Important #1), so a historical id only counts as shipped at the
// effort it actually shipped with — never at whatever the current default's
// effort happens to be. This keeps a deliberate effort override on an
// otherwise-default id from being misreported as Inherited, in either
// direction.
func everShipped(def tableEntry, tierDefaultEffort, id, effort string) bool {
	effective := effort
	if effective == "" {
		effective = tierDefaultEffort
	}
	matches := func(shippedID, shippedEffort string) bool {
		if shippedEffort == "" {
			shippedEffort = tierDefaultEffort
		}
		return id == shippedID && effective == shippedEffort
	}
	if matches(def.ID, def.Effort) {
		return true
	}
	for _, h := range def.History {
		if matches(h.ID, h.Effort) {
			return true
		}
	}
	return false
}

// TierDefaultEffort returns the effort a tier resolves to when an entry
// omits one (design D3) — the same table MirrorValue canonicalizes against.
// Exposed so the update path's effort migration (D16) can skip minting a
// per-entry override that only restates the tier default, which the mirror
// rendering would canonicalize away on the next run.
func TierDefaultEffort(tier string) string {
	return defaults.TierDefaultEffort[tier]
}

// MirrorValue renders a resolved entry as the value half of a gen-10 mirror
// row (design D9): the model id, followed by " @ <effort>" only when the
// entry's effective effort deviates from its tier's default. An effort equal
// to the tier default is omitted so the common case stays a bare id and the
// suffix always signals a real deviation.
func MirrorValue(e Entry) string {
	if e.Effort != "" && e.Effort != defaults.TierDefaultEffort[e.Tier] {
		return e.ID + " @ " + e.Effort
	}
	return e.ID
}

// MirrorRows renders the embedded defaults as the gen-10 WORKFLOW.md mirror
// rows (design D8): one two-space-indented "<flavor>.<tier>: <value>" line
// per entry, flavors sorted, tiers in Tiers order, values column-aligned.
// Rendered from the table — never from literals in a template — so a
// defaults change propagates to init/update with no template edit. Embedded
// defaults only: no repo is consulted.
func MirrorRows() []string {
	type row struct{ key, val string }
	var rows []row
	width := 0
	for _, flavor := range flavorsOf(defaults) {
		for _, tier := range Tiers {
			def := defaults.Flavors[flavor][tier]
			e := Entry{Flavor: flavor, Tier: tier, ID: def.ID, Effort: def.Effort}
			if e.Effort == "" {
				e.Effort = defaults.TierDefaultEffort[tier]
			}
			k := flavor + "." + tier + ":"
			if len(k) > width {
				width = len(k)
			}
			rows = append(rows, row{k, MirrorValue(e)})
		}
	}
	lines := make([]string, 0, len(rows))
	for _, r := range rows {
		lines = append(lines, "  "+r.key+strings.Repeat(" ", width-len(r.key)+1)+r.val)
	}
	return lines
}

func flavorsOf(t table) []string {
	out := make([]string, 0, len(t.Flavors))
	for f := range t.Flavors {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// HistoricalIDs returns every model id the embedded (flavor, tier) entry
// shipped as a default before the current one, in shipped order. The routing
// audit maps transcript tokens from pre-refresh dispatches through these by
// exact id — historical ids carry no aliases (I033), so an alias token in an
// old transcript deliberately does not match them. Unknown flavor or tier
// returns nil.
func HistoricalIDs(flavor, tier string) []string {
	var ids []string
	for _, h := range defaults.Flavors[flavor][tier].History {
		ids = append(ids, h.ID)
	}
	return ids
}

// RoutingKeys parses the model_routing: block out of WORKFLOW.md content —
// the single parser of that block (I037 consolidation, design D13). Before
// this, three independent readers (this package's override reader, update's
// ExtractKeys, audit's mapping reader) each parsed it with diverging
// block-termination and comment rules; every consumer now reads through here
// (the audit indirectly, via Resolve).
//
// Grammar: the block starts at a column-0 "model_routing:" line and is the
// contiguous run of two-space-indented, non-blank lines after it; the first
// whitespace-only or non-indented line ends it, and scanning stops there.
// The mirror is machine-rendered contiguous, so nothing past a break is
// safely attributable to the block — the conservative rule, kept from the
// resolver's and audit's original readers (2 of the 3; ExtractKeys used to
// scan on past whitespace-only lines and could read indented prose as
// routing entries).
//
// Keys: bare known-tier keys ("fallback" — the gen ≤9 format the un-swept
// fleet runs until I039) and dotted "<flavor>.<tier>" gen-10 mirror keys
// (design D8). Unknown keys are ignored by contract; the last occurrence of
// a duplicated key wins (map semantics — duplicates never occur in a
// machine-rendered mirror). Values come back with any trailing comment
// stripped first, per CommentIndex, before any caller splits on the " @ "
// effort separator — D9's corrected order.
func RoutingKeys(content string) map[string]string {
	keys := map[string]string{}
	inBlock := false
	for _, line := range strings.Split(content, "\n") {
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
		trimmed := strings.TrimSpace(line)
		k, v, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		key := strings.TrimSpace(k)
		if !isKnownTier(key) {
			if _, dotted := DottedRoutingKey(trimmed); !dotted {
				continue
			}
		}
		if i := CommentIndex(v); i >= 0 {
			v = v[:i]
		}
		keys[key] = strings.TrimSpace(v)
	}
	return keys
}

// DottedRoutingKey reports whether trimmed begins with a gen-10
// "<flavor>.<tier>:" mirror key (design D8) and returns the dotted key. The
// flavor half is open-ended data, so any dot-free non-empty token counts;
// the tier half must be a known tier. (Moved from internal/update in the
// I037 parser consolidation; update's line-signature scan still uses it.)
func DottedRoutingKey(trimmed string) (string, bool) {
	head, _, found := strings.Cut(trimmed, ":")
	if !found {
		return "", false
	}
	flavor, tier, dotted := strings.Cut(head, ".")
	if !dotted || flavor == "" || strings.ContainsAny(flavor, " \t.") || !isKnownTier(tier) {
		return "", false
	}
	return head, true
}

// CommentIndex finds the offset of a comment-starting '#' in s: one that
// begins the value (only whitespace, if any, precedes it) or is itself
// preceded by whitespace. A '#' embedded inside a value (e.g.
// "quality#framing") is data, not a comment, and is skipped. Returns -1
// when s has no comment-starting '#'. (Moved from internal/update in the
// I037 consolidation so every WORKFLOW.md reader shares one comment rule.)
func CommentIndex(s string) int {
	start := len(s) - len(strings.TrimLeft(s, " \t"))
	for i := start; i < len(s); i++ {
		if s[i] != '#' {
			continue
		}
		if i == start || s[i-1] == ' ' || s[i-1] == '\t' {
			return i
		}
	}
	return -1
}

// override is one on-disk "<flavor>.<tier>" mirror value.
type override struct {
	id     string
	effort string
}

// readOverride looks up repoDir's WORKFLOW.md model_routing block for the
// dotted "<flavor>.<tier>" key design D8 defines for the gen-10 mirror,
// falling back to the bare "<tier>" key for the claude flavor — the
// TRANSITIONAL gen ≤9 affordance described in the package comment (I035,
// retired by I036's mirror): claude is the only flavor any gen ≤9 mirror
// ever rendered, and bare keys stay invisible to every other flavor. A
// dotted key wins over a bare one. Absence of repoDir, the file, the block,
// or the key all report not-found, never an error — that is the "no
// override" and "outside a spine repo" cases, not failures.
func readOverride(repoDir, flavor, tier string) (override, bool) {
	if repoDir == "" {
		return override{}, false
	}
	raw, err := os.ReadFile(filepath.Join(repoDir, "WORKFLOW.md"))
	if err != nil {
		return override{}, false
	}
	keys := RoutingKeys(string(raw))
	if v, ok := keys[flavor+"."+tier]; ok {
		return parseValue(v), true
	}
	if flavor == "claude" {
		if v, ok := keys[tier]; ok {
			return parseValue(v), true
		}
	}
	return override{}, false
}

// parseValue splits one mirror value into id and optional effort per D9:
// "<id>" or "<id> @ <effort>". RoutingKeys has already stripped any trailing
// comment — D9's corrected order (comment first, then the separator split).
func parseValue(v string) override {
	id, effort, hasEffort := strings.Cut(v, "@")
	ov := override{id: strings.TrimSpace(id)}
	if hasEffort {
		ov.effort = strings.TrimSpace(effort)
	}
	return ov
}
