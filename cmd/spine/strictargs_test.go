package main

// I119: no spine subcommand silently discards input. The ordering sweep
// enumerates one trailing-flag invocation per converted subcommand and is
// deliberately the conversion checklist — a FlagSet site missing from it is
// a review finding. Every entry leads with --dir into a TempDir so a red
// run can never act on the real repo.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOrderingGuardSweepAllSubcommands(t *testing.T) {
	tmp := t.TempDir()
	cases := []struct {
		prefix string // the command's error prefix (its FlagSet name)
		args   []string
		tok    string // the flag-like token the error must name
	}{
		{"init", []string{"init", "--dir", tmp, "stray", "--name", "n"}, "--name"},
		{"update", []string{"update", "--dir", tmp, "stray", "--write"}, "--write"},
		{"adr new", []string{"adr", "new", "--dir", tmp, "T", "--supersedes", "3"}, "--supersedes"},
		{"adr list", []string{"adr", "list", "--dir", tmp, "stray", "--json"}, "--json"},
		{"handoff new", []string{"handoff", "new", "--dir", tmp, "T", "--json"}, "--json"},
		{"handoff list", []string{"handoff", "list", "--dir", tmp, "stray", "--json"}, "--json"},
		{"handoff latest", []string{"handoff", "latest", "--dir", tmp, "stray", "--json"}, "--json"},
		{"doctor", []string{"doctor", "--dir", tmp, "stray", "--json"}, "--json"},
		{"eval new", []string{"eval", "new", "--dir", tmp, "T", "--json"}, "--json"},
		{"eval add-run", []string{"eval", "add-run", "--dir", tmp, "stray", "--eval", "e"}, "--eval"},
		{"eval list", []string{"eval", "list", "--dir", tmp, "stray", "--json"}, "--json"},
		{"gate", []string{"gate", "--dir", tmp, "go@1", "tskip", "--nope"}, "--nope"},
		{"audit routing", []string{"audit", "routing", "--dir", tmp, "stray", "--since", "2026-01-01"}, "--since"},
		{"audit stages", []string{"audit", "stages", "--dir", tmp, "stray", "--json"}, "--json"},
		{"checkpoint new", []string{"checkpoint", "new", "--dir", tmp, "stray", "--gate", "pass"}, "--gate"},
		{"checkpoint latest", []string{"checkpoint", "latest", "--dir", tmp, "stray", "--json"}, "--json"},
		{"checkpoint list", []string{"checkpoint", "list", "--dir", tmp, "stray", "--json"}, "--json"},
		{"cursor", []string{"cursor", "--dir", tmp, "stray", "--quiet"}, "--quiet"},
		{"cursor start", []string{"cursor", "start", "--dir", tmp, "stray", "--effort", "e"}, "--effort"},
		{"cursor tick", []string{"cursor", "tick", "--dir", tmp, "grill", "--prd", "p"}, "--prd"},
		{"cursor here", []string{"cursor", "here", "--dir", tmp, "grill", "--json"}, "--json"},
		{"cursor set", []string{"cursor", "set", "--dir", tmp, "stray", "--prd", "p"}, "--prd"},
		{"model", []string{"model", "--dir", tmp, "claude", "primary", "--json"}, "--json"},
		{"adopt", []string{"adopt", "--dir", tmp, "stray", "--write"}, "--write"},
	}
	for _, tc := range cases {
		t.Run(tc.prefix, func(t *testing.T) {
			code, _, errs := runCmd(t, tc.args...)
			if code != 2 {
				t.Fatalf("code=%d stderr=%q", code, errs)
			}
			if !strings.Contains(errs, tc.prefix+": flags must precede positionals") {
				t.Fatalf("ordering rule not named with prefix %q: stderr=%q", tc.prefix, errs)
			}
			if !strings.Contains(errs, fmt.Sprintf("saw %q", tc.tok)) {
				t.Fatalf("offending token %q not named: stderr=%q", tc.tok, errs)
			}
		})
	}
}

