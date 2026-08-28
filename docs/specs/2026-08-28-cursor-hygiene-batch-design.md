---
title: "Cursor hygiene batch: comma-list tickets grammar + close-fence canonical span + D13 hardening"
tickets: I113-I115
created: 2026-08-28
status: draft
---

# Cursor hygiene batch (I113 + I114 + I115) — design

## Problem Statement

Three small hygiene gaps, all fallout the doctor-hygiene batch and the I109
verify gate left behind, each currently visible or latent on spine itself:

1. **The cursor `tickets:` grammar cannot express a two-ticket batch** (I114).
   The first multi-ticket effort (`I065,I106`) ran with its issues/implement
   evidence unjudged — the value resolves under none of the three grammar
   forms, and the range form would have resolved to ~40 tickets the effort
   never touched. The degradation is visible (the D9 warn on
   `.superpowers/sdd/progress.md` is live today, spine's only remaining
   doctor warn) but real: batches are now the normal working shape, and every
   one of them would run unjudged.

2. **`NonCanonical` misses one hand-edit shape** (I113). The guard compares
   `content[open.start:closeF.end]`, and `closeF.end` stops at the closing
   tag text — so trailing whitespace after the closing fence is outside the
   compared span. The same whitespace on the *opening* fence line is
   correctly caught. A hand edit the sole-writer rule forbids goes
   undetected, asymmetrically. Pre-existing, not an I109 regression; found by
   the I109 verify gate's fresh-context probe.

3. **The new D13 per-ticket checks have four hardening gaps** (I115), found
   by the dhyg batch's final whole-branch review: an uncommented deliberate
   parser divergence (no `#`-comment stripping — `#` is significant in
   `batch:` ids) that a future DRY pass would silently break; a relative
   `workspace:` stat'd against process CWD rather than `--dir`; YAML-quoted
   values reading as malformed; and an intentional silence (fence-less
   ticket) that no test asserts.

Separately, the dhyg spec-review found the **generation-bump rule** (every
content-changing template change appends its predecessors' dropped lines to
the superseded set) stated only as a code comment — nothing binds a template
author who never opens that file.

## Solution

One effort, `cursor-hygiene-batch`, three tickets in two lanes:

- **Lane A (serial, I114 → I113)** — both land in the cursor package.
  I114 adds a comma-list form to the tickets grammar; because the WORKFLOW
  template embeds the grammar line verbatim, this is a template-content
  change and the gen-bump rule fires: the old grammar line joins the
  superseded set in the same change, and the binding gen-bump one-liner
  gains its home in the WORKFLOW template's authoring notes. I113 then
  extends the canonical-form comparison span to the end of the closing
  fence's line, closing the asymmetry without changing a byte of what
  `spine cursor` writes.
- **Lane B (parallel, I115)** — the four D13 hardening items, disjoint from
  lane A.

Deploy carries the estate sweep for I114's WORKFLOW refresh, same checklist
shape as the dhyg sweep. Post-batch, the next effort's cursor
(`tickets: I113-I115` — a contiguous range, resolvable under the *current*
grammar, so this effort itself runs fully judged) supersedes the dhyg
cursor, and spine's doctor returns to exit 0.

## User Stories

1. As an effort lead, I want a `tickets:` form that names exactly the tickets
   of a non-adjacent batch, so that batch efforts run with their
   issues/implement evidence judged instead of degraded.
2. As the estate operator, I want the D9 warn on spine's ledger retired by
   fixing the grammar rather than by rewording the value, so that the doctor
   signal stays honest.
3. As a session reading an unresolvable-tickets note, I want the note to name
   the comma-list form, so that the remediation is visible at the point of
   failure.
4. As the cursor's sole writer, I want a malformed element to make the whole
   comma-list unresolvable, so that evidence is never judged against a
   silently partial ticket set.
5. As the cursor's sole writer, I want a duplicated element to make the whole
   value unresolvable, so that a hand-mangled list fails loudly instead of
   double-counting.
6. As the cursor's sole writer, I want no whitespace tolerance inside the
   list, so that a spaced hand edit is caught rather than legitimized —
   consistent with the canonical-form philosophy.
7. As a template author, I want the gen-bump rule stated in the WORKFLOW
   template's own authoring notes, so that the rule binds me without my ever
   opening the updater's source.
8. As the estate operator, I want I114's template change to append the old
   grammar line to the superseded set in the same change, so that the next
   sweep refreshes estate WORKFLOW files cleanly instead of stranding them.
