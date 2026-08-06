[Relocated 2026-08-06 by I056 — runner now lives in the /model-eval skill (deepthought skills/model-eval/scripts/), specs in local-model-evaluation docs/evals/2026-07-02-govmomi-vsphere-inventory-cli/mutation-specs/]

# Overnight batch — behavioural mutation battery

## Run it

```
./scripts/overnight/run-batch.sh
```

Copies each tree to scratch, runs the battery, writes `results-<timestamp>/combined.txt`.
**The eval corpus is never modified.** Re-runnable; scratch is recreated and removed.

Runtime: ~4–6 min per tree, 5 trees wired → **~25–30 min**, not a full night. Sized by
what has verified specs, not by what would fill the hours.

## Wired and verified (n=5)

All five ran green at baseline and produced the rates below before being queued.

| Spec | Tree | Kill rate |
|---|---|:---:|
| `opus` | claude-code-opus-4.7 | 2/8 = 25% |
| `gpt55` | gpt-5.5 | 2/8 = 25% |
| `laguna` | laguna-s-2.1/vsphere-inventory | 5/8 = 62% |
| `ornith397b` | ornith-1.0-397B | 2/8 = 25% |
| `qwen36-27b` | qwen-3.6-27b | 0/8 = 0% |

Because every rate is already known, **the batch's value is regression-detection and
reproducibility, not discovery.** It proves the numbers in the research doc are
reproducible from a clean checkout by someone who was not in the session.

## NOT wired — and this is the finding, not an oversight

Three buildable trees have **no spec**, because mutation sites are hand-authored per
tree and each costs ~8 source lookups:

| Tree | Go files | Blocker |
|---|---|---|
| `orinth-1.0-35b-fp16/govmomi-cli` | 21 | sites not authored |
| `qwen-agentworld-35b-a3b/govmomi-cli-eval-prompt` | 7 | sites not authored |
| `gemma-4-31b` | 17 | sites not authored |

`scripts/overnight/sites.sh` (candidate discovery by grep) gets ~80% of the way — it
locates the classifier, portgroup gate, headers, sorts, RAM math, TLS construction and
logout in an unfamiliar tree — but the exact literal strings still need a human or agent
pass, because `mutate.py` does literal replacement and a near-miss silently reports
`NO-SITE`.

**This is the hinge the grill must resolve.** At ~8 lookups per tree, the battery is a
manual instrument. Unattended 24×7 operation needs site discovery to be automatic —
most likely AST-based (`go/ast` for Go) rather than grep-based, and per-language. Until
then "queue it overnight" means "queue the trees someone already prepared."

## Adding a tree

1. `./scripts/overnight/sites.sh <tree-relative-path>` → candidate lines
2. Read each site, copy the **exact** source text into `scripts/specs/<name>.json`
3. Verify standalone before adding to the batch:
   `python3 scripts/mutate.py <scratch-copy> scripts/specs/<name>.json`
   — a `NO-SITE` or `BUILD-ERR` row means the spec is wrong, not the tree
4. Add a `name|relative-path|extra-env` row to `TREES` in `run-batch.sh`

A spec is only valid if the tree is **green at baseline** — `mutate.py` aborts otherwise.
