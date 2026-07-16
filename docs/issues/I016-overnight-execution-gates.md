---
id: I016
title: Overnight execution gates
severity: med
status: fixed
affects: [process]
blocked-by: []
labels: [wayfinder:grilling]
parent: I010
assignee: russell
---

## Question

How does "subagents deliver overnight" meet the workflow's own mandatory gates (PRD up front, tickets, spec-review, verify)? Requirements-attack note: shipping the stage-skipping fix by skipping the prd/issues stages would repeat the incident this effort exists to prevent.

## Resolution

(2026-07-15, owner) **Full gates, same night.** The charting session itself runs /to-spec (compact PRD pair in spine `docs/specs/` — fast, decisions already locked) and /to-tickets (build tickets in this ledger, tier/effort annotated). Overnight subagents execute under subagent-driven development with ESCALATION/FALLBACK records and `spine audit routing` at verify. Morning: /spec-review of the finished diff against the PRD + verification with the owner. Map task tickets as direct build tickets rejected (no PRD to spec-review against); build-tomorrow-HITL rejected (loses the overnight window).
