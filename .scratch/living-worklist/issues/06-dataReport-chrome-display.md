# 06: Chrome Domain projectors → `dataReport` / `chromeDisplay`

**What to build:** Domain-only aggregate chrome (today’s loadedExtracted final results) is authored as `kind: dataReport` with a JSONata `expr` that emits the full display DTO. Go evaluates it on Load/Commit into `bindings.chromeDisplay[fieldId]`; the SPA paints only. Feragame-specific chrome ids/widgets and the SPA `project-final-results` Domain walk go away. Plant math and Italian copy live in YAML.

**Blocked by:** None (can start immediately). Cleanup `fc0d1e2` already landed.

**Status:** done

- [x] Truck-loop YAML (server + app fixture mirror) uses generic field id + `kind: dataReport` + JSONata display DTO (labels/bands in expr); no `feragameFinalResults` / `finalResults`
- [x] `BuildTaskFormBundle` evaluates `dataReport` into `bindings.chromeDisplay[id]`; ui maps to `dataReportField`
- [x] SPA `dataReportField` reads server `chromeDisplay` only; no live Domain aggregate for report chrome; `project-final-results` / `feragameFinalResultsField` gone from live path
- [x] Load/Commit JSON tests assert `chromeDisplay`; Go expr fixture coverage; ADR-0005 + GLOSSARY amended (`dataReport`, `chromeDisplay`)

**Refs:** `.scratch/living-worklist/GRILL-dataReport-chrome-2026-10-30.md`; ADR-0005; handoff `handoff-implement-dataReport-chrome-2026-10-30.md`.

**Out of scope:** live `recap` / `readonly` / `valuePrint` / Derived to server; Roles/readyWhen chrome; JobChrome delete; twin fixture DRY.
