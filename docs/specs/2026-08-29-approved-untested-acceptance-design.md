---
title: "Approved-untested acceptance records"
tickets: I050
created: 2026-08-29
status: approved
---

# Approved-untested acceptance records (I050) design

## Status and authority

This PRD is the binding product and implementation contract for I050. The lead
approved Option A from `.superpowers/sdd/I050-worker3-grill.md`: one shared
ticket-record validator, an estate-wide doctor adapter, and a cursor-scoped
`audit stages` adapter. Where the issue used "waived" and
"approved-untested" interchangeably, this PRD resolves the feature to the
literal state `APPROVED-UNTESTED`.

The doctor finding ID is `D15`. Current source already assigns `D14` to I108's
Go toolchain-skew advisory at commit `3eae6e8`; `D15` is the first unclaimed
numeric ID as of this PRD.

## Problem

An unchecked acceptance criterion currently has one machine-visible meaning:
it is unfinished. In practice, an owner may accept a criterion without a test
because a required environment is unavailable or a follow-up ticket owns the
deferred run. The decision now survives only as prose in a handoff or review
note. `spine doctor`, `spine audit stages`, and a later ticket reader cannot
distinguish that decision from a silently skipped check.

Spine needs an opt-in disclosure record on the affected criterion. The record
must be strict enough to catch incomplete annotations, but it must not claim to
authenticate an approver or change the meaning of every ordinary unchecked
box in the existing fleet.

## Goals

- Define one exact, single-line marker grammar for an acceptance criterion that
  remains applicable and unchecked but was consciously accepted without a
  test.
- Keep the record on the ticket and under its acceptance-criteria heading.
- Require durable, repository-relative provenance and verify that the
  referenced artifact file exists.
- Make malformed records visible through `spine doctor` and
  `spine audit stages` without adding a new workflow stage or changing stage
  blocking rules.
- Ship the convention through template generation 12 and safe
  config-preserving migration.
- Preserve byte-for-byte command output and exit behavior when no candidate
  marker exists.

## Non-goals

- Proving that the named approver is a real person, owns the repository, or
  approved the referenced text.
- Resolving or validating the Markdown fragment inside the approval artifact.
- Adding signatures, HMACs, session authentication, authorization policy, or
  network lookups.
- Defining a separate `WAIVED` state. A requirement that no longer applies
  must be removed or changed in the ticket or PRD.
- Checking that a reason names a follow-up ticket, parsing such a ticket ID, or
  checking that the follow-up exists.
- Turning unchecked acceptance criteria into a general doctor or audit gate.
- Storing acceptance decisions in `.superpowers/sdd/progress.md`, the stage
  cursor, a new frontmatter list, or a second ledger.
- Rewriting or auto-annotating existing ticket files.
- Adding a command, a workflow stage, an ADR, or a fleet sweep.

## Binding marker grammar

The canonical record occupies one physical Markdown line:

```text
- [ ] <criterion> -- APPROVED-UNTESTED <YYYY-MM-DD> by <approver> ref: <docs/YYYY-MM-DD-artifact.md#anchor> reason: <one-line reason>
```

Canonical example:

```text
- [ ] Exercise the hardware failover path -- APPROVED-UNTESTED 2026-08-29 by owner ref: docs/handoffs/2026-08-29-i050-approval.md#hardware-failover reason: lab hardware unavailable; I123 tracks the deferred run
```

Every byte-level rule below is binding:

1. The canonical line starts at column 0 with the exact bytes `- [ ] `.
   Tabs, indentation, a checked state, alternate bullet characters, or
   different checkbox spacing are invalid.
2. `<criterion>` is nonempty after trimming ASCII spaces and tabs.
3. The criterion and marker are separated by the exact ASCII bytes ` -- `.
4. `APPROVED-UNTESTED` is a case-sensitive literal.
5. The date token has exactly the shape `YYYY-MM-DD` and parses as a real
   Gregorian calendar date. The date may be in the future; I050 adds no clock
   policy.
6. The exact delimiters around the remaining fields are one ASCII space,
   `by `, ` ref: `, and ` reason: ` as shown above.
7. `<approver>` is one nonempty, whitespace-free token. It records the name
   supplied by the author; spine does not interpret or authenticate it.
8. The reference is one nonempty, whitespace-free token with exactly one
   base-path/fragment split at its first `#`. Both parts are nonempty.
9. `<one-line reason>` is the trimmed remainder of the physical line and is
   nonempty. It may contain spaces, punctuation, additional colons, and ticket
   IDs. It cannot continue onto another line.
