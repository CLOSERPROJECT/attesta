# Unify weightCheck Domain chrome → `dataReport`

**Date:** 2026-10-30  
**Branch:** `feat/attesta-rework` (tip has `5354984` dataReport)  
**Why:** weightCheck `registeredData` is Domain+instance only (not draft-driven). Same seam as loadedExtracted — should be server `dataReport` → `bindings.chromeDisplay`, SPA paint-only.

## Law

1. Convert weightCheck chrome id `registeredData` from `kind: recap` → `kind: dataReport` + JSONata `expr` emitting display DTO (truck weight, pallet sum, Match % + bands → tone).
2. Keep `currentOperator` as live `kind: readonly` (`$operator.displayName`) — not Domain; do **not** move in this slice.
3. Both truck-loop fixtures (server + app mirror).
4. **DTO / widget:** support flat report shape so weightCheck does not look like multi-load finalResults:
   - Flat: `{ value, error, title?, rows: [{label,value,error,tone}] }`
   - Sections (existing): `{ value, error, title?, sections: [...] }`
   - `dataReportField`: if `rows` present (and no/empty sections), paint like today’s recap (title + rows, no loadId headers, no “Nessun carico primario”).
   - Soften empty: only show plant “Nessun carico primario” when sections-mode and empty; flat empty → muted empty or nothing.
5. Go already evals all `dataReport` into `chromeDisplay` with `$domain` + `$instance` — weightCheck needs `$instance.id` (already wired).
6. Tests: BuildTaskFormBundle / JSON assert `chromeDisplay.registeredData` for weightCheck with probe domain; SPA unit for flat rows path.
7. Docs: light ADR-0005 / GLOSSARY touch if wording still implies dataReport is only multi-section / loadedExtracted-only.
8. Do **not** delete live `recap` mechanism yet (other uses may remain — grep; if register has none, leave mechanism for now).
9. No commit unless parent asks after green. No push.

## weightCheck expr sketch (Domain-only)

```jsonata
(
  $load := $lookup($domain.load, $instance.id);
  $truck := $load.truckWeight;
  $palletSum := $sum($append([], $domain.pallet.*[loadId=$instance.id].weight) );
  $m := $max([$truck, $palletSum]);
  $pct := $m = 0 ? 0 : 100 * $min([$truck, $palletSum]) / $m;
  $tone := $pct >= 98 ? "green" : $pct >= 90 ? "warning" : "danger";
  $kg := function($v){ $type($v) = "number" ? $string($v) : null };
  {
    "value": null,
    "error": null,
    "rows": [
      {"label": "Peso camion live", "value": $kg($truck), "error": null, "tone": null},
      {"label": "Somma pesi bancali live", "value": $kg($palletSum), "error": null, "tone": null},
      {"label": "Match %", "value": $string($pct), "error": null, "tone": $tone}
    ]
  }
)
```

Use jsonata-go gotchas from `NOTES-dataReport-contract-2026-10-30.md` (`$sum`/`$append`, `$type` checks).

## Verify

```bash
cd server && go test ./internal/living/... ./cmd/server -count=1
cd app && bun run test:unit -- --run src/lib/living
```

## Out of scope

- Moving `$operator` readonly / draft-driven chrome
- Deleting `recapField` wholesale
- Re-grilling dataReport kind
