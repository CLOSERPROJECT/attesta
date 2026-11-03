# Brief: stock clearable only (peel enum-null / Domain unset)

**Date:** 2026-11-02  
**Worktree:** `/Users/giovanniabbatepaolo/.cursor/worktrees/fix-remove-normalize-36fadf53/attesta-714729698316`  
**Branch:** `fix/remove-normalize`  
**Do not push.** Conventional commits. Leave dirty `.scratch/**` unstaged unless editing these briefs.

## Settled law (human, 2026-11-02)

1. **Never unset Domain facts** via Commit. Partial map = set-or-skip only.
2. Cell enum options are **real values only** (`partial`, `done`, …). No `null` in enum.
3. **Empty is client-only:** stock sjsf clearable wipes a bad draft pick → omit/`undefined` → **drop the key**. Same as never touching the row for Commit.
4. **No** `enum: [null,…]`, **no** `clearable: false` stamp, **no** null-via-`setValue` bypass, **no** SPA normalize return.

## Keep (still good)

- Concrete map projection (`materializeDynamicMapSchema`)
- Honest seed (no first-enum densify; no `default: null` densify)
- Delete SPA `form-cell-map` normalize / identity seal
- Demo seed through lineLoad (`demo_commits.go`)
- Go `filterFilledCells` as Commit authority for **enum strings + omit**; not nil-clear

## Peel / delete

| Area | Action |
| --- | --- |
| `feragame-config.yaml` lineLoad progress | `enum: [partial, done]`; `enumNames: [Parziale, Fatto]`; drop null/clear comments; stock clearable default (do not stamp false) |
| `form_project.go` | Remove `stampClearableOffWhenEnumNull`, `enumNullSelectPlaceholder`, `schemaEnumContainsNull`, `schemaHasNestedEnumNull` and call sites from `stampConcreteMapUi` |
| `map-table-cells.svelte.ts` | Scalar: undefined/clear → delete key (or omit write); no `out[rowKey]=null` / dual `setValue` for null; quiet omit↔omit OK |
| `object-row-weigh.ts` | Align quiet with omit-only if still used; null-as-quiet was for wrong law |
| `rows.go` `filterFilledCells` | Treat `nil` like untouched (skip) OR reject; **do not** keep nil as Domain clear. Update comments. |
| `line_load_test.go` | Remove/rewrite “clear must unset progress” — that product is gone |
| `rows_test.go` / `form_bundle_test.go` | Assert no leading null; no clearable:false; enumNames length 2 |
| SPA tests | `progress-*.test.ts`, `map-table-cells.test.ts`, cascade probe — stock clearable / drop key; no null clear validates-as-feature |
| `GLOSSARY.md` Domain reads | Stock clearable for draft fix; omit=untouched; no enum-null; avoid list updated |
| `prep.go` comment | Drop “prefer enum:[null,…]” |

## Out of scope

- FakeHttp, Completo, progress rename, ticket 13, merge-back, push
- Post-Commit jobId (separate peel)
- Re-adding SPA normalize

## Done when

- [x] Tip has no enum-null clear law in fixture/projection/SPA (`71dfd18` + `21804a9`)
- [x] Stock clearable can empty a cell → key absent in draft
- [x] Go does not unset facts from nil cells
- [x] Tests green for touched packages
- [ ] Conventional commit(s); human smoke later
