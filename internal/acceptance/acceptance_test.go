package acceptance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const canonical = "- [ ] Exercise the hardware failover path -- APPROVED-UNTESTED 2026-08-29 by owner ref: docs/handoffs/2026-08-29-i050-approval.md#hardware-failover reason: lab hardware unavailable; I123 tracks the deferred run"

func TestScanTicketAcceptsCanonicalRecord(t *testing.T) {
	dir := ticketRepo(t)
	write(t, dir, "docs/handoffs/2026-08-29-i050-approval.md", "# Approval\n")
	path := writeTicket(t, dir, "I050-example.md", "I050", "open", "## Acceptance criteria\n\n"+canonical+"\n")

	got := ScanTicket(dir, path)
	if got.ValidCount() != 1 || got.InvalidCount() != 0 || got.CandidateCount() != 1 {
		t.Fatalf("counts: valid=%d invalid=%d candidates=%d", got.ValidCount(), got.InvalidCount(), got.CandidateCount())
	}
	want := Record{Path: "docs/issues/I050-example.md", Line: 10, Criterion: "Exercise the hardware failover path", Date: "2026-08-29", Approver: "owner", Reference: "docs/handoffs/2026-08-29-i050-approval.md#hardware-failover", Reason: "lab hardware unavailable; I123 tracks the deferred run"}
	if got.Records[0] != want {
		t.Fatalf("record mismatch:\n got %#v\nwant %#v", got.Records[0], want)
	}
}

func TestScanTicketAggregatesMissingFieldsPerCandidate(t *testing.T) {
	dir := ticketRepo(t)
	write(t, dir, "docs/handoffs/2026-08-29-i050-approval.md", "# Approval\n")
	cases := map[string]string{
		"criterion":    "- [ ]  -- APPROVED-UNTESTED 2026-08-29 by owner ref: docs/handoffs/2026-08-29-i050-approval.md#a reason: why",
		"date":         "- [ ] C -- APPROVED-UNTESTED  by owner ref: docs/handoffs/2026-08-29-i050-approval.md#a reason: why",
		"bad date":     "- [ ] C -- APPROVED-UNTESTED 2026-02-30 by owner ref: docs/handoffs/2026-08-29-i050-approval.md#a reason: why",
		"approver":     "- [ ] C -- APPROVED-UNTESTED 2026-08-29 by  ref: docs/handoffs/2026-08-29-i050-approval.md#a reason: why",
		"reference":    "- [ ] C -- APPROVED-UNTESTED 2026-08-29 by owner ref:  reason: why",
		"fragment":     "- [ ] C -- APPROVED-UNTESTED 2026-08-29 by owner ref: docs/handoffs/2026-08-29-i050-approval.md# reason: why",
		"reason":       "- [ ] C -- APPROVED-UNTESTED 2026-08-29 by owner ref: docs/handoffs/2026-08-29-i050-approval.md#a reason: ",
		"separator":    "- [ ] C - APPROVED-UNTESTED 2026-08-29 by owner ref: docs/handoffs/2026-08-29-i050-approval.md#a reason: why",
		"by":           "- [ ] C -- APPROVED-UNTESTED 2026-08-29 BY owner ref: docs/handoffs/2026-08-29-i050-approval.md#a reason: why",
		"ref delim":    "- [ ] C -- APPROVED-UNTESTED 2026-08-29 by owner REF: docs/handoffs/2026-08-29-i050-approval.md#a reason: why",
		"reason delim": "- [ ] C -- APPROVED-UNTESTED 2026-08-29 by owner ref: docs/handoffs/2026-08-29-i050-approval.md#a REASON: why",
	}
	for name, line := range cases {
		t.Run(name, func(t *testing.T) {
			path := writeTicket(t, dir, "I051-"+strings.ReplaceAll(name, " ", "-")+".md", "I051", "open", "## Acceptance criteria\n"+line+"\n")
			got := ScanTicket(dir, path)
			if got.ValidCount() != 0 || got.InvalidCount() != 1 || got.CandidateCount() != 1 || len(got.Problems) != 1 {
				t.Fatalf("want one aggregated problem, got %#v", got)
			}
		})
	}
}

