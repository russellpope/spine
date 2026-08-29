---
id: I118
title: "No documented portable install path, and spine version cannot answer which build a device runs"
severity: low
status: open
affects: []
blocked-by: []
execution-mode: inline
tier: primary
effort:
risk-triggers: []
review-tier: n/a
---

## Problem

The owner wants spine mildly portable across macOS devices. The pieces are
already there — the repo is public, templates are go:embed'd, paths go
through `os.UserHomeDir` — so a bare `go install
github.com/russellpope/spine/cmd/spine@latest` works on any Mac with Go.
But nothing documents it: the README's only build story is
clone-and-`make build` under Development.

Once two devices run spine, drift becomes the question, and `spine version`
cannot answer it: it prints only the compiled template generation. The
handoffs currently identify builds by sha256 prefix of the binary, by hand.

## Fix

Filed from the 2026-08-28 ergonomics batch grill (Q6–Q8):

1. README: an Install subsection with the `go install …@latest` one-liner,
   noting the binary is self-contained and `spine version` identifies the
   build.
2. `spine version`: keep line 1 exactly as today (`spine template
   generation N`), add a `build:` line from `runtime/debug.ReadBuildInfo`
   (module version, vcs revision, vcs time, dirty flag), degrading to
   `build: (no build info)` when the read fails. No ldflags, no release
   machinery.

Tests: `version` exits 0 with the unchanged first line and a `build:` line;
the formatter handles nil build info and the dirty flag.

## Related

- docs/specs/2026-08-28-ergonomics-portability-batch-design.md — the
  batch PRD carrying scope decisions (binary only; no goreleaser).
