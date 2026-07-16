---
id: I019
title: Derivation engine + spine audit stages + doctor advisory check
severity: med
status: open
affects: [audit, doctor, cursor]
blocked-by: [I018]
execution-mode: subagent-driven
tier: routine
risk-triggers: [cross-task-integration]
review-tier: primary
---

## What to build

Plan Task 2. The shared stage-derivation engine judging the cursor's anchored artifacts bidirectionally (ticked-but-missing blocks; present-but-unticked blocks; conservative implement heuristic — absence never blocks); `spine audit stages` (table output, non-zero exit on blocking findings, warn+exit-0 when no progress.md); newest-handoff cursor-block check (blocking in audit, advisory in doctor); doctor advisory D-check on the same engine; wire the real verdict into `spine cursor`.

## Acceptance criteria

- [ ] Fixture matrix from the design's Testing Decisions all pass: clean/ticked-missing/present-unticked/no-ledger-warn/handoff-missing-block
- [ ] `spine audit stages` exit codes: non-zero only on blocking findings
- [ ] Doctor check severity is `warn`, never `error`; doctor exit behavior unchanged
- [ ] `spine cursor` prints the live verdict, still always exit 0
- [ ] `go test ./...` green
