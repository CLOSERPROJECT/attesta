# 04: Lock Load recap = live passport.snapshot

**What to build:** On Passport Lock Task Load, operator recap is the same `passport.snapshot` projection evaluated live (Chrome / `dataReport`) as the value frozen on Commit—one authored expression, no second drift-prone recap expr. SPA displays Config-driven recap; Go evaluates projection generically.

**Blocked by:** 02 (root `passport.snapshot` in Org Config and evaluation path for freeze).

**Status:** done (`e863138`; Spec+Standards APPROVE 2026-11-04)

- [x] Load bundle / dataReport exposes live snapshot for Lock Task from Org Config `passport.snapshot`
- [x] Commit freeze uses identical projection input as operator saw on Load (tests or fixture assert parity)
- [x] No hand-authored Lock-only recap separate from snapshot block
- [x] Dumb executor: no Feragame Lock Task id special cases in Go or SPA
