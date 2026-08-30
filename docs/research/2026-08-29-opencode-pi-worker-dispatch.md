# OpenCode and Pi worker dispatch, checked 2026-08-29

I105 asks whether a ladderbench-style `ladder-worker` should steer local-model
worker dispatch toward OpenCode or Pi. This note checks the current upstream
documentation and source, not the 2026-08-20 claim alone.

## Verified facts

OpenCode has a declarative agent record. An `agent.<name>` may set `mode:
subagent`, a fixed `prompt`, a model, and per-agent permissions. The official
agent guide documents JSON and Markdown definitions, including a fixed prompt
file and a read-only Markdown subagent. [OpenCode agents documentation](https://opencode.ai/docs/agents/)

Its `permission.task` rules restrict which named subagents a parent can ask the
Task tool to run. A denied agent is removed from that tool's description. This
does not prevent a human from selecting the same agent with `@`. [Task permission documentation](https://opencode.ai/docs/agents/#task-permissions)

The Task implementation confirms the two inheritance claims that matter here.
When a subagent has no `model`, it takes the parent message's provider and
model, and it also receives the parent variant. When it declares a model, the
child invocation passes no variant. The same code makes a child session with
the parent session ID, agent name, and derived permissions. [Task source](https://github.com/anomalyco/opencode/blob/dev/packages/opencode/src/tool/task.ts)

OpenCode's default child-depth limit is one. Background Task calls require
`OPENCODE_EXPERIMENTAL_BACKGROUND_SUBAGENTS=true`; the implementation creates a
background job and injects its eventual result back into the parent. [Task source](https://github.com/anomalyco/opencode/blob/dev/packages/opencode/src/tool/task.ts)

`opencode serve` exposes a programmatic server. Its published API includes
asynchronous prompting, abort, session status and children, and the `/event`
SSE stream. The live docs place the global stream at `/global/event` and the
instance stream at `/event`. [OpenCode server documentation](https://dev.opencode.ai/docs/server/)

Pi has no built-in declarative subagent schema comparable to OpenCode's
`agent` configuration. Its official repository ships a subagent *example*
extension instead. That extension reads Markdown agent definitions with a body
as the system prompt plus optional `model` and `tools`, then launches one
separate `pi --mode json -p --no-session` process per worker. [Pi subagent agent parser](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/examples/extensions/subagent/agents.ts) and [Pi subagent extension](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/examples/extensions/subagent/index.ts)

The supplied Pi example can pin a model and allowlist tools, stream child
output, run a bounded parallel batch, collect per-child usage, and terminate a
child process when its parent tool call is aborted. When a definition omits its
model, the example forwards the invoking Pi session's model and thinking level;
a model-pinned worker does not take the parent thinking level. The example does
not define a permission policy for which agent names may run, a child-depth
limit, a durable parent-child session link for its ephemeral workers, or a
background-job API. [Pi subagent example README](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/examples/extensions/subagent/README.md)

Pi does provide a usable control plane for an extension or a supervisor. RPC
can prompt, steer, queue follow-ups, abort, inspect session state, set model
and thinking level, and return aggregate token, cost, tool-call, and context
usage. JSON mode emits session, lifecycle, message, tool, queue, and usage
events. These are per Pi process, not an automatic worker tree. [Pi RPC documentation](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/rpc.md) and [Pi JSON event-stream documentation](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/json.md)

## Capability matrix

| Dispatch concern | OpenCode | Pi's shipped equivalent | Consequence |
| --- | --- | --- | --- |
| Fixed worker prompt | Declarative agent `prompt` | Example extension Markdown body | Both can do it. Pi needs the extension installed. |
| Restrict worker tools | Per-agent `permission` and legacy `tools` | Example passes `--tools` allowlist | Both can constrain tools, but OpenCode's rule lives in agent policy. |
| Restrict names a parent may spawn | `permission.task` glob rules | No shipped equivalent | Pi needs custom tool validation. |
| Parent model and reasoning | Inherited only when child has no model | Opt-in example forwards model and thinking when its child is unpinned | Same result, but Pi's policy is extension code. |
| Child depth | `subagent_depth`, default 1 | No shipped limit | Pi needs an extension-owned depth counter. |
| Background workers | Experimental, feature-flagged Task jobs | Parallel child processes only | Pi needs a supervisor/job model for detached work. |
| Prompt, abort, events | Serve API plus SSE | JSON stream and stdin/stdout RPC | Both are controllable. OpenCode is already a server. |
| Session and token evidence | Child sessions, status/children endpoints, event bus | JSONL/RPC stats and sample per-child usage | Pi has the raw data, but no automatic durable worker tree. |

## Implications for spine and maipipe

**Verified repo context.** `spine model pi <tier>` only resolves a model and
effort. The local-harness conventions explicitly leave Pi prompt translation,
worker execution, and session observation to the Pi extension or maipipe.
I102 is separate: it unifies how the audit pairs worker prompts after a worker
has already been spawned. It should not gain a harness-selection rule from this
ticket.

**Inference.** A maipipe or ladderbench instrument that requires one fixed
worker prompt, an allowlisted worker type, depth one, explicit policy, and
externally inspectable sessions should use OpenCode now. The important edge is
not model quality. It is that OpenCode can state and enforce the worker contract
in the checked-in agent configuration.

**Inference.** Pi remains viable when the owner wants to own an extension and
accept its maintenance cost. Before Pi runs the same instrument, that extension
needs immutable project agent definitions, a parent allowlist, a depth counter,
job IDs and cancellation, and a persisted parent-child record with normalized
usage. It already demonstrates model-and-thinking forwarding. Without the
other controls, a Pi worker run is not equivalent evidence.

## Recommendation and owner decision

Do not change spine code or I102 for I105. Use OpenCode for the constrained
`ladder-worker` benchmark and any maipipe worker lane that needs the same
enforcement and observation now. Keep Pi as a model-resolution target, not a
drop-in worker orchestrator.

The owner decision is whether Pi warrants the extension work listed above. If
yes, create a separately scoped Pi-extension ticket with an acceptance test
that compares its persisted worker record to OpenCode's. If not, close I105 by
adopting OpenCode for this lane and leave Pi for interactive or
extension-specific work.

## Scope and source note

This is a documentation and upstream-source check dated 2026-08-29. It does
not claim that a particular local install has these features enabled. In
particular, OpenCode background workers remain experimental, and Pi's
subagent facility above is an official example extension rather than a
declarative core feature.
