---
id: I070
title: What do proxied 3rd-party model ids look like in Claude Code transcripts?
severity: med
status: in-progress
affects: [audit, model]
blocked-by: []
labels: [wayfinder:research]
parent: I066
assignee: codex-team
---

## Question

`spine audit routing` reads per-turn model ids from Claude/Codex session
formats. When Claude Code drives a 3rd-party or local model through a gateway
(work-laptop custom gateway → GPT; OpenAI-compatible endpoint → open weights),
what id strings actually land in the transcript — the upstream model's real id,
the gateway's alias, or something mangled? Survey real transcripts from each
gateway path in use. Does the resolver/table need alias rows mapping
transcript-observed ids to canonical model ids, and is the observed id stable
enough for the confirm half of declare-then-confirm
([I069](I069-attribution-declare-then-confirm.md))?
