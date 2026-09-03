---
id: I127
title: "add an /install-style skill (toolchain check, build, free verification lane, live check) and choose a LICENSE"
severity: low
status: open
affects: []
blocked-by: []
execution-mode:
tier:
effort:
risk-triggers: []
review-tier:
---

## Problem

spine's README (landed 2026-08-24) has an Install section and the repo has a
deterministic zero-cost verification lane (`go test ./...`, the gate-pack
positive controls), but there is no wrapper that walks a newcomer through
toolchain check, build, the free lane, and a live launch check and reports
the result. There is also no LICENSE file (checked 2026-09-02), which blocks
redistribution outright. spine is first in the redistribution readiness
order (spine, then maikanban, then maipipe).

Source: fusion-harness borrow hitlist item 5
(`docs/research/2026-08-26-fusion-harness-borrow-hitlist.md`): clone, one
dependency command, deterministic verification lane, live launch check, and
an agentic `/install` command that walks all of it.

## Fix

1. Owner chooses the LICENSE (the hitlist's reference is MIT); add the file.
2. Add a project skill under `.claude/skills/install/` that: checks the Go
   toolchain against `go.mod` (reusing the D14 comparison), runs `make
   install`, runs `go test ./...`, runs one gate-pack positive control, then
   runs `spine version` and `spine doctor --dir .` on the repo itself, and
   prints a pass/fail summary with the exact commands.
3. Point the README's Install section at the skill.

## Acceptance criteria

- [ ] A LICENSE file exists at the repo root and the README names it.
- [ ] Running the install skill on a clean clone with a matching Go toolchain ends with every step passing and both binary paths (`~/bin`, `~/.local/bin`) explained.
- [ ] Negative control: with an older Go on PATH the skill stops at the toolchain step naming the mismatch, without building.

<!-- Record an approved-without-test exception using the exact grammar in WORKFLOW.md's Acceptance exceptions section. -->
