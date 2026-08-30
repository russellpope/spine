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
//     optional " @ <effort>" suffix (D9); gen-10 repos render it (I036).
//     TRANSITIONAL (I035, emitted format retired by I036): a bare tier key
//     (`fallback: claude-opus-4-8`) — the format every gen ≤9 mirror
//     actually carries — is also read, as a claude-flavored override, since
//     claude is the only flavor those generations ever rendered. Without
//     this, the I035 refresh rule would see zero overrides in every real
//     repo while passing dotted-key-only tests. A dotted key wins over a
//     bare one for the same tier; bare keys are invisible to every other
//     flavor. The EMITTED format is retired in gen-10 mirrors (I036), but
//     the READ affordance stays live: 18 gen-9 worktrees remain until their
//     branches merge/rebase, and resolution with --dir may target a worktree.
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
//   - A cell may carry an optional alternate (CONTEXT.md "alternate"): an
//     (id, effort) pair rendered on the mirror row as a trailing
//     " alt: <id> @ <effort>" and read back under the same rules, and part
//     of the pair everShipped compares, so editing or deleting only a cell's
//     alternate reads as the deliberate choice it is (I079).
//   - A flavor may declare an effort vocabulary in the table
//     (effortVocabulary); an effort outside it — the pi harness has no
//     "high" — is refused at resolution rather than mapped onto a
//     neighbouring level. Translating an effort to a model's own reasoning
//     aliases is the harness's job, not spine's. Flavors with no declared
//     vocabulary are unconstrained, as claude and codex always were.
//   - A (flavor, tier) pair with no id in the table (an unshipped tier on an
//     otherwise-known flavor — the shape a partially-populated third flavor
//     would take) is a hard error, never a zero-value Entry: an empty model
//     id silently interpolated into a dispatch command is exactly the loud-
//     failure principle D8 itself is justified by. Guarded twice: load-time
//     validation in mustLoadDefaults fails a bad models/defaults.json at
//     test time, and Resolve itself refuses defensively.
package model

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/russellpope/spine/internal/hostconfig"
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

// Alternate is a cell's owner-tuned alternate (CONTEXT.md "alternate"): the
// (id, effort) pair a second dispatch on the same cell uses — the maipipe
// critic against the author, say — so the difference between the two is
// data in the table, not a dispatch-time heuristic (I079). The same id at a
// different effort is a legitimate alternate.
type Alternate struct {
	ID     string `json:"id"`
	Effort string `json:"effort,omitempty"`
}

// Entry is one resolved (flavor, tier) row. Alternate is nil for a cell
// that ships none — most cells do not.
type Entry struct {
	Flavor     string
	Tier       string
	ID         string
	Effort     string
	Aliases    []string
	Alternate  *Alternate
	Provenance Provenance
}

// tableEntry is one (flavor, tier) row as shipped in models/defaults.json.
type tableEntry struct {
	ID        string         `json:"id"`
	Effort    string         `json:"effort,omitempty"`
	Aliases   []string       `json:"aliases"`
	Alternate *Alternate     `json:"alternate,omitempty"`
	History   []historyEntry `json:"history"`
}

// historyEntry is one previously shipped default as the (id, effort) pair it
// actually shipped as (D11: inherited means matching "a shipped historical
// pair", not a bare id). Effort "" means the pair shipped at the tier's
// default effort, resolved at comparison time exactly like a current entry's
// omitted effort (D3). Without the pair, a historical id would be compared
// against the CURRENT default's effort — harmless until the first time a
// default's effort changes across a history entry, then wrong.
// A history entry may carry an alternate too, so inherited-vs-override
// detection covers the alternate half of a cell rather than only its id and
// effort (I079).
type historyEntry struct {
	ID        string     `json:"id"`
	Effort    string     `json:"effort,omitempty"`
	Alternate *Alternate `json:"alternate,omitempty"`
}

type forbiddenPattern struct {
	Name string `json:"name"`
	RE   string `json:"re"`

	compiled *regexp.Regexp
}

type modelValidationPolicy struct {
	IDPattern         string             `json:"idPattern"`
	ForbiddenTokens   []string           `json:"forbiddenTokens"`
	ForbiddenPatterns []forbiddenPattern `json:"forbiddenPatterns"`

	idPattern *regexp.Regexp
}

type table struct {
	TierDefaultEffort map[string]string `json:"tierDefaultEffort"`
	// TierDefaultEffortByFlavor overrides TierDefaultEffort for one flavor
	// (I079 fix round 1): the global map's primary/fallback "high" is a word
	// the pi harness's effort vocabulary does not have, so a bare-id pi
	// mirror row would inherit an effort pi cannot run. A flavor listed here
	// supplies the tier defaults for its own cells; every tier it omits, and
	// every flavor absent from the map, falls back to the global map.
	TierDefaultEffortByFlavor map[string]map[string]string     `json:"tierDefaultEffortByFlavor"`
	EffortVocabulary          map[string][]string              `json:"effortVocabulary"`
	ModelValidation           *modelValidationPolicy           `json:"modelValidation"`
	Flavors                   map[string]map[string]tableEntry `json:"flavors"`
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
	t, err := decodeTable(raw)
	if err != nil {
		panic("models/defaults.json invalid: " + err.Error())
	}
	validateTable(t)
	return t
}

