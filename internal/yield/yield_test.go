package yield

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const acceptedTask = "REVIEW I076 harness:codex model:gpt-5.6-terra tier:routine round:1 verdict:accepted scope:task"

func writeLedger(t *testing.T, dir, body string) {
	t.Helper()
	path := filepath.Join(dir, ".superpowers", "sdd")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "progress.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseReviewAcceptsExactTaskRecord(t *testing.T) {
	rec, diag, ok := parseReview(7, acceptedTask)
	if !ok || diag != "" {
		t.Fatalf("parseReview = (%+v, %q, %v), want valid", rec, diag, ok)
	}
	if rec.Ticket != "I076" || rec.Harness != "codex" || rec.ModelID != "gpt-5.6-terra" || rec.Tier != "routine" || rec.Round != 1 || rec.Verdict != VerdictAccepted || rec.Scope != ScopeTask {
		t.Fatalf("record = %+v", rec)
	}
}

func TestParseReviewRejectsMalformedCandidatesWithoutEchoingInput(t *testing.T) {
	cases := []string{
		"REVIEW I076 flavor:codex model:gpt-5.6-terra tier:routine round:1 verdict:accepted scope:task",
		"REVIEW I076 model:gpt-5.6-terra harness:codex tier:routine round:1 verdict:accepted scope:task",
		" REVIEW I076 harness:codex model:gpt-5.6-terra tier:routine round:1 verdict:accepted scope:task",
		"REVIEW I076 harness:codex model:\"gpt-5.6-terra\" tier:routine round:1 verdict:accepted scope:task",
		"REVIEW I076 harness:codex model:gpt-5.6-terra tier:routine round:0 verdict:accepted scope:task",
		"REVIEW I076 harness:codex model:gpt-5.6-terra tier:routine round:01 verdict:accepted scope:task",
		"REVIEW I076 harness:unknown model:gpt-5.6-terra tier:routine round:1 verdict:accepted scope:task",
		"REVIEW I076 harness:codex model:gpt-5.6-terra tier:unknown round:1 verdict:accepted scope:task",
		"REVIEW I076 harness:codex model:gpt-5.6-terra tier:routine verdict:accepted scope:task",
		"REVIEW I076 harness:codex model:gpt-5.6-terra tier:routine round:1 verdict:accepted scope:task extra:nope",
	}
	for _, line := range cases {
		t.Run(strings.ReplaceAll(line, " ", "_"), func(t *testing.T) {
			_, diag, ok := parseReview(11, line)
			if ok || !strings.Contains(diag, "line 11") {
				t.Fatalf("parseReview(%q) = ok %v diag %q", line, ok, diag)
			}
			if strings.Contains(diag, line) || strings.Contains(diag, "gpt-5.6-terra") {
				t.Fatalf("diagnostic leaked ledger content: %q", diag)
			}
		})
	}
}

func TestParseReviewSeparatesFinalForms(t *testing.T) {
	cases := []struct {
		line       string
		wantOK     bool
		wantTicket string
		wantCond   string
	}{
		{"REVIEW I076 harness:codex model:gpt-5.6-terra tier:routine round:1 verdict:accepted scope:final", true, "I076", ""},
		{"REVIEW I076 harness:- model:- tier:- round:2 verdict:needs-fixes scope:final", true, "I076", ""},
		{"REVIEW - harness:- model:- tier:- round:1 verdict:needs-fixes scope:final condition:F-001", true, "", "F-001"},
		{"REVIEW - harness:- model:- tier:- round:1 verdict:accepted scope:final condition:F-001", false, "", ""},
		{"REVIEW - harness:codex model:- tier:- round:1 verdict:needs-fixes scope:final condition:F-001", false, "", ""},
		{"REVIEW I076 harness:codex model:gpt-5.6-terra tier:routine round:1 verdict:accepted scope:final condition:F-001", false, "", ""},
	}
	for _, tc := range cases {
		rec, _, ok := parseReview(1, tc.line)
		if ok != tc.wantOK {
			t.Fatalf("parseReview(%q) ok=%v, want %v", tc.line, ok, tc.wantOK)
		}
		if ok && (rec.Ticket != tc.wantTicket || rec.Condition != tc.wantCond) {
			t.Fatalf("record=%+v", rec)
		}
	}
}

