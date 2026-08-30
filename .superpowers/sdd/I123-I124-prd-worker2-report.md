# I123 / I124 PRD worker2 report

Date: 2026-08-30

## Delivered paths

- `docs/specs/2026-08-30-i123-update-gate-config-advisory-design.md`
- `docs/specs/2026-08-30-i123-update-gate-config-advisory-plan.md`
- `docs/specs/2026-08-30-i124-update-force-file-authority-design.md`
- `docs/specs/2026-08-30-i124-update-force-file-authority-plan.md`
- `docs/issues/I123-update-advises-on-unconfigured-gate-classes.md`
- `docs/issues/I124-update-force-file-scopes-overwrite-authority.md`

## Decisions

- I123 requiredness is explicit update gate-pack metadata, distinct from
  optional environment consumption. It advises only the four required classes,
  sorted bytewise by class, on stdout after plan/preflight and before writes.
  Empty `tskip_allow` and config-free classes are excluded; advice changes no
  exit code, stage, disable list, or write bytes.
- I124 adds repeatable flags-first `--force-file` values normalized to exact
  current-plan report paths. Empty, absolute, traversal, duplicate, unknown,
  and unmanaged values fail before writes. `--force` and `--force-file` are
  mutually exclusive fail-closed modes; this preserves standalone global
  behavior without an ambiguous union.
- Scoped authority enters the existing candidate/preflight/atomic-write path.
  It cannot repair damaged markers or bypass maipipe validation; a refused
  selected candidate writes no file.
- The lead's round-2 ruling provides implementation and closure authority.
  Both plans therefore retain ordinary contradiction/scope stops but no
  redundant owner-acceptance gate; I124 keeps its primary review floor.

## Requirements attacks completed

- I123: optional-versus-required `tskip`, disabled precedence, frozen pinned
  classes, ordering nondeterminism, output channel/exits, write timing,
  preflight refusal, one-key/one-disable isolation, and implicit-disable
  regressions.
- I124: normalization bypasses, normalized duplicates, non-plan membership,
  global/scoped ambiguity, marker damage, clean selection behavior, exact
  report wording, maipipe candidate preflight order, and all-or-nothing
  byte stability.

## Evidence read

`WORKFLOW.md`; I123, I124, and I093; ADR 0015; current `cmd/spine` update
grammar/tests; `internal/update` gate-pack metadata/rendering/tests; I096/I097
and their preflight specs; and `internal/fsutil` atomic writes. No production
code was changed.

## SHA

`9d7ac1c` — documentation PRD/plan and ticket-link commit.
