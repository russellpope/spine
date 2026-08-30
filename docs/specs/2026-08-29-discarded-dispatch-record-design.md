# Discarded dispatch records (I078)

## Product requirement document

**Status:** approved for implementation

**Ticket:** I078, discarded-dispatch record grammar for audit routing

**Depends on:** I111 and I102 are already on main. Their source-versus-flavor
split and first-prompt-only team pairing are the baseline for this work.

## Goal

Let `spine audit routing` report an abandoned lower-tier prototype as
`discarded-with-reason` without allowing that declaration to hide a separate
lower-tier dispatch that was used for the same ticket.

## The requirement attack

The tempting rule, `DISCARDED <ticket> tier:<tier>`, is unsafe. Audit evidence
is often plural: a ticket can have a prototype, a retry, and a worker actual.
That rule would excuse every routine token for a primary ticket, including a
later landed routine dispatch. It is rejected.

The audit has no reliable link from a transcript event to a merged diff. A
model token does not name files edited, a commit, a branch, or a patch hash.
Trying to infer landed work from a git diff would be both expensive and false:
two agents can make the same edit, and a worker can change files that are later
rewritten. I078 must not claim that it can prove a prototype did not land.

The smallest stable correlation already retained by the raw inputs is the
dispatch event inside an immutable transcript:

| Reader | Transcript identity | Dispatch event identity | Status |
| --- | --- | --- | --- |
| Claude layout | session JSONL basename | `tool_use.id` for the `Task`, `Agent`, or recognized team-spawn Bash event | already parsed as `dispatch.toolUseID` |
| Codex layout | `session_meta.payload.session_id`, falling back to `id` | `response_item.payload.call_id` for the `spawn_agent` or command event | present in raw JSON but not retained today |

Each key is meaningful only with its source and session. The implementation
must carry the three-part identity through dispatch parsing, linked Claude
subagent evidence, and `evidenceToken` judgment. A Codex worker actual is
currently linked only to a root session, not to its parent `call_id`; it has no
usable I078 identity and must remain unexcused. That fail-closed boundary is
intentional.

This gives a surgical guard, not a diff-attribution claim. A discarded record
can cover one identified event only. A distinct landed lower-tier event for the
same ticket remains `silent-descent`. An operator who falsely labels the very
event that landed as discarded can still lie, just as an operator can today
write a false `ESCALATION` or `FALLBACK` record. The audit makes that claim
visible and limits its blast radius; it cannot establish provenance that the
inputs do not contain.

## Decision

Add this exact, one-line ledger grammar in
`.superpowers/sdd/progress.md`:

```text
DISCARDED <ticket-id> source:<claude|codex> session:<session-id> dispatch:<event-id> tier:<mechanical|routine|primary|fallback> reason: <one line>
```

The space-separated prefix has exactly six tokens before `reason:`. `source`,
`session`, `dispatch`, and `tier` use the shown literal field names. The
`session` and `dispatch` values are non-empty and contain no whitespace. The
reason is non-empty after trimming. The parser does not accept reordered
fields, aliases, quoted values, a spaced `reason :`, an omitted field, an
unknown source or tier, or extra prefix fields.

Example:

```text
DISCARDED I078 source:claude session:4c8d2e1e dispatch:toolu_014 tier:routine reason: prototype was discarded before the primary implementation began
```

The record is written when the prototype is discarded, or immediately after
that decision in the same effort, before verification. The audit has no
trusted per-event timestamp contract, so it must not pretend to prove that
ordering. The record's exact identity and reason are the auditable declaration.
A later, retrospective record is still parsed, but it gets no ticket-wide
effect.

`ESCALATION` and `FALLBACK` keep their exact grammar and current behavior.
`DISCARDED` is not an escalation, does not influence `pickTier`, and never
changes fallback ambiguity resolution.

## Correlation and judgment

The implementation adds a private identity value equivalent to:

```go
type evidenceIdentity struct {
    source   string
    session  string
    dispatch string
}
```

It is attached only when the reader can populate every member exactly. Claude
main-session dispatches use the session basename plus `tool_use.id`. A linked
Claude subagent carries its sidecar `toolUseId` and its parent session basename.
Codex dispatch records retain the response item's `call_id` with the root
session id. Existing root-only Codex worker scans, guardian and near-miss
material, records without an event id, and malformed transcript events carry
no identity.

After `tiersOf` and `pickTier` resolve an evidence token's actual tier, the
normal verdict order remains:

1. A matching declared tier is `match`.
2. The current fallback and `ESCALATION` paths run unchanged.
3. Only an otherwise lower-than-declared token can consult `DISCARDED`.
4. It becomes `discarded-with-reason` only when one usable record matches the
   ticket id, source, session, dispatch id, and resolved actual tier.
5. Every other lower token remains `silent-descent`.

