---
id: I068
title: Host-scoped availability and tier equivalence pins
severity: med
status: fixed
affects: [model, workflow, fleet]
blocked-by: []
labels: [wayfinder:grilling]
parent: I066
assignee: russell
---

## Question

Which harnesses and models are reachable varies per machine (work laptop: GPT
models behind Claude Code + custom gateway; home: open weights via local
endpoints). Where does that availability live, and what happens to tier
semantics when the same ticket can resolve to different models on different
hosts?

## Resolution

(2026-08-10, owner) Availability is a **per-host constraint in spine config**
— estate defaults and per-repo overrides remain preferences; the host config
filters them by what is reachable here. Tiers become **owner-ratified
equivalence pins per host**: the owner can still target e.g. gpt-5.6-sol @
high, just driven by Claude Code instead of codex, and pins a tier to whatever
they deem comparable on each system. Same ticket → different host → different
model is **accepted behavior**, not drift — provided the pin was ratified and
the dispatch declares what actually ran ([I069](I069-attribution-declare-then-confirm.md)).
Concrete schema and precedence order are open in [I072](I072-host-config-schema-and-precedence.md).
