---
id: I075
title: Effort as a first-class dispatch parameter
severity: med
status: fixed
commits: [94660a6, e3c6867, a806e6e, 9f30808, 1cc85da, 8829724]
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

## Resolution

Fixed 2026-08-30 at final I075 product SHA `8829724`. Dispatch resolution now
retains a byte-exact raw `(harness, model, effort)` declaration after final
target selection, supports a JSON-only requested-effort helper, recognizes
native complete Codex `spawn_agent` records and Claude launch arguments, and
records exact ticket-local effort authorization pairs without defining a
cross-family order. Routing audit appends expected/declared/observed effort
state while preserving legacy model verdicts and blocking behavior; observed
effort remains `-` until I074 has documented evidence. Decorated or malformed
records authorize nothing, incomplete transports remain unconfirmable, and
arbitrary non-space raw effort bytes stay valid where a harness vocabulary
permits them. A fresh routine review and a different independent routine
verifier passed hostile source-built, focused/full/race/vet/build, and Windows
compile gates. I074 may now consume the declaration seam. The batch-final
exact-SHA maipipe lane remains the ship gate.