func TestScanTicketRejectsCheckboxAndSectionDamage(t *testing.T) {
	dir := ticketRepo(t)
	write(t, dir, "docs/handoffs/2026-08-29-i050-approval.md", "# Approval\n")
	cases := map[string]string{
		"checked-state": "## Acceptance criteria\n" + strings.Replace(canonical, "- [ ]", "- [x]", 1),
		"indentation":   "## Acceptance criteria\n  " + canonical,
		"checkbox":      "## Acceptance criteria\n-  [ ] " + strings.TrimPrefix(canonical, "- [ ] "),
		"wrong heading": "## Verification\n" + canonical,
		"after next h2": "## Acceptance criteria\n## Resolution\n" + canonical,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			path := writeTicket(t, dir, "I060-"+name+".md", "I060", "open", body+"\n")
			got := ScanTicket(dir, path)
			if got.InvalidCount() != 1 || got.ValidCount() != 0 {
				t.Fatalf("want invalid candidate, got %#v", got)
			}
		})
	}
}

func TestScanTicketIgnoresNonCandidates(t *testing.T) {
	dir := ticketRepo(t)
	body := "## Acceptance criteria\nplain APPROVED-UNTESTED prose\n- [ ] ordinary unchecked item\n- [ ] lowercase approved-untested prose\n```\nAPPROVED-UNTESTED\n```\n"
	path := writeTicket(t, dir, "I061-ignore.md", "I061", "open", body)
	got := ScanTicket(dir, path)
	if got.CandidateCount() != 0 {
		t.Fatalf("non-candidates were scanned: %#v", got)
	}
}

func TestScanTicketCountsMultipleRecordsWithLineAttribution(t *testing.T) {
	dir := ticketRepo(t)
	write(t, dir, "docs/handoffs/2026-08-29-i050-approval.md", "# Approval\n")
	bad := strings.TrimSuffix(canonical, "lab hardware unavailable; I123 tracks the deferred run")
	path := writeTicket(t, dir, "I062-multiple.md", "I062", "fixed", "## Acceptance criteria\n"+canonical+"\n"+bad+"\n")
	got := ScanTicket(dir, path)
	if got.ValidCount() != 1 || got.InvalidCount() != 1 || got.CandidateCount() != 2 {
		t.Fatalf("counts: %#v", got)
	}
	if got.Records[0].Line != 9 || got.Problems[0].Line != 10 || got.Problems[0].Path != "docs/issues/I062-multiple.md" {
		t.Fatalf("attribution: %#v", got)
	}
}

