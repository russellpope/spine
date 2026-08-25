package gate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunRecoversPlainCheckPanics protects Run's exit-code contract: a future
// plain check that panics must remain an internal error rather than killing the
// gate process before it can report an outcome.
func TestRunRecoversPlainCheckPanics(t *testing.T) {
	const panicValue = "plain check panic"
	old := checks["tskip"]
	checks["tskip"] = func(string, Config) ([]Finding, error) {
		panic(panicValue)
	}
	t.Cleanup(func() { checks["tskip"] = old })

	var stdout, stderr bytes.Buffer
	if got := Run("go@1", "tskip", t.TempDir(), &stdout, &stderr, EnvConfig()); got != 2 {
		t.Fatalf("Run exit = %d, want 2; stdout=%q stderr=%q", got, stdout.String(), stderr.String())
	}
	if got := stderr.String(); !strings.Contains(got, "internal error") || !strings.Contains(got, panicValue) || !strings.Contains(got, "goroutine ") {
		t.Errorf("stderr = %q, want internal-error message with panic value and stack", got)
	}
	if got := stdout.String(); got != "" {
		t.Errorf("stdout = %q, want no emitted result", got)
	}
}

// TestRunRecoversReportCheckPanics protects the same boundary for rich checks,
// whose Report return shape must not bypass Run's panic handling.
func TestRunRecoversReportCheckPanics(t *testing.T) {
	const panicValue = "report check panic"
	old := reportChecks["mutate"]
	reportChecks["mutate"] = func(string, Config) (Report, error) {
		panic(panicValue)
	}
	t.Cleanup(func() { reportChecks["mutate"] = old })

	var stdout, stderr bytes.Buffer
	if got := Run("go@1", "mutate", t.TempDir(), &stdout, &stderr, EnvConfig()); got != 2 {
		t.Fatalf("Run exit = %d, want 2; stdout=%q stderr=%q", got, stdout.String(), stderr.String())
	}
	if got := stderr.String(); !strings.Contains(got, "internal error") || !strings.Contains(got, panicValue) || !strings.Contains(got, "goroutine ") {
		t.Errorf("stderr = %q, want internal-error message with panic value and stack", got)
	}
	if got := stdout.String(); got != "" {
		t.Errorf("stdout = %q, want no emitted result", got)
	}
}

// TestRunPanicDoesNotEmitResults protects the ordering that keeps a failed
// check from publishing a verdict document before Run returns its error exit.
func TestRunPanicDoesNotEmitResults(t *testing.T) {
	const panicValue = "results check panic"
	old := checks["tskip"]
	checks["tskip"] = func(string, Config) ([]Finding, error) {
		panic(panicValue)
	}
	t.Cleanup(func() { checks["tskip"] = old })

	resultsPath := filepath.Join(t.TempDir(), "results.json")
	t.Setenv(ResultsEnvVar, resultsPath)
	var stdout, stderr bytes.Buffer
	if got := Run("go@1", "tskip", t.TempDir(), &stdout, &stderr, EnvConfig()); got != 2 {
		t.Fatalf("Run exit = %d, want 2; stdout=%q stderr=%q", got, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(resultsPath); !os.IsNotExist(err) {
		t.Errorf("recovered panic wrote results file: err=%v", err)
	}
}
