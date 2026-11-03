# Locked — one-pallet weigh + QR resolve (2026-11-03)

Branch: `feat/living-pallet-weigh-one` (from `feat/living-groups-rework`).  
Grill + interactive Config walk. **Do not re-litigate** without human.

## Product

- **Job:** `line` / Task group `palletWeight` / child `weigh` (logistic).
- **Cadence:** one pallet = one group occurrence — Spawn → weigh → explicit Close; serialize open pesate; `closeRequiresChildSubmission: true`.
- **Form:** flat `palletId` + weight + tare (+ custom) + `averagePanelWeight` + idoneita (required, default NIR) + derived `panelCount`.
- **Commit:** `patch` that one pallet; `averagePanelWeight` stays Submission fact.
- **No** multi-pallet `weights` mapTable / full-map.
- **Delivery:** weigh Config first; QR encode/resolve/topbar follow-on.
- **Ignore for now:** `loadedExtracted` / `$totalTare` flat-facts update.

## QR encode

- Baked into register labels PDF (`lots/register` → HttpPort → Sidecar).
- Config authors string: `$qrEndpoint & "?entityId=" & $id` (or equivalent).
- `$qrEndpoint` = runtime binding (ops/env → submit bindings).
- Sidecar dumb-draws `labels[].qr`.
- Cold https link desired (same string).

## QR resolve

- Topbar Scan → `POST` `{ raw }` only (no `$place` v1).
- Go resolve + Org Config `qr.rules`; SPA thin.
- Exactly one Worklist candidate → navigate + seed; else toast.
- Org from token. Decode: fixed convention (`entityId` query or bare id) — **no Config `parse`**.
- Lands on weigh form (`new` or open instance) with seed.

## Frozen Config — `qr`

```yaml
qr:
  rules:
    - id: line-pallet-weigh
      when: $type($scan.entityId) = "string" and $length($trim($scan.entityId)) > 0
      match:
        jobId: line
        groupId: palletWeight
        taskId: weigh
        allowSpawn: true
      eligible: |
        (
          $p := $lookup($domain.pallet, $scan.entityId);
          $exists($p) and $not($exists($p.weight))
        )
      seed:
        palletId: $scan.entityId
      messages:
        onZero: Nessuna pesatura disponibile per questo bancale
        onMany: Più destinazioni possibili: apri il turno giusto dal Worklist
        onIneligible: Bancale non pesabile (assente o già pesato)
```

## Frozen Config — group + weigh

```yaml
- id: palletWeight
  kind: group
  repeatable: true
  startLabel: Nuova pesata
  closeLabel: Chiudi pesatura
  closeRequiresChildSubmission: true
  substeps:
    - id: weigh
      roles: [logistic]
      reads:
        pallets:
          entity: pallet
          where: $not($exists(weight))
          key: id
      enumReads:
        palletId: pallets
      derived:
        panelCount:
          overridable: true
          expr: |
            (
              $avg := $data.averagePanelWeight;
              $kind := $data.tareKind;
              $tare := $kind = 'custom'
                ? $data.customTareWeight
                : $lookup($catalogs.tareKinds, $kind);
              $w := $data.weight;
              $avg > 0 and $type($tare) = "number" and $type($w) = "number"
                ? $floor(($w - $tare) / $avg)
                : undefined
            )
      schema:
        type: object
        title: Pesa bancale
        required: [palletId, averagePanelWeight, tareKind, weight, idoneita]
        properties:
          palletId: { type: string, title: Bancale }
          averagePanelWeight:
            type: number
            exclusiveMinimum: 0
            title: Peso medio pannello (kg)
            default: 20
          tareKind:
            type: string
            title: Tara
            enum: [euro, industrial, custom]
            default: euro
          customTareWeight:
            type: number
            exclusiveMinimum: 0
            title: Tara personalizzata (kg)
          weight: { type: number, exclusiveMinimum: 0, title: Peso }
          idoneita:
            type: string
            title: IR / NIR
            enum: [NIR, IR]
            default: NIR
          panelCount: { type: number, title: Pannelli }
      uiSchema:
        ui:options:
          order:
            [palletId, averagePanelWeight, tareKind, customTareWeight, weight, idoneita, panelCount]
        tareKind:
          ui:options:
            enumNames: ['euro (25 kg)', 'industriale (30 kg)', 'personalizzata']
        idoneita:
          ui:options:
            enumNames: [NIR, IR]
        panelCount:
          ui:field: derivedLeaf
      onSubmit:
        transform:
          - op: patch
            entity: pallet
            id: $facts.palletId
            set:
              weight: $facts.weight
              tareKind: $facts.tareKind
              idoneita: $facts.idoneita
              panelCount: $facts.panelCount
```

## Frozen Config — label QR fragment

```yaml
# lots/register onSubmit.http body
{ "id": $id, "qr": $qrEndpoint & "?entityId=" & $id }
```

## Deferred

- Last-weigh Submission prefill for `averagePanelWeight` (schema `default: 20` only for now).
- `$place` narrowing on resolve.
- `loadedExtracted` / `$totalTare` for flat weigh facts.
- Glossary / ADR until implement.
