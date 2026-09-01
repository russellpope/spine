---
title: "Routing yield: review records and read-only yield report"
tickets: I076
created: 2026-08-30
status: accepted-for-documentation
---

# Routing yield review record (I076) — design

## Product requirement document

**Status:** accepted design. This document and its plan may land now. The
original I073 prerequisite required I073 to be fixed and independently
verified at its exact final implementation SHA, including a standalone
`maipipe run full --wait` result at that SHA. That historical requirement is
superseded only by the owner-directed batch-lane amendment below.

**Ticket:** I076, Routing-yield forward build — REVIEW record line + `spine
yield`.

**Decision:** add an explicit review-time record to
`.superpowers/sdd/progress.md` and a read-only `spine yield` command. The
command aggregates recorded task-gate outcomes by canonical harness, actual
model ID, and effective implementer tier. It never reconstructs outcomes from
review filenames, transcripts, ticket history, commit history, or model-name
guesswork.

I076 reports two separate recorded series: task-gate outcomes and final-review
outcomes. Only the task-gate series has first-pass and rework rates. The final
series contextualizes that rate; it is never folded into a task cell or used to
decide routing policy, recommend a replacement model, or invent a rating
algorithm.

## Owner-directed batch-lane amendment (2026-08-31)

This amendment changes gate sequencing, not product behavior or the accepted
record contract. I073's prerequisite is durably satisfied by fixed product SHA
`46b2324`, the PASS all-20 primary fleet result, closure commit `dcb1c3e`, and
the fresh primary post-fleet PASS. I076 may therefore consume I073's canonical
`harness` terminology and its bounded compatibility surface now.

The owner-directed open-ledger sequence is: finish I073 post-fleet acceptance,
finish I076's gated tail and close it as `fixed, pending batch ship`, then
complete the final whole-branch review, run the routing audit, and create a
fresh handoff. Once every included ticket has a fixed, blocked, or surfaced
disposition, run one `maipipe run full --wait` at the resulting exact final
main SHA. No standalone I073 lane and no standalone I076 lane is required or
claimed. The single batch-final lane is the ship verdict. Any commit after that
lane invalidates its result and requires the lane to be rerun at the new exact
SHA.

This sequencing amendment does not relax the grammar, the no-inference and
privacy boundary, confidence thresholds, fleet isolation, task/final-series
separation, or the prohibition on a `flavor` alias.

## Why a forward record is necessary

The feasibility survey found that filename patterns miss known rework and that
ticket frontmatter records current status rather than review history. Existing
`ESCALATION` and `FALLBACK` lines already expose escalation frequency, but
they do not contain task-gate verdicts. A review record is the smallest durable
source for first-pass acceptance and rework counts.

The pre-I073 proposal used `flavor/tier`. That spelling would freeze a retired
term in a new public ledger grammar. I076 therefore uses I073's canonical
`harness` term and does not accept a `flavor` alias. This is why I073's
verified completion, rather than merely its merge or source-level rename, is
the implementation prerequisite.

## Record grammar and validity

One physical, column-zero line is written for a task-gate verdict:

```text
REVIEW <ticket-id> harness:<harness> model:<actual-model-id> tier:<effective-tier> round:<positive-integer> verdict:<accepted|needs-fixes> scope:task
```

An attributable final-review verdict uses the same ordered fields with
`scope:final`. A final-review condition that cannot honestly be assigned to
any ticket, harness, model, or tier uses this bounded form:

```text
REVIEW - harness:- model:- tier:- round:<positive-integer> verdict:needs-fixes scope:final condition:<opaque-condition-id>
```

The fields are ordered and all are required. Literal one-space separators are
part of the grammar. Values are single non-empty tokens with no whitespace or
quoting. Where a field is known, `harness` is one of I073's canonical table
keys (`claude`, `codex`, `pi`, or `openweights`) and `tier` is one of
`mechanical`, `routine`, `primary`, or `fallback`. The only permitted
unknown value is the literal `-`, under the scope-specific rules below.
`round` is base-10, has no sign or leading zero, and is at least 1. `model`
is the actual dispatched model ID as the reviewer records it. It is opaque:
the parser neither resolves it through the current defaults table nor derives
it from a filename or transcript.

For `scope:task`, every identity field is known and `-` is invalid. For an
attributable `scope:final` line, `ticket-id` is known; `harness`, `model`,
and `tier` may individually be `-` only when that particular value is
genuinely unavailable. The bounded unattributable form requires `-` in all
four identity fields, permits only `verdict:needs-fixes`, and requires the
final `condition:<opaque-condition-id>` field. The condition ID is a
non-empty, non-quoted token with no meaning inferred by spine. A clean final
review has no unattributable record to write.

The aggregation key is exactly `(harness, model, tier)`. The record's tier is
the effective implementer tier after any applicable model-tier `ESCALATION`.
The reviewer writes that resolved tier. `spine yield` does not reverse-engineer
or overwrite it from a later ledger line.

