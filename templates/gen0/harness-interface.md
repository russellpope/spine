# Functional-test harness — interface contract

The implement stage runs functional tests against the RUNNING system in a
generator/evaluator loop until the feature is complete. Each project declares a
harness type in `WORKFLOW.md` (`functional_harness`).

- `cli` — a command to invoke + expected exit code / output assertions; Claude walks the CLI.
- `rest` — a base URL + endpoints/requests to exercise + expected responses.
- `framebuffer` — a virtual-framebuffer endpoint with LLM interaction (user's harness); pluggable, internals out of scope here.
- `none` — no functional harness (e.g. presentation profile).

Contract: the harness exposes (1) a `run` command, (2) machine-checkable success criteria,
(3) idempotent re-runs. The loop stops when success criteria pass or a max-iteration budget is hit.
