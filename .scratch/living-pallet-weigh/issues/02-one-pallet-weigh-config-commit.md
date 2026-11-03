# 02: One-pallet palletWeight weigh Config + Commit

**What to build:** Logistic weighs one pallet per Task group occurrence: Spawn → choose one unweighted pallet → enter weight, tare (catalog default), average panel weight, required idoneita (default NIR) → Commit patches that pallet → Close; repeat for the next pallet. Replaces the multi-pallet full-map weigh. Group requires a child Submission before Close. Frozen Config in `LOCKED.md`. No QR; no loadedExtracted changes.

**Blocked by:** 01 (Schema defaults seed on every Task Load).

**Status:** ready-for-agent

- [ ] Feragame `palletWeight` / `weigh` matches frozen flat form + `patch` + `closeRequiresChildSubmission`
- [ ] One Commit weighs exactly one selected pallet; other unweighted pallets stay unweighted
- [ ] After Close, another Spawn can weigh the next pallet; lineLoad still unlocks when the group predecessor law says so
- [ ] Living tests updated off the old full-map weigh helpers
