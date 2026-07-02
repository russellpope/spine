---
id: 0005
title: Pre-spine ADRs report as info and info findings do not fail doctor
status: Accepted
date: 2026-07-02
---

# 0005: Pre-spine ADRs report as info and info findings do not fail doctor

## Context

The fleet holds ~135 hand-rolled pre-spine ADRs with no YAML front matter. The spec's three
requirements — never mass-migrate legacy artifacts, D6 status validation, clean doctor on the
acceptance target — were mutually inconsistent: hbmview's own ADRs failed D6 as 'invalid status ""'.

## Decision

Pre-spine ADRs (no front-matter block) yield a D6 info finding, not a warn: spine conventions
apply to new ADRs only. Doctor's exit code ignores info findings — exit 1 requires warn or error.
Supersede refuses pre-spine targets (no front-matter status) rather than guessing.

## Consequences

Doctor stays honest (findings still print and appear in --json) while legacy fleets pass. The
acceptance bar 'clean doctor on hbmview' holds without migrating history. New ADRs get full
validation.
