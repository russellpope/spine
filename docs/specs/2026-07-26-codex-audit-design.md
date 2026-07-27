# Codex Audit — design

Date: 2026-07-26. Grilled 2026-07-25/26 (codex-audit grill; decisions ratified
one-by-one). Continues the decision sequence of
`2026-07-24-flavor-model-table-design.md` (D13–D19); decisions here are
D20–D28. Companion plan: `2026-07-26-codex-audit-plan.md` (via /to-tickets).

Resolves: I009 (no codex transcript reader), I008 (cross-build ticket-id
collision), plus the two edges D15 explicitly deferred to this effort.
Glossary: CONTEXT.md "Audit evidence" section (added 2026-07-26). ADR trail:
ADR 0012 (record-wins fallback reading) written during the grill.

## Problem Statement

`spine audit routing` is the verify-stage enforcement layer for the routing
contract ("artifacts declare tiers; actual models are verified from
transcripts, every build"). It reads only the claude harness's transcripts.
Every codex-run ticket — herdr teams (M4b) and cmux teams (M4a) alike —
degrades to `no-transcript`: the gate exits 0 while verifying nothing, and
silent tier descent is undetectable exactly where the estate now runs most of
its builds. Live consequence: moo-clone reports 28 tickets `no-transcript`,
and M4b's merge gate (moo-clone I028) cannot go green on any codex-run
ticket.

Two adjacent defects compound it. Ticket-id tokens collide across builds and
repos (ticket ids restart at I001 per repo; praxis and maipipe each have
their own I024), which has already produced a FALSE BLOCKING silent-descent
verdict from a shared controller dir — the worst failure class this tool has,
because records cannot excuse descent by design (I008). And pre-convention
tickets emit permanent `unannotated` noise that trains operators to ignore
the column.

## Solution

Teach the audit to read codex transcripts as a first-class evidence source,
with attribution rules strict enough that the audit never manufactures a
confident wrong verdict: dispatch records and spawned-thread actuals where
they exist, a tightly discriminated worker-session scan where they don't,
guardians and orchestrators structurally excluded, and everything
repo-scoped so cross-repo token collisions are impossible by construction.
Where evidence exists but fails attribution, say so honestly with a new
verdict rather than pretending nothing was found. Fix the claude-side
collision class with repo qualification and operator flags. Give
pre-convention tickets an explicit exemption. After this lands, `spine audit
routing` on moo-clone judges I024 `match`, judges M4a honestly, and the I028
merge gate has teeth on codex builds.

## User Stories

1. As an operator running the verify stage, I want codex-run tickets judged
   from their actual transcripts, so that the routing gate verifies the
   builds I actually run instead of exiting 0 vacuously.
2. As an operator, I want the audit to find codex sessions automatically
   (CODEX_HOME respected, sensible default), so that a plain `spine audit
   routing` is sufficient on any machine.
3. As an operator, I want a `--codex-sessions` override mirroring
   `--transcripts`, so that I can point the audit at an archived or copied
   session dir.
4. As an operator, I want repo scoping to be automatic (cwd or known
   commit), so that praxis's I024 can never contaminate moo-clone's audit.
5. As an operator auditing a repo whose team ran in throwaway worktrees, I
   want sessions tied to the repo by git commit identity, so that
   /private/tmp worktree builds are still visible.
6. As an operator, I want the lead's orchestration-tier model excluded from
   ticket evidence, so that a sol lead mentioning I024 ninety-six times does
   not judge a routine ticket escalated.
7. As an operator, I want guardian auto-review threads structurally
   excluded, so that the synthetic `codex-auto-review` model never scores a
   ticket as routed to a review model.
8. As an operator, I want per-turn model evidence (not per-file), so that a
   session that switched models mid-run is judged on what each turn actually
   used.
9. As an operator, I want a `unattributed-transcript` verdict distinct from
   `no-transcript`, so that "material exists but none was attributable" sends
   me to the right diagnosis instead of a false "nothing found".
10. As an operator, I want every judged codex verdict's detail to name its
    source transcript file, so that a surprising verdict is diagnosable in
    one glance rather than via manual grepping.
11. As an operator, I want a recorded refusal-rerun (FALLBACK record) on a
    shared ordered/fallback id to judge escalated-with-reason, so that a
    properly documented fallback is never a standing false blocker (ADR
    0012).
12. As a team lead dispatching with explicit models, I want my `-m` spawn
    flags and `spawn_agent` model fields read as dispatch records, so that
    the models I declared are the evidence I am judged by.
13. As a team lead on cmux, I want the team skill to write an ESCALATION
    record when the cluster highest-tier rule up-tiers a ticket, so that
    deliberate cluster routing judges escalated-with-reason instead of
    accumulating standing warns.
14. As the maintainer of pre-convention tickets, I want a `tier: n/a`
    exemption (mirroring `review-tier: n/a`), so that I001–I007-era tickets
    stop generating permanent noise while a genuinely missing tier stays
    loud.
15. As the owner of the M4b merge gate (moo-clone I028), I want I024 to
    judge `match` from real transcripts, so that the gate can go green on
    evidence rather than on a waiver.
16. As an operator auditing M4a retroactively, I want honest verdicts for
    pre-I038 builds (`unmapped-dispatch` for gpt-5.5 workers, `match` for
    sol-on-primary), so that history is reported as it was, not laundered
    into matches.
17. As an operator hit by the I008 collision, I want claude-side dispatches
    qualified to the audited repo (repo reference in dispatch text or session
    cwd evidence), so that another repo's same-numbered ticket cannot produce
    a false blocking verdict.
18. As an operator, I want `--since` and `--session` filters, so that I can
    hand-scope pathological transcript sets without copying files to a
    scratch dir.
19. As a future auditor of a mixed build (some tasks claude, some codex), I
    want flavor derived per transcript source and threaded per token (D15),
    so that each dispatch is judged within the flavor that actually ran it.
20. As a spine maintainer, I want the codex reader to degrade-never-fail
    (missing dir, unreadable file, unrecognized shape → warning), so that an
    undocumented format shift can never break the verify stage outright.
21. As a spine maintainer, I want the audit refused on a
    newer-than-compiled template generation exactly as today (D14), so that
    codex parsing never emits confident verdicts from a misparsed repo.
22. As a skill maintainer, I want the audit's attribution rules documented in
    the glossary (CONTEXT.md), so that team skills and the audit share one
    vocabulary for dispatch records, workers, and orchestrators.

## Implementation Decisions

**D20 — Codex evidence sources.** Three, mirroring the claude reader's
semantics. (1) Dispatch records: `spawn_agent` function calls (explicit
`model` field; ticket token matched case-insensitively in `task_name`, which
is lowercase by convention) and team spawn commands whose arguments carry an
explicit model (`herdr agent start … -- -m X` and the cmux equivalent).
(2) Spawned-thread actuals: `thread_spawn` subagent files' per-turn
`turn_context.payload.model`, linked to their tree by root session id;
actuals supersede the dispatch's declared model where linkable, exactly as
claude subagent transcripts supersede dispatch aliases. (3) A worker-session
scan (D21) covering builds that predate explicit-model dispatch (pre-I038
cmux teams). Model evidence is always per-turn: `session_meta.payload.model`
is present but null and is never read; a session that switched models
mid-run contributes each turn's model.

**D21 — Worker attribution rule.** A top-level codex session (thread_source
"user", no parent) counts as worker evidence for ticket T iff: it is
repo-scoped (D22), AND T's token appears in the FIRST LINE of the session's
opening user message (the dispatch brief's title line — amended at I042
review: whole-message matching let a brief's context sentence naming a
higher-tier neighbor manufacture a blocking verdict on it; first-line
matching mirrors the claude reader's existing firstLine(d.prompt) treatment,
which exists for exactly this reason, and the estate's brief convention
carries the token in the title. A brief whose title omits the token degrades
toward no-transcript/unattributed — honest, never a manufactured blocker),
AND the session contains no dispatch records of its own. The third clause is
the orchestrator exclusion — any session that dispatches is an orchestrator
and its own models are never ticket evidence, generalizing the claude
reader's main-session rule. (Ratified at I042 review: "contains dispatch
records" means any spawn-SHAPED record — a spawn_agent call or team-spawn
command with or without a usable model field — per the glossary's broad
definition; a model-less dispatcher is still an orchestrator. The asymmetry
governs: false-orchestrator costs one missed worker session, false-worker
attributes orchestration turns and can manufacture a blocking verdict.) Validated against M4a:
opening-message matching excludes the neighboring-ticket bleed present in
later messages; a worker that itself spawned a spec-review subagent loses its
own turn evidence but keeps equivalent evidence through its `spawn_agent`
dispatch record, so the rules compose without loss.

