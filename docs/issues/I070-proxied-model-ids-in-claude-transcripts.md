---
id: I070
title: What do proxied 3rd-party model ids look like in Claude Code transcripts?
severity: med
status: fixed
affects: [audit, model]
blocked-by: []
labels: [wayfinder:research]
parent: I066
assignee: codex-team
---

## Question

`spine audit routing` reads per-turn model ids from Claude/Codex session
formats. When Claude Code drives a 3rd-party or local model through a gateway
(work-laptop custom gateway → GPT; OpenAI-compatible endpoint → open weights),
what id strings actually land in the transcript — the upstream model's real id,
the gateway's alias, or something mangled? Survey real transcripts from each
gateway path in use. Does the resolver/table need alias rows mapping
transcript-observed ids to canonical model ids, and is the observed id stable
enough for the confirm half of declare-then-confirm
([I069](I069-attribution-declare-then-confirm.md))?

## Resolution

(2026-08-11, I070 research) **Observed local-proxy result: the selected model
identifier and the persisted assistant-event `message.model` were equal.** The
one controlled local LM Studio run selected `google/gemma-4-12b` and recorded
that same string on both emitted assistant records (a thinking record and a
text record). The evidence does not establish whether Claude Code, LM Studio,
or the protocol response is the component that supplies the persisted value.
It establishes only within-session repeatability for this LM Studio path, not
cross-version/gateway stability.

### Requirements attack

The question and dispatch contain four tensions; the evidence-based resolution
is recorded rather than silently assuming an answer.

1. **"Survey ... each gateway path in use" conflicts with this host's
   availability.** The handoff says the work-laptop `claude-auto` custom-GPT
   gateway is not on this host. Local checks confirmed `claude-auto` is absent,
   no `ANTHROPIC_BASE_URL`/auth override was set before the experiment, and the
   checked local config surfaces had no gateway/base-url/token setting keys.
   *Resolution:* the LM Studio path below is empirical; the work-laptop custom
   gateway remains explicitly unobserved, not generalized from it.
2. **The handoff calls the local lane "OpenAI-compatible," while Claude Code
   uses Anthropic Messages requests.** LM Studio's current official integration
   documents an Anthropic-compatible `POST /v1/messages` endpoint and the
   `ANTHROPIC_BASE_URL` setup for Claude Code. *Resolution:* exercise that
   documented endpoint, not an OpenAI endpoint guessed to be interchangeable.
3. **"Per-turn" does not mean all turns are audit evidence.**
   `internal/audit/audit.go` parses each assistant event's `message.model`, but
   deliberately excludes main-session assistant models from ticket evidence;
   only a linked subagent transcript's actuals are used. *Resolution:* this
   controlled controller session proves field shape; a ticket-shaped subagent
   run is still required to validate the audit correlation path for a gateway.
4. **One successful run cannot prove "stable enough."** A gateway may expose a
   provider id, a configured alias, or change either after a configuration or
   version change. *Resolution:* accept exact equality as a confirmation only
   when it is recorded against the host/gateway configuration; otherwise report
   it unconfirmable rather than inferring identity by substring or family.

### Evidence

All commands below were run 2026-08-11. No credential was printed; the local
LM Studio integration used its documented non-secret placeholder token.

```text
$ claude --version
2.1.227 (Claude Code)

$ claude --help
  --effort <level>                      Effort level for the current session
  --model <model>                       Model for the current session.

$ command -v claude-auto || true

$ printf 'ANTHROPIC_BASE_URL=%s ANTHROPIC_AUTH_TOKEN=%s ANTHROPIC_API_KEY=%s\n' \
  "${ANTHROPIC_BASE_URL:+set}" "${ANTHROPIC_AUTH_TOKEN:+set}" "${ANTHROPIC_API_KEY:+set}"
ANTHROPIC_BASE_URL= ANTHROPIC_AUTH_TOKEN= ANTHROPIC_API_KEY=

$ jq -r 'paths(scalars) | map(tostring) | join(".")' \
  ~/.claude/settings.json | rg -i 'anthropic|base.?url|auth.?token|api.?key|gateway' || true

$ lms server status; lms ls
The server is running on port 1234.
google/gemma-4-12b (1 variant)    12B    gemma4    12.84 GB    Local

$ lms load --estimate-only google/gemma-4-12b
Model: google/gemma-4-12b
Estimated Total Memory: 11.96 GiB

$ lms load google/gemma-4-12b --context-length 32768 --ttl 180 -y
Model loaded successfully ...
To use the model in the API/SDK, use the identifier "google/gemma-4-12b".

$ ANTHROPIC_BASE_URL=http://127.0.0.1:1234 \\
  ANTHROPIC_AUTH_TOKEN=lmstudio \\
  CLAUDE_CODE_ATTRIBUTION_HEADER=0 \\
  claude --bare --tools '' --model google/gemma-4-12b --effort low \\
  -p 'Reply exactly: I070-proxy-smoke.'
I070-proxy-smoke.

$ jq -c 'select(.type == "assistant") |
  {type, cwd: (if .cwd then "present" else "absent" end),
   model: .message.model,
   content_types: [.message.content[].type]}' \\
  ~/.claude/projects/<throwaway-project>/<session>.jsonl
{"type":"assistant","cwd":"present","model":"google/gemma-4-12b","content_types":["thinking"]}
{"type":"assistant","cwd":"present","model":"google/gemma-4-12b","content_types":["text"]}

$ find ~/.claude/projects -type f -name '*.jsonl' -exec \
  jq -r 'select(.type == "assistant") | .message.model // empty' {} \; 2>/dev/null \
  | sort | uniq -c | sort -nr
35946 claude-fable-5
23817 claude-opus-5
16368 claude-sonnet-5
11403 claude-opus-4-8
 1782 claude-haiku-4-5-20251001
  360 claude-sonnet-4-6
  257 claude-opus-4-7
  102 <synthetic>
    2 google/gemma-4-12b

$ sub_file=$(find ~/.claude/projects -type f -path '*/subagents/agent-*.jsonl' | head -n 1)
$ jq -c 'select(.type == "assistant") |
  {type, cwd: (if .cwd then "present" else "absent" end),
   model: .message.model,
   content_types: [.message.content[].type]}' \
  "$sub_file" | sed -n '1p'
{"type":"assistant","cwd":"present","model":"claude-opus-4-8","content_types":["text"]}
```

