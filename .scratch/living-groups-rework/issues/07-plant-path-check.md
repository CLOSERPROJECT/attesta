# 07: Plant path check (intake lots / line weigh)

**What to build:** Fixture and automated (or scripted) verification that the plant path works on rework: several intake lots with labels, Close lots, then the next truck step; and the line weigh-series group pattern if present in the fixture. Confirms Sidecar (or test double) labels, overview Spawn/Close, and post-Close advance together.

**Blocked by:** 05 — After submit with downloads — stay, Start another, Close; 06 — Home Available includes nested ready Tasks; 08 — Proper labels on group leaf Commit (Sidecar HttpPort)

**Status:** done

- [x] Feragame (or successor) fixture authors Task groups needed for intake lots and line weigh series without regressing rework seed/demo shape
- [x] Automated path covers: start Job → Spawn/submit multiple group instances (with links where authored) → Close → next Job Task available / navigable
- [x] Line weigh-series equivalent covered or explicitly waived with a note in the coordinator
- [x] No product FakeHttpPort; labels via Sidecar or intentional test double only in tests

## Proof (automated)

```bash
cd server && go test ./internal/living/ ./cmd/server -count=1
# plant lock:
cd server && go test ./internal/living/ -count=1 -run 'TestPlantPathLotsAndLineWeigh'
```

| Path | Covers |
| --- | --- |
| `TestPlantPathLotsAndLineWeigh` | arrive → 2× lot Spawn+register (FakeHttp links) → Close lots → weightCheck → openTurno → weigh series → Close → lineLoad |
| `TestCommitWeighCompleteThenCloseUnlocksLineLoad` | line weigh group Close unlocks lineLoad |
| Seed staged matrix | split-role Available: lots/register, weightCheck, weigh, lineLoad |

Fixture: `server/internal/living/fixtures/feragame-config.yaml`  
Helpers: `server/internal/living/plant_flow_test.go`

## Compose smoke (optional; Sidecar labels)

Same env as 08 / living-http 01C: `LIVING_HTTP_URL=http://living-http:8090`, shared `LIVING_HTTP_TOKEN`.

1. `task up` so `attesta` + `living-http` are up (no seed `--replace` unless human OK).
2. Administration: Nuovo camion → truckArrive → overview Spawn lot → Commit register → **Stampa etichette** link → `/api/living/artifacts/<32-hex>` PDF.
3. Spawn another lot → Close lots → weightCheck available.
4. Line: openTurno → Spawn weigh → Close palletWeight → lineLoad available.
