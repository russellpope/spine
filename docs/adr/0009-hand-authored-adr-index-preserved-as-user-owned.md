---
id: 0009
title: hand-authored ADR index preserved as user-owned
status: Accepted
date: 2026-07-02
---

# 0009: hand-authored ADR index preserved as user-owned

## Context

praxis carries a hand-authored 106-entry ADR index at `docs/adr/README.md`: a MADR-format
Decisions-Log mirror with per-decision form/status columns, actively maintained by hand as new
ADRs land. `spine adopt` treats every `docs/adr/README.md` as machine-owned (ADR 0002's
ownership split, `update.simpleFiles`), so the praxis index reads as 106 lines of unrecognized
content — permanent `D4 warn` on `doctor`, `update` exiting 1 forever, and `--force` would
silently overwrite curated content with the four-paragraph spine template. This is exactly the
ownership-model gap ADR 0005 covers for individual pre-spine ADR files (no front matter -> info,
not warn) but ADR 0005 doesn't reach the README itself, which spine still claims outright.

## Decision

`docs/adr/README.md` — and only this file; `docs/issues/README.md`, `docs/issues/_template.md`,
and `docs/harness-interface.md` stay strictly machine-owned — is preserved as-is when it exists
with unrecognized content: `update` treats it as up-to-date (not Skipped, not Pending), `doctor`
reports it as D4 info (not warn), and `adopt` labels it `preserve` with an info line. `--force`
remains the explicit, opt-in path to regenerate it from the template, exactly like any other
machine-owned file. A `docs/adr/README.md` that is absent or already matches the template is
unaffected — create/regenerate behavior is unchanged. This applies ADR 0005's pre-spine spirit
(hand-authored content is legitimate, not drift) narrowly to this one file, and treats the
unrecognized content itself as a deliberate choice under ADR 0002's choice-vs-default rule,
rather than as noise to flag or an edit to silently drop.

## Consequences

praxis (and any repo with a similarly curated ADR index) is adoptable with the index intact —
Task 13's real-fixture post-condition (doctor info-only, update no-op) now holds with the actual
praxis README in the fixture, not a stand-in. `--force` is the one documented, explicit path to
convert a hand-authored index over to the spine template; nothing else in `update`/`doctor`/
`adopt` regenerates it without that flag. No other machine-owned file's ownership status changes:
`docs/issues/README.md`, `docs/issues/_template.md`, and `docs/harness-interface.md` remain
strict, so this is a scoped exception, not a general softening of the ownership split.
