---
id: I115
title: "D13/frontmatter hardening: parser-divergence guard, absolute-path warn, quoted values, fence-less test"
severity: low
status: open
affects: [I106]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort: cursor-hygiene-batch
risk-triggers: []
review-tier: routine
---

## Problem

Filed 2026-08-27 from the doctor-hygiene batch's final whole-branch review
(spec docs/specs/2026-08-27-doctor-hygiene-batch-design.md): four small,
non-blocking gaps in the new D13 per-ticket checks, all touching the same
~20 lines of `internal/doctor/tickets.go`:

1. The doctor frontmatter parser is a fourth near-copy (audit, stages, and
   adr carry their own) and deliberately DIVERGES from audit's: it must not
   strip `#`-comments because `#` is significant inside a `batch:` id. The
   divergence is correct but uncommented — a future DRY consolidation of the
   four parsers would silently break D13.
2. A relative `workspace:` value is stat'd against the process CWD, not
   `--dir`, so `spine doctor --dir <other-repo>` can false-warn. The
   convention mandates absolute paths, and nothing enforces that.
3. A YAML-quoted `batch:` value (`batch: "2026-08-27-dhyg#1"`) reads as
   malformed because the quotes survive trimming; the live ledger writes
   bare values, so this is latent until a board emits quoted YAML.
4. A ticket file with no frontmatter fence is intentionally silent, but no
   test asserts the silence.

## Fix

In `internal/doctor/tickets.go` (+ its tests): a one-sentence guard comment
on the parser naming the no-comment-stripping divergence and why; a
`filepath.IsAbs` warn for `workspace:` (which also removes the CWD
dependence of the existence stat); strip surrounding quotes in the
frontmatter values (or document bare-values-only in the convention); and a
fence-less-ticket test asserting zero D13 findings.

## Rulings (grill 2026-08-28, spec docs/specs/2026-08-28-cursor-hygiene-batch-design.md)

- Item 2: **warn on non-absolute and do not stat it** — the convention
  mandates absolute paths, so resolving a relative value against `--dir`
  would legitimize a forbidden form. Absolute-and-missing keeps the
  existing existence warn.
- Item 3: **strip surrounding quotes** (tolerant reader), not
  document-bare-values-only — quoting is a property of whichever board
  emits the YAML, not of the value; a false warn on semantically identical
  input erodes trust in D13.
