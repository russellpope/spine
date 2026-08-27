---
id: I109
title: "cursor block scanning matches a marker anywhere in the file, so a marker quoted in prose hijacks the block"
severity: med
status: open
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
