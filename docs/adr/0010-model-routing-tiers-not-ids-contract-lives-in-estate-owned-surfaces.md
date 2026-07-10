---
id: "0010"
title: "Model routing: tiers not ids, contract lives in estate-owned surfaces"
status: Accepted
date: 2026-07-10
---

# 0010: Model routing: tiers not ids, contract lives in estate-owned surfaces

## Context

`docs/specs/2026-07-09-model-routing-design.md` (D1–D10) makes model routing a
first-class, enforced contract: every ticket declares an execution mode and a
model tier, the scaffolded WORKFLOW.md carries the dispatch rules, and
`spine audit routing` verifies declared tier against actual model from
transcripts. Two placement questions had to be settled before gen 6 could
ship: what a ticket or plan is allowed to *name* (a tier, or a model id
directly), and where the contract text itself is allowed to *live* (an
estate-owned surface, or patched into the upstream superpowers plugin the
estate does not control).

## Decision

**Tiers, not ids.** Every artifact that names routing — ticket frontmatter
(`tier:`), the ledger's ESCALATION/FALLBACK records, plan prose — references
one of the four semantic tiers (`primary` / `routine` / `mechanical` /
`fallback`), never a model id. The tier→id mapping lives once, in each
repo's scaffolded `WORKFLOW.md` `model_routing` block, and is per-repo
remappable. `spine audit routing` resolves ids back to tiers from that
mapping at verification time; nothing downstream of a ticket ever needs to
know which vendor or model family a tier currently points at.

**Estate-owned contract placement.** The dispatch contract itself — the
WORKFLOW.md template text, the tier vocabulary, the audit's parsing rules,
and any local skill behavior that assigns tiers — lives only in surfaces the
estate controls: the spine binary's compiled templates and local skills.
None of it is patched into the upstream superpowers plugin cache. Upstream
already mandates explicit model choice at dispatch; the estate contract
supplies the vocabulary and values upstream lacks, layered on top rather
than forked into it. Filing first-class per-task model/mode fields upstream
remains optional, and gated on the owner's explicit word.

## Consequences

- **Hard to reverse.** Once a fleet sweep propagates gen 6 and tickets,
  plans, and ledger records accumulate tier-only references, reversing to
  id-based routing means rewriting the audit's parsing contract, every
  template consumer, and retroactively translating a growing body of
  historical tickets — a real migration, not a flag flip.
- **Surprising.** The lower-friction instinct is to name a model id directly
  in WORKFLOW.md and tickets, as gen ≤5 effectively did with `primary:
  claude-fable-5` read as configuration rather than as a value behind a
  named role. The tier indirection is a layer that costs something to hold
  in your head at dispatch time and pays for itself only on the day the
  estate needs to remap — a new model release, a local model, another
  provider — which is invisible until that day arrives.
- **Real trade-off.** Every dispatch now requires translating a tier to an
  id via WORKFLOW.md instead of reading a model name straight off the
  ticket; in exchange, no historical plan or ticket is ever coupled to a
  specific vendor's model naming, and the contract's blast radius on a
  plugin update is confined to the mapping and audit code spine owns —
  placing it inside the upstream-managed plugin cache would risk silent
  overwrite the next time that plugin updates.
