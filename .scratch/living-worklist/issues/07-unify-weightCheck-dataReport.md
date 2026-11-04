# 07: Unify weightCheck Domain chrome onto `dataReport`

**What to build:** weightCheck’s Domain-only `registeredData` chrome uses `kind: dataReport` + server `chromeDisplay` (same seam as loadedExtracted). Flat rows DTO so the paint matches today’s recap. Live `currentOperator` readonly stays client.

**Blocked by:** 06 dataReport chrome (done).

**Status:** done (uncommitted)

- [x] YAML (both fixtures): `registeredData` → `dataReport` + expr; `currentOperator` unchanged
- [x] `dataReportField` paints flat `rows` without load-section chrome
- [x] Tests assert `bindings.chromeDisplay.registeredData`; living Go + SPA unit green
- [x] Docs wording if needed

**Refs:** `.scratch/living-worklist/NOTES-unify-weightCheck-dataReport-2026-10-30.md`
