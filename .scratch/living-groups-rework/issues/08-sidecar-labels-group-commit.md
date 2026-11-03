# 08: Proper labels on group leaf Commit (Sidecar HttpPort)

**What to build:** When an operator Commits a Task group first leaf that authors label HTTP effects (e.g. lot register), product Commit uses the existing **Sidecar HttpPort** / living-http peer so real label artifacts (PDF/CSV) are produced and returned as Commit links—same path as flat Jobs on rework. No FakeHttp in product. Proves the groups Commit path did not regress label generation.

**Blocked by:** 02 — Server — Spawn via first-leaf Commit, Close API, and where-next

**Status:** done

- [x] Group leaf Commit with authored `/fake/labels` (or successor) effects hits Sidecar when `LIVING_HTTP_URL` is set; links are fetchable via Attesta artifact proxy
- [x] Missing/unset living-http still soft-fails into `httpErrors` without rolling back Domain (existing HttpPort law)
- [x] Automated or Compose smoke: spawn+Commit one lot → at least one label artifact link works end-to-end
- [x] Tests may keep FakeHttp/recording doubles; product wiring stays `HttpPortFromEnv` / Sidecar
- [x] Does not rebuild living-http from scratch (reuse rework sidecar); out of scope: multi-org HttpPort tranche B

## Proof (automated)

No product code change required: nested leaf Commit already runs `taskOnSubmitHTTP` through the same `Committer.HttpPort` as flat Jobs (`HttpPortFromEnv` → Sidecar). Ticket proves the groups path.

```bash
cd server && go test ./internal/living/ -count=1 -run 'TestCommitGroupLeaf'
```

| Test | Covers |
| --- | --- |
| `TestCommitGroupLeafSidecarLabelsAndProxy` | Spawn+Commit group leaf → Sidecar `/fake/labels` → proxy-rewritten link → Attesta artifact GET returns PDF |
| `TestCommitGroupLeafUnconfiguredHttpSoftFail` | Unconfigured port → `httpErrors`, Domain group instance persists |
| `TestCommitGroupLeafSidecarViaHTTPHandler` | `/api/living/commit` wire returns Sidecar links for nested spawn path |

## Compose smoke (optional; plant groups fixture is ticket 07)

Env (same as `.scratch/living-http/` 01C): `LIVING_HTTP_URL=http://living-http:8090`, shared `LIVING_HTTP_TOKEN` (Compose default `dev-living-http-token`).

Until **07** authors intake lots as a Task group with `/fake/labels` on the leaf, plant UI cannot Spawn a labeled lot. Sidecar itself is already green for flat feragame register (living-http 01C). After 07:

1. `task up` so `attesta` + `living-http` are up.
2. Administration: open Job overview → Spawn lot → Commit register leaf.
3. Commit links include **Stampa etichette**; click → `/api/living/artifacts/<32-hex>` PDF (session required).
4. With `LIVING_HTTP_URL` unset, Commit still persists Domain; UI shows soft httpErrors (no FakeHttp).
