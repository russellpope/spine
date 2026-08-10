---
id: I069
title: Attribution stance — declare-then-confirm, not audit-first gating
severity: med
status: fixed
affects: [audit]
blocked-by: []
labels: [wayfinder:grilling]
parent: I066
assignee: russell
---

## Question

Is auditability a hard gate — no (harness, model) combo enters the pool until
`spine audit routing` can attribute its transcripts — or can pool membership
lead while audit catches up? Heterogeneous pools are exactly where silent
descent hides, but the work-laptop reality (GPT behind Claude Code) already
precedes the tooling.

## Resolution

(2026-08-10, owner) **Declare-then-confirm.** The model is known going in even
when proxied (GPT, GLM, open weights): the dispatch declares (harness, model,
effort) explicitly, and the audit confirms the declaration against the claude
subagent's work product/transcript. Attribution is therefore anchored on the
dispatch record, with the transcript as confirmation — not reconstructed from
transcripts alone. What confirmation looks like for proxied ids is research
([I070](I070-proxied-model-ids-in-claude-transcripts.md)); the verdict
vocabulary for confirmed/mismatched/unconfirmable dispatches is
[I074](I074-audit-heterogeneous-verdicts.md).
