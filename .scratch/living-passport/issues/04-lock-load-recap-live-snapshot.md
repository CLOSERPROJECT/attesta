# 04: Lock Load recap = live passport.snapshot

**What to build:** On Passport Lock Task Load, operator recap is the same `passport.snapshot` projection evaluated live (Chrome / `dataReport`) as the value frozen on Commit—one authored expression, no second drift-prone recap expr. SPA displays Config-driven recap; Go evaluates projection generically.

**Blocked by:** 02 (root `passport.snapshot` in Org Config and evaluation path for freeze).

**Status:** ready-for-agent

- [ ] Load bundle / dataReport exposes live snapshot for Lock Task from Org Config `passport.snapshot`
- [ ] Commit freeze uses identical projection input as operator saw on Load (tests or fixture assert parity)
- [ ] No hand-authored Lock-only recap separate from snapshot block
- [ ] Dumb executor: no Feragame Lock Task id special cases in Go or SPA
