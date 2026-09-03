# fusion-harness borrow hitlist

date: 2026-08-26
status: dispositioned 2026-09-02 (see Disposition at the end; spine tickets I126, I127 cut)
source: survey of IndyDevDan's fusion-harness (clone at `~/Projects/github.com/fusion-harness`, MIT) against maipipe, spine, and maikanban, 2026-08-24.

## Context and verdict

fusion-harness is a ~25-file TypeScript Pi extension that runs 2 to 5 frontier models against one prompt and lets them compare (`/fh-opinion`), debate without a judge (`/fh-debate`), or merge through a sole-writer FUSION agent (`/fh-fusion`). Coordination is per-session and ephemeral.

It does not compete with the estate. Its whole product lives in the exploration and advisory layer that our floor already treats as non-authoritative, and its collaborate/DAG execution is a single-session toy next to maipipe's relay and team machinery. The verdict from the survey: use it alongside (its opinion/debate modes fit the `grill` stage and the eval Compare stage), and borrow the five mechanisms below.

Standing guard for every item: acceptance stays a deterministic function of run facts. Nothing borrowed here may put model judgment in the floor. Model output stays advisory.

## The hitlist

### 1. The `pi --mode json -p` clean-room child driver

Source: `extensions/fusion-harness/modules/child-runner.ts` (~280 lines).

The highest-value steal. A complete recipe for driving pi programmatically: clean-room spawn (`--no-skills --no-extensions --no-context-files`), line-buffered JSON event parsing with cache-aware token accounting (input + cacheRead + cacheWrite, because a cold cache bills the whole prompt as a write), thinking-stream capture, session fork/resume precedence, and close-aware SIGTERM to SIGKILL process-group escalation.

Targets: maipipe `team --member` specs for local-model workers (pi is already the weak-local driver in the routing table), and automating the `/model-eval` loop instead of hand-driving pi per run.

### 2. The atomic writer lease

Source: `extensions/fusion-harness/modules/writer-lease.ts` (~60 lines).

A hashed canonical-CWD lock file in `/tmp`: `O_EXCL` create, owner recorded as pid + uuid + label, pid-liveness check with EPERM counted as alive, stale-lock reclaim, owner-verified release. Cheap and complete.

Today the estate enforces single-writer three ways: worktree-per-run (maipipe, mechanical), spine's sole-writer cursor rule (mechanical), and prose in dispatch briefs ("MUST NOT EDIT in this phase", prompt-enforced). The briefs are the weak link. This pattern makes that promise mechanical for any two agents sharing a checkout: claude-team leads, relay builders, and maikanban's ad-hoc per-file `.lock` files (three of which were sitting stale and untracked in maikanban's `docs/issues/` as of 2026-08-24).

Target: probably a small shared convention rather than shared code, given three languages. Same lock path scheme and JSON payload, implemented twice (Rust for maipipe/maikanban, Go if spine ever needs it).

### 3. Prompts as versioned template files

Source: `extensions/fusion-harness/prompts/` — every agent contract is a `SYSTEM_PROMPT_*.md` / `USER_PROMPT_*.md` file; the code interpolates but never contains prose. Edit files, not code.

The estate's equivalent artifact is ~600 hand-written one-off dispatch briefs under `.superpowers/sdd/` across repos, plus frozen contract text as Rust consts. spine already ships document templates via `go:embed` with generation migrations; shipping dispatch-brief templates the same way would turn our most-duplicated, least-governed artifact into a managed one.

Target: spine `templates/` gains a `dispatch/` family (worker brief, reviewer brief, relay builder brief), stamped and regenerated like every other managed file.

### 4. ACK-with-hash fan-in

Source: `/fh-fusion` context sync (see `extensions/fusion-harness/modules/cmd-fusion.ts` and README "Fusion context synchronization"). After the merge, every slot receives the complete result in a no-tools turn and must reply `ACK FUSION <run-id>`; `acks/<slot>.md` and `summary.json` record status plus a common SHA-256. Failed ACKs mark the run context-sync-incomplete instead of pretending.

This is evidence-provenance thinking applied to fan-in: proof each worker actually received the merged plan, not an assumption. progress.md already cites artifacts by path and SHA-256; this extends the same discipline to dispatch.

Target: claude-team lead protocol. When a lead distributes a merged plan to workers, require a hash ACK per worker and record it in the batch evidence dir.

### 5. The install playbook

Source: repo root — README install section, `.claude/commands/install.md`, `npm test` (34 deterministic tests, zero paid calls), plus a separate demo playground repo with canned prompts.

The shape worth copying: clone, one dependency command, `.env` from example, a deterministic zero-cost verification lane, a live launch check, and an agentic `/install` command that walks all of it. spine's gate positive controls and maikanban's `cargo test` already qualify as the verification lane; what is missing is the wrapper. Each estate repo should have a `/install`-style skill that checks toolchain, builds, runs the free lane, and reports.

This item feeds the redistribution effort directly (readiness order spine, then maikanban, then maipipe; maipipe distribution is I029, open). spine's README landed 2026-08-24; LICENSE is still unchosen.

## Deliberately not borrowing

- The YAML model stack. spine's tier table with history-tracked inherited-vs-override defaults is strictly richer than fusion-harness's flat slot list.
- The collaborate DAG executor. Session-scoped, no durability, no crash recovery. maipipe relay/team already does this properly.
- The TUI layer. maikanban's ratatui stack is ahead of fusion-harness's grid primitives.
- Opinion/debate/fusion as products. Use fusion-harness itself for that (it is installed and working); reimplementing it inside the estate buys nothing until a concrete eval-Compare automation ticket exists.

## Next steps if ratified

Cut one ticket per adopted item in the owning repo: pi child driver (maipipe), writer-lease convention (maipipe + maikanban), dispatch templates (spine), ACK fan-in (claude-team skill), install skills (one per repo). Items 1 and 3 first; they compound (templated briefs feed the child driver).

## Disposition (2026-09-02, ledger-burndown effort, autonomous grill)

Each item was re-checked against the estate as it stands today. The standing
guard held throughout: nothing below puts model judgment in the acceptance
floor.

| # | Item | Owner repo | Verdict | Evidence |
|---|---|---|---|---|
| 1 | pi child driver | maipipe | survives; cut there, not from spine | pi remains the weak-local driver in every fleet routing table; `/model-eval` is still hand-driven per run. Not spine's to write. |
| 2 | writer lease | maipipe + maikanban | survives as a convention, lower urgency | the three stale `.lock` files cited on 2026-08-24 are gone from maikanban's `docs/issues/` (0 on 2026-09-02); the brief-prose weak link still exists. Cut in the Rust repos when a shared checkout is next scheduled. |
| 3 | dispatch templates | spine | survives; **I126** cut | 847 hand-written briefs across spine (286), maikanban (243), maipipe (318). spine already ships `go:embed` templates with generation migrations. |
| 4 | ACK-with-hash fan-in | claude-team skill | survives; cut with the skill's next revision | no spine change; the lead protocol lives in `~/.claude/skills/claude-team`. |
| 5 | install playbook | spine first | survives; **I127** cut | no `/install`-style skill exists and there is no LICENSE file, which blocks redistribution regardless of the wrapper. |

Owner follow-ups outside spine: items 1, 2, and 4 need tickets in maipipe,
maikanban, and the claude-team skill respectively. This note is adopted as
the effort's research record.
