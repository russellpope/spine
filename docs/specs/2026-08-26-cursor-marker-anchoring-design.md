---
title: "Cursor fence anchoring"
ticket: I109
created: 2026-08-26
status: draft
---

# Cursor fence anchoring — design

## Problem Statement

A handoff that *explains* the stage cursor convention breaks the tool that
reads it.

The stage cursor is delimited by an opening and closing HTML comment. The
parser finds its block by scanning for the opening delimiter as a bare
substring, matching anywhere — mid-sentence, inside backticks, inside a code
block. So a handoff that quotes the delimiter in prose above the real block
hijacks the parse: the scan opens at the prose occurrence, runs forward to the
genuine closing delimiter, and swallows everything between, including the real
opening delimiter, which then surfaces as a malformed body line.

The operator sees a wall of quoted prose fragments and a complaint about an
unknown key. Nothing in the output says *there are two opening delimiters*, and
nothing points at the line eleven rows up where the author mentioned one.
`spine audit stages` exits 1 and `spine doctor` D9 warns, both against a
committed snapshot that is byte-perfect.

Three things make this worse than an ordinary parser bug:

1. **No policy can avoid it.** Any document explaining the cursor convention is
   a candidate, and explaining the convention is what handoffs are for. The
   working rule has been "never write the delimiter in prose" — carried forward
   in every handoff's gotchas for months. That is not a rule a documentation
   tool can ask its authors to keep.
2. **The diagnosis names the wrong thing.** Findings quote the content the
   parser choked on and never the structural cause.
3. **Backticks do not help.** The parser has no notion of inline code or fenced
   blocks, so the natural way to write about the delimiter is exactly the way
   that breaks it.

