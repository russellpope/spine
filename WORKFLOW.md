# Workflow — spine

profile: library-cli
template_version: 3
reviewers: [go-reviewer, python-reviewer]
functional_harness: cli    # cli | rest | framebuffer | none
gates: [grill, verify]             # mandatory; everything else advisory. verify = fresh-context verifier subagent(s) against the PRD/spec, not self-review
model_routing:
  primary: claude-fable-5          # long-horizon, ambiguous, or first-shot-complex work (design, plan, implement, orchestrate)
  fallback: claude-opus-4-8        # auto on stop_reason: refusal (cyber/bio/reasoning-extraction); also context/usage exhaustion
  routine: claude-sonnet-5         # mechanical subagent roles: doc edits, plan-transcription implementers, build fixers, simple reviews
effort: high                       # default; xhigh for security-critical analysis + final verification; medium/low for routine subagents
model_default: claude-fable-5      # swappable; re-evaluate on major model/platform releases
security_routing: quality-framing-opus-4-8
stages: [grill, prd, issues, implement, functional-test, review, verify, ship, deploy, docs, handoff]

See `docs/harness-interface.md` for the functional-test harness contract.
Mandatory gates: a PRD up front (grill-with-docs -> to-prd) and verification before completion.
Execution mode per plan: live-system mutation, secrets, or interactive steps -> inline with the human; otherwise subagent-driven.
