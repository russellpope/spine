# Estate-wide offload candidates: work outside the toolkit that software should do

Date: 2026-09-07. Third of three companions (retrospective, CLI gaps, this). Scope widened from the three tools to everything: all 9,955 transcripts (91,846 Bash calls, 9,477 user messages, 5,821 session-opening asks), the atuin shell history (4,733 commands since 2026-05-15), every repo under `~/Projects`, launchd, chezmoi, hooks, and the Obsidian vault. Two read-only agents covered the archive and the repos; the mining was scripted. Nothing was modified. Filed in spine only to keep the trio together; most proposals belong in deepthought.

## The shape of the work

| Project | Sessions | Bash calls |
|---|---|---|
| Codex rollouts (date tree, not parsed for Bash) | 3,919 | |
| praxis | 955 | 15,621 |
| maipipe | 806 | 16,864 |
| ultima-dci-edition | 500 | 8,049 |
| deepthought | 484 | 8,726 |
| OB1 / open-brain | 440 + 248 | 4,564 |
| moo-clone | 416 | 6,130 |
| ladderbench | 402 | 5,456 |
| maikanban | 376 | 6,273 |
| spine | 349 | 5,813 |

The toolkit is a third of the volume. praxis, the game engines, the model-evaluation work, and the notes system are the rest.

**Your own shell is an agent launcher.** Of 4,733 commands, 2,290 launch `claude` or `codex`, using 191 distinct flag combinations; 245 of them inline a brief file. Navigation is another 957. The remainder is brew (204, with 15 typos), `maipipe ui` and `maikanban` (80), btop (35), disk-space hunting in `~/models` (43), model servers (llama-server, ollama, opencode, tuicr: 118), and `yt-dlp` (20 runs, 19 failed). There is almost no hand-done engineering in the history. The work already goes to agents; what you do by hand is dispatch and watch.

**Sessions open with a hand-written brief.** Of 5,821 session-opening messages, the top thirty openers are all role templates: "you are implementing task" 614, "you are reviewing one" 428, "you are an adversarial" 273, "you are a senior" 188, "you are a skeptical" 177, and so on. Roughly 3,000 briefs were composed by a lead agent or by you, one at a time. spine ticket I126 counted 847 of them in the estate; the true number is about three and a half times that.

**Agents outside the toolkit have the same habits** found inside it: 1,197 Python JSON parses of tool output, 752 Python find/replace edits, 377 hand-rolled poll loops, 3,026 wc/grep counting calls, and 6,600 direct `go test` and `cargo test` runs. The CLI-gaps report's fixes apply estate-wide.

## What is already automated

launchd runs the Open Brain servers, a weekday follow-ups refresh (one of two logs shows FAILED), and a moo-clone team monitor every 30 minutes. chezmoi has seven on-change scripts that build and install the agent stack. Claude hooks cover session start, commit staging, cost tracking, and the maipipe gate. `ccq` exists for transcript analytics and was called 566 times from deepthought. There is no crontab, no fish abbreviations, no Hammerspoon or Raycast. The style is bash `run.sh` plus a compiled Go, Rust, or Swift binary plus a launchd plist. Proposals below follow that style.

## Findings by category

**1. Checks that nobody runs.** This is the largest gap and it is not a tool gap. Thirteen repos are one commit ahead with an identical dirty `WORKFLOW.md` from the 2026-08-31 sweep, committed everywhere and pushed nowhere; ladderbench is 101 ahead with 13 worktrees; total unpushed is about 120 commits. The Obsidian vault's git backup stopped on 2026-07-31 and the live vault at `~/Documents/jarvis` is not even the checkout. praxis's `security` GitHub workflow has been red since 2026-08-17 and no handoff diagnosed it. Four of 45 follow-ups runs failed unnoticed. cmux is 370 commits behind its fork origin. Every one of these is a one-line check.

**2. Routines built and never scheduled.** `jarvis`, a Go binary for daily meeting prep into Obsidian, exists and has no launchd entry; `DailyBriefing/` stopped on 2026-06-19. `WeeklyReview/` stopped at ISO week 21. A prompt file titled daily-meeting-prep-task sits in the vault waiting to be pasted into a scheduler. ai_infra_notes relies on a copy-pasted kickoff prompt. The follow-ups job proves the pattern works when it is wired.

**3. Dispatch and launch.** The brief count above, plus 191 launch flag combinations by hand, plus spine already resolving the model for a harness and tier. The pieces exist; the composition does not.

**4. Cost.** Cost, tokens, and spend appear in 21% of all user messages, concentrated in praxis and OB1. A 2026-06-02 praxis session found the cost-tracker hook overcharging cache reads by about 8.6 times. The hook bug is one line; the reconciliation is a nightly job over `ccq`.

