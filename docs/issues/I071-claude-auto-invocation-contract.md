---
id: I071
title: claude-auto invocation contract — selecting model and effort per dispatch
severity: med
status: fixed
affects: [workflow, fleet]
blocked-by: []
labels: [wayfinder:research]
parent: I066
assignee: codex-team
---

## Question

Driving a non-Anthropic model through Claude Code changes the invocation
itself: dispatches will invoke `claude-auto` with arguments rather than bare
`claude`. Pin down the contract: which arguments/env select the model and the
effort per dispatch, per gateway path in use (work custom gateway, local
OpenAI-compatible endpoints)? What must the dispatch transports — cmux/herdr
claude-team skills, plain SDD Agent-tool dispatches — pass through so that the
declared (harness, model, effort) of
[I069](I069-attribution-declare-then-confirm.md) is exactly what the
invocation runs? Deliverable: the invocation matrix + the list of skill/config
touchpoints that must change.

## Resolution

### Requirements attack: premises that cannot be silently combined

1. **The requested `claude-auto` contract is not observable on this host.**
   `command -v claude-auto` produced no path, and a repository-wide search of
   the inspected skills found only this ticket/handoff's references.  The
   handoff identifies it as an owner work-laptop wrapper.  Its executable
   name, accepted arguments, environment handling, and gateway mapping are
   therefore unknown here.  *Resolution:* this answer establishes the stock
   Claude Code contract and marks every work-laptop `claude-auto` matrix cell
   **OWNER VERIFY**; it does not invent wrapper-compatible arguments.
2. **“Local OpenAI-compatible endpoint” is not, by itself, a Claude Code
   transport.** Claude Code's LLM-gateway contract requires an Anthropic
   Messages (`/v1/messages` and `/v1/messages/count_tokens`), Bedrock, or
   Vertex API shape—not the OpenAI API shape.  *Resolution:* a local endpoint
   needs a verified adapter/gateway that implements one of those shapes before
   it can appear in a dispatch contract.  Do not point stock
   `ANTHROPIC_BASE_URL` directly at a raw OpenAI-compatible server.
3. **“Exactly what the invocation runs” is stronger than selecting startup
   values.** A `model` setting is an initial selection, not enforcement; a
   user can switch models unless an allowlist/enforcement policy is in place.
   Also, `CLAUDE_CODE_EFFORT_LEVEL` overrides the launch flag, and a gateway
   can reject or reinterpret an effort parameter.  *Resolution:* dispatches
   must pass explicit per-session model and effort inputs, keep the effective
   environment auditable, and record post-launch evidence.  Enforcement and
   host-config schema belong to I072; gateway acceptance remains an owner
   verification, not a claim made by the dispatcher.
4. **Unknown model IDs may disable effort features.** Claude Code normally
   recognizes effort capability from known ID patterns.  The official gateway
   guidance supplies `CLAUDE_CODE_ALWAYS_ENABLE_EFFORT=1` only when a custom
   model is known to accept it; this is not a safe universal default.
   *Resolution:* do not add that variable speculatively.  The owner must prove
   the selected gateway/model receives and accepts the requested effort level.
5. **The requested Claude model and provider-facing model ID can differ.**
   `modelOverrides` maps an Anthropic model ID to a provider-specific ID, and a
   managed mapping may not be visible in the user settings file. *Resolution:*
   record both the requested `$MODEL` and the effective provider-facing ID,
   plus redacted mapping provenance. If the effective mapping cannot be read
   or observed at the provider, it is **unconfirmable / OWNER VERIFY**—never
   assumed equal to `$MODEL`.

### Evidence collected (2026-08-11; secrets omitted)

On this host, the installed executable is `/opt/homebrew/bin/claude`, a symlink
to `/opt/homebrew/Caskroom/claude-code@latest/2.1.227/claude` (Mach-O arm64).
The following commands and relevant output establish the stock contract:

```console
$ command -v claude; command -v claude-auto
/opt/homebrew/bin/claude

$ claude --version
2.1.227 (Claude Code)

$ claude --help
  --effort <level>  Effort level for the current session
                    (low, medium, high, xhigh, max)
  --model <model>   Model for the current session. Provide an alias ...
                    or a model's full name ...
```

