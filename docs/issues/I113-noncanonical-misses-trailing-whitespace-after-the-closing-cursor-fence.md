---
id: I113
title: "NonCanonical misses trailing whitespace after the closing cursor fence, so one hand-edit shape goes undetected"
severity: low
status: open
affects: [I109]
blocked-by: []
execution-mode: inline
tier: mechanical
effort:
risk-triggers: []
review-tier: routine
---

## Problem

`NonCanonical` is the guard that catches hand edits: a grammatically valid
cursor block whose bytes differ from `Block()`'s canonical serialization is
evidence someone edited the block by hand, which the sole-writer rule forbids.
`spine audit stages` fails on it and `spine doctor` advises on it.

It compares `content[open.start:closeF.end]` against `Block()`. `closeF.end` is
the byte just past the closing tag text — so anything *after* the tag on that
same line is outside the compared span.

The result is an asymmetry:

- Trailing whitespace on the **opening** fence line is inside the span, so it
  correctly sets `NonCanonical`. Covered by
  `TestTrailingWhitespaceFenceIsRecognizedButNonCanonical`.
- Trailing whitespace on the **closing** fence line is outside the span, so it
  does not. The block is recognized (the scanner tolerates the padding when
  matching), reports no findings, and reports canonical form — while its bytes
  are not what spine writes.

Found by the I109 verify gate (2026-08-26, fresh-context probe), which built
the case rather than reasoning about it.

**This is pre-existing, not an I109 regression.** The prior implementation
computed `blockEnd = start + len(openTag) + endRel + len(closeTag)` — the same
span, ending at the closing tag rather than at the end of its line. I109
changed how the region is *found*, not what is compared. Verified against the
original source before filing.

## Fix

Extend the compared span to the end of the closing fence's line, excluding its
line terminator: compare `content[open.start:closeLineEnd]` where
`closeLineEnd` is the offset of the `\n` that ends the close fence line (or
`len(content)` when the document ends without one).

`scanFences` already computes that boundary while walking lines; carrying it on
the `fence` struct as a third offset is enough, and `locate` already returns the
struct.

Constraint, unchanged from I109: this must not alter what `spine cursor` writes.
`Block()` stays exactly as it is — this is about what the guard *compares*, not
what the tool *emits*.

## Acceptance

- A block whose closing fence line carries trailing spaces or tabs reports
  `NonCanonical` true, with no findings (it is valid, just not canonical).
- The existing opening-fence case still reports `NonCanonical` true.
- A byte-canonical block still reports `NonCanonical` false — including one at
  end-of-file with no trailing newline, which is the boundary the new offset
  has to get right.
- A CRLF document is judged on its own terms and not made noisier than it is
  today (see I109's `TestCRLFDocumentFencesAreRecognized`).

## Related

- **I109** — line-anchored cursor fences. Found during its verify gate; shares
  `scanFences`/`locate`, which is where the fix lands.
- The I109 verify gate also found and fixed a genuine regression in the same
  area (CRLF fence lines were no longer recognized because the scanner trimmed
  only space and tab). That one shipped with I109; this one did not, because it
  predates the change.
