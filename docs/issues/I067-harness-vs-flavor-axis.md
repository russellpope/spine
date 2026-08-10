---
id: I067
title: Harness vs flavor — does the identity axis split?
severity: med
status: fixed
affects: [model, workflow]
blocked-by: []
labels: [wayfinder:grilling]
parent: I066
assignee: russell
---

## Question

Today `flavor` conflates the execution vehicle with the model family
(claude = Claude Code + Anthropic models). When a claude-flavored harness runs
a non-Anthropic model through a gateway, what does the schema do — keep flavor
as-is, split harness × model-family into two axes, or mint compound flavors
per combo?

## Resolution

(2026-08-10, owner) **Split, and rename flavor → harness.** Harness is the
execution vehicle (claude, codex) and stays the dispatch key; the model cell
carries the actual model id, whatever its family. No compound flavors. Scope
fence: this batch is claude-harness-first — Claude Code as the vehicle for
3rd-party/local models; the codex harness is untouched and symmetry is out of
scope on the map. Migration is mapped as [I073](I073-flavor-to-harness-rename-migration.md),
sequenced after the schema design ([I072](I072-host-config-schema-and-precedence.md)).
