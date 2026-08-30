package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validAcceptanceLine = "- [ ] Exercise failover -- APPROVED-UNTESTED 2026-08-29 by owner ref: docs/handoffs/2026-08-29-approval.md#failover reason: lab unavailable"

func TestD15SilentOnValidAndAbsentRecords(t *testing.T) {
	dir := acceptanceRepo(t)
	writeAcceptanceFile(t, dir, "docs/handoffs/2026-08-29-approval.md", "# Approval\n")
	writeAcceptanceTicket(t, dir, "I001-valid.md", "open", "## Acceptance criteria\n"+validAcceptanceLine+"\n")
	if got := acceptanceCheck(dir); len(got) != 0 {
		t.Fatalf("valid record produced D15: %#v", got)
	}
	if err := os.Remove(filepath.Join(dir, "docs", "issues", "I001-valid.md")); err != nil {
		t.Fatal(err)
	}
	if got := acceptanceCheck(dir); len(got) != 0 {
		t.Fatalf("absent record produced D15: %#v", got)
	}
}

func TestD15WarnsOnceForReasonlessRecord(t *testing.T) {
	dir := acceptanceRepo(t)
	writeAcceptanceFile(t, dir, "docs/handoffs/2026-08-29-approval.md", "# Approval\n")
	line := strings.TrimSuffix(validAcceptanceLine, "lab unavailable")
	writeAcceptanceTicket(t, dir, "I002-reasonless.md", "open", "## Acceptance criteria\n"+line+"\n")
	got := acceptanceCheck(dir)
	if len(got) != 1 || got[0].ID != "D15" || got[0].Severity != "warn" {
		t.Fatalf("want one D15 warn, got %#v", got)
	}
}

func TestD15WarnsForMissingArtifactAndClosedTicket(t *testing.T) {
	dir := acceptanceRepo(t)
	writeAcceptanceTicket(t, dir, "I003-closed.md", "fixed", "## Acceptance criteria\n"+validAcceptanceLine+"\n")
	got := acceptanceCheck(dir)
	if len(got) != 1 || got[0].Path != "docs/issues/I003-closed.md" || !strings.Contains(got[0].Message, "must exist") {
		t.Fatalf("want missing-artifact D15 on closed ticket, got %#v", got)
	}
}

func TestD15FindingCarriesPathLineAndAggregatedFailures(t *testing.T) {
	dir := acceptanceRepo(t)
	line := strings.Replace(validAcceptanceLine, "2026-08-29", "2026-02-30", 1)
	writeAcceptanceTicket(t, dir, "I004-bad.md", "open", "## Verification\n"+line+"\n")
	got := acceptanceCheck(dir)
	if len(got) != 1 {
		t.Fatalf("want one finding per candidate, got %#v", got)
	}
	if got[0].Path != "docs/issues/I004-bad.md" || !strings.Contains(got[0].Message, "line 9:") || strings.Count(got[0].Message, "; ") < 2 {
		t.Fatalf("finding lacks path, line, or aggregated failures: %#v", got[0])
	}
}

func TestD15SurfacesAcceptanceTicketReadError(t *testing.T) {
	dir := acceptanceRepo(t)
	if err := os.Symlink(filepath.Join(dir, "missing-ticket.md"), filepath.Join(dir, "docs", "issues", "I005-broken.md")); err != nil {
		t.Fatal(err)
	}

	got := acceptanceCheck(dir)
	if len(got) != 1 || got[0].ID != "D15" || got[0].Severity != "warn" ||
		got[0].Path != "docs/issues/I005-broken.md" || !strings.Contains(got[0].Message, "unable to read ticket") {
		t.Fatalf("ticket read error not surfaced through D15: %#v", got)
	}
}

func TestD15PinsDeterministicAggregatedFailureMessage(t *testing.T) {
	dir := acceptanceRepo(t)
	line := "- [ ]  -- APPROVED-UNTESTED 2026-02-30 by owner ref: outside/approval.txt#x reason: "
	writeAcceptanceTicket(t, dir, "I006-aggregate.md", "open", "## Verification\n"+line+"\n")

	got := acceptanceCheck(dir)
	const want = "line 9: invalid APPROVED-UNTESTED record: criterion is required; date must be a real YYYY-MM-DD date; reference path must be a clean relative docs/ path using slash separators; reference path must end in .md; reference basename must begin with a real YYYY-MM-DD date; reference target must exist; reason is required; record must appear under ## Acceptance criteria"
	if len(got) != 1 || got[0].Message != want {
		t.Fatalf("aggregated D15 message:\n got %#v\nwant %q", got, want)
	}
}

func acceptanceRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "issues"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeAcceptanceTicket(t *testing.T, dir, name, status, body string) {
	t.Helper()
	writeAcceptanceFile(t, dir, filepath.Join("docs", "issues", name), "---\nid: I000\ntitle: test\nseverity: low\nstatus: "+status+"\n---\n\n"+body)
}

func writeAcceptanceFile(t *testing.T, dir, rel, body string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
