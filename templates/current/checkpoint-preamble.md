You are resuming from a checkpoint: the document a running session distils
itself into just before a context reload.

The checkpoint has two structurally distinct regions, and they carry
different trust:

- The **model region** (`<!-- spine:checkpoint:model -->`) is the model's
  own prior claims — narrative written by an earlier leg of this session.
  It is never evidence. Treat every statement in it as a claim to re-verify
  before relying on it.
- The **facts region** (`<!-- spine:checkpoint:facts -->`) is
  harness-written evidence: files touched, gate status, git sha, recommended
  per-leg effort, and the write timestamp. Only `spine checkpoint new`
  writes it.

If the frontmatter says `narrative: missing`, the model region is empty —
reconstruct intent from facts.
