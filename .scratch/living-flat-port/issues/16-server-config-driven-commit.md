# 16: Server — config-driven Commit (ELT), drop Feragame Task switch

**What to build:** Make Go Commit authority execute Org Config (`onSubmit.transform`, mint/mintMany/patch, `instanceId`/`label` exprs, leftovers `reads.where`) instead of hand-coded `apply*Commit` switches for truck-loop Task paths. Feragame content stays in the fixture YAML only.

**Why:** Flat port 02–08 shipped behaviour by reimplementing each intake/line Task in Go. Fixture ELT/`reads` are largely decorative. Product law (decisions dump) says Go is Commit authority with the same mechanics as experiment — not a Feragame-specific switch.

**Blocked by:** none for a first ELT slice (Cerbos/Unlock/startsWhen can keep feeding compact attrs). Coordinates with **14** (UI should consume config, not duplicate plant logic) and **15** (gap inventory).

**Status:** done — reviewer APPROVE; squash into rework; human smoke

- [x] Commit resolves Task from Active Org Config and runs authored transforms (no per-Task-path Go plant logic for register/codiceFir/palletWeight/weightCheck/shift/lineLoad/containerReg/loadedExtracted)
- [x] Leftover eligibility comes from config `reads` / where, not hardcoded pallet/lineLoad filters
- [x] Job instance id/label from config exprs (not `"Turno: "` / client NIR- mint as product law)
- [x] Domain entity kinds follow config `domain.entities` (not fixed load/pallet/container-only types baked as the only model)
- [x] Truck-loop fixture still demos the same behaviour via YAML; Cerbos + Unlock + startsWhen still gate
- [x] Tests: fixture walk + at least one assertion that a transform change in YAML changes Commit without a new `apply*` function
- [x] Document removed hardcoding in ticket notes / COORDINATOR

**Out of scope:** FormPlan UI (14); readyWhen/HttpPort can be follow-on; Mongo durability; ticket 13 rename; Task groups.

**Refs:** experiment `transform-runner` / local-commit-port; `server/internal/living/commit.go`; decisions port policy; hardcoding audit in ticket 15.

## Implementer notes

**Expr engine:** `github.com/blues/jsonata-go` v1.5.4 — JSONata parity with experiment fixture exprs (`$filter` / `$count` / `$exists` / `$load` aliases). `ErrUndefined` omitted in `evalSet`; explicit `$value` null kept for progress clear.

**Removed hardcoding:**
- Deleted per-path `applyRegisterCommit`…`applyLoadedExtractedCommit` switch in `commit.go`
- Leftover ids from Task `reads.<key>.where` via `ResolveRead` (not pallet/lineLoad Go filters)
- Job instance id/label from Job `instanceId` / `label` exprs after transform
- Entity kinds validated against `domain.entities`; mutate supports Extra kinds beyond load/pallet/container

**Still plant-shaped (prep, not Commit switch):** `flattenContainerRegFacts` material key list (schema-detected containerReg); object-row weigh field validation (idoneita/tareKind). ReadyWhen (17) / HttpPort (18) untouched.
