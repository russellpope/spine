---
title: "Effort dispatch declarations PRD"
tickets: I075
created: 2026-08-30
status: accepted contract
depends-on: [I071]
followed-by: [I074]
---

# Effort dispatch declarations

## Purpose

I075 makes effort a raw, resolved part of every dispatch declaration. A
dispatch declares `(harness, model, effort)` before launch. The declaration
records the value passed to a transport. It does not claim that a provider,
gateway, or agent runtime used that value.

The current public model API still calls the execution axis `flavor`. I075
does not rename that API, its CLI arguments, existing output, or generated
documents. I073 owns that public migration after I072 is verified. This PRD
uses "harness" only for the approved declaration concept and the local host
configuration's existing field.

## Requirements attack

| Conflict | Binding resolution |
| --- | --- |
| A ticket default and a command line cannot show what a retry requested. | Resolve and record a complete triple per launch and per retry. |
| `low -> high` has no meaning across Claude, Codex, Pi, and proxied families. | Preserve raw family-specific tokens. Never derive an ordering, severity, or tier change from effort strings. |
| A CLI flag can lose to settings, a wrapper, or a gateway. | Treat flags and Agent-tool arguments as declarations only. I075 reports no observed or effective effort. |
| An omitted token could accidentally select a transport default. | Resolve omission to the final target's exact effort before dispatch. `-` means no declaration was recorded, not "default". |
| I072 can pin both model and effort. | Apply an override only after final target selection. I075 consumes a target-resolution seam and neither reads a host file nor changes host precedence. |
| Existing audit clients consume model-only rows. | Preserve current model verdicts, leading table columns, blocking behavior, aliases, and model-tier escalation behavior. I075 adds declared-only data for I074. |

## Target resolution and validation

The implementation adds a model-package resolver for a dispatch target. Its
input is repository, current public flavor, tier, and an optional requested
effort. It returns the selected `model.Entry` with the selected model,
provenance, and final raw effort.

1. Resolve the target through the current route-selection path. Before I072
   is verified this is the ordinary repository result. When host-aware
   resolution is available, use its final `Resolution.Entry`, not its
   repository preference.
2. When no requested effort exists, retain the target entry's resolved effort.
3. When a non-empty requested effort exists, replace only that entry's effort
   after validating it against the selected flavor's vocabulary.
4. Reject an empty or whitespace-only supplied value and an invalid
   vocabulary value. Do not trim it into a different token, translate it,
   choose another model, or fall back to a default.

The existing `checkEffort` rule remains authoritative: a flavor with an
`effortVocabulary` accepts only byte-exact members; an absent vocabulary keeps
the established permissive behavior. A host route may later restrict accepted
tokens after final selection, but I075 does not invent that capability or a
universal scale.

`model.Resolve`, `spine model --effort`, table defaults, history,
provenance, mirrors, and alternate selection keep their current meanings.
A new JSON-only helper flag, `--dispatch-effort VALUE`, may request this
resolver. It is invalid without `--json` and follows the current
flags-before-positionals grammar. The old boolean `--effort` remains exactly
"print the resolved effort".

## Declaration, transport, and retry contract

Every controlled launch writes a declaration with these fields:

```text
harness=<raw execution vehicle> model=<exact selected ID> effort=<exact raw token>
```

`tier` remains the ticket's intent and is not a substitute for the triple.
Each retry gets a new declaration. A retry may use a different effort only
when it has its own exact authorization described below.

| Transport | Required declared fields | Binding pass-through boundary |
| --- | --- | --- |
| Stock Claude and verified claude-team cmux/herdr paths | Parsed `--model` and `--effort` | `claude ... --model "$MODEL" --effort "$EFFORT"` from I071. |
| Codex agent dispatch | A native `spawn_agent` function-call record with explicit `model` and `reasoning_effort` (or the documented legacy `effort` alias) is the raw controller vehicle record and declares `(codex, model, effort)` | Pass explicit platform fields. The audit accepts a documented legacy `effort` transcript alias only where fixture evidence proves it. The function name and explicit arguments, not a model ID or coarse transcript source, supply this raw record. |
| `claude-auto`, raw OpenAI endpoints, or an Agent transport with no proven effort field | The triple is still required | OWNER VERIFY. I075 emits no wrapper argv, adapter, or invented Agent parameter. |