func TestScanTicketRejectsUnsafeApprovalReferences(t *testing.T) {
	dir := ticketRepo(t)
	write(t, dir, "docs/reviews/2026-08-29-ok.md", "# Review\n")
	outsideDir := t.TempDir()
	outsidePath := filepath.Join(outsideDir, "2026-08-29-out.md")
	if err := os.WriteFile(outsidePath, []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs", "reviews", "2026-08-29-dir.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "missing.md"), filepath.Join(dir, "docs", "reviews", "2026-08-29-broken.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsidePath, filepath.Join(dir, "docs", "reviews", "2026-08-29-outside.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "docs", "reviews", "2026-08-29-ok.md"), filepath.Join(dir, "docs", "reviews", "2026-08-29-inside.md")); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"absolute": "/docs/2026-08-29-x.md#a", "parent": "../docs/2026-08-29-x.md#a", "clean-parent": "docs/../docs/reviews/2026-08-29-ok.md#a",
		"backslash": `docs\reviews\2026-08-29-ok.md#a`, "outside-docs": "outside/2026-08-29-out.md#a", "text": "docs/reviews/2026-08-29-ok.txt#a",
		"undated": "docs/reviews/approval.md#a", "no-fragment": "docs/reviews/2026-08-29-ok.md", "missing": "docs/reviews/2026-08-29-missing.md#a",
		"directory": "docs/reviews/2026-08-29-dir.md#a", "broken-link": "docs/reviews/2026-08-29-broken.md#a", "outside-link": "docs/reviews/2026-08-29-outside.md#a",
	}
	for name, ref := range cases {
		t.Run(name, func(t *testing.T) {
			line := "- [ ] C -- APPROVED-UNTESTED 2026-08-29 by owner ref: " + ref + " reason: why"
			path := writeTicket(t, dir, "I070-"+name+".md", "I070", "open", "## Acceptance criteria\n"+line+"\n")
			if got := ScanTicket(dir, path); got.InvalidCount() != 1 || got.ValidCount() != 0 {
				t.Fatalf("got %#v", got)
			}
		})
	}
	for name, ref := range map[string]string{"nested": "docs/reviews/2026-08-29-ok.md#does-not-exist", "inside-link": "docs/reviews/2026-08-29-inside.md#missing"} {
		t.Run(name, func(t *testing.T) {
			line := "- [ ] C -- APPROVED-UNTESTED 2026-08-29 by owner ref: " + ref + " reason: why"
			path := writeTicket(t, dir, "I071-"+name+".md", "I071", "open", "## Acceptance criteria\n"+line+"\n")
			if got := ScanTicket(dir, path); got.ValidCount() != 1 || got.InvalidCount() != 0 {
				t.Fatalf("got %#v", got)
			}
		})
	}
}

func TestScanAllTicketsIncludesClosedTicketsAndSkipsLedgerDocs(t *testing.T) {
	dir := ticketRepo(t)
	write(t, dir, "docs/handoffs/2026-08-29-i050-approval.md", "# Approval\n")
	writeTicket(t, dir, "I080-open.md", "I080", "open", "## Acceptance criteria\n"+canonical+"\n")
	writeTicket(t, dir, "I081-fixed.md", "I081", "fixed", "## Acceptance criteria\n"+canonical+"\n"+canonical+"\n")
	write(t, dir, "docs/issues/README.md", canonical+"\n")
	write(t, dir, "docs/issues/_template.md", canonical+"\n")
	write(t, dir, "docs/issues/not-a-ticket.md", "## Acceptance criteria\n"+canonical+"\n")
	if err := os.Mkdir(filepath.Join(dir, "docs", "issues", "I999-dir.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := ScanAllTickets(dir)
	if got.ValidCount() != 3 || got.CandidateCount() != 3 {
		t.Fatalf("got %#v", got)
	}
}

func TestScanTicketIDsUsesExactFrontmatterIDs(t *testing.T) {
	dir := ticketRepo(t)
	write(t, dir, "docs/handoffs/2026-08-29-i050-approval.md", "# Approval\n")
	writeTicket(t, dir, "I090-picked.md", "I090", "open", "## Acceptance criteria\n"+canonical+"\n")
	writeTicket(t, dir, "I0900-prefix.md", "I0900", "open", "## Acceptance criteria\n"+canonical+"\n")
	writeTicket(t, dir, "I091-unselected.md", "I091", "open", "## Acceptance criteria\n"+canonical+"\n")
	got := ScanTicketIDs(dir, []string{"I090"})
	if got.ValidCount() != 1 || got.Records[0].Path != "docs/issues/I090-picked.md" {
		t.Fatalf("got %#v", got)
	}
}

func ticketRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "issues"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeTicket(t *testing.T, dir, name, id, status, body string) string {
	t.Helper()
	return write(t, dir, filepath.Join("docs", "issues", name), "---\nid: "+id+"\ntitle: test\nseverity: low\nstatus: "+status+"\n---\n\n"+body)
}

func write(t *testing.T, dir, rel, content string) string {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