func decodeTable(raw []byte) (table, error) {
	if err := rejectDuplicateJSONMembers(raw); err != nil {
		return table{}, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var t table
	if err := dec.Decode(&t); err != nil {
		return table{}, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return table{}, fmt.Errorf("multiple JSON values")
		}
		return table{}, err
	}
	return t, nil
}

func rejectDuplicateJSONMembers(raw []byte) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := scanJSONValue(dec); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}

func scanJSONValue(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]struct{}{}
		for dec.More() {
			keyToken, err := dec.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("JSON object member name is not a string")
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("duplicate JSON member %q", key)
			}
			seen[key] = struct{}{}
			if err := scanJSONValue(dec); err != nil {
				return err
			}
		}
		_, err = dec.Token()
		return err
	case '[':
		for dec.More() {
			if err := scanJSONValue(dec); err != nil {
				return err
			}
		}
		_, err = dec.Token()
		return err
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
}

// validateTable enforces the completeness Resolve depends on: every flavor
// carries a non-empty id for all four Tiers, and tierDefaultEffort covers
// every tier. Without this, a data edit that ships a flavor with a partial
// tier table would resolve silently to an empty id at runtime instead of
// failing the build's own tests (task review Important #2 / Minor #6).
func validateTable(t table) {
	validateModelPolicy(t)
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
			validateShippedModelIDs(t, flavor, tier, tiers[tier])
			checkAlternate := func(what string, alt *Alternate) {
				if alt == nil {
					return
				}
				if alt.ID == "" {
					panic(fmt.Sprintf("models/defaults.json: flavor %q tier %q %s alternate has no id", flavor, tier, what))
				}
				if alt.Effort == "" {
					panic(fmt.Sprintf("models/defaults.json: flavor %q tier %q %s alternate has no effort", flavor, tier, what))
				}
			}
			checkAlternate("current", tiers[tier].Alternate)
			for _, h := range tiers[tier].History {
				if h.ID == "" {
					panic(fmt.Sprintf("models/defaults.json: flavor %q tier %q has a history entry with no id", flavor, tier))
				}
				checkAlternate("history", h.Alternate)
			}
			// A shipped cell must speak its own flavor's effort vocabulary,
			// so a data edit that ships (say) a pi cell at "high" fails the
			// build's tests rather than erroring at every resolution.
			// The effort a cell INHERITS must be speakable too (I079 fix
			// round 1): without this, a flavor whose vocabulary excludes the
			// global tier default ships a table where a bare-id mirror row
			// is unresolvable, which is exactly the bug the per-flavor
			// tierDefaultEffortByFlavor override exists to prevent.
			efforts := append(shippedEfforts(tiers[tier]), tierDefaultEffortOf(t, flavor, tier))
			for _, effort := range efforts {
				if err := checkEffort(t, flavor, effort); err != nil {
					panic(fmt.Sprintf("models/defaults.json: flavor %q tier %q: %v", flavor, tier, err))
				}
			}
		}
	}
}

func validateModelPolicy(t table) {
	p := t.ModelValidation
	if p == nil {
		panic("models/defaults.json: modelValidation is missing")
	}
	if p.IDPattern == "" {
		panic("models/defaults.json: modelValidation.idPattern is empty")
	}
	idPattern, err := regexp.Compile(p.IDPattern)
	if err != nil {
		panic("models/defaults.json: modelValidation.idPattern: " + err.Error())
	}
	p.idPattern = idPattern
	if len(p.ForbiddenTokens) == 0 {
		panic("models/defaults.json: modelValidation.forbiddenTokens is empty")
	}
	tokens := map[string]bool{}
	for _, token := range p.ForbiddenTokens {
		if token == "" {
			panic("models/defaults.json: modelValidation.forbiddenTokens contains an empty token")
		}
		if tokens[token] {
			panic(fmt.Sprintf("models/defaults.json: duplicate forbidden token %q", token))
		}
		tokens[token] = true
	}
	if len(p.ForbiddenPatterns) == 0 {
		panic("models/defaults.json: modelValidation.forbiddenPatterns is empty")
	}
	names := map[string]bool{}
	for i := range p.ForbiddenPatterns {
		pattern := &p.ForbiddenPatterns[i]
		if pattern.Name == "" {
			panic("models/defaults.json: modelValidation.forbiddenPatterns contains an empty name")
		}
		if names[pattern.Name] {
			panic(fmt.Sprintf("models/defaults.json: duplicate forbidden pattern name %q", pattern.Name))
		}
		names[pattern.Name] = true
		if pattern.RE == "" {
			panic(fmt.Sprintf("models/defaults.json: forbidden pattern %q has an empty re", pattern.Name))
		}
		compiled, err := regexp.Compile(pattern.RE)
		if err != nil {
			panic(fmt.Sprintf("models/defaults.json: forbidden pattern %q: %v", pattern.Name, err))
		}
		pattern.compiled = compiled
	}
}

func validateShippedModelIDs(t table, flavor, tier string, entry tableEntry) {
	p := t.ModelValidation
	if !p.idPattern.MatchString(entry.ID) {
		panic(fmt.Sprintf("models/defaults.json: flavor %q tier %q current id %q fails modelValidation.idPattern", flavor, tier, entry.ID))
	}
	if rule := deniedModelRule(p, entry.ID); rule != "" {
		panic(fmt.Sprintf("models/defaults.json: flavor %q tier %q current id %q matches deny rule %q", flavor, tier, entry.ID, rule))
	}
	for _, h := range entry.History {
		if !p.idPattern.MatchString(h.ID) {
			panic(fmt.Sprintf("models/defaults.json: flavor %q tier %q historical id %q fails modelValidation.idPattern", flavor, tier, h.ID))
		}
	}
	tokens := map[string]bool{}
	for _, token := range p.ForbiddenTokens {
		tokens[token] = true
	}
	for _, alias := range entry.Aliases {
		if alias != entry.ID && !tokens[alias] {
			panic(fmt.Sprintf("models/defaults.json: flavor %q tier %q shorthand alias %q is absent from forbiddenTokens", flavor, tier, alias))
		}
	}
}

