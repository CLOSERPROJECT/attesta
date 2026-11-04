# Coordinator — one-pallet weigh + QR

**Branch:** `feat/living-pallet-weigh-one`  
**Frozen design:** [`LOCKED.md`](./LOCKED.md)  
**Base:** `feat/living-groups-rework` (groups through tip / accordion done).

## Tickets

| # | File | Blocked by |
| --- | --- | --- |
| 01 | `issues/01-flat-and-map-schema-defaults-on-load.md` | — |
| 02 | `issues/02-one-pallet-weigh-config-commit.md` | 01 |
| 03 | `issues/03-label-pdf-qr-endpoint.md` | — |
| 04 | `issues/04-qr-resolve-rules-http.md` | 02 |
| 05 | `issues/05-topbar-scan-cold-entityid.md` | 04 |

**Frontier:** board complete — awaiting human smoke → land.

| # | Status |
| --- | --- |
| 01 | Done (Standards‖Spec APPROVE; uncommitted) |
| 02 | Done (human **B** lots cadence; LOCKED amended; Standards‖Spec APPROVE; uncommitted) |
| 03 | Done (Standards‖Spec APPROVE; uncommitted; seed --replace done) |
| 04 | Done (Standards‖Spec APPROVE after fixer; uncommitted) |
| 05 | Done (Standards‖Spec APPROVE after fixer; uncommitted) |

### Settled hard calls
- Cadence **B** (2026-11-03): N× Spawn→weigh, one Close.
- `$facts`→`$.*` OK; QR eligible expr restated for jsonata-go (Spec OK).
- `seed --replace` OK for Config; do not push without ask.
- **2026-11-03:** drop `LIVING_QR_ENDPOINT`; `$qrEndpoint` derived from `PUBLIC_BASE_URL` + `/app` only (DRY) — Done APPROVE.

### Bugs (post-board)
- **2026-11-03:** one-pallet register → labels PDF “(no label ids…)”: jsonata-go unwraps one-element `$each` to a bare object; sidecar `labelEntries` only accepted arrays. Fixed in living-http (+ Config `[$each(…)]` wrap). `task living-http:rebuild` done; re-print labels (old artifact URLs stale). Optional `seed --replace` for Config wrap.

### Next (human)
1. Smoke on Compose (rebuild living-http for PDF QR; ensure `PUBLIC_BASE_URL` — `$qrEndpoint` = that + `/app`).
2. Commit / squash onto `feat/living-groups-rework` when OK; later `feat/attesta-rework`.

## Verify

```bash
cd server && go test ./internal/living/... -count=1
cd app && bun run test:unit -- --run src/lib/living src/lib/router
```
