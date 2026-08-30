---
id: I123
title: "spine update advises on enabled gate classes missing required config"
severity: low
status: fixed
commits: [708c0fe, 9f86b11, c07acc6]
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

- [x] Empty config reports exactly the enabled classes whose inputs are absent.
- [x] Supplying one key or disabling one class removes only that advisory.
- [x] Config-free checks and `tskip` with an empty allowlist do not warn.
- [x] Existing configured update plans and write behavior remain unchanged.

## Related

- I093 item 3. Owner selected the pre-write advisory option on 2026-08-30.
- Accepted design and implementation plan: `docs/specs/2026-08-30-i123-update-gate-config-advisory-design.md` and `docs/specs/2026-08-30-i123-update-gate-config-advisory-plan.md`.
- Round-2 ruling: implementation and closure are already owner-authorized; close after the plan's review, independent verification, ticket evidence, and exact-SHA lane gates, unless a genuine contradiction or out-of-scope expansion requires a stop.

## Resolution

Fixed 2026-08-30. `708c0fe` adds explicit required-input metadata and sorted
advisory derivation; `9f86b11` emits the exact stdout advice after candidate
preflight and before refusal or writes without changing render or exit state;
`c07acc6` locks every configured-key and disabled-class isolation case plus
configured pre-I123 plan/write bytes. Fresh routine review and a different
independent verifier passed at `c07acc6`, including mutation controls, full and
race suites, vet, build, and compiled CLI probes. The single batch-final
exact-SHA maipipe lane remains the ship gate for this closure commit.
