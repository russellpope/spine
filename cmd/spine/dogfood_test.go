package main

// Dogfood tests: they run the binary against this repo's own gitignored
// .superpowers/sdd/progress.md and so skip on a fresh clone. They live in
// their own file so the go@1 tskip allowlist (gate_pack_config.tskip_allow)
// can name exactly this file — nothing else in the package may skip.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCursorCommandOnRealRepoLedger(t *testing.T) {
	// The ledger is gitignored, so a fresh clone has no progress.md — skip
	// rather than fail in that case; the live-machine check still runs
	// wherever the ledger exists.
	repoRoot := filepath.Join("..", "..")
	ledgerPath := filepath.Join(repoRoot, filepath.FromSlash(".superpowers/sdd/progress.md"))
	if _, err := os.Stat(ledgerPath); os.IsNotExist(err) {
		t.Skip("no .superpowers/sdd/progress.md on this checkout (gitignored) — skipping")
	}
	code, out, errs := runCmd(t, "cursor", "--dir", repoRoot)
	if code != 0 {
		t.Fatalf("code=%d errs=%q", code, errs)
	}
	if strings.Contains(out, "finding:") {
		t.Errorf("want the real ledger to parse cleanly with zero findings, out=%q", out)
	}
	// The live verdict depends on this build's real, evolving on-disk state
	// (its own dogfood cursor, ticket files, and handoffs) rather than a
	// fixed fixture — assert the format landed, not a specific outcome that
	// would go stale as the build progresses toward its own handoff.
	if !strings.Contains(out, "derivation: clean") && !strings.Contains(out, "derivation: blocking") {
		t.Errorf("want a live derivation verdict (clean or blocking), out=%q", out)
	}
}
