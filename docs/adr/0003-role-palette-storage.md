# ADR-0003: Palette-only role storage and YAML color removal

- **Status:** Accepted
- **Date:** 2026-07-07
- **Scope:** Appwrite team prefs role shape, org-admin write/read paths, `GET /api/catalog`, workflow YAML loading (Go structs only — no file edits), org-admin template styling, and Formata Builder role schema/serialization. Builds on [ADR-0002](0002-role-color-appwrite-source.md) workflow UI resolution (already on `feat/roles-colors`).
- **Out of scope:** Cerbos authorization; role slug assignment on substeps; editing existing files under `server/config/*.yaml`; `tokens.css` palette definitions; org-admin palette picker UX (still 17 keys in `role_palette_options.html`); one-shot production migration scripts for Appwrite.
- **Related:** [ADR-0002](0002-role-color-appwrite-source.md) (workflow UI reads Appwrite → palette key → CSS), [ADR-0001](0001-css-architecture-refactor.md) (`data-role-palette` pattern), [implementation plan](../plans/0003-role-palette-storage.md), `.gestalt/plans/appwrite-identity.org`, `docs/css.md`.

## Context

[ADR-0002](0002-role-color-appwrite-source.md) moved **workflow rendering** to Appwrite-backed palette keys (`RoleMeta.Palette`, `data-role-palette` on `process`, `action_list`, `dpp`). That ADR intentionally left several areas transitional:

| Area | ADR-0002 state |
|------|----------------|
| Appwrite team prefs | Still stores `color` / `border` as CSS var strings (`var(--role-blue-bg)`, …) |
| Org-admin write path | Expands palette picker value → CSS vars before `UpdateOrganization` |
| Org-admin display | Inline `style="--pill-bg: …; --border: …"` from `RoleColor` / `RoleBorder` |
| `GET /api/catalog` | Returns `color` / `border` for Formata and other consumers |
| Workflow YAML | `roles[].color` / `border` still unmarshaled; legacy `departments` migration copies colors into `cfg.Roles` |
| Formata Builder | Requires `color` / `border` on roles; embeds them in saved stream YAML |

This leaves **three representations** of the same concept (palette key, CSS var pair, YAML fields) and allows Formata to re-introduce colors into stream documents after org admins have centralized palette in Appwrite.

### Intended model (refined)

| Layer | Responsibility |
|-------|----------------|
| Appwrite team prefs | Canonical store: `{ slug, name, palette }` — palette is a named key (`blue`, `emerald`, …), not a color value |
| Backend | Resolves palette on read; never persists CSS var strings to Appwrite on write |
| `tokens.css` + `components.css` | Single source of appearance; maps palette key → `--role-*-bg` / `--role-*-border` |
| Workflow YAML | Slug/org/name for validation only; **color fields not loaded** by Go |
| Formata Builder | Steps/substeps/roles for structure only; **no color management** |

## Decision

### 1. Appwrite stores `palette` only (not `color` / `border`)

Team prefs role entries use:

```json
{ "slug": "chemist", "name": "Chemist", "palette": "blue" }
```

- Field name is **`palette`** (matches ADR-0002 view models, CSS `data-role-palette`, org-admin form field).
- New writes set `palette` and **omit** `color` / `border`.
- `schemaVersion` in team prefs **does not require a bump**; `palette` is additive. Legacy entries may still carry `color` / `border` until edited.

### 2. Legacy read fallback; strip on write (no migration script)

On **read**, resolve palette in order:

1. If `palette` is set and valid → use it.
2. Else if legacy `color` / `border` present → derive via `rolePaletteKeyFromStyle()`.
3. Else → `"fallback"` (or hash-derived default only when creating a new role without a picker value).

On **write** (org-admin create/update role), persist `palette` only and do not write `color` / `border`. Unedited legacy roles in the same org may retain old fields until that role is saved again.

No one-shot Appwrite migration script is required.

### 3. Stop loading color fields from workflow YAML

- Remove `Color` / `Border` from Go config structs (`WorkflowRole`, `Department`, `WorkflowOrganization`) or mark them `yaml:"-"` so unmarshaling ignores them.
- `normalizeWorkflowConfig()` must not copy `departments[].color` / `border` into synthesized `cfg.Roles`.
- **Do not edit** existing `server/config/*.yaml` or Mongo stream files; in-file `color` / `border` become inert.
- `validateWorkflowRefs` continues to check slug/org existence only.

