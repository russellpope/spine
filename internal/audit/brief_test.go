package audit

import "testing"

// This catches losing the transcript-backed brief mapping: a dispatch can
// reference the same recorded brief through shell substitution, --brief, or a
// bare markdown argument, and each spelling must recover the exact body that
// the lead wrote rather than anything on disk.
func TestBriefTableResolvesRecordedHeredocAcrossReferenceForms(t *testing.T) {
	table := newBriefTable()
	cwd := "/work/repo"
	table.recordCommand("WS=/work/repo/.superpowers/sdd", cwd, 1)
	table.recordCommand("cat > $WS/a.md <<'EOF'\nI101 routine-tier task\nprimary repo: /work/repo\nEOF", cwd, 2)
	table.recordCommand("cat >> $WS/a.md <<'EOF'\ncontext\nEOF", cwd, 3)

	for _, command := range []string{
		`herdr agent prompt impl "$(cat $WS/a.md)"`,
		"herdr agent prompt impl --brief .superpowers/sdd/a.md",
		"herdr agent prompt impl /work/repo/.superpowers/sdd/a.md",
	} {
		ref, ok := referencedBriefPath(command)
		if !ok {
			t.Fatalf("referencedBriefPath(%q) = no reference, want one", command)
		}
		got, ok := table.resolve(ref, cwd, 4)
		if !ok {
			t.Fatalf("resolve(%q) = no brief, want recorded write", ref)
		}
		if got.path != "/work/repo/.superpowers/sdd/a.md" {
			t.Errorf("resolve(%q) path = %q, want normalized absolute path", ref, got.path)
		}
		if got.body != "I101 routine-tier task\nprimary repo: /work/repo\ncontext\n" {
			t.Errorf("resolve(%q) body = %q, want appended recorded body", ref, got.body)
		}
	}

	if _, ok := table.resolve("$MISSING/a.md", cwd, 4); ok {
		t.Error("unexpanded variable resolved, want no recorded brief")
	}
}

// This catches a temporal leak: a later rewrite must never change the brief
// evidence available to a dispatch that appeared before it in the transcript.
func TestBriefTableResolvesMostRecentWriteAtOrBeforePosition(t *testing.T) {
	table := newBriefTable()
	cwd := "/work/repo"
	table.recordCommand("cat > dispatch.md <<'EOF'\nI101 first brief\nEOF", cwd, 10)
	table.recordCommand("cat > dispatch.md <<'EOF'\nI102 later rewrite\nEOF", cwd, 20)

	for _, tc := range []struct {
		position int
		want     string
	}{
		{position: 15, want: "I101 first brief\n"},
		{position: 20, want: "I102 later rewrite\n"},
	} {
		got, ok := table.resolve("dispatch.md", cwd, tc.position)
		if !ok {
			t.Fatalf("resolve at position %d = no brief", tc.position)
		}
		if got.body != tc.want {
			t.Errorf("resolve at position %d body = %q, want %q", tc.position, got.body, tc.want)
		}
	}
}