10. The line appears after an exact column-0 heading
    `## Acceptance criteria` and before the next column-0 level-one or
    level-two Markdown heading. A level-three or deeper heading does not end
    the section. A marker under any other heading is invalid.
11. More than one canonical record may appear on a ticket. Each line is a
    separate record and contributes one to the count.

### Candidate detection

Candidate detection is intentionally broader than successful parsing. A
physical line is a candidate when it has optional leading spaces or tabs,
then a hyphen, optional spaces or tabs, a bracketed checkbox field, and later
contains the case-sensitive byte substring `APPROVED-UNTESTED`. This catches
checked, indented, and spacing-damaged attempts so they cannot masquerade as
valid exceptions.

A candidate either produces one valid record or one aggregated problem. The
problem carries the ticket's slash-form repository-relative path, the 1-based
physical line number, and every failed requirement on that line. One malformed
line never fans out into several D15 findings or several audit warnings.

Plain prose, fenced examples that are not checklist-shaped, lowercase
`approved-untested`, and unchecked checklist items with no uppercase sentinel
are not candidates and produce no output.

## Approval-reference and path validation

The reference base path is accepted only when all of these checks pass:

- It uses `/` separators and is a relative path beginning with `docs/`.
- `path.Clean(base)` equals the supplied base path. Empty components, `.`,
  `..`, a leading slash, a volume-qualified path, and backslashes are invalid.
- The path ends in the case-sensitive suffix `.md`.
- Its basename begins with `YYYY-MM-DD-`, and that prefix parses as a real
  calendar date.
- Joining the base path to the repository root cannot escape that root.
- Symlink evaluation cannot escape the resolved repository root.
- The resolved target exists and is a regular file. A directory, device,
  socket, broken link, or outside-root symlink is invalid.
- The fragment after `#` is nonempty. Spine records it verbatim and does not
  resolve it against Markdown headings.

The reference date does not need to equal the marker date. Approval recorded
only in a live session must first be written to a dated Markdown artifact under
`docs/`, such as a review note, spec, or handoff.

These checks establish durable provenance. They do not establish authenticity.
A repository author can still write a false approver token, false prose, or a
misleading fragment. I050 deliberately stops at a verifiable local file
reference because the repository has no identity or signature seam.

## Ticket-local scope

The acceptance decision lives only on the qualifying checklist line in a real
`docs/issues/I*.md` ticket. `README.md`, `_template.md`, directories, and files
whose names do not start with `I` and end with `.md` are never scanned as live
tickets. Doctor scans tickets at every status, including `fixed`, `wontfix`,
and `superseded`, because malformed historical provenance is still malformed.

`audit stages` uses the cursor only to choose tickets. It resolves
`cursor.tickets` with the existing `internal/stages.resolveTicketIDs` grammar,
then scans only ticket files whose frontmatter `id` matches those concrete IDs.
The cursor never stores the record. A marker on an unselected ticket does not
contribute to the audit summary. If the ticket expression is unresolvable,
the existing cursor warning remains and acceptance scanning does not guess an
estate-wide scope.

## Shared validation model

One new `internal/acceptance` package owns candidate detection, section scope,
canonical parsing, calendar checks, approval-reference validation, artifact
existence, ticket discovery, and counting. Doctor and stages consume its
results; neither command implements a second parser.

The package exposes records with ticket path, line, criterion, date, approver,
reference, and reason; problems with ticket path, line, and aggregated failed
requirements; and a summary containing valid records and problems. Candidate,
valid, and invalid counts derive from that summary rather than being maintained
independently.

## Doctor behavior

I050 adds D15, `approved-untested acceptance records`.

- `doctor.Run` scans every real ticket in `docs/issues/` through the shared
  validator.
- A valid record is silent. Doctor adds no info finding or success line.
- Each invalid candidate produces one D15 `warn`. `Finding.Path` is the
  ticket's slash-form repository-relative path. `Finding.Message` includes
  the 1-based line number and the aggregated failed requirements.
- A syntactically complete candidate whose approval artifact is missing,
  outside the repository, or not a regular in-repository file is invalid and
  produces the same D15 warning shape.
- Doctor's existing severity contract is unchanged. Any warn, including D15,
  makes `spine doctor` exit 1. D15 never has severity `error` or `info`.
- `--json` emits the existing `Finding` fields without schema changes:
  `id`, `severity`, `path`, and the line-bearing `message`.