This order preserves the existing reasoned-descent rule and makes the landed
work guard load-bearing. In particular, a primary ticket with one discarded
routine event and one later routine event gets both per-token results; worst
token aggregation returns blocking `silent-descent`, not a passing result.

## Parsing, ambiguity, and output

`readLedger` continues to tolerate a missing ledger. It gains discarded-record
parsing and returns diagnostics to `Report.Warnings` for this new grammar.
Existing malformed `ESCALATION` and `FALLBACK` handling remains byte-for-byte
compatible.

- A line beginning with `DISCARDED` that misses the exact grammar is ignored,
  produces one non-blocking warning naming the safe line number, and excuses
  nothing.
- Two parsed records with the same complete identity and tier are duplicates.
  Neither is usable; emit one non-blocking duplicate warning for that key.
- A parsed record that matches zero eligible evidence tokens, or more than one,
  is unused and emits one non-blocking warning. It may be stale, filtered out,
  or too coarse. It excuses nothing.
- An identity-less token cannot match a discarded record. It stays
  `silent-descent` when lower tiered.

`VerdictDiscardedWithReason` has the exact string `discarded-with-reason`.
It is visible on the ticket row with the model, resolved tier, and recorded
reason. Its severity equals `escalated-with-reason`: advisory and nonblocking.
The CLI retains its one-row-per-ticket format. `Report.Blocking` still blocks
when any unexcused token is `silent-descent`; it must not treat a discarded
row as a pass that hides a worse token.

## Compatibility and generated documentation

The public report gains one additive verdict name. The text of all existing
verdicts, exit codes, `ESCALATION`, `FALLBACK`, no-transcript handling, D28
repo qualification, I111 source/flavor behavior, and I102 first-prompt-only
pairing stays unchanged.

This is user-facing workflow grammar. Update the current repository
`WORKFLOW.md` and `templates/current/WORKFLOW.md.tmpl`, then bump
`templates/VERSION` from 11 to 12. The bump is required because new and updated
repositories must receive the record contract through `init`, `adopt`, and
`update`. The implementation must register the exact predecessor workflow
lines in `internal/update`'s superseded-line set and add a generation-11 to
generation-12 migration test. It must not hand-edit an arbitrary fleet repo or
claim that a template bump migrates an existing ledger record.

`CHANGELOG.md` receives a concise entry after implementation. The I078 ticket
is closed only with actual commit ids and review evidence.

## Non-goals

- No git-diff, commit, branch, patch, filesystem, or semantic attribution of
  a transcript to landed work.
- No ticket-wide discarded rule, wildcard identity, tier-only grammar, or
  default excuse for a source/session.
- No change to `ESCALATION`, `FALLBACK`, model resolution, I111 flavor
  derivation, I102 pairing, or I074 heterogeneous verdict design.
- No attempt to excuse a root-only Codex worker actual until a future change
  can retain an exact parent dispatch correlation.
- No template fleet sweep or installed-binary deployment.

## Acceptance criteria

1. A primary-tier ticket with one exact discarded Claude routine dispatch
   reports `discarded-with-reason`, displays the reason, and exits zero.
2. The same transcript without the record reports `silent-descent` and exits
   one. This is the required negative control.
3. A ticket with a discarded routine event and a different routine event that
   represents landed work still reports `silent-descent` and exits one.
4. A record with the wrong source, session, dispatch id, or tier cannot excuse
   evidence. Each mismatch remains visible as `silent-descent` plus a warning.
5. Malformed, duplicate, and one-to-many discarded records never suppress
   evidence and issue their specified non-blocking diagnostics.
6. A root-only Codex worker actual remains unexcused even when a same-ticket
   discarded record names its session; a direct Codex dispatch with a retained
   `call_id` can be discarded only by that exact identity.
7. Existing `ESCALATION`, `FALLBACK`, I111, I102, D28, and I090 tests stay
   green without changed expectations.
8. Fresh scaffolded and updated workflow documentation shows the exact
   `DISCARDED` grammar at template generation 12, while a hand-edited
   predecessor line remains an unrecognized local edit.

## Requirements-attack resolutions

| Attack | Resolution |
| --- | --- |
| A tier-only record excuses every low-tier token for a ticket. | Rejected. Matching requires ticket, source, session, dispatch id, and actual tier. |
| A lower-tier prototype and landed retry share one ticket. | The retry has a different event id and remains blocking. |
| A linked Codex worker has only root-level correlation. | It is identity-less for I078 and cannot be excused. |
| A record contains a typo or is copied twice. | It cannot suppress evidence and emits a warning. |
| A record claims discarded work actually landed. | The audit cannot prove landed provenance. It treats the explicit, reasoned record as a narrow operator assertion and never widens it by ticket or tier. |
| Template docs drift from the parser. | Generation 12, scaffold/update tests, and exact superseded-line registration make the grammar part of the emitted contract. |
