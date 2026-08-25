---
title: "i107-gate-panic-misconfiguration-codex"
created: 2026-08-24
handoff_ordinal: 17
---

# Handoff — i107-gate-panic-misconfiguration-codex (2026-08-24)

## Context

Primary repo: **`/Users/ldh/Projects/github.com/spine`**. Conventions in
`AGENTS.md` (Codex twin of CLAUDE.md) and `WORKFLOW.md`.

Build **I107** — convert a panic inside a gate check into `gate.Run`'s
documented misconfiguration exit. The ticket was grilled with the owner on
2026-08-24 and the grill **overturned part of it**; the design doc, not the
ticket's Fix section, is authoritative where they differ.

Read in this order, all paths relative to the primary repo:

1. `docs/specs/2026-08-24-i107-gate-panic-misconfiguration-plan.md` — the
   task-by-task plan you execute. Seven tasks, TDD steps with an explicit
   negative control each.
2. `docs/specs/2026-08-24-i107-gate-panic-misconfiguration-design.md` — the PRD.
   Decisions **D38–D44**.
3. `docs/adr/0021-a-gate-panic-becomes-a-contractual-misconfiguration-exit-toolchain-version-comparison-is-rejected-as-a-proxy.md`
   — why recovering panics is deliberate and why version comparison is rejected.
4. `docs/issues/I107-gate-analysis-classes-panic-when-the-binary-predates-the-toolchain.md`
   — the ticket, including its "Grilled 2026-08-24 — two corrections" section.

**Two things in the ticket's own Fix section are wrong and must not be built:**

- The ticket says "Exit 2 is right". Today's 2 is the **Go runtime's exit status
  for an unrecovered panic**, not `gate.Run`'s misconfiguration code — `Run`
  never finishes. Same for the missing results document: the process dies before
  `emit` writes it. Both guarantees are accidents of where the process dies, and
  this work makes them real.
- The ticket offers a `runtime.Version()`-vs-PATH preflight as "cheaper and
  stronger". It is **rejected**, not deferred (ADR 0021). The panic keys on the
  export-data *format* version, which does not change every Go release, so a
  1.26.7-built binary reads a 1.26.8 cache fine and the comparison would refuse
  a working setup. Do not build a preflight. Detection keys on the importer
  actually failing.

## State (verify before relying)

- `main` = **`f930083`** ("I107: PRD + ADR 0021 …"), clean tree except untracked
  `.DS_Store` and `README.md` — **both are the owner's; leave them alone and
  never stage them**. `main` is **not pushed** (owner's call, deliberately left).
- Lane at HEAD: `maipipe run full` **#27 passed @f930083**.
- `spine cursor`: effort `i107-gate-panic-misconfiguration`, PRD as above,
  `grill[x] prd[x] issues[x] implement[<]`. Derivation reports `blocking` on one
  thing only — the newest handoff carries a stale block — which **this document
  resolves**. Re-check with `spine cursor` after reading.
- `~/bin/spine` built at `7e020cb` with Go 1.27.0; machine toolchain 1.27.0.
  **In sync**, so the I107 bug does not reproduce naturally here — that is why
  Task 1 uses an injected seam rather than a second toolchain.
- `spine doctor` exits **1** on two long-standing D4 notes
  (`docs/issues/README.md`, `docs/adr/README.md`). **Pre-existing, not yours** —
  verified identical at `25eb985`. Do not "fix" it.
- Open tickets: I107 (in-progress, yours), I102 (low), I105 (low, no code),
  I106 (docs), I108 (low, filed today, blocked by I107 — **not yours**).
  Next free id: I109.

## Next steps

1. Branch from `f930083`: `git switch -c i107-gate-panic-misconfiguration`.
   Work on the branch; the owner merges `--no-ff`.
2. Execute the plan's Tasks 1–7 in order. Task 6 is explicitly optional and has
   a written stop condition — take it.
3. Every commit message cites **I107**.
4. Report per task: the exact negative-control command and its **observed red
   output**. A negative control that was reasoned about but not run does not
   count — this repo's standing rule is evidence before assertion.
5. Append `I107: gate panic misconfiguration done` to
   `.superpowers/sdd/progress.md` when Tasks 1–7 are complete.

## Task breakdown hints

- Files in play are few: `internal/gate/load.go` (Tasks 1, 2),
  `internal/gate/gate.go` (Task 3), plus two new `_test.go` files. The seams are
  `loadModule` (`load.go:101`) and the `fn(abs, cfg)` / `rfn(abs, cfg)` calls in
  `Run` (`gate.go:307-313`).
- Tasks 1 and 3 are independent and can run in parallel. Task 2 depends on 1;
  Task 4 depends on 3; Tasks 5–7 come after.
- The whole existing error contract you are plugging into is four lines:
  `gate.go:315-318` — `runErr != nil` → stderr → `return 2`, `emit` unreached.
  Read it before writing anything.
- `emit` writes a real file at `$SPINE_GATE_RESULTS` (`results.go:53-65`). That
  is what Task 4 asserts is absent.
- Keep the two message classes lexically distinct from the existing
  `--dir %s does not type-check` phrasing (D42). That phrasing belongs to a
  module that genuinely fails to compile and must be left untouched.

## Gotchas

- **Read exit codes unpiped.** In fish, `cmd 2>&1 | tail; echo $status` prints
  *tail's* status and hides a real failure. Use `cmd >/dev/null 2>&1; echo $status`
  or `bash -c '…; echo $?'`. This produced two false "exit 0" claims in an
  earlier session.
- **A gate stage exiting 2 with a `go/types`/`gcimporter` panic is I107 itself**,
  not your change. `make install` clears it. It fires on docs-only commits too.
- **No `maipipe daemon` restart** is needed after `make install` — stages exec
  `spine` from PATH per run.
- The maipipe stop hook demands `maipipe run full --wait` whenever HEAD moves,
  docs-only included. Commit first, then run the lane.
- `.superpowers/sdd/progress.md` is **gitignored** — a worktree's ledger dies
  with the worktree. If you work in a worktree, the per-ticket `done` line must
  also land on the branch the owner merges.
- **Never hand-edit the spine cursor block** (the HTML-comment-delimited
  `spine:cursor` region at the foot of a handoff). `spine` is the sole
  legal writer (`spine cursor start | tick | here | set`). Do not tick stages
  past `implement` — review/verify/ship are the owner's.
- **Stage explicit paths only** — never `git add -A` or `git add .`.
  `.DS_Store` and `README.md` stay untracked.
- Go stdlib only (ADR 0001). No `golang.org/x/tools`, no new dependencies.
- Do **not** install a second Go toolchain (`golang.org/dl/go1.26.7`). Named
  out of scope in the PRD.
- `gofmt -l .` and `go vet ./...` must be clean; the full lane is
  `SPINE_REQUIRE_MAIPIPE=1 make test` (18 packages) — run all of it, not just
  `internal/gate`.

<!-- spine:cursor -->
effort: i107-gate-panic-misconfiguration
prd: docs/specs/2026-08-24-i107-gate-panic-misconfiguration-design.md
tickets: I107
stages: grill[x] prd[x] issues[x] implement[<] functional-test[ ] review[ ] verify[ ] ship[ ] deploy[ ] docs[ ] handoff[ ]
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
