# 14: /app — experiment FormPlan rendering pipeline

**What to build:** Replace hand-wired living `IntakeJob` / `LineJob` smoke forms with the attesta-experiment rendering pipeline so `/app` paints Tasks the same way: FormPlan → live form chrome → sjsf/custom fields → derived-leaf (readonly + overridable). Copy+adapt to living Job/Task nouns; keep living Commit/Load HTTP contracts.

**Blocked by:** flat port 09–11 (done). Coordinate with 13 rename if overlapping.

**Status:** done — reviewer APPROVE; squash into rework; human smoke

- [x] FormPlan (or living equivalent) drives intake + line Task UIs from Org Config, not hand-built field lists
- [x] Custom field widgets used by the truck-loop fixture are available (e.g. leftover-table / map-table equivalents)
- [x] Derived readonly fields and derived overridable fields (override/restore) match experiment behaviour; sealed Commit values include effective derived facts
- [x] Living Commit/Load ports and Cerbos path unchanged in contract (path + Job instance + value)
- [x] No harness / snapshot-replace UI revived
- [x] `bun run check` + living unit tests green; smoke intake + one line instance still works in `/app`

**Out of scope:** ticket 13 placeholder rename; Task groups / lot intake; HTMX `/my` / `web/` templates.

**Refs:** `docs/research/attesta-app-living-product-decisions.md`; experiment ADRs 0016 (derived), 0022 (chrome); source `attesta-experiment`.
