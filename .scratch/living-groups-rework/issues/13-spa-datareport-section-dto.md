# 13: Neutralize SPA dataReport section DTO (no loadId)

**What to build:** Remove Feragame `loadId` / `secondaryLoadIds` from living SPA chrome types and normalizer. Use domain-agnostic section keys (recommended: `id`, `secondaryIds`). Update feragame Config `dataReport` expr that emits sections, SPA paint (`data-report-field.svelte`), and tests. Go engine stays opaque (no plant keys in living projection code).

**Blocked by:** —

**Status:** open

**Audit:** [Audit](9c5fdd0e-e3e8-498e-938a-3f4996773ad7) medium; human 2026-11-03: fix now.  
**Scout:** [Scout](6460ee8a-5d26-4fc8-bc1e-661fb0a24e94) — Go opaque; SPA enforces keys; ship Config `$section` + SPA together (option A: `id`/`secondaryIds`). Domain pallet `loadId` facts out of scope.

- [x] `ChromeDataReportSectionDisplay` + `normalizeSection` use generic keys only
- [x] `data-report-field.svelte` paints `id` / `secondaryIds`; drop `data-load-id` plant attr name
- [x] Feragame fixture `dataReport` section JSON keys aligned
- [x] Unit/bundle tests updated; no SPA living lib requiring `loadId` as chrome DTO key
- [x] Domain entity field `loadId` on pallets (facts) may remain — out of scope (authored Domain shape, not chrome DTO)