func TestStrayPositionalSweep(t *testing.T) {
	tmp := t.TempDir()
	cases := []struct {
		prefix string
		args   []string
		tok    string // the unexpected positional the error must name
	}{
		{"doctor", []string{"doctor", "--dir", tmp, "junk"}, "junk"},
		{"update", []string{"update", "--dir", tmp, "junk"}, "junk"},
		{"cursor", []string{"cursor", "--dir", tmp, "junk"}, "junk"},
		{"cursor start", []string{"cursor", "start", "--dir", tmp, "--effort", "e", "junk"}, "junk"},
		{"adr new", []string{"adr", "new", "--dir", tmp, "T1", "T2"}, "T2"},
		{"gate", []string{"gate", "--dir", tmp, "go@1", "tskip", "extra"}, "extra"},
		{"model", []string{"model", "--dir", tmp, "claude", "primary", "extra"}, "extra"},
	}
	for _, tc := range cases {
		t.Run(tc.prefix, func(t *testing.T) {
			code, _, errs := runCmd(t, tc.args...)
			if code != 2 {
				t.Fatalf("code=%d stderr=%q", code, errs)
			}
			if !strings.Contains(errs, tc.prefix+": unexpected argument") {
				t.Fatalf("unexpected-argument rule not named with prefix %q: stderr=%q", tc.prefix, errs)
			}
			if !strings.Contains(errs, fmt.Sprintf("%q", tc.tok)) {
				t.Fatalf("stray token %q not named: stderr=%q", tc.tok, errs)
			}
		})
	}
}

// The live repro that filed I119: `spine cursor show --dir X` read the CWD
// repo and exited 0. An unknown cursor sub-subcommand now errors like every
// other dispatcher, naming the real verbs.
func TestCursorUnknownSubcommandErrors(t *testing.T) {
	code, out, errs := runCmd(t, "cursor", "show", "--dir", t.TempDir())
	if code != 2 {
		t.Fatalf("code=%d out=%q stderr=%q", code, out, errs)
	}
	if !strings.Contains(errs, `unknown cursor subcommand "show"`) {
		t.Fatalf("unknown subcommand not named: stderr=%q", errs)
	}
	for _, verb := range []string{"start", "tick", "here", "set"} {
		if !strings.Contains(errs, verb) {
			t.Fatalf("usage does not name verb %q: stderr=%q", verb, errs)
		}
	}
}

// Green control on the narrowed contract: flag-only cursor invocations stay
// exit-0-always, including the no-cursor case (hook consumers).
func TestCursorFlagOnlyInvocationKeepsExitZeroContract(t *testing.T) {
	code, out, errs := runCmd(t, "cursor", "--dir", t.TempDir())
	if code != 0 || !strings.Contains(out, "no spine cursor found") {
		t.Fatalf("code=%d out=%q stderr=%q", code, out, errs)
	}
}

// gate joins the house grammar: flags first is valid (an empty tree passes
// tskip), and the previously silently-working trailing form now names the
// rule (covered by the ordering sweep above).
func TestGateAcceptsFlagsFirstGrammar(t *testing.T) {
	code, out, errs := runCmd(t, "gate", "--dir", t.TempDir(), "go@1", "tskip")
	if code != 0 {
		t.Fatalf("flags-first gate run: code=%d out=%q stderr=%q", code, out, errs)
	}
}

// First-position exemption survives the generalization: a deliberate
// flag-like positional behind an explicit "--" reaches the command's own
// logic, never the ordering error.
func TestFirstPositionExemptionSurvivesGeneralization(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	code, out, errs := runCmd(t, "adr", "new", "--dir", dir, "--", "-Title")
	if strings.Contains(errs, "flags must precede positionals") {
		t.Fatalf("exemption lost: stderr=%q", errs)
	}
	if code != 0 || !strings.Contains(out, ".md") {
		t.Fatalf("adr new did not run: code=%d out=%q stderr=%q", code, out, errs)
	}
	// The model arm: a flag-like first positional behind "--" reaches
	// model.Resolve's own unknown-harness error, not the ordering message.
	code, _, errs = runCmd(t, "model", "--dir", dir, "--", "-weird", "primary")
	if strings.Contains(errs, "flags must precede positionals") {
		t.Fatalf("model exemption lost: stderr=%q", errs)
	}
	if code != 2 || !strings.Contains(errs, "unknown harness") {
		t.Fatalf("model -- form did not reach Resolve: code=%d stderr=%q", code, errs)
	}
}

// version stays lenient (Q11): informational output, no wrong answer
// possible, so stray input keeps being ignored.
func TestVersionStaysLenientAboutStrayInput(t *testing.T) {
	code, out, _ := runCmd(t, "version", "--junk")
	if code != 0 || !strings.Contains(out, "spine template generation") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

// takeForce interplay pin: the documented trailing --force on cursor tick
// still works because the guard runs on post-takeForce, post-parse args.
func TestCursorTickTrailingForceStillWorks(t *testing.T) {
	dir := cursorWriteRepo(t, "grill, review")
	if code, _, errs := runCmd(t, "cursor", "start", "--dir", dir, "--effort", "e"); code != 0 {
		t.Fatalf("start: code=%d stderr=%q", code, errs)
	}
	if code, _, errs := runCmd(t, "cursor", "tick", "--dir", dir, "review", "--force"); code != 0 {
		t.Fatalf("trailing --force tick: code=%d stderr=%q", code, errs)
	}
}
