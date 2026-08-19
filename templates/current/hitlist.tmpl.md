# Hitlist

<!--
Copy this file to `docs/remediation/<effort>/hitlist-N.md` and fill it in.
N is the round number.

The "no fix text" rule below is dose-scoped: it is the property of the
`findings-only` dose, which is the default. A `prescriptive` or `raw-review`
hitlist carries more, which is why the header must state the dose — a reader
can then tell one dose from another without reading the whole file.
-->

effort: <kebab-case effort name, as the stage cursor spells it>
round: 1
dose: findings-only
source run id: <the run id these findings came from>

## Findings

### `go@1/tskip`

- file:line — `internal/thing/thing_test.go:42`
- finding: the test skips instead of asserting when the fixture is absent.
- why it matters: a skip is a green build with no evidence behind it; the
  check class exists to make missing evidence visible.

### `go@1/deferred-cleanup-errcheck`

- file:line — `internal/thing/run.go:88`
- finding: the deferred cleanup call discards its error return.
- why it matters: the failure is invisible at the call seam, so the caller
  proceeds on a value that was never produced.

## Do not regress

These mutation rows were killed by the current tests. A fix that lets any of
them survive is a regression, whatever else it improves. Rows are named by
their `code` and probe id, as `spine gate go mutate` reports them.

- `go@1/mutate` M001 — `internal/thing/run.go:51`
- `go@1/mutate` M004 — `internal/thing/run.go:73`
- `go@1/mutate` M009 — `internal/thing/parse.go:12`