9. As the estate operator, I want the deploy-stage sweep recorded as a
   per-repo checklist with exit codes, so that the rollout is verified
   rather than assumed.
10. As the owner of a genuine local edit in a machine-owned WORKFLOW
    (ultima), I want the sweep to skip my file and say so, so that the guard
    keeps working and the staleness is a named, deliberate residue.
11. As an auditor of the sole-writer rule, I want trailing whitespace on the
    *closing* fence line to report `NonCanonical`, so that no hand-edit
    shape is silently exempt.
12. As a `spine cursor` user, I want the fix confined to what the guard
    *compares*, so that what the tool *emits* — `Block()` — never changes.
13. As an operator with a cursor block at end-of-file without a trailing
    newline, I want it still judged byte-canonical, so that the widened span
    does not invent a new false positive at the document boundary.
14. As an operator with a CRLF document, I want it judged on its own terms
    and no noisier than today, so that the fix does not regress I109's CRLF
    recognition.
15. As a future maintainer consolidating the four frontmatter parsers, I
    want a guard comment on the doctor parser naming its deliberate
    divergence, so that DRY work cannot silently break `batch:` ids.
16. As the estate operator running `spine doctor --dir <other-repo>`, I want
    `workspace:` existence checked free of process-CWD dependence, so that
    the check cannot false-warn based on where I happened to stand.
17. As the convention's owner, I want a non-absolute `workspace:` value to
    warn, so that the absolute-path mandate is enforced rather than assumed.
18. As a maikanban board that emits quoted YAML, I want surrounding quotes
    stripped before validation, so that a semantically identical value never
    reads as malformed.
19. As a doctor maintainer, I want a test asserting a fence-less ticket
    yields zero D13 findings, so that the intentional silence is pinned.
20. As the batch's team lead, I want the ledger keys
    (`batch:`/`workspace:`/`commits:`/`review:`) dogfooded on these three
    tickets with the lead as sole writer, so that the I106 convention gets
    its second live exercise.
21. As the human reviewer, I want the final whole-branch review to include
    the requirements-attack step, so that spec contradictions surface with
    proposed resolutions instead of being silently resolved.

## Implementation Decisions

- **Lanes:** I113 and I114 both modify the cursor package, so lane A runs
  them serial, I114 first (it retires the live warn and carries the template
  work; I113 rebases trivially). I115 is disjoint and runs parallel in
  lane B.
- **I114 grammar:** a comma-list form, each element a bare ticket id. No
  whitespace tolerance — spine emits the value, so a spaced list is a hand
  edit and fails loudly. A malformed element or a duplicate element makes
  the whole value unresolvable; there is no partial resolution. The grammar
  is documented once (the cursor package's canonical grammar text) and the
  unresolvable-tickets note names the new form.
- **I114 is a template-content change.** The WORKFLOW template embeds the
  grammar line verbatim, so the gen-bump rule fires: the outgoing grammar
  line joins the superseded-lines set in the same change. Per the dhyg
  ruling (upheld at its final review), a content-bearing change to a
  current-generation template file takes no version bump; the propagation
  proof is the migration fixtures re-passing against the changed template.
- **The gen-bump rule's binding one-liner folds into I114** (its natural
  vehicle — I114 fires the rule anyway): stated in the WORKFLOW template's
  authoring notes, and propagated by the same sweep the change already
  requires.
- **I113 widens only the compared span**: from the closing tag's end to the
  end of the closing fence's line, excluding its line terminator (or
  end-of-content when the document ends without one). The fence scanner
  already walks that boundary; the fence value carries it as a third
  offset. The canonical serialization — what `spine cursor` writes — is
  untouched.
- **I115, four items in the doctor's per-ticket checks:** (a) a one-sentence
  guard comment naming the no-comment-stripping divergence and why; (b) an
  absolute-path warn on `workspace:` — non-absolute values warn and are not
  stat'd, which also removes the CWD dependence (the convention mandates
  absolute paths; resolving relative values against `--dir` would
  legitimize a forbidden form); (c) strip surrounding quotes from
  frontmatter values before validation (tolerant reader — quoting is a
  property of the emitter, not the value); (d) pin the fence-less silence
  with a test.
