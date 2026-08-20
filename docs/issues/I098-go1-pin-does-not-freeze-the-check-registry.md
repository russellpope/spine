---
id: I098
title: "`gate_pack: go@1` does not freeze anything: adding a check to the registry silently adds a blocking stage to every pinned repo"
severity: high
status: fixed
affects: [I082, I083, I084, I086]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

## Problem

Filed 2026-08-19 from `docs/research/2026-08-19-gate-pack-region-ownership-analysis.md`.

ADR 0015 item 2 and spec story 23
(`docs/specs/2026-08-18-local-harness-conventions-design.md`) both promise that
a repo pins `gate_pack: go@1` and "moves to go@2 deliberately, so that a pack
release never silently changes my gate."

The pin enforces nothing. `PackVersion = 1` is an unrelated const
(`internal/gate/gate.go:31`), and `CheckNames()` (`:147-158`) enumerates the two
live maps. Registering a check in `checks` (or `reportChecks`) without touching
`PackVersion` inserts a stage into the rendered region of every repo pinned at
go@1 on their next `spine update` — a **blocking** stage, since `gate-go` is a
gating lane. That is the outcome the requirement forbids: a pinned gate changes
enforcement without the owner acting.

Two costs, and the semantic one is the serious one:

- **Semantic** — enforcement changes under a pin.
- **Byte** — the new stage changes the region's bytes, hence the file's blob,
  hence maipipe's `definition_hash`
  (`maipipe/src/provenance.rs:377-397`), so each adopting repo must run
  `maipipe gate approve-definition`. Bounded today by there being one adopting
  repo, and by drift only blocking once a passed-full baseline exists
  (`provenance.rs:255-262`).

## Fix

Bind the registry to the version with a test rather than prose:

1. A golden list of the go@1 class names in `internal/gate/gate_test.go`.
   Adding, removing or renaming a class fails the build until the author either
   bumps `PackVersion` and forks the golden list, or deliberately updates the
   go@1 list with a recorded reason.
2. Decide and document the pinning semantics, because today the word means
   neither of its two possible readings:
   - **frozen list** — a newer binary still renders go@1 from its frozen class
     list, and new classes are only reachable via `go@2`; or
   - **attribution only** — the version identifies the finding-attribution
     strings, not the class set, in which case ADR 0015 item 2 and story 23 must
     be downgraded to say so, and repos are told a pack release can add checks.
   An old binary already refuses a newer pack name
   (`internal/update/gatepack.go:146-153`), which is the other half of the
   frozen-list reading.
3. Whichever wins, `spine update`'s plan should call out "this render adds N
   stage(s) not previously present" so the byte/approval cost is visible before
   `--write`.

## Acceptance criteria

- [x] Golden-list test present; adding a dummy check to `checks` fails it, and
      the failure message names the pin contract
- [x] Pinning semantics recorded in ADR 0015 (amendment) and reflected in the
      spec's story 23 wording
- [x] If the frozen-list reading wins: rendering `go@1` from a binary that also
      ships `go@2` produces exactly the go@1 class list, with a test
- [x] `spine update` plan flags added/removed stages for an adopting repo

## Resolution (2026-08-20)

The pin is now a **frozen class list**, and the render is keyed by it.

- `internal/gate/gate.go`: `packClasses` maps a pack version to the exact
  class list that version renders; `PackClassesFor("go@1")` and `PackIDs()`
  expose it. The registry no longer decides what a pinned repo runs.
- `internal/update/gatepack.go`: `renderGateRegion` iterates the pinned
  version's frozen list (not `gate.CheckNames()`), and `planMaipipe` accepts
  any pack version this binary ships, naming all of them when it refuses one
  it does not.
- `internal/gate/gate_test.go`: `TestGo1FrozenClassList` holds both
  registries and the go@1 list to a literal golden list; its failure message
  states the pin contract and the two legitimate ways out (fork a list under
  a new `PackVersion`, or edit go@1's with a recorded reason).
- `internal/update/gatepin_test.go`: a test seam (`packClassesFor`) stands in
  a binary shipping go@1 **and** go@2; go@1 renders exactly the go@1 classes
  and no trace of go@2's, at both the render and the full-`update` level,
  while go@2 renders its own list.
- `FileReport.StagesAdded/StagesRemoved` + `cmd/spine/main.go`: the plan
  prints "maipipe.toml: this render adds N stage(s) not previously present:
  …" and names the `maipipe gate approve-definition` re-approval the changed
  region bytes cost. Computed only where a region already exists — a
  first-time region is wholly visible in the plan diff. It is plan output
  only, so it composes with I096's all-or-nothing write refusal rather than
  competing with it: the refusal still aborts the whole run before anything
  is written.

**Deviations from this ticket's Fix items, both controller rulings:**

1. Fix item 2 asked for an **ADR 0015 amendment; ADR 0015 was not edited.**
   `docs/adr/README.md` permits exactly one edit to an accepted ADR (the
   status flip a supersede performs). Frozen-list is what ADR 0015 item 2 and
   its "a repo pins `gate_pack: go@1` and moves deliberately" consequence
   already promise, so nothing in its Decision changes and there is no
   reversal to record. The explicit semantics live in spec story 23's
   wording, in `gate.packClasses`' doc comment, and in the golden-list test's
   failure message, each citing ADR 0015 as the source of the promise.
2. Fix item 2's two readings were **not weighed here**: the controller ruled
   the semantics are frozen-list, because that is what ADR 0015 item 2 and
   story 23 already promise and because an old binary already refuses a newer
   pack name (`internal/update/gatepack.go`) — the other half of the same
   reading. Acceptance box 3's frozen-list branch therefore applies.

## Evidence

- `go vet ./...` — clean.
- `make test` (`go test ./...`) — all packages ok, including
  `cmd/spine` and `internal/{gate,update}`. No test skipped.
- New tests: `TestGo1FrozenClassList`,
  `TestPackClassesForRejectsPacksNotShipped`,
  `TestPackClassesForReturnsACopy` (gate);
  `TestPinnedPackRendersItsOwnFrozenClassList`,
  `TestPinnedRepoUpdateIgnoresLaterPackClasses`,
  `TestPlanReportsAddedAndRemovedStages`, `TestUnchangedStageListReportsNoChurn`
  (update); `TestUpdatePlanFlagsAddedGateStages` (cmd/spine).
- **Negative control (required):** adding `"dummy-negative-control"` to
  `checks` and running `go test ./internal/gate/ -run TestGo1FrozenClassList`
  → `FAIL`, with `the check registry and the go@1 class list disagree:` and
  the contract text "``gate_pack: go@1`` pins a frozen class list (ADR 0015
  item 2, spec story 23)…". Reverted.
- **Negative control (churn reporting):** `TestUnchangedStageListReportsNoChurn`
  — a `gate_pack_config` change re-renders the region and reports no added or
  removed stage, so the plan's stage lines are not "the region changed" by
  another name.
- spine's own `maipipe.toml` region is byte-identical: the go@1 frozen list is
  the registry's current contents, so no adopting repo re-approves anything
  for this change.

## Notes

The estate's blast radius is currently one repo (spine's own), which is why this
is filed now rather than after the second adopter — it is cheap while the golden
list has one consumer.
