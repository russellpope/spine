# spine

A single Go binary that installs and enforces one development workflow across every repo you point it at.

spine scaffolds a standardized set of workflow files into a repository, regenerates the machine-owned parts of those files when the templates evolve, and audits whether the work that happened actually followed the rules the files declare. It never calls a model API and it never runs your agents. The agent harnesses (Claude Code, Codex, pi) do the work; spine owns the paperwork and checks the receipts.

The bet behind it: agentic development goes wrong quietly. An agent hand-edits a progress file, a "quick fix" runs on a cheaper model than the ticket declared, a review gets skipped because nothing noticed. spine makes those failures loud. Machine-owned regions have exactly one legal writer, model routing is declared in tiers and audited against real transcripts after the fact, and stage progression lives in a cursor block that only `spine cursor` may touch.

## Install

Requires Go 1.26 and `git` on PATH. The module has zero dependencies outside the standard library, and all templates compile into the binary.

```sh
git clone https://github.com/russellpope/spine
cd spine
make install        # builds to ~/bin/spine
```

Or build in place with `make build` (output in `bin/spine`). Verify with:

```sh
spine version       # prints the compiled template generation
spine doctor        # read-only health checks against the current repo
```

## Quick start

In a new repo:

```sh
spine init          # scaffold the workflow files
```

In a repo that predates spine:

```sh
spine adopt         # dry-run: shows what it would write
spine adopt --write
```

`init` writes `WORKFLOW.md`, `CLAUDE.md`, `AGENTS.md`, the `docs/` scaffolding described below, and stamps a `maikanban.repositorySlug` git config entry as the repo's fleet identity. When the shipped templates change generation, `spine update --write` regenerates the machine-owned regions in every managed file while preserving anything you deliberately overrode.

## Commands

| Command | What it does |
|---|---|
| `init` | Scaffold the unified workflow into a repo |
| `adopt` | Retrofit a pre-spine repo (dry-run by default; `--write` applies) |
| `update` | Regenerate machine-owned workflow files (dry-run by default; `--write` applies) |
| `adr` | Manage architecture decision records (`new`, `list`) |
| `handoff` | Manage `docs/handoffs` (`new`, `list`, `latest [--fleet DIR]`); `new` embeds the current cursor block and the newest checkpoint |
| `eval` | Manage `docs/evals` (`new`, `add-run`, `list`) |
| `doctor` | Read-only workflow health checks |
| `audit` | Verify declared model routing (`routing`) or stage cursor derivation (`stages`) against on-disk artifacts |
| `gate` | Run a gate-pack check class (`gate <pack>[@<v>] <check> [--dir D]`) |
| `checkpoint` | Write or replay a session checkpoint (`new`, `latest`, `list`) |
| `cursor` | Print or update the stage cursor (`start`, `tick`, `here`, `set`) |
| `model` | Resolve the model table for a (harness, tier) pair (read-only; flags precede the positionals) |
| `version` | Print the compiled template generation |

## The workflow it installs

Every managed repo gets the same shape:

- `WORKFLOW.md` declares the profile, the stage sequence (`grill → prd → issues → implement → functional-test → review → verify → ship → deploy → docs → handoff`), the mandatory gates (`grill` and `verify`), and the model routing table.
- `CLAUDE.md` and `AGENTS.md` are twins. Each carries a spine-owned region between `<!-- spine:begin -->` markers, so Claude Code and Codex read the same contract. Your own notes outside the markers survive every update.
- `docs/issues/` is the issue ledger: one markdown file per ticket with frontmatter (`id`, `title`, `severity`, `status`, `affects`, `blocked-by`, plus optional routing fields). The files are canonical; there is no separate tracker.
- `docs/specs/` holds paired `<date>-<topic>-design.md` and `-plan.md` files. `docs/adr/` holds numbered decision records. `docs/handoffs/` holds session-end briefs. `docs/evals/` holds model evaluation runs. `docs/remediation/` holds bounded fix-it rounds with a three-round budget.
- The stage cursor is a `<!-- spine:cursor -->` block at the head of `.superpowers/sdd/progress.md`. Only `spine cursor` may write it. `spine audit stages` treats a valid-looking block in non-canonical form as evidence of hand-editing and fails.
- The block's delimiters are **fences**, and a fence counts only when it is the whole line starting at column 0. A delimiter quoted mid-sentence — as in the line above — is prose and is skipped, so documents can explain the convention freely. To show a *complete* block in a document, indent the example rather than fencing it; `WORKFLOW.md`'s grammar reference does exactly that. A document carrying two open fences is refused outright, naming both line numbers, rather than silently parsing one of them.

