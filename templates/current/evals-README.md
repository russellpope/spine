# Evals

Machine-checkable convention (owned by `spine`; see `spine eval --help`):

- One directory per eval: `YYYY-MM-DD-<slug>/`, created by `spine eval new "<title>"`.
- `eval.md` — front matter `title`, `created`, `prompt` (path), `rubric` (path); prose body free.
- `runs/<name>.md` — one record per run, created by `spine eval add-run --eval E --name N`.
  Front matter: `name`, `created`, `model`, `stage`, `score`.

`stage` and `score` are written by the process driving the eval (the
/model-eval skill) and read back verbatim by `spine eval list` — spine never
interprets them. The canonical loop stages are the run template's body
sections: Wire, Audit, Score, Compare, Remediate, Rescore.

`spine doctor` (D7) validates structure only: parseable front matter with the
required keys present. Values — including empty ones — are yours.

## Optional pin-evidence profile

An owner may cite a run while ratifying a host pin with the exact reference
`eval:<eval-dir>/runs/<run>.md`. `<eval-dir>` is `YYYY-MM-DD-<slug>` where the
date is real and `<slug>` is lowercase letters or digits separated by hyphens.
`<run>` starts with an ASCII letter or digit and then uses only ASCII letters,
digits, `_`, or `-`. Doctor reads only that selected repository-local run.

For a cited run, add a complete I077 battery record:

```text
battery_version: 1
battery_verdict: pass
battery_results: invocation=KILLED,wiring=KILLED,flag-honoured=KILLED,column-presence=KILLED,column-order=KILLED,ordering=KILLED,units-labels=KILLED,security-default=REPORT-ONLY,lifecycle=REPORT-ONLY,error-path-behaviour=KILLED
```

The keys are ordered exactly as shown. Values are `KILLED`, `SURVIVED`,
`NO-SITE`, `BUILD-ERR`, or `REPORT-ONLY`; `REPORT-ONLY` is only valid for
`security-default` and `lifecycle`. A `pass` verdict requires all other eight
keys to be `KILLED`. A `fail` verdict requires at least one of those eight to
have another valid value. The run's `created` date must be an unquoted or
double-quoted `YYYY-MM-DD` UTC calendar date, no more than 90 days old
inclusive, and its `model` must exactly match the pinned model.

This is an advisory pin-evidence profile. It is optional for ordinary evals,
does not change `spine eval list`, and never de-ratifies or blocks a pin. See
`docs/mutation-battery-checklist.md` for how to run and report the ten probes.
