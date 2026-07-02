# ADR 0001 — Target LM Studio as the first data source

Status: Accepted
Date: 2026-06-28

## Context
`hbmview` needs per-component memory data for locally-running LLMs (weights, KV cache,
context, supporting buffers). The OS exposes only a process's total RSS, which is
insufficient for the semantic breakdown that is the product's reason to exist.
The user primarily runs models via **LM Studio** (also uses Ollama). LM Studio ships a
CLI (`lms`), an OpenAI-compatible REST API, a native `/api/v0` REST surface, and a
TS/Python SDK.

## Decision
Target **LM Studio** as `hbmview`'s first and primary data source. Consume its `lms`
CLI JSON output to start (`lms ls --json`, `lms ps --json`, `lms load --estimate-only`),
keeping room to add the SDK/REST surfaces if they expose richer live detail.

## Consequences
- LM Studio reports **totals only**, not a breakdown — see ADR-0002 (derivation strategy).
  `hbmview` must compute the weights/KV/context split itself.
- Apple-Silicon unified memory is the primary target environment; GPU ≈ Total memory.
- Other runtimes (Ollama, llama.cpp) are out of scope for v1 but the data layer should
  not hard-code LM Studio assumptions so a `Source` abstraction can be added later.
- Verified capabilities (2026-06-28): `lms ls --json` gives weights/arch/quant/maxctx;
  `lms load --estimate-only -c CTX` gives total memory at a context (delta across
  contexts isolates KV-cache cost); `lms ps --json` lists loaded instances.
