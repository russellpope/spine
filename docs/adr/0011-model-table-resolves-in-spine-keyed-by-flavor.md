---
id: "0011"
title: "Model table resolves in spine, keyed by flavor"
status: Accepted
date: 2026-07-24
supersedes: "0010"
---

# 0011: Model table resolves in spine, keyed by flavor

## Context

ADR 0010 settled two things: artifacts name **tiers, never model ids**, and the
tier→id mapping lives in each repo's scaffolded `WORKFLOW.md`, per-repo
remappable. The first has held and is the reason no plan or ticket in the estate
is coupled to a model family. The second has not.

Three failures accumulated against the placement half. **It cannot migrate.** The
choice-vs-default rule of ADR 0002 classifies any on-disk value differing from the
current template default as a deliberate per-repo choice and carries it forward;
verified empirically 2026-07-24 with a probe that returned a changed fallback
value as a choice. Changing a model id in the template therefore propagates
nothing, and each repo's stale value silently wins. Generations 5→6 never exposed
this because they changed comments and structure, never a value. **It covers only
half the estate.** Codex model ids live in team-skill prose, where nothing
validates them — a pinned worker model flattened per-tier routing in the field, so
every worker ran the routine-tier model regardless of its ticket's tier. **It is
parsed twice.** `spine audit routing` re-parses `WORKFLOW.md` with an independent
parser that has no flavor dimension and no generation gate, so codex work cannot be
audited at all and an un-upgraded binary emits confident verdicts from a misparse.

Underneath is an economic need the placement cannot serve: choosing per project
which real model backs the primary tier — the most capable model for genuinely hard
work, a cheaper capable one otherwise — without editing seventeen repos.

## Decision

**Tiers, not ids — unchanged and restated.** Every artifact that names routing —
ticket frontmatter, ESCALATION/FALLBACK records, plan prose — references one of the
four semantic tiers, never a model id. This half of ADR 0010 is preserved in full.
This ADR supersedes 0010 only because the convention requires whole-file
supersession; it is a refinement of the placement clause, **not** a reversal of tier
indirection.

**The mapping resolves in spine, keyed by flavor.** Spine owns the tier→id table as
versioned data embedded at build time, keyed by `(flavor, tier)` where **flavor** is
the agent runtime — `claude` or `codex` — and each entry carries a model id plus an
optional effort. The defaults file records every default ever shipped, which is what
makes an inherited value distinguishable from a deliberate override by direct lookup
rather than by re-rendering a template. Each repo's `WORKFLOW.md` keeps a rendered
mirror marked spine-managed; editing a value in place makes that entry an override.
A single resolver serves both the `spine model` command and the routing audit, so
what is dispatched and what is verified cannot disagree.

**Tickets stay flavor-neutral; the dispatcher supplies flavor.** Adding a flavor axis
does not push flavor into artifacts. A ticket declares a tier and nothing else, and the
same ticket may be executed by either flavor. The dispatching skill knows its own
flavor by construction and supplies it at resolution time. ADR 0010's principle —
nothing downstream of a ticket needs to know which vendor or model family a tier
points at — is strengthened rather than broken, because the id now resolves further
from the artifact than before.

**Estate-owned placement, unchanged.** The contract still lives only in surfaces the
estate controls — the spine binary, its embedded templates and data, and local
skills. None of it is patched into the upstream plugin cache.

## Consequences

- **Hard to reverse.** Once the fleet is swept to the flavor-aware mirror and the team
  skills resolve through the CLI, reverting means restoring a second parser in the
  audit, re-materializing ids into seventeen repos, and re-pinning literals into skill
  prose — reintroducing by hand the exact flattening bug this removes.
- **Surprising without context.** The instinct on reading a `WORKFLOW.md` is that its
  values are the configuration. After this ADR they are a *mirror*: authoritative only
  where they differ from a shipped default, and refreshed automatically where they do
  not. A reader who edits one expecting it to be read verbatim is right only by
  accident; the marking comment exists to say so.
- **Real trade-off.** The estate gains a build-time dependency for a model change (edit
  the data file, rebuild, sweep) and the team skills gain a hard dependency on the spine
  binary being installed, refusing early when it is not. In exchange, a model release
  stops being a seventeen-repo migration, the codex half of the routing table becomes
  expressible and auditable for the first time, and the choice-vs-default trap that made
  value migration impossible is removed structurally rather than worked around.
- **A machine-local config layer was rejected**, not forgotten. It would let a model
  change skip the rebuild, but an uncommitted layer makes fleet state unauditable from
  git. The precedence chain is ordered so one can be inserted later without disturbing
  either end.
- **Drift remains unaddressed and is deliberately out of scope.** A table declares
  intent; a runtime that silently moves a session to another model overrides intent
  after the fact. The audit's verdict vocabulary is left open for a drift verdict
  distinct from dispatcher-caused silent descent.
