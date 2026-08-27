package doctor_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/doctor"
	"github.com/russellpope/spine/internal/scaffold"
)

// ticketFixture scaffolds a repo and writes the given ticket files into
// docs/issues/ (name → frontmatter body, fence included).
func ticketFixture(t *testing.T, tickets map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "library-cli", "demo"); err != nil {
		t.Fatal(err)
	}
	for name, content := range tickets {
		if err := os.WriteFile(filepath.Join(dir, "docs", "issues", name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func ticketFindings(t *testing.T, dir string) []doctor.Finding {
	t.Helper()
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []doctor.Finding
	for _, f := range fs {
		if f.ID == "D13" {
			out = append(out, f)
		}
	}
	return out
}

// ticket renders a minimal ledger ticket carrying the given extra keys.
func ticket(id, status string, extra ...string) string {
	var b strings.Builder
	b.WriteString("---\nid: " + id + "\ntitle: \"demo\"\nseverity: low\nstatus: " + status + "\n")
	for _, line := range extra {
		b.WriteString(line + "\n")
	}
	b.WriteString("---\n\n## Problem\n\ndemo\n")
	return b.String()
}

// TestD13SilentOnHealthyLedger is the primary negative control: well-formed
// keys, an existing workspace path, and an open/fixed mix must produce
// nothing — otherwise doctor warns on every healthy batch in the fleet.
func TestD13SilentOnHealthyLedger(t *testing.T) {
	live := t.TempDir()
	dir := ticketFixture(t, map[string]string{
		"I001-open.md":  ticket("I001", "in-progress", "batch: 2026-08-27-dhyg#1", "workspace: "+live),
		"I002-fixed.md": ticket("I002", "fixed", "batch: 2026-08-27-dhyg#2", "commits: [abc1234, def5678]", "review: approved"),
		"I003-plain.md": ticket("I003", "open"),
	})
	if fs := ticketFindings(t, dir); len(fs) != 0 {
		t.Fatalf("want no D13 findings on a healthy ledger, got %#v", fs)
	}
}

func TestD13WarnsOnMissingWorkspacePath(t *testing.T) {
	gone := filepath.Join(t.TempDir(), "no-such-worktree")
	dir := ticketFixture(t, map[string]string{
		"I001-open.md": ticket("I001", "in-progress", "batch: 2026-08-27-dhyg#1", "workspace: "+gone),
	})
	fs := ticketFindings(t, dir)
	if len(fs) != 1 || fs[0].Severity != "warn" {
		t.Fatalf("want one D13 warn, got %#v", fs)
	}
	if fs[0].Path != "docs/issues/I001-open.md" {
		t.Errorf("path = %q, want the ticket file", fs[0].Path)
	}
	if !strings.Contains(fs[0].Message, "I001-open.md") || !strings.Contains(fs[0].Message, gone) {
		t.Errorf("message %q must name the ticket file and the path", fs[0].Message)
	}
}

// TestD13WarnsOnMissingWorkspaceAtAnyStatus: condition (a) is unscoped — a
// vanished worktree is wrong however the ticket stands.
func TestD13WarnsOnMissingWorkspaceAtAnyStatus(t *testing.T) {
	gone := filepath.Join(t.TempDir(), "no-such-worktree")
	for _, status := range []string{"open", "in-progress", "fixed", "wontfix", "superseded"} {
		dir := ticketFixture(t, map[string]string{
			"I001-x.md": ticket("I001", status, "workspace: "+gone),
		})
		var missing int
		for _, f := range ticketFindings(t, dir) {
			if f.Severity != "warn" {
				t.Errorf("status %q: severity = %q, want warn", status, f.Severity)
			}
			if strings.Contains(f.Message, "does not exist") {
				missing++
			}
		}
		if missing != 1 {
			t.Errorf("status %q: want one missing-path warn, got %d", status, missing)
		}
	}
}

// TestD13WarnsOnWorkspaceOnClosedTicket is condition (b): the presence
// itself is the finding, so the path here exists.
func TestD13WarnsOnWorkspaceOnClosedTicket(t *testing.T) {
	live := t.TempDir()
	for _, status := range []string{"fixed", "wontfix", "superseded"} {
		dir := ticketFixture(t, map[string]string{
			"I001-closed.md": ticket("I001", status, "workspace: "+live),
		})
		fs := ticketFindings(t, dir)
		if len(fs) != 1 || fs[0].Severity != "warn" {
			t.Fatalf("status %q: want one D13 warn, got %#v", status, fs)
		}
		if !strings.Contains(fs[0].Message, "I001-closed.md") || !strings.Contains(fs[0].Message, status) {
			t.Errorf("status %q: message %q must name the ticket file and the status", status, fs[0].Message)
		}
	}
}

// TestD13SilentOnWorkspaceOnOpenTicket: the open half of (b)'s scoping — a
// live worktree on live work is exactly right.
func TestD13SilentOnWorkspaceOnOpenTicket(t *testing.T) {
	live := t.TempDir()
	for _, status := range []string{"open", "in-progress"} {
		dir := ticketFixture(t, map[string]string{
			"I001-open.md": ticket("I001", status, "workspace: "+live),
		})
		if fs := ticketFindings(t, dir); len(fs) != 0 {
			t.Errorf("status %q: want no D13 findings, got %#v", status, fs)
		}
	}
}

func TestD13WarnsOnMalformedBatchOnOpenTicket(t *testing.T) {
	bad := []string{"dhyg#1", "2026-08-27-dhyg", "2026-08-27-dhy#1", "2026-08-27-dhygg#1",
		"2026-8-27-dhyg#1", "2026-08-27-dhyg#", "batch-2"}
	for _, b := range bad {
		for _, status := range []string{"open", "in-progress"} {
			dir := ticketFixture(t, map[string]string{
				"I001-open.md": ticket("I001", status, "batch: "+b),
			})
			fs := ticketFindings(t, dir)
			if len(fs) != 1 || fs[0].Severity != "warn" {
				t.Errorf("batch %q status %q: want one D13 warn, got %#v", b, status, fs)
				continue
			}
			if !strings.Contains(fs[0].Message, "I001-open.md") || !strings.Contains(fs[0].Message, "malformed") {
				t.Errorf("batch %q: message %q must name the ticket file and the problem", b, fs[0].Message)
			}
		}
	}
}

// TestD13SilentOnMalformedBatchOnClosedTicket is (c)'s scoping control: a
// historical malformation on a closed ticket must not warn eternally.
func TestD13SilentOnMalformedBatchOnClosedTicket(t *testing.T) {
	for _, status := range []string{"fixed", "wontfix", "superseded"} {
		dir := ticketFixture(t, map[string]string{
			"I001-closed.md": ticket("I001", status, "batch: totally-malformed"),
		})
		if fs := ticketFindings(t, dir); len(fs) != 0 {
			t.Errorf("status %q: want no D13 findings on a historical batch, got %#v", status, fs)
		}
	}
}

// TestD13SilentOnWellFormedBatch is (c)'s positive control.
func TestD13SilentOnWellFormedBatch(t *testing.T) {
	for _, b := range []string{"2026-08-27-dhyg#1", "1999-01-02-a1b2#12"} {
		dir := ticketFixture(t, map[string]string{
			"I001-open.md": ticket("I001", "open", "batch: "+b),
		})
		if fs := ticketFindings(t, dir); len(fs) != 0 {
			t.Errorf("batch %q: want no D13 findings, got %#v", b, fs)
		}
	}
}

// TestD13SilentOnUnknownKeys keeps the pre-I106 behavior: frontmatter keys
// outside the four are none of this check's business.
func TestD13SilentOnUnknownKeys(t *testing.T) {
	dir := ticketFixture(t, map[string]string{
		"I001-open.md": ticket("I001", "open", "assignee: someone", "wibble: /nonexistent/path", "parent: [I020]"),
	})
	if fs := ticketFindings(t, dir); len(fs) != 0 {
		t.Fatalf("want no D13 findings for unknown keys, got %#v", fs)
	}
}

// TestD13SkipsReadmeAndTemplate: the ledger convention and the ticket
// template are not tickets, even though they live in docs/issues/ and carry
// the key names in their prose.
func TestD13SkipsReadmeAndTemplate(t *testing.T) {
	dir := ticketFixture(t, map[string]string{
		"README.md":    "---\nstatus: fixed\nworkspace: /nonexistent/readme\nbatch: nope\n---\n",
		"_template.md": "---\nstatus: fixed\nworkspace: /nonexistent/template\nbatch: nope\n---\n",
	})
	if fs := ticketFindings(t, dir); len(fs) != 0 {
		t.Fatalf("README/_template must not be scanned as tickets, got %#v", fs)
	}
}
