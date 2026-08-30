---
title: "Heterogeneous routing verdicts PRD"
tickets: I074
created: 2026-08-30
status: accepted contract
depends-on: [I070, I072, I075]
---

# Heterogeneous routing verdicts

## Purpose

I074 extends `spine audit routing` from a preference-only model-tier check to
declare-then-confirm judgment for explicitly declared heterogeneous dispatches.
It consumes I075's raw `(harness, model, effort)` declaration and I072's
host-local exact `observed_ids` data. It does not reconstruct declarations
from transcript models, infer a provider family, or turn a missing effort
field into a default.

I075 must land first. I073 remains a separate public naming migration after
verified I072. Until I073 lands, current public `flavor` CLI and output names
stay compatible; this contract uses harness only for the declaration and host
capability identity.

## Requirements attack

| Conflict | Binding resolution |
| --- | --- |
| A proxy may emit an alias, a canonical ID, or an unknown raw ID. | Confirm only through a host-local, byte-exact `observed_ids` mapping for the declared final route. An unmapped raw ID is unconfirmable, not a guessed mismatch. |
| Existing linkage can associate broad session evidence with more than one ticket. | A heterogeneous confirmation needs the complete dispatch identity `(source, session, dispatch)` and one linked worker-event correlation. Coarse root linkage may remain legacy model evidence but cannot confirm a triple. |
| A host pin deliberately differs from the repository preference. | Compare against the host-resolved final pin, including its exact model and effort, never against the repository preference. A valid pin is conformant only through its host-local observed mapping. |
| A transport argument is not runtime proof. | I075 declaration fields are never observed effort. An observed effort exists only when a documented harness extractor returns a raw event field tied to the same dispatch identity. |
| A warning could hide a provider/model mismatch. | A proven model mismatch or proven observed-effort mismatch blocks. Absence of proof is an explicit nonblocking unconfirmable result. |
| Legacy audit tables and model-only fixtures are already consumers. | Do not reinterpret old rows. The new path is additive for explicitly declared heterogeneous dispatches, with stable leading output columns. |

## Exact correlation and host equivalence

For each declared dispatch event, the audit retains:

```text
identity=(source, session, dispatch)
declared=(harness, model, effort)
expected=(final host-selected model, final target effort)
```

An observed model can participate only when it came from a linked worker event
with the same complete identity. The declared harness must equal the host
configuration harness being examined. The model is confirmed only when the
raw observed ID byte-exactly equals an `observed_ids` member on the declared
host route with the declared final `(model, effort)` pair. No case folding,
trimming, substring or family match, global alias, historical ID, fallback,
or equality-to-canonical-model shortcut is allowed.

An exact observed ID assigned by the host file to a different route is a
model mismatch. A missing identity, missing host configuration, unavailable
harness, no raw observed model, or an ID absent from the host-local mapping is
unconfirmable. I072 already makes `observed_ids` globally unique, so no
ambiguous mapping may be selected. A malformed host file remains the existing
exit-2 preflight error, not a verdict.

A valid host pin is equivalent to its approved tier for this judgment. The
expected pair is `Resolution.Entry`, not `Resolution.Requested`. The pin is
not an escalation, and it does not become conformant merely because its model
name resembles an observed ID. I074 may relax I072's controlled-launch
identity gate only after this exact model confirmation path and its tests land.

## Effort evidence

I075 supplies expected effort, declared effort, and exact-pair authorization.
I074 adds an optional harness-specific observed-effort extractor. A harness
may register an extractor only when its transcript field, field semantics,
and event correlation have been documented with durable fixtures. The
extractor returns one raw string or absent. It cannot read command flags,
settings, environment, model IDs, gateway defaults, or a dispatch declaration.

The observed effort confirms only when it byte-exactly equals the declared
effort on the same complete identity. A different observed raw token is a
proven effort mismatch. An absent declaration, absent documented extractor,
absent raw field, or incomplete correlation is unconfirmable. I074 does not
translate values, compare their quality, or give an effort record an ordering.

