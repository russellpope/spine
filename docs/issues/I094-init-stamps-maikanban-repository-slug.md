---
id: I094
title: "spine init / doctor: stamp and check `maikanban.repositorySlug` when scaffolding docs/issues/ (new repos currently arrive unconfigured and break maikanban fleet discovery)"
severity: low
status: fixed
affects: []
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

## Problem

Filed 2026-08-19 from maikanban (`docs/issues/I030-…`, ADR 0008). maikanban discovers its fleet by
scanning the projects root for `docs/issues/` and requires every discovered repository to carry
`git config maikanban.repositorySlug owner/repo` (maikanban ADR 0007). `spine init` is what
creates `docs/issues/` (via the `workflow-init` skill, a thin shim), and it never sets the slug —
so every freshly scaffolded repo is unconfigured until the owner remembers. Observed 2026-08-19:
`pi-pack` (no remote, empty ledger) made `maikanban` fail to open from `maipipe`. maikanban I030
turns that into a per-repo exclusion; this ticket stops the exclusion from happening in the
first place.

## Fix

1. `spine init` (`cmd/spine/main.go:cmdInit` → `internal/scaffold.Init`): after creating
   `docs/issues/`, if the dir is a Git repo and `maikanban.repositorySlug` is unset, set it to
   `<owner>/<basename>` where owner is parsed from `origin` (`github.com[:/]<owner>/…`), else
   from a new `--owner` flag, else from the global `maikanban.defaultOwner` git config if set;
   if no owner can be determined, print `note: set git config maikanban.repositorySlug
   owner/repo (maikanban fleet identity)` and exit 0 (never fail init over it). Report
   `create: git config maikanban.repositorySlug <value>` in the created list; never overwrite
   an existing value. Honour the slug grammar from maikanban ADR 0007 (1–100 ASCII bytes per
   component, alphanumeric ends, `._-` inside).
2. `spine doctor`: report a missing/malformed slug on a repo that has `docs/issues/` as a warning
   with the exact command.
3. `workflow-init` SKILL.md (deepthought repo `skills/workflow-init/SKILL.md`): one-line gotcha
   — "`spine init` stamps `maikanban.repositorySlug`; if it printed the `note:` line, run the
   command before opening maikanban." Commit in deepthought with an explicit path.
4. Tests: scaffold into a temp repo with an `origin` → slug set; without origin and no
   `--owner` → note printed, exit 0, no config; pre-existing slug untouched; `doctor` warning.

## Acceptance criteria

- [x] `spine init` in a temp git repo with `origin=git@github.com:acme/x.git` sets
      `maikanban.repositorySlug=acme/x` and lists it under `create:`; re-running does not change it
- [x] No origin / no `--owner` / no `maikanban.defaultOwner` → init exits 0 and prints the
      `note:` line; `--owner acme` sets `acme/<basename>`
- [x] `spine doctor` warns on a `docs/issues/` repo with missing or malformed slug, with the command
- [~] `workflow-init` SKILL.md note committed (deepthought) — **out of scope**, ruled by the
      controller 2026-08-20: it is a side effect in another repository and stays with the owner.
      Full lane green; shellcheck/deny stages unchanged.

## Blocked by

- None. Related: maikanban I030 / ADR 0007 / ADR 0008.

## Evidence

Fixed 2026-08-20 on `i094-maikanban-slug`. Item 3 (the `workflow-init` SKILL.md note in
deepthought) was ruled **out of scope** by the controller and is left to the owner — nothing
outside this repository was touched.

- `internal/scaffold/slug.go`: `StampSlug` (slug from `origin`, else `--owner`, else global
  `maikanban.defaultOwner`; never overwrites; silent on non-git dirs and on failure),
  `ValidSlug`/`ValidOwner` (ADR 0007 grammar), `SlugNote`, `SlugRemedy`, `SlugKey`.
- **Deviation from Fix item 1's wording, on review 2026-08-20 (round 1).** Item 1 says the slug
  is `<owner>/<basename>` with only the owner taken from `origin`. Implemented instead: when
  `origin` resolves, **both** halves come from the remote (trailing `.git` stripped); the
  basename is used only on the `--owner` and `maikanban.defaultOwner` paths, where no remote is
  available to say otherwise. Reason: a clone or worktree checked out under a different
  directory name (`spine-wt-i094` for `russellpope/spine`) would otherwise be stamped
  permanently with an identity maikanban cannot resolve. This is a deliberate correction, not
  an implementation slip; `TestInitStampsSlugFromOrigin` uses a directory name that differs
  from the remote's repo name so the basename behaviour cannot pass.
- `cmd/spine/main.go`: `spine init --owner`; the stamp reports as
  `create: git config maikanban.repositorySlug <value>`, otherwise the `note:` line, exit 0.
- `internal/doctor/slug.go`: new **D12** (warn) — a repo carrying `docs/issues/` with a
  missing or malformed slug, message naming `git config maikanban.repositorySlug owner/repo`.
- `go vet ./...` clean; `make test` green (all 18 packages `ok`, 2026-08-20).
- Round-1 review fixes: port-bearing ssh remotes (`ssh://git@github.com:22/acme/x.git`) no
  longer parse the port as the owner; an invalid `--owner` is reported on stderr (exit still 0);
  `slugCheck` moved to its own file, matching the D11 precedent.
- Negative controls: unregistering `slugCheck` → `TestD12WarnsOnMissingSlug` and
  `TestD12WarnsOnMalformedSlug` FAIL; neutering `StampSlug` to a no-op →
  `TestInitStampsSlugFromOrigin`, `TestInitNoOwnerPrintsNote`,
  `TestInitOwnerFlagStampsBasename`, `TestInitDefaultOwnerConfigStamps` FAIL.
