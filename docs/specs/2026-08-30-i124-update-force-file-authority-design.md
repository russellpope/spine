# I124 — scoped `spine update --force-file` overwrite authority

**Status:** Accepted design

**Ticket:** `I124`

## Problem and outcome

The current `--force` changes every forceable managed file with an
unrecognized local edit. That global authority is useful and must remain
compatible, but it is unsafe when an owner intends to regenerate only one
managed candidate such as `maipipe.toml`. Add repeatable scoped authority that
is exact, validated against this invocation's plan, and participates in the
same candidate preflight and atomic write protocol as global force.

## CLI grammar and validation

The only new syntax is repeatable, flags-first:

```
spine update [--dir D] [--write] [--force-file <repo-relative-managed-path>]... [--force]
```

The shared strict parser remains authoritative: flags must precede any
positionals and `update` accepts no positionals. `--force-file` takes exactly
one value and may occur more than once. Flag order among `--dir`, `--write`,
and repeated `--force-file` values is immaterial; encounter order does not
affect the normalized authorization set or output.

Each supplied path is normalized with the host's `filepath.Clean` rules for
the repository rooted by `--dir`, then compared only as a repository-relative
managed path. An empty value, absolute input, or any raw `..` path component
is rejected before normalization can hide it. `./maipipe.toml` normalizes to
`maipipe.toml`; a trailing separator normalizes normally. Duplicate normalized
values are rejected, including `./maipipe.toml` plus `maipipe.toml`.

After planning but before the local-edit policy or any write, every normalized
value must exactly equal a `FileReport.Path` in *this invocation's* managed
plan. A file absent because this profile does not own it, a `maipipe.toml`
absent because the pack is not planned, and any arbitrary repository file are
unmanaged/unknown and fail. Membership does not authorize marker damage or
other reports lacking regenerable content; those remain existing manual-repair
skips.

All malformed, duplicate, unknown/unmanaged, absolute, and traversal inputs
exit 2, print no plan/report diff, and write no file. CLI errors are
deterministic and include the normalized target when normalization succeeded:

```
update: --force-file "<value>" must name a managed file in this update plan
update: duplicate --force-file "<normalized-path>"
update: --force-file "<value>" must be repository-relative and must not contain ".."
```

## Exact authority semantics

There are two modes, deliberately not a union:

1. No `--force`: an unrecognized local edit remains skipped, except where an
   exact normalized `--force-file` target authorizes that one report.
2. `--force`: retain today's global behavior exactly, including forceable
   legacy-preserved files and every forceable unrecognized report.

`--force` combined with any `--force-file` is invalid and exits 2 before
planning or writes:

```
update: --force cannot be combined with --force-file; choose one overwrite authority
```

This explicit fail-closed rule avoids treating an otherwise global command as
a misleading scoped operation. A `--force-file` never widens authority beyond
the exact normalized `FileReport.Path`, never authorizes a report with empty
`newContent`, and never overrides the existing damaged-marker protection.

For a successfully scoped forced report, the plan prints to stdout immediately
before its diff (or before its `updated:` line under `--write`):

```
<path>: local edits will be overwritten (authorized by --force-file <path>)
```

For an unselected skipped report in scoped-force mode, stderr uses the exact
safer remedy:

```
skipped <path> — unrecognized local edits (use --force-file <path> to drop only this file, or --force to drop all):
```

Existing standalone `--force` plan text and behavior stay byte-compatible;
the changed remediation wording is emitted only when one or more
`--force-file` flags were supplied. Selecting an already clean managed plan
member is valid but grants no effective overwrite and prints no authority
line.

## Planning, preflight, and atomicity

Build every report first, normalize and validate all scoped inputs, then apply
the selected force policy to the complete report set. A scoped forced
`maipipe.toml` becomes a normal `Pending` candidate and therefore receives the
existing maipipe validation during the plan pass. The renderer/preflight
sequence, `FileReport` diff, and `fsutil.WriteFileAtomic` writes are not
forked for scoped force.

If any candidate has an I096/I104 refusal, `--write` aborts before the first
write, including when another selected file was otherwise safe. In particular,
a selected tampered `maipipe.toml` that fails candidate validation leaves that
file, `WORKFLOW.md`, and every other planned file byte-identical. Dry-runs
remain write-free. Scoped authority is permission to transition only one
eligible skipped report to the existing candidate path, not permission to
bypass candidate validation or whole-plan atomicity.

## Requirements attack and resolutions

| Attack | Resolution |
|---|---|
| A raw `a/../maipipe.toml` could normalize inside the repo while hiding traversal intent. | Reject any raw `..` element before `Clean`; normalize only safe relative values. |
| `./maipipe.toml` and `maipipe.toml` can authorize the same file twice. | De-duplicate after normalization and fail before policy/writes. |
| Validating against all known templates could authorize a file this profile/plan does not manage. | Validate against this invocation's `FileReport.Path` set only. |
| A scoped forced maipipe region could dodge preflight if force is applied after candidate selection. | Apply authorization before the existing pending-candidate preflight loop. |
| Combining global and scoped flags would obscure whether the list limits global force. | Reject mixed mode with the exact fail-closed error. |
| A damaged region could be force-written merely because its path is managed. | Regenerable-content/manual-repair guard remains stronger than any authority flag. |
| A valid selected clean file could become a spurious update. | It is valid membership but does not create content, a diff, or an authority note. |

## Scope and compatibility

Change only the update options/policy, update CLI flag grammar and report
wording, and their tests. Do not change template ownership, the gate-pack
render, maipipe grammar, marker recovery, `--force` behavior when used alone,
or atomic write mechanics. This design composes with I123: advisories remain
plan information, and neither feature grants an implicit disable or bypasses
preflight.
