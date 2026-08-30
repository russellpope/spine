---
id: I123
title: "spine update advises on enabled gate classes missing required config"
severity: low
status: open
affects: [I093]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

## What to build

When a gate pack enables a config-driven class but its required
`gate_pack_config` value is empty, `spine update` reports a pre-write advisory
that names the class, missing key, and both remedies: configure the key or add
the class to `gate_pack_disabled`. The generated stage stays unchanged;
missing configuration does not become an implicit disable.

## Acceptance criteria

- [ ] Empty config reports exactly the enabled classes whose inputs are absent.
- [ ] Supplying one key or disabling one class removes only that advisory.
- [ ] Config-free checks and `tskip` with an empty allowlist do not warn.
- [ ] Existing configured update plans and write behavior remain unchanged.

## Related

- I093 item 3. Owner selected the pre-write advisory option on 2026-08-30.