### 4. Catalog API returns `palette` only

`GET /api/catalog` role objects:

```json
{ "orgSlug": "organization-1", "slug": "chemist", "name": "Chemist", "palette": "blue" }
```

Remove `color` and `border` from `PublicCatalogRole`. This is a **breaking change** for catalog consumers.

### 5. Formata Builder does not manage colors

Formata is concerned with workflow structure (steps, substeps, role slugs) only. In this delivery:

- Remove `color` / `border` from Formata `Role` JSON Schema, zod fetch schema, mocks, and sample YAML.
- Saved stream documents must not require or emit role color fields.
- Formata UI does not display role palette swatches.

### 6. Org-admin uses `data-role-palette` (same as workflow UI)

Migrate `org_admin.html` role pills from inline `--pill-bg` / `--border` to `data-role-palette="{{ .Palette }}"`, reusing `components.css` rules.

Remove `RoleColor` / `RoleBorder` from `OrgAdminRoleRow` and `OrgAdminRoleOption` view models. `resolveRolePaletteStyle()` remains for internal validation of picker values, not for template injection.

### 7. Invalid or missing palette → `"fallback"`

When identity is unavailable or a role cannot be resolved, workflow and org-admin surfaces use palette key `"fallback"` (CSS default), not config-embedded colors and not a hash-derived `"red"` except when **creating** a role with no palette submitted (`defaultRolePaletteFromInput`).

## Consequences

### Positive

- Single persistence shape for role appearance identity: `palette` in Appwrite.
- No CSS var strings round-tripping through Go, Appwrite, catalog, or Formata streams.
- YAML and stream documents cannot accidentally become the source of truth for colors again.
- Org-admin and workflow UI share one styling mechanism (`data-role-palette`).

### Negative / trade-offs

- **Breaking catalog API** for any external consumer expecting `color` / `border`.
- **Formata change required** in the same effort (previously out of scope in ADR-0002).
- **Legacy Appwrite rows** may carry stale `color` / `border` until each role is edited.
- **Internal `Role` struct** (`store.go`) and `rolesFromIdentityOrg()` need a `Palette` field instead of `Color` / `Border`.

### Risks and mitigations

| Risk | Mitigation |
|------|------------|
| Legacy team prefs break org-admin | Read fallback from `color`/`border` before `palette` |
| Formata save fails on old streams with role colors | Formata schema drops required `color`/`border`; serde ignores extra YAML keys |
| Org-admin pills unstyled after template change | Reuse existing `.role-pill[data-role-palette]` rules; update template tests |
| Tests assert stored `Color`/`Border` | Assert `Palette` on `IdentityRole` and `data-role-palette` in HTML |

## Supersedes / amends ADR-0002

ADR-0002 §7 (“Public catalog API — no behavioral change”) and its “two persistence shapes temporarily” consequence are **superseded** by this ADR.

ADR-0002 org-admin inline preview exception (“out of scope”) is **closed** by §6 here.

## Acceptance criteria

- [x] Appwrite role writes persist `palette` only (no `color` / `border` on new/updated roles).
- [x] Legacy Appwrite roles with `color`/`border` still resolve to correct palette on read.
- [x] `GET /api/catalog` returns `palette`; no `color` / `border`.
- [x] Go does not load `color` / `border` from workflow YAML structs.
- [x] No changes to `server/config/*.yaml` files.
- [x] Org-admin role pills use `data-role-palette`; no `RoleColor` / `RoleBorder` in view models.
- [x] Formata schema, mocks, and saved streams do not require role `color` / `border`.
- [x] `task cover` passes; template lint passes.
- [x] Manual: org-admin palette change → workflow timeline / action list / DPP / org-admin pills all update without YAML edit.

## References

- `server/cmd/server/identity.go` — `IdentityRole`
- `server/cmd/server/role_meta.go` — `roleMetaIndex`
- `server/cmd/server/main.go` — org-admin handlers, catalog, `normalizeWorkflowConfig`, `rolePaletteKeyFromStyle`
- `server/templates/org_admin.html`
- `server/cmd/server/formata-arch/src/core/config/schema.ts`
- `server/cmd/server/formata-arch/src/core/api/index.ts`
- `deployment/appwrite/appwrite-seed.sql`
