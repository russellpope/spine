---
title: "I125 — ticket closure as implement evidence"
tickets: I125,I105
created: 2026-09-02
status: Accepted design (autonomous grill; assumptions flagged in the grill record)
---

# I125 — ticket closure as implement evidence

**Ticket:** `I125` (substance). `I105` rides the same effort cursor but is a
recorded owner decision, not build work; see Further notes.

## Problem statement

An operator ticks `implement` after closing every ticket in the effort the
way the ledger lifecycle prescribes: `status: fixed` and the fix SHAs in
`commits:`. `spine audit stages` then blocks with `implement (ticked-missing)`
because the only evidence the derivation reads is a `<id>: … done|complete|completed`
line in `.superpowers/sdd/progress.md`, and this effort never wrote one. In
the same output the `issues` row proves those ids resolve to real closed
ticket files. The operator has to `--force` the tick over verified reality.

Observed live in maikanban on 2026-08-28 (I047/I048). The block reproduces at
HEAD 68aa28f.

## Solution

Implement evidence gains a second source. A **closure record** — a ticket file
whose `status` is `fixed` and whose `commits:` names at least one SHA — proves
implement for its id, OR'd with the existing progress-ledger line scan. The
derivation stays bidirectional and conservative: a closure record is presence
evidence in both directions, absence of both sources still only blocks a
stage that is ticked done, and the zero-evidence detail on the implement row
names the real rule (no ledger line and no closure record) instead of leaving
a bare count.

## User stories

1. As an operator closing tickets by the ledger lifecycle, I want a ticked
   `implement` to derive `match` when every anchored ticket has a closure
   record, so that I never `--force` a tick over verified reality.
2. As an operator who still writes progress-ledger lines, I want that path to
   keep working unchanged, so that existing efforts derive exactly as before.
3. As an operator with a mixed effort, I want one id evidenced by a ledger
   line and another by a closure record to derive `match` together, so that
   the two sources genuinely OR.
4. As a reviewer, I want `status: open` or `in-progress` tickets, `fixed`
   tickets with an empty or absent `commits:`, and `wontfix`/`superseded`
   tickets to contribute no implement evidence, so that the new path is
   load-bearing and not a blanket pass.
5. As a reviewer, I want a `commits:` value that is not SHA-shaped (`[pending]`,
   `[n/a]`) to contribute no evidence, so that a placeholder cannot certify
   implementation.
6. As an operator reading a ticked-missing implement row with no ledger line
   for any id, I want the detail to name the rule (no progress-ledger
   implement line and no closure record for the ids), so that I know which
   artifact to produce.
7. As an operator, I want the `tickets:` typo hint to stay on the `issues`
   row only, so that a proven-good tickets value is never called a typo.
8. As an operator whose implement stage is still pending while a ticket in
   the range already carries a closure record, I want `present-unticked` to
   fire as it does today for a stray ledger line, so that a stale cursor is
   still caught.
9. As an operator with a malformed or unreadable ticket file, I want the
   derivation to treat it as no evidence and never error, so that a broken
   file degrades to under-detection, not a crash or a false block.
10. As a reader of `spine audit stages`, I want the implement row label to
    say `implement evidence` rather than `ledger implement evidence`, so that
    the label does not misname closure-sourced evidence.
11. As a doctor user, I want D9 to use the same engine and therefore stop
    advising on this false block, so that doctor and audit agree.
12. As the maintainer, I want the package doc, the glossary, and the
    changelog to record the second source and its residuals, so that the
    next reader does not reintroduce the ledger-only rule.

## Implementation decisions

- **Closure record definition.** `status: fixed` and a `commits:` value whose
  bracketed, comma-separated list contains at least one token matching a
  hex SHA of 7 to 40 characters. Any other status is non-evidence, including
  `in-progress`, `wontfix`, and `superseded`. Case of `status` is exact
  (the lifecycle vocabulary is lowercase). Parsing is the same leading
  `---` fence walk the derivation already uses for `id:`; only a same-line
  value counts, so a YAML block list under `commits:` reads as empty
  (documented residual; the estate writes inline lists).
- **OR semantics.** Per id, evidence is ledger-line OR closure-record. The
  result feeds the existing bidirectional judge unchanged, so a closure
  record on a pending implement stage yields `present-unticked` exactly as a
  stray ledger line does today. No new direction, no new verdict.
- **Ticket file resolution.** The closure scan reads the same docs/issues
  set and the same `id:` frontmatter match the `issues` row uses, so the two
  rows can no longer contradict each other on a closed ticket.
- **Zero-evidence detail on the implement row.** Three cases, first match
  wins: (a) some anchored ledger line exists without a done-word — the I117
  wording message, unchanged; (b) no anchored line — a new message naming
  both sources: no progress-ledger implement line and no closure record for
  the id(s); (c) partial misses keep the bare named-ids detail as today. The
  `tickets:` typo hint remains issues-row only (I032 already scoped it; this
  change adds a test that the new message never carries it).
- **Judge refactor.** The judge receives the zero-evidence hint text from
  its caller instead of a boolean plus the raw tickets value, so each row
  owns its own wording and the judge stays row-agnostic.
- **Label.** The implement row's label becomes `implement evidence`. This is
  a visible output change; the changelog records it.
- **Conservative rule preserved.** Unreadable directory, unreadable file, or
  malformed frontmatter yields no evidence and never an error.