Observed live twice: 2026-08-09 (I064, the first reproduction) and 2026-08-24
(during I107's handoff). Both were resolved by rewording the document.

## Solution

Delimiters become **fences**: they are recognized only when they occupy an
entire line, starting at column 0. A delimiter mentioned mid-sentence, or
indented as a worked example, is prose and is skipped.

Because a fence is now unambiguous, the parser can count them. A document
carrying more than one open fence is refused outright with a finding naming
both line numbers, rather than silently picking the first. Every finding gains
a whole-document line number, so a complaint about a malformed line tells the
operator where to look.

Authors get their escape hatch back: to show a fence inside a document, indent
the example. That is already how `WORKFLOW.md` ships its grammar reference, so
the convention exists and is proven — it simply becomes load-bearing.

The standing "never write the delimiter" rule is retired along with the defect.

## User Stories

1. As a handoff author, I want to quote the cursor fence in a sentence, so that
   I can explain the convention to the next session without breaking the audit.
2. As a handoff author, I want to show a complete worked cursor block in a
   document by indenting it, so that I can document the grammar without the
   parser mistaking my example for the real thing.
3. As a handoff author, I want the delimiter inside backticks to be inert, so
   that ordinary Markdown habits are safe.
4. As an operator running `spine audit stages`, I want a red result to mean the
   cursor is genuinely wrong, so that I keep trusting the gate.
5. As an operator running `spine doctor`, I want D9 to stay silent when the
   committed snapshot is pristine, so that the advisory retains signal.
6. As an operator facing a malformed cursor, I want the finding to name the
   structural cause — two open fences — so that I do not have to reverse-engineer
   it from quoted fragments.
7. As an operator facing a malformed cursor, I want every finding to carry a
   whole-document line number, so that I can jump straight to the line in an
   editor.
8. As an operator, I want the duplicate-fence finding to tell me the remedy
   (indent the example), so that the error teaches the convention instead of
   only reporting a violation.
9. As an operator, I want a document with zero fences *of either kind* to remain
   the quiet "no cursor block" case, so that dormant repos and fresh clones do
   not warn. (Amended 2026-08-26 after review: the original wording said "zero
   open fences", which contradicted stories 11 and 12 — a close-only document
   has zero open fences yet must report. Quiet requires no fences at all.)
10. As an operator, I want a missing close fence to stay a distinct finding from
    a duplicate open fence, so that two different mistakes read differently.
11. As an operator, I want a close fence appearing before any open fence to be
    reported, so that a truncated or reordered hand edit is caught rather than
    ignored.
12. As an operator, I want two close fences reported the same way as two open
    ones, so that the rule has no asymmetric gap to fall through.
13. As an agent resuming a session, I want the cursor to be the single source of
    truth for where an effort stands, so that a resume never starts from a
    hijacked parse.
14. As an agent writing a handoff, I want `spine handoff new` to keep embedding
    the committed snapshot unchanged, so that the fix costs me no workflow
    change.
15. As a spine maintainer, I want the same fence scanner behind every call site
    that looks for a cursor block, so that the three cannot drift apart and
    reopen the defect at a seam nobody fixed.
16. As a spine maintainer, I want the presence test and the parse to agree
    exactly on what counts as a block, so that a caller cannot observe "a block
    exists" and "there is no block" in the same document.
17. As a spine maintainer, I want the write path to refuse a rewrite whenever it
    finds a fence-rule violation, so that a destructive replacement never runs
    against an ambiguous file. (Amended 2026-08-26 after review: was "duplicate
    fences" — see D8.)
18. As a spine maintainer, I want that write refusal to name every offending
    line number, so that an unreachable-in-practice corruption is diagnosable
    when it does fire.
19. As a spine maintainer, I want the canonical serialization and the
    non-canonical byte comparison untouched, so that this change cannot alter
    what the tool writes.
20. As a spine maintainer, I want the live repo's own ledger to keep parsing
    clean, so that the change is verified against the real artifact and not only
    synthetic fixtures.
21. As a spine maintainer, I want existing committed handoffs that parse today to
    keep parsing, so that the stricter rule does not retroactively condemn
    honest documents.
22. As a documentation reader, I want `CONTEXT.md` to name the fence concept
    distinctly from the stage-state marker, so that a single sentence about both
    is possible.
23. As a documentation reader, I want `README.md` to state the line-anchored rule
    and the indent escape hatch, so that the convention is discoverable without
    reading source.
24. As the next session, I want the "never write the delimiter" gotcha gone from
    the handoff, so that a retired hazard stops consuming attention.

## Implementation Decisions

Ratified in the 2026-08-26 grill; each was put to the owner and confirmed.

**D1 — Strict column 0.** A fence is recognized only as an entire line with no
leading whitespace. Trailing whitespace is tolerated, because it is invisible
and cannot be an authoring gesture; leading whitespace is a deliberate one and
means "this is an example". This immunizes indented code blocks by
construction, and changes nothing the tool writes, since the canonical
serialization already emits at column 0.

**D2 — One shared scanner, three call sites.** The `internal/cursor` package
exposes three entry points that look for a block: the parse used by `Load`, the
handoff-snapshot parse (`ParseBlockResult`), and the presence test (`HasBlock`).
All three route through a single line-anchored scan.

This is not tidiness. `HasBlock` is a bare substring test, and the derivation
engine in `internal/stages` calls it as a guard *before* parsing the newest
handoff. Anchoring the parse without anchoring the presence test produces a
contradiction: a prose mention makes the presence test true, the parse then
finds nothing and returns an empty result with **zero findings**, and the
derivation falls past its findings branch into the stale-effort branch to report
an empty block effort. One confusing symptom traded for another.

**D3 — More than one open fence is a hard finding.** No parse is attempted. The
finding names every offending line number. Silently choosing one of two
candidate blocks is the exact failure class this work exists to eliminate, and
"exactly one committed snapshot per handoff" is already the contract in
`CONTEXT.md`.

**D4 — Whole-document line numbers on every finding that quotes an offending
line.** Body parsing currently quotes offending lines with no position. Numbers
are 1-based against the whole document, not relative to the block, because the
operator has the file open in an editor. This requires threading the block's
starting line into body parsing.

A finding about the block as a whole — a missing required key — carries no line
number: there is no offending line to point at, and the key names the fix.
(Amended 2026-08-26 after review: the original wording said "on every finding",
which the missing-key class cannot satisfy. D4's own rationale — jump straight
to the line — does not apply where there is no line.)

**D5 — Fences, not markers.** `CONTEXT.md` already uses *marker* for the
per-stage state characters. The delimiters are **open fence** and **close
fence**, bounding a **fenced region**. Findings, the glossary, and this document
use that vocabulary throughout.

**D6 — No code-block awareness.** D1 already covers the indented case.
Remaining exposure is a triple-backtick example with fences at column 0, which
occurs nowhere in the repo today. Tracking fence state properly — backtick
versus tilde, info strings, varying fence lengths, nesting — is real machinery
guarding a hypothetical, and under D3 that hypothetical now fails loudly with
line numbers pointing straight at the example.

Instead the duplicate finding carries the remedy: *to show a fence in a
document, indent the example instead of fencing it.* The escape hatch becomes a
clause in an error message rather than a subsystem.

**D7 — Duplicate counting scans the whole document.** Counting only up to the
first close fence still catches the hijack, but silently ignores a complete
block pasted *below* the real one. Whole-document counting states the invariant
in one clause — exactly one open fence in the file — which is a rule an author
can hold in their head.

**D8 — The write path refuses on any fence-rule violation.** The save path
replaces first-open-through-close, so against an ambiguous file it can destroy
everything in between. It only ever writes the working home, which is
machine-owned, so violations there should be unreachable. That is precisely when
a loud failure is wanted: if it fires, the ledger has been hand-edited or
corrupted, and a destructive rewrite is the worst available response. The error
names every offending line number. This is the sole-writer rule enforced at the
write rather than only at the audit.

(Amended 2026-08-26 after review: the original wording scoped the refusal to
*duplicate* fences. It refuses on any violation D9 defines, including a stray
close and a close-before-open. Those previously got a fresh block prepended,
which would leave the ledger holding one open and two closes — a document that
fails the very next parse. Refusing is the outcome consistent with D9.)

**D9 — Symmetric fence rules.** Exactly one open fence, exactly one close
fence, close after open. Each violation is its own finding. Today the close is
searched only after the open, so a stray close above the block is invisible and
a second close below is ignored. A column-0 close fence in prose is as much a
hand-edit signal as an open one.

**D10 — Documentation retires the rule.** Fence vocabulary into `CONTEXT.md`
§Stage cursor; the line-anchored rule and the indent escape hatch into
`README.md`; the standing gotcha dropped from the next handoff rather than
carried forward. Docs-only, at the docs stage.

**Hard constraint, carried from the ticket.** The canonical serialization and
the non-canonical byte comparison are untouched. This work is about *finding*
the fenced region, never about formatting it.

## Testing Decisions

A good test here asserts what an operator or author observes — whether a
document parses, what a finding says, whether the audit blocks — never how the
scan is implemented. Fence recognition is deliberately tested through the
package's public entry points rather than against an internal scanner, so the
tests survive the scanner being rewritten.

Three seams, all existing, no new ones. Confirmed with the owner before
drafting.

**S1 — the package load path** (`internal/cursor`, external test package).
The highest seam in the package; drives parsing end-to-end from real files.
Carries D1, D3, D4, D7, D9. Prior art is already in place in both styles the
package uses: a temp directory with written fixture files for one-off cases,
and golden `testdata/<scenario>/repo` trees for durable ones. The presence test
and the handoff-snapshot parse are exported and reachable from this same
package, so D2's third call site needs no seam of its own — and it gets an
explicit test that presence and parse agree on the same document, which is the
contradiction D2 exists to prevent.

**S2 — the save path** (same package, same harness). Carries D8. Worth stating
plainly: **this function has no test coverage anywhere in the repo today** —
verified by grepping every test file. So this is not a new seam but an existing
exported one that was never exercised. Given D8 makes it refuse a destructive
rewrite, closing that gap is warranted independently of this ticket.

**S3 — the derivation engine** (`internal/stages`, golden `testdata/<scenario>`
trees). The operator-visible seam, and the only one that observes the actual
reported failure: a handoff carrying a prose fence turning `spine audit stages`
red and D9 with it. One fixture proving that document now derives clean.

**Negative control, both arms.** A repo-history audit of all 40 committed
handoffs turned up a genuine specimen: `2026-07-24-flavor-model-table-i033-i039`
carries its real block at lines 8–13 and a second complete literal **mid-line**
at line 17. It parses correctly today only because first-match-wins happens to
open at line 8.

- That shape must parse **clean** — proving strict column 0 is not over-broad.
- The same fixture with the line-17 fence moved to **column 0** must **block**
  with the duplicate finding — proving the rule is load-bearing rather than
  vacuous.

**Fixtures are synthetic.** Both live reproductions (2026-08-09 and 2026-08-24)
were repaired by rewording before commit, so no currently-breaking document
exists in the repo to regress against. Fixtures are reconstructed from those two
descriptions. This is stated so a reviewer does not go looking for a live
failing file and conclude the defect was imagined.

**Free coverage.** The package's existing dogfood test parses this repo's own
live ledger and must keep passing — the standing check that the change works
against the real artifact, not only synthetic input. It skips on a fresh clone,
since the ledger is gitignored.

**Retroactive safety.** Only the newest handoff is read by the derivation
engine, and it currently carries exactly one open and one close fence. So no
stricter rule can retro-condemn the file the gate actually reads. The six other
handoffs that mention the cursor in prose all use the bare word in backticks,
never the full literal, and are unaffected in either direction.

## Out of Scope

- Code-fence and indented-code-block awareness (D6). Reconsider only on evidence
  of a real document that needs it.
- Any change to the canonical serialization or the non-canonical byte
  comparison.
- Any change to what `spine cursor` writes, or to the stage grammar itself.
- The two long-standing `spine doctor` D4 notes that make it exit 1. Pre-existing,
  filed as I065, explicitly not a side quest here.
- Migrating or rewriting existing committed handoffs. They are historical and
  never retro-mutated.
- The stage-state marker vocabulary. D5 renames the delimiters only; `[x]`,
  `[<]` and `[ ]` stay markers.

## Further Notes

The ticket ranked its fixes F1 through F4. This design takes F1 (as D1/D2), F2
(as D3/D7/D9), and F3 (as D4), and declines F4 (D6) after establishing that
strict column 0 already covers every instance in the repo.

The ticket's "Related" section flags the save path as having the same weakness,
and notes it is worse in kind because its replacement is destructive. D8 goes
further than the ticket proposed — refusing rather than merely anchoring —
because an anchored-but-still-first-match write against a corrupted ledger would
still destroy text.

I064 was closed on 2026-08-26 as a duplicate superseded by I109 and carries the
first field reproduction. The defect was never fixed; this is its first
treatment.
