# Host routing configuration (I072)

## Product requirement document

**Status:** approved for implementation

**Ticket:** I072, host config schema and preference/constraint precedence

**Decision:** adopt Design C from
`.superpowers/sdd/I072-worker4-grill.md`: a schema-versioned local JSON
capability and pin file. It is a host constraint, not a third preference
table.

## Goal

Let `spine model` produce the model and effort that this host can actually
run, while keeping the embedded estate defaults and repository-owned routing
mirror portable, reviewable, and unchanged by host facts.

## Context and outcome

The existing resolver has two preference layers: the embedded estate default
and a repository's `WORKFLOW.md` `model_routing` row. I068 adds a different
kind of information: a host may have a harness, executable, gateway profile,
and model routes that another host lacks. A host may select an explicit final
route for one tier. That information must constrain the final dispatch without
silently rewriting repository policy.

The command remains `spine model <flavor> <tier>` in I072 because this is the
current public interface. The new local file calls the execution vehicle a
`harness`; I073 owns the public flavor-to-harness rename and compatibility
migration.

## Canonical local file

The only default path is:

```text
os.UserConfigDir()/spine/routing-host.json
```

`os.UserConfigDir` supplies the platform behavior and may consult the standard
platform configuration environment, including `XDG_CONFIG_HOME` on Linux-like
hosts. That means
`$XDG_CONFIG_HOME/spine/routing-host.json`, or `~/.config/spine/routing-host.json`
when XDG is unset, on Linux-like hosts, and
`~/Library/Application Support/spine/routing-host.json` on macOS. Tests inject
an explicit directory provider or an absolute fixture path through private
argument-based helpers. They never replace a package global, so parallel tests
cannot race. I072 adds no ordinary CLI flag or environment-variable override
for this path.

The file is a trusted owner- or fleet-managed authority, outside repositories.
It is not created by `init`, `adopt`, or `update`, and is not copied into a
handoff, ticket, report, template, or `WORKFLOW.md`. I072 proves that its
declared routes are feasible. It does not prove, record, or ratify why a pin
was approved.

## Schema version 1

The file is UTF-8 JSON. Every object at every depth is closed: the root,
each harness, each model-route object, and each pin value accept only the
members listed below. Unknown members are schema errors. Duplicate JSON object
keys are rejected before semantic validation. All identifiers and reference
strings reject empty values and control characters.

```json
{
  "schema_version": 1,
  "host_id": "work-mac-2026",
  "harnesses": {
    "claude": {
      "available": true,
      "executable": "claude-auto",
      "launch_contract_ref": "owner-verified: work-gateway-2026-08-11",
      "models": {
        "gpt-5.6-sol": {
          "efforts": ["high"],
          "observed_ids": ["gateway/gpt-5.6-sol"],
          "gateway_ref": "work-claude-gateway"
        }
      }
    }
  },
  "pins": {
    "claude.primary": {
      "model": "gpt-5.6-sol",
      "effort": "high",
      "evidence_refs": [
        "owner-ratification:I068",
        "eval:docs/evals/2026-08-29-gpt-sol/eval.md"
      ]
    }
  }
}
```

The example shows shape only. It makes no claim about a currently installed
gateway or model.

### Required and allowed members

| Object | Members and rules |
| --- | --- |
| Root | `schema_version` is the integer `1`; `host_id` is an opaque non-empty label; `harnesses` is a non-empty object; `pins` may be empty. |
| `harnesses.<name>` | `<name>` is one of the current model flavors. `available` is required and true for a usable harness. `executable` and `launch_contract_ref` are required non-empty strings. `models` is a non-empty object when `available` is true. |
| `harnesses.<name>.models.<model>` | `efforts` is a non-empty, duplicate-free list of opaque non-empty strings. `observed_ids`, when present, is a duplicate-free list of non-empty strings. `gateway_ref`, when present, is a non-empty opaque reference. |
| `pins.<harness>.<tier>` | The key has exactly one dot. `tier` is exactly one of `primary`, `routine`, `mechanical`, or `fallback`. The value contains non-empty `model` and `effort`, plus optional duplicate-free non-empty `evidence_refs`. The named harness must be available and its declared model route must include the exact effort. |