- **No generation bump.** `docs/issues/README.md` is template-managed;
  its `commits` bullet is not edited. The convention note lives in the
  changelog, the glossary, and the package doc.
- **No ADR.** The rule is a derivation heuristic, cheaply reversible, and
  its rationale is captured in the package doc where the ledger-only rule
  was pinned. Two of the three ADR criteria fail.

## Testing decisions

- Black-box tests in the stages package, written the way the I117 and I032
  tests are: a temp repo with WORKFLOW.md, ticket files, and a progress
  ledger, then `Derive`, then assert verdict and detail substrings. Each
  positive case pairs with the negative controls in stories 4 and 5.
- One white-box table test for the closure-record predicate, next to the
  existing word-boundary table test.
- The compiled-CLI byte-exact output test updates its expected label line.
- The full lane (`go test ./...`, then `maipipe run full --wait`) must be
  green at the final SHA. Live check: `spine audit stages` on this repo
  must still derive 21/21 for the prior effort, since every batch ticket
  already carries a ledger line.

## Out of scope

- Editing `docs/issues/README.md` or any template (generation bump).
- Git or commit-graph inspection as evidence; the SHAs are not resolved.
- Any rule for stages other than `implement`.
- Multi-line YAML `commits:` values.
- I105's product decision (owner-held; see below).

## Further notes

**I105.** The 2026-08-29 research record recommends adopting OpenCode for the
constrained worker lane and keeping Pi as a model-resolution target, and
names the owner's call: fund a scoped Pi-extension ticket, or close I105 on
that adoption. No code changes either way. This effort carries I105 so the
ruling lands in the ledger; the ADR is drafted only when the owner rules.

## Grill record

Answers marked **ticket** or **code** are settled by I125's text or by
observed code. Answers marked **assumption** were the recommended answer in
an autonomous session and are the ones a reviewer should challenge.

| # | Question | Answer | Source |
|---|---|---|---|
| Q1 | What is a closure record? | `status: fixed` plus non-empty `commits:` | ticket |
| Q2 | Must `commits:` tokens look like SHAs? | Yes, hex 7–40; placeholders are non-evidence. Strictness can only keep a today-blocking case blocking; leniency could create a new present-unticked block | assumption |
| Q3 | Do `in-progress`, `wontfix`, `superseded` count? | No | ticket |
| Q4 | Does closure evidence feed the pending direction? | Yes, symmetric with ledger lines; the ledger already has this shape for pre-shipped tickets in a range | assumption |
| Q5 | Which file evidences an id? | Same `id:` match as the issues row | code |
| Q6 | Is the typo hint still on the implement row? | No; I032 removed it at HEAD. Remaining work is naming the rule | code |
| Q7 | When does the new rule text fire? | Zero evidence and no anchored line, whether or not ticket files exist | assumption (ticket conditions on files existing; unconditional is strictly more informative and satisfies the criterion) |
| Q8 | Anchored line without done-word plus a closure record? | Closure wins; match | ticket (OR) |
| Q9 | Rename the row label? | Yes, `implement evidence` | assumption |
| Q10 | ADR? | No; two criteria fail | assumption |
| Q11 | Touch the issues README convention text? | No; template-managed | code |
| Q12 | I105 disposition? | Owner ruling required; no autonomous ADR | research record |

## Requirements attack

| Attack | Resolution |
|---|---|
| Criterion 1 says "non-empty commits" while Q2 requires SHA shape; a `[pending]` list is non-empty. | SHA shape is the stricter reading and is recorded as an assumption; criterion 2's "empty commits" control still holds. Review caveat (2026-09-02): a fixed ticket whose `commits:` names no SHA (`[see PR #12]`) still blocks a ticked implement — a known residual recorded in the package doc, not a neutral tightening. |
| Criterion 4 conditions the rule text on ticket files existing; Q7 fires it unconditionally. | Unconditional wording still satisfies the criterion's observable (rule named, no typo hint) and adds no false block. |
| The ticket calls closure "a stronger record" than the ledger, yet both are OR'd equally. | Equal weight is correct for a presence check; strength ordering would only matter for a contradiction verdict the package does not have. |
| The ticket's Problem says the typo hint "still gets" emitted on the implement row; at 68aa28f it already could not (I032 passed an empty tickets value for that row). | Fix part 2 is read as "name the rule", which is what criterion 4 tests and what was built. The ticket's Resolution records the stale sentence. |
| Q4 symmetry: closure records exist for every closed ticket forever, so `present-unticked` exposure on a range or prefix cursor that spans old closed tickets widens beyond what progress-ledger lines already caused. | Story 8 stands: a pending implement over a closed ticket is a stale cursor. The widening is a recorded residual; a cursor that deliberately spans pre-closed tickets should tick implement, as the batch efforts do. |

## Review corrections (2026-09-02)

- **Quote stripping dropped.** The frontmatter walk does not strip quotes,
  so `id:` resolution on the issues row stays byte-identical to 68aa28f.
  A quoted `status: "fixed"` is therefore not a closure record; the ledger
  convention writes it unquoted.
- **Bare `commits:` value accepted.** "Bracketed" above over-specified: the
  predicate reads the same-line value with brackets tolerated, so a bare
  `commits: a9ddea5` counts. A real SHA is a real SHA either way.
- **Both rows in one derivation.** The positive test also asserts the
  issues row matches, since ending the row contradiction is the ticket's
  first aggravation.
