# Toolkit retrospective: spine, maipipe, maikanban

Date: 2026-09-07. Method: seven parallel read-only audits (three over the conversation archive, three over each repo's ledger, handoffs, ADRs and git history, one over the seams between the tools), followed by hand spot-checks of the three most consequential claims. Nothing was modified. Evidence pointers are ticket ids, file paths, or handoff dates.

## Headline

The three tools work when their gates are actually invoked; the credible closures (I125, I128, I221, I073 to I075) prove that. The cost is that most of the process is either unenforced or enforced in the wrong place. The declared mandatory gates are inert, the one gate that really blocks a session is a stop hook the workflow does not name, the ledger lags what is merged, and the owner is the serial bottleneck for deploy, push, and every design ruling. Fix-to-feature commit ratios of 1.7:1 (spine), 2.1:1 (maipipe) and 3.4:1 (maikanban) since July say the toolkit is spending more on repairing itself than on new capability.

## Cross-cutting findings

**1. Declared gates are not enforced; enforced gates are not declared.**
WORKFLOW.md line 7 says `gates: [grill, verify]`; line 36 and CLAUDE.md say PRD, spec-review, verify. Neither list is read by code: `internal/stages/stages.go:20-59` derives evidence only for prd, issues and implement and states that every other stage "can never block". The `gates`, `functional_harness` and `reviewers` keys are render-only. The only thing that actually stops a session is maipipe's Stop-hook claim gate, which is absent from WORKFLOW.md's vocabulary. Consequences observed: `deploy[x]` ticked while the daemon predates both September merges (maipipe handoff 2026-09-02), `functional-test[x]` ticked with "No VFB command ran" (maikanban), zero tickets in any repo carrying a review record, and retroactive PRDs clustering on the estate-wide model remaps the process was built to catch (spine specs 2026-09-02, maipipe habitability retro 2026-08-25).

**2. The cursor is a tax on every session.**
The cursor lives in two homes (gitignored `progress.md` and the newest handoff), the handoff stage cannot be ticked without invalidating the handoff that describes it, `tick implement` needs a ledger line that only exists after the tick, and a merged branch's newer handoff flips derivation to blocking ("happened twice today", spine 2026-08-22). Fifteen of 59 spine handoffs re-teach the sole-writer rule; 34 embed the same dormant 2026-08-19 checkpoint. Half of the notes in my persistent memory for this project are spine gotchas. The ticket chain I013 through I125 shows the "ticked-missing" hint re-scoped four times before evidence-derivation landed for one stage. The design problem underneath: the cursor is per-effort while tickets are per-item, so an all-`[x]` cursor says little about any given ticket.

**3. The ledger is not the truth.**
maipipe: I168, I170 to I178 are merged on main and still open; 19 `fixed` tickets have zero ticked acceptance boxes; two files share id I104; statuses `closed` and `prepared` and severity `medium` are off-vocabulary. maikanban: I058, I061, I062 were verified merge-ready on 2026-09-05 and sit 22 commits ahead of main, unmerged, with PICKUP.md describing a run from 2026-09-02; `done` on two tickets renders as Archive. spine: an id collision on 2026-08-20 forced a renumber because untracked tickets are invisible to worktrees. Neither `spine doctor` nor the board flags any of this. The schema is byte-identical across repos, but it is parsed four ways (three parsers inside spine, one in maikanban) with divergent rules for SHA length, quoting, YAML lists and comments; spine never parses `affects`, `blocked-by`, `labels` or `parent`, so the dependency graph has no upstream validator.

**4. The gate and the flakes are the largest single drag.**
maipipe's stop hook cannot await an in-flight run (I101, open since 2026-07-27) and re-blocks every turn; the 2026-08-14 transcript shows nine identical hook nags in one red state and the owner saying "you're just repeating the same thing over". Every commit, docs-only included, owes a 10 to 15 minute full lane (34 of 84 maipipe handoffs, 30 of 59 spine handoffs). Fourteen maipipe flake tickets (I079 through I227) and nine maikanban ones (I025 through I076) come from wall-clock budgets and fixture daemons on a shared host, culminating in an estate-wide ban on synthetic load. Lane red does not reliably mean code is wrong, so agents learn to re-run instead of read.

**5. The owner is the serial bottleneck.**
Daemon redeploy is owner-gated and lags every merge (20 of 84 handoffs). Pushes back up across 11 to 17 repos; 14 repos still hold uncommitted WORKFLOW.md refreshes from the I128 sweep. LICENSE (I127), the I112 axis ruling, the I208 shims predicate, the blocked-lane weights (maikanban I038), the SCA security drafts (un-grilled since 2026-08-06) and the 11-ticket I104 roadmap (untouched since 2026-08-05) all wait on one person. When the owner is absent, grills are self-answered and the assumptions are never challenged later (spine handoffs 39 to 41).

**6. Integration is thinner than the docs suggest.**
spine to maipipe is a one-way config pipe (the gate-pack region), deployed in one of three repos; no Rust gate pack exists. spine never reads run facts, so `verify` and `functional-test` are unconnected to any maipipe run. "Handoff" has forked into spine's four-section file and maipipe's three-section daemon record. `docs/harness-interface.md` is byte-identical everywhere and unchanged since 2026-07-01; it describes a generator/evaluator loop nothing implements, while the real functional-test surface is 15,887 lines of maipipe vfb scripts, a 70-line maikanban convention, and nothing in spine. `docs/remediation/` holds only templates in all three repos; the round loop has never run. maikanban is the integration hub (1,503 lines of maipipe client, spine-region guards) yet has no doctor, no hook, and a free-form status field.

**7. Model-id changes are fleet incidents, and ADRs churn where coupling is highest.**
I035, I063, I128 and I130 each followed a remap; the mirror-row plus defaults plus generation-lock mechanism is the recurring fragile seam and there was no rollout signal until D18. Three of spine's last thirteen ADRs were reversed within weeks, all in the model-table and gate-pack programmes.

**8. Monoliths and debris.**
`daemon.rs` 24.9k lines, `ui/render.rs` 17.1k, `tests/api_seam.rs` 22.2k in maipipe; `tests/app_loop.rs` 22.9k and `fleet.rs` 5.8k in maikanban. maipipe carries 22 worktrees, 15 of them post-merge, kept "as evidence" with no retention rule. The two Rust repos have zero TODO markers: debt lives entirely in the ledger, which is fine only if the ledger is trusted (see finding 3).

## Recommendations to retrofit now

Ordered by leverage per effort.

1. **Make stage state derivable, not ticked.** I125 already did this for implement. Extend it: `verify` derives from a `docs/reviews/<id>.md` or `review:` frontmatter, `functional-test` from a maipipe run fact or vfb proof path, `deploy` from a binary version match, `handoff` from the newest handoff's mtime. Stages with no derivable artifact leave the cursor. Then drop the embedded snapshot (revisit ADR 0014) and the two-homes problem, the handoff-tick paradox, and the ledger-line ordering all disappear.
2. **One ticket parser, exported as JSON.** Collapse spine's three parsers into one package, add `spine tickets --json`, have maikanban consume it or vendor the same grammar with a conformance fixture shared by both test suites. Close the status and severity vocabulary in doctor, add id-collision and "commits on main reference an open ticket" checks, and reserve ids through the binary.
3. **Make the stop hook stateful and awaitable.** Dedupe registration, remember the last run id it nagged about, add an await verb (I101), and path-filter docs-only commits out of the full lane.
4. **Ban wall-clock test budgets.** Replace them with relative baselines or opt-in `#[ignore]` behind an env var; give fixture daemons per-test ports and tmpdirs. This closes most of 23 flake tickets at the root.
5. **Reconcile the gate declaration.** Either enforce `gates:` or delete line 7. Per-ticket review records become the verify evidence; the effort-level cursor stops standing in for per-ticket review.
6. **Automate the owner-gated mechanics.** `spine deploy` installs to both paths and asserts the same build line; maipipe daemon restart becomes a lane stage with arming preserved across restarts; `cmux.json` points at the installed binary, not `target/debug`.
7. **Prune by rule.** Closure deletes merged worktrees; a doctor check flags a candidate branch ahead of main whose tickets are still open.
8. **Turn every memory gotcha into an error message.** Each spine rule I carry in persistent memory should be a refusal that names the recovery command.

## Principles for a new system, if one is built

- State is a predicate over artifacts. No cursor file, no ticks, no snapshots.
- One schema, one parser, one binary that exports JSON; board and CI are consumers.
- Ticket is the unit; an effort is a query over tickets, not a separate object.
- Gates run out of band, are idempotent, and can be awaited; hooks carry state.
- Owner decisions are a queue on the board, visible and batched, not prose in handoffs.
- Model routing is data in one place with fleet rollout as a first-class, dry-runnable command.
- Delete process that has never run: remediation rounds, the harness-interface loop, checkpoints.
- Tests never assert wall-clock on a shared host.

## Decisions only the owner can make

LICENSE (I127); I112 openweights axis; I208 shims predicate; merge or discard maikanban I058/I061/I062; blocked-lane weights (I038); grill the SCA drafts; I195 root cause; redeploy the maipipe daemon; whether to revisit ADR 0014.
