---
id: I077
title: Eval evidence feeding tier equivalence-pin ratification
severity: low
status: fixed
commits: [020caf5, ef23d56, 3640ff5, 07f76dc, 72f5336, c9cae3a, 6a9c39d, e1095f0, 1e6520d, 3b8a07e, 24d677f]
affects: [model, workflow]
blocked-by: [I072]
labels: [wayfinder:task]
parent: I066
assignee:
---

## Question

Pins are owner-ratified judgments of comparability
([I068](I068-host-scoped-availability-and-tier-pins.md)) — but spine already
owns comparability evidence: the `/model-eval` skill + `docs/evals/` convention
+ mutation battery (I053–I056) for local models, and eventually per-model yield
from [I076](I076-routing-yield-review-record-and-yield-verb.md). What is the
lightweight link from evidence to pin — an eval reference recorded alongside a
pin at ratification time? a doctor advisory when a pinned model has no eval
record or a failing battery? — without turning pin ratification into a gate
the owner didn't ask for?

## Owner ruling and accepted design

On 2026-08-30 the owner selected the **BOTH** combined policy: a pin ratification
should carry a repo-local exact-model eval-run reference, and doctor reports
missing, malformed, stale, mismatched, missing-battery, or failing evidence as
a warning only. The reference reader is limited to the audited repository's
`docs/evals/` tree. It never crawls a fleet, follows a symlink, reads a host
home or URL, or mines a transcript. I076 yield evidence is not eligible in
this first policy.

The ruled contract is in the accepted
[I077 design](../specs/2026-08-30-eval-informed-equivalence-pins-design.md)
and its [implementation plan](../specs/2026-08-30-eval-informed-equivalence-pins-plan.md).
They preserve I072 compatibility: missing evidence remains a doctor finding,
not a host-config schema error. No advisory can de-ratify, replace, block, or
gate a pin, or change model or audit command behavior.

Implementation, primary review, and independent verification evidence are
recorded in the resolution below; the batch-final lane remains the shared ship
gate.

## Superseded decision-only brief

The pre-ruling
[`docs/research/2026-08-30-eval-informed-pin-decision-brief.md`](../research/2026-08-30-eval-informed-pin-decision-brief.md)
is retained as the options record. Its wording that no implementation is
authorized until an owner selects an option is superseded by the ruling above.

## Resolution

Fixed 2026-08-30 at final I077 product SHA `24d677f`. Pin records retain
optional repo-local exact-model eval references, while doctor D17 emits only
warning-level missing, malformed, stale, mismatched, missing-battery, or
failing-evidence advisories. Selected reads are descriptor-rooted, bounded,
duplicate-safe, and resistant to symlink and atomic object replacement across
every path component. Evidence health never de-ratifies a pin or changes model,
controlled validation, or audit behavior. A fresh primary review and a
different independent primary verifier passed deterministic and repeated swap
attacks, the exact D16/D17/D7 and redaction matrix, command invariance,
template/update migration, full/race/vet/build, and Windows compile gates. The
single batch-final exact-SHA maipipe lane remains the ship gate.
