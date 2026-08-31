# CONTEXT — unified workflow (spine estate)

Glossary only. Implementation decisions live in docs/adr/; specs in docs/specs/.

## Execution modes

How work runs. Orthogonal to model tiers (what runs it). Every ticket/stage
declares exactly one.

- **inline** — the session itself executes the work. The rare, justified
  exception, not a default: tightly-coupled sequential chains over shared
  files, pre-specified verbatim diffs, or live-system/secret/interactive
  steps. The session model does the work, so no model routing applies.
- **subagent-driven** — sequential conveyor per superpowers
  subagent-driven-development: one fresh implementer + reviewer per task,
  gate between tasks. The default for planned build work.
- **ultracode** — multi-agent Workflow orchestration: parallel fan-out,
  judge panels, adversarial verify inside a single step. For work whose
  shape demands it (unknown-size discovery, cross-cutting audits,
  grounding sweeps, N-perspective verification). NOT a synonym for
  subagent-driven; historical notes conflate the two.

## Model routing

- **flavor** — deprecated compatibility name for **harness**. Generation 14
  keeps only the equal JSON `flavor` field and legacy defaults `flavors`
  reader; current CLI, source, and generated files use `harness`.
- **model tier** — a semantic role name, deliberately provider-agnostic:
  - **primary** — the default thinker: design, judgment, orchestration,
    final review.
  - **routine** — mechanical-but-multi-step subagent roles: implementers
    working from prose, doc edits, build fixers, task-scoped reviews.
  - **mechanical** — definitionally narrow: verbatim plan-transcription
    implementers and single-file mechanical fixes ONLY (the plan text
    already contains the code).
  - **fallback** — where primary-refused or pre-flagged dual-use/security
    work runs.
  Artifacts (plans, tickets) reference tiers, never model ids (decided
  2026-07-09, unchanged). The mapping itself is keyed by **(harness, tier)**
  and resolves in spine; each repo's WORKFLOW.md carries a **mirror** of it
  so the estate can still remap per repo (revised 2026-07-24, ADR 0011).
- **model table** — the estate's `(harness, tier)` → model id + optional
  effort mapping. Spine ships its defaults and remembers every default it has
  ever shipped, which is what makes an inherited value distinguishable from a
  deliberate override.
- **active launch ID** (decided 2026-08-29, I051) — the one model ID a new
  launch may use for a requested `(harness, tier)` under the current repository
  snapshot: the embedded current ID, an exact current mirror value, or a safe
  deliberate mirror override. Matching is byte-for-byte. No trimming, case
  folding, family inference, alias expansion, or historical lookup can make a
  candidate active. Aliases and historical IDs are **audit evidence**: they
  keep old transcripts interpretable but never authorize a new launch.
- **model launch validation** (decided 2026-08-29, I051) — the fail-closed,
  model-ID-only check exposed as `spine model [--dir D] validate [--expect
  MODEL_ID] <harness> <tier>`. Outer `--dir` precedes the `validate` positional;
  nested `--expect` precedes harness/tier. It reads and classifies one
  `WORKFLOW.md` snapshot, rejects unsafe, forbidden, retired, mismatched, and
  unmapped IDs, and has no bypass. It does not validate or transport the
  resolved **effort** (I075 owns that dispatch parameter) and does not validate
  a cell's **alternate**. Atomicity ends when the command returns: a later
  repository edit may change what audit sees, and the local checkout writer
  and launcher remain trusted. I051 adds no receipt, lock, or policy digest.
- **mirror** — the rendered copy of the resolved model table in a repo's
  WORKFLOW.md, marked spine-managed. Read by humans and by Spine's resolver and
  launch validator; vendor launchers do not interpret it directly. A value
  matching a shipped default is **inherited** and refreshed automatically; any
  other value is an **override** and is preserved.
- **reviewer floor** (decided 2026-07-09) — a task's reviewer is never a
  lower tier than its implementer; plan-time risk triggers (cross-task
  integration, concurrency/subtle state, security surfaces, plan-flagged
  ambiguity) force a primary-tier review; the final whole-branch review +
  acceptance simulation always runs primary. Review procedure (re-run the
  claims, demand raw transcripts) is mandatory at every tier. Inline
  tickets carry `review-tier: n/a` — no per-task review cycle exists;
  verify-stage gates still apply (refined 2026-07-10, I004 review).
- **routing purpose** (decided 2026-07-09) — quality ceiling first: the
  primary model is the default thinker; down-routing exists to stop waste on
  provably mechanical work, not to chase spend. Auditability is the
  enforcement layer: actual model per task is verified against declared
  routing from transcripts, every build.
