# I077 — eval-informed equivalence-pin ratification

**Status:** Accepted design

**Ticket:** `I077`

## Problem and outcome

I068 makes a host pin an owner-ratified judgment. I072 made that judgment a
valid local configuration value, with `evidence_refs` kept as opaque,
optional provenance. That is the right authority boundary. It also means a
pin can silently outlive the eval that supported it.

The owner selected the combined policy from I077's decision brief. A pin
ratification should name the eval run that justified it, and doctor should
report evidence that no longer supports that claim. The report is a proxy for
an owner's judgment, not a second ratifier. I077 never changes a pin's
validity, route selection, launchability, audit result, or any model-command
exit.

The apparent tension is deliberate: new ratifications should carry an eval
reference, while old pins and exceptional owner decisions must remain loadable.
An absent reference is therefore a D17 finding, not a `hostconfig.Load` schema
error. `evidence_refs` stays optional and continues to accept I072's existing
safe opaque strings.

## Binding contract

### Scope and authority

I077 reads only the repository passed to `spine doctor`, beneath that
repository's physical `docs/evals/` directory. It never reads a fleet root,
another local root, the host home, a URL, a transcript, a session directory,
or I076 yield records. I076 is not eligible evidence in this first policy.

The host configuration remains owner-local authority. I077 must not:

- make `hostconfig.Load` reject a pin because its evidence is absent, malformed,
  stale, mismatched, or failing;
- modify `Pin`, `Resolution`, model/effort resolution, controlled launch,
  `spine model`, `spine audit routing`, or their exit rules;
- replace, remove, de-ratify, block, or gate a pin; or
- mine an eval body, score, stage, transcript, file mtime, or an unreferenced
  eval for a conclusion.

Doctor retains its current general rule that a `warn` finding can make
`spine doctor` exit 1. D17 is a normal doctor warning under that existing rule.
It does not add a new exit policy or change any model or audit command exit.
This is the same warning-only proxy posture used by D14.

### Ratification reference grammar

At ratification time, the owner should add at least one `eval:` reference to
the pin's existing `evidence_refs` array. I077 recognizes only this exact
grammar:

```
eval:<eval-dir>/runs/<run>.md
<eval-dir> = YYYY-MM-DD-<slug>
<slug>     = [a-z0-9]+(?:-[a-z0-9]+)*
<run>      = [A-Za-z0-9][A-Za-z0-9_-]*
```

`YYYY-MM-DD` must be a real Gregorian calendar date. The reference has no
leading slash, no `.` or `..` path component, no query or fragment, and no
second acceptable spelling. For example:

```
"evidence_refs": [
  "owner:I068",
  "eval:2026-08-30-routing-check/runs/gpt-5-6-sol.md"
]
```

The reference denotes exactly
`docs/evals/2026-08-30-routing-check/runs/gpt-5-6-sol.md` in the audited
repository. `eval:` is a new typed subset of the already-compatible opaque
array. Other opaque references remain valid but do not count as eval evidence.
An old pin with no `eval:` value is valid and gets one absence warning.

The evaluator must use `Lstat` for `docs`, `docs/evals`, the selected eval
directory, `runs`, and the selected run file. Every directory must be a real
directory and the run must be a real regular file. A symlink at any of those
locations is malformed referenced evidence. The evaluator does not resolve it
and never follows a link outside the repository.

### Eligible run and exact model comparison

A referenced run is eligible only when all of these hold:

1. Its `eval:` reference has the grammar above and its selected path exists as
   a regular, non-symlinked file under the audited repository's `docs/evals/`.
2. Its parent `eval.md` is a regular, non-symlinked file and has the existing
   `title`, `created`, `prompt`, and `rubric` front-matter keys. The run has
   the existing `name`, `created`, `model`, `stage`, and `score` keys.
3. `created` in the run is an unquoted or Go-double-quoted `YYYY-MM-DD` date.
   It is not in the future and is at most 90 calendar days old, measured from
   the doctor's injected/current UTC calendar date. Age exactly 90 days is
   fresh. File mtime is never read for freshness.
