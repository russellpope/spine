package eval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	pinEvidenceEval  = "2026-08-30-routing-check"
	pinEvidenceRun   = "gpt-5-6-sol"
	pinEvidenceRef   = "eval:" + pinEvidenceEval + "/runs/" + pinEvidenceRun + ".md"
	pinEvidenceModel = "gpt-5.6-sol"
)

var pinEvidenceToday = time.Date(2026, 8, 30, 18, 0, 0, 0, time.UTC)

func TestPinEvidenceAcceptsOnlyFreshExactPassingRun(t *testing.T) {
	dir := pinEvidenceRepo(t, "2026-06-01", pinEvidenceModel, fullPassingBattery())
	findings := CheckPinEvidence(dir, []PinEvidencePin{{Key: "codex.primary", Model: pinEvidenceModel, EvidenceRefs: []string{pinEvidenceRef}}}, pinEvidenceToday)
	if len(findings) != 0 {
		t.Fatalf("findings = %#v, want none", findings)
	}
}

func TestPinEvidenceClassifiesSelectedRunFailures(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		mutate   func(t *testing.T, dir string)
		want     PinEvidenceKind
		wantPath string
	}{
		{name: "no eval reference", ref: "owner:I068", want: PinEvidenceNoReference, wantPath: "routing-host.json"},
		{name: "bad ref date", ref: "eval:2026-02-30-routing-check/runs/run.md", want: PinEvidenceBadReference, wantPath: "routing-host.json"},
		{name: "bad ref slug", ref: "eval:2026-08-30-Routing/runs/run.md", want: PinEvidenceBadReference, wantPath: "routing-host.json"},
		{name: "bad ref run", ref: "eval:2026-08-30-routing-check/runs/.run.md", want: PinEvidenceBadReference, wantPath: "routing-host.json"},
		{name: "missing run", ref: "eval:2026-08-30-routing-check/runs/missing.md", want: PinEvidenceMissing, wantPath: "docs/evals/2026-08-30-routing-check/runs/missing.md"},
		{name: "missing eval doc", ref: pinEvidenceRef, mutate: func(t *testing.T, dir string) {
			mustRemove(t, filepath.Join(dir, "docs", "evals", pinEvidenceEval, "eval.md"))
		}, want: PinEvidenceMissing, wantPath: "docs/evals/2026-08-30-routing-check/eval.md"},
		{name: "bad front matter", ref: pinEvidenceRef, mutate: func(t *testing.T, dir string) { mustWrite(t, pinEvidenceRunPath(dir), "not front matter\n") }, want: PinEvidenceMalformed, wantPath: "docs/evals/2026-08-30-routing-check/runs/gpt-5-6-sol.md"},
		{name: "future date", ref: pinEvidenceRef, mutate: func(t *testing.T, dir string) {
			writePinEvidenceRun(t, dir, "2026-08-31", pinEvidenceModel, fullPassingBattery())
		}, want: PinEvidenceMalformed, wantPath: "docs/evals/2026-08-30-routing-check/runs/gpt-5-6-sol.md"},
		{name: "day 91 stale", ref: pinEvidenceRef, mutate: func(t *testing.T, dir string) {
			writePinEvidenceRun(t, dir, "2026-05-31", pinEvidenceModel, fullPassingBattery())
		}, want: PinEvidenceStale, wantPath: "docs/evals/2026-08-30-routing-check/runs/gpt-5-6-sol.md"},
		{name: "case mismatch", ref: pinEvidenceRef, mutate: func(t *testing.T, dir string) {
			writePinEvidenceRun(t, dir, "2026-06-01", "GPT-5.6-SOL", fullPassingBattery())
		}, want: PinEvidenceModelMismatch, wantPath: "docs/evals/2026-08-30-routing-check/runs/gpt-5-6-sol.md"},
		{name: "prefix mismatch", ref: pinEvidenceRef, mutate: func(t *testing.T, dir string) {
			writePinEvidenceRun(t, dir, "2026-06-01", pinEvidenceModel+"-preview", fullPassingBattery())
		}, want: PinEvidenceModelMismatch, wantPath: "docs/evals/2026-08-30-routing-check/runs/gpt-5-6-sol.md"},
		{name: "missing battery", ref: pinEvidenceRef, mutate: func(t *testing.T, dir string) { writePinEvidenceRun(t, dir, "2026-06-01", pinEvidenceModel, "") }, want: PinEvidenceNoBattery, wantPath: "docs/evals/2026-08-30-routing-check/runs/gpt-5-6-sol.md"},
		{name: "failed battery", ref: pinEvidenceRef, mutate: func(t *testing.T, dir string) {
			writePinEvidenceRun(t, dir, "2026-06-01", pinEvidenceModel, failingBattery())
		}, want: PinEvidenceFailedBattery, wantPath: "docs/evals/2026-08-30-routing-check/runs/gpt-5-6-sol.md"},
		{name: "bad battery order", ref: pinEvidenceRef, mutate: func(t *testing.T, dir string) {
			writePinEvidenceRun(t, dir, "2026-06-01", pinEvidenceModel, strings.Replace(fullPassingBattery(), "invocation=KILLED,wiring=KILLED", "wiring=KILLED,invocation=KILLED", 1))
		}, want: PinEvidenceMalformed, wantPath: "docs/evals/2026-08-30-routing-check/runs/gpt-5-6-sol.md"},
		{name: "inconsistent battery verdict", ref: pinEvidenceRef, mutate: func(t *testing.T, dir string) {
			writePinEvidenceRun(t, dir, "2026-06-01", pinEvidenceModel, strings.Replace(fullPassingBattery(), "battery_verdict: pass", "battery_verdict: fail", 1))
		}, want: PinEvidenceMalformed, wantPath: "docs/evals/2026-08-30-routing-check/runs/gpt-5-6-sol.md"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := pinEvidenceRepo(t, "2026-06-01", pinEvidenceModel, fullPassingBattery())
			if tc.mutate != nil {
				tc.mutate(t, dir)
			}
			findings := CheckPinEvidence(dir, []PinEvidencePin{{Key: "codex.primary", Model: pinEvidenceModel, EvidenceRefs: []string{tc.ref}}}, pinEvidenceToday)
			if len(findings) != 1 || findings[0].Kind != tc.want || findings[0].Path != tc.wantPath {
				t.Fatalf("findings = %#v, want kind %v path %q", findings, tc.want, tc.wantPath)
			}
		})
	}
}

