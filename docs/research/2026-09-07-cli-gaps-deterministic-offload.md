# CLI gaps: what agents do by hand that a verb should do

Date: 2026-09-07. Companion to `2026-09-07-toolkit-retrospective-spine-maipipe-maikanban.md`. Method: a script over 1,593 transcripts (28,971 Bash calls in spine, maipipe and maikanban sessions) clustered by command family and normalised chain; three parallel read-only audits of the skill library, the 182 handoffs plus PICKUP files, and the live `--help` surface of spine, maipipe, maikanban, herdr, cmux and vfb. Nothing was modified.

## Headline

The model is doing three kinds of deterministic work at scale: editing ticket frontmatter with sed and Python, hand-rolling poll loops, and parsing JSON that the tools emit in a shape nobody can read at a glance. spine has no ticket verb, its cursor has no `--json`, and maikanban has no non-TUI surface at all. Most of the proposed verbs are thin wrappers over calls that already exist, in a fixed order the handoffs already spell out.

## The evidence

| What agents did by hand | Bash calls | Where the recipe is written down |
|---|---|---|
| sed/awk/python edits to `docs/issues/*.md` frontmatter (status alone: 161) | 395 | claude-team, wayfinder, to-tickets, model-eval skills; 83 handoffs |
| "Next free id: I1xx" computed and carried in the handoff | 46 handoffs | 2 id collisions on 2026-08-20 and 2026-08-21 |
| Poll loops with sleep; 209 wait for a subagent report file to exist and stop growing, 103 for a maipipe run or the daemon | 645 | claude-team SKILL.md L122 |
| Python parsing a tool's JSON: `maipipe gate` 247 (71 identical one-liners), `herdr agent get` 123 | 720 | |
| Python as a find/replace editor on source and docs | 822 | |
| `uptime` before deciding to run a lane | 279 | |
| pgrep/ps hunting daemons, test seams, orphans | 356 | I174, orphan counts in 4 handoffs |
| cmux `/exit`, enter, sleep, relaunch-claude sequences | 136 | claude-team L100-114 |
| Heredoc appends to `progress.md` | 285 | required by `cursor tick implement` |
| shasum or version checks on binaries | 119 | two-binary-path memory note |
| Session wrap (ledger, handoff, PICKUP, commit, lane, push) | 44 handoffs | spine PICKUP.md L28-32 |
| Batch teardown (exit lead, close workspace, remove worktree, delete branch, kill vfb) | 58 handoffs | maikanban PICKUP.md L41-43 |
| Fleet WORKFLOW.md sweep, 4 steps × 8 to 17 repos | 6 sweeps | 3 consecutive spine handoffs |

Rough read-only share of all Bash calls is 44%, which is expected under auto mode. The numbers above are the write and wait side.

## Proposed verbs, ranked by calls removed per line of code

**1. `spine ticket` family.** spine owns the schema and already validates it in doctor D13, yet nothing writes it. `spine ticket new "<title>" [--blocked-by ..] [--label ..] [--parent ..]` allocates max+1 under a lock and copies the template. `spine ticket close <id> --commits <sha..>` sets status, records commits, drops `workspace:`, appends the resolution stub, and writes the progress ledger line that `cursor tick implement` demands, in that order. `spine ticket set <id> k=v [--unset k]` covers claim, batch, assignee, workspace. `spine ticket frontier [--batch X] --json` answers "what can start now", which three skills compute by hand and maikanban computes on every frame. `spine ticket spec <id>` resolves the design/plan pair that spec-review and review-ticket look up by globbing. This one family removes the 395 frontmatter edits, the 46 id carries, the two collisions, and the off-vocabulary statuses the retrospective found.

**2. `--json` on the surfaces agents actually read.** `spine cursor` has 99 references and no `--json`; its `stages:` line is the de facto machine contract. `spine audit routing|stages` has 59 references and prints only a table. Add both. In maipipe, `gate` already emits one JSON line but 247 calls pipe it through Python for a five-word summary; add `gate --brief` that prints `verdict | stage:disposition ...`. In herdr, `agent get` returns a nested envelope that 123 calls unwrap for two fields; a `herdr agent status <name>` one-liner. These are output flags, not features, and they remove 720 Python parses.

**3. One await primitive.** Six wait verbs exist across tools with six different contracts (`maipipe run --wait`, `maipipe watch`, `gate --wait`, `herdr agent wait`, `herdr pane wait-output`, `vfb wait-for`) and none covers the hottest loop: a subagent's report file appearing and going size-stable. Ship `wait-report <path> [--slot <agent>] [--timeout]` in deepthought's `skills/lib` first (shell, an afternoon), exposing the dead-worker check that claude-team L127 describes. Then make `herdr agent wait` reliable enough that the memory note's retry dance is unnecessary. This retires the 645 poll loops, which are also the most fragile code the model writes.

