# Plan: ADR-0003 — Palette-only role storage and YAML color removal

- **ADR:** [0003-role-palette-storage.md](../adr/0003-role-palette-storage.md)
- **Prerequisite:** [ADR-0002](../adr/0002-role-color-appwrite-source.md) workflow UI work on branch `feat/roles-colors` (commit `30605ee` or later): `roleMetaIndex`, `data-role-palette` on process/action_list/dpp, `components.css` palette rules.
- **Branch:** continue on `feat/roles-colors` or cut `feat/role-palette-storage` from it.
- **Date:** 2026-07-07

## Goal

Complete the role-color architecture: **Appwrite stores palette keys only**, **YAML does not load colors**, **catalog and Formata do not carry colors**, **org-admin uses `data-role-palette`**.

## Settled decisions (from design review)

| Topic | Decision |
|-------|----------|
| Catalog + Formata | No color fields; catalog returns `palette` only; Formata updated in this effort |
| Legacy Appwrite data | Read-time fallback from `color`/`border`; strip on write; no migration script |
| Field name | `palette` everywhere (not `colorKey`) |
| Org-admin UI | `data-role-palette` in same PR; drop `RoleColor`/`RoleBorder` view models |
| Invalid/missing palette | `"fallback"` at render time |
| YAML files | Do not edit `server/config/*.yaml`; ignore color fields in Go unmarshaling |
| `schemaVersion` | No bump; `palette` is additive |
| Internal `Role` struct | `Palette` replaces `Color`/`Border` in bridge types |

## Current vs target

### Appwrite team prefs (per role)

```
# Before
{ "slug": "chemist", "name": "Chemist", "color": "var(--role-blue-bg)", "border": "var(--role-blue-border)" }

# After (write)
{ "slug": "chemist", "name": "Chemist", "palette": "blue" }
```

### Catalog API (`GET /api/catalog`)

```
# Before                          # After
{ color, border, ... }     →      { palette, ... }
```

### Workflow YAML (Go loading only)

```
# Files unchanged; Go ignores:
roles[].color, roles[].border, departments[].color, departments[].border
```

### Formata stream `roles` section

```
# Before                          # After
{ slug, name, orgSlug, color, border }  →  { slug, name, orgSlug }
```

---

## Phase 1 — Core resolution helper

**Objective:** Single function for palette resolution used by `roleMetaIndex`, org-admin builders, and catalog.

### 1.1 Add `resolveRolePalette(role IdentityRole) string`

Location: `server/cmd/server/main.go` (near `rolePaletteKeyFromStyle`).

```go
func resolveRolePalette(role IdentityRole) string {
    if key := canonifySlug(role.Palette); key != "" {
        if _, ok := rolePaletteStyles[key]; ok {
            return key
        }
    }
    if key := rolePaletteKeyFromStyle(role.Color, role.Border, role.Name); key != "" {
        return key
    }
    return "fallback"
}
```

### 1.2 Extend `IdentityRole`

`server/cmd/server/identity.go`:

```go
type IdentityRole struct {
    Slug    string `json:"slug"`
    Name    string `json:"name"`
    Palette string `json:"palette,omitempty"`
    Color   string `json:"color,omitempty"`  // legacy read only
    Border  string `json:"border,omitempty"` // legacy read only
}
```

Keep `Color`/`Border` on struct for legacy JSON unmarshaling; do not write them.

### 1.3 Tests

- `resolveRolePalette` with `palette` set → returns key
- legacy `color`/`border` only → derived key
- unknown values → `"fallback"`
- Update `server/cmd/server/role_meta_test.go` to include `Palette`-only fixtures

**Exit criteria:** `go test ./cmd/server -run 'RolePalette|RoleMeta'` passes.

---

## Phase 2 — Appwrite write path (org-admin)

**Objective:** Persist `palette` only on role create/update.

### 2.1 `handleOrgAdminRoles` — `create_role` / `set_role`

File: `server/cmd/server/main.go` (~lines 4216–4308).

Replace:

```go
paletteStyle := resolveRolePaletteStyle(palette)
IdentityRole{ ..., Color: paletteStyle.Color, Border: paletteStyle.Border }
```

With:

```go
IdentityRole{ Slug: roleSlug, Name: name, Palette: canonifySlug(palette) }
```

On `set_role`, when updating the matched role, set `Palette` and clear `Color`/`Border` on that struct before encode (empty strings → `omitempty` drops them from JSON).

### 2.2 `buildOrgAdminRoleRows` / `buildOrgAdminRolePills`

- Input: `rolesFromIdentityOrg()` with `Palette` field.
- Output: `OrgAdminRoleRow{ Palette: resolveRolePalette(...) }` only.
- **Remove** `RoleColor`, `RoleBorder` from `OrgAdminRoleRow` and `OrgAdminRoleOption` structs.

