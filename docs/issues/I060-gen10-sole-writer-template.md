---
id: I060
title: "Cursor writes: gen 10 template — sole-writer rule text, embed rule superseded"
severity: med
status: fixed
affects: [templates, update, workflow-md]
blocked-by: [I057, I058]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: [cross-task-integration]
review-tier: primary
---

## Parent

PRD: docs/specs/2026-08-06-cursor-writes-design.md (sequencing note: the text
mandates tooling that must already work — hence blocked by I057/I058).

## What to build

Template generation 10: the machine-owned WORKFLOW.md stage-cursor section
declares the sole-writer rule (mutate only via the cursor verbs; hand-editing
is a workflow violation) and the automatic-embed rule for handoffs. The
existing handoff-rule line ("MUST embed the verbatim output of `spine
cursor`") is **superseded — replaced, never duplicated**: grill flagged that
leaving it alongside would have the fleet declare two contradictory
procedures.

- Gen 9→10 migration in the per-generation update seam, including the
  supersededLines reconciliation for the replaced handoff-rule text.
- `spine init` / `spine adopt` seed the gen 10 text for new repos.
- No cursor grammar change rides along.

## Acceptance criteria

- [x] Gen 9→10 migration test proves the old embed-verbatim line is gone and
      the sole-writer + auto-embed text is present — replaced, not duplicated
- [x] A gen 9 repo with hand-customized surrounding sections upgrades cleanly
      via plain `spine update --write` (prior-art fixture pattern)
- [x] `spine version` reports gen 10; init/adopt emit gen 10 text
- [x] `go test ./...` green

## Blocked by

- [I057] — the verbs the text mandates must exist.
- [I058] — the automatic embed the text describes must exist.

## Resolution

Fixed in `a080f31` and swept into spine at `cef7166`. Generation 10 replaces
the superseded manual-copy rule with one sole-writer rule and one automatic
handoff rule, preserves customized surrounding configuration, and is covered
through update, init, adopt, and version tests.
