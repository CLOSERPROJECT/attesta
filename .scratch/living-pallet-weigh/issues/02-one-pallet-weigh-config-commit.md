# 02: One-pallet palletWeight weigh Config + Commit

**What to build:** Logistic weighs one pallet per Spawned weigh Task: Spawn → choose one unweighted pallet → enter weight, tare (catalog default), average panel weight, required idoneita (default NIR) → Commit patches that pallet; repeat Spawn→weigh for more pallets; **one** Close ends the group. Replaces the multi-pallet full-map weigh. Group requires a child Submission before Close. Frozen Config in `LOCKED.md`. No QR; no loadedExtracted changes.

**Blocked by:** 01 (Schema defaults seed on every Task Load).

**Status:** Done (human cadence B 2026-11-03; Standards‖Spec APPROVE)

- [x] Feragame `palletWeight` / `weigh` matches frozen flat form + `patch` + `closeRequiresChildSubmission`
- [x] One Commit weighs exactly one selected pallet; other unweighted pallets stay unweighted
- [x] N× Spawn→weigh then one Close; lineLoad still unlocks when the group predecessor law says so
- [x] Living tests updated off the old full-map weigh helpers
