---
id: I089
title: "Self-enable gate_pack go@1 on spine (WORKFLOW.md keys, maipipe.toml region, doctor D10 round-trip)"
severity: med
status: fixed
affects: [I085]
blocked-by: [I088]
execution-mode: inline
tier: primary
effort:
risk-triggers: []
review-tier: n/a
---

## Problem

spine ships the go@1 gate pack (I082–I086) but does not run it on itself.
Dogfood plan (deepthought handoff 2026-08-19 §1b–d): set `gate_pack: go@1`
+ `gate_pack_config.tskip_allow` in spine's WORKFLOW.md; `spine update
--dir . ` dry-run then `--write`; inspect the `# spine:begin gate-pack go@1`
region in `maipipe.toml`; `spine doctor` → D10 silent; hand-edit one line
inside the region → D10 fires; revert via `spine update --write`. Then the
maipipe composition check (a `full` lane stage `pipeline = "gate-go"`
outside the region; `maipipe validate`; findings carry `code = go@1/<check>`)
and `mutate` with a `docs/mutation-spec.json` (2–3 probes incl. one
`report_only`; deliberate build break → control failure exit 1).

## Fix

Record each step's command + output in the ticket resolution; any defect
found gets its own ticket.

## Resolution (2026-08-19)

- WORKFLOW.md: `gate_pack: go@1`, `gate_pack_disabled: [test-enum-vs-spec,
  n-plus-one, fixture-manifest]` (spine has no enum-marked spec, client
  verbs, or fixture manifest), `build_outputs: bin/spine`,
  `tskip_allow: cmd/spine/dogfood_test.go,internal/cursor/dogfood_test.go`.
- `spine update --dir .` dry-run showed the region diff; `--write` created
  `maipipe.toml` (commit 43793a6; re-created after I091 → `maipipe validate`
  OK). Owner lanes `fast` (vet, test) and `full` (fast → `pipeline =
  "gate-go"`) appended below the region; `spine update` → `up-to-date`.
- Doctor: no D10 on the canonical region; hand edit inside the region
  (`--verbose` on the tskip `run`) → `D10 warn maipipe.toml: 1 line(s) …
  not canonical`; `spine update --write` refuses (unrecognized local edit),
  `--write --force` restores it — and also drops the unrelated README.md
  local edit (I093.4), restored by hand.
- maipipe: `maipipe run gate-go --wait` → passed, 5 stages pass/0 (env
  delivery proven: tskip and gitignore-control only pass with their env).
  Seeded `t.Skip` on a scratch ref → tskip fail/1, DB row `code =
  go@1/tskip` (`maipipe findings` CLI omits `code` — maipipe-side note).
  maipipe runs the definition at the pinned commit, so `maipipe.toml` must
  be committed before a lane runs.
- mutate: `docs/mutation-spec.json` (3 probes) → 1 KILLED, 1 SURVIVED,
  1 report-only SURVIVED; kill rate scorable 1/2; deliberate build break →
  control failure, exit 1, verify tail. Under maipipe: I092 (three contract
  defects) then `mutation-go` passed with 3 rows.
- Checkpoint (§1f): `spine checkpoint new … --gate pass --effort medium` →
  001; `latest` ×2 byte-identical (`cmp`); shape tamper (`gate:  pass`) →
  `D11 warn … facts region malformed`; restored → silent; `--facts-only` →
  002 `narrative: missing`. Value-preserving edits of valid values
  (`pass`→`fail`) are not detected — spec scope (I093.5).
- Model (§1g): `spine model --alternate pi routine` → qwen3.8-27b-q8_0
  (`--effort` → xhigh; primary medium); repo with edited `pi.routine … alt:
  ornith-1.0-35b @ low` → `--json` provenance `override`, alternate
  ornith/low.
- Routing audit (§1h): I079–I087 `no-transcript` reproduced → I090.
