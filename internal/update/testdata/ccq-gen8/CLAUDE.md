<!-- spine:begin v8 -->
# ccq

Uses the **unified workflow** — see `WORKFLOW.md` for the active profile (`library-cli`) and stages.

- Specs / PRDs / plans -> `docs/specs/` (pairs: `<date>-<topic>-design.md` + `-plan.md`)
- Decisions (ADRs) -> `docs/adr/` (convention in `docs/adr/README.md`)
- Issue / bug ledger -> `docs/issues/` (dependency convention in `docs/issues/README.md`)
- Handoffs -> `docs/handoffs/`

### Issue tracker

Issues live as markdown files in `docs/issues/` (the ledger above). `/wayfinder` and `/to-tickets` publish here too — see "Wayfinding operations" in `docs/issues/README.md`.

**Mandatory gates:** a PRD up front (run `/grill-with-docs` -> `/to-spec`), `/spec-review` of the finished diff against the PRD, and verification before completion.
**Model:** see `WORKFLOW.md` `model_routing` (primary / routine / mechanical / fallback; swappable).
<!-- spine:end -->
