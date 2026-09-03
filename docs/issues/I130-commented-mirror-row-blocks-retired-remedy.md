---
id: I130
title: "A trailing comment on a model_routing mirror row makes spine update skip WORKFLOW.md, so the retired-model remedy loops"
severity: low
status: open
affects: [I128, I036]
blocked-by: []
execution-mode:
tier:
effort:
risk-triggers: []
review-tier:
---

## Problem

Found by the I128 correctness review (2026-09-03) and reproduced with the
I128 binary. A mirror row spelled with a trailing comment, for example
`claude.primary: claude-fable-5 @ xhigh   # pinned by owner`, is read by
the resolver (comments are stripped per `CommentIndex`) but `spine update`
classifies the line as an unrecognized local edit and skips WORKFLOW.md:

    skipped WORKFLOW.md — unrecognized local edits (use --force to drop):

Launch validation refuses the row as `retired-model` and prints the
`spine update --dir R --write` remedy; doctor D18 prints the same remedy.
Following it changes nothing, so this spelling defeats I128's acceptance
criterion 2 ("reach a validating state by following the printed remedy
alone"). Pre-existing update behaviour, not introduced by I128.

## Fix

Either treat a comment on a machine-rendered mirror row as recognised
(the value is what the resolver reads; the comment is the owner's note and
should survive the retired-override migration verbatim), or make the
retired-model and D18 remedies detect the skipped-file case and say
`--force-file WORKFLOW.md` will drop the comment. The first keeps the
owner's note; prefer it.

## Acceptance criteria

- [ ] A retired override with a trailing comment migrates to the successor id, keeps its effort and the comment, and validates after the printed remedy alone.
- [ ] A current-id override with a trailing comment is preserved verbatim and WORKFLOW.md is not reported as unrecognized.

<!-- Record an approved-without-test exception using the exact grammar in WORKFLOW.md's Acceptance exceptions section. -->
