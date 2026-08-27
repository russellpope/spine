---
id: I115
title: "D13/frontmatter hardening: parser-divergence guard, absolute-path warn, quoted values, fence-less test"
severity: low
status: open
affects: [I106]
blocked-by: []
execution-mode:
tier:
effort:
risk-triggers: []
review-tier:
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
