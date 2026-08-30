---
id: "0022"
title: "Model ids derive audit flavor; transcript source breaks ties"
status: Accepted
date: 2026-08-29
---

# 0022: Model ids derive audit flavor; transcript source breaks ties

## Context

Design D15 made transcript source the routing audit's flavor selector. That
worked while Claude-layout transcripts contained only Claude models and Codex
had a separate session store. I110 added `openweights`, whose models run
through the ordinary Claude CLI and therefore share Claude's transcript
layout. Source-derived flavor would judge every such dispatch against the
Claude table and report it as unmapped.

Source also controls behavior unrelated to model resolution. D28 applies repo
qualification to records from the Claude layout, while Codex source records
arrive repo-scoped. Codex verdict details also name their source file. Treating
source and flavor as one value made those rules fragile once a Claude-layout
record could carry an openweights flavor.

## Decision

Extend D15. For each observed model token, the routing audit searches every
resolved flavor table, including current ids, aliases, and shipped historical
ids.

- A token found under exactly one flavor uses that flavor.
- A token found under more than one flavor uses the transcript-derived flavor.
  This retains D15's source tiebreaker.
- A token found under no flavor keeps the transcript-derived flavor, preserving
  the prior unmapped behavior and detail.

Transcript source and model flavor are separate axes. Flavor selects the model
table used for routing judgment. Source controls transcript-layout rules,
including D28 qualification, Codex case folding, and source-file disclosure.

The shipped model table must keep ids, aliases, and historical ids disjoint
across flavors. That invariant makes openweights derivation unambiguous. If a
future openweights tier points at a `claude-*` id, the invariant breaks and the
source tiebreaker becomes load-bearing for that token.

## Consequences

- Claude-layout transcripts may mix Claude and openweights records. The audit
  judges each token against its own resolved flavor table.
- Openweights records still pass through D28 because D28 keys on their Claude
  transcript source, not their model flavor.
- Deliberate repo overrides can create a cross-flavor collision. The retained
  source tiebreaker gives those records the same reading D15 gave them before.
- A table-level regression test rejects cross-flavor token collisions in the
  shipped defaults. Another test drives default-resolution coverage from
  `Flavors()` so a new flavor cannot ship without tier assertions.

## Related

- I111, which implements this decision.
- I110, which added the openweights flavor and its disjoint model ids.
- D15, extended here rather than replaced.
- D28 and I047, whose repo-qualification rule remains source-dependent.
