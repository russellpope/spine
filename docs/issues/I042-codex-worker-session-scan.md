---
id: I042
title: Codex worker-session scan — opening-message rule, orchestrator exclusion
severity: high
status: open
affects: [audit, I009]
blocked-by: [I041]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: primary
---

## What to build

Design D21. Builds that predate explicit-model dispatch (pre-I038 cmux
teams, M4a-class) become auditable: a top-level codex session counts as
worker evidence for ticket T iff it is repo-scoped AND T's token appears in
its opening user message (the dispatch brief) AND it contains no dispatch
records of its own. The third clause is the orchestrator exclusion — any
session that dispatches is an orchestrator whose own models are never
ticket evidence, generalizing the claude main-session rule.

Attribution correctness is the whole point: the neighboring-ticket bleed in
later messages must NOT attribute; a lead whose opening message names the
ticket must NOT attribute; a worker that itself spawned a subagent loses
its own turn evidence but keeps equivalent evidence through its own
dispatch records.

Amended at review (spec-level rulings, see design D21): token matching is
against the FIRST LINE of the opening user message only (context-sentence
mentions of neighbor tickets must not attribute — probe-proven blocking
bleed otherwise), and the orchestrator exclusion triggers on any
spawn-SHAPED record, with or without a usable model (a model-less M4a-class
lead is still an orchestrator).

## Acceptance criteria

- [ ] Worker fixture with token in opening message and no dispatch records attributes its per-turn models
- [ ] Neighboring-ticket token appearing only in later user messages does not attribute
- [ ] Orchestrator fixture (opening message names the ticket, contains spawn records) contributes no own-turn evidence
- [ ] Worker-that-spawns fixture keeps ticket evidence via its dispatch record while its own turns are excluded
- [ ] M4a-shaped fixture (worker on an undeclared model) judges `unmapped-dispatch`, not match or silence
- [ ] `go test ./...` green

## Blocked by

- I041