**D22 — Repo scoping (codex).** A session belongs to the audited repo iff
its cwd resolves inside the repo, OR its `session_meta.payload.git.commit_hash`
is a commit known to the audited repo (one git object-existence probe per
distinct hash; spine already shells to git elsewhere). This covers worktree
cwds like /private/tmp team dirs and makes cross-repo token collision
impossible unless repos share history. A failing or absent git probe degrades
to cwd-only plus a report warning.

**D23 — Guardian exclusion.** Threads whose source marks them as guardian
auto-review are structurally excluded from all evidence paths and never
judged. Their reported model is synthetic (`codex-auto-review`); reading it
naively would score tickets as routed to a review model — the
confident-wrong-answer class this design treats as strictly worse than
`no-transcript`. Should a synthetic token leak through anyway, it maps to no
table entry and surfaces as `unmapped-dispatch` rather than passing silently.

**D24 — Verdict vocabulary.** New warn-level verdict
`unattributed-transcript`, same non-blocking severity band as
`no-transcript`: ticket-relevant, repo-scoped material exists, but none met
attribution (guardian-only matches, token absent from opening message,
orchestrator-only mentions). The detail names what was found, why it was
excluded, and the source file. `no-transcript` narrows to mean literally
nothing found. Judged codex verdicts name their source transcript file in the
detail line; the I008 requirement (silent-descent names its source) is
satisfied as a special case.