func TestAggregateCountsTaskRoundsAndFinalSeriesWithoutInference(t *testing.T) {
	dir := t.TempDir()
	writeLedger(t, dir, strings.Join([]string{
		"REVIEW I076 harness:codex model:actual-opaque-id tier:routine round:1 verdict:needs-fixes scope:task",
		"REVIEW I076 harness:claude model:later-model tier:primary round:2 verdict:accepted scope:task",
		"REVIEW I077 harness:codex model:actual-opaque-id tier:routine round:1 verdict:accepted scope:task",
		"REVIEW I076 harness:codex model:actual-opaque-id tier:routine round:1 verdict:accepted scope:final",
		"REVIEW I077 harness:codex model:actual-opaque-id tier:routine round:1 verdict:needs-fixes scope:final",
		"REVIEW - harness:- model:- tier:- round:1 verdict:needs-fixes scope:final condition:F-001",
		"ESCALATION I076 routine->primary reason: review requires more depth",
		"ESCALATION I077 effort low->high reason: raw declaration",
		"FALLBACK I077 reason: provider refusal",
		"transcript.jsonl says model: guessed-model",
	}, "\n"))
	report, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if report.Totals.ValidReviewLines != 6 || report.Totals.Escalations != 1 || report.Totals.Fallbacks != 1 {
		t.Fatalf("totals=%+v", report.Totals)
	}
	if report.Totals.FinalAccepted != 1 || report.Totals.FinalNeedsFixes != 1 || report.Totals.FinalUnattributableNeedsFixes != 1 {
		t.Fatalf("final totals=%+v", report.Totals)
	}
	if len(report.Cells) != 1 {
		t.Fatalf("cells=%+v", report.Cells)
	}
	cell := report.Cells[0]
	if cell.Harness != "codex" || cell.ModelID != "actual-opaque-id" || cell.Tier != "routine" || cell.N != 2 || cell.AcceptedFirstPass != 1 || cell.NeedsFixesFirstPass != 1 || cell.ReworkRounds != 1 {
		t.Fatalf("cell=%+v", cell)
	}
	if cell.Rate != "refused" || cell.Confidence != "insufficient" || !report.Refused() {
		t.Fatalf("threshold result cell=%+v report=%+v", cell, report)
	}
}

func TestAggregateRejectsConflictMalformedAndNonContiguousTaskSequences(t *testing.T) {
	dir := t.TempDir()
	writeLedger(t, dir, strings.Join([]string{
		acceptedTask,
		"REVIEW I076 harness:codex model:gpt-5.6-terra tier:routine round:1 verdict:needs-fixes scope:task",
		"REVIEW I077 harness:codex model:gpt-5.6-terra tier:routine round:1 verdict:accepted scope:task",
		"REVIEW I077 harness:codex model:gpt-5.6-terra tier:routine round:2 verdict:needs-fixes scope:task",
		"REVIEW I078 harness:codex model:gpt-5.6-terra tier:routine round:2 verdict:accepted scope:task",
		"REVIEW I079 flavor:codex model:secret-reviewer-note tier:routine round:1 verdict:accepted scope:task",
	}, "\n"))
	report, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Cells) != 0 || report.Totals.IgnoredIdentities != 4 || !report.Invalid() {
		t.Fatalf("report=%+v", report)
	}
	for _, diag := range report.Diagnostics {
		if strings.Contains(diag.Message, "secret-reviewer-note") {
			t.Fatalf("diagnostic leaked malformed value: %q", diag.Message)
		}
	}
}

