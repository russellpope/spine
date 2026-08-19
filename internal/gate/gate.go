// Package gate implements spine's gate pack: a spine-authored,
// independently versioned battery of deterministic check classes for one
// language, executed by maipipe as the enforcement floor (ADR 0015).
//
// This file holds the pack-wide seams every check class shares: the pack
// version constant, the finding type, the check registry, and Run — the one
// entry point the CLI calls, which owns the exit-code contract (0 pass,
// 1 findings, 2 misconfiguration) and hands results to the emitter.
//
// Adding a check class is: write a Check func in its own file, register it
// in the map below, and ship its positive control pair (a known-good input
// the check passes and a seeded violation it fails). A class that owns its
// own stage judgement (only the advisory mutation battery) returns a Report
// and registers in reportChecks instead.
package gate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Pack identity. A finding is attributable as "<pack>@<version>/<check>";
// the results contract's code field carries that string. The pack version
// is independent of the template generation (ADR 0015) and lives here only.
const (
	PackName    = "go"
	PackVersion = 1
)

// PackID is the versioned pack identifier, e.g. "go@1".
func PackID() string { return fmt.Sprintf("%s@%d", PackName, PackVersion) }

// Code is the results-contract code for a finding from one check class,
// e.g. "go@1/tskip".
func Code(check string) string { return PackID() + "/" + check }

// Severity strings used in the results contract.
const SeverityError = "error"

// SeverityWarning and SeverityInfo are the other two values of maipipe's
// finding-severity vocabulary (error | warning | info — I092: "warn" is
// rejected as an invalid v0 document, failing the stage as results_invalid).
const (
	SeverityWarning = "warning"
	SeverityInfo    = "info"
)

// A Finding is one attributable problem found by a check class. Its field
// set is exactly the results contract's finding keys.
type Finding struct {
	Severity string
	Message  string
	File     string
	Line     int
	Code     string
}

// Config is a check class's read-only view of its configuration. Per-check
// inputs are declared once in WORKFLOW.md's gate_pack_config and reach a
// stage as environment variables named SPINE_GATE_ + the upper-snake of the
// config key (gate_pack_config.tskip_allow -> SPINE_GATE_TSKIP_ALLOW).
type Config struct {
	lookup func(string) (string, bool)
}

// EnvConfig reads configuration from the process environment.
func EnvConfig() Config { return Config{lookup: os.LookupEnv} }

// EnvVar is the environment variable name for a gate_pack_config key. It is
// exported so misconfiguration messages can name the variable the operator
// has to set.
func EnvVar(key string) string {
	return "SPINE_GATE_" + strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
}

// Get returns the value configured for a gate_pack_config key, and whether
// it was set at all. An unset key is never in itself a misconfiguration —
// only a check that requires the key may say so.
func (c Config) Get(key string) (string, bool) {
	if c.lookup == nil {
		return "", false
	}
	v, ok := c.lookup(EnvVar(key))
	return v, ok
}

// env reads an environment variable by its literal name, for the few
// tuning knobs that are env-only and so have no gate_pack_config key —
// SPINE_GATE_CLEANUP_FUNCS. It shares Config's lookup so a check class
// never reaches around the configuration seam to the process environment.
func (c Config) env(name string) string {
	if c.lookup == nil {
		return ""
	}
	v, _ := c.lookup(name)
	return v
}

// A Check is one check class: it inspects dir and returns its findings.
// A returned error is a misconfiguration (exit 2), not a finding.
type Check func(dir string, cfg Config) ([]Finding, error)

// A Report is a check class's whole outcome when the class owns its own
// stage-level judgement rather than taking the pack default (findings mean
// fail, summary is a finding count). Only the mutation battery needs this
// seam: it is the pack's advisory lane, so its rows never fail the stage,
// and its summary is the two kill rates the checklist prescribes.
type Report struct {
	Findings []Finding
	// Summary replaces the emitter's default summary line when non-empty.
	Summary string
	// Detail is human-only trailer printed under the table (the survivor
	// list); it never reaches the results contract, whose finding rows
	// already carry every row's outcome.
	Detail []string
	// Advisory means findings are reported but do not fail the stage:
	// status stays "pass" and the exit code stays 0.
	Advisory bool
}

