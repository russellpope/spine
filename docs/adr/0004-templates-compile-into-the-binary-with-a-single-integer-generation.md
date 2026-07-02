---
id: 0004
title: Templates compile into the binary with a single integer generation
status: Accepted
date: 2026-07-02
---

# 0004: Templates compile into the binary with a single integer generation

## Context

scaffold.sh read templates from a skill directory, coupling every invocation to a checkout path
and making version skew invisible.

## Decision

Templates embed via go:embed and compile into the binary. templates/VERSION holds a single
monotonic integer stamped into WORKFLOW.md (template_version) and the CLAUDE.md marker block.
Staleness = stamped < compiled; stamped > compiled is a hard error (never downgrade). The one
archived generation is gen0 (pre-48d5960), embedded solely to claim the legacy fleet.

## Consequences

The binary is self-contained — subagents and shims call it from anywhere. Template changes ship
by bumping VERSION and reinstalling; spine update propagates. gen0 was corrected once already
(fixture-driven) to match the generation that actually scaffolded hbmview.