Final records are a separate series. An attributable final record reports a
ticket's final-review verdict, but no final record contributes to a
harness/model/tier task denominator. The unattributable form reports only a
condition that requires fixes, without manufacturing a ticket or route.

For each repository, task and attributable-final identities are
`(scope, ticket-id, round)`; an unattributable-final identity is
`(scope, condition-id, round)`. Each identity must have exactly one valid
REVIEW line. An exact duplicate counts once and emits a duplicate warning. Two
non-identical valid lines for an identity, or a malformed candidate line, are
not repaired by choosing the last line. The conflicted identity is excluded
from its series and appears in ignored-record counts.

A task ticket's valid review sequence is contiguous: it has exactly one
unambiguous task record for every round from 1 through its maximum round. Every
round after 1 must follow a `needs-fixes` verdict, and an `accepted` verdict
ends the sequence. A missing predecessor, a gap, a round after acceptance, or
a conflicting round invalidates the whole task sequence. Yield excludes that
ticket's task records from all task cells, counts one ignored task sequence,
and returns exit 1 rather than inventing a rework count.

## Parser ownership and compatibility

`internal/yield` owns parsing of REVIEW records, their identity checks, and
yield aggregation. It receives physical lines from only the configured
progress-ledger path and returns typed records plus non-sensitive diagnostics.
`internal/audit` remains the owner of routing-audit ledger interpretation. Its
current `readLedger` behavior and routing verdicts must not change as a side
effect of I076.

The yield package also has a narrow, read-only parser for the published
model-tier `ESCALATION <ticket> <from>-><to> reason: <one line>` and
`FALLBACK <ticket> reason: <one line>` forms. It counts only valid complete
records. It never treats effort escalations as model-tier escalations, and it
does not use either record to assign an escalation to a yield cell because
those legacy records do not carry an actual model ID.

Compatibility is intentionally one-way:

- A missing ledger or a ledger with no REVIEW lines produces zero counts. It
  needs no migration.
- Existing `ESCALATION`, `FALLBACK`, `DISCARDED`, prose, and historical
  lines remain readable by their present consumers. Unknown lines are ignored.
- A legacy REVIEW line using `flavor`, a loose field order, quoting, or an
  added field is malformed for I076 and is not counted. The sole allowed
  added field is `condition:` on the bounded unattributable-final form.
  There is no flavor alias.
- I076 does not modify archived ledgers or add a template-generation or fleet
  write. New workflow wording may be considered only after I073's generation
  14 sequence has finished.

## Read scope and fleet isolation

The canonical invocation is:

```text
spine yield [--dir D] [--json]
spine yield --fleet P [--json]
```

All flags precede positionals. There are no positionals. `--dir` defaults to
`.` and reads only `D/.superpowers/sdd/progress.md`. It does not read sibling
ledgers, handoffs, ticket files, transcripts, git history, host configuration,
or user-home data.

`--fleet P` is mutually exclusive with `--dir`. It follows the existing
fleet reader convention of scanning only immediate, non-hidden children of
`P`, in lexical repository-name order. A child contributes only when its
`.git` is a directory. That excludes linked worktrees, whose `.git` is a
file, so the same ledger cannot be counted through a primary and a worktree.
The scan never recurses, follows no symlinks, and does not include `P` itself.

Each eligible child is isolated. A missing progress ledger is a zero-count
repository. An unreadable ledger or failed child inspection emits one
repository-scoped diagnostic and leaves that child out of aggregate counts;
other repositories still report. An unreadable `--fleet` parent, an invalid
explicit `--dir`, or invalid flags are command errors and stop before a report.

Within fleet aggregation, task and attributable-final identity is
`(repo, scope, ticket-id, round)`; unattributable-final identity is
`(repo, scope, condition-id, round)`. That keeps same-number tickets in
different repositories and separate final conditions from colliding. Fleet
output sorts aggregate cells by harness, model ID, then tier, and repository
status rows by repository name. Sorting and conflict-exclusion happen before
formatting in both text and JSON.

## Counts, confidence, output, and exits

For every task aggregate cell, `n` is the number of complete, unambiguous
`round:1` task sequences. `accepted_first_pass` and
`needs_fixes_first_pass` partition `n`. `rework_rounds` is the sum of
`max(round)-1` for those sequences, attributed to the cell of each ticket's
round-1 record even if a later review used a different actual model or tier.
This measures the rework caused by the initial reviewed dispatch without
creating a second first-pass denominator. Separately, the report totals final
attributable `accepted` and
`needs-fixes` verdicts and unattributable final `needs-fixes` conditions.
Final totals have no rate and never enter task cells. The command also prints
totals for valid REVIEW lines, ignored REVIEW identities, valid model-tier
ESCALATION records, and valid FALLBACK records. Escalation and fallback totals
are report-wide counts only. I076 does not pretend they belong to a
harness/model/tier cell.

