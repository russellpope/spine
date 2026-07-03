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
