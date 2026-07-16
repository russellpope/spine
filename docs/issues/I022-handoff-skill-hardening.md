---
id: I022
title: /handoff skill hardening — verbatim cursor rule
severity: med
status: open
affects: [skills]
blocked-by: [I018]
execution-mode: subagent-driven
tier: mechanical
review-tier: routine
---

## What to build

Plan Task 5. Edit `~/.claude/skills/handoff/SKILL.md`: for spine repos, every handoff MUST embed the verbatim output of `spine cursor` in a dedicated section; prose paraphrases of stage state are defined incomplete; resume/kickoff prompts inherit the same rule. Single-file doc edit — the template-side rule text ships with I020.

## Acceptance criteria

- [ ] Skill instructs running `spine cursor` and pasting output verbatim, with the incompleteness rule stated
- [ ] Wording consistent with the gen 8 WORKFLOW.md section (I020) and the I014 resolution
- [ ] No other skill semantics changed
