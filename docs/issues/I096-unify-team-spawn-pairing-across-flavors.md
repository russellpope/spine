---
id: I096
title: "two team-spawn recognizers with divergent worker-prompt pairing (claude teamspawn.go vs codex codex.go)"
severity: low
status: open
affects: [I090, I041]
blocked-by: []
execution-mode:
tier:
effort:
risk-triggers: []
review-tier:
---

## Problem

The audit now recognizes the same herdr/cmux commands twice, in two places
with different pairing semantics:

- `internal/audit/codex.go` (`codexTeamSpawnStartRe` /
  `codexTeamSpawnPromptRe`, ticket I009/I041) pairs by worker handle and
  **accumulates all** of a worker's prompt text.
- `internal/audit/teamspawn.go` (ticket I090) pairs by worker handle and
  takes the **first prompt only** after a spawn, so a later prompt naming
  another ticket does not reattach the worker's evidence.

Both readings are defensible. Differing silently by transcript flavor is
not: the same lead behavior audits differently depending on which harness
recorded it, and neither file's rule is discoverable from the other.

## Fix

Share one worker-keyed pairing across both readers, decide the accumulate-vs-
first-only question once, and record the decision where both can see it. Each
file currently carries a comment cross-referencing the other and this ticket;
those come out when the rule is shared.
