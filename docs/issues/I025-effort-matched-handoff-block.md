---
id: I025
title: Handoff backstop is presence-only — accept only effort-matched cursor blocks
severity: med
status: open
affects: [audit]
blocked-by: []
execution-mode: subagent-driven
tier: routine
review-tier: routine
---

## Problem

Final-review Important 3 (gen-8 build): any `<!-- spine:cursor -->` block in the newest handoff satisfies the audit-stages backstop — including a stale block from a previous effort. Matches I014's literal wording, but a stale-effort block defeats the intent.

## Fix

Require the newest handoff's cursor block `effort:` to match the live cursor's effort; mismatch = the same blocking finding with a "stale effort" detail. Fixture: handoff carrying a block for a different effort name.
