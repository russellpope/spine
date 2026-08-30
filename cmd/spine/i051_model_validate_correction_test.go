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