A route is the exact pair `(model, effort)`. There is no separate
"duplicate semantic route" rule: duplicate JSON keys, duplicate effort values
within one route, and globally duplicate `observed_ids` are the complete
duplicate rules. Equal model@effort pairs in different harnesses are allowed.
Efforts are opaque strings in I072. The implementation does not translate
Claude effort names to GPT reasoning names and does not infer support from the
model ID.

`observed_ids` is capability evidence for a declared route. Each item is a raw
transcript ID that may confirm that route later. Matching is exact string
equality only, with no alias expansion, normalization, substring match,
family match, provider inference, or fallback to `model`. An ID may appear
only once in the complete configuration. It is never an alias in
`models/defaults.json` and it does not affect resolution.

## Security boundary

This file declares capability. It is not a gateway installer, shell-command
template, or credential transport.

The parser may read the JSON and check whether `executable` resolves on the
host. That executable lookup may consult the host `PATH`; it must never execute
the resolved executable. JSON values never expand environment references,
execute, or trigger network activity. Apart from executable lookup and the
platform behavior inside `os.UserConfigDir` while locating the default file,
the parser does not consult environment state, make a network request, or write
the file. The field
names are intentionally narrow. URLs, base endpoints, tokens, auth
headers, credentials or credential-file references, `modelOverrides` values,
arbitrary `args`, environment maps, shell fragments, and unknown fields are
rejected. Error messages and JSON output name the config path and safe labels
only. They never print file contents, secrets, endpoint values, or an
unredacted transport configuration.

`launch_contract_ref` and `gateway_ref` are non-secret opaque references.
I071 remains the authority for executable argv and protected transport
environment. A valid host configuration never authorizes inventing or running
a `claude-auto` command.

## Resolution contract

For `spine model [--dir REPO] <flavor> <tier>`, resolve in this order:

1. Read the embedded `models/defaults.json` cell.
2. Apply the repository's current `WORKFLOW.md model_routing` row through
   `model.Resolve`. This produces the **requested preference pair** and its
   existing `default`, `inherited`, or `override` provenance.
3. If the host file is absent, return that requested pair unchanged. This is
   the legacy path and records host status `unconfigured` in the host-aware
   result.
4. If the host file is present, validate its schema and host capability before
   resolving a final pair. The requested harness must be available and its
   executable must resolve.
5. If `pins["<flavor>.<tier>"]` exists, return that pin's exact
   `(model, effort)` as the final pair. Preserve the requested pair,
   provenance, host ID, config path, and pin evidence in the trail.
6. If no pin exists, the requested `(model, effort)` must occur exactly in
   the selected harness's declared reachable routes. Return it unchanged when
   it does. Otherwise fail; do not select a nearby, default, alias, or
   alternate model.

The precedence is therefore:

```text
embedded estate default -> repository preference -> host constraint or pin
```

The final host pin wins over a repository override because the trusted host
authority declares an executable feasible final route. The command must expose
that replacement; it must not make the repository override appear to have
vanished. A repository cannot add reachable routes. I072 does not judge whether
the owner or fleet should have approved a replacement.

I072 does not apply host rules to `--alternate`. Alternate reachability and
selection need their own decision. Existing alternate behavior stays intact.
The `pi` row likewise gains no special or accidental host behavior.

## Command output and errors

Text output still prints the final dispatchable ID. `--effort` still prints
the final effort. Thus a configured valid pin is safe input to a later
dispatcher.

`--json` retains its current top-level `flavor`, `tier`, `id`, `effort`,
`aliases`, `alternate`, and `provenance` meanings. `id` and `effort` mean the
final pair. I072 may only add fields, never remove or reinterpret old ones:

```json
{
  "requested": {"id": "claude-fable-5", "effort": "", "provenance": "default"},
  "host": {"id": "work-mac-2026", "status": "pinned", "config_path": ".../routing-host.json"},
  "pin": {"model": "gpt-5.6-sol", "effort": "high", "evidence_refs": ["owner-ratification:I068"]}
}
```

