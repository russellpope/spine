---
id: I102
title: "two team-spawn recognizers with divergent worker-prompt pairing (claude teamspawn.go vs codex codex.go)"
severity: low
status: fixed
commits: [35808b3]
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

## Resolution

Fixed 2026-08-29 in `35808b3`. Both transcript readers now use the shared
`firstTeamPrompt` pairing rule in `internal/audit/teamspawn.go`: a worker's
first prompt after its spawn is the assignment, while later prompts are
follow-up conversation and cannot attach that worker's model evidence to a
different ticket. This chooses first-only based on I090's existing accepted
misattribution guard: its `I603` fixture already requires a second prompt to
remain `no-transcript`. The Codex reader's new end-to-end regression pins the
same outcome; its pre-fix run reported `I904` as `escalated-no-reason` rather
than `no-transcript`.

Focused cross-reader pairing tests, `go test ./internal/audit -count=1`, and
`go test ./... -count=1` were run with repository-local Go caches; `git diff
--check` passed.