- No candidate marker means no D15 finding and no exit-code change.

"A valid record passes doctor" means it adds no D15 finding. It does not
promise that unrelated checks produce no finding.

## Audit stages behavior

`internal/stages.Report` gains an acceptance summary. Its zero value means no
candidates, no valid records, and no invalid records. `Report.Blocking()` must
not inspect the summary or its problems.

For the concrete ticket IDs resolved from the active cursor:

- Valid records are clean and nonblocking.
- Each invalid candidate prints one `warning:` line to stderr with its ticket
  path, 1-based line, and aggregated failed requirements.
- If one or more candidates exist, stdout prints this exact summary after all
  stage rows and before the existing handoff line:

  ```text
  acceptance: approved-untested=<valid-count> invalid=<invalid-count>
  ```

- With zero candidates, stdout omits the acceptance line. Existing stdout,
  stderr, and exit status remain byte-for-byte unchanged.
- Invalid records alone do not change audit exit 0. Existing malformed or
  noncanonical cursor checks, stage contradictions, and handoff checks remain
  the only blockers.
- An existing blocker remains blocking regardless of acceptance counts.

The command asymmetry is intentional. Doctor is the estate-wide hygiene
command and its longstanding warn contract yields exit 1. `audit stages` is a
cursor-scoped report whose acceptance warning is advisory, so it does not
change `Report.Blocking()`.

## Template generation 12 and migration

I050 bumps `templates/VERSION` from 11 to 12.

- `templates/current/WORKFLOW.md.tmpl` gains an `Acceptance exceptions`
  section adjacent to the ESCALATION grammar. It contains the canonical line
  grammar, ticket-local placement rule, doctor and audit posture, summary
  shape, and provenance-not-authentication limit.
- `templates/current/issue.tmpl.md` gains an empty
  `## Acceptance criteria` section plus an author note pointing to the exact
  grammar in `WORKFLOW.md`. It must not contain a live checklist-shaped
  `APPROVED-UNTESTED` example, because doctor would treat the template as
  documentation only today but copied tickets would inherit a live candidate.
- `templates/current/issues-README.md` gains a short semantics note and points
  to `WORKFLOW.md` as the only grammar authority. It does not duplicate the
  canonical grammar.
- Repository mirrors `WORKFLOW.md`, `docs/issues/_template.md`, and
  `docs/issues/README.md` receive the same generated content.
- `AGENTS.md` and `CLAUDE.md` receive only their ordinary generation-12 marker
  stamps through `spine update`; their managed prose does not change for I050.
- Fresh non-knowledge scaffolds emit generation 12 and the issue-ledger
  convention. The `knowledge` profile receives the WORKFLOW wording but still
  does not gain `docs/issues/` or issue templates, preserving its manifest.
- A pristine generation-11 repository migrates without unrecognized lines and
  is idempotent after the write. A locally edited managed file still reports
  `SkippedUnrecognized` unless the owner chooses the existing force behavior.
- Existing ticket files are never rewritten.
- Historical generation-11 fixtures remain stamped 11. Assertions for the
  compiled current generation move to 12, and the future-generation refusal
  moves from 12 to 13.

All generation-12 content changes are additive. No emitted predecessor line is
removed or reworded, so I050 adds no `supersededLines` entry. If implementation
cannot remain additive, it must add each exact removed predecessor to
`internal/update.supersededLines` in the same commit and obtain a spec-review
resolution before proceeding.

## Compatibility

- Repositories with no uppercase checklist candidate behave exactly as before.
- Ordinary unchecked boxes remain ordinary unchecked boxes. Spine neither
  warns on them nor treats them as incomplete stage evidence.
- Existing checked criteria and ticket status conventions are unchanged.
- Existing doctor formatting and JSON schema are unchanged; only D15 findings
  are additive when invalid candidates exist.
- Existing `audit stages` formatting is unchanged unless at least one scoped
  candidate exists.
- Existing cursor grammar, stage evidence, handoff rules, and blocking logic
  are unchanged.
- Existing approval prose without the canonical marker is ignored.
- The implementation uses the Go standard library only, consistent with ADR
  0001.

## Requirements-attack resolutions

The grill found and resolved the following ambiguities. These resolutions are
part of the contract.

1. **Waived versus approved-untested.** The feature records only
   `APPROVED-UNTESTED`. The criterion still applies and stays unchecked.
2. **Warn versus block.** D15 is a doctor `warn`, which preserves doctor's
   existing exit 1. The audit warning does not affect `Report.Blocking()`.