The unconfigured path retains today's text behavior and old JSON fields. Its
additive host status must not change those fields. A malformed or unsupported
file, unavailable harness, missing executable, unreachable requested pair, or
unreachable pin makes every `spine model` mode exit 2, print one diagnostic to
stderr, and print no fallback ID or effort to stdout.

### I051 compatibility: fail-closed controlled launch validation

`spine model [--dir REPO] validate [--expect MODEL_ID] <flavor> <tier>` keeps
I051's strict repository read and launch policy. In one process it reads one
strict repository snapshot and one host-config snapshot, validates the
requested repository route under I051, then inspects this document's host
constraint against that validated request.

For an absent host file, I051's current stdout, stderr, and exit behavior stay
byte-compatible. For a present file, a pin whose model ID is byte-identical to
the validated repository active ID may validate after its model passes I051's
positive-ID and deny policy. A forbidden or syntactically unsafe pin refuses
before stdout.

A safe divergent pin remains valid host configuration and may replace the
preference in plain host-aware `spine model`, `--effort`, and `--json` output.
Doctor diagnoses that divergence. Controlled `model validate` must refuse it
with exit 2 and no stdout because audit remains preference-only and cannot yet
prove the launched host ID conforms. Neither `--expect` value can bypass this
refusal. I074 is the gate that may later enable divergent pins by adding host
conformance to audit's active vocabulary. Until then, `--expect` compares only
after the host pin passes this identity gate. I075 still owns recording or
auditing dispatch effort.

The implementation must not fall back from `model validate` to plain
`spine model` after a validation failure. Tests cover plain inspection and
doctor diagnosis of a safe divergent pin, controlled refusal of that pin, an
identical pin that validates, a forbidden pin, `--expect` after the identity
gate, and no-host byte/exit compatibility.

## Portable mirrors and update behavior

`model.MirrorRows`, `internal/tmpl.Render`, `update.applyModelRouting`,
`spine init`, `spine adopt`, and `spine update` remain host-blind. They use
only embedded defaults and repository preferences. A host config cannot alter
a rendered mirror row, create an update plan item, change inherited-versus-
override classification, or write its pin into `WORKFLOW.md`.

I072 does not change `models/defaults.json`, its aliases, or its history. It
does not change `transcriptFlavor` or the existing source/model derivation in
the audit. No template-generation bump is required solely to add this local
capability file. If a later approved template wording change is made, it must
follow the normal template-generation and migration rules as separate work.

## Doctor and audit boundary

I108 must take D14 despite its stale ticket text claiming D11. I050 also adds
a doctor check. I072 does not reserve a number now. At integration, after the
I108 and I050 changes have been merged, the implementer must allocate the
first unclaimed D-number from current source and use it consistently in code,
tests, documentation, and the ticket resolution. If both land before I072,
the expected allocation is D16. If merge ordering differs, the checked source,
not this prediction, decides the number.

`spine doctor` reports local configuration health read-only. Its check is
deterministic: for each available declared harness, in lexical harness order,
and each tier in `model.Tiers` order, resolve that harness's repository
preference without host substitution. For every unpinned exact pair absent
from that harness's declared routes, emit one finding. Do not inspect absent or
unavailable harnesses. A valid divergent pin replaces the plain-output route
but produces a warning that controlled validation cannot launch it until I074
makes it auditable. A byte-identical pin suppresses the corresponding
preference reachability warning because it supplies the same active model ID.

| Condition | I072 allocated finding result |
| --- | --- |
| File absent | Silent. Existing unconfigured hosts remain healthy. |
| Unreadable file, malformed JSON, duplicate key, unsupported schema, prohibited member, invalid declared executable, unavailable pinned harness, missing pinned model, or unsupported pinned effort | The allocated I072 finding reports one or more `error` results on the host-config path. Existing doctor exit handling returns 1. |
| Valid config, an available declared harness, an unpinned tier, and an unreachable repository preference | One allocated-ID `warn` per harness-tier, naming repository, harness, tier, and requested pair. Existing doctor exit handling returns 1. |
| Valid divergent pin | One allocated-ID `warn` naming the harness-tier and stating that controlled validation refuses the host model until I074 makes it auditable. |
| Valid byte-identical pin without `evidence_refs` | No I072 finding. |

