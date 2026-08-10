# Routing-yield feedback (I052) — feasibility of per-(flavor, tier) yield from conveyor artifacts

**Date:** 2026-08-05 · **Kind:** feasibility note (charter I052) ·
**Related:** [I052](../issues/I052-routing-yield-feedback-charter.md),
[CONTEXT.md model routing](../../CONTEXT.md), WORKFLOW.md record grammar (lines 49–51),
maipipe `docs/research/2026-08-03-agentflow-steal-list.md` (provenance)

Sources: empirical survey 2026-08-05 of `spine:.superpowers/sdd/` (265 entries; dispatch
triples I033–I049; five archived progress ledgers I002–I032) and
`maipipe:.superpowers/sdd/` (381 top-level entries + 7 effort subdirs; `progress.md`
~1,200 lines spanning I017–I167), ticket frontmatter in both `docs/issues/` ledgers
(spine 54 files, maipipe 174), `spine` source (`cmd/spine/main.go`).

## Question

Can accepted-first-pass rate, rework rounds, and escalation frequency per
(flavor, tier) be harvested from artifacts the SDD conveyor already leaves — or does the
derivation need a new record written at review time?

## Requirements-attack: charter vs. observed artifacts

Four frictions in the charter's premises, each with a proposed resolution — none
silently resolved:

