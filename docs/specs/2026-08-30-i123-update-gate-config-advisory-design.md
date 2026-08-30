# I123 — `spine update` gate-configuration advisory

**Status:** Accepted design

**Ticket:** `I123`

## Problem and outcome

`spine update` currently renders every enabled class in a known pinned gate
pack even when a class will later exit 2 because its required
`gate_pack_config` input is empty. The render is deliberate: ADR 0015 only
permits `gate_pack_disabled` to omit a class. The update command must instead
tell the owner, before any write, which enabled classes are misconfigured and
how to resolve each one. It must not change the rendered stage, silently add a
disable, or turn an advisory into a command failure.

## Binding contract

### Source of truth

The required-input relation belongs beside the frozen gate-pack metadata used
by `internal/update/gatepack.go`, not in a second CLI list and not in the
rendered TOML. For the shipped `go@1` pack, an enabled class is
advisory-worthy only when its metadata marks its input as required:

| Class | Required `WORKFLOW.md` input |
|---|---|
| `fixture-manifest` | `gate_pack_config.fixture_manifest` |
| `gitignore-control` | `gate_pack_config.build_outputs` |
| `n-plus-one` | `gate_pack_config.n_plus_one_clients` |
| `test-enum-vs-spec` | `gate_pack_config.test_enum_spec` |

`gateCheckConfig` may still describe optional environment rendering, but it is
not itself the requiredness authority. `tskip` consumes the optional
`tskip_allow` environment value; an empty allowlist is valid and never
advisory-worthy. Every class absent from the required-input metadata is
config-free for this feature, including `deferred-cleanup-errcheck`,
`dead-code-callgraph`, `binary-hygiene`, and `mutate`.

An advisory exists iff all of these are true: the pack is shipped and pinned,
the class is in that pin's frozen class list, it is not named in
`gate_pack_disabled`, the class has a required key, and the extracted value is
empty after the existing `ExtractKeys` treatment. Unknown packs retain their
existing unrecognized-edit report and produce no guessed advisory.

### Text, ordering, and exits

One missing class yields exactly one stdout line:

```
advisory: enabled gate class "<class>" lacks required gate_pack_config.<key>; configure gate_pack_config.<key> or add "<class>" to gate_pack_disabled
```

The list is sorted by class bytewise ascending; the key follows its class from
the table above. Therefore the all-missing `go@1` case is printed in this exact
order: `fixture-manifest`, `gitignore-control`, `n-plus-one`,
`test-enum-vs-spec`. No map iteration may determine either ordering or text.

The lines are advisory output on stdout, never stderr. They do not increment
the update command's outstanding count and do not alter its exit contract:
dry-run with another pending file remains exit 1; a dry-run with only these
advisories and up-to-date files exits 0; a successful `--write` exits 0; an
existing planning/preflight/refusal error keeps its existing exit 2. A missing
configuration never causes `update` to add a disabled class, omit its stage,
or suppress that stage's existing runtime exit-2 behavior.

### Timing and write semantics

For `spine update --write`, collect the advisories during the same plan pass
that derives `maipipe.toml`, then run all existing candidate preflights. Emit
the sorted advisory list after that planning/preflight work and before the
first `fsutil.WriteFileAtomic` call. Emit it even if a later whole-plan
preflight refusal prevents all writes. This makes the dry-run review surface
and write execution say the same thing while retaining I096/I104's candidate
validation and whole-plan no-write refusal.

For a dry run, emit the same lines before the report/diff stream. No advisory
calculation writes, changes `FileReport.State`, changes a diff, or changes
`maipipe.toml` bytes. Existing configured plans and writes must be
byte-identical to their current behavior, except for the absence/presence of
the advisory lines themselves.

### One-key / one-disable behavior

Setting one required key removes only that class's advisory. Adding one class
to `gate_pack_disabled` removes only that class's advisory and retains the
existing deliberate rendering effect of omitting that class. The other enabled
missing classes remain rendered and advised. An empty optional `tskip_allow`
is not repaired, synthesized, or reported.

## Requirements attack and resolutions

| Attack | Resolution |
|---|---|
| Treating every `gateCheckConfig` entry as required would incorrectly warn on an empty `tskip_allow`. | Requiredness is explicit metadata, separate from environment consumption. |
| Printing after `Run` returns would mean a `--write` can replace a file before the alleged pre-write notice. | `Run` gains a narrow pre-write reporting seam invoked after preflight and before writes; the CLI uses it for `--write`. |
| A map-backed output could flicker in tests and owner review. | Collect value objects and sort by class before formatting. |
| Omitting a bad stage would hide the misconfiguration but violates ADR 0015 and I123. | The renderer is unchanged; only plan output changes. |
| A stale/unknown pack might be diagnosed using a later binary's registry. | Only a shipped pinned pack uses its frozen `PackClassesFor` result; unknown packs keep their present error path. |
| A refusal might hide useful advice. | Advice is emitted before the whole-plan refusal decision and still no file writes. |

## Scope and compatibility

Modify only `internal/update` planning metadata/reporting and the `spine
update` presentation seam, with focused unit and CLI coverage. Do not alter
gate implementation, template defaults, WORKFLOW schema, disabled-class
semantics, maipipe syntax, or any exit code. This design supplements the
I104/I097 preflight design: maipipe remains grammar authority and the existing
atomic candidate plan remains intact.