Text output always begins with a scope summary and a totals line, even when no
cell is countable. It then prints one deterministic cell row per key and, in
fleet mode, one deterministic repository status row per eligible child. JSON
contains the same task and final totals, cell rows, repository statuses, and
confidence state. Neither format prints a ledger line, review reason,
condition-ID-derived description, transcript content,
filesystem path below the repository name, or a guessed model family.

The only rate is `accepted_first_pass / n`, rendered as a percentage to one
decimal place with these fixed confidence states:

| `n` | Rate output | Confidence | Process exit |
| --- | --- | --- | --- |
| 0-19 | `refused` | `insufficient` | 1 |
| 20-39 | percentage | `low-confidence` | 0 if no other refusal or data error |
| 40 or more | percentage | `stated` | 0 if no other refusal or data error |

For a multi-cell report, exit 1 if any cell is refused, any REVIEW identity is
ignored, or any fleet child had a read/inspection error. Exit 2 is reserved for
usage and root-scope errors. Exit 0 means every reported cell cleared 20 and
the report had no ignored REVIEW data or isolated fleet error. A
low-confidence percentage is visible evidence, not a policy claim.

## Privacy and failure boundaries

The only persisted inputs are operator-written ledger fields. The command does
not open transcripts and does not learn model IDs from session data, filenames,
branches, prompts, ticket text, or route configuration. It retains no cache
and writes no file. Diagnostics identify a repository and line number at most;
they never echo a malformed line or its free text. The model ID itself is shown
only because it is the requested aggregation key.

One bad REVIEW record cannot change a valid neighbor's cell, and one bad fleet
child cannot erase counts from other children. A malformed or duplicated record
is visible and excluded. A missing ledger is ordinary zero evidence, never an
invented accepted outcome.

## Requirements attack

| Attack | Resolution |
| --- | --- |
| Review filenames, transcripts, or current ticket state seem easier to mine. | Do not mine them. Only explicit REVIEW records create outcome evidence. |
| A new line says `flavor` while I073 is still in flight. | Use only `harness`; block code on I073's independently verified exact SHA. |
| A model lookup can "correct" a mistyped actual ID. | Treat the recorded model ID as opaque and reject malformed syntax only. |
| Replayed or contradictory lines can inflate a denominator. | Deduplicate exact repeats, exclude conflicting identities, count exclusions, and return exit 1. |
| A round-two line appears without a complete review history. | Require a contiguous task sequence, stop after accepted, and count rework as max round minus one. |
| Final review failures make task-gate acceptance look better than it is. | Report attributable and unattributable final outcomes separately beside task rates; never fold them into a task denominator. |
| Existing ESCALATION/FALLBACK lines manufacture model-cell rates. | Count them only as report-wide totals. Do not assign them to cells. |
| Fleet scans double-count linked worktrees or hide a bad repository. | Scan immediate primary `.git` directories only; print every eligible child's status and isolate failures. |
| A low-sample percentage becomes a routing instruction. | Refuse below 20, label 20-39 low confidence, and add no recommendation or rating rule. |
| Diagnostics leak reviewer notes or prompt data. | Print aggregate counts and bounded line diagnostics only; never print line text or read transcripts. |

## Acceptance criteria

1. REVIEW has the published canonical-harness, actual-model-ID, effective-tier,
   round, task, attributable-final, and bounded unattributable-final grammar;
   task sequences are contiguous and terminate on acceptance; malformed,
   legacy-flavor, duplicate, conflicting, and noncontiguous inputs have
   deterministic visible treatment.
2. `spine yield` reads only the selected repository progress ledger, and
   `--fleet` reads only immediate primary child repositories with isolated
   failures and no linked-worktree double count.
3. Counts, sorting, text output, JSON output, confidence labels, and exit codes
   are deterministic. Counts print even when every rate is refused.
4. Task rates are absent below 20, low-confidence at 20-39, and stated at 40
   or more. Final outcomes print separately with no rate. The command adds no
   rating, recommendation, or model inference.
5. Existing model-tier ESCALATION/FALLBACK records are counted without a new
   record type and without unsupported per-model attribution.
6. The implementation reads no filenames or transcripts to derive outcomes and
   prints no raw ledger content, transcript content, or reviewer reason.
7. Before I076 proceeds, I073 has durable prerequisite evidence: fixed product
   SHA `46b2324`, a PASS all-20 primary fleet result, closure `dcb1c3e`, and a
   fresh primary post-fleet PASS. This makes canonical harness terminology and
   its bounded compatibility surface available to I076; it does not authorize a
   `flavor` alias.
8. Focused red-green tests, task review, independent verification, and a final
   requirements attack pass before I076 closes as `fixed, pending batch ship`.
   The ship verdict is delegated to the owner-directed batch-final `maipipe run
   full --wait` after every included ticket has a fixed, blocked, or surfaced
   disposition, and after final whole-branch review, routing audit, and a fresh
   handoff, at that batch's exact final SHA. No standalone I076 lane is required
   or claimed; a post-lane commit invalidates the lane and requires a rerun.
