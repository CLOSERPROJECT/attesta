# 09: Bind transform + containerReg flatten in config

**What to build:** `onSubmit.transform` supports a dumb `bind` step (authored JSONata replaces `$` for later ops). Feragame `containerReg` reshapes keyed material arrays → `{ sourceLoadId, rows }` in config via `$catalogs.materials`, then existing `mintMany`. Go keyed-array flatten and its prep gate are deleted. Container registration still works end-to-end.

**Blocked by:** 08 Org catalogs.

**Status:** ready-for-agent

**Execution:** sequential subagents only (finish A before B).

### Subagent 09A — Bind op

- [ ] Transform list accepts `bind` with a JSONata `expr`; result becomes `$` for following ops
- [ ] Unit tests with synthetic facts prove bind-then-mint/patch without Feragame YAML
- [ ] Unsupported / failing bind surfaces as a normal transform error

### Subagent 09B — Plant cutover

- [ ] `containerReg` YAML uses `bind` + `$catalogs.materials` then `mintMany` on `$.rows`
- [ ] Go containerReg flatten types/helpers and prep special-case gate are gone
- [ ] Commit / fixture tests for dual-source and single-source rows stay green
- [ ] Schema remains the validator (weight / dual secondary); bind only reshapes

**Decisions (grill 2026-11-02):** all reshape in Org Config; no parameterized Go flatten; `material` is fixture vocabulary only; one bind then mintMany (not flatten pasted into every `set` field).

**Out of scope:** Domain kind map; leftovers grammar; schema codegen for material array properties.
