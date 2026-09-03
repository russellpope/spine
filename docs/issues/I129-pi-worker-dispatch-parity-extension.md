---
id: I129
title: "Pi worker-dispatch parity extension — owner-maintained Pi extension matching OpenCode's constrained, inspectable worker contract"
severity: low
status: open
affects: [I105, I102]
blocked-by: []
execution-mode:
tier:
effort:
risk-triggers: []
review-tier:
---

## Problem

I105's research (`docs/research/2026-08-29-opencode-pi-worker-dispatch.md`)
verified that OpenCode already ships the constrained worker contract a
ladderbench or maipipe worker lane needs, while Pi has only an example
subagent extension. The owner ruled on 2026-09-02: fund the Pi extension
now, so Pi stays a viable second harness if it becomes the chosen one,
rather than deciding the harness by default. Without the controls below a
Pi worker run is not equivalent evidence to an OpenCode run.

## Fix

Build an owner-maintained Pi extension (its own repo or the Pi extensions
tree, not spine) that closes every gap in the research's capability matrix:

1. **Immutable project agent definitions** — a checked-in worker definition
   (fixed prompt, tool allowlist) the run cannot alter; the Pi equivalent of
   OpenCode's `agent.<name>` with `mode: "subagent"`.
2. **Parent allowlist** — which worker names a parent may spawn; the
   equivalent of `permission.task` globs.
3. **Depth counter** — no grandchildren by default (OpenCode
   `subagent_depth` 1).
4. **Job ids and cancellation** — start, inspect, and abort a worker by id.
5. **Persisted parent-child record with normalized usage** — a durable worker
   tree with per-child token accounting, the equivalent of OpenCode's child
   sessions carrying `parent_id`, `agent`, and cumulative token columns.
6. Keep the already-demonstrated model-and-thinking forwarding for an
   unpinned child.

spine's part: once the extension exists, `spine audit routing` must
attribute Pi worker dispatches from the persisted record with the same
`(source, session, dispatch)` correlation it applies to Claude and Codex
(a follow-up ticket here; this one owns the extension contract).

## Acceptance criteria

- [ ] A parity test runs one identical worker instrument through OpenCode and through the Pi extension and diffs the two persisted worker records field by field (worker name, prompt hash, tool allowlist, depth, parent id, usage); the diff is empty apart from harness-native ids.
- [ ] Negative controls: a parent spawning a non-allowlisted worker name is refused; a worker attempting to spawn a grandchild at depth 1 is refused; both refusals appear in the persisted record.
- [ ] Cancelling a running worker by job id stops it and records the cancellation.
- [ ] The extension's install is one documented command and a clean Pi install without it fails the parity test loudly, not silently.

## Related

- I105 (decision record), I102 (audit prompt pairing, unchanged by this ticket).

<!-- Record an approved-without-test exception using the exact grammar in WORKFLOW.md's Acceptance exceptions section. -->
