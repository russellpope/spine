package cursor_test

// Dogfood test: parses this repo's own gitignored .superpowers/sdd/progress.md
// and so skips on a fresh clone. It lives in its own file so the go@1 tskip
// allowlist (gate_pack_config.tskip_allow) can name exactly this file —
// nothing else in the package may skip.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/russellpope/spine/internal/cursor"
)

// This repo's own ledger dogfoods the I018 grammar; the parser must accept
// it cleanly, matching the plan's requirement that the parser reconcile
// against the real ledger, not just synthetic fixtures. The ledger is
// gitignored, so a fresh clone has no progress.md — skip rather than fail in
// that case; the live-machine check still runs wherever the ledger exists.
func TestDogfoodLedgerParsesCleanly(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	ledgerPath := filepath.Join(repoRoot, filepath.FromSlash(".superpowers/sdd/progress.md"))
	if _, err := os.Stat(ledgerPath); os.IsNotExist(err) {
		t.Skip("no .superpowers/sdd/progress.md on this checkout (gitignored) — skipping dogfood check")
	}
	res, err := cursor.Load(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !res.HasCursor {
		t.Fatal("want the repo's own .superpowers/sdd/progress.md to parse as a cursor")
	}
	if len(res.Findings) != 0 {
		t.Fatalf("want the dogfood ledger to parse with no findings, got %#v", res.Findings)
	}
}
