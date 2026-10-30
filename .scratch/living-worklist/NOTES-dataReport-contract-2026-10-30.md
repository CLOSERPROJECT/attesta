# dataReport implement contract (for agents)

**Branch:** `feat/attesta-rework` @ ~`fc0d1e2`  
**Ticket:** `.scratch/living-worklist/issues/06-dataReport-chrome-display.md`  
**Law:** `.scratch/living-worklist/GRILL-dataReport-chrome-2026-10-30.md`  
**Handoff:** `$TMPDIR/handoff-implement-dataReport-chrome-2026-10-30.md`

## Product law (do not re-grill)

- Authored aggregation in YAML JSONata → Go runs → SPA paints only
- `kind: dataReport`, widget `dataReportField`, result `bindings.chromeDisplay[fieldId]`
- Refresh: Load + successful Commit only (not draft keystrokes)
- Feragame purge: no `feragameFinalResults` / `finalResults` / `project-final-results` live path; plant math+copy in YAML
- Leave alone: live recap/readonly/valuePrint/Derived; Roles/readyWhen; JobChrome delete; twin DRY
- Docs: amend ADR-0005 + GLOSSARY (no new ADR)
- Hard call: do not put plant Task ids / materials / tare tables into `server/internal/living` or SPA living packages

## Wire contract

### YAML (both fixtures)

Replace:

```yaml
feragameFinalResults:
  kind: finalResults
  title: Risultati finali Feragame
  before: checkedLoadedWeight
```

With generic id e.g. `resultsReport`:

```yaml
resultsReport:
  kind: dataReport
  title: Risultati finali
  before: checkedLoadedWeight
  expr: |-
    ( … full display DTO … )
```

Files:

- `server/internal/living/fixtures/truck-loop-steps.yaml`
- `app/src/lib/living/form/fixtures/truck-loop-steps.yaml`

### Display DTO (expr output)

```json
{
  "value": null,
  "error": null,
  "sections": [
    {
      "loadId": "L1",
      "secondaryLoadIds": ["L2"],
      "rows": [
        { "label": "Camion lordo", "value": "1000", "error": null, "tone": null },
        { "label": "Plastica", "value": "70", "error": null, "tone": "info" },
        { "label": "Pannelli vs estratto", "value": "70 kg (60%)", "error": null, "tone": "danger" }
      ]
    }
  ]
}
```

Go may set `title` from chrome rule if expr omits it.

### JSONata gotchas (jsonata-go)

- No `/* */` comments inside expr (parsed as regex)
- `$max([$a,$b])` / `$min([$a,$b])` — not two-arg
- `$map([], …)` → null; use `$count($ids)=0 ? [] : $map(...)`
- `$safeSum := function($arr){ $reduce($arr ? $arr : [], function($a,$b){$a+$b}, 0) }`
- Empty containers: gate on `$count($keys($domain.container))=0 ? [] : …`
- Bindings: `$domain` from `buildFormBindings` (load/pallet/container/submissions; tareKinds only if on that Task — **put euro/industrial defaults + materials labels inside the expr**)
- Input `$` unused; pass `{}`

### Proven probe math (synthetic)

Domain L1: truck 1000/200; pallets 125 done + 100 partial; weigh euro×2 → tare 50; containers plastica 50+20, alluminio 30, mystery 5, secondary L2.

Expect: net panels 175; extracted 105; delta 70; pct 60 → tone danger; secondaryLoadIds `[L2]`.

Full materials order (from former `CONTAINER_REG_MATERIALS`): polvereDiVetro, granellaDiVetro, granellaDiVetro2, polvereDiSilicio, plastica, alluminio, sopravaglio, polvereDiAspirazione — labels Italian as today.

Match bands: ≥98 green, ≥90 warning, else danger (when both sides numeric).

### Go

- `readChromeRules`: `dataReport` requires non-empty `expr`
- `stringFieldForChrome`: `dataReport` → `dataReportField`; delete `finalResults` → `feragameFinalResultsField`
- `BuildTaskFormBundle`: after `buildFormBindings`, eval each dataReport into `bindings["chromeDisplay"].(map)[id]`
- Bad expr → bundle error (authored fixture must compile)
- Prefer JSON-shape tests on Load/Commit or `BuildTaskFormBundle` marshal; fixture expr coverage

### SPA

- Rename/adapt `feragame-final-results-field.svelte` → `data-report-field.svelte` (`dataReportField`)
- `LiveTaskChrome`: `kind: dataReport` reads server display from ports (bindings.chromeDisplay); no Domain walk
- Delete live use of `project-final-results.ts` / `evaluateFinalResultsEntry` / `getSnapshot` for report
- Keep `CONTAINER_REG_MATERIALS` for containerReg UI only
- Update `task-form-plan.ts`, `form-theme.ts`, `TaskForm.svelte`, bundle tests

### Verify

```bash
cd server && go test ./internal/living/... ./cmd/server -count=1
cd app && bun run check && bun run test:unit -- --run src/lib/living
```

### Commit

One conventional commit on `feat/attesta-rework` when green (parent will commit unless told otherwise). Include scratch grill/NEXT/ticket/notes with docs or feature commit.

**Do not push.**
