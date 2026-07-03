<!-- spine:begin v2 -->
# spine

Uses the **unified workflow** — see `WORKFLOW.md` for the active profile (`library-cli`) and stages.

- Specs / PRDs / plans -> `docs/specs/` (pairs: `<date>-<topic>-design.md` + `-plan.md`)
- Decisions (ADRs) -> `docs/adr/` (convention in `docs/adr/README.md`)
- Issue / bug ledger -> `docs/issues/` (dependency convention in `docs/issues/README.md`)
- Handoffs -> `docs/handoffs/`

**Mandatory gates:** a PRD up front (run `/grill-with-docs` -> `/to-prd`) and verification before completion.
**Model:** see `WORKFLOW.md` `model_routing` (primary / fallback-on-refusal / routine; swappable).
<!-- spine:end -->