```console
$ jq -r '"settings.schema=" + (."$schema" // "<absent>"),
  "settings.model=" + (.model // "<absent>"),
  "settings.effortLevel=" + (.effortLevel // "<absent>"),
  "settings.has.env=" + (has("env")|tostring),
  "settings.env.keys=" + (if has("env") then (.env | keys | join(",")) else "<absent>" end),
  "settings.has.modelOverrides=" + (has("modelOverrides")|tostring),
  "settings.modelOverrides.keys=" + (if has("modelOverrides") then (.modelOverrides | keys | join(",")) else "<absent>" end)' \
  ~/.claude/settings.json
settings.schema=<absent>
settings.model=claude-fable-5[1m]
settings.effortLevel=high
settings.has.env=true
settings.env.keys=
settings.has.modelOverrides=false
settings.modelOverrides.keys=<absent>

$ printenv | cut -d= -f1 | rg '^(ANTHROPIC|CLAUDE|AWS_|VERTEX|GOOGLE_)' | sort
# no matching names

$ spine model --dir /private/tmp/spine-i071 claude primary
claude-fable-5
$ spine model --dir /private/tmp/spine-i071 -effort claude primary
high
```

The installed **user** settings file demonstrably contains `model`,
`effortLevel`, and `env` (`has("env")=true`), and it has no
`modelOverrides` key. No project or local settings file exists in this
worktree. That does not prove an absence of managed settings or a managed
`modelOverrides` mapping, so dispatch evidence must still capture effective
settings-source provenance and the provider-observed ID where available.

