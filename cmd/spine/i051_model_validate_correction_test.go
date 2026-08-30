package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestModelValidateQuotesControlBytesInRepositoryPathOnOneLine(t *testing.T) {
	notDirectory := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(notDirectory, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, control, escaped string
	}{
		{"newline", "\n", `\n`},
		{"tab", "\t", `\t`},
		{"carriage return", "\r", `\r`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repoDir := notDirectory + string(filepath.Separator) + tc.control + "repo"
			workflowPath := filepath.Join(repoDir, "WORKFLOW.md")
			code, out, errs := runCmd(t, "model", "--dir", repoDir, "validate", "codex", "primary")
			if code != 2 || out != "" {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
			}
			if strings.Count(errs, "\n") != 1 {
				t.Fatalf("stderr has %d physical lines: %q", strings.Count(errs, "\n"), errs)
			}
			line := strings.TrimSuffix(errs, "\n")
			if strings.ContainsAny(line, "\n\t\r") {
				t.Fatalf("stderr contains raw control byte: %q", errs)
			}
			if !strings.Contains(line, strconv.Quote(workflowPath)) || !strings.Contains(line, tc.escaped) {
				t.Fatalf("stderr=%q, want quoted path %q", errs, workflowPath)
			}
		})
	}
}

func TestI051ModelValidateUsageDiagnosticsArePrefixed(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "duplicate nested expect", args: []string{"model", "validate", "--expect", "gpt-5.6-sol", "--expect", "gpt-5.6-sol", "codex", "primary"}},
		{name: "outer expect", args: []string{"model", "--expect", "gpt-5.6-sol", "validate", "codex", "primary"}},
		{name: "outer equals expect", args: []string{"model", "--expect=gpt-5.6-sol", "validate", "codex", "primary"}},
		{name: "trailing expect", args: []string{"model", "validate", "codex", "primary", "--expect", "gpt-5.6-sol"}},
		{name: "nested dir", args: []string{"model", "validate", "--dir", ".", "codex", "primary"}},
		{name: "nested equals dir", args: []string{"model", "validate", "--dir=.", "codex", "primary"}},
		{name: "trailing dir", args: []string{"model", "validate", "codex", "primary", "--dir", "."}},
		{name: "outer alternate", args: []string{"model", "--alternate", "validate", "codex", "primary"}},
		{name: "nested alternate", args: []string{"model", "validate", "--alternate", "codex", "primary"}},
		{name: "trailing alternate", args: []string{"model", "validate", "codex", "primary", "--alternate"}},
		{name: "outer effort", args: []string{"model", "--effort", "validate", "codex", "primary"}},
		{name: "nested effort", args: []string{"model", "validate", "--effort", "codex", "primary"}},
		{name: "trailing effort", args: []string{"model", "validate", "codex", "primary", "--effort"}},
		{name: "outer json", args: []string{"model", "--json", "validate", "codex", "primary"}},
		{name: "nested json", args: []string{"model", "validate", "--json", "codex", "primary"}},
		{name: "trailing json", args: []string{"model", "validate", "codex", "primary", "--json"}},
		{name: "outer force", args: []string{"model", "--force", "validate", "codex", "primary"}},
		{name: "nested force", args: []string{"model", "validate", "--force", "codex", "primary"}},
		{name: "trailing force", args: []string{"model", "validate", "codex", "primary", "--force"}},
		{name: "outer unknown", args: []string{"model", "--bogus", "validate", "codex", "primary"}},
		{name: "nested unknown", args: []string{"model", "validate", "--bogus", "codex", "primary"}},
		{name: "trailing unknown", args: []string{"model", "validate", "codex", "primary", "--bogus"}},
		{name: "missing expect value", args: []string{"model", "validate", "--expect"}},
		{name: "empty expect value", args: []string{"model", "validate", "--expect=", "codex", "primary"}},
		{name: "missing flavor and tier", args: []string{"model", "validate"}},
		{name: "missing tier", args: []string{"model", "validate", "codex"}},
		{name: "extra positional", args: []string{"model", "validate", "codex", "primary", "extra"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, out, errs := runCmd(t, tc.args...)
			if code != 2 || out != "" || !strings.HasPrefix(errs, "model validate:") {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
			}
		})
	}
}

func TestI051ModelValidateKnownKeyConfigurationDiagnostics(t *testing.T) {
	unreadable := filepath.Join(t.TempDir(), "codex.primary")
	if err := os.Mkdir(unreadable, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(unreadable, "WORKFLOW.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	malformed := writeModelWorkflow(t, "model_routing:\n  codex.primary: one @ high @ low\n")
	newer := writeModelWorkflow(t, "template_version: 14\n")

	for _, tc := range []struct {
		name string
		dir  string
	}{
		{name: "unreadable WORKFLOW", dir: unreadable},
		{name: "malformed selected row", dir: malformed},
		{name: "newer template generation", dir: newer},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, out, errs := runCmd(t, "model", "--dir", tc.dir, "validate", "codex", "primary")
			if code != 2 || out != "" {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
			}
			if !strings.HasPrefix(errs, "model validate:") || strings.Count(errs, "\n") != 1 {
				t.Fatalf("stderr=%q, want one model validate diagnostic line", errs)
			}
			if strings.Count(errs, "codex.primary") != 1 {
				t.Fatalf("stderr=%q, want known route key exactly once", errs)
			}
			if strings.ContainsAny(strings.TrimSuffix(errs, "\n"), "\t\r") {
				t.Fatalf("stderr contains a raw control byte: %q", errs)
			}
		})
	}
}

func TestI051ModelValidateUnknownRouteDiagnosticsStayQuoted(t *testing.T) {
	for _, tc := range []struct {
		name       string
		flavor     string
		tier       string
		wantQuoted string
	}{
		{name: "unknown flavor", flavor: "bad\nflavor", tier: "primary", wantQuoted: `unknown flavor "bad\nflavor"`},
		{name: "unknown tier", flavor: "codex", tier: "bad\ttier", wantQuoted: `unknown tier "bad\ttier"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, out, errs := runCmd(t, "model", "validate", tc.flavor, tc.tier)
			if code != 2 || out != "" || strings.Count(errs, "\n") != 1 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
			}
			if !strings.HasPrefix(errs, "model validate: "+tc.wantQuoted) {
				t.Fatalf("stderr=%q, want safely quoted compatible diagnostic %q", errs, tc.wantQuoted)
			}
			if strings.ContainsAny(strings.TrimSuffix(errs, "\n"), "\t\r") {
				t.Fatalf("stderr contains a raw control byte: %q", errs)
			}
		})
	}
}

func TestI051ModelValidateDiagnosticAdapterDoesNotRewritePlainModelErrors(t *testing.T) {
	code, out, errs := runCmd(t, "model", "--bogus", "codex", "primary")
	if code != 2 || out != "" || !strings.HasPrefix(errs, "flag provided but not defined: -bogus\n") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
	}
	if strings.HasPrefix(errs, "model validate:") {
		t.Fatalf("plain model diagnostic was rewritten: %q", errs)
	}
}
