# spine v2 — adopt, handoff, eval, new profiles (design)

**Date:** 2026-07-02 · **Status:** approved-pending-plan · **Repo:** github.com/russellpope/spine

## Problem

spine v1 (gen 1: `init`, `update`, `adr`, `doctor`, `version`) shipped and un-stranded hbmview, but
the fleet audit behind the v2 brainstorm found four standing gaps:

- **12 repos have no WORKFLOW.md.** praxis, ultima-dci-edition, moo-clone, notetui, jarvis,
  ai-virt-framebuffer, the four knowledge repos, home-lab-admin, pure-automation. Retrofitting was
  explicitly deferred from v1 (`init` just skip-if-exists; no claim intelligence for hand-authored
  files).
- **52 handoffs across 9 repos, zero tooling.** All follow `YYYY-MM-DD-<topic>.md` under
  `docs/handoffs/`, but there is no way to list them, open the latest, or scaffold a new one — the
  naming convention lives only in the /handoff skill's prompt.
- **The eval corpus has no machine-readable state.** local-model-evaluation holds 8 model
  submissions; scores live in a README table, audits in per-model REVIEW.md files, and the
  wire→audit→score→compare→remediate→rescore loop is re-typed every session (~15 sessions so far).
  The planned `/model-eval` skill needs an artifact convention to drive against — and spine must own
  it, or the schema lands back in prompts.
- **Three fleet stacks are unservable by current profiles**: swift (ai-virt-framebuffer, moo-clone),
  knowledge (ai_infra_notes, observability_notes, obsidian-ep-vault, home-lab-admin), infra
  (home-lab-admin ansible/+helm/, pure-automation ansible/).

AI agents are the primary consumer of this tool. That biases every choice toward deterministic
output, stable exit codes, and `--json` on anything structured.

## Decisions (settled during brainstorm)

1. **Eval seam: schema in spine, process in skill** (→ ADR 0007). spine owns the `docs/evals/`
   structure as versioned templates and validates shape; `/model-eval` drives the loop and writes
   all results. Stage names ship as template *data* — spine's Go never branches on them, never
   scores, never compares. Loop changes are a template bump, not a code change.
2. **Adopt: compose, don't map** (→ ADR 0008). One unified dry-run plan built from existing init
   and update machinery. Legacy trees stay where they are with info-level findings (the ADR 0005/D5
   pattern); no mapping keys in WORKFLOW.md; dirs outside spine's vocabulary are ignored. The fleet
   converges on one shape instead of institutionalizing divergence.
3. **Handoff: `new`, `list`, `latest`, and `latest --fleet`.** Same seam as eval: spine owns naming
   plus skeleton schema; the /handoff skill owns content.
4. **Profiles: swift, knowledge, and infra.** Infra included on fleet evidence (home-lab-admin,
   pure-automation) — its detection signals live one level below root, so detection must scan
   subdirectories.
5. **One release, stdlib holds, generation 1→2** (→ ADR 0006). The v2 command tree is exactly two
   levels — the shape `adr new|list` already proves. Sub-actions stay single-token (`add-run`, not
   `eval run new`). Cobra's trigger moves to "three levels or persistent flags."
6. **Acceptance mutates representative live targets, one per new surface** (v1's hbmview
   precedent): adopt praxis (go, hard legacy), moo-clone (swift), home-lab-admin (infra),
   obsidian-ep-vault (knowledge); eval-retrofit local-model-evaluation; fleet handoff scan
   (read-only). The remaining ~8 repos converge post-ship.

## Non-goals (v2)

- No per-repo generation pin (single-operator fleet; the downgrade guard already exists).
- No `--migrate` / no mass-moves of legacy trees (standing non-goal, upheld).
- No network, no config file, no deletion — unchanged from v1.
- No eval judgment in Go: no scoring, no comparing, no stage validation, no stage graph.
- No cobra. No `--fleet` on `list` (only `latest`).
- The `/model-eval` skill itself is the next project, not this one.

## Architecture

```
spine/
  cmd/spine/main.go          # dispatch: adds adopt, handoff, eval (two-level, stdlib)
  internal/meta/             # NEW: shared front-matter read/write (extracted from adr)
  internal/adopt/            # NEW: detection + plan + apply (composes scaffold/update)
  internal/handoff/          # NEW: new/list/latest/--fleet
  internal/eval/             # NEW: new/add-run/list
  internal/scaffold/         # profile manifests gain swift/knowledge/infra
  internal/update/           # consults per-profile machine-owned manifest
  internal/doctor/           # D1 profile-aware; new D7, D8
  templates/                 # + handoff skeleton, evals README, eval.md, run-record,
                             #   3 profile definitions; VERSION → 2
```

