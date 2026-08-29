---
id: I117
title: "Implement-tick failure with zero evidence blames the tickets value, not the ledger wording that failed the done-word match"
severity: low
status: fixed
batch: 2026-08-28-ergo#2
commits: [6e22c98]
affects: [I019, I032]
blocked-by: []
execution-mode: inline
tier: primary
effort:
risk-triggers: []
review-tier: n/a
---

## Problem

Implement evidence requires a ledger line that starts with the ticket id AND
contains `done`/`complete`/`completed` as a whole word (the I019
word-boundary match). A resolution line like "I110: … declared" is silently
not evidence.

When every anchored ticket misses, the stage-derivation detail appends the
I032 typo hint: `tickets: "…" resolved but every id is missing; check it for
a typo`. That hint assumes the miss means the ids are wrong — but in the
wording case the ids are correct and present in the ledger; what failed is
the done-word requirement. The message sends the operator to audit the
tickets value while the actual fix is one word in a ledger line. Hit live on
2026-08-25 (I110 rollout) and carried since as a handoff gotcha.

## Fix

When zero implement evidence coincides with the anchored ids being *present*
in the ledger (lines starting with the id exist but none carries a done-word),
say so: name the done/complete whole-word requirement instead of, or before,
the typo hint. The typo hint stays for the case where no line starts with the
id at all — the two causes are distinguishable at derivation time and should
get different messages. Tests: a ledger with "I0NN: … declared" and the stage
ticked yields the wording message, not the typo hint; a ledger with no line
for the id at all still yields the typo hint (negative control that the
split is load-bearing).

## Related

- **I019** — introduced the whole-word match (no false-block from negations).
- **I032** — introduced the typo hint whose scope this ticket narrows.
- docs/handoffs/2026-08-25-openweights-docs-and-the-i112-axis-question.md —
  the gotcha entry this ticket retires.

## Resolution

Commit `6e22c98` (2026-08-28 ergonomics batch). `implementAnchoredLines`
records, per anchored id, whether any ledger line starts with `<id>:`
regardless of done-word; `judgeSet` gains an `anchoredNoEvidence`
parameter used only in the zero-evidence ticked-missing branch. Any
anchored line ⇒ the detail names the done/complete/completed whole-word
requirement and the typo hint is suppressed (the ids demonstrably
resolved); no anchored line ⇒ the I032 typo hint verbatim (negative
control observed red on the split); mixed case ⇒ wording message wins
while the missing-ids list still names every id, per the grill's Q3/Q4
rulings. prd/issues judging untouched. Verified live on a scratch repo,
both arms, and exercised for real when this very batch's implement tick
was refused until the ledger carried done-words. Work complete.