### 2.3 `rolesFromIdentityOrg` + internal `Role`

`server/cmd/server/store.go`:

```go
type Role struct {
    // ...
    Palette string `bson:"palette,omitempty"`
    // remove Color, Border (or deprecate with bson:"-")
}
```

`rolesFromIdentityOrg()` maps `Palette: resolveRolePalette(identityRole)`.

### 2.4 `role_meta.go`

```go
Palette: resolveRolePalette(role),
```

Remove direct `rolePaletteKeyFromStyle(role.Color, ...)` call.

### 2.5 Tests to update

| File | Change |
|------|--------|
| `admin_handler_identity_test.go` | Assert `updatedRoles[0].Palette == "…"` instead of `Color`/`Border` non-empty |
| `org_admin_template_role_style_test.go` | Assert `data-role-palette="emerald"` instead of inline `--pill-bg` |
| `org_admin_template_role_usage_test.go` | Same |
| `org_admin_template_sidebar_test.go` | Same |
| `identity_mapping_test.go` | Prefs round-trip with `palette` |
| `identity_appwrite_test.go` | Fixture JSON uses `palette` where testing writes |

**Exit criteria:** org-admin role CRUD tests pass; no test requires stored CSS var strings.

---

## Phase 3 — Catalog API

**Objective:** Breaking change to catalog response.

### 3.1 `PublicCatalogRole`

`server/cmd/server/main.go`:

```go
type PublicCatalogRole struct {
    OrgSlug string `json:"orgSlug"`
    Name    string `json:"name"`
    Slug    string `json:"slug"`
    Palette string `json:"palette"`
}
```

### 3.2 `handlePublicCatalog`

Map `Palette: resolveRolePalette(role)`; remove `Color`/`Border` assignment.

### 3.3 Tests

- `public_catalog_handler_test.go`
- `helpers_misc_coverage_test.go` (if catalog covered)

**Exit criteria:** catalog JSON tests expect `palette` key only.

---

## Phase 4 — Org-admin template

**Objective:** Same styling model as workflow surfaces.

### 4.1 `server/templates/org_admin.html`

Replace every:

```html
style="--pill-bg: {{ .RoleColor }}; --border: {{ .RoleBorder }};"
```

With:

```html
data-role-palette="{{ .Palette }}"
```

on elements that already have class `role-pill` (or add class where missing).

Palette picker JS (`data-role-color` on options) stays for **client-side swatch preview** in the dropdown only — it does not persist colors.

### 4.2 `docs/css.md`

Remove org-admin exception for `--pill-bg` / `--border` inline styles on role pills (or note that only the palette picker dropdown may use transient client-side vars).

### 4.3 `deployment/scripts/check-template-inline-styles.sh`

Ensure org-admin role pills are not allowlisted for `--pill-bg` on `.role-pill` rows (picker internals may still need `--pill-bg` on swatch elements — verify script scope).

**Exit criteria:** `bash deployment/scripts/check-template-inline-styles.sh` passes; org-admin template tests assert `data-role-palette`.

---

## Phase 5 — Stop loading YAML colors

**Objective:** Config files unchanged; Go ignores color fields.

### 5.1 Struct changes (`server/cmd/server/main.go`)

```go
type WorkflowRole struct {
    OrgSlug string `yaml:"orgSlug"`
    Slug    string `yaml:"slug"`
    Name    string `yaml:"name"`
}

type Department struct {
    ID   string `yaml:"id"`
    Name string `yaml:"name"`
}

type WorkflowOrganization struct {
    Slug string `yaml:"slug"`
    Name string `yaml:"name"`
}
```

Alternatively keep fields with `yaml:"-"` if removal breaks compile elsewhere — prefer deletion if grep shows no non-display usage.

### 5.2 `normalizeWorkflowConfig()`

When synthesizing roles from `departments`, copy only `OrgSlug`, `Slug`, `Name`. Remove default org `Color`/`Border` injection (~lines 6923–6929) if only used for display.

### 5.3 Verify grep-clean

```bash
rg 'role\.Color|dept\.Color|WorkflowRole.*Color' server/cmd/server --glob '*.go'
```

Expected: no hits outside legacy `IdentityRole` / `resolveRolePalette` fallback.

**Exit criteria:** `task cover` passes; config load tests unchanged (YAML files still parse).

---

## Phase 6 — Formata Builder

**Objective:** Formata does not manage or persist role colors.

### 6.1 JSON Schema — `formata-arch/src/core/config/schema.ts`

`Role` def:

```ts
required: ['name', 'slug', 'orgSlug'],
properties: {
  name: { type: 'string' },
  slug: { type: 'string' },
  orgSlug: { type: 'string' },
},
```

