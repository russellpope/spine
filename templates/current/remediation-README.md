# Remediation

Remediation is the loop that turns gate pack findings into fixes: a
**hitlist** of findings goes out, the author submits a fix, and the result is
recorded as a **round record**. This directory is where the records live.

## Layout

    docs/remediation/_hitlist.template.md
    docs/remediation/_round.template.md
    docs/remediation/<effort>/hitlist-N.md
    docs/remediation/<effort>/round-N.md

One directory per effort (the kebab-case effort name the stage cursor uses),
one hitlist and one round record per round, numbered from 1.

The two `_*.template.md` files at the top of the directory are the templates
spine ships (`hitlist.tmpl.md` and `remediation-round.tmpl.md` in the binary),
scaffolded here the same way `docs/issues/_template.md` is: machine-owned,
created by `spine init` and refreshed by `spine update`. There is no
`spine remediation new` verb — instantiate a record by copying the template:

    cp docs/remediation/_hitlist.template.md docs/remediation/<effort>/hitlist-1.md
    cp docs/remediation/_round.template.md   docs/remediation/<effort>/round-1.md

then fill it in. spine does not render round records: rendering them from run
facts is the evidence renderer's job.

## Round budget

The budget is **3 rounds per effort**, derived from the round record's
ordinal (`round-N.md`; sequential numbering is the convention) in the
effort's directory — there is no separate counter to keep in sync. A
4th or later round is legal only when its record carries
`extension-ratified-by:` in its frontmatter, naming the owner who ratified
the extension. `spine audit stages` advises (never blocks) on a round beyond
budget without ratification.

## Dose escalation

A hitlist's **dose** is how much of the fix is handed to the author:

1. `findings-only` — the default. File:line, the finding, why it matters, and
   a do-not-regress block. No fix text.
2. `prescriptive` — the finding plus a prescribed remedy.
3. `raw-review` — the raw review prose, verbatim.

Escalate one step at a time, and only after a round **fails on the same
finding**. Sameness is keyed by the results-contract `code` (e.g.
`go@1/tskip`), which is why the round record's per-finding table keys on the
code rather than on prose: "did this finding fail last round?" is a lookup,
not a judgment.

## Rescoring

A remediated submission is rescored as a fresh submission. That rule belongs
to the eval seam — see ADR 0007 and `docs/evals/README.md` — and is not
restated here, so that there is one definition of it.