4. After existing front-matter trimming and optional Go-string unquoting, the
   run's `model` is a nonempty control-free string exactly equal byte-for-byte
   to the pin's `model`. There is no alias, observed-ID, prefix, case-fold,
   whitespace, provider, harness, or effort comparison.
5. The run supplies a valid I077 battery record with `battery_verdict: pass`.

`stage` and `score` remain opaque under ADR 0007. I077 must not branch on
either field. The ordinary `spine eval list` output remains compatible; it may
show the new battery fields only if a later ticket explicitly asks it to.

### I077 battery record

I077 adds an optional pin-evidence profile to a run. It does not make a
battery record mandatory for ordinary evals. Only a run cited by `eval:` must
have these additional scalar front-matter fields:

```
battery_version: 1
battery_verdict: pass
battery_results: invocation=KILLED,wiring=KILLED,flag-honoured=KILLED,column-presence=KILLED,column-order=KILLED,ordering=KILLED,units-labels=KILLED,security-default=REPORT-ONLY,lifecycle=REPORT-ONLY,error-path-behaviour=KILLED
```

`battery_results` is one comma-separated list with no whitespace. It contains
each of these keys exactly once and in the shown order:

```
invocation,wiring,flag-honoured,column-presence,column-order,ordering,
units-labels,security-default,lifecycle,error-path-behaviour
```

The allowed uppercase values are `KILLED`, `SURVIVED`, `NO-SITE`,
`BUILD-ERR`, and `REPORT-ONLY`. `REPORT-ONLY` is permitted only for
`security-default` and `lifecycle`. Those two classes do not decide the
binary verdict. For the other eight classes, `pass` requires `KILLED` for all
eight. `fail` requires a complete valid matrix with at least one of those
eight values not equal to `KILLED`. Any other version, key, order, duplicate,
value, missing field, inconsistent verdict, malformed front matter, or
unreadable/symlinked selected file is malformed evidence.

This is a narrow evidence claim, not a global mutation threshold. The
mutation-battery checklist remains the source for how the ten probes are run,
reported, and explained. I077's consumer only asks whether this cited run
states a complete passing result in the selected grammar. Its consequence is
one doctor warning, never a workflow gate.

### Aggregation and D17 findings

For each configured pin, doctor collects only entries beginning `eval:`. A
pin needs at least one eligible eval run. Each declared `eval:` reference must
be healthy. One healthy reference does not mask a second stale, mismatched, or
failing one. Opaque non-`eval:` entries neither satisfy nor invalidate the
requirement.

D17 is allocated to I077. Every D17 result has severity `warn`, contains one
line of fixed text, and never prints an opaque reference, model ID, host ID,
host-home path, eval body, score, stage, or unreadable-file error. Doctor
sorts pins by `flavor.tier` bytewise and their `eval:` references bytewise. It
appends the resulting D17 findings after current D16 findings. This gives
stable human and JSON order without depending on Go map iteration.

| Condition | D17 path | Exact D17 message |
| --- | --- | --- |
| No `eval:` reference, including a pin with only opaque references | `routing-host.json` | `pin <flavor.tier> has no eligible eval reference` |
| An `eval:` entry fails the reference grammar | `routing-host.json` | `pin <flavor.tier> has a malformed eval reference` |
| A grammatical selected path does not exist | selected repository-relative run path | `pin <flavor.tier> references missing eval evidence` |
| A selected component is a symlink, non-regular, unreadable, has invalid required front matter, invalid created date, a future date, or invalid battery grammar | selected repository-relative run path, or the parent `eval.md` when that is the defective file | `pin <flavor.tier> references malformed eval evidence` |
| A valid run's declared date is older than 90 calendar days | selected repository-relative run path | `pin <flavor.tier> references stale eval evidence` |
| A valid, fresh run's model differs from the pin's model | selected repository-relative run path | `pin <flavor.tier> eval model does not exactly match pinned model` |
| A valid, fresh exact-model run lacks the I077 battery fields | selected repository-relative run path | `pin <flavor.tier> eval evidence has no battery record` |
| A valid, fresh exact-model run declares `battery_verdict: fail` | selected repository-relative run path | `pin <flavor.tier> eval battery verdict is fail` |

