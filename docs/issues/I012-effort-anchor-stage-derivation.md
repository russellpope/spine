---
id: I012
title: Effort anchor for stage derivation
severity: med
status: fixed
affects: [audit, template]
blocked-by: []
labels: [wayfinder:grilling]
parent: I010
assignee: russell
---

## Question

What anchors an "effort" so stage derivation knows which PRD/tickets/commits to judge? Naive heuristics (PRD exists ⇒ prd done) break on multi-effort repos — ultima has 60+ ledger issues and several PRDs, so bare artifact existence says nothing about *this* effort.

## Resolution

(2026-07-15, owner) **A structured, machine-parseable cursor block** in `.superpowers/sdd/progress.md` anchors the effort: effort name, PRD path, ticket-id set/prefix, stage checklist with the `← YOU ARE HERE` marker. `spine audit stages` parses it and verifies each stage **bidirectionally** against exactly those artifacts — a ticked stage with a missing artifact blocks, and present artifacts with an unticked stage block (stale cursor). No `progress.md` ⇒ warn-only (repo not mid-effort). The cursor format becomes a gen 8 spec; /handoff and skills must write it correctly. Timestamp/newest-artifact heuristics rejected as noisy.
