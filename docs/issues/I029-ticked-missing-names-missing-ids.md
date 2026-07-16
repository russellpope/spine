---
id: I029
title: ticked-missing detail doesn't name missing ticket ids or hint at a tickets: typo
severity: low
status: fixed
affects: [I026]
blocked-by: []
execution-mode: subagent-driven
tier: mechanical
review-tier: routine
---

## Problem

A resolvable-but-wrong `tickets:` range (e.g. `I01-I04` typo for `I001-I004`, or any range/prefix that resolves to real-looking ids none of which happen to exist) blocks `spine audit stages` as `ticked-missing — marked done but 4/4 ticket file(s) missing` without naming which ticket ids were expected or hinting that the `tickets:` value itself might be the typo. I026 already gives grammar discoverability to the *unresolvable* class (a Notes entry names the bad value verbatim) — this is the adjacent gap: the *resolvable-but-wrong* class degrades to an opaque count with no id list and no pointer back at `tickets:`.

Scope note: the final whole-branch review (I024-I027 batch) that raised this also asked whether `spine cursor`/doctor should be extended to surface I026's unresolvable-tickets Notes entry, gating the extension on it being a clean ≤5-line addition following the existing D9 pattern. That extension turned out to be exactly that (`spine cursor` now prints `warning: <note>` lines for `rep.Notes`, and doctor's D9 check now emits one `Finding` per `rep.Notes` entry) and shipped as part of that review's fix wave, not skipped — nothing carried over into this ticket from that side. This ticket is the separate, still-open gap: the *resolvable* wrong-value case's detail message.

## Fix

In `internal/stages` `judgeSet` (or wherever the `ticked-missing` detail string for the issues/implement stages is built): when the verdict is `VerdictTickedMissing`, name the missing ticket ids in the detail — first few ids plus a "+N more" count for long sets, rather than just the raw missing/total count. When ALL resolved ids in the set are missing (0 present out of N), also mention the live `tickets:` value in the detail, since an all-missing set from a resolvable range/prefix is the shape a typo produces (e.g. `I01-I04` resolving to a numerically-valid-but-wrong range) and the reader should be pointed at the most likely cause. Add coverage alongside the existing `internal/stages` ticked-missing tests (see `TestUnresolvableTicketsNeverBlocks` and neighbors in `stages_test.go` for the established fixture/assertion style).

## Resolution

Fixed 26de369 (gen9-sweep-i029-i030 batch, shipped 2026-07-16). `judgeSet` gained `ids`/`ticketsRaw` params;
ticked-missing details now name the missing ids (first 5 + "+N more"), and an all-missing set from a resolvable
`tickets:` value appends the typo hint naming the live value. TDD (3 new tests), per-task review Approved with
0 findings, final review READY TO MERGE. Follow-up [I032](I032-implement-row-typo-hint-scope.md): scope the
typo hint to the issues row (it also fires on implement's all-missing evidence set, where the issues row can
prove `tickets:` correct) + decouple the truncation test from the cap constant.