1. **"dispatch/review/rereview files" is not a stable convention.** The triple
   (`dispatch-I0NN-implementer/-reviewer/-rereview.md`) holds only for spine's latest
   effort. maipipe's codex-team era uses `-worker/-review/-fix/-confirmation/
   -integration-rebase`; claude batch efforts use `task-N-brief/report`; I132's re-reviews
   are eight SHA-keyed files (`review-I132-<sha>-dispatch.md`). Even inside spine,
   I039's rework hides behind `dispatch-I039-re-review.md` and
   `fixround1-I039-objectstudio.md` — a filename glob for `-rereview` misses it.
   *Resolution:* the substrate must be progress-ledger records, never filenames.
2. **"ticket status history" does not exist as an artifact.** Frontmatter carries a
   current `status:`, not a history; only `git log` of the issue file approximates one,
   and it records edit times, not acceptance rounds. *Resolution:* drop it as a source.
3. **Flavor attribution contradicts the estate's own rule.** CONTEXT.md (2026-07-24):
   "Artifacts never name a flavor; the dispatcher supplies it." Accordingly no dispatch
   file or ticket frontmatter sampled names a flavor or model. Flavor is only
   recoverable from model ids in ledger lines (`gpt-5.6-sol` → codex,
   `claude-sonnet-5` → claude) or audit actuals. *Resolution:* any new record must carry
   flavor explicitly; retrospective flavor comes from model ids where present.
4. **"Accepted first pass" at the task gate is a flattering metric.** Spine's own
   archived ledger notes the "fifth consecutive build where final review out-caught task
   gates," and final-review conditions land as cross-ticket fix waves (e.g. the
   C1a/C1b/C2/M8 wave) attributable to no single ticket. A task-gate-only metric will
   overstate yield. *Resolution:* scope the metric to the task gate explicitly and count
   final-review conditions as a separate, possibly unattributed, series.

## Findings

### 1. Is "accepted first pass" derivable from existing files? — No, not reliably

In spine's surviving effort (16 tickets with reviewer dispatches, I033–I049), the
`-rereview.md` glob finds 11 reworked tickets and misses I039 (naming drift, above) —
a demonstrated false negative in a 16-ticket sample. Absence is also not acceptance:
I035's review is "Approved — 0 Critical, 0 Important, 5 Minor" with the minors deferred
to final review, where they became part of a cross-ticket fix wave. So rereview-file
absence conflates true first-pass acceptance, deferred-minor acceptance, and rework
that happened under another filename or at final review. In maipipe the question cannot
even be posed via filenames — the conventions differ per era (finding 1 above). The one
trustworthy source is maipipe's codex-team ledger grammar in `progress.md`
(`— model: …; tier: …; ticket-id: …; status: NEEDS_FIXES/READY`), which is exactly a
review-time record — evidence that the record, not the file tree, is the right substrate.
Verdict: **needs a new (or standardized) record at review time.**

### 2. Can each dispatch be attributed to (flavor, tier)? — Tier partially, flavor only via records

- **Dispatch files:** name neither model nor tier (sampled
  `spine:dispatch-I042-implementer.md`, `maipipe:dispatch-I122-worker.md`; the latter
  hints codex only via the `codex-i122` branch name — too weak to key on).
- **Ticket frontmatter:** declared tier exists for spine 44/54 tickets (24 routine,
  9 primary, 7 mechanical) and maipipe 127/174 (80 routine, 42 primary, 2 mechanical) —
  but declared ≠ dispatched (see ESCALATION below), and blank-valued `tier:` fields
  occur in both.
- **Progress ledgers:** spine's carry prose like "implementer dispatched
  (sonnet=routine per ESCALATION record above)" — mineable but irregular. maipipe's
  codex-era lines carry `model` + `tier` + `ticket-id` per dispatch — fully attributable.
- **`spine audit routing`:** yields actuals, but only where transcripts survive and are
  readable; maipipe's own ledger records `no-transcript` for all cmux dispatches of the
  I060–I074 wedge, and audit output is not persisted (ad hoc `.txt` files only).

### 3. What do ESCALATION/FALLBACK records add? — The effective tier, and the escalation metric itself

The grammar (`WORKFLOW.md:49–51`) is the only place the *dispatched* tier diverges from
the annotated one with a reason. Spine's ledgers hold ~25 ESCALATION records, maipipe's
~25 plus 3 FALLBACK. Two consequences: (a) yield keyed on frontmatter tier without
folding in ESCALATION records mis-bins dispatches — e.g. `ESCALATION I018
primary->routine` (implementers actually ran routine) and `ESCALATION I024/I027/I029
mechanical->routine`; (b) escalation frequency, the charter's third metric, is **already
fully derivable** from these records — ticket, direction, reason — with no new artifact.
A sobering corollary: the mechanical cell is nearly empty of actual dispatches (its
tickets get escalated to routine), and all three FALLBACK records are
"not dispatchable" notes — the two cells the routing table most needs feedback on have
almost no outcome data and will accumulate it slowest.

### 4. Sample sizes per (flavor, tier) — no cell supports a rate today

Reliably attributable task-gate outcomes across both repos:

| Cell | n (approx) | If observed 80% accepted, Wilson 95% CI |
|---|---|---|
| (codex, routine) | ~17 | 14/17 ≈ 82% → **58–94%** |
| (codex, primary) | ~13 | 10/13 ≈ 77% → 50–92% |
| (claude, routine) | ~10–15 (spine I033–I049 + mineable maipipe prose) | 8/10 → 49–94% |
| (claude, primary) | ~5 | uninformative |
| (·, mechanical), fallback | ~0–2 actual dispatches | none |

The routing question ("does routine yield justify the down-route?") needs to
distinguish roughly ≥85% from ≤65%; that takes n≈30–40 per cell, and ±10pp precision at
80% takes n≈60. The largest cell is at half the minimum, and its CI spans "fine" to
"unacceptable." Retrospective mining cannot reach the floor — the older artifacts that
would grow n are exactly the ones with unmineable naming (finding 1). Forward
accumulation can: the estate runs ~10–20 reviewed dispatches per week, so the two big
cells cross n=30 within a month or two of record-writing.

### 5. Where should the view live?

A `spine` verb with `--fleet`, exactly the `handoff latest --fleet DIR` shape
(`cmd/spine/main.go:349`). Per-repo cells are a third the size of estate cells, so a
repo-only view would near-always print "below floor"; fleet aggregation is what makes
the floor reachable at all. Outside spine is wrong: the parser for ESCALATION/FALLBACK
grammar and the tier vocabulary already live here (`internal/audit`, `internal/model`).

## Recommendation: build-differently

**Do not build** the charter's implied retrospective harvester over
dispatch/review/rereview files — filename derivation has demonstrated false negatives,
flavor is structurally absent from files, and no historical cell clears an honest
sample floor.

**Do build** the forward-looking half: extend the existing record grammar with one
review-time line, have spine parse *only records*, and ship `spine yield` (per-repo +
`--fleet`) that always prints counts and refuses rates below a floor (suggest n≥20
"low-confidence", n≥40 for a stated rate; never a bare percentage).

Minimal record the conveyor writes to the progress ledger at each review verdict:

    REVIEW <ticket-id> <flavor>/<effective-implementer-tier> round=<n> verdict=accepted|needs-fixes scope=task|final

- `flavor` explicit (resolves finding 3); `effective-implementer-tier` = annotation
  after any ESCALATION record (resolves finding 2/3 mis-binning).
- `round` increments per re-review of the same ticket; accepted-first-pass =
  `round=1 verdict=accepted`; rework rounds = max round.
- `scope=final` lines carry final-review conditions per affected ticket (ticket-id `-`
  when unattributable), keeping the task-gate metric honest (attack-finding 4).
- Escalation frequency needs no new record — derive from existing ESCALATION/FALLBACK
  lines.

maipipe's codex-team ledger grammar is the existence proof that orchestrators will
write such lines; this proposal is a standardization of what the best-instrumented
effort already does, not a new burden.
