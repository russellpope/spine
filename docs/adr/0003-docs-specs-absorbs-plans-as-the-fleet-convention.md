---
id: 0003
title: docs specs absorbs plans as the fleet convention
status: Accepted
date: 2026-07-02
---

# 0003: docs specs absorbs plans as the fleet convention

## Context

Five repos + deepthought write specs/plans to docs/superpowers/{specs,plans} (skill defaults)
while the scaffold said docs/specs/ — two competing homes for the same artifact class, and every
future tool would need two code paths.

## Decision

docs/specs/ absorbs plans fleet-wide: <date>-<topic>-design.md + -plan.md pairs, PRDs alongside.
Templates and doctor (D5) steer superpowers skills there via the CLAUDE.md header. Existing
docs/superpowers/ trees are never mass-moved.

## Consequences

One tree for the workflow's primary artifacts, colocated with docs/{adr,issues,handoffs}. D5
nudges (info) appear on repos still accumulating artifacts in the legacy location.
