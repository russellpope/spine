# Gate-pack region ownership — is the byte range the wrong model?

- **Date:** 2026-08-19
- **Kind:** design research + adversarial verification record. **Result: negative** —
  keep the region, add four rules.
- **Related:** ADR 0016 (spine-managed region), ADR 0015 (gate packs), ADR 0002
  (ownership split), `docs/specs/2026-08-18-local-harness-conventions-design.md`,
  I091, I093; maipipe `docs/specs/2026-08-19-execution-floor-phase-1-design.md`
  and its I202–I206. Tickets filed from this doc: I095–I099.
- **Method:** two static passes plus live reproduction on a clean `git archive
  HEAD` copy of spine — one design pass over `internal/update/`,
  `internal/gate/`, `internal/doctor/`, `internal/model/` and maipipe's
  `src/pipeline.rs`/`src/provenance.rs`, then an independent adversarial pass
  that re-opened every citation and reproduced each failure mode. Claims are
  **[C]** confirmed by both passes, **[C-repro]** additionally reproduced live,
  **[R]** single pass, **[X]** refuted and corrected.
- **Source of the framing:** _A Programming Paradigm for Spatiotemporal
  Composability_ (Shi, Zhang, Cui, draft 2026-08-13), §5.2 and §6.1.

## The question

ADR 0016 gives spine a byte range inside another tool's file. maipipe's Phase 1
is simultaneously making `init` additive with "never rewrite existing bytes".
Both sides are hand-deriving the paper's §5.2 loader: entries with stable ids,
per-field dispatch, and a delimiter that answers *which binding is mine*. Should
spine adopt that model?

**No.** One file in the estate carries a region — spine's own
(`grep -l spine:begin */maipipe.toml`; `pi-pack/WORKFLOW.md:21` has `gate_pack`
empty) **[C]**. The entry model costs more than that blast radius is worth. What
the paper does contribute here is §6.1's boundary test and one design rule that
falls out of it.

## How the region works today

The region is a **line splice, not a TOML operation** **[C]**. spine owns bytes
between `# spine:begin gate-pack ` and `# spine:end`
(`internal/update/gatepack.go:19-20`); `renderGateRegion` (`:106-134`) is a
`strings.Builder` — spine never parses TOML and never emits through a
serializer; `planMaipipe` reassembles `lines[:begin] + render + lines[end+1:]`
(`:194-195`, bounds at `:209-236`), whole-file atomic write at
`update.go:182`. Everything outside the markers is preserved because it is never
touched.

The ownership test is a **per-line shape matcher**, not ownership
(`unrecognizedRegionLines`, `:244-288`): it accepts any line that *could* have
been rendered by *some* configuration — including `env = { <known
SPINE_GATE_* var> = <any quoted string> }` (`:277-284`). A report with
unrecognized lines flips to `SkippedUnrecognized` unless `--force`
(`update.go:156-171`), which then drops the flagged lines.

The "spine remembers every default it has ever shipped" property is real but
lives elsewhere: `History` on the **model table** (`internal/model/model.go:121`,
consumed at `:201`, `:240`, `:402`, `:502`; glossary at `CONTEXT.md:45-48`). The
gate-pack region has no history and no provenance **[C]**.

**The fact that governs everything:** maipipe's `definition_hash` is the **git
blob SHA of `maipipe.toml`'s bytes** (`maipipe/src/provenance.rs:377-397`), and a
mismatch against the passed-full baseline is a blocking error carrying `maipipe
gate approve-definition <hash>` (`provenance.rs:255-262`, `gate.rs:526-543`)
**[C]**. Bytes are load-bearing: a cosmetic re-render costs a human approval.
"Do not change bytes you do not have to" is a grounded rule, not a preference.

## Reachable failure modes