**5. Per-repo rituals written in PICKUP files.** local-model-evaluation: a two-commit freeze per run preceded by a manual `git status --ignored` and a hunt for untracked files over 1 MB. ladderbench: conformance and efficiency tables built by hand from verdict YAML although a `bench score` verb exists. vfb: a PII grep that must be empty before first push, and a root-owned ColorSync profile leaked into `/Library` on every probe. moo-clone: you run the signed XCUITests by hand. pi-pack: deferred-minor lines in a progress ledger that need copying into tickets. Worktree garbage across ladderbench, ultima, and moo-clone: 13, 18, and 9.

**6. Shell papercuts.** 15 brew typos, 19 of 20 yt-dlp failures, 43 disk-space hunts, long llama-server argument lists retyped. Small, but daily.

## Ranked candidates

Ordered by hours recovered per hour built. Kind: S = shell script in deepthought, L = launchd job, V = verb in an existing binary, H = hook.

1. **`estate doctor` (S+L).** Per repo: ahead/behind, dirty, unmerged worktrees, last push age; plus vault backup age, last-24h launchd error logs, `gh run list` for user repos, fork staleness. Exit non-zero, print a table, run at 07:00 and from SessionStart. Covers finding 1 entirely. Half a day.
2. **`dispatch` (V in spine, or S).** `dispatch <role> <ticket> [--harness codex|claude] [--tier primary|routine]` renders a brief from a role template plus the ticket, resolves the model via `spine model`, writes the brief file, and prints or runs the launch line. Retires the 3,000 hand-written briefs and the 191 flag combinations. I126 is the ticket; this is its scope. Two days.
3. **`run <profile>` launcher (S).** Named profiles over the flag combinations you actually use: `run opus-low`, `run fable-high`, `run sol-xhigh`, each reading the brief from stdin or a path. An afternoon; also the front end of item 2.
4. **Schedule what exists (L).** `com.jarvis.daily` weekdays 06:00; a Monday `weekly-review` job that scaffolds the week file from a template with estate `git log` and `ccq` stats pre-filled; a `claude -p` wrapper for each kickoff prompt file, modelled on `open-brain/followups/run.sh`. An afternoon.
5. **Vault backup (L+H).** Init git in the live vault, point origin at obsidian-ep-vault, hourly add-commit-push, and a doctor assertion that the last commit is under 24 hours old. An hour, and it closes a silent data-loss risk.
6. **Cost reconcile (H+L).** Fix the cache-read rate in the cost-tracker hook, then a nightly `ccq cost` roll-up into a gitignored report with a variance threshold. Two hours.
7. **`wt-prune` (S).** List worktrees whose branch is merged, dry-run by default, remove worktree and branch together. Shared with the toolkit's batch-close verb. Two hours.
8. **`freeze-run` (S, local-model-evaluation).** Ignore audit, refuse on untracked over 1 MB, make both commits. Two hours.
9. **`bench table` (V, ladderbench).** Tabulate verdict YAML into the conformance and efficiency tables the handoff asks for by hand. Half a day.
10. **vfb pre-push hook and ColorSync cleanup (H+S).** The PII grep as a pre-push hook; a sudo-able cleanup with a count assertion that doubles as the test fixture. Two hours.
11. **moo-clone signed test runner (S).** Makefile target that runs the signed XCUITest bundle and drops the xcresult export where the handoff wants it. Visual QA stays human. Half a day.
12. **Ledger triage extractor (S).** Pull "minor (deferred)" lines out of any progress ledger into draft tickets. Belongs beside the spine ticket family. Two hours.
13. **Shell abbreviations (fish).** `bu` for update-and-upgrade, a `yt` wrapper with the working flags, a `models-du` alias, a `llama` function with the standard arguments. Fifteen minutes, and it removes the most frequent typos.

## Not worth building

- A generic "commit and push" wrapper. The staging rule is explicit paths only, which is judgement; the push decision is yours by policy. Item 1 makes the backlog visible, which is the actual gap.
- A YAML or source editing CLI for the 752 Python find/replace calls. That is auto mode's Bash instruction, not a missing tool.
- Anything on top of cmux or herdr. Both are pure upstream clones with zero user commits; you build around them, and the customisation surface you use is two resume commands.
- Rebuilding ccq, jarvis, or the follow-ups job. They exist. Schedule and check them.

## Order

- **This week:** items 1, 5, 13, and the cost-tracker one-liner from item 6. All are checks or aliases; none needs a design.
- **Next:** items 2 and 3 together, since the launcher is the dispatch verb's output. Grill I126 first; it already exists.
- **Then:** items 4, 7, 8, 9, 10, 11, 12 as single-afternoon scripts in the repos they serve.