The audit stores declaration source, for example `--effort`,
`reasoning_effort`, legacy `effort`, or absent. An orchestration argument is
not observed provider evidence.

## Exact effort authorization grammar

The existing workflow record is retained exactly:

```text
ESCALATION <ticket> effort <from>-><to> reason: <one line>
```

The arrow is one unspaced literal ASCII `->`. `<from>` and `<to>` are
non-empty raw tokens without spaces and may otherwise retain arbitrary bytes.
`reason:` must occur once with non-empty one-line text. The parser must reject
a second `->`, a spaced arrow, missing endpoint, missing reason, empty reason,
or extra grammar that changes the ordered record. A malformed line authorizes
nothing.

A valid record authorizes only the same ticket's declaration whose expected
target effort is byte-exactly `<from>` and whose declared effort is
byte-exactly `<to>`. It does not authorize another ticket, a model-tier
change, an absent declaration, a reversed pair, or a different retry pair.
Repeated distinct pairs are legal. The ledger word "ESCALATION" is retained
for compatibility. It must never imply that `<to>` is higher quality or that
the values have a global order.

The audit keeps effort records in a separate ledger collection. The current
model-tier `ESCALATION`, `FALLBACK`, and `DISCARDED` parsers retain their
current behavior and outcomes.

## Declared-only audit contract

I075 exposes, per attributed dispatch and in each aggregated ticket result:

| Field | Meaning in I075 |
| --- | --- |
| Expected effort | The final selected target effort before a retry override. |
| Declared effort | The raw dispatch token, or `-` when the transcript contains no declaration. |
| Declaration status | target match, exact authorized deviation, unauthorized declaration, or unconfirmable when absent. |
| Observed effort | Always `-` in I075. |

I075 may identify an unauthorized declared effort, but it does not assign a
new public verdict, severity, or blocking result. I074 owns the model and
effort combination matrix, observed-effort correlation, and verdict names.
I075 must not infer effective effort from model IDs, table defaults, command
flags, `/status`, settings, environment variables, or gateway defaults.
Likewise, a missing declaration harness cannot be filled from a model ID or a
coarse transcript source: only the native Codex `spawn_agent` record above,
with its explicit complete arguments, supplies `codex` as the raw controller
vehicle. Non-`spawn_agent` records, linked worker actuals, and incomplete
records remain incomplete and never provide observed effort.

The CLI keeps the current ticket table's leading `ticket tier actual verdict
detail` layout and unmatched model rendering. It may append stable
`declared-effort` and `observed-effort` fields. Existing model-only inputs
retain their existing verdict and report `-` for absent effort data. No output
may say that `-` means a default was used.

## Host pins and compatibility

I072 makes a final host pin an exact `(model, effort)` target. A pin is not a
model-tier escalation, and neither a pin nor a host `observed_ids` entry
confirms runtime effort. I075 must route a selected pin through the same raw
effort validation step, preserve the requested/final trail, and leave the
I072 controlled-launch gate intact until I074 supplies host-conformant audit
semantics.

No host file path, credentials, endpoint, wrapper configuration, or pin
precedence rule belongs in I075. The work must keep no-host output byte
compatible and must not change host-blind mirrors, updates, templates, or
alternate behavior.

## Non-goals

- Define a universal effort scale or compare effort across families.
- Confirm provider-effective effort or add an observed-effort extractor.
- Change `claude-auto`, gateway, raw OpenAI, or unverified Agent-tool syntax.
- Rename public flavor names or migrate the estate. I073 owns that work.
- Define I074 verdict severity, aggregation, or blocking policy.

## Acceptance criteria

1. Each controlled dispatch and retry can carry a resolved raw triple.
2. Omission resolves deterministically to the selected target effort; supplied
   tokens are byte-exact and validated for the selected flavor.
3. The exact effort-record grammar authorizes only matching ticket and
   `(from, to)` declaration pairs, with no implied ordering.
4. Stock verified transports pass the selected value explicitly. Unverified
   transports remain OWNER VERIFY and gain no invented argument form.
5. Audit exposes expected and declared effort but emits no fabricated observed
   effort, new public I074 verdict, or blocking behavior.
6. I072 pin behavior, legacy model-only routing results, model-tier records,
   CLI prefixes, mirrors, aliases, history, alternate selection, and public
   flavor naming remain compatible.
