# 16: Server — config-driven Commit (ELT), drop Feragame Task switch

**What to build:** Make Go Commit authority execute Org Config (`onSubmit.transform`, mint/mintMany/patch, `instanceId`/`label` exprs, leftovers `reads.where`) instead of hand-coded `apply*Commit` switches for truck-loop Task paths. Feragame content stays in the fixture YAML only.

**Why:** Flat port 02–08 shipped behaviour by reimplementing each intake/line Task in Go. Fixture ELT/`reads` are largely decorative. Product law (decisions dump) says Go is Commit authority with the same mechanics as experiment — not a Feragame-specific switch.

**Blocked by:** none for a first ELT slice (Cerbos/Unlock/startsWhen can keep feeding compact attrs). Coordinates with **14** (UI should consume config, not duplicate plant logic) and **15** (gap inventory).

**Status:** ready-for-agent

- [ ] Commit resolves Task from Active Org Config and runs authored transforms (no per-Task-path Go plant logic for register/codiceFir/palletWeight/weightCheck/shift/lineLoad/containerReg/loadedExtracted)
- [ ] Leftover eligibility comes from config `reads` / where, not hardcoded pallet/lineLoad filters
- [ ] Job instance id/label from config exprs (not `"Turno: "` / client NIR- mint as product law)
- [ ] Domain entity kinds follow config `domain.entities` (not fixed load/pallet/container-only types baked as the only model)
- [ ] Truck-loop fixture still demos the same behaviour via YAML; Cerbos + Unlock + startsWhen still gate
- [ ] Tests: fixture walk + at least one assertion that a transform change in YAML changes Commit without a new `apply*` function
- [ ] Document removed hardcoding in ticket notes / COORDINATOR

**Out of scope:** FormPlan UI (14); readyWhen/HttpPort can be follow-on; Mongo durability; ticket 13 rename; Task groups.

**Refs:** experiment `transform-runner` / local-commit-port; `server/internal/living/commit.go`; decisions port policy; hardcoding audit in ticket 15.
