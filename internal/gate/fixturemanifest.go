package gate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// fixtureManifestKey is the gate_pack_config key holding the fixture
// manifest path; it reaches the stage as SPINE_GATE_FIXTURE_MANIFEST.
const fixtureManifestKey = "fixture_manifest"

// checkFixtureManifest reports that the configured fixture manifest exists
// and is non-empty. Content judgment is the evaluator's and is never done
// here: the class only refuses the two states that make the manifest a
// fiction — absent, or present with nothing but whitespace in it.
//
// The manifest is the thing being checked, not an input to the check, so an
// absent or empty manifest is a finding (exit 1), not misconfiguration. An
// unset SPINE_GATE_FIXTURE_MANIFEST is misconfiguration (exit 2): without
// it there is nothing to check. So is a manifest that exists but cannot be
// read.
func checkFixtureManifest(dir string, cfg Config) ([]Finding, error) {
	rel, _ := cfg.Get(fixtureManifestKey)
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return nil, fmt.Errorf("%s is unset or empty: set it to the fixture manifest path relative to --dir", EnvVar(fixtureManifestKey))
	}
	path := resolveUnder(dir, rel)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Finding{{
				Severity: SeverityError,
				Message:  fmt.Sprintf("fixture manifest missing: %s does not exist (%s)", rel, EnvVar(fixtureManifestKey)),
				File:     rel,
				Line:     0,
				Code:     Code("fixture-manifest"),
			}}, nil
		}
		return nil, fmt.Errorf("%s: reading %s: %w", EnvVar(fixtureManifestKey), rel, err)
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return []Finding{{
			Severity: SeverityError,
			Message:  fmt.Sprintf("fixture manifest empty: %s has no content (%s)", rel, EnvVar(fixtureManifestKey)),
			File:     rel,
			Line:     0,
			Code:     Code("fixture-manifest"),
		}}, nil
	}
	return nil, nil
}

// resolveUnder turns a configured path into an absolute one: relative
// values are relative to --dir, absolute values are taken as given.
func resolveUnder(dir, rel string) string {
	p := filepath.FromSlash(rel)
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(dir, p)
}