- **ultracode opt-in** — the harness requires explicit user opt-in for
  Workflow orchestration. Plan-gated: tickets marked ultracode by
  /to-tickets, approved by the user, carry the opt-in; mid-build escalation
  is recommend-only (user says the word).
- **escalation** (decided 2026-07-09) — a dispatch-time tier or effort
  increase above the ticket's annotation, always with a recorded reason.
  Permitted freely. The inverse — **silent descent**, dispatching below the
  annotation without a recorded reason — is a gate failure. The ESCALATION
  record grammar covers reasoned deviation in either direction — a downward
  record keeps descent advisory (refined 2026-07-10, final review).
- **fallback routing** (decided 2026-07-09) — two paths: proactive
  (security-FRAMED work is pre-flagged at intake/plan time and routed to
  fallback from the first dispatch — the classifier keys on framing, not
  file contents) and reactive (a primary refusal triggers orchestrator-
  mediated re-dispatch on fallback with quality framing, ledger-recorded,
  push-notified). Never described as "auto" — the orchestrator is the
  mechanism.
- **effort routing** (decided 2026-07-09) — effort follows the tier's
  default (primary=high, routine=medium, mechanical=low, fallback=high)
  (amended 2026-07-10, final review); per-ticket overrides follow the
  escalation rule. A model-table entry may carry its own effort, overriding
  the tier default for that (harness, tier) — the tier's default effort does
  not always produce the expected behavior from a given model. Resolution
  always yields a determinate effort; an entry that omits one inherits the
  tier default rather than deferring to a runtime's own setting. "xhigh
  reserved for final verification and security-critical passes" is guidance,
  not a gate (revised 2026-07-24).
- **routing audit** (decided 2026-07-09) — deterministic post-build diff of
  declared tier annotations vs actual models in the transcript, per task
  (`spine audit routing`). Required at the verify stage: reasoned
  escalations advisory, silent descent blocking.
- **dispatch effort declaration** (decided 2026-08-30, I075) — the exact raw
  triple written before one controlled launch:
  `harness=<raw execution vehicle> model=<exact selected ID> effort=<exact raw token>`.
  It records a transport request, not a provider, gateway, or runtime result;
  a missing recorded declaration is `declared-effort=-` and I075 always reports
  `observed-effort=-`. Raw effort tokens have no global ordering.
- **effort authorization** (decided 2026-08-30, I075) — one exact ledger
  record, `ESCALATION <ticket-id> effort <from>-><to> reason: <one line>`,
  authorizing only a declaration for that ticket whose selected target token is
  exactly `<from>` and whose declared token is exactly `<to>`. The arrow is
  unspaced; missing endpoints, duplicate or empty `reason:`, spaced arrows,
  reversed pairs, and other grammar changes authorize nothing. This record is
  declared-effort-only and never changes the model-tier routing verdict.
- **heterogeneous routing verdict** (decided 2026-08-30, I074) — a complete
  declared triple on a configured host is compared with the final host-selected
  pair, never the repository preference. Confirmation needs the exact
  `(source, session, dispatch)` identity, a linked worker event, and a
  byte-exact host-local `observed_ids` mapping. `declared-observed-mismatch`
  and `declared-effort-mismatch` block; absent proof is `unconfirmable` and
  does not block. Current transcript readers expose no documented observed
  effort, so real records keep `observed-effort=-`; no alias, history,
  canonical-ID shortcut, normalization, family inference, or cross-host
  lookup fills that gap. I073 owns this public harness migration.
- **harness** (ratified 2026-08-10, I067) — the execution vehicle that runs a
  dispatch: claude, codex, and from 2026-08-18 **pi** (the weak-local /
  open-weight driver). It is the model table's first axis;
  the model cell carries the actual model id whatever its family. Reachability
  of a model from a given host is a separate, per-host constraint (I068/I072),
  never a harness. _Avoid_: "local harness" when referring to a model's
  location; local is a property of where a model is served, not a harness.
