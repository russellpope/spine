---
id: I072
title: Host config schema and the preference/constraint precedence story
severity: med
status: fixed
commits: [06f0628, f14e927, 4fbf72b, 2d5843d, 04f9ea4, 97a20ad, fa41be9, 518a73b, 4ba395f, 50a6a6d]
affects: [model, workflow, fleet, cli]
blocked-by: [I070, I071]
labels: [wayfinder:grilling]
parent: I066
assignee:
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: [cross-task-integration, plan-flagged-ambiguity]
review-tier: primary
---

## Question

[I068](I068-host-scoped-availability-and-tier-pins.md) ratified the shape:
per-host spine config as constraint, estate/repo tables as preference,
owner-ratified tier equivalence pins. Design the concrete schema: where
exactly the host config lives; how it declares available harnesses, reachable
models/gateways, and per-tier pins (model @ effort); the precedence order
estate default → repo override → host constraint/pin; what `spine model` and
`MirrorRows()`/WORKFLOW.md mirror rows display when the true routing is
host-dependent; and what doctor/audit check when a pinned model is not
reachable from the current host. Grill with the owner, then feed the standard
gate chain (PRD → tickets).

## Correction evidence (2026-08-30)

Status remains **open** pending the required fresh primary requirements review
and independent verification. Commit `2d5843d` (`fix(I072): keep alternate
host-blind`) removes the remaining review regression: a present malformed host
file still fails structural loading, while a valid file cannot gate, replace,
or filter `spine model --alternate`. The correction has command/model red-green
coverage for a config missing the selected flavor, an unavailable selected
harness, an unreachable selected primary route, and byte-identical no-config
alternate output. Its exact-SHA clean-worktree evidence includes focused/full/
race Go suites, vet, build, compiled alternate plus ordinary/effort/validate
matrix, both go@1 lanes, and `maipipe run full --wait` (`run HEAD #11`, passed).

## Correction 2 scope (2026-08-30)

Status remains **open**. Plan correction `4e2ce60`
(`docs(I072): ratify host config validation boundary`) makes the evidence-backed
`hostconfig.Load(path, flavors, lookup)` interface binding and records that
`Load` is the complete exported validation boundary. It does not add an
exported `Validate` seam or remove any schema, routing, audit, or validation
requirement.

The correction adds regressions for legacy no-config and valid-config
`model --alternate --json` bytes with no requested/host/pin trail, while a
malformed present config remains an exit-2/no-stdout failure. It also adds the
missing parser attacks for `base_url`, `modelOverrides`, and `auth_header`,
retains every prior prohibited-field case, and covers two available doctor
harnesses in lexical order with parallel explicit-path seams. Required fresh
primary requirements review, independent verification, both go@1 lanes, and
exact-final-SHA maipipe evidence remain pending before this ticket can close.

## Verifier correction 3 (2026-08-30)

Status remains **open** pending a fresh primary re-review and independent
verifier. `d684ef0` clarifies that executable lookup may consult `PATH`, while
JSON never expands environment values or executes anything. `97a20ad` makes
schema member spelling exact at every object depth, rejects present `null` for
typed optional members, rejects malformed UTF-8 before JSON decoding, and
keeps parser diagnostics free of raw host-config values. `fa41be9` removes the
unrequested `model.ValidateHostConfig` export in favor of `hostconfig.Load`,
preflights a present host config before default Claude/Codex discovery at the
CLI boundary, adds the required command-level no-host/identical/divergent/
forbidden-pin/post-identity-`--expect` matrix, and gives D16 a normalized
repository path, harness/tier key, and requested pair. `518a73b` additionally
redacts a forbidden host pin before launch-policy formatting can disclose it.

The correction preserves alternate compatibility, host-blind mirrors and
update behavior, existing audit verdict bytes after a valid preflight, and the
I073/I074/I077 boundaries. Fresh full/race/static/source-built-audit/go@1/
maipipe evidence is recorded in
`.superpowers/sdd/I072-resume-correction-worker4-report.md`; the ticket is not
closed by this implementation correction.

## Correction 4 (2026-08-30)

Status remains **open**. Documentation commit `20a2e5b` makes `executable` a
bare host executable name under current-platform Go path semantics, permits
omitted or empty `models` only for unavailable harnesses, locks the no-pin host
status to `reachable`, and specifies final-entry metadata. Code commit
`4ba395f` rejects absolute and relative path-bearing executable values before
lookup for every harness (including unavailable ones), while retaining lookup
and preference-routing skips for unavailable harnesses. It keeps every final
entry field for a byte-identical pin; a divergent pin clears requested-entry
aliases and alternate data, records final provenance `host`, and retains the
repository provenance in the additive requested trail.

The correction also removes audit's exported host-path/lookup test fields in
favor of its private explicit-path helper. Its official matrix covers all five
model configuration failures in plain, effort, and JSON modes; absent and
identical-pin doctor behavior; and malformed, executable, pin, and unpinned
audit boundaries. The reviewer assertion for no-pin `reachable` was accepted
because the locked design and plan both name the only status vocabulary as
`unconfigured`, `reachable`, and `pinned`.

Fresh primary re-review, a different independent verifier, and exact-final-SHA
verification evidence remain required before closure. This correction does not
expand into I073, I074, or I077.

## Resolution

Fixed 2026-08-30 at final I072 product SHA `50a6a6d`. The implementation adds
the closed, owner-local host capability/pin schema; exact estate → repository
→ host precedence; safe final-route output and controlled-launch refusal;
host-blind alternate/mirror/update behavior; D16 doctor diagnostics; and
preference-only audit preflight. Four adversarial correction rounds closed
schema, redaction, CLI ordering, executable-path, metadata, status, private
seam, and required regression-matrix gaps. A fresh primary review and a
different independent primary verifier passed focused/full/race/vet/build,
compiled CLI, mutation, and source-built audit checks at `50a6a6d`. The
superseded ticket-level maipipe attempt is non-evidence; the batch-final exact
SHA lane remains the ship gate. I073 and I077 are now unblocked.
