# 02: Unlock `unlockFrom` predecessor

**What to build:** Generic Config opt-in `unlockFrom: <taskId>` so a Task’s Unlock predecessor is that Task’s satisfaction (leaf Submission / group Close), not the previous ordered sibling. Parse + validate. Default linear law unchanged. No plant YAML in this ticket (synthetic fixtures only).

**Blocked by:** Prefer **01** (parallel OK if Worklist/Unlock edits do not fight). Prefer land 01 then 02 then 03.

**Status:** implemented

## Build exactly

1. Locked key name: **`unlockFrom`** (see [`LOCKED.md`](../LOCKED.md)). Optional string on TaskConfig.
2. In `IsTaskUnlocked` (and any path that chooses the predecessor id): if `unlockFrom` is set, resolve predecessor to that id within the same sibling `OrderedIDs` / `Tasks`; else `OrderedIDs[index-1]` as today.
3. Satisfaction law unchanged: leaf predecessor → Submission on instance; group predecessor → group **Close** (not child Submission). Nested group-child Unlock keeps the same opt-in against its sibling list.
4. Validate in `validateTaskConfigs`: when present, `unlockFrom` must name an earlier sibling in the same list (reject missing, self, forward, non-string, empty). First sibling must not author `unlockFrom`.
5. Unit tests (synthetic Job/Task ids only):
   - Fan-out: after predecessor Submission/Close, **two** successors unlock when both `unlockFrom` that predecessor (or one linear + one `unlockFrom`).
   - Default linear unchanged: intake-shaped `lots` (group) → `weightCheck` still requires Close.
   - Validation rejects bad `unlockFrom`.
6. **No** feragame-config.yaml edits; **no** unlock-on-childSubmission.

## Verify

```bash
cd server && go test ./internal/living/ -count=1 -run 'Unlock|ValidateTask|PathUnlock'
cd server && go test ./internal/living/... -count=1
```

## When done

- [x] `unlockFrom` read + used as Unlock predecessor; omit ⇒ linear previous sibling
- [x] Fan-out unit test: two Tasks unlock from same predecessor satisfaction
- [x] Intake lots→weightCheck Close regression still passes
- [x] Config validate covers unknown / self / forward / first-task
- [x] No plant YAML; no Go special-case on plant Task ids
- [x] Group predecessor still requires Close (not first child Submission)