The malformed-battery row excludes an absent full record. Absence gets its own
message; a partially supplied or internally inconsistent record is malformed.
One referenced run produces at most one D17 finding, using the table's order
from reference grammar through verdict. A pin with zero `eval:` entries gets
one finding. A pin with multiple bad references gets one finding per bad
reference in sorted reference order.

The existing D7 structural scan still covers every ordinary eval file. A
malformed eval unrelated to any pin remains only D7 behavior. I077 neither
adds D17 for it nor changes D7's severity, wording, scan, or read scope.

## Implementation shape

`internal/eval` gains a narrow pin-evidence reader with an injected UTC date
for deterministic tests. It accepts pin keys, pinned model IDs, and their
already-loaded references as values, so it neither imports host configuration
nor knows how a pin resolves. It performs strict component checks and returns
sanitized typed outcomes, not raw errors.

`internal/doctor` loads host config once for the existing D16 and new D17
checks. Invalid host config continues to return D16's existing error and does
not attempt an evidence judgment. A valid config is passed unchanged to model
resolution and the evidence reader. The doctor formatter maps the typed
outcomes to the exact table above. `internal/hostconfig` keeps `evidence_refs`
opaque and optional. `internal/model`, `internal/audit`, and model CLI behavior
do not change.

The eval README, run template, and mutation checklist document the new
optional profile. A template-content migration records superseded predecessor
lines per `WORKFLOW.md`; it does not make legacy runs invalid for generic eval
commands.

## Requirements attack and resolutions

| Attack | Resolution |
| --- | --- |
| Requiring an eval ref in `hostconfig.Load` would de-ratify every legacy or exceptional pin. | Keep the array optional and opaque in the host schema. D17 reports the missing typed subset only after a valid load. |
| A valid reference might name a path through a symlink or `..` into host data. | The grammar has no traversal spelling, and every selected component is `Lstat`-checked before read. |
| File mtime can be touched to make old evidence look current. | Freshness uses only the declared run `created` date and a fixed 90-calendar-day rule. |
| An alias, observed ID, or model-family prefix can look comparable without proving the pinned ID ran. | Compare the parsed run model to the pin model byte-for-byte after the one documented unquote step. |
| A passing reference could hide another stale or failed reference. | Every declared `eval:` reference must pass. Aggregate all bad references in deterministic order. |
| A malformed unrelated eval could make an unrelated pin look unhealthy. | D17 opens only exact referenced paths. Existing whole-tree D7 remains separate. |
| Parsing `stage` or `score` would violate ADR 0007. | I077 reads new battery fields only and leaves those two values opaque. |
| A low mutation rate has never been a hard failure. | The new complete pass record is an advisory evidence claim for a cited pin, not a workflow threshold or a gate. |
| Adding a doctor warning could change route or audit behavior by accident. | Keep all evaluator calls inside doctor after normal host loading. Add model and audit no-change controls, including exits. |
| Doctor output could leak host-local opaque references or body text. | Return only enum outcomes and repository-relative selected paths. Formatter uses fixed text and never includes raw values. |
| I076 yield records sound relevant but are still a different, blocked evidence source. | Exclude them explicitly. This policy permits only repo-local `docs/evals/` exact-model runs. |

## Out of scope

I077 does not add a ratification CLI, auto-write a host config, repair an eval,
scan a fleet, fetch external evidence, change I076, rate models, change the
mutation runner, add a global threshold, change `spine eval list`, or close
the ticket. I077 remains open until the implementation plan, review, fresh
verification, and final exact-SHA lane evidence are complete.
