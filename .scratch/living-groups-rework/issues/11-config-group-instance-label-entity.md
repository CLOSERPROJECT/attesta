# 11: Config declares group-instance label entity (no Feragame in Go)

**What to build:** Remove hardcoded Domain kind `"load"` from Worklist group-instance label projection. Org Config on a `kind: group` node declares which entity kind supplies the overview label (e.g. `instanceLabelEntity: load` on feragame `lots`). Go joins nested Submission `IDsTouched` ∩ that authored kind only. Groups without the field omit `label` (SPA keeps `label || id`).

**Blocked by:** 10

**Status:** done

**Hard call (human 2026-11-03):** Golang living engine must know nothing about Feragame Domain entity vocabulary.

- [x] Config field on Task group (name TBD, prefer `instanceLabelEntity`); parsed/ignored safely when absent
- [x] `groupInstanceLoadLabel` (rename) takes entity kind from Config; no string `"load"` in living projection code
- [x] Feragame `lots` sets the field to `load`; `palletWeight` omits
- [x] Go tests updated; ADR-0006 wording: label from Config-declared entity kind, not hard-coded load
- [x] No SPA Domain walk; wire field remains `label`
