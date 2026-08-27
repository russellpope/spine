---
id: I109
title: "cursor block scanning matches a marker anywhere in the file, so a marker quoted in prose hijacks the block"
severity: med
status: fixed
affects: [I107]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: primary
---

## Problem

`internal/cursor/cursor.go` finds the cursor block with a bare substring scan:

```go
start := strings.Index(content, openTag)      // parse(), :267
rest  := content[start+len(openTag):]
endRel := strings.Index(rest, closeTag)
```

`strings.Index` matches the marker **anywhere**, including mid-sentence and
inside backticks. A document that *mentions* the marker in prose above the real
block therefore hijacks it: the parser opens at the prose occurrence, scans
forward to the real closing marker, and swallows everything between — including
the genuine opening marker, which surfaces as a body line.

Observed live 2026-08-24 while writing `docs/handoffs/2026-08-24-i107-gate-panic-misconfiguration-codex.md`.
A Gotchas line reading ``never hand-edit the `<!-- spine:cursor -->` block``
produced:

```
derivation: blocking
  handoff: ... carries a malformed spine:cursor block: unrecognized line in
  cursor block: "` block.** `spine` is the sole"; ... ; unknown key
  "<!-- spine" in cursor block
```

`spine audit stages` exited 1. The fix was to stop naming the marker in prose.

Three things are wrong here, in descending order of importance:

1. **The hazard is unavoidable by policy.** Any document that *explains* the
   cursor convention is a candidate. `WORKFLOW.md` already carries the markers in
   its grammar-reference code block, and `README.md` quotes
   `<!-- spine:cursor -->` inline while describing the cursor. Telling authors
   "don't write the marker" is not a workable rule for a tool whose whole point
   is documenting a convention.
2. **The diagnosis points at the wrong thing.** The findings quote the *content*
   the parser choked on, never the structural cause. Nothing in the output says
   "there are two opening markers". An operator reading that wall of quoted
   prose has no path to "you mentioned the marker eleven lines up".
3. **Backticks do not protect it.** The parser has no notion of inline code or
   fenced blocks, so the natural way to write about the marker is exactly the
   way that breaks it.

## Fix

Ranked; the first is small and closes the reported failure on its own.

**F1 — anchor markers to their own line.** Require the open and close markers to
occupy a whole line (leading/trailing whitespace tolerated), rather than matching
anywhere. The 2026-08-24 occurrence was mid-sentence and would have been skipped
outright. Cheap, and it makes the sole-writer rule enforceable by construction:
`Block()` already emits the markers on their own lines, so nothing spine writes
is affected.

**F2 — count markers and report the structural cause.** Scan for all
line-anchored opens and closes. Zero opens is today's "no cursor block". More
than one is an error naming the line numbers: `2 opening spine:cursor markers
(lines 118, 143) — a handoff must contain exactly one`. This is what turns a
confusing symptom into a one-line diagnosis, and it also catches a whole block
pasted into a doc, which F1 alone would not.

**F3 — put line numbers on every finding.** `parseBody` quotes offending lines
with no position. `line 121: unknown key "<!-- spine"` is strictly more useful
than the quoted text, and costs an index in the loop.

**F4 — consider ignoring fenced and indented code.** Would let `WORKFLOW.md`'s
grammar reference and a README's worked example carry a full block safely. More
work than F1–F3, and F2 already fails *loudly* rather than silently in that case,
so this is optional. Decide it on evidence, not up front.

Constraint: none of this may change what `spine cursor` writes. `Block()`'s
canonical serialization and the `NonCanonical` byte-comparison stay exactly as
they are — this ticket is about *finding* the block, not formatting it.

## Design

PRD: `docs/specs/2026-08-26-cursor-marker-anchoring-design.md` (grilled and
ratified 2026-08-26). Effort `i109-cursor-marker-anchoring`.

Where the design departs from the Fix section above:

- **F1 anchors at strict column 0**, not "whitespace tolerated". Whitespace
  tolerance would still match the indented grammar reference `WORKFLOW.md`
  ships, so an author pasting that example into a handoff would be flagged for
  doing nothing wrong. Strict column 0 makes indented examples safe by
  construction and delivers most of F4's value for free.
- **F1 covers a third call site the Fix section missed: `HasBlock`.** It is a
  bare `strings.Contains`, and `internal/stages` calls it as a guard *before*
  parsing the newest handoff. Anchoring `parse` without anchoring `HasBlock`
  yields a presence test that is true while the parse finds nothing — an empty
  result with zero findings, which falls through to the stale-effort branch and
  reports an empty block effort. All three route through one shared scanner.
- **F4 is declined.** No document in the repo has a column-0 fence inside a
  triple-backtick block; the remedy ships as a clause in F2's finding ("indent
  the example instead of fencing it") rather than as fence-state machinery.
- **F2 counts whole-document and is symmetric** — exactly one open, exactly one
  close, close after open, each violation its own finding.
- **The `Save` seam below is escalated from anchoring to refusal.** An
  anchored-but-still-first-match write against a corrupted ledger would still
  destroy the intervening text.

Repo audit backing these calls (all 40 committed handoffs):
`2026-07-24-flavor-model-table-i033-i039.md` carries a real second full literal
mid-line at :17 above its genuine block at :8–13 — the negative control, which
must keep parsing clean. Six other handoffs mention the cursor in prose using
the bare backticked word, never the full literal, and are unaffected. No
currently-breaking document exists, so regression fixtures are synthetic,
reconstructed from I064's and I107's descriptions.

## Related

- `Save` (`cursor.go:229`) has the same first-match-anywhere weakness on the
  write side, and there it is worse in kind: it replaces prose-occurrence-through-
  real-close with the new block, destroying the intervening text. In practice the
  blast radius is small because `Save` only ever writes
  `.superpowers/sdd/progress.md`, which is machine-owned and unlikely to contain
  prose about the marker. F1 closes both seams at once and should be applied to
  both.
- Checked: nothing parses `WORKFLOW.md` for a cursor block. The one
  `spine:cursor` reference in `internal/update/update.go:559` is a comment, not a
  second scanner. So WORKFLOW.md's grammar reference and README's inline mention
  are safe **today** — they are evidence that documenting the marker is normal,
  not live instances of the bug. The exposure is handoffs, which are parsed.
- I107 — surfaced during that ticket's handoff, unrelated to its subject matter.
- **[I064] is the same defect, filed earlier and closed 2026-08-26 as a
  duplicate superseded by this ticket** (ledger reconciliation). It carries the
  first live reproduction — the i062-tiebreak codex handoff on 2026-08-09, where
  a backticked marker inside a Gotchas bullet took both `spine doctor` D9 and
  `spine audit stages` red while the real spine-owned block was pristine. Use it
  as the field case for the regression fixture; the defect itself is unfixed and
  owned here.

## Resolution

Fixed 2026-08-26. Cursor delimiters are now **fences**, recognized only as a
whole line starting at column 0. `internal/cursor` gains a `scanFences`/`locate`
pair that `parse`, `Save` and `HasBlock` all route through, so the three cannot
disagree about what a block is. `parseBody` takes a whole-document base line and
numbers the findings that quote an offending line.

What this retires: the standing "never write the literal marker in prose" rule
that had appeared in every handoff's gotchas for months. A document may now
quote a fence mid-sentence, and may show a complete worked block by indenting
it — the form `WORKFLOW.md` already uses for its grammar reference.

Behavior, each covered by a test:

- Mid-line and indented occurrences are prose and are skipped.
- More than one open (or close) fence is refused outright, naming every line
  number and carrying the remedy: indent the example instead of fencing it.
- Fence rules are symmetric — exactly one open, exactly one close, close after
  open; a stray close with no open reports rather than falling silent. Quiet
  means no fences of either kind.
- `Save` refuses to rewrite on any fence-rule violation rather than replacing
  open-through-close against an ambiguous file, leaving the ledger untouched.
- `Block()`'s canonical serialization and the `NonCanonical` byte comparison are
  unchanged, per the ticket's constraint.

Evidence: 17 assertions in `internal/cursor/fence_test.go` plus a golden
`internal/stages/testdata/handoff-prose-fence` tree; 14 observed RED before
implementation, reproducing `unknown key "<!-- spine" in cursor block` — the
string this ticket quotes from the live 2026-08-24 occurrence. Negative control
run both arms: reverting only `cursor.go` takes the stages fixture red and
restores byte-identically. Five scenarios against the live installed binary
covered the documented handoff, duplicate fences, a stray close, the write path
preserving prose, and the refused write leaving the ledger SHA-256 unchanged.

Gates: cold primary-tier spec-axis review PASS (0 missing/partial); fresh-context
primary-tier verification SHIP, having re-run the negative control itself.

Two things the gates found, recorded because neither was in the original Fix
section:

- The reviewer found `Save` refusing more broadly than decision D8 authorized.
  Owner ratified widening D8; the code stands.
- The verifier found a **regression this change introduced** — the scanner
  trimmed only space and tab, so a CRLF document's `<tag>\r` matched nothing and
  a valid block reported as missing. Fixed here with its own test and both
  control arms, rather than deferred: the substring scan this replaced handled
  CRLF, so shipping without it would have been strictly worse than before.

The spec carries three owner-ratified amendments made during review (story 9,
D4, D8), each with its reason inline. A pre-existing gap found by the same
verify probe — trailing whitespace after the closing fence escapes the
`NonCanonical` comparison — is **[I113]**, deliberately not fixed here because it
predates this change; confirmed by reading the original source rather than
assuming.
