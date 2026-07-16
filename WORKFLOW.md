# Workflow — spine

profile: library-cli
template_version: 9
reviewers: [go-reviewer, python-reviewer]
functional_harness: cli    # cli | rest | framebuffer | none
gates: [grill, verify]             # mandatory; everything else advisory. verify = fresh-context verifier subagent(s) against the PRD/spec, not self-review
model_routing:
  primary: claude-fable-5          # default thinker: design, judgment, orchestration, final review
  routine: claude-sonnet-5         # multi-step mechanical subagent roles
  mechanical: claude-haiku-4-5     # verbatim plan-transcription + single-file mechanical fixes ONLY
  fallback: claude-opus-4-8        # primary-refused or security-framed work
effort: high                       # tier default: primary=high, routine=medium, mechanical=low, fallback=high; xhigh reserved for final verification and security-critical passes; per-ticket effort: only on deviation
model_default: claude-fable-5      # swappable; re-evaluate on major model/platform releases
stages: [grill, prd, issues, implement, functional-test, review, verify, ship, deploy, docs, handoff]

See `docs/harness-interface.md` for the functional-test harness contract.
Mandatory gates: a PRD up front (grill-with-docs -> to-spec), spec-review of the finished diff against the PRD, and verification before completion.

## Stage cursor (consistency rule)

Stages run **in order**; none may be silently skipped (the miss mode is a handoff that names an
abbreviated path — e.g. "grill -> to-spec -> build" quietly dropping `issues`/`to-tickets`). To
prevent it, every SDD effort's `.superpowers/sdd/progress.md` opens with a machine-readable
stage cursor block — one `<!-- spine:cursor -->` block naming the effort, PRD, ticket range, and
every stage's marker. `[x]` marks a done stage, `[<]` marks YOU ARE HERE (exactly one, among the non-done stages), `[ ]` marks pending.
The cursor is the single source of truth for "where are we"; check it at session start before acting.

Grammar reference (documentation only — the real block lives at the head of
`.superpowers/sdd/progress.md`, never here):

    <!-- spine:cursor -->
    effort: <kebab-name>
    prd: docs/specs/<file>.md
    tickets: I0NN | I0NN-I0MM | prefix I0
    stages: grill[x] prd[x] issues[x] implement[<] functional-test[ ] review[ ] verify[ ] ship[ ] ...
    <!-- /spine:cursor -->

**Handoff rule:** `/handoff` and any resume/kickoff prompt MUST embed the verbatim output of `spine cursor` — a prose paraphrase of stage state is incomplete; the reader can't see which upstream stage was skipped from a summary alone. Alongside `spine audit stages` blocking on a missing/stale cursor block in the newest handoff, `spine doctor` advises (warns) on the same condition.

## Model routing

Artifacts (plans, tickets) reference tiers, never model ids — the mapping above is per-repo remappable (new model families, local models, other providers).

Escalation: dispatch may exceed a ticket's annotated tier or effort freely, WITH a recorded reason; dispatching below the annotation without a matching record is silent descent and fails the verify gate. Record grammar (exact — arrow is unspaced `->`; spaced arrows do not parse), one line each in `.superpowers/sdd/progress.md`:

    ESCALATION <ticket-id> <from-tier>-><to-tier> reason: <one line>
    ESCALATION <ticket-id> effort <from>-><to> reason: <one line>
    FALLBACK <ticket-id> reason: <one line>

A record excuses exactly its to-tier, nothing else. Any record not matching the grammar exactly excuses nothing — spaced arrows, missing `reason:`, missing tokens, all of it.

Reviewer floor: review-tier is never below tier; inline tickets carry `review-tier: n/a` — no per-task review cycle exists, verify-stage gates still apply. Risk triggers force primary-tier review — cross-task-integration, concurrency-subtle-state, security-surface, plan-flagged-ambiguity. The final whole-branch review and acceptance simulation always run primary. Reviewers re-run claims and demand raw transcripts at every tier.

Fallback routing: proactive — security-framed work (attacker/exploit framing) routes to fallback from the first dispatch; security-touching but quality-framed work stays on its natural tier with the security-surface trigger. Reactive — on a primary refusal the orchestrator re-dispatches on fallback with quality framing, writes a FALLBACK record, and push-notifies the owner.

Dispatch conventions the audit depends on: every subagent dispatch carries an explicit model (never inherit), and its description contains the ticket id token (the correlation contract). Verify stage: run `spine audit routing` (add `--transcripts <dir>` when the controller session runs in a different repo than the audited one) — reasoned escalations are advisory, silent descent blocks.

## Execution modes

subagent-driven is the default for planned build work. ultracode is for work whose shape demands parallel orchestration (unknown-size discovery, cross-cutting audits, N-perspective verification); opt-in is granted by the owner's ticket approval, mid-build escalation is recommend-only. inline is the rare justified exception — tightly-coupled sequential chains, verbatim pre-specified diffs, live-system/secret/interactive steps — and requires a one-line justification in the ticket.