func deniedModelRule(p *modelValidationPolicy, id string) string {
	for _, token := range p.ForbiddenTokens {
		if id == token {
			return "token:" + token
		}
	}
	for _, pattern := range p.ForbiddenPatterns {
		if pattern.compiled.MatchString(id) {
			return pattern.Name
		}
	}
	return ""
}

// shippedEfforts lists every effort a shipped cell names — its own, its
// alternate's, and those of its history entries. An omitted effort is the
// tier default, which tierDefaultEffort validation already covers, so it is
// not repeated here.
func shippedEfforts(def tableEntry) []string {
	var efforts []string
	add := func(e string) {
		if e != "" {
			efforts = append(efforts, e)
		}
	}
	add(def.Effort)
	if def.Alternate != nil {
		add(def.Alternate.Effort)
	}
	for _, h := range def.History {
		add(h.Effort)
		if h.Alternate != nil {
			add(h.Alternate.Effort)
		}
	}
	return efforts
}

// checkEffort enforces a flavor's effort vocabulary (design: pi speaks
// low | medium | xhigh and has no "high"). A flavor absent from the
// vocabulary table accepts any effort — the pre-existing behavior for
// claude and codex, whose efforts spine never constrained. Translating an
// effort to a model's own reasoning aliases is the harness's job, not
// spine's; spine only refuses a word the harness does not have.
func checkEffort(t table, flavor, effort string) error {
	vocab, ok := t.EffortVocabulary[flavor]
	if !ok || effort == "" {
		return nil
	}
	for _, v := range vocab {
		if v == effort {
			return nil
		}
	}
	return fmt.Errorf("effort %q is not in the %s effort vocabulary (known: %s)", effort, flavor, strings.Join(vocab, ", "))
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

// HostStatus describes how the local capability file affected a resolution.
type HostStatus string

const (
	HostUnconfigured HostStatus = "unconfigured"
	HostConstrained  HostStatus = "constrained"
	HostPinned       HostStatus = "pinned"
)

// HostResolution records the local authority that constrained a final route.
// It contains no transport configuration or executable value.
type HostResolution struct {
	ID         string     `json:"id,omitempty"`
	Status     HostStatus `json:"status"`
	ConfigPath string     `json:"config_path,omitempty"`
}

// PinResolution is the safe, opaque provenance of a selected host pin.
type PinResolution struct {
	Model        string   `json:"model"`
	Effort       string   `json:"effort"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

// Resolution preserves the estate-to-repository-to-host trail. Entry is the
// final dispatchable route and Requested is the unmodified repository
// preference.
type Resolution struct {
	Entry     Entry          `json:"-"`
	Requested Entry          `json:"requested"`
	Host      HostResolution `json:"host"`
	Pin       *PinResolution `json:"pin,omitempty"`
}

// ResolveForHost applies a local host capability constraint after ordinary
// estate and repository preference resolution. configPath is an internal
// test seam when non-empty; production callers pass "" and use DefaultPath.
// lookup is normally exec.LookPath and is never used to execute anything.
func ResolveForHost(repoDir, configPath, flavor, tier string, lookup func(string) (string, error)) (Resolution, error) {
	requested, err := Resolve(repoDir, flavor, tier)
	if err != nil {
		return Resolution{}, err
	}
	return resolveForHost(requested, configPath, lookup)
}

// ValidateHostConfig checks a present local host configuration without
// selecting a harness route. It preserves the absent-file compatibility path
// for callers whose behavior is intentionally host-blind, such as legacy
// alternate selection.
func ValidateHostConfig(configPath string, lookup func(string) (string, error)) error {
	path, err := hostConfigPath(configPath)
	if err != nil {
		return err
	}
	_, err = hostconfig.Load(path, Flavors(), hostLookup(lookup))
	if errors.Is(err, hostconfig.ErrNotConfigured) {
		return nil
	}
	return err
}

func resolveForHost(requested Entry, configPath string, lookup func(string) (string, error)) (Resolution, error) {
	path, err := hostConfigPath(configPath)
	if err != nil {
		return Resolution{}, err
	}
	config, err := hostconfig.Load(path, Flavors(), hostLookup(lookup))
	if errors.Is(err, hostconfig.ErrNotConfigured) {
		return Resolution{Entry: requested, Requested: requested, Host: HostResolution{Status: HostUnconfigured, ConfigPath: path}}, nil
	}
	if err != nil {
		return Resolution{}, err
	}
	return applyHostConfig(requested, path, config)
}

func hostConfigPath(configPath string) (string, error) {
	if configPath != "" {
		return configPath, nil
	}
	return hostconfig.DefaultPath()
}

func hostLookup(lookup func(string) (string, error)) func(string) (string, error) {
	if lookup != nil {
		return lookup
	}
	return exec.LookPath
}

func applyHostConfig(requested Entry, path string, config hostconfig.Config) (Resolution, error) {
	harness, ok := config.Harnesses[requested.Flavor]
	if !ok || !harness.Available {
		return Resolution{}, fmt.Errorf("host routing configuration %q: harness %q is unavailable", path, requested.Flavor)
	}
	trail := HostResolution{ID: config.HostID, Status: HostConstrained, ConfigPath: path}
	key := requested.Flavor + "." + requested.Tier
	if pin, pinned := config.Pins[key]; pinned {
		final := requested
		final.ID = pin.Model
		final.Effort = pin.Effort
		final.Aliases = nil
		trail.Status = HostPinned
		return Resolution{Entry: final, Requested: requested, Host: trail, Pin: &PinResolution{Model: pin.Model, Effort: pin.Effort, EvidenceRefs: append([]string(nil), pin.EvidenceRefs...)}}, nil
	}
	route, reachable := harness.Models[requested.ID]
	if !reachable || !containsString(route.Efforts, requested.Effort) {
		return Resolution{}, fmt.Errorf("host routing configuration %q: requested %s.%s route %q @ %q is not reachable", path, requested.Flavor, requested.Tier, requested.ID, requested.Effort)
	}
	return Resolution{Entry: requested, Requested: requested, Host: trail}, nil
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// LaunchReason is a stable policy-refusal token returned by ValidateLaunch.
// Repository configuration and invocation errors are ordinary errors instead.
type LaunchReason string

const (
	ReasonForbiddenModel   LaunchReason = "forbidden-model"
	ReasonInvalidModelID   LaunchReason = "invalid-model-id"
	ReasonRetiredModel     LaunchReason = "retired-model"
	ReasonRouteMismatch    LaunchReason = "route-mismatch"
	ReasonUnmappedDispatch LaunchReason = "unmapped-dispatch"
)

// LaunchRequest identifies one active route and, optionally, the exact model
// ID a caller intends to pass to a launcher.
type LaunchRequest struct {
	RepoDir            string
	Flavor             string
	Tier               string
	Expected           string
	MaxTemplateVersion int
}

// LaunchRefusal is a well-formed request refused by model launch policy.
// Detail carries the other active route key for route-mismatch.
type LaunchRefusal struct {
	Reason LaunchReason
	Key    string
	Value  string
	Rule   string
	Detail string
}

func (e *LaunchRefusal) Error() string {
	if e == nil {
		return ""
	}
	message := fmt.Sprintf("%s: %s %q", e.Reason, e.Key, e.Value)
	if e.Rule != "" {
		message += fmt.Sprintf(" (rule: %s)", e.Rule)
	}
	if e.Detail != "" {
		message += fmt.Sprintf(" (%s)", e.Detail)
	}
	return message
}

// ActiveIDMatches is the single launch-safe active-ID predicate. It is
// intentionally byte-exact: aliases and history belong only to audit
// evidence, and callers may not normalize an ID into an active route.
func ActiveIDMatches(activeID, candidate string) bool {
	return activeID == candidate
}

// ValidateHostPinForLaunch is the I072 integration seam that preserves
// I051's same-policy validation-to-audit invariant. Until I074 teaches audit
// to recognize host-conformant active IDs, controlled launch validation may
// accept only a host pin whose model ID is byte-identical to the repository
// active ID. This function parses no host configuration and applies no host
// precedence itself.
func ValidateHostPinForLaunch(key, repositoryActiveID, hostPinnedID string) error {
	if ActiveIDMatches(repositoryActiveID, hostPinnedID) {
		return nil
	}
	return fmt.Errorf(
		"%s host pin %q differs from repository active ID %q and is not auditable until I074",
		key, hostPinnedID, repositoryActiveID,
	)
}

type launchSnapshot struct {
	rows          map[string][]string
	malformedRows []string
}

// ValidateLaunch resolves and validates one route from one WORKFLOW.md read.
func ValidateLaunch(req LaunchRequest) (Entry, error) {
	if _, ok := defaults.Flavors[req.Flavor]; !ok {
		return Entry{}, fmt.Errorf("unknown flavor %q (known: %s)", req.Flavor, strings.Join(flavorsOf(defaults), ", "))
	}
	if !isKnownTier(req.Tier) {
		return Entry{}, fmt.Errorf("unknown tier %q (known: %s)", req.Tier, strings.Join(Tiers, ", "))
	}
	snap, err := readLaunchSnapshot(req.RepoDir, req.MaxTemplateVersion)
	if err != nil {
		return Entry{}, err
	}
	return validateLaunchFrom(defaults, snap, req.Flavor, req.Tier, req.Expected)
}

// ValidateLaunchForHost performs I051 validation and local host inspection
// from one strict repository snapshot. A divergent host pin is deliberately
// non-launchable until I074 makes that final ID auditable.
func ValidateLaunchForHost(req LaunchRequest, configPath string, lookup func(string) (string, error)) (Resolution, error) {
	if _, ok := defaults.Flavors[req.Flavor]; !ok {
		return Resolution{}, fmt.Errorf("unknown flavor %q (known: %s)", req.Flavor, strings.Join(flavorsOf(defaults), ", "))
	}
	if !isKnownTier(req.Tier) {
		return Resolution{}, fmt.Errorf("unknown tier %q (known: %s)", req.Tier, strings.Join(Tiers, ", "))
	}
	snap, err := readLaunchSnapshot(req.RepoDir, req.MaxTemplateVersion)
	if err != nil {
		return Resolution{}, err
	}
	requested, err := validateLaunchFrom(defaults, snap, req.Flavor, req.Tier, "")
	if err != nil {
		return Resolution{}, err
	}
	path, err := hostConfigPath(configPath)
	if err != nil {
		return Resolution{}, err
	}
	config, err := hostconfig.Load(path, Flavors(), hostLookup(lookup))
	if errors.Is(err, hostconfig.ErrNotConfigured) {
		entry, err := validateLaunchFrom(defaults, snap, req.Flavor, req.Tier, req.Expected)
		if err != nil {
			return Resolution{}, err
		}
		return Resolution{Entry: entry, Requested: entry, Host: HostResolution{Status: HostUnconfigured, ConfigPath: path}}, nil
	}
	if err != nil {
		return Resolution{}, err
	}
	resolution, err := applyHostConfig(requested, path, config)
	if err != nil {
		return Resolution{}, err
	}
	if resolution.Pin != nil {
		if refusal := modelPolicyRefusal(defaults.ModelValidation, req.Flavor+"."+req.Tier, resolution.Pin.Model); refusal != nil {
			return Resolution{}, refusal
		}
		if err := ValidateHostPinForLaunch(req.Flavor+"."+req.Tier, requested.ID, resolution.Pin.Model); err != nil {
			return Resolution{}, err
		}
	}
	if _, err := validateLaunchFrom(defaults, snap, req.Flavor, req.Tier, req.Expected); err != nil {
		return Resolution{}, err
	}
	return resolution, nil
}

// ResolveStrictActive resolves one audit active row with the same strict
// model_routing selection used by ValidateLaunch. It deliberately does not
// apply launch-only deny or retired-ID policy; audit layers retained aliases
// and history over this exact active row. Audit keeps its existing template
// version and unreadable-file compatibility behavior.
func ResolveStrictActive(repoDir, flavor, tier string) (Entry, error) {
	if _, ok := defaults.Flavors[flavor]; !ok {
		return Entry{}, fmt.Errorf("unknown flavor %q (known: %s)", flavor, strings.Join(flavorsOf(defaults), ", "))
	}
	if !isKnownTier(tier) {
		return Entry{}, fmt.Errorf("unknown tier %q (known: %s)", tier, strings.Join(Tiers, ", "))
	}
	snap := launchSnapshot{rows: map[string][]string{}}
	if repoDir != "" {
		raw, err := os.ReadFile(filepath.Join(repoDir, "WORKFLOW.md"))
		if err == nil {
			snap, err = parseRoutingSnapshot(string(raw), 0, false)
			if err != nil {
				return Entry{}, err
			}
		}
	}
	entry, _, err := resolveStrictCell(defaults, snap, flavor, tier)
	return entry, err
}

func readLaunchSnapshot(repoDir string, maxTemplateVersion int) (launchSnapshot, error) {
	if repoDir == "" {
		return launchSnapshot{rows: map[string][]string{}}, nil
	}
	workflowPath := filepath.Join(repoDir, "WORKFLOW.md")
	raw, err := os.ReadFile(workflowPath)
	if err != nil {
		if os.IsNotExist(err) {
			return launchSnapshot{rows: map[string][]string{}}, nil
		}
		var pathErr *os.PathError
		if errors.As(err, &pathErr) {
			return launchSnapshot{}, fmt.Errorf("read WORKFLOW.md at %q: %w", pathErr.Path, pathErr.Err)
		}
		return launchSnapshot{}, fmt.Errorf("read WORKFLOW.md at %q: %w", workflowPath, err)
	}
	return parseLaunchRouting(string(raw), maxTemplateVersion)
}

func parseLaunchRouting(content string, maxTemplateVersion int) (launchSnapshot, error) {
	return parseRoutingSnapshot(content, maxTemplateVersion, true)
}

func parseRoutingSnapshot(content string, maxTemplateVersion int, validateTemplateVersion bool) (launchSnapshot, error) {
	snap := launchSnapshot{rows: map[string][]string{}}
	lines := strings.Split(content, "\n")
	versionCount := 0
	routingCount := 0
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if validateTemplateVersion && strings.HasPrefix(line, "template_version:") {
			versionCount++
			if versionCount > 1 {
				return launchSnapshot{}, fmt.Errorf("WORKFLOW.md has duplicate template_version")
			}
			value := strings.TrimSpace(strings.TrimPrefix(line, "template_version:"))
			if comment := CommentIndex(value); comment >= 0 {
				value = strings.TrimSpace(value[:comment])
			}
			if value == "" || !regexp.MustCompile(`^[0-9]+$`).MatchString(value) {
				return launchSnapshot{}, fmt.Errorf("WORKFLOW.md template_version %q is not one decimal integer", value)
			}
			version, err := strconv.Atoi(value)
			if err != nil {
				return launchSnapshot{}, fmt.Errorf("WORKFLOW.md template_version %q: %w", value, err)
			}
			if version > maxTemplateVersion {
				return launchSnapshot{}, fmt.Errorf("WORKFLOW.md template_version %d is newer than supported generation %d", version, maxTemplateVersion)
			}
		}

		if !strings.HasPrefix(line, "model_routing:") {
			continue
		}
		if !launchRoutingHeader(line) {
			return launchSnapshot{}, fmt.Errorf("WORKFLOW.md has malformed model_routing header %q", line)
		}
		routingCount++
		if routingCount > 1 {
			return launchSnapshot{}, fmt.Errorf("WORKFLOW.md has multiple model_routing blocks")
		}
		for i++; i < len(lines); i++ {
			row := lines[i]
			if !strings.HasPrefix(row, "  ") || strings.TrimSpace(row) == "" {
				i--
				break
			}
			body := strings.TrimPrefix(row, "  ")
			rawKey, value, ok := strings.Cut(body, ":")
			if !ok {
				snap.malformedRows = append(snap.malformedRows, body)
				continue
			}
			key := strings.TrimSpace(rawKey)
			if rawKey != key {
				snap.malformedRows = append(snap.malformedRows, body)
				continue
			}
			snap.rows[key] = append(snap.rows[key], value)
		}
	}
	return snap, nil
}

func launchRoutingHeader(line string) bool {
	if !strings.HasPrefix(line, "model_routing:") {
		return false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, "model_routing:"))
	return rest == "" || strings.HasPrefix(rest, "#")
}

func parseLaunchValue(value string) (override, error) {
	if comment := CommentIndex(value); comment >= 0 {
		value = value[:comment]
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return override{}, fmt.Errorf("empty model id")
	}
	if strings.Count(value, " alt:") > 1 {
		return override{}, fmt.Errorf("repeated alternate clause")
	}
	head, alt, hasAlt := strings.Cut(value, " alt:")
	id, effort, err := parseLaunchPair(head)
	if err != nil {
		return override{}, err
	}
	ov := override{id: id, effort: effort}
	if hasAlt {
		altID, altEffort, err := parseLaunchPair(alt)
		if err != nil {
			return override{}, fmt.Errorf("alternate: %w", err)
		}
		ov.alternate = &Alternate{ID: altID, Effort: altEffort}
	}
	return ov, nil
}

func parseLaunchPair(value string) (string, string, error) {
	if strings.Count(value, "@") > 1 {
		return "", "", fmt.Errorf("repeated effort separator")
	}
	id, effort, hasEffort := strings.Cut(value, "@")
	id = strings.TrimSpace(id)
	if id == "" {
		return "", "", fmt.Errorf("empty model id")
	}
	if len(strings.Fields(id)) != 1 {
		return "", "", fmt.Errorf("multiple model ids in %q", id)
	}
	if hasEffort {
		effort = strings.TrimSpace(effort)
		if effort == "" || len(strings.Fields(effort)) != 1 {
			return "", "", fmt.Errorf("malformed effort %q", effort)
		}
	}
	return id, effort, nil
}

func validateLaunchFrom(t table, snap launchSnapshot, flavor, tier, expected string) (Entry, error) {
	entry, err := resolveLaunchCell(t, snap, flavor, tier)
	if err != nil {
		return Entry{}, err
	}
	if expected == "" || ActiveIDMatches(entry.ID, expected) {
		return entry, nil
	}
	key := flavor + "." + tier
	if refusal := modelPolicyRefusal(t.ModelValidation, key, expected); refusal != nil {
		return Entry{}, refusal
	}
	for _, otherTier := range Tiers {
		if otherTier == tier {
			continue
		}
		other, err := resolveLaunchCell(t, snap, flavor, otherTier)
		if err == nil && ActiveIDMatches(other.ID, expected) {
			return Entry{}, &LaunchRefusal{Reason: ReasonRouteMismatch, Key: key, Value: expected, Detail: flavor + "." + otherTier}
		}
	}
	for _, candidateTier := range Tiers {
		for _, historical := range t.Flavors[flavor][candidateTier].History {
			if historical.ID == expected {
				return Entry{}, &LaunchRefusal{Reason: ReasonRetiredModel, Key: key, Value: expected}
			}
		}
	}
	return Entry{}, &LaunchRefusal{Reason: ReasonUnmappedDispatch, Key: key, Value: expected}
}

func resolveLaunchCell(t table, snap launchSnapshot, flavor, tier string) (Entry, error) {
	entry, selected, err := resolveStrictCell(t, snap, flavor, tier)
	if err != nil {
		return Entry{}, err
	}
	key := flavor + "." + tier
	if selected && !flavorHasCurrentID(t, flavor, entry.ID) && flavorHasHistoricalID(t, flavor, entry.ID) {
		return Entry{}, &LaunchRefusal{Reason: ReasonRetiredModel, Key: key, Value: entry.ID}
	}
	if refusal := modelPolicyRefusal(t.ModelValidation, key, entry.ID); refusal != nil {
		return Entry{}, refusal
	}
	return entry, nil
}

func resolveStrictCell(t table, snap launchSnapshot, flavor, tier string) (Entry, bool, error) {
	tiers, ok := t.Flavors[flavor]
	if !ok {
		return Entry{}, false, fmt.Errorf("unknown flavor %q (known: %s)", flavor, strings.Join(flavorsOf(t), ", "))
	}
	if !isKnownTier(tier) {
		return Entry{}, false, fmt.Errorf("unknown tier %q (known: %s)", tier, strings.Join(Tiers, ", "))
	}
	def := tiers[tier]
	key := flavor + "." + tier
	value, found, err := selectedLaunchValue(snap, flavor, tier)
	if err != nil {
		return Entry{}, false, err
	}
	entry := Entry{
		Flavor: flavor, Tier: tier, ID: def.ID, Effort: def.Effort,
		Aliases: def.Aliases, Alternate: def.Alternate, Provenance: Default,
	}
	if found {
		ov, err := parseLaunchValue(value)
		if err != nil {
			return Entry{}, false, fmt.Errorf("%s: %w", key, err)
		}
		entry.ID = ov.id
		entry.Effort = ov.effort
		entry.Alternate = ov.alternate
		if everShipped(def, tierDefaultEffortOf(t, flavor, tier), ov) {
			entry.Provenance = Inherited
		} else {
			entry.Provenance = Override
			entry.Aliases = nil
		}
	}
	if entry.Effort == "" {
		entry.Effort = tierDefaultEffortOf(t, flavor, tier)
	}
	if entry.Alternate != nil && entry.Alternate.Effort == "" {
		entry.Alternate = &Alternate{ID: entry.Alternate.ID, Effort: entry.Effort}
	}
	if err := checkEffort(t, flavor, entry.Effort); err != nil {
		return Entry{}, false, fmt.Errorf("%s: %w", key, err)
	}
	if entry.Alternate != nil {
		if err := checkEffort(t, flavor, entry.Alternate.Effort); err != nil {
			return Entry{}, false, fmt.Errorf("%s alternate: %w", key, err)
		}
	}
	return entry, found, nil
}

func selectedLaunchValue(snap launchSnapshot, flavor, tier string) (string, bool, error) {
	dotted := flavor + "." + tier
	for _, malformed := range snap.malformedRows {
		if malformedLaunchRowMatches(malformed, dotted) {
			return "", false, fmt.Errorf("malformed model_routing row %q for key %q", malformed, dotted)
		}
	}
	if len(snap.rows[dotted]) > 1 {
		return "", false, fmt.Errorf("duplicate model_routing key %q", dotted)
	}
	if len(snap.rows[dotted]) == 1 {
		return snap.rows[dotted][0], true, nil
	}
	if flavor == "claude" {
		for _, malformed := range snap.malformedRows {
			if malformedLaunchRowMatches(malformed, tier) {
				return "", false, fmt.Errorf("malformed legacy model_routing row %q for key %q", malformed, tier)
			}
		}
		if len(snap.rows[tier]) > 1 {
			return "", false, fmt.Errorf("duplicate legacy model_routing key %q", tier)
		}
	}
	if flavor == "claude" && len(snap.rows[tier]) == 1 {
		return snap.rows[tier][0], true, nil
	}
	return "", false, nil
}

func malformedLaunchRowMatches(raw, key string) bool {
	if rawKey, _, ok := strings.Cut(raw, ":"); ok {
		return strings.TrimSpace(rawKey) == key
	}
	trimmed := strings.TrimSpace(raw)
	return trimmed == key || strings.HasPrefix(trimmed, key+" ") || strings.HasPrefix(trimmed, key+"\t")
}

func flavorHasCurrentID(t table, flavor, id string) bool {
	for _, tier := range Tiers {
		if t.Flavors[flavor][tier].ID == id {
			return true
		}
	}
	return false
}

func flavorHasHistoricalID(t table, flavor, id string) bool {
	for _, tier := range Tiers {
		for _, historical := range t.Flavors[flavor][tier].History {
			if historical.ID == id {
				return true
			}
		}
	}
	return false
}

func modelPolicyRefusal(policy *modelValidationPolicy, key, value string) *LaunchRefusal {
	if !policy.idPattern.MatchString(value) {
		return &LaunchRefusal{Reason: ReasonInvalidModelID, Key: key, Value: value}
	}
	if rule := deniedModelRule(policy, value); rule != "" {
		return &LaunchRefusal{Reason: ReasonForbiddenModel, Key: key, Value: value, Rule: rule}
	}
	return nil
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
	tierDefaultEffort := tierDefaultEffortOf(t, flavor, tier)

	entry := Entry{
		Flavor: flavor, Tier: tier,
		ID: def.ID, Effort: def.Effort, Aliases: def.Aliases,
		Alternate:  def.Alternate,
		Provenance: Default,
	}

	if ov, found := readOverride(repoDir, flavor, tier); found {
		entry.ID = ov.id
		entry.Effort = ov.effort
		entry.Alternate = ov.alternate
		if everShipped(def, tierDefaultEffort, ov) {
			entry.Provenance = Inherited
		} else {
			entry.Provenance = Override
			// A deliberate override means exactly its on-disk id (I037 fix
			// round 1): the table entry's aliases describe the shipped
			// defaults, not the owner's pin. Carrying them over would let a
			// dispatch on the displaced default id read as this entry
			// downstream — the routing audit judges through these, and in a
			// repo that pinned something else, a default-id dispatch is the
			// drift the override exists to make visible, not a match.
			// Default and Inherited entries keep their aliases: same tier
			// lineage, and real pre-sweep repos (inherited claude-opus-4-8)
			// must still match the current default's alias tokens.
			entry.Aliases = nil
		}
	}

	if entry.Effort == "" {
		entry.Effort = tierDefaultEffort
	}
	// An alternate that names no effort of its own runs at the cell's own
	// effort — the alternate is then purely a different model, not a
	// different effort. (Shipped defaults always name one; this covers a
	// hand-written mirror row that omits it.)
	if entry.Alternate != nil && entry.Alternate.Effort == "" {
		entry.Alternate = &Alternate{ID: entry.Alternate.ID, Effort: entry.Effort}
	}
	// The vocabulary check runs on the resolved pair, so a per-repo override
	// asking pi for "high" fails here — the only place an effort outside a
	// flavor's vocabulary can still enter after load-time validation.
	if err := checkEffort(t, flavor, entry.Effort); err != nil {
		return Entry{}, fmt.Errorf("%s.%s: %w", flavor, tier, err)
	}
	if entry.Alternate != nil {
		if err := checkEffort(t, flavor, entry.Alternate.Effort); err != nil {
			return Entry{}, fmt.Errorf("%s.%s alternate: %w", flavor, tier, err)
		}
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
// The alternate is part of the pair (I079): a repo that edited only the
// alternate half of a cell — or deleted it — made a deliberate choice, and
// reporting that as Inherited would let the refresh rule silently revert it.
func everShipped(def tableEntry, tierDefaultEffort string, ov override) bool {
	effective := ov.effort
	if effective == "" {
		effective = tierDefaultEffort
	}
	// An alternate's effort defaults to the cell's own effective effort, the
	// same rule resolveFrom applies, so both sides compare as resolved.
	altKey := func(alt *Alternate, cellEffort string) string {
		if alt == nil {
			return ""
		}
		effort := alt.Effort
		if effort == "" {
			effort = cellEffort
		}
		return alt.ID + " @ " + effort
	}
	onDisk := altKey(ov.alternate, effective)
	matches := func(shippedID, shippedEffort string, shippedAlt *Alternate) bool {
		if shippedEffort == "" {
			shippedEffort = tierDefaultEffort
		}
		return ov.id == shippedID && effective == shippedEffort &&
			onDisk == altKey(shippedAlt, shippedEffort)
	}
	if matches(def.ID, def.Effort, def.Alternate) {
		return true
	}
	for _, h := range def.History {
		if matches(h.ID, h.Effort, h.Alternate) {
			return true
		}
	}
	return false
}

// tierDefaultEffortOf is the effort (flavor, tier) resolves to when a cell
// or a mirror row omits one (design D3): the flavor's own override first,
// the global map second (I079 fix round 1). Every effort comparison —
// resolution, everShipped, the update path's refresh check — goes through
// here, so a harness with its own vocabulary never inherits a word it
// cannot speak.
func tierDefaultEffortOf(t table, flavor, tier string) string {
	if e := t.TierDefaultEffortByFlavor[flavor][tier]; e != "" {
		return e
	}
	return t.TierDefaultEffort[tier]
}

// MirrorValue renders a resolved entry as the value half of a gen-10 mirror
// row (design D9): the model id, followed by " @ <effort>" only when the
// entry's effective effort deviates from its tier's default. An effort equal
// to the tier default is omitted so the common case stays a bare id and the
// suffix always signals a real deviation.
// The threshold here is deliberately the GLOBAL tier default, not a
// flavor's own override (I079 fix round 1): the mirror is read by humans who
// cannot see a per-harness default table, so a pi row rendered bare would
// leave "what effort does this actually run at" unanswerable on the page.
// Rendering is an economy, not a comparison — resolution, everShipped, and
// the refresh check all consult the flavor-scoped default, and both
// spellings of a pi row (bare id and "@ xhigh") resolve to the same pair.
// A cell with an alternate appends " alt: <id> @ <effort>" on the same line
// (I079). The alternate's effort is always spelled out — it is a deliberate
// tuning knob, and a bare id there would read as "same effort as the cell"
// rather than as the owner's choice.
func MirrorValue(e Entry) string {
	v := e.ID
	if e.Effort != "" && e.Effort != defaults.TierDefaultEffort[e.Tier] {
		v += " @ " + e.Effort
	}
	if e.Alternate != nil {
		effort := e.Alternate.Effort
		if effort == "" {
			effort = e.Effort
		}
		v += " alt: " + e.Alternate.ID + " @ " + effort
	}
	return v
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
			e := Entry{Flavor: flavor, Tier: tier, ID: def.ID, Effort: def.Effort, Alternate: def.Alternate}
			if e.Effort == "" {
				e.Effort = tierDefaultEffortOf(defaults, flavor, tier)
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
	id        string
	effort    string
	alternate *Alternate
}

// readOverride looks up repoDir's WORKFLOW.md model_routing block for the
// dotted "<flavor>.<tier>" key design D8 defines for the gen-10 mirror,
// falling back to the bare "<tier>" key for the claude flavor — the
// TRANSITIONAL gen ≤9 affordance described in the package comment (I035):
// claude is the only flavor any gen ≤9 mirror ever rendered, and bare keys
// stay invisible to every other flavor. The EMITTED format is retired in
// gen-10 mirrors (I036), but the READ affordance stays live for gen-9
// worktrees until they merge/rebase. A dotted key wins over a bare one.
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

// parseValue splits one mirror value into id, optional effort, and optional
// alternate per D9 plus I079: "<id>[ @ <effort>][ alt: <id>[ @ <effort>]]".
// RoutingKeys has already stripped any trailing comment — D9's corrected
// order (comment first, then the separator split). The alternate clause is
// cut off first so the effort split never sees the alternate's own " @ ".
func parseValue(v string) override {
	var ov override
	head, alt, hasAlt := strings.Cut(v, " alt:")
	if hasAlt {
		altID, altEffort, _ := strings.Cut(alt, "@")
		if id := strings.TrimSpace(altID); id != "" {
			ov.alternate = &Alternate{ID: id, Effort: strings.TrimSpace(altEffort)}
		}
	}
	id, effort, hasEffort := strings.Cut(head, "@")
	ov.id = strings.TrimSpace(id)
	if hasEffort {
		ov.effort = strings.TrimSpace(effort)
	}
	return ov
}