func TestAggregateDeduplicatesExactRecordsAndUsesAllThresholds(t *testing.T) {
	for _, count := range []int{0, 19, 20, 40} {
		t.Run(fmt.Sprintf("n=%d", count), func(t *testing.T) {
			dir := t.TempDir()
			var lines []string
			for i := 0; i < count; i++ {
				lines = append(lines, fmt.Sprintf("REVIEW I%03d harness:codex model:opaque tier:routine round:1 verdict:accepted scope:task", i+100))
			}
			if count > 0 {
				lines = append(lines, lines[0])
			}
			writeLedger(t, dir, strings.Join(lines, "\n"))
			report, err := Run(Options{Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			if count == 0 {
				if !report.Refused() || len(report.Cells) != 0 {
					t.Fatalf("empty report=%+v", report)
				}
				return
			}
			cell := report.Cells[0]
			if cell.N != count || report.Totals.ValidReviewLines != count+1 || report.Totals.IgnoredIdentities != 0 {
				t.Fatalf("count=%d cell=%+v totals=%+v", count, cell, report.Totals)
			}
			wantRate, wantConfidence := "refused", "insufficient"
			if count >= 20 {
				wantRate, wantConfidence = "100.0%", "low-confidence"
			}
			if count >= 40 {
				wantConfidence = "stated"
			}
			if cell.Rate != wantRate || cell.Confidence != wantConfidence || report.Refused() != (count < 20) {
				t.Fatalf("count=%d cell=%+v refused=%v", count, cell, report.Refused())
			}
		})
	}
}

func TestAggregateCountsOnlyCompleteModelTierEscalationsAndFallbacks(t *testing.T) {
	dir := t.TempDir()
	writeLedger(t, dir, strings.Join([]string{
		"ESCALATION I076 routine->primary reason: accepted wording",
		"ESCALATION I076 effort low->high reason: not model tier",
		"ESCALATION I076 routine -> primary reason: spaced arrow",
		"ESCALATION I076 routine->primary reason: first reason: second",
		"FALLBACK I076 reason: provider refusal",
		"FALLBACK I076 reason: first reason: second",
	}, "\n"))
	report, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if report.Totals.Escalations != 1 || report.Totals.Fallbacks != 1 {
		t.Fatalf("totals=%+v", report.Totals)
	}
}

func makeFleetRepository(t *testing.T, parent, name, ledger string) string {
	t.Helper()
	dir := filepath.Join(parent, name)
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if ledger != "" {
		writeLedger(t, dir, ledger)
	}
	return dir
}

func TestFleetReadsImmediatePrimaryRepositoriesOnceAndKeepsChildrenIsolated(t *testing.T) {
	fleet := t.TempDir()
	writeLedger(t, fleet, "REVIEW I999 harness:claude model:parent-only tier:primary round:1 verdict:accepted scope:task")
	makeFleetRepository(t, fleet, "zeta", "REVIEW I076 harness:codex model:shared tier:routine round:1 verdict:accepted scope:task")
	makeFleetRepository(t, fleet, "alpha", "REVIEW I076 harness:codex model:shared tier:routine round:1 verdict:accepted scope:task")
	makeFleetRepository(t, fleet, "missing", "")
	broken := makeFleetRepository(t, fleet, "broken", "")
	if err := os.MkdirAll(filepath.Join(broken, ".superpowers", "sdd", "progress.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(fleet, "linked")
	if err := os.MkdirAll(linked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(linked, ".git"), []byte("gitdir: elsewhere"), 0o644); err != nil {
		t.Fatal(err)
	}
	hidden := makeFleetRepository(t, fleet, ".hidden", acceptedTask)
	nested := makeFleetRepository(t, filepath.Join(hidden, "nested"), "inner", acceptedTask)
	_ = nested
	if err := os.Symlink(filepath.Join(fleet, "alpha"), filepath.Join(fleet, "symlink")); err != nil {
		t.Fatal(err)
	}

	report, err := Run(Options{Fleet: fleet})
	if err != nil {
		t.Fatal(err)
	}
	if report.Scope != "fleet" || len(report.Cells) != 1 || report.Cells[0].N != 2 || report.Cells[0].ModelID != "shared" {
		t.Fatalf("report=%+v", report)
	}
	if report.Totals.ValidReviewLines != 2 || report.Totals.IgnoredIdentities != 0 || report.ExitCode() != 1 {
		t.Fatalf("totals=%+v exit=%d", report.Totals, report.ExitCode())
	}
	if got := report.Repositories; len(got) != 4 || got[0] != (RepositoryStatus{Name: "alpha", Status: "ok"}) || got[1] != (RepositoryStatus{Name: "broken", Status: "error"}) || got[2] != (RepositoryStatus{Name: "missing", Status: "missing-ledger"}) || got[3] != (RepositoryStatus{Name: "zeta", Status: "ok"}) {
		t.Fatalf("repositories=%+v", got)
	}
	for _, diagnostic := range report.Diagnostics {
		if strings.Contains(diagnostic.Message, fleet) || strings.Contains(diagnostic.Message, "parent-only") {
			t.Fatalf("diagnostic leaked path or ledger content: %+v", diagnostic)
		}
	}
}

func TestFleetRejectsInvalidParentAndDoesNotFollowGitSymlinks(t *testing.T) {
	parentFile := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(parentFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(Options{Fleet: parentFile}); !errors.Is(err, ErrInvalidRoot) {
		t.Fatalf("Run invalid fleet error=%v", err)
	}
	fleet := t.TempDir()
	child := filepath.Join(fleet, "symlinked-git")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(fleet, "missing-target"), filepath.Join(child, ".git")); err != nil {
		t.Fatal(err)
	}
	report, err := Run(Options{Fleet: fleet})
	if err != nil || len(report.Repositories) != 0 || report.Totals.ValidReviewLines != 0 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
}
