# 01: Prefactor — Config-authored Task initialData from Domain

**What to build:** Org Config can seed a Task form’s `initialData` from Domain (and catalogs) via a generic authored expression, including array rows. A synthetic fixture Task proves seed works end-to-end on Load—no plant `containerReg` yet. Go stays a dumb evaluator of that Config; no Feragame-named branches.

**Blocked by:** None (can start immediately).

**Status:** done

- [x] Authored Config mechanism seeds Task bundle `initialData` from `$domain` / `$catalogs` (omit/blank ⇒ today’s schema-default seed only)
- [x] Array (and object) shapes can be prefilled, not only dynamicProperties map cells
- [x] Synthetic fixture + Load/Commit-adjacent tests prove seed without plant Task ids
- [x] No plant-named Go/TS; existing Tasks that omit the mechanism behave unchanged
