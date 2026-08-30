// Package hostconfig loads the owner- or fleet-managed local routing
// capability file. It intentionally contains no transport, credential, or
// command-execution behavior.
package hostconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ErrNotConfigured reports the compatible case where the local capability
// file does not exist. Callers keep their existing repository-only behavior.
var ErrNotConfigured = errors.New("host routing is not configured")

// Config is the closed schema-v1 local capability declaration.
type Config struct {
	SchemaVersion int
	HostID        string
	Harnesses     map[string]Harness
	Pins          map[string]Pin
}

type Harness struct {
	Available         bool
	Executable        string
	LaunchContractRef string
	Models            map[string]ModelRoute
}

type ModelRoute struct {
	Efforts     []string
	ObservedIDs []string
	GatewayRef  string
}

type Pin struct {
	Model        string
	Effort       string
	EvidenceRefs []string
}

type rawConfig struct {
	SchemaVersion *int                  `json:"schema_version"`
	HostID        *string               `json:"host_id"`
	Harnesses     map[string]rawHarness `json:"harnesses"`
	Pins          map[string]rawPin     `json:"pins"`
}

type rawHarness struct {
	Available         *bool                    `json:"available"`
	Executable        *string                  `json:"executable"`
	LaunchContractRef *string                  `json:"launch_contract_ref"`
	Models            map[string]rawModelRoute `json:"models"`
}

type rawModelRoute struct {
	Efforts     []string `json:"efforts"`
	ObservedIDs []string `json:"observed_ids"`
	GatewayRef  *string  `json:"gateway_ref"`
}

type rawPin struct {
	Model        *string  `json:"model"`
	Effort       *string  `json:"effort"`
	EvidenceRefs []string `json:"evidence_refs"`
}

// DefaultPath returns the only production path for the local file.
func DefaultPath() (string, error) { return defaultPath(os.UserConfigDir) }

func defaultPath(userConfigDir func() (string, error)) (string, error) {
	dir, err := userConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user configuration directory: %w", err)
	}
	if dir == "" {
		return "", fmt.Errorf("locate user configuration directory: empty path")
	}
	return filepath.Join(dir, "spine", "routing-host.json"), nil
}

// Load parses and validates path. lookup may only inspect executable
// availability; callers normally pass exec.LookPath. It is injected so tests
// have no mutable package-global path or executable seams.
func Load(path string, flavors []string, lookup func(string) (string, error)) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, ErrNotConfigured
		}
		return Config{}, configError(path, "read configuration")
	}
	config, err := decode(raw, flavors)
	if err != nil {
		return Config{}, configError(path, err.Error())
	}
	if err := validateExecutables(config, lookup); err != nil {
		return Config{}, configError(path, err.Error())
	}
	return config, nil
}

func configError(path, detail string) error {
	return fmt.Errorf("host routing configuration %q: %s", path, detail)
}

