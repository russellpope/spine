---
id: I075
title: Effort as a first-class dispatch parameter
severity: med
status: open
affects: [workflow, audit]
blocked-by: [I071]
labels: [wayfinder:task]
parent: I066
assignee:
---

## Question

Effort must be settable per dispatch rather than always carrying the tier
default in (owner, 2026-08-10). Today `effort:` exists as ticket-frontmatter
override and as table-cell notation (`claude-opus-5 @ low`). What changes so
effort flows dispatch-by-dispatch: the dispatch/record grammar (does the
ESCALATION grammar gain an effort dimension, or does the declared triple of
[I069](I069-attribution-declare-then-confirm.md) suffice?), the invocation
pass-through ([I071](I071-claude-auto-invocation-contract.md)), and effort
naming across families (Anthropic low/medium/high/xhigh/max vs GPT-style
reasoning levels — does the pin notation normalize or pass raw)?