- **Generation 2.** New templates: handoff skeleton, `docs/evals/README.md`, `eval.md`, run-record,
  and the three profile definitions. Existing gen-1 repos (ccq, hbmview, spine) see a near-empty
  `update` diff: stamp bump plus any template drift.
- **Per-profile machine-owned manifest.** The machine-owned file set becomes profile-dependent
  (knowledge repos have no `docs/harness-interface.md` — nothing to harness). Each profile compiles
  in a manifest of the files it owns; `update` and doctor D1 consult the stamped profile. The
  knowledge profile scaffolds `docs/{handoffs,adr}` by default; specs/issues are opt-in.
- **Opt-in machine-owned class.** `docs/evals/README.md` is machine-owned but created only on first
  `eval new`; `update` regenerates it only where `docs/evals/` exists. init/adopt never create
  `docs/evals/`.
- **`internal/meta` refactor.** Front-matter parse/render moves out of adr into a shared package;
  adr, eval, handoff, and doctor all consume it. No behavior change to adr intended.
- **AI-first output contract, uniform:** exit 0 = clean/current/created, 1 = findings-or-pending,
  2 = hard error, across all commands. `--json` on `handoff list|latest`, `eval list`, `adopt`
  (the plan), and retrofitted onto `adr list`. Errors → stderr; data/diffs/JSON → stdout.

## Commands

### `spine adopt [--dir D] [--profile P] [--name N] [--write] [--force]`

Retrofit a pre-spine repo. Dry-run by default: prints the full plan, exit **1 if pending, 0 if
nothing to do** (idempotent — adopt on an adopted repo is a no-op), 2 on error. `--write` applies
atomically. Composition of existing machinery; adopt introduces no new file-mutation semantics.

