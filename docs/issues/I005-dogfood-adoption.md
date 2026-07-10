---
id: I005
title: Dogfood adoption — spine, deepthought, objectstudio on gen 6
severity: med
status: fixed
affects: []
blocked-by: [I002, I003]
parent: [I001]
labels: [ready-for-agent]
execution-mode: inline
tier: primary
effort: high
risk-triggers: []
review-tier: n/a
---

## Parent

I001 — rollout stage 1 of `docs/specs/2026-07-09-model-routing-design.md` (D10).

## What to build

spine self-updates to generation 6; deepthought and objectstudio adopt via
`spine update` (dry-run diff reviewed with the owner, then write). As a
live smoke of the audit path, `spine audit routing` runs against
objectstudio's real transcript records — the repo whose unrouted build
motivated this work becomes the first real-world audit input.

Inline is the justified exception here: live mutation of three real repos
plus owner judgment on the update diffs.

## Acceptance criteria

- [ ] spine, deepthought, objectstudio at template generation 6; `spine doctor` clean (info-level legacy notes acceptable)
- [ ] Owner reviewed each update diff before write
- [ ] `spine audit routing` produces a real report against objectstudio's transcripts (expected: unannotated/no-transcript verdicts — the point is the path runs live)

## Blocked by

- I002, I003 — adoption needs the command and the generation to exist.