The [official model configuration](https://code.claude.com/docs/en/model-config)
states that startup model precedence is `--model`, then `ANTHROPIC_MODEL`, then
the `model` setting; it also documents `--effort`,
`CLAUDE_CODE_EFFORT_LEVEL`, and `effortLevel` (the environment variable has
the highest effort precedence).  The [official settings reference](https://code.claude.com/docs/en/settings)
documents `model`, `effortLevel`, `env`, `modelOverrides`,
`availableModels`, and `enforceAvailableModels`.  The [official environment
reference](https://code.claude.com/docs/en/env-vars) defines
`ANTHROPIC_BASE_URL` as the endpoint override, `ANTHROPIC_AUTH_TOKEN` as the
bearer token value, and `CLAUDE_CODE_ALWAYS_ENABLE_EFFORT` for known compatible
custom models.  The [official LLM-gateway contract](https://code.claude.com/docs/en/llm-gateway)
defines the accepted API shapes and the optional gateway model discovery.

### Stock Claude Code invocation contract

For a dispatch whose declared triple has already been resolved by `spine`, the
stock executable's explicit startup contract is:

```console
claude --permission-mode auto --model "$MODEL" --effort "$EFFORT" "$PROMPT"
```

- `$MODEL` is the exact model string resolved for the declared `claude` tier
  (for example, `spine model --dir <sdd-cwd> claude <tier>`), not an implicit
  user-setting default. `ANTHROPIC_BASE_URL` chooses the request destination;
  it does **not** choose the answering model.
- `$MODEL` is the requested Claude Code model, not automatically the
  provider-facing ID. `modelOverrides` maps Anthropic model IDs to
  provider-specific IDs. If an applicable effective mapping is known, record
  its output as `$EFFECTIVE_PROVIDER_MODEL`; if none applies, it is `$MODEL`.
  Record non-secret provenance as `modelOverrides: applied|none|unavailable`,
  its settings scope/source (`user`, `project`, `local`, `managed`, or
  `unknown`), and `$EFFECTIVE_PROVIDER_MODEL`. The current user file reports
  `none`, but managed settings are not observable here: without an observable
  source or provider/gateway confirmation, the effective provider ID is
  **unconfirmable / OWNER VERIFY**, not presumed to be `$MODEL`.
- `$EFFORT` is the exact effort string resolved for that same tier
  (`spine model --dir <sdd-cwd> -effort claude <tier>`). Before launch, the
  controller must detect and either unset or deliberately record
  `CLAUDE_CODE_EFFORT_LEVEL`, because it overrides `--effort`. `max` may be
  session-only depending on the active model; an unsupported level can be
  reduced by Claude Code, so the post-launch display/status is evidence, not
  the command line alone.
- For an Anthropic-Messages gateway, provide the endpoint and authentication
  through a protected environment/settings source (never paste a token into a
  dispatch prompt or ledger): `ANTHROPIC_BASE_URL` and, where the gateway
  requires bearer auth, `ANTHROPIC_AUTH_TOKEN`. The gateway must preserve the
  required `anthropic-beta` and `anthropic-version` headers. Optional gateway
  model discovery is `CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY=1`; discovery
  does not prove a model/effort pair is runnable.
- Record the executable path/version; redacted environment *names* and endpoint
  host; declared model; requested `$MODEL`; effective provider model ID (when
  observed); redacted `modelOverrides` provenance; requested/effective effort;
  complete non-secret argv; and the resulting `/status`/model-and-effort
  display. `/status` identifies setting sources but does not name the source
  of each setting, so it cannot by itself prove an effective mapping. This is
  evidence that can connect I069's declaration to a later transcript audit;
  it is not a substitute for the gateway's own model-selection logs.

### Invocation matrix

`OWNER VERIFY` means a work-laptop observation is mandatory before a transport
may claim the cell. The cells are deliberately not filled with plausible
`claude-auto` syntax.

| Harness/path | Harness invocation | Requested model selection | Effective provider model / mapping | Effort selection |
| --- | --- | --- | --- | --- |
| Stock `claude`, first-party or supported provider | **Verified:** invoke `claude --permission-mode auto --model "$MODEL" --effort "$EFFORT" "$PROMPT"`. | **Verified:** `--model "$MODEL"`; it wins over `ANTHROPIC_MODEL` and `settings.model` for startup. | **Verified semantics:** an applicable `modelOverrides` entry can map `$MODEL` to `$EFFECTIVE_PROVIDER_MODEL`. This host's user file has none; a managed mapping is **unconfirmable / OWNER VERIFY** until its source or provider-observed ID is available. | **Verified:** `--effort "$EFFORT"`; first ensure `CLAUDE_CODE_EFFORT_LEVEL` is absent/recorded because that environment variable wins. |
| Stock `claude` through an Anthropic-Messages gateway | **Verified contract, endpoint untested here:** same argv, with a protected launch environment containing the gateway's `ANTHROPIC_BASE_URL` and appropriate auth. | **Verified Claude Code surface:** same `--model`; the gateway must accept that exact ID. `ANTHROPIC_BASE_URL` is transport only. | **Verified semantics; endpoint-specific result unverified:** record the effective ID and `modelOverrides` provenance. If managed mapping cannot be inspected or the gateway cannot show the received ID, it is **unconfirmable / OWNER VERIFY**. | **Verified Claude Code surface; gateway acceptance unverified:** same `--effort`; use `CLAUDE_CODE_ALWAYS_ENABLE_EFFORT=1` only after the owner proves the custom model accepts it. |
| Stock `claude` against a raw local OpenAI-compatible endpoint | **Not a supported stock invocation.** An Anthropic-Messages/Bedrock/Vertex adapter is a prerequisite. | N/A until an adapter supplies and documents an accepted model ID. | N/A until an adapter documents the provider-facing mapping. | N/A until the adapter/model proves effort behavior. |
| Plain SDD Agent-tool subagent | **Verified installed-artifact boundary:** this is an Agent-tool dispatch, not a `claude` argv invocation. | **Verified installed template:** it requires an explicit `model` field. | **OWNER VERIFY:** the installed templates do not show whether CLI settings or `modelOverrides` are applied, nor an effective provider ID. Record platform/settings provenance and provider-observed ID; do not infer a CLI mapping. | **Verified stock Claude Code sources:** skill/subagent Markdown frontmatter may set `effort`, and `CLAUDE_CODE_EFFORT_LEVEL` overrides it. **OWNER VERIFY:** the installed Agent-tool templates expose no effort field, and whether the platform accepts/propagates either stock source is unknown. Do not invent an Agent-tool effort parameter. |
| Work-laptop `claude-auto` through its custom gateway | **OWNER VERIFY:** exact executable path, argv grammar, whether it delegates to stock `claude`, and endpoint/auth source. | **OWNER VERIFY:** does it forward `--model`, rewrite it, select by another flag/env, or override it? | **OWNER VERIFY:** wrapper/config mapping, mapping provenance, and gateway-observed effective provider ID. | **OWNER VERIFY:** does it forward `--effort`; does any wrapper environment override it; and does the selected model accept the resulting request? |
| Work-laptop `claude-auto` toward a local OpenAI-compatible endpoint | **OWNER VERIFY:** exact wrapper/adapter command. A raw OpenAI endpoint is not enough for the stock contract. | **OWNER VERIFY:** adapter-visible requested-model mapping and any wrapper default/override. | **OWNER VERIFY:** adapter-visible effective provider ID and mapping provenance; no mapping may be inferred from the stock CLI. | **OWNER VERIFY:** wrapper-to-adapter effort mapping/acceptance; no contract may be inferred from the stock flag. |

### Dispatch touchpoints and required follow-up

| Exact touchpoint | Current behavior | What must be made host/harness-aware |
| --- | --- | --- |
| `/Users/ldh/Projects/github.com/deepthought/skills/claude-team/SKILL.md` — shared preflight; cmux dispatch at line 58 | Preflight checks `command -v claude`; cmux sends a shell command beginning `claude --permission-mode auto --model <model> --effort <effort>`. | Resolve a host-declared executable/argv contract before the shell command is composed. Preserve the already explicit model and effort values. Do not replace `claude` with `claude-auto` until its owner-verified grammar exists. |
| Same file — herdr lead launch at line 92 | `herdr agent start ... --kind claude ... -- --permission-mode auto --model ... --effort ...`. | Verify whether `--kind claude` can select a wrapper or only the stock binary; if it cannot, add an owner-approved herdr transport/config capability rather than assuming trailing args change the executable. |
| Same file — herdr worker launch at line 114 | The worker uses `--kind claude` with explicit `--model` and `--effort`. | Apply the same verified executable-selection mechanism as the lead, while passing model and effort separately and retaining the fresh-process rule. |
| `/Users/ldh/Projects/github.com/deepthought/skills/lib/frontend-preflight.sh` | Validates the `claude` herdr integration after frontend detection; it does not select an executable. | Keep this integration check, and add a distinct wrapper/capability check only once the owner defines what `claude-auto` is and how herdr exposes it. |
| `/Users/ldh/Projects/github.com/deepthought/skills/lib/test-no-hardcoded-models.sh` | Guards that claude-team resolves IDs through `spine model`; it passed here (15 pass, 0 fail). | Extend only after the host config is specified: test the selected transport receives both resolved values and no bare/default model path is introduced. Do not hard-code wrapper syntax in the test. |
| Installed plain SDD transport: `/Users/ldh/.claude/plugins/cache/claude-plugins-official/superpowers/6.2.0/skills/subagent-driven-development/SKILL.md` and its `implementer-prompt.md` / `task-reviewer-prompt.md` | The templates require an explicit Agent-tool `model`; they specify no CLI executable, gateway, or effort parameter. | Keep the explicit model field. Stock Claude Code documents `effort` in skill/subagent frontmatter and `CLAUDE_CODE_EFFORT_LEVEL` precedence, but whether this platform propagates either source—and whether it applies `modelOverrides`—is **OWNER VERIFY**. The Agent matrix row is the binding boundary; do not invent an Agent parameter. |
| Host Claude settings: `~/.claude/settings.json`; future host capability config in I072 | This host's observable user defaults are `model=claude-fable-5[1m]`, `effortLevel=high`, `env` present but empty, and no user `modelOverrides`. | Keep gateway URL/token out of versioned dispatch artifacts. Record requested/effective provider IDs plus redacted mapping provenance. I072 must define the authoritative host capability/config source, precedence, validation, and how a dispatch records the resolved harness executable and non-secret configuration provenance; until then a managed mapping is **unconfirmable / OWNER VERIFY**. |

`sh /Users/ldh/Projects/github.com/deepthought/skills/lib/test-frontend-preflight.sh`
also passed (16 pass, 0 fail). These checks validate the currently inspected
skill wiring; they do not validate an absent `claude-auto` wrapper.