1. **Region deletion is unimplemented and the obvious implementation breaks
   spine's own repo [C-repro].** With `gate_pack:` cleared, `planMaipipe`
   returns `ok=false` and leaves the region alone (`:137-143`), maipipe.toml
   drops out of the report entirely, and `doctor` is silent because
   `gatePackCheck` returns nil on an empty pack (`doctor.go:170-172`) — the
   stages keep executing. But simply splicing the region out yields `pipeline
   "full" stage "gates": composes unknown pipeline "gate-go"`, exit 1: spine's
   own `maipipe.toml:66-68` composes the pack from *outside* the region, which
   is what ADR 0016 intends. There is no uninstall, and the naive one is worse
   than none.
2. **spine can write a definition maipipe cannot load, with no `--force` and no
   warning [C-repro].** Move a `[[pipelines.gate-go.stage]]` block three lines
   past `# spine:end` — a plausible hand edit, and a plausible merge
   resolution. `spine update` reports nothing; `spine update --write`
   re-renders the stage back inside the region; the file now has two stages
   named `tskip`; `maipipe validate` exits 1 and every lane in the repo stops
   loading. The same class fires from a pre-existing `[pipelines.gate-go]`
   outside the region (duplicate key) **[C-repro]**, and it is the I091 class
   ("spine's positive controls assert spine's own string shape, not maipipe's
   grammar") recurring.
3. **`--force` is all-or-nothing and already destructive [C-repro].** I093 item
   4 records it dropping a local edit in `docs/issues/README.md`; a pristine
   checkout of spine *today* reports `skipped docs/issues/README.md —
   unrecognized local edits`, so the D10 remedy would drop it again.
4. **Region `env` values are shape-recognized, not value-recognized [C-repro].**
   A hand-edited `SPINE_GATE_TSKIP_ALLOW` is reverted by `spine update --write`
   with no `--force`. But this is **misclassification, not silence** **[X]** —
   `doctor` does fire `D10 warn … region is stale for the pinned pack` and
   `spine update` prints the exact `-/+` diff. The first pass ranked this the
   sharpest defect on the strength of "no warning"; that was wrong.
5. **A new check class in `go@1` re-renders every adopting repo** — bytes,
   blob, approval. Bounded today by there being one such repo **[C]**, but see
   contradiction 2: the semantic half is worse than the byte half.
6. **Markers can be made unrepairable** by a stray `# spine:end`: bounds error,
   `--force` refused, `D10 error`, hand repair only **[C-repro]**.

## Mapping — and where it breaks

| Paper §5.2 | Estate | Honest status |
|---|---|---|
| Entry (Def. 74) | a `[pipelines.X]` + its stage array | Real; `(pipeline, stage)` is a de-facto id |
| Per-field dispatch | region replaced wholesale | Absent — the actual gap |
| Thm 73: quiesced state = f(final config) | `spine update` ∘ `maipipe init`, either order | **Half.** Parsed config is order-free (`pipelines: BTreeMap`, `pipeline.rs:100`); bytes are not, and bytes are what the gate hashes |
| Alg. 7 delimiters, `own(γ)` | `# spine:begin`/`# spine:end` | Breaks. The paper's tag is a *derivation-lineage* tag inherited by derived contexts and redrawn at each reassignment; the region is a **positional** range, so moving a stage across the marker transfers ownership — which is failure mode 2 |
| Fresh never-reused uid | none | Absent; its absence is failure mode 4 |
| §6.1 boundary | the region | Fails the exclusivity half only: git and the user write it. **[X]** maipipe does *not* — it is contracted not to (`maipipe/CONTEXT.md:278-282`; Phase-1 design story 11 and `:80`), and the file is git-tracked, so `git checkout -- maipipe.toml` is a restore path |
| Withholding / emission | `spine update` plan vs `--write` | Already implemented; the dry-run plan *is* output-commit withholding |

What does not translate at all: there is no runtime, no dependents to notify, no
quiescence to prove. Reconciliation happens between two CLIs in two languages,
possibly weeks apart, with git in between. "Reconcile", "quiesce" and
"delimiter" applied to this file are vocabulary. Three things are not: per-field
dispatch, provenance as a recorded fact, and the boundary test.

## What to ship, in order

1. **Parse before write (I096).** After splicing, parse the result and — when
   the binary is found — run `maipipe validate <path>` before
   `WriteFileAtomic`; refuse rather than write. Confirmed feasible: `maipipe
   validate` accepts a path, returns OK on spine's real file and exit 1 on both
   the duplicate-key and moved-stage variants **[C-repro]**. This is the only
   proposed rule that stops spine writing an unloadable definition, and it is
   the direct fix for the I091 class.
2. **`--force <path>` scoping** — already filed as I093 item 4, already caused
   live loss, blocked on an owner call. Unblock it; do not re-file.
3. **Region removal on opt-out (I097), corrected.** `gate_pack: ""` with an
   existing region must plan its removal — but must **refuse** while any
   out-of-region stage composes `gate-go` (or, once maipipe I204 lands,
   `mutation-go`), reporting which stages to remove first. Sequenced after
   I096 so the plan passes the validate gate.
4. **Pack pinning enforced by a golden list (I098).** See contradiction 2.

**Withdrawn: value-evident `env` recognition.** The first pass called this "do
this first regardless of everything else". It rests on a predicate that provably
cannot decide **[C-repro]**: a legitimate `gate_pack_config` change in
WORKFLOW.md and a region hand-edit produce byte-identical plan states (Pending,
zero unrecognized, the same `-/+` env diff). Comparing on-disk to the current
render flags both, which turns the documented configuration workflow into a
`--force`-only path — and `--force` is the unsafe verb of failure mode 3.

**Withdrawn: a gitignored `.spine/` sidecar.** It dies on every clone, worktree
and branch switch (six worktrees exist in this estate today; maipipe runs every
pipeline in a detached worktree, `workspace.rs:18-42`, so no gate-lane
invocation would ever see one), it misreads every revert and bisect as N user
overrides, and spine does not own adopting repos' `.gitignore`. If a last-render
memory is ever wanted, it belongs **in-band on the marker line spine already
rewrites** — `# spine:begin gate-pack go@1 <fingerprint>` — which is a comment
(so maipipe's `deny_unknown_fields` is untouched) and travels with the file,
which is the property the paper's delimiter actually has.

**Withdrawn: keying entries by stage `name`.** Circular — spine claims the right
to rewrite `name`, so a rename would read as "stage deleted by the user" and
silently uninstall a gate check. Today's shape matcher is strictly safer there.

## Contradictions for the owner

1. **ADR 0016 vs ADR 0002 vs the code (I095).** ADR 0016 says edits inside the
   region are "unrecognized and reported, never silently kept"; the code
   silently discards divergent `env` values and cites ADR 0002 at
   `gatepack.go:238-243` for doing the *reverse* of ADR 0002's rule ("only
   divergent values survive"). This decides whether any preservation machinery
   is needed at all.
2. **`go@1` pinning is not enforced (I098).** `PackVersion = 1` is an unrelated
   const (`internal/gate/gate.go:31`) and `CheckNames()` enumerates two live
   maps (`:147-158`), so adding a check silently adds a **blocking** stage to
   every repo pinned at go@1 — precisely what ADR 0015 item 2 and spec story 23
   forbid.
3. **D10's implemented scope exceeds its spec (I099).** Story 18 scopes D10 to
   region integrity; the implementation also emits a staleness warn on any
   Pending plan and an *error* sourced from WORKFLOW.md
   (`doctor.go:178-195`, `:185-187`).
4. **`mutation-go` is defined and composed nowhere (I099).** spine's own
   `maipipe.toml:34-39` renders the pipeline; no audit lane composes it. The
   scaffolding is maipipe-side I204 — open and blocked — so ADR 0015 item 5 is
   satisfied by neither side today, in the only repo that adopted the pack.
