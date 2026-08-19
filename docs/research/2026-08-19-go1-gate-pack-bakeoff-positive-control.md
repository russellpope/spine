# go@1 gate pack — real-world positive control on the 2026-08-18 bake-off corpus

**Date:** 2026-08-19 · **Kind:** dogfood record (deepthought handoff 2026-08-19 §1e) ·
**Related:** spec `docs/specs/2026-08-18-local-harness-conventions-design.md` (story 19),
ADR 0015, tickets I082–I084, I088–I092; deepthought `docs/research/` bake-off I038.

Sources: the three frozen `audit` workspaces under
`~/bakeoff-runs/2026-08-18-full-tier/audit/` (`opencode-workspace`,
`claude-workspace`, `pi-workspace/vsphere-inventory`) — each a Go
`vsphere-inventory` build by a different harness against the same task.
The workspaces are not git repos; `binary-hygiene` and `gitignore-control`
read `git ls-files`, so each was copied to scratch and `git init && git add
-A && git commit` — i.e. "what would be committed if the agent pushed as-is",
which is the corpus-defect carrier the classes were derived from. The frozen
trees were not touched. spine binary: gen 11 at `e1d8047` (post I088–I092).

## Config used

`SPINE_GATE_BUILD_OUTPUTS` = each Makefile's `BINARY`/`VCSIM`
(`vsphere-inventory,vcsim-bin` / `vsphere-inventory` / `bin/vsphere-inventory,bin/vcsim`);
`SPINE_GATE_N_PLUS_ONE_CLIENTS=Retrieve,RetrieveOne,Properties,Find,FindAll,List`
(govmomi client verbs). `test-enum-vs-spec` and `fixture-manifest` were run
against `task.md` to confirm the misconfiguration path only — the corpus has
no `<!-- spine:enum -->`-marked spec and no fixture manifest.

## Results

| class | opencode | claude | pi |
|---|---|---|---|
| tskip | 0 | 0 | 0 |
| deferred-cleanup-errcheck | 0 | **exit 2** (tree does not type-check) | **1**: `internal/inventory/simulator_test.go:81` `defer os.RemoveAll()` |
| dead-code-callgraph | 0 | **exit 2** (same) | 0 |
| n-plus-one | 0 | 0 | 0 |
| binary-hygiene | **2**: `vcsim-bin`, `vsphere-inventory` (Mach-O) | 0 | **2**: `bin/vcsim`, `bin/vsphere-inventory` (Mach-O) |
| gitignore-control | **2**: both outputs not ignored | **1**: `vsphere-inventory` not ignored | **2**: both outputs not ignored |
| test-enum-vs-spec | exit 2: `task.md has no enum marker` (remedy named) | — | exit 2 |
| fixture-manifest | not run (no manifest; unset → exit 2 by design) | — | — |

Exit codes: 0 pass, 1 findings, 2 misconfiguration — as the contract says.

## What this shows

- **The two hygiene classes catch the corpus's headline defect on 3/3 arms**
  (committed binaries and/or un-ignored declared outputs). `claude`'s tree
  had no binary committed but still would have committed its build output
  on first `make`; `opencode` and `pi` carried *two* Mach-O executables each.
- **`deferred-cleanup-errcheck` found one real site in the pi arm** (a test
  helper's `defer os.RemoveAll()`), none in the others.
- **`dead-code-callgraph`, `tskip`, `n-plus-one` were clean** wherever they
  could run — consistent with small single-purpose trees; n-plus-one's
  silence is with a guessed client verb list and says little.
- **`claude` does not compile** against the pinned govmomi
  (`internal/inventory/inventory.go:215` `ext.CanonicalName undefined`, and
  7 more). The type-checked classes refuse with exit 2 — correct — but the
  message names the *downstream* symptom (`could not import
  vsphere-inventory/cmd (no export data)`) rather than the first compile
  error. Worth a small follow-up: surface the first `go list -e` package
  error in the misconfiguration message (filed as part of I093).

## Not covered here

`mutate` against the corpus (needs per-tree authored probes — an eval-round
activity, not a positive control); `fixture-manifest` / `test-enum-vs-spec`
on real inputs (the corpus has none); running the pack via maipipe on these
trees (done on spine itself, I089/I092).