**D25 — Record-wins fallback reading (ADR 0012).** FALLBACK-record
consultation moves before tier resolution: when a ticket carries a FALLBACK
record and an observed token's candidate tiers include fallback, the token
resolves as fallback and judges escalated-with-reason. Without a record the
ordered reading stands and real descent still blocks. Trusts
operator-authored records, consistent with ESCALATION-record semantics.

**D26 — Cluster up-tiering records at source.** The cmux team skill appends
a model-tier ESCALATION record per up-tiered ticket at cluster spawn (the
D18 highest-tier rule made ledger-visible), so audited cluster builds judge
escalated-with-reason. The audit learns nothing about clusters — it cannot
verify membership from transcripts and must not excuse what it cannot see.
This is a deepthought-side work item within this effort; until it lands,
cluster up-tiers judge escalated-no-reason (warn), which is accurate.

**D27 — `tier: n/a` exemption.** A ticket may declare `tier: n/a` to opt out
of routing judgment, mirroring `review-tier: n/a`. The audit reports it as
exempt (distinct from unannotated, never judged); an empty tier stays loud
because absence is a gap while n/a is a decision. One-time edits to
pre-convention tickets in affected repos ride along with adoption.

**D28 — Claude-side repo qualification and operator flags (I008).** A claude
dispatch claims a ticket only if its description/prompt also references the
audited repo (absolute path or basename token) or its session shows cwd
evidence inside the repo. New `--since <time>` and `--session <id>` filters
scope the transcript set as operator escape hatches. No started-date
anchoring: time-anchoring to the current build would blind multi-milestone
repos to earlier builds' transcripts (M4a), the wrong default for the estate.

