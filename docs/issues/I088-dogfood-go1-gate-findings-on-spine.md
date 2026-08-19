---
id: I088
title: "Dogfood: go@1 gate pack findings on spine itself (tskip 4, deferred-cleanup-errcheck 2, dead-code 1)"
severity: med
status: fixed
affects: [I082, I084]
blocked-by: []
execution-mode: inline
tier: primary
effort:
risk-triggers: []
review-tier: n/a
---

## Problem

Baseline run of every go@1 class against spine (2026-08-19, `spine gate go
<class> --dir ~/Projects/github.com/spine`, gen 11 binary) before spine sets
`gate_pack: go@1` on itself:

| class | exit | findings |
|---|---|---|
| tskip | 1 | `cmd/spine/main_test.go:1373`, `internal/cursor/cursor_test.go:192` (dogfood tests skip when the gitignored `.superpowers/sdd/progress.md` is absent); `internal/handoff/handoff_test.go:231` (Windows symlink guard); `internal/scaffold/scaffold_test.go:302` (`t.Skipf` on git failure) |
| deferred-cleanup-errcheck | 1 | `internal/audit/audit.go:1211`, `internal/gate/binaryhygiene.go:108` — `defer f.Close()` on read-only opens |
| dead-code-callgraph | 1 | `internal/model/model.go:428` `model.TierDefaultEffort` — exported helper whose stated consumer (update's D16 effort migration) goes through `MirrorValue` instead; never called |
| test-enum-vs-spec, n-plus-one, gitignore-control, fixture-manifest | 2 | config-driven, `SPINE_GATE_*` unset (expected) |
| binary-hygiene | 0 | clean |

Spine cannot self-enable the pack while its own lane would be red.

## Fix

- handoff_test: drop the Windows guard (no other test guards GOOS; spine
  targets darwin/linux). scaffold_test: `Skipf` → `Fatalf` (every other
  test treats a missing `git` as fatal).
- The two dogfood-ledger skips are intentional environment-conditional skips
  on a gitignored file; express them through the spec'd mechanism —
  `gate_pack_config.tskip_allow` (`cmd/spine/main_test.go:<line>,
  internal/cursor/cursor_test.go:<line>`) — when spine self-enables (I089).
- `defer f.Close()` → deferred func literal that reports the close error
  (audit: append to `warnings`; binaryhygiene: named error return).
- Delete `model.TierDefaultEffort` (keep `tierDefaultEffortOf`).
- Re-run all classes → tskip/errcheck/dead-code exit 0 (tskip with the
  allowlist env set). Negative control: scratch `_test.go` with `t.Skip`
  → finding; remove.

## Resolution (2026-08-19)

- `internal/handoff/handoff_test.go`: Windows guard removed (+ unused `runtime` import).
- `internal/scaffold/scaffold_test.go`: `t.Skipf` → `t.Fatalf` on git failure.
- Dogfood tests moved to `cmd/spine/dogfood_test.go` and
  `internal/cursor/dogfood_test.go`; the skip stays (gitignored ledger) and is
  allowlisted by file in I089 — bare-path entries, no line pins to drift.
- `internal/audit/audit.go` / `internal/gate/binaryhygiene.go`: close errors
  now reported (warnings slice / named error return).
- `model.TierDefaultEffort` deleted.

Evidence: `make test` green (17 pkgs); `go vet ./...` ok;
`spine gate go deferred-cleanup-errcheck|dead-code-callgraph --dir .` → no
findings, exit 0; `SPINE_GATE_TSKIP_ALLOW=cmd/spine/dogfood_test.go,internal/cursor/dogfood_test.go
spine gate go tskip --dir .` → no findings, exit 0. Negative controls:
scratch `_test.go` with `t.Skip` → 1 finding exit 1; scratch `defer f.Close()`
→ 1 finding exit 1; both removed.
