# Coordinator: stock clearable peel

**Brief:** `.scratch/living-worklist/BRIEF-stock-clearable-2026-11-02.md`  
**Worktree:** `/Users/giovanniabbatepaolo/.cursor/worktrees/fix-remove-normalize-36fadf53/attesta-714729698316`  
**Branch:** `fix/remove-normalize` @ `21804a9` (+ uncommitted: en-dash placeholder + Go shadcn expand)

## Done

| Peel | SHA / state | Notes |
| --- | --- | --- |
| SPA MapTableCells | `71dfd18` | Stock clear → delete draft key |
| Go enum-null drop | `21804a9` | Fixture/projection/filter; living tests ok |
| En-dash placeholder | uncommitted | Authored `placeholder: –`; Go expands to shadcn4* |
| Scout leftovers | [scout](3e424781-368f-44e4-8984-bda076a1ddb6) | Soft residue only (below) |

## Law (one line)

Partial map: omit/stock-clearable-empty = skip row; enum string = set fact; never unset Domain. Authored `placeholder` only — Go expands shadcn widgets.

## Scout — optional next peels

1. Drop `anyOf` / null-in-enum cell-map classifier + `TestAdditionalPropertiesIsCellMapAllowsNullAnyOf` (`prep.go`, `rows.go`, `rows_test.go`)
2. Fail closed if cell enum contains null (optional footgun guard)
3. Do **not** remove general PatchOp nil→delete (`mutate.go`) — not form cell-map clear

## Still open

- [ ] Commit uncommitted placeholder/expand work
- [ ] Human smoke on `:10108`
- [ ] Post-Commit jobId (deferred)
- [ ] Merge-back when human OK
