---
id: I036
title: Template gen 10 — flavor mirror, effort key retired
severity: high
status: fixed
affects: [tmpl, update, templates]
blocked-by: [I035]
execution-mode: subagent-driven
tier: primary
effort:
risk-triggers: [cross-task-integration]
review-tier: primary
---

## What to build

Design D8–D9, D16–D17. Move the mirror to its flavor axis and stamp generation 10.

The routing block renders one line per entry keyed `<flavor>.<tier>`, with effort appended to the value as ` @ <effort>` when set, under a comment marking the block as spine-managed defaults that are overridden by editing a value in place.

**The dotted syntax is a correctness decision, not a cosmetic one.** An un-upgraded binary reading this file finds no recognized bare tier key, so the mapping comes back empty and the existing "no tier mapping found" warning fires — loud and obviously broken. Nested flavor blocks were rejected because the current parser breaks only on a non-two-space line, so nested tier lines would parse as bare tiers with the flavor stripped and the last flavor silently winning, producing confident wrong verdicts. Any later proposal to "clean up" dotted keys into nested blocks is a correctness regression unless every read path is gated.

The top-level `effort:` key retires into the table. A repo carrying a customized value has it migrated into per-entry effort overrides on that repo's entries rather than discarded, and the retired line joins the superseded-line set so it reads as machine-emitted rather than as a local edit. Per-ticket `effort:` frontmatter and the escalation-record grammar are untouched — the two are distinct and must not be conflated.

Shape from the grill (encodes the decision more precisely than prose):

```
model_routing:                      # spine-managed defaults; edit a value to override
  claude.primary:    claude-fable-5
  claude.fallback:   claude-opus-5
  codex.primary:     gpt-5.6-sol @ xhigh
  codex.fallback:    gpt-5.6-terra @ xhigh
```

## Acceptance criteria

- [ ] A captured generation-9 repo upgrades with only sanctioned content-diff lines, per the established migration-test pattern
- [ ] Rendered mirror carries every flavor and tier; generation stamps 10
- [ ] The top-level `effort:` key is absent after upgrade; a customized value survives as per-entry overrides
- [ ] The retired key's prior line is recognized as machine-emitted, not flagged as an unrecognized local edit
- [ ] A repo with a model override keeps it through the format change
- [ ] Value parsing strips the trailing comment FIRST, then splits id from effort (corrected 2026-07-24 to match D9 — the original ordering would let an `@` inside a comment corrupt the id; reordering the parser is a regression, not conformance); a value with neither still parses
- [ ] Per-ticket effort annotations and escalation records are unaffected
- [ ] `go test ./...` green