// A ReportCheck is a check class returning a Report instead of bare
// findings. See Report for why the seam exists.
type ReportCheck func(dir string, cfg Config) (Report, error)

// checks is the check-class registry for the go pack: name -> implementation.
var checks = map[string]Check{
	"tskip":                     checkTskip,
	"binary-hygiene":            checkBinaryHygiene,
	"gitignore-control":         checkGitignoreControl,
	"fixture-manifest":          checkFixtureManifest,
	"test-enum-vs-spec":         checkTestEnumVsSpec,
	"deferred-cleanup-errcheck": checkDeferredCleanup,
	"dead-code-callgraph":       checkDeadCode,
	"n-plus-one":                checkNPlusOne,
}

// reportChecks is the registry for check classes that own their stage-level
// judgement. A name lives in exactly one of the two registries.
var reportChecks = map[string]ReportCheck{
	"mutate": checkMutate,
}

// CheckNames returns the registered check classes, sorted.
func CheckNames() []string {
	names := make([]string, 0, len(checks)+len(reportChecks))
	for n := range checks {
		names = append(names, n)
	}
	for n := range reportChecks {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Run executes one check class of one pack against dir and reports through
// the results contract. It is the single owner of the exit-code contract:
//
//	0 — pass (no findings)
//	1 — findings
//	2 — misconfiguration (unknown pack, unknown check, unusable --dir,
//	    missing or unreadable required config, unwritable results file)
//
// Misconfiguration messages go to stderr and name the problem (and the
// environment variable, when configuration is the problem).
func Run(pack, check, dir string, stdout, stderr io.Writer, cfg Config) int {
	if pack != PackName {
		fmt.Fprintf(stderr, "gate: unknown pack %q (known: %s)\n", pack, PackName)
		return 2
	}
	fn, plain := checks[check]
	rfn, rich := reportChecks[check]
	if !plain && !rich {
		fmt.Fprintf(stderr, "gate %s: unknown check %q (known: %s)\n", pack, check, strings.Join(CheckNames(), ", "))
		return 2
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(stderr, "gate %s %s: --dir %s: %v\n", pack, check, dir, err)
		return 2
	}
	info, err := os.Stat(abs)
	if err != nil {
		fmt.Fprintf(stderr, "gate %s %s: --dir %s: %v\n", pack, check, dir, err)
		return 2
	}
	if !info.IsDir() {
		fmt.Fprintf(stderr, "gate %s %s: --dir %s: not a directory\n", pack, check, dir)
		return 2
	}
	var (
		rep    Report
		runErr error
	)
	if plain {
		var findings []Finding
		findings, runErr = fn(abs, cfg)
		rep = Report{Findings: findings}
	} else {
		rep, runErr = rfn(abs, cfg)
	}
	if runErr != nil {
		fmt.Fprintf(stderr, "gate %s %s: %v\n", pack, check, runErr)
		return 2
	}
	sortFindings(rep.Findings)
	if err := emit(check, rep, stdout); err != nil {
		fmt.Fprintf(stderr, "gate %s %s: %v\n", pack, check, err)
		return 2
	}
	if len(rep.Findings) > 0 && !rep.Advisory {
		return 1
	}
	return 0
}

// sortFindings puts findings in a stable, deterministic order so the
// results contract is byte-comparable across runs.
func sortFindings(f []Finding) {
	sort.SliceStable(f, func(i, j int) bool {
		if f[i].File != f[j].File {
			return f[i].File < f[j].File
		}
		if f[i].Line != f[j].Line {
			return f[i].Line < f[j].Line
		}
		return f[i].Message < f[j].Message
	})
}
