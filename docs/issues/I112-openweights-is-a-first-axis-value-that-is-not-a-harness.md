---
id: I112
title: "openweights is a model-table first-axis value that is not an execution vehicle, contradicting the ratified harness definition"
severity: low
status: open
affects: [I110, I111]
blocked-by: []
execution-mode:
tier:
effort:
risk-triggers: [plan-flagged-ambiguity]
review-tier:
---

## Problem

This is a domain-model question, not a defect. Nothing is broken: `spine model
openweights <tier>` resolves correctly and dispatch works. But the shipped table
and the ratified glossary now disagree, and that disagreement is load-bearing
for I111.

CONTEXT.md ratified **harness** (I067, 2026-08-10) as the model table's first
axis, defined as *"the execution vehicle that runs a dispatch"*. The same entry
is explicit about the boundary:

> Reachability of a model from a given host is a separate, per-host constraint
> (I068/I072), **never a harness**. _Avoid_: … "local flavor" — local is a
> property of where a model is served, not a harness.

`openweights` (I110) fails that definition. An openweights dispatch runs the
**ordinary Claude Code binary**, via a wrapper that passes `--model` through.
The downstream design says so in as many words: *"`claude-auto` is not a
separate runtime… an openweights team is therefore a claude-runtime team on
different models."* It is the claude harness pointed at other models — which is
to say, a property of where the models are served. Precisely the thing the
glossary says is never a harness.

`pi` is not a precedent for this. `pi` is its own driver binary, so it *is* an
execution vehicle; that it also serves open weights is incidental.

## Why it matters

The clearest evidence is that **I111 has to exist at all.** Open-weights
sessions land in `~/.claude/projects` *because they are Claude Code sessions*.
Deriving the first axis from the transcript source therefore misfiles every one
of them, and the fix is to derive it from the observed model id instead. Having
to abandon source-derivation for one value of an axis is a strong signal that
the value does not belong on that axis.

Second-order: the spec's story 5 requires open-weights ids stay disjoint from
every other harness's ids, because that disjointness is what makes id-derived
resolution unambiguous. That requirement exists only because two different
"harnesses" now share one execution vehicle. If the axis were modelled as
(harness, model-family), the ambiguity would not arise and the disjointness
constraint would be unnecessary rather than merely unenforced (see I111's
inherited guard).

## Options

Not ranked — this is an owner decision, and each has a real cost.

1. **Ratify the widened definition.** Amend CONTEXT.md so the first axis is
   "routing family" rather than "execution vehicle", and accept that one value
   can share another's runtime. Cheapest; costs the crispness that made the
   I067 definition useful, and leaves the I073 flavor→harness migration
   describing something other than what it set out to describe.
2. **Rename the value** to something that does not claim to be a harness while
   leaving the axis definition intact. Cosmetic, and arguably dishonest — the
   axis would still hold a non-harness.
3. **Split the axis into (harness, model-family).** Most faithful to the
   ratified model and would dissolve I111's disjointness requirement outright.
   Also by far the most invasive: it changes the model table's shape, every
   repo's mirror, `spine model`'s signature, and the audit. Almost certainly
   not worth it for one value.

## Constraint

Whatever is chosen, do not silently drop the I067 boundary sentence. If option
1 wins, the "never a harness" clause must be rewritten deliberately and dated,
not deleted — it was ratified for a reason (I068/I072 host reachability), and
that reason still holds for the case it was written about.

## Related

- **I110** shipped the value; its Resolution records what was verified.
- **I111** is where the tension becomes operational — its whole premise is that
  the first axis cannot be derived from the transcript source for this value.
- CONTEXT.md `harness` and `flavor` entries carry the same note, so a reader of
  the glossary finds this ticket rather than an unexplained inconsistency.
- ADR 0011 (model table resolves in spine, keyed by flavor) is the decision
  this would amend if option 1 or 3 is taken.