## Model routing

Tickets and plans name tiers, never model ids. The four tiers are `primary`, `routine`, `mechanical`, and `fallback`, mapped per harness (`claude`, `codex`, `openweights`, `pi`) in the `model_routing:` block of each repo's `WORKFLOW.md`. Shipped defaults live in `models/defaults.json` and compile into the binary.

`openweights` (added 2026-08-25, I110) maps every tier to an open-weights model served through a gateway, each at effort `high`, with `fallback` deliberately equal to `primary` so a refusal re-run cannot silently leave open weights. Note that it is the one first-axis value that is **not** its own execution vehicle — those dispatches run the ordinary Claude Code binary through a wrapper that passes `--model` through. See CONTEXT.md's `harness` entry for why that distinction matters and what it currently costs.

Two rules make this workable:

1. Escalation is free with a record. Moving a ticket up or down a tier requires an `ESCALATION` or `FALLBACK` line in the progress ledger. Running below the declared tier without one ("silent descent") is a blocking audit failure.
2. Overrides survive updates. spine keeps a history of every default it has ever shipped, so `spine update` can tell an inherited default (refresh it) from a deliberate per-repo override (keep it).

`spine audit routing` closes the loop. It reads the actual agent transcripts on disk, reconstructs which model ran which ticket, including subagent dispatches, and checks the result against the declared tiers. Nothing is taken on the agent's word.

## Gate packs

`spine gate` runs deterministic code checks as versioned packs (currently `go@1`). Each check class ships with a positive control pair: a known-good input it must pass and a seeded violation it must catch. A check without both is not shippable. Exit codes are fixed: 0 pass, 1 findings, 2 misconfiguration.

Gate stages are designed to run under [maipipe](https://github.com/russellpope/maipipe), a local CI daemon, and write structured results to `$MAIPIPE_RESULTS`. That integration is optional. With no `gate_pack` configured, spine never needs maipipe at all; with one configured, `spine update` needs `maipipe` on PATH to validate the rendered pipeline region, and skips that file otherwise.

## Companion tools

spine works alone, but it was built as one corner of a triangle:

- [maipipe](https://github.com/russellpope/maipipe) runs the pipelines and owns the deterministic merge gate.
- [maikanban](https://github.com/russellpope/maikanban) renders `docs/issues/` ledgers from many repos as a terminal kanban board.

The coupling is thin by design. maikanban only needs the ledger file convention and the `maikanban.repositorySlug` git config that `spine init` stamps. maipipe only meets spine at the managed region inside `maipipe.toml`.

## Caveats

- `spine audit routing` reads transcripts where the harnesses write them: `~/.claude/projects/<slug>` for Claude Code and `~/.codex/sessions` (or `$CODEX_HOME`) for Codex. Those formats are undocumented upstream and drift; the audit tracks them but may lag a harness release.
- **Known gap:** the audit derives the harness from the *transcript source*, so `openweights` dispatches — which run Claude Code and therefore land in the Claude transcript layout — are currently judged against the `claude` tier table, where their model ids do not appear. Until **I111** lands, treat routing verdicts on open-weights runs as unreliable. Every other harness is unaffected.
- The workflow templates assume the superpowers plugin's `.superpowers/sdd/` layout for progress files and checkpoints.
- Developed and used on macOS; the code is portable Go with no cgo, but other platforms get little testing.

## Development

```sh
make build      # go build -o bin/spine ./cmd/spine
make test       # go test ./...
```

Behavior changes visible to consuming repos are recorded in `CHANGELOG.md`, referencing the ticket or ADR that carries the detail. Design history lives in `docs/adr/`; domain vocabulary in `CONTEXT.md`.

## License

No license has been chosen yet, so default copyright applies. If you want to use spine and that matters to you, open an issue.