The exhaustive local Claude transcript model-field survey (content not read)
contained the listed Claude-shaped ids, `<synthetic>`, and exactly these two
`google/gemma-4-12b` records. It contains no evidence for a separate
work-laptop gateway path. A real local subagent sample has the same
`type: "assistant"` / `message.model` shape. Source inspection confirms
the parser behavior, but not a proxied-subagent result:

```text
$ rg -n 'Main-session assistant models are never ticket evidence|message.model is the actual' internal/audit/audit.go
43://     Main-session assistant models are never ticket evidence — inline
1250:// message.model is the actual, and Task/Agent tool_use blocks are dispatch

$ rg -n -C 1 'aliases describe|explicit aliases replace substring containment|Override matches by its exact on-disk id only' internal/model/model.go internal/audit/resolve_test.go
internal/model/model.go:218:// round 1): the table entry's aliases describe the shipped
internal/audit/resolve_test.go:88:// Acceptance (D13): explicit aliases replace substring containment. A
internal/audit/resolve_test.go:288:// Fix round 1, I-2: a deliberate Override matches by its exact on-disk id

$ GOCACHE=/private/tmp/i070-go-build GOMODCACHE=/private/tmp/i070-go-mod go test ./internal/audit ./internal/model
ok   github.com/russellpope/spine/internal/audit    0.647s
ok   github.com/russellpope/spine/internal/model    0.396s
```

The local integration is supported by LM Studio's official
[Claude Code guide](https://lmstudio.ai/docs/integrations/claude-code) and
[Anthropic-compatible endpoint reference](https://lmstudio.ai/docs/developer/anthropic-compat).
This is an interoperability experiment, not an Anthropic-supported deployment:
Anthropic's official [gateway documentation](https://code.claude.com/docs/en/llm-gateway)
states that it does not support routing Claude Code to non-Claude models through
a gateway. That boundary reinforces the need to record observed behavior per
host/gateway rather than infer it from provider identity.

### Decisions for the follow-on design

- **Observed-id shape and stability:** retain the raw `message.model` string
  exactly. It is a non-empty arbitrary identifier (including `/`) and is
  repeatable within this session. Do not normalize it or apply containment
  matching. The current audit already compares only exact ids, declared aliases,
  and exact historical ids; its tests reject substring matching.
- **Alias rows:** **none are warranted in `models/defaults.json` now.** The
  observed LM Studio id equals the selected canonical id, so an alias would add
  no information. Moreover, that file's aliases belong to shipped
  flavor/tier defaults, and `internal/model` deliberately removes them for a
  per-repo override; using it as a host-gateway alias registry would make a
  gateway alias look like a default model and can mask routing drift. This is a
  Spine design inference from the table/audit behavior, not evidence that the
  work-laptop gateway has such an alias. If that gateway returns a stable raw
  id different from its declared canonical id, consider an explicit,
  host-scoped, one-to-one observed-id mapping in the later host-config/audit
  design (I072/I074), with exact-match tests; do not add a global table alias
  speculatively.
- **Declare then confirm:** conditionally viable for **model identity**. A
  dispatch that declares `harness=claude`, canonical model
  `google/gemma-4-12b`, and effort `low` can have its model confirmed when a
  linked subagent's raw `message.model` is exactly that id (or an approved
  host-scoped mapping). That linked-subagent conclusion follows from the parser
  source and has not been observed through this proxy. The transcript field
  does not independently prove the upstream provider behind an alias, nor does
  this evidence prove effort; the declaration remains the authority for those
  claims. Until the work-laptop gateway and a linked subagent are sampled, its
  alias and correlation result must remain **unconfirmable**, not assumed
  matched.

### Owner repro for the unobserved work-laptop gateway

Run the production `claude-auto` invocation that selects a known 3rd-party
model in a throwaway directory, preserving its gateway configuration and
without printing any token. Capture the new
`~/.claude/projects/<escaped-cwd>/<session>.jsonl` and its
`<session>/subagents/agent-*.jsonl` files. For both controller and linked
subagent files, run the redacted-field query above, then correlate the
subagent through its adjacent `agent-*.meta.json` `toolUseId`. Record the
declared model, transcript-observed `message.model`, gateway/version, and
whether the raw strings exactly match. If a dispatch-shaped subagent is present, run
`spine audit routing --transcripts ~/.claude/projects/<escaped-cwd>` against
the ticket repo. A mismatch or a missing subagent is an unconfirmable result
for I069/I074, not license to add an alias row.
