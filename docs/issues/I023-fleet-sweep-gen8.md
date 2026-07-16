---
id: I023
title: Fleet sweep — all 17 repos to gen 8, ultima last
severity: med
status: open
affects: [fleet]
blocked-by: [I019, I020]
execution-mode: subagent-driven
tier: routine
review-tier: routine
---

## What to build

Plan Task 6. Rebuild/install the spine binary, then per repo (enumerate live; 17 expected per the 2026-07-15 audit): `spine update` dry-run → review diff + unrecognized report → `spine update --write` → `spine doctor` → commit with a uniform message. Dormant gen-5 notes repos first, active repos next, **ultima-dci-edition last** via the supersededLines path — `--force` only as a flagged fallback with the dry-run diff preserved for owner review. A repo that fails to update cleanly is reported and skipped, never forced silently.

## Acceptance criteria

- [ ] Every cleanly-updated repo shows `template_version: 8` and a passing `spine doctor`
- [ ] Ultima updated WITHOUT `--force` (or the forced diff + justification flagged for the owner)
- [ ] Per-repo results table (gen before/after, state, commit) appended to the build ledger
- [ ] No repo force-updated silently; failures listed for morning review