**4. `maipipe runs` and `maipipe logs --run <id> --stage <name>`.** Both are already open tickets (I185, I098). Eleven handoffs re-paste the sqlite3 or `nc -U` recipe for reading a failed stage. `runs` also unblocks the deploy-drift check below.

**5. `maikanban --print [--repo X] [--json]`.** The board recomputes columns from files on every frame, so a headless projection is cheap. Today the only way to ask "what is blocked and by what" without a TTY is to reimplement the lane logic, which is what the skills do. This also gives spine's `ticket frontier` and the board one shared answer to compare in tests.

**6. Deploy and drift.** `make install` in spine writes one path while the SessionStart hook runs the other; 24 handoffs restate the copy step and the stale binary went unnoticed for six weeks. Make `install` write both and print both `build:` lines. In maipipe, `make deploy` already sequences the restart; what is missing is the read-only `maipipe doctor --deploy-drift` that reports daemon pid, schema version and binary age against HEAD so a session knows a redeploy is owed rather than discovering it through a ticked `deploy[x]` on a stale daemon.

**7. `spine update --fleet <root> --write`.** The sweep is four steps per repo across 8 to 17 repos, logged to /tmp and summarised by hand into a pre/post exit table in three consecutive handoffs. spine already has `--fleet` on `handoff latest` and `yield`. The only judgement step, classifying residual skips as local edits versus missed stock, can be printed as a column.

**8. Batch lifecycle: `spine batch open <ids..>` and `spine batch close <slug>`.** Open stamps assignee and batch, creates the worktree with the ADR 0008 naming rule, seeds the allowlist, and writes the kickoff brief skeleton. Close does the five-step teardown and prints the workspace-close commands the permission classifier will not let an agent run. Both are fully specified in maikanban's PICKUP.md and the 2026-09-02 handoff; the failure modes in the herdr memory note are exactly what a verb encodes once. Recorded failures this month: prompt stalls, `/exit` typed-not-submitted, branches surviving worktree removal, 354 leaked vfb sessions purged by hand.

**9. `spine model preflight <harness> <tier>`.** The same 12-line exit-code triage block is copied in eight places and has its own regression test purely to keep the copies in sync. Return the remedy text and one exit code.

**10. Load and process hygiene in maipipe.** 279 `uptime` checks precede lane runs because agents gate on load by hand; `admission_load_threshold` exists but is disarmed by default (docs/cli.md line 59). Arm it, or add `run --when-quiet`. Promote the orphan sweep that lives in `tests/orphan_sweep.rs` to `maipipe ps --orphans [--kill]`, since 356 pgrep calls are doing that job.

## Do not build these

- **Another wait or handoff verb.** `maipipe watch` (both forms), `herdr agent wait`, `spine checkpoint`, `spine yield`, `spine adopt`, and cmux `events` have zero references outside their own repos. The gap is discoverability and reliability, not absence. Point the skills at them and fix `agent wait`.
- **A YAML editing CLI for source files.** The 822 Python find/replace calls are mostly on Rust and Go source under auto mode's "use Bash" instruction. That is an editing-tool policy question, not a CLI gap. The docs subset is covered by the ticket family.
- **A generic vfb wrapper yet.** maipipe already has 15 per-ticket `prove-*-vfb.sh` scripts totalling 15,887 lines. Extract the common launch, snapshot, kill, scratch-DB prologue into one script before designing a verb; the shape will fall out of the extraction.

## Placement rule

Put the verb where the schema owner is. spine owns tickets, cursor, handoffs, and the fleet. maipipe owns runs, the daemon, and load. maikanban owns the projection. Orchestration glue (await, preflight, workspace ensure, allowlist seed, agent restart) belongs in deepthought's `skills/lib` as shell, because it changes with the frontends and should not gate a Go release. Every new verb emits JSON on request and a documented exit code, and the skills stop reading tables by eye.

## Suggested order

- **Wave 1, days:** `wait-report` script; `spine cursor --json`; `maipipe gate --brief`; Makefile install to both paths; arm the admission threshold.
- **Wave 2, one to two weeks:** `spine ticket` family; `maipipe runs` and `logs`; `maikanban --print`; `spine model preflight`.
- **Wave 3, after wave 2 settles:** `spine batch open/close`; `spine update --fleet`; `maipipe doctor --deploy-drift`; `maipipe ps --orphans`.

Live finding from the inventory: `herdr agent list` currently returns `protocol_mismatch` (client 22, server 20). The herdr server needs a restart before any agent verb works.