func TestPinEvidenceDateQuoteAggregationAndContainment(t *testing.T) {
	t.Run("day 90 and quoted exact model pass", func(t *testing.T) {
		dir := pinEvidenceRepo(t, "2026-06-01", `"gpt-5.6-sol"`, fullPassingBattery())
		if got := CheckPinEvidence(dir, []PinEvidencePin{{Key: "codex.primary", Model: pinEvidenceModel, EvidenceRefs: []string{pinEvidenceRef}}}, pinEvidenceToday); len(got) != 0 {
			t.Fatalf("findings = %#v", got)
		}
	})
	t.Run("all selected references are checked in byte order", func(t *testing.T) {
		dir := pinEvidenceRepo(t, "2026-06-01", pinEvidenceModel, fullPassingBattery())
		refs := []string{pinEvidenceRef, "eval:2026-08-30-routing-check/runs/a.md"}
		got := CheckPinEvidence(dir, []PinEvidencePin{{Key: "codex.primary", Model: pinEvidenceModel, EvidenceRefs: refs}}, pinEvidenceToday)
		if len(got) != 1 || got[0].Kind != PinEvidenceMissing || !strings.HasSuffix(got[0].Path, "/a.md") {
			t.Fatalf("findings = %#v", got)
		}
	})
	t.Run("unreferenced malformed eval is ignored", func(t *testing.T) {
		dir := pinEvidenceRepo(t, "2026-06-01", pinEvidenceModel, fullPassingBattery())
		mustWrite(t, filepath.Join(dir, "docs", "evals", "2026-08-29-unrelated", "runs", "bad.md"), "secret body\n")
		if got := CheckPinEvidence(dir, []PinEvidencePin{{Key: "codex.primary", Model: pinEvidenceModel, EvidenceRefs: []string{pinEvidenceRef}}}, pinEvidenceToday); len(got) != 0 {
			t.Fatalf("findings = %#v", got)
		}
	})
	for _, rel := range []string{"docs", "docs/evals", "docs/evals/" + pinEvidenceEval, "docs/evals/" + pinEvidenceEval + "/runs", "docs/evals/" + pinEvidenceEval + "/runs/" + pinEvidenceRun + ".md"} {
		t.Run("symlink "+rel, func(t *testing.T) {
			dir := pinEvidenceRepo(t, "2026-06-01", pinEvidenceModel, fullPassingBattery())
			target := t.TempDir()
			mustRemove(t, filepath.Join(dir, rel))
			if err := os.Symlink(target, filepath.Join(dir, rel)); err != nil {
				t.Fatal(err)
			}
			got := CheckPinEvidence(dir, []PinEvidencePin{{Key: "codex.primary", Model: pinEvidenceModel, EvidenceRefs: []string{pinEvidenceRef}}}, pinEvidenceToday)
			if len(got) != 1 || got[0].Kind != PinEvidenceMalformed || strings.Contains(got[0].Path, target) {
				t.Fatalf("findings = %#v", got)
			}
		})
	}
}

func pinEvidenceRepo(t *testing.T, created, model, battery string) string {
	t.Helper()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "docs", "evals", pinEvidenceEval, "eval.md"), "---\ntitle: demo\ncreated: 2026-08-30\nprompt: prompt.md\nrubric: rubric.md\n---\n")
	writePinEvidenceRun(t, dir, created, model, battery)
	return dir
}

func writePinEvidenceRun(t *testing.T, dir, created, model, battery string) {
	t.Helper()
	extra := ""
	if battery != "" {
		extra = battery + "\n"
	}
	mustWrite(t, pinEvidenceRunPath(dir), "---\nname: "+pinEvidenceRun+"\ncreated: "+created+"\nmodel: "+model+"\nstage: raw\nscore: 1\n"+extra+"---\nsecret body\n")
}

func pinEvidenceRunPath(dir string) string {
	return filepath.Join(dir, "docs", "evals", pinEvidenceEval, "runs", pinEvidenceRun+".md")
}

func fullPassingBattery() string {
	return "battery_version: 1\nbattery_verdict: pass\nbattery_results: invocation=KILLED,wiring=KILLED,flag-honoured=KILLED,column-presence=KILLED,column-order=KILLED,ordering=KILLED,units-labels=KILLED,security-default=REPORT-ONLY,lifecycle=REPORT-ONLY,error-path-behaviour=KILLED"
}
func failingBattery() string {
	value := strings.Replace(fullPassingBattery(), "battery_verdict: pass", "battery_verdict: fail", 1)
	return strings.Replace(value, "invocation=KILLED", "invocation=SURVIVED", 1)
}
func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
func mustRemove(t *testing.T, path string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
}