3. **"Passes doctor."** A valid record creates no D15 finding. Unrelated
   doctor findings remain possible.
4. **Approval reference.** Spine checks that a clean, dated, in-repository
   Markdown base file exists and is regular. It does not authenticate the
   approver or fragment.
5. **Record location.** The decision stays on the ticket criterion, never in
   the progress ledger or cursor.
6. **Checked markers.** `[x]` plus `APPROVED-UNTESTED` is contradictory and
   invalid.
7. **Detached markers.** A candidate outside the exact acceptance section is
   invalid.
8. **Malformed attempts.** Checklist-shaped uppercase candidates are warned
   about and never counted as valid. Ordinary prose is ignored.
9. **Multiple records.** They are allowed and counted per physical line.
10. **Deferred-ticket references.** Authors should name a follow-up in the
    reason when one exists, but I050 does not parse or validate it.
11. **Migration.** Generation 12 is additive, opt-in per criterion, and never
    rewrites existing tickets. There is no fleet sweep.
12. **Policy ownership.** ADRs 0002, 0004, 0005, and 0014 already govern
    regeneration, compiled generations, doctor severity, and audit-stage
    blocking. I050 needs no new ADR.

## Acceptance criteria

1. The shared validator accepts the canonical example only when its dated
   Markdown base file exists as a regular file under the resolved repository
   root, and returns every parsed field with correct ticket path and 1-based
   line attribution.
2. Missing criterion, date, approver, reference, fragment, or reason;
   impossible dates; altered delimiters; checked state; indentation; malformed
   checkbox spacing; and placement outside `## Acceptance criteria` each
   produce exactly one aggregated invalid problem for the candidate line.
3. Absolute, traversal, backslash, non-`docs/`, non-Markdown, undated,
   missing, directory, broken-link, and outside-root-symlink approval targets
   are invalid. A valid fragment is recorded but not resolved.
4. Plain prose, lowercase prose, README, `_template.md`, and unchecked items
   without the uppercase sentinel produce no record, problem, or count.
5. Multiple records preserve file and line attribution and yield exact valid,
   invalid, and candidate counts.
6. Doctor assigns I050 the source-verified ID D15. A valid record adds no D15;
   a reason-less record, missing artifact, and malformed record on a closed
   ticket each add exactly one D15 warn. D15 makes the CLI exit 1 under the
   existing contract and appears unchanged in JSON fields.
7. With no candidates anywhere in the ledger, doctor produces no D15 output
   and no I050-driven exit change.
8. `audit stages` scans only concrete cursor-resolved ticket IDs. One valid
   record prints `acceptance: approved-untested=1 invalid=0`; one invalid
   record prints one stderr warning plus
   `acceptance: approved-untested=0 invalid=1`; either remains nonblocking
   when all existing stage and handoff checks pass.
9. A marker on an unscoped ticket is excluded. An unresolvable cursor ticket
   expression does not trigger an all-ticket fallback. An existing audit
   blocker remains blocking with any acceptance summary.
10. With no scoped candidates, `audit stages` stdout, stderr, and exit status
    match the pre-I050 behavior byte for byte.
11. Fresh non-knowledge scaffolds emit template generation 12, the WORKFLOW
    grammar, the issue-template acceptance section, and the README pointer.
    Fresh knowledge scaffolds emit the WORKFLOW wording but no issue ledger.
12. A captured pristine generation-11 repository dry-runs with zero
    unrecognized lines, writes generation 12, includes the convention, and is
    idempotent on a second pass. A genuine local edit to each touched
    machine-owned file still yields `SkippedUnrecognized` without force.
13. Generation-12 migration adds no `supersededLines` entry while all emitted
    prose changes remain additive. The future-generation guard refuses 13.
14. Existing package and CLI tests stay green, including doctor severity,
    cursor resolution, stage blocking, handoff blocking, historical
    generation fixtures, and zero-marker output controls.
15. The finished implementation passes a fresh primary-tier spec review
    against this PRD, independent verification, `go test ./... -count=1`,
    `go vet ./...`, `make verify`, `git diff --check`, applicable spine audits,
    and the final exact-SHA maipipe lane before shipment.

## Delivery boundaries

Implementation closes only I050. It may update I050's ledger status,
`commits`, and Resolution after verification, and it may add the required
consumer-facing CHANGELOG entry. It must not close or absorb I072, I073,
I074, I077, I108, or another open-ledger ticket. It must stage explicit paths
and preserve concurrent working-tree changes.
