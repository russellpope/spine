---
title: "go@1 dogfood on spine: pack self-enabled, four seam defects fixed, maipipe I201, follow-ups I090/I093"
created: 2026-08-19
handoff_ordinal: 7
---

# Handoff — go@1 dogfood on spine: pack self-enabled, four seam defects fixed, maipipe I201, follow-ups I090/I093 (2026-08-19)

## Context

Dogfood pass over the just-shipped local-harness conventions (handoff
ordinal 6), driven from the deepthought session per its 2026-08-19 handoff
§1a–h. No cursor effort — six inline bug/dogfood tickets I088–I093 (I088,
I089, I091, I092 fixed; I090, I093 open), each carrying its commands and
output. Commits f9d2bf0 … 265efc9 on main.

What the pass proved: the eight classes + `mutate` run on spine itself and
through maipipe (`gate-go` 5/5 pass, seeded violations fail the right stage
with `code = go@1/<check>` in the daemon DB; `mutation-go` 3 rows); the
checkpoint doc round-trips byte-identically and embeds here; `spine model
--alternate` resolves default/override provenance. What it found: the
rendered region was **not loadable by maipipe** (plural `stages`, no
`schema = 0` — I091) and the results contract had three maipipe-invalid
shapes plus a battery env leak (I092). Spine's positive controls could not
see any of it because none exercises maipipe's parser — the lesson is in
maipipe I201 item 5.

## State (verify before relying)

- main `265efc9`, clean except untracked `.DS_Store`; **not pushed** (origin
  still behind since 2132d89). `~/bin/spine` rebuilt at 9fb68e3+ (`spine
  version` → gen 11). `make test` green; `go vet` clean.
- spine now carries `gate_pack: go@1` (WORKFLOW.md), a committed
  `maipipe.toml` (region + owner lanes `fast`/`full`), `docs/mutation-spec.json`.
  `spine doctor` → only pre-existing D4; `spine audit stages` clean.
- maipipe daemon was running during the pass; runs #1–#5 on project spine.
- `.superpowers/sdd/checkpoints/` has 001–003 from the round-trip (gitignored).
- Worktree `spine-wt-local-harness` + herdr team workspace still open;
  scratch branches `scratch/gate-neg*` deleted.

## Next steps

1. Owner: decide on push; remove the worktree (`git worktree remove
   ../spine-wt-local-harness && git branch -d local-harness-conventions`);
   close the herdr workspace.
2. I090 (routing audit blind to herdr/cmux team spawns) — next build ticket
   on the audit; I093 items 1–2 are code+test, 3–5 need an owner call.
3. Consider a producer-side contract test: a fixture that round-trips every
   class's `MAIPIPE_RESULTS` JSON through `maipipe validate` when the binary
   is on PATH (skip-free: gate on `exec.LookPath`, record as positive control
   in the test name). Would have caught I092 before ship.
4. Fleet: gen 11 is on spine only; `spine update` dry-run per repo before any
   sweep (deepthought is gen 10).

## Gotchas

- `maipipe run` reads `maipipe.toml` at the **pinned commit** — commit the
  rendered region before the first lane run or it refuses (`exists on disk,
  but not in <sha>`).
- `spine update --write` refuses to overwrite a tampered region (local edit);
  the D10 remedy is `--force`, which is repo-wide (drops README.md's local
  edit too — I093.4). Back up, force, restore.
- Unconfigured config-driven classes render as stages that exit 2 — set
  `gate_pack_config` or list them in `gate_pack_disabled` before composing
  `gate-go` into a lane (I093.3).
- The battery's verify env is now scrubbed of `MAIPIPE_RESULTS`/`SPINE_GATE_*`;
  any other stage-inherited env still reaches the tree's suite.
- D11 is shape-evident only: `gate: pass → fail` by hand stays silent.

<!-- spine:cursor -->
effort: local-harness-conventions
prd: docs/specs/2026-08-18-local-harness-conventions-design.md
tickets: I079-I087
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[x] deploy[x] docs[x] handoff[x]
<!-- /spine:cursor -->

## Checkpoint (newest): 003-dogfood-the-shipped-local-harness-conventions-on-spine-itsel.md

<!-- spine:checkpoint:facts -->
touched:
- internal/update/gatepack.go
- internal/gate/results.go
- internal/gate/mutate.go
- maipipe.toml
- WORKFLOW.md
gate: pass
sha: 265efc9ede4c229f135c38b558bfe722ec918427
effort_recommended: medium
written: 2026-08-19T16:31:36Z
<!-- /spine:checkpoint:facts -->

### Prior narrative (model-authored, not evidence)

## Task

Dogfood the shipped local-harness conventions on spine itself (deepthought handoff 2026-08-19 §1a–h) and close the cross-repo follow-through (§2).

## Conclusions

- go@1 pack is self-enabled on spine (I089); five classes + mutation-go pass under maipipe at the pinned commit.
- First live maipipe seam found four defects, all fixed: region TOML grammar + schema (I091); results line 0 / file "." / severity "warn" and battery env leak (I092).
- Bake-off positive control: hygiene classes catch committed binaries on 3/3 arms (docs/research/2026-08-19-…).
- Checkpoint round-trip, model alternate provenance, routing blind spot (I090) verified; minor follow-ups in I093.
- Cross-repo: maipipe I201 filed; deepthought spine PRD amended; /model-eval runs the binary.

## Next moves

- Owner: push spine (main ahead of origin, unpushed since 2132d89); close herdr team workspace; remove worktree spine-wt-local-harness.
- Owner call on I093 items 3–5 (unconfigured-class stages, --force scoping, D11 value tamper).
- Phase 1 continues: `/grill-with-docs` in maipipe with deepthought's maipipe execution-floor PRD.
