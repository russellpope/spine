---
id: I033
title: Model table + flavor-aware resolver
severity: med
status: fixed
affects: [models]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: [cross-task-integration]
review-tier: primary
---

## What to build

Design D1–D5, D11. A new module owning the estate's model table and answering "what model id and effort back this (flavor, tier) in this repo?".

Ships the defaults as versioned data embedded at build time alongside the existing template assets, keyed by flavor (`claude`, `codex`) as open-ended map keys rather than an enum, each entry carrying a model id plus optional effort and explicit aliases. The data records not only the current default per entry but **every default previously shipped** — the history that makes an inherited value distinguishable from a deliberate override by direct lookup instead of re-render-and-diff.

Exposes one pure resolution function: repo directory + flavor + tier in; resolved id, effort and provenance (**default, inherited, or override** — three values, not two; `inherited` is what the shipped-default history exists to produce) out. Repo context comes from the working directory as with other commands, overridable by flag; outside a spine repo it returns embedded defaults. Effort resolution always yields a determinate value — an entry omitting effort inherits its tier's default, which also moves from prose-in-a-comment into this data.

Claude's fallback default is `claude-opus-5`, with `claude-opus-4-8` recorded in its history. Codex ships all four tiers, primary and fallback carrying explicit effort overrides.

Purely additive — no consumers in this ticket, nothing existing changes behaviour.

## Acceptance criteria

- [ ] Resolution with no repo context returns embedded defaults for every (flavor, tier)
- [ ] A repo carrying no override resolves to defaults; a repo carrying an override resolves to the override, reported as such
- [ ] An entry omitting effort resolves to its tier default; an entry carrying effort resolves to that effort
- [ ] A value matching any historical default reports as inherited; an unrelated value reports as override
- [ ] Unknown flavor and unknown tier are rejected with a clear error, not silently defaulted
- [ ] Flavors are data-driven — adding a third requires no code change
- [ ] `go test ./...` green
