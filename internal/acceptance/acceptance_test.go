package acceptance

import (
	"errors"
	"fmt"
	"io"
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

func TestScanTicketRejectsWhitespaceInApproverAndReferenceToken(t *testing.T) {
	dir := ticketRepo(t)
	for _, rel := range []string{
		"docs/reviews/2026-08-29-ok.md",
		"docs/review notes/2026-08-29-ok.md",
		"docs/review\tnotes/2026-08-29-ok.md",
		"docs/review\u2003notes/2026-08-29-ok.md",
	} {
		write(t, dir, rel, "# Approval\n")
	}
	cases := []struct {
		name string
		line string
		want []string
	}{
		{"approver ASCII space", "- [ ] C -- APPROVED-UNTESTED 2026-08-29 by owner team ref: docs/reviews/2026-08-29-ok.md#a reason: why", []string{"approver must be a whitespace-free token"}},
		{"approver tab", "- [ ] C -- APPROVED-UNTESTED 2026-08-29 by owner\tteam ref: docs/reviews/2026-08-29-ok.md#a reason: why", []string{"approver must be a whitespace-free token"}},
		{"approver Unicode whitespace", "- [ ] C -- APPROVED-UNTESTED 2026-08-29 by owner\u2003team ref: docs/reviews/2026-08-29-ok.md#a reason: why", []string{"approver must be a whitespace-free token"}},
		{"reference base ASCII space", "- [ ] C -- APPROVED-UNTESTED 2026-08-29 by owner ref: docs/review notes/2026-08-29-ok.md#a reason: why", []string{"reference must be a whitespace-free token"}},
		{"reference base tab", "- [ ] C -- APPROVED-UNTESTED 2026-08-29 by owner ref: docs/review\tnotes/2026-08-29-ok.md#a reason: why", []string{"reference must be a whitespace-free token"}},
		{"reference base Unicode whitespace", "- [ ] C -- APPROVED-UNTESTED 2026-08-29 by owner ref: docs/review\u2003notes/2026-08-29-ok.md#a reason: why", []string{"reference must be a whitespace-free token"}},
		{"reference fragment ASCII space", "- [ ] C -- APPROVED-UNTESTED 2026-08-29 by owner ref: docs/reviews/2026-08-29-ok.md#a b reason: why", []string{"reference must be a whitespace-free token"}},
		{"reference fragment tab", "- [ ] C -- APPROVED-UNTESTED 2026-08-29 by owner ref: docs/reviews/2026-08-29-ok.md#a\tb reason: why", []string{"reference must be a whitespace-free token"}},
		{"reference fragment Unicode whitespace", "- [ ] C -- APPROVED-UNTESTED 2026-08-29 by owner ref: docs/reviews/2026-08-29-ok.md#a\u2003b reason: why", []string{"reference must be a whitespace-free token"}},
		{"ordered token failures", "- [ ] C -- APPROVED-UNTESTED 2026-08-29 by owner team ref: docs/reviews/2026-08-29-ok.md#a b reason: why", []string{"approver must be a whitespace-free token", "reference must be a whitespace-free token"}},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTicket(t, dir, fmt.Sprintf("I3%02d-whitespace.md", i), "I300", "open", "## Acceptance criteria\n"+tc.line+"\n")
			got := ScanTicket(dir, path)
			if got.ValidCount() != 0 || got.InvalidCount() != 1 || !slicesEqual(got.Problems[0].Failed, tc.want) {
				t.Fatalf("whitespace token failures:\n got %#v\nwant %#v", got, tc.want)
			}
		})
	}
}

func TestScanTicketSelectsUniqueStructuralMarker(t *testing.T) {
	dir := ticketRepo(t)
	write(t, dir, "docs/reviews/2026-08-29-ok.md", "# Approval\n")
	line := "- [ ] Document APPROVED-UNTESTED semantics -- APPROVED-UNTESTED 2026-08-29 by owner ref: docs/reviews/2026-08-29-ok.md#a reason: why"
	path := writeTicket(t, dir, "I310-structural.md", "I310", "open", "## Acceptance criteria\n"+line+"\n")

	got := ScanTicket(dir, path)
	if got.ValidCount() != 1 || got.InvalidCount() != 0 || got.Records[0].Criterion != "Document APPROVED-UNTESTED semantics" {
		t.Fatalf("structural marker selection = %#v", got)
	}
}