**Flavor threading (per D15's named seam).** Flavor derives per transcript
source — claude layout vs codex sessions — and travels beside each token into
judgment, so mixed builds judge each dispatch within its own flavor's
resolved table. The audit continues to own no WORKFLOW.md parser; both
flavors resolve through the shared resolver (D13). Discovery is always on:
default `$CODEX_HOME/sessions` else `~/.codex/sessions`; missing or
unreadable degrades to a warning. (Warning rule, ratified at I041 review:
a missing *explicitly requested* dir warns; a missing un-overridden
*default* is a silent skip — a codex-less machine is normal, and a
standing warning on every audit there is exactly the permanent-noise
failure the problem statement decries. At the library boundary an empty
codex-sessions option means "no codex discovery", which is what keeps
claude-only callers byte-identical.) The audit entry point takes an options
struct (repo dir, claude transcripts dir, codex sessions dir, filters) so
future inputs don't churn the signature. D14's generation gate applies
unchanged. Codex ticket-token matching is case-insensitive; the claude
reader's matching is untouched.

## Testing Decisions

A good test here asserts external behavior at the existing package boundary:
given a repo state and transcript fixtures on disk, what Report does the
audit entry point return. Tests must not reach into reader internals or
intermediate structures — the codex format is undocumented and the parser
will shift; tests coupled to it would obstruct exactly the adaptation this
effort exists to make routine. (Scope, ratified at I040 review: this ban
targets the undocumented parsing layer — scanJSONL/parseLine, codex JSON
shapes — not the resolution seam D15 named and stabilized as a contract;
a white-box test of per-token flavor resolution is legitimate where no
Run-level fixture can yet construct the discriminating input.)

**Audit module (the single behavioral seam).** Scenario fixture directories
following the existing clean/degraded/mixed/vacuous convention, extended
with hand-written minimal codex JSONL fixtures encoding each verified format
fact. New scenarios: dispatch-record evidence (spawn_agent and -m spawn);
spawned-thread actuals superseding declared models; worker attribution
(opening-message hit counts, mid-transcript mention does not, orchestrator
session never); guardian exclusion; per-turn evidence from a model-switching
session; repo scoping by cwd and by known commit (tiny real git repos built
in test temp dirs — precedented, spine already shells to git); cross-repo
token collision yielding no contamination; `unattributed-transcript` for
guardian-only and mid-transcript-only matches; record-wins fallback reading
(with record → escalated-with-reason; without → silent-descent); `tier: n/a`
reported exempt; empty tier still unannotated; mixed claude+codex evidence
judged per-token per-flavor. The existing silent-descent and
escalation-record scenarios must pass unchanged — this work must not alter
what the audit blocks on except where ADR 0012 says so.

**CLI.** The existing command-runner tests extend for `--codex-sessions`,
`--since`, `--session`: flag parsing, default derivation, degrade warnings.
The printer stays a thin projection of Report.

**Skill regression (deepthought).** The grep-style shell test pattern beside
the existing preflight test asserts the cmux team skill emits an ESCALATION
record line at cluster spawn for up-tiered tickets. Deliberately a weak
behavioral test and a strong regression guard, per the I038 precedent.

**Live acceptance (verify stage, not CI).** Against the real session store:
moo-clone I024 → match; M4a I008–I015 → unmapped-dispatch (gpt-5.5 was never
a declared id — history reported honestly); I021/I022 → match; guardian
threads contribute nothing anywhere; praxis/maipipe audits unaffected by
moo-clone's tokens and vice versa.

## Out of Scope

- Model drift within a session (a runtime silently moving a session to
  another model mid-flight) — verdict vocabulary deliberately left open for
  it (ADR 0011's note); per-turn evidence here reports what ran, it does not
  distinguish dispatcher intent from runtime drift.
- Adding `gpt-5.5` (or any never-shipped id) to the model table's history to
  make M4a judge match — fabricating a shipped default to launder history is
  exactly what provenance-scoped matching exists to prevent.
- Effort auditing (verifying `@ xhigh`-style effort suffixes from
  transcripts) — the escalation grammar's effort records remain accepted and
  unused, as today.
- Claude-team worker model selection bridging to the table (noted as a
  follow-up candidate in D18's scope clause) — unchanged here.
- The C4/C5 deferred minors from the gen-6 build (transcript-slug
  derivation, last-FALLBACK-reason-wins) unless they fall out trivially
  during implementation; they remain on I008's tail otherwise.
- Any new subcommand or output format redesign; the printer's table shape
  stays.

## Further Notes

- The verified codex transcript format facts (line kinds, null
  session_meta model, thread tree ids, guardian markers, spawn shapes,
  worktree cwds, git payload contents, M4a survival window) are recorded
  dated in I009 and were re-verified against ground truth on 2026-07-25;
  the format is undocumented and version-drifts, so the reader keeps the
  claude reader's degrade-never-fail posture everywhere.
- Expected honest outcomes on today's estate, for calibration: moo-clone
  I024 match; M4a I008–I015 unmapped-dispatch (warn, non-blocking — the
  explicit-model convention postdates those builds); I021/I022 match;
  two M4a sessions with empty opening tokens contribute nothing
  (unattributed-transcript at worst).
- Ticket-id collision across repos remains a fact of the estate (ids restart
  at I001 per repo); this design makes the audit immune to it rather than
  renumbering the estate.