- **openweights** (added 2026-08-25, I110) — a fourth first-axis value, mapping
  every tier to an open-weights model served through the Cascade gateway at
  effort `high`, `fallback` deliberately equal to `primary`.
  **It does not satisfy the definition above, and that is a known, unresolved
  tension rather than an oversight.** An openweights dispatch runs the ordinary
  Claude Code binary via a wrapper that passes `--model` through — it is not a
  distinct execution vehicle but the *claude* harness pointed at other models.
  By this glossary's own test it is closer to "where a model is served", which
  the entry above says is *never* a harness and names as the thing to avoid.
  Two consequences, both real:
  - It is why **I111** has to exist. Those sessions write to
    `~/.claude/projects` *because they are Claude Code sessions*, so deriving
    the axis from the transcript source misfiles every one of them. Needing to
    derive the first axis from the observed model id instead is the clearest
    evidence that this value is not a harness.
  - `pi` is a genuine counter-example and not a precedent: it is its own
    driver binary, so it *is* an execution vehicle even though it also happens
    to serve open weights.
  Filed as **I112** for the owner to decide: rename the value, split the axis
  into (harness, model-family), or ratify the widened definition. Nothing
  blocks on it — resolution and dispatch work today — but the domain model and
  the shipped table currently disagree, and the glossary should not pretend
  otherwise.
- **alternate** (decided 2026-08-18, local-harness grill) — an optional second
  `(id, effort)` a model-table cell names for the same (harness, tier), used
  when a critic should differ from the author (pure self-review is
  structurally weak: zero self-flagged shortcuts across the corpus). A
  different model is preferred; the same model at a different effort is a
  legitimate, owner-tuned choice (cf. claude routine = opus-5 @ low over a
  smaller model). Owner-ratified data, never computed at dispatch; absent
  means same-model-fresh-session is the only critic available.

## Stage cursor (decided 2026-08-06, cursor-writes grill)

- **stage cursor** — the machine-parseable record of where an effort stands:
  effort name, PRD, ticket range, and per-stage state (done / here / pending).
  The single source of truth for "where are we"; every resume reads it first.
- **working home** — the cursor's live, mutable location (the effort ledger
  head). Uncommitted by convention; reflects the current moment.
- **committed snapshot** — the cursor copy captured inside a handoff at the
  moment of its creation. Historical, never retro-mutated; the newest
  snapshot is required to match the working home.
- **sole-writer rule** — only the spine tool mutates the cursor; hand-editing
  the block is a workflow violation. Rationale: cursor mistakes (duplicate
  here-markers, stale fields, unearned ticks) came from hand edits under
  context pressure.
- **canonical form** — the byte-deterministic serialization the tool emits.
  A valid-but-non-canonical block is evidence of a hand edit and fails the
  stage audit.
- **open fence** / **close fence** (added 2026-08-26, I109) — the two HTML
  comments delimiting the cursor, and the **fenced region** they bound. A fence
  counts only when it is the whole line starting at column 0; trailing
  whitespace is tolerated, leading whitespace is not. So a delimiter quoted
  mid-sentence, or indented as a worked example, is prose — which is what lets a
  document explain the convention without hijacking its own parse. Exactly one
  fence of each kind per document, close after open. _Avoid_ "marker" for these:
  that word is taken by the per-stage state characters below.
- **marker** — a per-stage state character inside the cursor: done, YOU ARE
  HERE, or pending. Distinct from a fence; the two collide in a single sentence
  otherwise ("the marker before the marker").
- **write-time tripwire** — stage-completion claims are checked against
  artifact derivation at the moment of the write, not only at audit time; a
  forced write defers the reckoning to the audit, never waives it.

## Checkpoint (decided 2026-08-18, local-harness grill)

- **checkpoint** — the document a running session distils itself into just
  before a context reload, so the next leg resumes without compaction: task,
  conclusions-with-why, and **next moves** (forward intent — the one thing
  measured lost across a context clear). Fires once or twice per long task;
  operational, not historical. Distinct from a **handoff** (session-end,
  human-facing, committed) and from the cursor's **committed snapshot**.
- **model region** — the checkpoint's model-authored narrative. Treated on
  reload as the model's own prior claims, never as evidence.
- **facts region** — the harness-appended machine facts (files touched, gate
  status, git sha, recommended per-leg effort). Sole-writer and canonical-form
  rules apply exactly as for the cursor; the two regions are structurally
  separate so narrative can never masquerade as fact.
- **checkpoint working home** — the uncommitted, ordinal-numbered location
  where checkpoints accumulate for the current effort; the newest one may be
  snapshotted into a handoff. _Avoid_: "state file", and bare "brief" (maipipe's
  relay term for what it hands a leg — a checkpoint is one possible brief
  payload). The two-word **dispatch brief** is a different, well-defined thing;
  see the Audit evidence section.