func TestScanTicketRejectsAmbiguousStructuralMarkers(t *testing.T) {
	dir := ticketRepo(t)
	write(t, dir, "docs/reviews/2026-08-29-ok.md", "# Approval\n")
	line := "- [ ] First -- APPROVED-UNTESTED 2026-08-29 by owner ref: docs/reviews/2026-08-29-ok.md#a reason: why -- APPROVED-UNTESTED 2026-08-30 by other ref: docs/reviews/2026-08-29-ok.md#b reason: second"
	path := writeTicket(t, dir, "I311-ambiguous.md", "I311", "open", "## Acceptance criteria\n"+line+"\n")

	got := ScanTicket(dir, path)
	want := []string{"record must contain exactly one ` -- APPROVED-UNTESTED ` structural marker"}
	if got.ValidCount() != 0 || got.InvalidCount() != 1 || !slicesEqual(got.Problems[0].Failed, want) {
		t.Fatalf("ambiguous structural markers:\n got %#v\nwant %#v", got, want)
	}
}

func TestI050Rereview2DamagedMultipleMarkerUsesContractOrder(t *testing.T) {
	dir := ticketRepo(t)
	write(t, dir, "docs/reviews/2026-08-29-ok.md", "# Approval\n")
	line := "- [ ]  -- APPROVED-UNTESTED 2026-02-30 by owner ref: docs/reviews/2026-08-29-ok.md#a reason: why -- APPROVED-UNTESTED 2026-08-30 by other ref: docs/reviews/2026-08-29-ok.md#b reason: second"
	path := writeTicket(t, dir, "I312-contract-order.md", "I312", "open", "## Acceptance criteria\n"+line+"\n")

	got := ScanTicket(dir, path)
	want := []string{
		"criterion is required",
		"record must contain exactly one ` -- APPROVED-UNTESTED ` structural marker",
		"date must be a real YYYY-MM-DD date",
	}
	if got.ValidCount() != 0 || got.InvalidCount() != 1 || got.CandidateCount() != 1 ||
		!slicesEqual(got.Problems[0].Failed, want) {
		t.Fatalf("damaged multiple-marker contract order:\n got %#v\nwant %#v", got.Problems, want)
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

func TestScanTicketIDsSkipsPreIDFailures(t *testing.T) {
	dir := ticketRepo(t)
	write(t, dir, "docs/handoffs/2026-08-29-i050-approval.md", "# Approval\n")
	writeTicket(t, dir, "I090-picked.md", "I090", "open", "## Acceptance criteria\n"+canonical+"\n")
	writeTicket(t, dir, "I091-unscoped.md", "I091", "open", "## Acceptance criteria\n"+strings.TrimSuffix(canonical, "lab hardware unavailable; I123 tracks the deferred run")+"\n")
	if err := os.Symlink(filepath.Join(dir, "missing-ticket.md"), filepath.Join(dir, "docs", "issues", "I092-broken.md")); err != nil {
		t.Fatal(err)
	}

	scoped := ScanTicketIDs(dir, []string{"I090"})
	if scoped.ValidCount() != 1 || scoped.InvalidCount() != 0 || len(scoped.ScanErrors) != 0 {
		t.Fatalf("scoped scan leaked unknown or unscoped tickets: %#v", scoped)
	}
	estate := ScanAllTickets(dir)
	if estate.ValidCount() != 1 || estate.InvalidCount() != 1 || len(estate.ScanErrors) != 1 || estate.ScanErrors[0].Path != "docs/issues/I092-broken.md" {
		t.Fatalf("estate scan lost unscoped failures: %#v", estate)
	}
}

func TestScanTicketIDsSkipsPreIDFailuresAndSurfacesWantedPostIDFailures(t *testing.T) {
	dir := ticketRepo(t)
	write(t, dir, "docs/handoffs/2026-08-29-i050-approval.md", "# Approval\n")
	for _, name := range []string{"I400-wanted.md", "I401-unscoped.md", "I402-pre-id.md"} {
		write(t, dir, filepath.Join("docs", "issues", name), "placeholder\n")
	}
	wantedErr := errors.New("wanted post-ID read failure")
	unscopedErr := errors.New("unscoped post-ID read failure")
	preIDErr := errors.New("pre-ID read failure")
	openFile := func(name string) (io.ReadCloser, error) {
		var reader io.Reader
		switch filepath.Base(name) {
		case "I400-wanted.md":
			reader = &errorAfterReader{data: []byte("---\nid: I400\n---\n\n## Acceptance criteria\n" + canonical), err: wantedErr}
		case "I401-unscoped.md":
			reader = &errorAfterReader{data: []byte("---\nid: I401\n---\n"), err: unscopedErr}
		case "I402-pre-id.md":
			reader = &errorAfterReader{data: []byte("---\n"), err: preIDErr}
		default:
			return os.Open(name)
		}
		return &trackingReadCloser{Reader: reader}, nil
	}

	got := scanTicketsWithOpen(dir, map[string]bool{"I400": true}, openFile)
	if got.ValidCount() != 1 || got.InvalidCount() != 0 || len(got.ScanErrors) != 1 || !errors.Is(got.ScanErrors[0].Err, wantedErr) {
		t.Fatalf("identity-scoped post-ID errors = %#v", got)
	}
}

func TestScanTicketIDsClosesEachDiscoveredTicketPerIteration(t *testing.T) {
	dir := ticketRepo(t)
	write(t, dir, "docs/issues/I410-first.md", "first\n")
	write(t, dir, "docs/issues/I411-second.md", "second\n")
	var first *trackingReadCloser
	openFile := func(name string) (io.ReadCloser, error) {
		switch filepath.Base(name) {
		case "I410-first.md":
			first = &trackingReadCloser{Reader: strings.NewReader("---\nid: I410\n---\n")}
			return first, nil
		case "I411-second.md":
			if first == nil || !first.closed {
				t.Fatal("first ticket remained open when the next iteration began")
			}
			return &trackingReadCloser{Reader: strings.NewReader("---\nid: I411\n---\n")}, nil
		default:
			return os.Open(name)
		}
	}

	got := scanTicketsWithOpen(dir, nil, openFile)
	if len(got.ScanErrors) != 0 || first == nil || !first.closed {
		t.Fatalf("per-iteration close contract = %#v, first=%#v", got, first)
	}
}

func TestTicketCloseErrorsFollowIdentityScope(t *testing.T) {
	dir := ticketRepo(t)
	write(t, dir, "docs/issues/I420-wanted.md", "placeholder\n")
	write(t, dir, "docs/issues/I421-unscoped.md", "placeholder\n")
	wantedCloseErr := errors.New("wanted close failure")
	unscopedCloseErr := errors.New("unscoped close failure")
	openFile := func(name string) (io.ReadCloser, error) {
		switch filepath.Base(name) {
		case "I420-wanted.md":
			return &trackingReadCloser{Reader: strings.NewReader("---\nid: I420\n---\n"), closeErr: wantedCloseErr}, nil
		case "I421-unscoped.md":
			return &trackingReadCloser{Reader: strings.NewReader("---\nid: I421\n---\n"), closeErr: unscopedCloseErr}, nil
		default:
			return os.Open(name)
		}
	}

	scoped := scanTicketsWithOpen(dir, map[string]bool{"I420": true}, openFile)
	if len(scoped.ScanErrors) != 1 || !errors.Is(scoped.ScanErrors[0].Err, wantedCloseErr) {
		t.Fatalf("scoped close errors = %#v", scoped)
	}
	estate := scanTicketsWithOpen(dir, nil, openFile)
	if len(estate.ScanErrors) != 2 || !errors.Is(estate.ScanErrors[0].Err, wantedCloseErr) || !errors.Is(estate.ScanErrors[1].Err, unscopedCloseErr) {
		t.Fatalf("estate close errors = %#v", estate)
	}
}

func TestScanAllTicketsAcceptsRelativeRoot(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "repo")
	if err := os.MkdirAll(filepath.Join(dir, "docs", "issues"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "docs/handoffs/2026-08-29-i050-approval.md", "# Approval\n")
	writeTicket(t, dir, "I092-relative.md", "I092", "open", "## Acceptance criteria\n"+canonical+"\n")
	t.Chdir(base)

	got := ScanAllTickets("repo")
	if got.ValidCount() != 1 || got.Records[0].Path != "docs/issues/I092-relative.md" {
		t.Fatalf("relative root scan = %#v", got)
	}
}

func TestScanTicketIDsAcceptsRelativeRoot(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "repo")
	if err := os.MkdirAll(filepath.Join(dir, "docs", "issues"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "docs/handoffs/2026-08-29-i050-approval.md", "# Approval\n")
	writeTicket(t, dir, "I093-relative.md", "I093", "open", "## Acceptance criteria\n"+canonical+"\n")
	writeTicket(t, dir, "I094-unselected.md", "I094", "open", "## Acceptance criteria\n"+canonical+"\n")
	t.Chdir(base)

	got := ScanTicketIDs("repo", []string{"I093"})
	if got.ValidCount() != 1 || got.Records[0].Path != "docs/issues/I093-relative.md" {
		t.Fatalf("relative root scoped scan = %#v", got)
	}
}

func TestScanTicketAcceptsArbitrarilyLongCandidateLine(t *testing.T) {
	dir := ticketRepo(t)
	write(t, dir, "docs/handoffs/2026-08-29-i050-approval.md", "# Approval\n")
	criterion := strings.Repeat("candidate-byte-", 6000)
	line := "- [ ] " + criterion + " -- APPROVED-UNTESTED 2026-08-29 by owner ref: docs/handoffs/2026-08-29-i050-approval.md#long#fragment reason: why"
	path := writeTicket(t, dir, "I095-long-candidate.md", "I095", "open", "## Acceptance criteria\n"+line+"\n")

	got := ScanTicket(dir, path)
	if got.ValidCount() != 1 || got.InvalidCount() != 0 || got.Records[0].Criterion != criterion ||
		got.Records[0].Reference != "docs/handoffs/2026-08-29-i050-approval.md#long#fragment" {
		t.Fatalf("long candidate scan = %#v", got)
	}
}

func TestScanTicketLongNoncandidateDoesNotHideLaterCandidates(t *testing.T) {
	dir := ticketRepo(t)
	write(t, dir, "docs/handoffs/2026-08-29-i050-approval.md", "# Approval\n")
	bad := strings.TrimSuffix(canonical, "lab hardware unavailable; I123 tracks the deferred run")
	body := "## Acceptance criteria\n" + strings.Repeat("ordinary prose ", 6000) + "\n" + canonical + "\n" + bad + "\n"
	path := writeTicket(t, dir, "I096-long-before.md", "I096", "open", body)

	got := ScanTicket(dir, path)
	if got.ValidCount() != 1 || got.InvalidCount() != 1 || got.Records[0].Line != 10 || got.Problems[0].Line != 11 {
		t.Fatalf("long noncandidate scan = %#v", got)
	}
}

func TestScanReaderSurfacesReadError(t *testing.T) {
	dir := ticketRepo(t)
	write(t, dir, "docs/handoffs/2026-08-29-i050-approval.md", "# Approval\n")
	wantErr := errors.New("injected read failure")
	reader := &errorAfterReader{data: []byte("## Acceptance criteria\n" + canonical), err: wantErr}

	got := scanReader(dir, "docs/issues/I097-read.md", reader)
	if got.ValidCount() != 1 || got.CandidateCount() != 1 {
		t.Fatalf("partial line before read failure was not processed: %#v", got)
	}
	if len(got.ScanErrors) != 1 || got.ScanErrors[0].Path != "docs/issues/I097-read.md" || !errors.Is(got.ScanErrors[0].Err, wantErr) {
		t.Fatalf("read failure not surfaced: %#v", got.ScanErrors)
	}
}

func TestScanTicketAggregatesEveryApplicableFailureDeterministically(t *testing.T) {
	dir := ticketRepo(t)
	line := "- [ ]  -- APPROVED-UNTESTED 2026-02-30 by owner ref: outside/approval.txt#x reason: "
	path := writeTicket(t, dir, "I098-aggregate.md", "I098", "open", "## Verification\n"+line+"\n")
	want := []string{
		"criterion is required",
		"date must be a real YYYY-MM-DD date",
		"reference path must be a clean relative docs/ path using slash separators",
		"reference path must end in .md",
		"reference basename must begin with a real YYYY-MM-DD date",
		"reference target must exist",
		"reason is required",
		"record must appear under ## Acceptance criteria",
	}
	for i := 0; i < 20; i++ {
		got := ScanTicket(dir, path)
		if got.InvalidCount() != 1 || got.CandidateCount() != 1 || !slicesEqual(got.Problems[0].Failed, want) {
			t.Fatalf("iteration %d aggregated failures:\n got %#v\nwant %#v", i, got.Problems, want)
		}
	}
}

func TestScanTicketAggregatesRecoverableGrammarAndReferenceFailures(t *testing.T) {
	dir := ticketRepo(t)
	line := " \t- [x] C - APPROVED-UNTESTED 2026-02-30 by owner ref: outside/approval.txt#x reason: "
	path := writeTicket(t, dir, "I099-grammar.md", "I099", "open", "## Acceptance criteria\n"+line+"\n")
	want := []string{
		"record must start at column 0 with the exact - [ ] prefix",
		"criterion and marker must be separated by exact ` -- ` bytes",
		"date must be a real YYYY-MM-DD date",
		"reference path must be a clean relative docs/ path using slash separators",
		"reference path must end in .md",
		"reference basename must begin with a real YYYY-MM-DD date",
		"reference target must exist",
		"reason is required",
	}

	got := ScanTicket(dir, path)
	if got.InvalidCount() != 1 || !slicesEqual(got.Problems[0].Failed, want) {
		t.Fatalf("recoverable grammar failures:\n got %#v\nwant %#v", got.Problems, want)
	}
}

func TestScanTicketRecognizesEveryColumnZeroH1H2Boundary(t *testing.T) {
	dir := ticketRepo(t)
	write(t, dir, "docs/handoffs/2026-08-29-i050-approval.md", "# Approval\n")
	boundaries := []string{"#", "# Title", "#\tTitle", "##", "## Title", "##\tTitle"}
	for i, heading := range boundaries {
		t.Run(strings.ReplaceAll(heading, "\t", "-tab-"), func(t *testing.T) {
			path := writeTicket(t, dir, fmt.Sprintf("I1%02d-boundary.md", i), "I100", "open", "## Acceptance criteria\n"+heading+"\n"+canonical+"\n")
			got := ScanTicket(dir, path)
			if got.ValidCount() != 0 || got.InvalidCount() != 1 ||
				!slicesEqual(got.Problems[0].Failed, []string{"record must appear under ## Acceptance criteria"}) {
				t.Fatalf("heading %q did not end acceptance scope: %#v", heading, got)
			}
		})
	}
}

func TestScanTicketKeepsH3HeadingsInsideAcceptanceSection(t *testing.T) {
	dir := ticketRepo(t)
	write(t, dir, "docs/handoffs/2026-08-29-i050-approval.md", "# Approval\n")
	for i, heading := range []string{"###", "### Title", "###\tTitle"} {
		t.Run(strings.ReplaceAll(heading, "\t", "-tab-"), func(t *testing.T) {
			path := writeTicket(t, dir, fmt.Sprintf("I2%02d-h3.md", i), "I200", "open", "## Acceptance criteria\n"+heading+"\n"+canonical+"\n")
			got := ScanTicket(dir, path)
			if got.ValidCount() != 1 || got.InvalidCount() != 0 {
				t.Fatalf("H3 heading %q ended acceptance scope: %#v", heading, got)
			}
		})
	}
}

func slicesEqual(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

type errorAfterReader struct {
	data []byte
	err  error
}

func (r *errorAfterReader) Read(p []byte) (int, error) {
	if len(r.data) > 0 {
		n := copy(p, r.data)
		r.data = r.data[n:]
		return n, nil
	}
	return 0, r.err
}

var _ io.Reader = (*errorAfterReader)(nil)

type trackingReadCloser struct {
	io.Reader
	closed   bool
	closeErr error
}

func (r *trackingReadCloser) Close() error {
	r.closed = true
	return r.closeErr
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