`spine audit routing` adds only a structural Claude-harness preflight before
transcript discovery and ticket verdict construction. It loads a present file,
enforces the closed schema, checks declared executables, and verifies every
pin is an exact model@effort route on an available declared harness. An invalid
configuration or pin is a configuration error with CLI exit 2. A valid
unreachable unpinned preference is not an audit configuration failure.

Audit keeps its existing preference-only mappings, all verdict strings, and
their output bytes. It does not call `ResolveForHost` for every flavor, replace
an audit mapping with a final host route, or test host-to-observed conformance.
Those questions wait for I074.

Until I074, a valid divergent host pin plus an `observed_ids` value is not
evidence that an existing source-derived transcript token conforms. I072 leaves
existing audit interpretation untouched; it neither adds conformance nor
creates a new silent-descent result. This is why `model validate` refuses a
divergent pin even though plain host-aware model output may display it.

## Migration and compatibility

Deployment is opt-in and reversible:

1. A host without the file keeps the exact two-layer resolution result.
2. Fleet management may provision a version-1 file for one host.
3. The owner runs `spine doctor --dir <repo>` and representative
   `spine model --dir <repo> --json <flavor> <tier>` checks.
4. A controlled dispatcher uses the host pair only when `model validate`
   accepts it. Before I074, that means no divergent model ID even after I071's
   launch contract is owner-verified. Plain host-aware output remains an
   inspection path, not launch authorization.
5. Removing the local file restores the legacy result without a repository
   rollback or estate sweep.

`host_id` is an opaque profile label, not a discovered hostname. Fleet status
may report its label and configuration status but must never dump the JSON.

## Explicitly reserved follow-up scope

- **I073:** owns public `flavor` to `harness` names, compatibility reads,
  CLI/output/document migration, and estate sweep. I072 only uses `harnesses`
  in new local configuration.
- **I074:** owns declared-versus-observed heterogeneous audit correlation and
  verdicts such as confirmed, mismatch, and unconfirmable. I072 provides plain
  host-aware resolution and exact raw observed-ID data, but keeps divergent
  controlled launch validation closed until I074 supplies conformance.
- **I077:** owns interpretation of `evidence_refs`, eval lookup, mutation
  evidence, and any no-evidence advisory. I072 stores opaque references and
  does not require an eval record for a valid owner pin.

I075 remains responsible for dispatch-record effort semantics. I072 merely
requires each host route and pin to carry an exact effort string.

## Requirements attack and resolutions

| Attack | Resolution |
| --- | --- |
| A local model table would evade git review and revive ADR 0011's rejected third preference layer. | The host file contains only local capability and explicit constraints. The two preference layers remain embedded defaults plus repository mirror. |
| Filtering availability could silently substitute an available but weaker model. | No-pin resolution requires the exact requested pair. Only an explicit pin may replace it, and output preserves both requested and final pairs. |
| Gateway IDs could be added as global aliases and falsely certify other hosts. | `observed_ids` stays local, exact, and non-resolving. `models/defaults.json` is untouched. |
| A configuration file could become a credential or shell-injection channel. | The closed schema forbids secrets, endpoints, env, args, and unknown fields. Validation never executes a command or makes a network call. |
| Rendering a host pin into WORKFLOW would make update and estate results machine-dependent. | Mirror, render, init, adopt, and update never load host state. `spine model --json` carries the visible local trail instead. |
| Host preflight could overwrite audit's preference-only judgment. | Audit checks only declared config and pins before Claude transcript discovery. It leaves mappings and verdict bytes unchanged; I074 owns heterogeneous confirmation. |
| A safe divergent pin could pass launch validation even though unchanged audit policy would call it unmapped. | Plain host-aware output may inspect the pin and doctor diagnoses it, but controlled validation refuses it until I074 adds the same host-active ID to audit. A byte-identical pin remains launchable. |
| Treating effort as decoration would turn one route into several indistinguishable choices. | Every route and pin carries the exact model@effort pair. I072 keeps strings opaque; I075 owns transport semantics. |
| I108 has stale D11 text, I050 also adds a doctor check, and implementation order is not fixed. | I108 takes D14. I072 allocates only after both integrations by reading the current source. D16 is expected when both precede I072, but not reserved. |