- **reload preamble** — the static, spine-shipped text that precedes the
  checkpoint in the reload prompt. Byte-stable so the prefix is cacheable;
  states the model-region/facts-region trust split explicitly.

## Gate pack (decided 2026-08-18, local-harness grill)

- **workflow gate** — a stage in a repo's WORKFLOW.md `gates:` list (grill,
  verify) that must pass before the effort advances. Human/agent-judged.
- **gate pack** — a spine-authored, independently versioned battery of
  deterministic **check classes** for one language (Go first), executed by
  maipipe as the enforcement floor. Versioned per pack (`go@1`), not by the
  template generation, so a finding is attributable to `<pack>@<v>/<check>`
  and a repo can opt out by dropping a check class. Not a workflow gate: it
  is content maipipe's verify gate runs. _Avoid_: "check pack" (rejected
  2026-08-18 — five PRDs already say gate pack), "gate stage" for the pack as
  a whole.
- **pack pin** (decided 2026-08-21, I103 grill) — the `<pack>@<v>` value a repo
  owns as `gate_pack` in WORKFLOW.md. It freezes *both* halves of what the
  identifier names: the check-class list and the attribution string
  `<pack>@<v>/<check>` on every finding. A pin naming a pack the running spine
  does not ship is refused, never approximated. `<pack>@<v>` is the one
  canonical form wherever the pin appears. _Avoid_: "pack version" for the
  repo-side value (that is the pack's own number, not the repo's choice).
- **check class** — one attributable, individually droppable check inside a
  gate pack (e.g. t.Skip zero-tolerance, deferred-cleanup errcheck). One
  maipipe stage per check class.
- **positive control** — the fixture pair every check class ships with in
  spine's own test suite: a known-good input the check must pass and a
  seeded violation it must fail. A check without both is not shippable.

## Audit evidence (decided 2026-07-26, codex-audit grill)

- **dispatch record** — an entry in an orchestrator's transcript declaring a
  worker dispatch and, where present, its model: a Claude Task/Agent
  tool-use, a codex `spawn_agent` call, or a team spawn command naming a
  model. Declared evidence; superseded by the dispatched agent's own actuals
  when linkable.
- **orchestrator session** — any session containing dispatch records. Its
  own models are never ticket evidence, in any harness; only what it
  dispatches counts. Generalizes the Claude reader's main-session rule.
  _Avoid_: lead session (herdr/cmux role name, narrower than this concept).
- **worker session** — a top-level codex session attributed to a ticket
  because the ticket's token appears in the first line of its opening user
  message (the dispatch brief's title; first-line rule ratified at I042
  review to kill context-sentence bleed) and it is not an orchestrator
  session. Its per-turn models are actual evidence.
- **dispatch brief** (decided 2026-08-24, I101 grill) — the instruction
  document a team lead writes for one worker before starting it, delivered to
  the worker by file reference rather than inline. Its first line names the
  ticket under work (the first-line rule again); its body routinely names other
  tickets for context and is therefore never attribution text. A brief is
  evidence only as the lead's transcript recorded it being written — the file
  on disk is not read. _Avoid_: bare "brief" (see checkpoint working home).
- **thread tree** — codex sessions form trees: each rollout file's
  session_meta carries its own thread id, its immediate parent, and the
  root's id (shared tree-wide). Membership is decided by root id, not by
  walking.
- **guardian thread** — a codex auto-review subagent thread; reports a
  synthetic review model, never a routed one. Structurally excluded from
  evidence, never judged.
- **attribution** — the rules deciding which observed models belong to which
  ticket (repo scoping, opening-message rule, orchestrator exclusion,
  guardian exclusion). Evidence that matches a ticket's token but fails
  attribution yields the unattributed-transcript verdict, distinct from
  no-transcript: found-but-unusable is not nothing-found.
- **tier: n/a** — explicit per-ticket exemption from routing judgment, for
  tickets predating the tier convention (mirrors `review-tier: n/a`). An
  empty tier stays loud: absence is a gap, n/a is a decision.

## Ticket batch (decided 2026-08-27, doctor-hygiene grill)

- **batch** — a set of tickets claimed together and worked as one unit,
  recorded on each member ticket by a board-issued id. The membership record
  is historical: it is never cleared, so a closed ticket keeps the batch it
  shipped in. Shared vocabulary across maikanban (which claims batches),
  claude-team (which executes them), and spine (which owns the ledger schema,
  I106). Distinct from the cursor's `tickets:` value — the cursor names what
  the current effort works; batch membership names what a board once claimed
  together, and the two need not coincide.