- **Profile detection precedence:** explicit `--profile` → code signals (go.mod → go-service or
  library-cli, Cargo.toml → rust, Package.swift or *.xcodeproj → swift, pyproject/setup.py →
  py-tool, package.json+UI framework → ui, *.pptx/*.key → presentation) → infra signals (`ansible/`
  containing ansible.cfg or playbooks, `helm/`, `terraform/`, k8s manifests — scanned one level
  below root) → knowledge (`.obsidian/` present, or ≥80% of git-tracked files are .md). Code beats infra beats
  knowledge: praxis lands go-service despite its runbooks; home-lab-admin lands infra;
  obsidian-ep-vault lands knowledge.
- **Plan actions:**
  - `create` — missing dirs and machine-owned files per the profile manifest; WORKFLOW.md stamped
    generation 2 with the detected profile.
  - `claim` — CLAUDE.md without markers takes update's existing legacy path (wholesale claim on
    gen-0 match, otherwise marker block inserted at top, all hand-written content preserved below);
    WORKFLOW.md present but unstamped takes the gen-0 key-extraction claim.
  - `skip` — already conventioned artifacts.
  - `info` — legacy trees noted for transparency (docs/superpowers accumulation, pre-spine ADRs,
    "not spine's" dirs like praxis's decisions/design/operations). No exit-code effect.
- `--force` passes through to update's unrecognized-local-edits behavior, relevant only when
  machine-owned files exist in odd states.
- **Post-condition (asserted in acceptance):** after `adopt --write`, `spine doctor` is clean
  (info-only) and `spine update` is a no-op.

### `spine handoff new <topic> | list | latest [--fleet DIR]` (`[--dir D] [--json]`)

- `new`: slugify → `docs/handoffs/YYYY-MM-DD-<slug>.md` from the skeleton template (front-matter:
  title, created; sections: Context / State / Next steps / Gotchas). Same-day collision on the same
  slug: exit 2, never overwrite. Prints the created path.
- `list`: newest-first table (date / topic / path). **Date parses from the filename**, title from
  front matter when present, else the filename slug — all 52 legacy handoffs (no front matter) list
  correctly with zero migration.
- `latest`: prints the single latest path (agents cat it). `--json` emits `{path, date, topic}`;
  `list --json` emits an array of the same shape.
- `latest --fleet DIR`: walks `DIR/*/docs/handoffs`, one row per repo (repo / age / latest path),
  silently skipping repos without the dir. Read-only. Fleet order: most recent first.

### `spine eval new <title> | add-run --eval E --name N | list` (`[--dir D] [--json]`)

The `docs/evals/` convention:

```
docs/evals/
  README.md                    # convention doc, machine-owned, created on first `eval new`
  2026-07-02-govmomi-cli/
    eval.md                    # front-matter: title, created, prompt, rubric (paths) + free prose
    runs/
      qwen-3.6-27b.md          # front-matter: name, created, model, stage, score + skeleton body
```

- `new <title>`: creates the dated eval dir, `eval.md`, empty `runs/`, and `docs/evals/README.md`
  if absent. Prints the eval dir path.
- `add-run --eval E --name N`: scaffolds `runs/<name>.md` from the run-record template; `--eval`
  accepts the dir name with or without date prefix — an ambiguous or unmatched suffix is exit 2
  listing the candidates. Exit 2 if the run exists.
- `list`: evals × runs, with each run's `stage` and `score` printed **verbatim** from front matter —
  opaque strings, never interpreted. Empty/missing `docs/evals/` lists nothing, exit 0.
- The run-record body ships skeleton sections — Wire / Audit / Score / Compare / Remediate /
  Rescore — as template data. `/model-eval` fills them and edits front matter directly; changing the
  loop is a template bump (new generation), never a Go change.

### Unchanged commands

`init`, `update`, `adr`, `doctor`, `version` keep their v1 contracts. Deltas: init/update consult
per-profile manifests; `adr list` gains `--json`; doctor grows the checks below.

## Doctor (delta)

| ID | Check |
|----|-------|
| D1 | Now profile-aware: required dirs/files come from the stamped profile's manifest. |
| D7 | *New.* Eval structure, only where `docs/evals/` exists: `eval.md` and `runs/*.md` parse with required front-matter keys. Malformed = warn. Stage/score **values** never validated (opaque per ADR 0007). |
| D8 | *New.* Handoff naming: files in `docs/handoffs/` not matching `YYYY-MM-DD-*.md` = info. |

Read-only, exit semantics unchanged (info never fails; ADR 0005).

## Error handling

Uniform contract: exit 0 / 1 / 2 as above, errors → stderr, data → stdout, atomic writes
(temp + rename), no deletion ever — adopt never deletes, `new`/`add-run` never overwrite.

## Testing

TDD (red → green per task); table-driven unit tests; golden-file integration tests against the
built binary in `t.TempDir()`. Real-file fixtures over synthetic constants (the v1 lesson — the
hbmview fixture caught an inverted constant that unit tests missed):

1. Golden init trees for swift / knowledge / infra.
2. Gen-1→2 update fixture from ccq/hbmview's actual current files → stamp-bump-only diff.
3. **Real-file adopt fixtures** — snapshots of praxis (hand CLAUDE.md with invariants, pre-spine
   ADRs, superpowers tree, decisions/), home-lab-admin (infra signals below root),
   obsidian-ep-vault (.obsidian), moo-clone (xcodeproj) → golden plans; post-adopt invariant
   asserted per fixture: *doctor clean, update no-op*.
4. local-model-evaluation fixture → eval scaffold + add-run golden; D7 on malformed records.
5. Legacy-handoff fixture (front-matter-less files) → list/latest correctness; multi-repo temp
   tree → `--fleet` table; same-day handoff collision → exit 2.
6. `internal/meta` extraction: adr behavior locked by existing v1 goldens (no diff expected).

`make test` gates everything. The final whole-branch review simulates the acceptance environment:
dry-run adopt plans against the four real target repos, human-reviewed before any `--write` (the
practice that caught v1's D6 legacy-ADR failure pre-live).

## Acceptance (v2 done means)

1. `make test` green; `~/bin/spine` reinstalled; `spine version` prints generation 2.
2. Four live adopts applied — praxis, moo-clone, home-lab-admin, obsidian-ep-vault — each via
   human-reviewed dry-run plan, then `--write`, ending doctor-clean (info-only) with `update` a
   no-op.
3. local-model-evaluation carries `docs/evals/` with the govmomi eval and 8 run records hand-filled
   once from its README/REVIEW.md data; `eval list` tabulates them; D7 clean.
4. `spine handoff latest --fleet ~/Projects/github.com` produces the correct fleet table.
5. ADRs 0006 (stdlib holds at two levels), 0007 (eval seam), 0008 (adopt composes) recorded via
   `spine adr new`.
6. Gen-1 repos (ccq, hbmview, spine) bumped to gen 2 via `spine update --write`.
7. No deepthought changes (the workflow-init shim calls `spine init`, unchanged).

## v3 candidates (explicitly deferred)

Per-repo generation pin (if a second operator or a bad generation ever ships), `handoff list
--fleet`, cobra (if the tree hits three levels or needs persistent flags), machine-level doctor
checks, `/model-eval` skill (next project, drives `docs/evals/`).
