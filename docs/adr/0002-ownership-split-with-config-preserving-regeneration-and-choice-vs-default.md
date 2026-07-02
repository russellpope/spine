---
id: 0002
title: Ownership split with config-preserving regeneration and choice-vs-default
status: Accepted
date: 2026-07-02
---

# 0002: Ownership split with config-preserving regeneration and choice-vs-default

## Context

Machine-owned workflow files (WORKFLOW.md, harness-interface.md, issues README/_template,
adr README) are ~100% template content with a handful of config values; CLAUDE.md accumulates
hand-written invariants. hbmview proved templates strand adopters without an upgrade path.

## Decision

Ownership split: machine-owned files are regenerated wholesale from the compiled templates with
extracted config keys reapplied; CLAUDE.md gets a spine-managed marker block, user content below is
never touched. Choice-vs-default: an extracted value equal to its own generation's rendered default
is not a user choice and takes the new default; only divergent values survive. Unrecognized local
edits skip the file unless --force, with the diff showing exactly what would drop.

## Consequences

spine update un-strands legacy repos (proven live on hbmview) without archiving template history
(one embedded gen0 exception). Hand-edits to machine-owned prose do not survive regeneration — they
are surfaced, not merged. Mixed-generation repos resolve conservatively (skip + warn, converge via
--force).
