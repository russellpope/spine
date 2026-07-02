---
id: 0001
title: Go with stdlib only, cobra reconsidered if v2 nests commands
status: Accepted
date: 2026-07-02
---

# 0001: Go with stdlib only, cobra reconsidered if v2 nests commands

## Context

The fleet has cobra/viper precedent (vmw, pure-gocli), but spine is a flat 4-command CLI whose
config IS the repo artifacts (WORKFLOW.md keys). The local-model-eval corpus documents the classic
cobra footgun (flags bound in PersistentPreRunE after argv parse). Subagents invoke spine thousands
of times; hard-to-hold-wrong beats featureful.

## Decision

Standard library only: map-based dispatch in cmd/spine/main.go plus flag.NewFlagSet per command.
No config file, no viper. Zero third-party dependencies.

## Consequences

Binary stays small, auditable, and fast to build; no supply chain. If v2 grows nested command
trees (eval, handoff), lifting four flat commands into cobra is an afternoon; the reverse
(extracting viper once configs exist) never happens. Revisit at v2 scoping.