func decode(raw []byte, flavors []string) (Config, error) {
	if !utf8.Valid(raw) {
		return Config{}, fmt.Errorf("configuration is not valid UTF-8")
	}
	if err := rejectDuplicateJSONMembers(raw); err != nil {
		return Config{}, err
	}
	if err := validateClosedSchema(raw); err != nil {
		return Config{}, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var value rawConfig
	if err := dec.Decode(&value); err != nil {
		return Config{}, fmt.Errorf("invalid JSON schema")
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return Config{}, fmt.Errorf("multiple JSON values")
	}
	return validate(value, flavors)
}

func validate(raw rawConfig, flavors []string) (Config, error) {
	if raw.SchemaVersion == nil || *raw.SchemaVersion != 1 {
		return Config{}, fmt.Errorf("schema_version must be integer 1")
	}
	if raw.HostID == nil || !safeString(*raw.HostID) {
		return Config{}, fmt.Errorf("host_id must be a non-empty control-free string")
	}
	if len(raw.Harnesses) == 0 {
		return Config{}, fmt.Errorf("harnesses must be a non-empty object")
	}
	known := make(map[string]struct{}, len(flavors))
	for _, flavor := range flavors {
		known[flavor] = struct{}{}
	}
	config := Config{SchemaVersion: *raw.SchemaVersion, HostID: *raw.HostID, Harnesses: make(map[string]Harness, len(raw.Harnesses)), Pins: make(map[string]Pin, len(raw.Pins))}
	observed := map[string]struct{}{}
	for name, rawHarness := range raw.Harnesses {
		if _, ok := known[name]; !ok || !safeString(name) {
			return Config{}, fmt.Errorf("harness is not a current flavor")
		}
		if rawHarness.Available == nil || rawHarness.Executable == nil || rawHarness.LaunchContractRef == nil {
			return Config{}, fmt.Errorf("harness is missing a required member")
		}
		if !safeString(*rawHarness.Executable) || !safeString(*rawHarness.LaunchContractRef) {
			return Config{}, fmt.Errorf("harness has an empty or unsafe reference")
		}
		if *rawHarness.Available && len(rawHarness.Models) == 0 {
			return Config{}, fmt.Errorf("available harness must declare models")
		}
		h := Harness{Available: *rawHarness.Available, Executable: *rawHarness.Executable, LaunchContractRef: *rawHarness.LaunchContractRef, Models: make(map[string]ModelRoute, len(rawHarness.Models))}
		for modelID, rawRoute := range rawHarness.Models {
			if !safeString(modelID) || len(rawRoute.Efforts) == 0 {
				return Config{}, fmt.Errorf("route has an empty model or efforts")
			}
			route := ModelRoute{Efforts: append([]string(nil), rawRoute.Efforts...), ObservedIDs: append([]string(nil), rawRoute.ObservedIDs...)}
			if rawRoute.GatewayRef != nil {
				if !safeString(*rawRoute.GatewayRef) {
					return Config{}, fmt.Errorf("route has unsafe gateway_ref")
				}
				route.GatewayRef = *rawRoute.GatewayRef
			}
			if err := validateUniqueStrings(route.Efforts, "efforts"); err != nil {
				return Config{}, fmt.Errorf("route: %w", err)
			}
			if err := validateUniqueStrings(route.ObservedIDs, "observed_ids"); err != nil {
				return Config{}, fmt.Errorf("route: %w", err)
			}
			for _, id := range route.ObservedIDs {
				if _, duplicate := observed[id]; duplicate {
					return Config{}, fmt.Errorf("observed_id appears more than once")
				}
				observed[id] = struct{}{}
			}
			h.Models[modelID] = route
		}
		config.Harnesses[name] = h
	}
	for key, rawPin := range raw.Pins {
		flavor, tier, ok := strings.Cut(key, ".")
		if !ok || flavor == "" || tier == "" || strings.Contains(tier, ".") {
			return Config{}, fmt.Errorf("pin key must be flavor.tier")
		}
		if _, ok := known[flavor]; !ok || !knownTier(tier) {
			return Config{}, fmt.Errorf("pin key names an unknown flavor or tier")
		}
		if rawPin.Model == nil || rawPin.Effort == nil || !safeString(*rawPin.Model) || !safeString(*rawPin.Effort) {
			return Config{}, fmt.Errorf("pin has an empty or unsafe model@effort")
		}
		if err := validateUniqueStrings(rawPin.EvidenceRefs, "evidence_refs"); err != nil {
			return Config{}, fmt.Errorf("pin: %w", err)
		}
		harness, exists := config.Harnesses[flavor]
		if !exists || !harness.Available {
			return Config{}, fmt.Errorf("pin names an unavailable harness")
		}
		route, exists := harness.Models[*rawPin.Model]
		if !exists || !contains(route.Efforts, *rawPin.Effort) {
			return Config{}, fmt.Errorf("pin model@effort is not declared by its harness")
		}
		config.Pins[key] = Pin{Model: *rawPin.Model, Effort: *rawPin.Effort, EvidenceRefs: append([]string(nil), rawPin.EvidenceRefs...)}
	}
	return config, nil
}

func validateExecutables(config Config, lookup func(string) (string, error)) error {
	names := make([]string, 0, len(config.Harnesses))
	for name, harness := range config.Harnesses {
		if !harness.Available {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		harness := config.Harnesses[name]
		if _, err := lookup(harness.Executable); err != nil {
			return fmt.Errorf("available harness executable is not resolvable")
		}
	}
	return nil
}

func validateUniqueStrings(values []string, field string) error {
	seen := map[string]struct{}{}
	for _, value := range values {
		if !safeString(value) {
			return fmt.Errorf("%s has an empty or control-containing value", field)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf("%s has duplicate value", field)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func safeString(value string) bool {
	if value == "" {
		return false
	}
	return !strings.ContainsFunc(value, unicode.IsControl)
}

func knownTier(tier string) bool {
	return tier == "primary" || tier == "routine" || tier == "mechanical" || tier == "fallback"
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// validateClosedSchema rejects the standard library decoder's permissive
// case-insensitive struct-field matching before that decoder sees the input.
// It also distinguishes an absent optional field from a present JSON null.
func validateClosedSchema(raw []byte) error {
	root, err := schemaObject(raw, "root")
	if err != nil {
		return err
	}
	if err := schemaMembers(root, []string{"schema_version", "host_id", "harnesses", "pins"}, nil, "root"); err != nil {
		return err
	}
	harnesses, err := schemaObject(root["harnesses"], "harnesses")
	if err != nil {
		return err
	}
	pins, err := schemaObject(root["pins"], "pins")
	if err != nil {
		return err
	}
	for _, harnessRaw := range harnesses {
		harness, err := schemaObject(harnessRaw, "harness")
		if err != nil {
			return err
		}
		if err := schemaMembers(harness, []string{"available", "executable", "launch_contract_ref", "models"}, nil, "harness"); err != nil {
			return err
		}
		models, err := schemaObject(harness["models"], "models")
		if err != nil {
			return err
		}
		for _, routeRaw := range models {
			route, err := schemaObject(routeRaw, "model route")
			if err != nil {
				return err
			}
			if err := schemaMembers(route, []string{"efforts"}, []string{"observed_ids", "gateway_ref"}, "model route"); err != nil {
				return err
			}
		}
	}
	for _, pinRaw := range pins {
		pin, err := schemaObject(pinRaw, "pin")
		if err != nil {
			return err
		}
		if err := schemaMembers(pin, []string{"model", "effort"}, []string{"evidence_refs"}, "pin"); err != nil {
			return err
		}
	}
	return nil
}

func schemaObject(raw json.RawMessage, label string) (map[string]json.RawMessage, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, fmt.Errorf("%s must be an object", label)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, fmt.Errorf("%s must be an object", label)
	}
	return object, nil
}

func schemaMembers(object map[string]json.RawMessage, required, optional []string, label string) error {
	allowed := make(map[string]struct{}, len(required)+len(optional))
	for _, member := range required {
		allowed[member] = struct{}{}
		value, ok := object[member]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("%s is missing a typed member", label)
		}
	}
	for _, member := range optional {
		allowed[member] = struct{}{}
		if value, ok := object[member]; ok && bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("%s has a null typed member", label)
		}
	}
	for member := range object {
		if _, ok := allowed[member]; !ok {
			return fmt.Errorf("%s has an unsupported member", label)
		}
	}
	return nil
}

func rejectDuplicateJSONMembers(raw []byte) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := scanJSONValue(dec); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		return fmt.Errorf("multiple JSON values")
	}
	return nil
}

func scanJSONValue(dec *json.Decoder) error {
	token, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]struct{}{}
		for dec.More() {
			member, err := dec.Token()
			if err != nil {
				return err
			}
			key, ok := member.(string)
			if !ok {
				return fmt.Errorf("JSON object member name is not a string")
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("duplicate JSON member")
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
		return fmt.Errorf("unexpected JSON delimiter")
	}
}