## Non-goals

- Build, configure, authenticate to, or probe a gateway.
- Store or print endpoint URLs, tokens, auth headers, environment values, or
  wrapper argv.
- Add a generic environment-variable or CLI config-path override.
- Alter embedded default aliases, history, repository override semantics, or
  update rendering.
- Rename the public flavor interface, define audit confirmation vocabulary,
  or interpret eval evidence.
- Infer model family, effort support, equivalence, or an alternate route.

## Acceptance criteria

1. `os.UserConfigDir()/spine/routing-host.json` is the only default location.
   Its platform lookup may use normal configuration environment behavior, but
   JSON values never expand environment input, execute, or cause network
   access. Private argument-based directory-provider and explicit-path seams
   are race-safe for model, doctor, and audit tests, and no repository command
   creates or writes the file.
2. Schema-version 1 accepts only the documented JSON shape at every nested
   object. Duplicate keys, invalid pin keys, duplicate efforts, duplicate
   observed IDs, empty/control-containing strings, unsupported versions, and
   every prohibited security-sensitive member fail without exposing config
   contents. There is no duplicate-semantic-route rule.
3. A route and pin are exact `model@effort` pairs. A pin must name an
   available harness, resolvable executable, declared model, and listed effort.
   `observed_ids` is unique exact evidence only and never a resolver alias.
4. Resolution visibly applies estate default, repository preference, then host
   constraint or pin. A valid pin supplies the final pair while preserving the
   requested pair and provenance. Without a pin, only the exact reachable
   requested pair succeeds; no automatic substitution occurs.
5. `spine model [--dir REPO] validate [--expect MODEL_ID] <flavor> <tier>`
   strictly validates the requested repository route under I051, then checks
   the host constraint in the same process. A byte-identical pin may validate.
   A safe divergent pin remains inspectable in plain host-aware model output
   but controlled validation refuses it as not yet auditable, regardless of
   `--expect`; forbidden pins fail; and no-host output and exits remain
   byte-compatible. I074 is the only gate that may later enable divergent
   launch validation.
6. Missing host config preserves current text output and existing JSON field
   meanings. A valid pin changes text and `--effort` to the final pair and
   adds a JSON trail without breaking existing fields.
7. Invalid configuration, unavailable harness, missing executable, unreachable
   requested pair, and unreachable pin make every model-output mode exit 2,
   emit no fallback stdout, and describe the safe failing path.
8. `MirrorRows`, render, init, adopt, and update are proven host-blind. A host
   fixture cannot change `WORKFLOW.md`, an update plan, default history, or
   inherited/override classification. Existing pi and alternate behavior stay
   unchanged, and I072 makes no template-generation change.
9. At code time, doctor rechecks the first unclaimed ID, currently expected
   D16. It reports config errors and deterministically reports one warning for
   every unreachable unpinned tier of every available declared harness. It
   also diagnoses each divergent pin that controlled validation refuses until
   I074. It ignores absent or unavailable harnesses and remains silent without
   a file.
10. Audit preflight validates only declared host config and pins before Claude
    transcript discovery. Bad configuration exits 2. Existing mappings,
    verdicts, and output bytes remain unchanged; unpinned reachability and
    host-to-observed conformance wait for I074.
11. The host file is a trusted owner/fleet authority. I072 proves feasibility,
    not ratification; `evidence_refs` remain optional opaque data for I077.
12. Documentation and ticket closure state the local, secret-free boundary,
    migration path, non-goals, and I073/I074/I077 ownership. The finished
    implementation receives a fresh requirements-attack spec review and
    independent verification before ship.