## Verdicts, aggregation, and blocking

I074 adds these public verdicts for explicit declared heterogeneous events:

| Verdict | Condition | Severity | Blocks |
| --- | --- | ---: | --- |
| `confirmed` | Model is exact-confirmed, declared effort matches or has exact authorization, and observed effort exactly equals declared effort. | 0 | No |
| `declared-observed-mismatch` | A correlated observed model maps to another route, or a correlated observed effort differs from declared effort. | 6 | Yes |
| `declared-effort-mismatch` | The declaration is present but neither equals expected effort nor has its exact I075 authorization. | 6 | Yes |
| `unconfirmable` | Any required model or observed-effort proof is absent or unmapped, without a proven mismatch. | 2 | No |

The strongest event verdict wins for a ticket. A proven mismatch outranks an
unconfirmable event. An unauthorized declared effort also blocks even if the
model cannot be confirmed: it is a complete local declaration failure, not a
claim about provider behavior. Existing `silent-descent` stays blocking and
continues to protect non-declaration audit paths. Its current precedence and
legacy model-tier escalation records stay intact.

The following matrix is exhaustive for the I074 declaration path. `D ok`
means expected effort matches the declaration or the I075 record authorizes
the exact pair. `O ok` means a documented observed field exactly matches the
declaration.

| Model state | Effort state | Result |
| --- | --- | --- |
| confirmed | D ok, O ok | `confirmed` |
| confirmed | D ok, observed absent | `unconfirmable` |
| confirmed | D ok, observed differs | `declared-observed-mismatch` blocking |
| confirmed | declared absent | `unconfirmable` |
| confirmed | declaration unauthorized | `declared-effort-mismatch` blocking |
| mismatch | any effort state | `declared-observed-mismatch` blocking |
| unconfirmable | declaration unauthorized | `declared-effort-mismatch` blocking |
| unconfirmable | every other effort state | `unconfirmable` |

Current transcript formats have no approved observed-effort extractor. Their
I074 result therefore remains `unconfirmable` rather than `confirmed`; this
is the deliberate declared-only continuation from I075.

## Output and compatibility

The current `ticket tier actual verdict detail` prefix remains unchanged for
legacy rows. I074 appends stable declaration fields: final expected pair,
declared triple, model confirmation state, declared-effort status,
observed-effort status, and correlation identity or `-` when unavailable.
`-` always means missing evidence, never a selected default.

The in-memory audit report gains additive structured event details so an
aggregate verdict remains explainable. The current text CLI retains its table
contract and does not gain a speculative JSON mode. Existing model-only audit
results, model aliases/history, source-derived flavor selection, model-tier
`ESCALATION`, `FALLBACK`, `DISCARDED`, unmatched dispatches, and no-host
behavior remain byte-compatible. I074 must not retroactively emit
heterogeneous verdicts for an old dispatch without a complete declaration.

## Non-goals

- Guess an observed model or effort from a command, table, or provider name.
- Alter I072 host schema, pin precedence, mirror behavior, or configuration
  security boundary.
- Normalize effort tokens or define cross-family equivalence.
- Rename public flavor names. I073 owns compatibility reads and migration.
- Change a gateway, wrapper, or Agent-tool transport without its owner-verified
  contract.

## Acceptance criteria

1. I074 consumes I075 declarations and validates exact per-retry correlation.
2. Exact host-local observed IDs confirm the final host target; different
   mapped routes mismatch; missing or unmapped evidence is unconfirmable.
3. Observed effort comes only from a documented exact extractor, never from a
   declared input or inferred default.
4. The matrix, severity ordering, aggregation, and blocking behavior above
   are covered by focused tests.
5. Host-pinned equivalence compares final pinned targets and leaves I072's
   security, precedence, and compatibility guarantees intact.
6. Legacy model-only output and routing semantics stay compatible. I073's
   public rename remains outside this diff.
