---
id: 0008
title: adopt composes init and update; no mapping, no migration
status: Accepted
date: 2026-07-02
---

# 0008: adopt composes init and update; no mapping, no migration

## Context

The fleet has pre-spine repos in several different legacy shapes (praxis, home-lab-admin,
obsidian-ep-vault, moo-clone, and others), and the goal is to bring them onto the current
template without hand-migration or a per-repo mapping table. `spine init` only works on repos
with nothing there yet, and `spine update` only works on repos that already have a WORKFLOW.md;
neither covers "detect what this repo is and stand up the workflow files for the first time."

## Decision

`spine adopt` composes the two: it runs profile detection, creates the profile's directories
via `ProfileDirs`/`MkdirAll`, and then invokes `update.Run` in adopt-mode, all under a single
dry-runnable plan (default dry-run, `--write` to apply — same contract as `update`). There is no
mapping table from legacy file/dir names to spine's conventions, and no migration of legacy
content: pre-existing non-spine artifacts (old ADRs, old plan docs) stay exactly where they are
and report as info findings, per the ADR 0005 / D5 pattern already established for pre-spine
ADRs.

## Consequences

Every repo in the fleet converges on one shape — the current template — regardless of what it
looked like before, without spine ever needing repo-specific knowledge baked into its code. The
proof obligation is doctor-clean immediately after adopt (info-only findings allowed, no warn/
error) and a no-op `update` run afterward; this is mechanically proven in Task 13's real-file
fixtures across all four acceptance targets (praxis/go-service, home-lab-admin/infra,
obsidian-ep-vault/knowledge, moo-clone/swift), not just asserted.
