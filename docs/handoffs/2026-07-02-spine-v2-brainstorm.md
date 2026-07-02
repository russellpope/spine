# Handoff Reference — spine v2 brainstorm (2026-07-02)

Seed context for a fresh `superpowers:brainstorming` session on spine v2. spine v1 shipped and
is in daily use (`~/bin/spine`, gen 1). This is a **brainstorm**, not a plan — the candidate set
below is input to explore/refine/cut, not a committed scope.

## What spine is (v1, shipped)

A single Go binary (Go 1.26, stdlib-only) that owns the fleet's unified workflow scaffolding.
Commands: `init` (scaffold a repo from a profile), `update` (regenerate machine-owned files;
dry-run exits 1 with diffs, `--write` applies), `adr` (new/list), `doctor` (read-only health
checks D1–D6, `--json`), `version` (prints the compiled template generation, currently 1).
Templates compile into the binary as a single integer generation. It absorbed the old
deepthought `workflow-init` skill (now a 20-line shim over `~/bin/spine`).

- Repo: `~/Projects/github.com/spine`, main @ `e8d4bf4`, **NOT pushed** (origin =
  `git@github.com:russellpope/spine.git`; no `origin/main` yet).
- Design: `docs/specs/2026-07-01-spine-cli-design.md`; plan: `docs/specs/2026-07-01-spine-cli-plan.md`.
- ADRs 0001–0005 (read these — they constrain v2): stdlib-only (cobra reconsidered only if v2
  nests commands), ownership-split + **choice-vs-default** regeneration rule, `docs/specs`
  absorbs plans, templates-compile-with-one-integer-generation, pre-spine ADRs report D6-info.

## v2 candidate set (explore, don't assume)

1. **`spine adopt`** — retrofit an *existing* repo that predates spine (praxis, ultima, the
   local-model-evaluation dirs) into the workflow without clobbering its real files. The hard
   part is the choice-vs-default rule (ADR 0002) applied to files that already exist and were
   hand-authored — what's a "default" to overwrite vs a user choice to preserve?
2. **`spine handoff list` / `handoff latest`** — the fleet has 56+ handoff files under
   `docs/handoffs/`; there's no way to list/open the latest per repo. Small, high-use.
3. **`spine eval` + a `docs/evals/` convention** — pairs with the next build, the `/model-eval`
   skill (FINDINGS' #1 new-skill finding: ~15 sessions hand-type wire→audit→score→compare→
   remediate→rescore across 8 `local-model-evaluation-*` dirs). spine would own the artifact
   convention; the skill drives the loop. **Decide the seam between them in this brainstorm.**
4. **New profiles** — `swift`, `infra`, `knowledge` (today: go-service, library-cli,
   presentation, py-tool, rust, ui). Which the fleet actually needs, and what each scaffolds.

Likely also worth raising: whether `update` needs a per-repo "generation pin" story as the fleet
spreads; whether `doctor` should grow checks for the new conventions.

## Why this is next

Russell's stated priority after ccq: "spine v2, then model-eval." spine v2 (esp. `spine eval` +
`docs/evals/`) sets up `/model-eval`, so it goes first. Each is its own brainstorm→spec→plan.

## Gotchas (don't rediscover)

- **spine is the fleet upgrade path**: `spine update` dry-run prints diffs + exits 1; `--write`
  applies; a stamped generation **>** the binary's generation is a hard error by design (never
  downgrades). `doctor` exit 0 tolerates info-only findings.
- **Choice-vs-default (ADR 0002)** is the load-bearing rule for anything touching existing files:
  a value equal to its own generation's rendered default is not a user choice → safe to replace;
  anything else is preserved. `adopt` will live or die on getting this right for pre-existing files.
- **Real-file fixtures beat synthetic constants** — the v1 review caught a bug only because an
  hbmview real-file fixture exposed that a "sync-to-reality" task had inverted a correct constant.
- **Final whole-branch review should simulate the acceptance environment** when the next step
  mutates a real repo (that's what caught v1's D6 legacy-ADR failure before the live run).
- Commits stay local everywhere; **push only when Russell says**. praxis's remote is `github`
  (not `origin`); per-repo invariants live in each repo's CLAUDE.md.
- Bash tool runs bash; login shell is fish (quote globs, no `cd` in compounds, `status` is read-only).
- Open Brain hooks fire recall each prompt and demand a Stop writeback — write real outcomes back.

## What shipped just before this (so it's not re-litigated)

`ccq` (transcript-analytics CLI) merged to its own main and pushed **private** to
`github.com/russellpope/ccq`, installed at `~/bin/ccq`. Its usage-meter half was built then
**retired** mid-flight: the par-skills statusline (`~/.claude/statusline.sh`) reads usage from
`api.anthropic.com/api/oauth/usage` with Claude Code's OAuth token — no session key, no
Cloudflare — which obsoleted ccq's claude.ai/Swift-shim meter. The claude.ai-session-key P0 is
resolved (nothing references the leaked key; rotation is now just hygiene).