- **Routing:** I114/I115 `tier: routine`, `review-tier: routine`; I113 as
  filed (`tier: mechanical`, `review-tier: routine`), its execution-mode
  updated to subagent-driven to match the batch dispatch. Codex team: sol
  leads; workers terra (lane A: I114 then I113) and luna (lane B: I115).
  Reviews stay claude-side — fable-5 @ high for the final whole-branch
  review, requirements-attack step included. Never claude-sonnet-5.
- **Ledger keys are dogfooded, lead as sole writer:** batch ids
  `2026-08-28-chyg#1` (I114), `#2` (I113), `#3` (I115); `workspace:` while
  the worktree exists (absolute path), cleared at close; `commits:` in the
  close commit; `review:` is the board/human's, not the team's.
- **Deploy-stage estate sweep**, dhyg checklist shape (per-repo
  pre/-write/post/doctor exit capture): the 8 clean repos — ccq,
  home-lab-admin, jarvis, notetui, observability_notes, pure-automation,
  deepthought, hbmview. The 3 residual-skip repos (praxis, moo-clone
  issues-READMEs; ultima WORKFLOW) are left alone — ultima's WORKFLOW stays
  one grammar-line stale until the owner reconciles its local edit, and the
  handoff names that explicitly. Sweep commits stay local-only; pushing is
  per-repo owner's call.
- **This effort's cursor** is `tickets: I113-I115` — contiguous, so it
  resolves under the current grammar and the effort runs fully judged (no
  repeat of the accepted dhyg degradation).

## Testing Decisions

External behavior only: assert on parse results, evidence-report notes,
doctor findings, and exit codes — never on internal tables. All seams exist;
no new ones.

- **Stages evidence seam** (I114): the ticket-grammar test pattern from the
  bare-id/prefix work. A two-element list resolves to exactly those ids; a
  longer list likewise; a list with a malformed element is unresolvable as a
  whole; a duplicated element is unresolvable; a spaced list is
  unresolvable; the note text names the comma form. The unresolvable cases
  are the negative controls proving no-partial-resolution is load-bearing.
- **Cursor parse seam** (I113 + I114's grammar text): the I109 test family.
  Closing-fence trailing spaces and tabs each report `NonCanonical` true
  with no findings; the existing opening-fence case still passes; a
  byte-canonical block reports false — including at end-of-file with no
  trailing newline, the boundary the new offset must get right; the CRLF
  document test stays green and no noisier.
- **Doctor seam** (I115): the D13 findings-table tests. Non-absolute
  `workspace:` warns without stat; absolute-and-missing still warns;
  quoted `batch:`/`workspace:` values validate as their bare equivalents; a
  fence-less ticket yields zero D13 findings. Existing negative control (a
  malformed `batch:` on a fixed ticket does not fire) stays green.
- **Template/updater seam** (I114's gen-bump): the migration-fixture
  pattern — a WORKFLOW frozen at the outgoing grammar line refreshes
  cleanly (proves the superseded entry is load-bearing); a genuine local
  edit still skips the file (proves it is not over-broad). Both arms run;
  a prescribed negative control is a hypothesis until observed red.
- **Dogfood seam**: after I114 lands and the next effort's cursor takes
  over, `spine doctor` on spine exits 0 (adr info note still printed) —
  the live D9 reproduction converted into the batch's end-to-end proof.

## Out of Scope

- DRY consolidation of the four frontmatter parsers (audit, stages, adr,
  doctor) — I115 only fences the divergence with a comment.
- A `spine batch` helper (unchanged from the dhyg ruling: not until
  `grep -l` proves fragile).
- Reconciling the praxis / moo-clone / ultima genuine local edits — owner's
  call, outside any sweep.
- I072 (host-config schema) — considered and passed over for this batch.
- The openweights programme (I111, I112) — parked.
- Any change to what `spine cursor` writes; `Block()` is untouched.

## Further Notes

- The D9 warn on `.superpowers/sdd/progress.md` self-cures in two parts:
  I114 makes `I065,I106` resolvable, and this effort's own cursor replaces
  the live value anyway. Neither part involves editing the dhyg record.
- The dhyg spec-review's second partial (story 17's sweep checklist living
  in handoff commits) is operational, not a defect; this batch keeps the
  same shape.
- Effort mechanics: `spine cursor start --force --effort
  cursor-hygiene-batch --tickets I113-I115` runs BEFORE any
  `spine handoff new`. Never tick the handoff stage. Stage explicit paths
  only. Batch commits so one `maipipe run full --wait` lane run covers
  them. Read exit codes unpiped under fish.