Remove `color`, `border` from `required` and `properties`. Set `additionalProperties: true` temporarily if old streams still carry inert keys (prefer `false` if serde strips unknowns).

### 6.2 API fetch schema — `formata-arch/src/core/api/index.ts`

```ts
const RoleSchema = z.object({
  orgSlug: z.string(),
  name: z.string(),
  slug: z.string(),
  palette: z.string().optional(), // catalog may include palette; Formata ignores for UI
});
```

Or omit `palette` entirely if Formata only needs slug/org/name for structure.

### 6.3 Mocks and samples

| File | Action |
|------|--------|
| `catalog.mock.json` | Remove `color`/`border` from roles; add `palette` if catalog includes it |
| `stream.mock.json` | Remove `color`/`border` from roles array |
| `config.sample.yaml` | Remove role color/border lines |
| Embedded stream YAML in `catalog.mock.json` | Strip `color`/`border` from `roles:` blocks |

### 6.4 `app.svelte.ts` `buildConfig()`

No change to logic if it copies full `Config.Role` objects — roles will simply lack color fields. Verify `Config.serialize` output has no role colors.

### 6.5 Formata tests

- `serde.test.ts` — round-trip without colors
- `schema.test.ts` — validates sample without color fields
- Run `cd server/cmd/server/formata-arch && bun run test`

**Exit criteria:** Formata tests pass; saved stream YAML has no `color`/`border` under `roles`.

---

## Phase 7 — Seed data and fixtures

### 7.1 `deployment/appwrite/appwrite-seed.sql`

Update team prefs JSON: replace `color`/`border` pairs with `"palette":"blue"` etc.

### 7.2 Any other fixtures

Grep repo for `"var(--role-` in JSON/SQL test fixtures; update to `palette` where representing Appwrite prefs.

---

## Phase 8 — Documentation

| Doc | Update |
|-----|--------|
| `docs/adr/0003-role-palette-storage.md` | Tick acceptance criteria when done |
| `docs/css.md` | Org-admin uses `data-role-palette`; catalog note |
| `AGENTS.md` | One-line: Appwrite `palette` → CSS via `data-role-palette` |
| `.gestalt/plans/appwrite-identity.org` | Prefs shape: `roles: [{slug,name,palette}]` |

---

## Implementation order

```
Phase 1 (resolveRolePalette)
    ↓
Phase 2 (write path + view models) ──→ Phase 4 (org_admin template)
    ↓
Phase 3 (catalog API)
    ↓
Phase 5 (YAML ignore)
    ↓
Phase 6 (Formata)
    ↓
Phase 7 (seed) + Phase 8 (docs)
```

Phases 2 and 4 can be one commit. Phase 6 can parallelize with 5 after Phase 3 lands.

## Verification

```bash
# Backend
cd server && task cover

# Template inline-style lint
bash deployment/scripts/check-template-inline-styles.sh

# Formata
cd server/cmd/server/formata-arch && bun run test

# Frontend bundle (if CSS touched)
cd web && npm run build
```

### Manual QA

1. Org with legacy `color`/`border` prefs → open `/org-admin/roles` → pills show correct colors via `data-role-palette`.
2. Change a role palette → save → workflow process page timeline/badge updates without YAML edit.
3. Multi-org workflow (`stream.yaml`): same role slug, different orgs → different palettes.
4. Formata Builder: load catalog, build workflow, save stream → inspect YAML has no role `color`/`border`.
5. `GET /api/catalog` (authenticated) → roles have `palette`, not `color`/`border`.

## File checklist

| File | Phase |
|------|-------|
| `server/cmd/server/identity.go` | 1 |
| `server/cmd/server/main.go` | 1–5 |
| `server/cmd/server/role_meta.go` | 1 |
| `server/cmd/server/store.go` | 2 |
| `server/templates/org_admin.html` | 4 |
| `server/cmd/server/formata-arch/src/core/config/schema.ts` | 6 |
| `server/cmd/server/formata-arch/src/core/api/index.ts` | 6 |
| `server/cmd/server/formata-arch/src/core/api/catalog.mock.json` | 6 |
| `server/cmd/server/formata-arch/src/core/api/stream.mock.json` | 6 |
| `server/cmd/server/formata-arch/src/core/config/config.sample.yaml` | 6 |
| `deployment/appwrite/appwrite-seed.sql` | 7 |
| `docs/css.md`, `AGENTS.md` | 8 |

## Out of scope reminders

- Editing `server/config/*.yaml` on disk
- Cerbos / substep role assignment
- New palette keys beyond existing 17
- Appwrite production migration script
- Formata UI color swatches (explicitly excluded)
