# ADR-0002: Appwrite as the sole source of role colors (org-scoped lookup)

- **Status:** Accepted
- **Date:** 2026-07-06
- **Scope:** Role palette resolution for server-rendered workflow UI (`process`, `action_list`, DPP traceability, home/workflow list): backend lookup from Appwrite, template data shape, and frontend CSS that maps palette keys to `--role-*-bg` / `--role-*-border`. Touches `server/cmd/server/main.go`, `server/cmd/server/dpp.go`, `server/templates/` (workflow surfaces), and `web/src/styles/`.
- **Out of scope:** Formata embed, Formata Builder, and the entire `server/cmd/server/formata-arch/` sub-application (including stream schema, serialization, and Mongo stream documents); org-admin palette picker UI and write path (`role_palette_options.html`, `resolveRolePaletteStyle` on create/update — may continue persisting CSS var strings to Appwrite); `GET /api/catalog` response shape; Cerbos authorization; role slug assignment on substeps (YAML still owns workflow structure); **editing existing workflow YAML files** under `server/config/` (fields may remain for backwards compatibility).
- **Related:** [ADR-0001](0001-css-architecture-refactor.md) (dynamic custom properties — role pills migrate from server-injected `--pill-bg`/`--border` to palette-key classes/attributes on workflow surfaces), `.gestalt/plans/appwrite-identity.org` (identity boundary), `docs/css.md` (allowed inline custom properties).

## Context

Attesta stores org role catalogs in **Appwrite team prefs** (`roles: [{slug, name, color, border}]`). Org admins edit colors in `/org-admin/roles`; the public catalog API (`GET /api/catalog`) already reads live Appwrite data.

Workflow YAML (`server/config/*.yaml` and Mongo-backed stream documents loaded as runtime config) still carries `roles[].color` and `roles[].border`. Runtime rendering resolves badge and timeline colors through `roleMetaMap(cfg)`, which reads **only** from that config — never from Appwrite.

Today the backend also resolves palette choices to **CSS custom-property strings** (`var(--role-blue-bg)`, `var(--role-blue-border)`) and injects them into templates via inline `style="--pill-bg: …; --border: …"`. That duplicates knowledge that already lives in `tokens.css` and ties Go templates to token names.

### Current data flow (broken)

```
Appwrite team prefs          Workflow / stream YAML (runtime config)
        │                              │
        ├─► org-admin UI               ├─► roleMetaMap(cfg)
        └─► GET /api/catalog           │         │
                                       │         ├─► process timeline
                                       │         ├─► action_list badges
                                       │         └─► DPP traceability
                                       │                   │
                                       └───────────────────┘
                                             both emit CSS var strings
                                             into template inline styles
```

When an org admin changes a role palette, org-admin updates immediately but process pages, action cards, and DPP traces keep showing YAML-embedded colors until someone edits the workflow file.

### Additional bug: slug-only lookup

`roleMetaMap` keys by role slug alone (`map[string]RoleMeta`). The same slug under different organizations can have different colors in Appwrite (e.g. `projectmanager` in `stream.yaml` appears under multiple `orgSlug` values). Slug-only lookup makes the last config entry win and is inconsistent with Appwrite's org-scoped catalog.

`substepOrganizationMap` already maps each substep to its step's `organization` slug; callers have org context at render time but do not pass it into role color resolution.

### Intended model (already documented elsewhere)

The Appwrite identity plan states workflow YAML `roles` are **validation-only** for slug/org existence — not the authority for display metadata. `validateWorkflowRefs` enforces slug presence against Appwrite but does not compare colors.

### Palette keys vs CSS tokens

The product uses **17 named palette keys** (`red`, `orange`, `amber`, … `rose`) defined in `rolePaletteStyles` / `role_palette_options.html`. Actual color values live once in `web/src/styles/tokens.css` as `--role-{key}-bg` and `--role-{key}-border`.

**Contract after this ADR:**

| Layer | Responsibility |
|-------|----------------|
| Appwrite | Stores role metadata; `color`/`border` fields may remain as legacy CSS var strings until a separate migration |
| Backend (workflow UI) | Resolves Appwrite → **palette key only** (e.g. `"blue"`); does **not** emit `var(--role-*-*)` strings to workflow templates |
| Frontend CSS | Maps palette key → bg and border tokens (class or `data-role-palette` attribute) |
| `tokens.css` | Single source of hex/oklch values |

## Decision

### 1. Appwrite is the only runtime source for role palette

At render time, role badge and timeline palette **must** be derived from Appwrite team prefs via the identity port. Values in workflow YAML or stream config **`roles[].color` / `roles[].border` must not be read** for display, even when present.

On read, the backend **normalizes** Appwrite `color`/`border` (CSS var strings) to a palette key using existing logic (`rolePaletteKeyFromStyle`). If normalization fails, use key `"red"` or a dedicated `"fallback"` key — not raw YAML colors.

**Fallback when identity is unavailable** (unit tests, `enforceAuth == false` demo, identity store nil): emit palette key `"fallback"` (or empty → CSS default). Do **not** fall back to config-embedded colors or CSS var strings from YAML.

### 2. Backend emits palette key only — not bg/border CSS values

View models for workflow surfaces (`TimelineSubstep`, `ActionRoleBadge`, `DPPTraceabilitySubstep`, etc.) expose a **single palette key field** (e.g. `Palette string`), not `RoleColor` / `RoleBorder` / `Color` / `Border` as `template.CSS` var strings.

```go
// Conceptual — workflow UI view models
type RoleMeta struct {
    ID      string
    Label   string
    Palette string // e.g. "blue", "emerald", "fallback"
}
```

Go must not pass `var(--role-blue-bg)` into workflow templates. Token resolution is a **frontend concern**.

Org-admin templates may continue their current inline preview pattern until separately migrated; that path is out of scope here.

### 3. Frontend maps palette key to bg and border

Add or extend CSS in `web/src/styles/components.css` (or a dedicated `role-palette.css` imported from the layer stack) so workflow templates only set the key:

```html
<!-- Preferred: data attribute -->
<span class="role-pill" data-role-palette="{{ .Palette }}">{{ .Label }}</span>

<!-- Alternative: modifier class -->
<span class="role-pill role-palette--{{ .Palette }}">{{ .Label }}</span>
```

CSS responsibility (example):

```css
.role-pill[data-role-palette="blue"] {
  --pill-bg: var(--role-blue-bg);
  --border: var(--role-blue-border);
}
/* …one rule per key, or a single attribute selector pattern… */
.role-pill:not([data-role-palette]),
.role-pill[data-role-palette="fallback"] {
  --pill-bg: var(--role-fallback);
  --border: var(--border);
}
```

Timeline substeps (`--dept-color` / `--dept-border`) follow the same pattern: `data-role-palette` on `.substep` (or equivalent), CSS sets `--dept-color` and `--dept-border` from the key.

**Remove** workflow-template inline styles that set `--pill-bg` / `--border` / `--dept-color` / `--dept-border` from Go-injected CSS var strings. ADR-0001's allowlist for role pills on workflow surfaces is superseded by this attribute/class approach.

### 4. Org-scoped lookup: `(orgSlug, roleSlug)`

Replace slug-only `map[string]RoleMeta` with an org-scoped index:

```go
type roleMetaKey struct {
    OrgSlug  string
    RoleSlug string
}
// map[roleMetaKey]RoleMeta  — RoleMeta.Palette is the palette key
```

Resolution rules:

| Caller context | Lookup key |
|----------------|------------|
| Substep on a step with `organization: <org>` | `(stepOrg, roleSlug)` |
| Substep with no step org; role unique in workflow | first org from config role refs or identity catalog that contains the slug |
| Org admin | out of scope for this ADR (existing `Palette` on `OrgAdminRoleRow`) |
| Unknown org or missing role in Appwrite | palette key `"fallback"`; label from slug |

`buildTimeline`, `buildActionList`, and `buildDPPTraceabilityView` must pass the substep's organization slug (from `substepOrganizationMap` or step metadata) into role meta lookup.

### 5. Replace `roleMetaMap(cfg)` with identity-backed loading

```go
func (s *Server) roleMetaIndex(ctx context.Context) (map[roleMetaKey]RoleMeta, error)
```

Call sites that today invoke `s.roleMetaMap(cfg)` for palette and labels switch to `roleMetaIndex` plus org-aware `roleMetaFor(orgSlug, roleSlug, index)`.

Labels (`RoleMeta.Label`) come from Appwrite `name`, not config `name`.

### 6. Ignore config-embedded colors; do not edit workflow YAML files

**Runtime behavior:** no code path may use `roles[].color`, `roles[].border`, or legacy `departments[].color` / `departments[].border` for workflow UI palette.

**File compatibility:** existing workflow YAML under `server/config/` and stream documents in Mongo **are not modified**. `color` and `border` fields may remain inert.

`validateWorkflowRefs` behavior unchanged for slug checks.

### 7. Public catalog API — no behavioral change

`handlePublicCatalog` may continue returning Appwrite `color`/`border` strings for out-of-scope consumers (e.g. Formata). Workflow server-rendered UI does not use that API for palette.

## Consequences

### Positive

- Single source of truth for palette **identity**: Appwrite; single source for palette **appearance**: `tokens.css`.
- Org-admin palette edits appear immediately on process, timeline, action list, and DPP without YAML edits.
- Org-scoped lookup fixes wrong colors when the same role slug exists in multiple organizations.
- Backend templates become simpler — no `template.CSS` color strings or `cssValue()` for role badges on workflow surfaces.
- Adding a new palette key requires CSS + picker options, not Go string passthrough to every view model.

### Negative / trade-offs

- **Runtime dependency on identity** for palette keys on workflow pages; degrade to `"fallback"`, not YAML.
- **Extra I/O** unless request-scoped `roleMetaIndex` caching is used.
- **Demo/offline mode** shows fallback palette for all roles, not config-embedded colors.
- **Two persistence shapes temporarily:** Appwrite may still store CSS var strings; backend derives keys on read until a future migration stores `palette` directly in team prefs.
- **Template + CSS migration** in the same delivery as backend — partial rollout would break styling.

### Risks and mitigations

| Risk | Mitigation |
|------|------------|
| Identity outage | Palette key `"fallback"`; log once per request |
| Performance | Request-scoped `roleMetaIndex`; reuse across timeline + action list + DPP |
| Tests assert on `#aaaaaa` or CSS var strings | Assert on palette key in view models; template tests check `data-role-palette` |
| Same slug, different org | Unit tests with `substepOrganizationMap` + multi-org `stream.yaml` fixture |
| Unknown Appwrite color string | `rolePaletteKeyFromStyle` + default `"red"` or `"fallback"` |
| CSS missing a palette key | Lint or test that every `rolePaletteKeys` entry has a CSS rule |

## Implementation plan

### Pass 1 — Backend resolution

1. Add `roleMetaKey`, `roleMetaIndex(ctx)`, org-aware `roleMetaFor`.
2. `RoleMeta` carries `Palette` key only; stop populating `RoleColor`/`RoleBorder` on workflow view models (or deprecate those fields).
3. Derive key from Appwrite via `rolePaletteKeyFromStyle(color, border, name)`.
4. Thread `orgSlug` through `buildTimeline`, `buildActionList`, `buildDPPTraceabilityView`.
5. Retire color reads from `roleMetaMap` for display.

### Pass 2 — Frontend + templates

1. Add palette-key → token mapping in CSS (`data-role-palette` or `role-palette--*` modifiers).
2. Update `process.html`, `action_list.html`, `dpp.html` — replace inline `--pill-bg`/`--dept-color` from Go with palette attribute/class.
3. Update `docs/css.md` allowlist and `deployment/scripts/check-template-inline-styles.sh` if needed.

### Pass 3 — Tests and docs

1. Builder/template tests assert palette keys, not CSS var strings.
2. `docs/css.md`, `AGENTS.md` — Appwrite → palette key → CSS tokens.
3. Cross-link from `.gestalt/plans/appwrite-identity.org`.

## Acceptance criteria

- [ ] Org-admin palette change updates workflow UI colors without editing YAML.
- [ ] Same role slug in different orgs renders distinct palettes on a multi-org workflow.
- [ ] Workflow templates contain **no** server-injected `var(--role-*-*)` strings for role badges or timeline substeps.
- [ ] View models expose palette key (e.g. `Palette: "blue"`); CSS maps key to `--role-blue-bg` and `--role-blue-border`.
- [ ] No changes to `server/config/*.yaml`.
- [ ] `task cover` passes; display tests use palette keys not hex/CSS vars.
- [ ] Offline/demo mode uses `"fallback"` palette, not config-embedded colors.

## References

- `server/cmd/server/main.go` — `roleMetaMap`, `roleMetaFor`, `rolePaletteKeyFromStyle`, `buildTimeline`, `buildActionList`
- `server/cmd/server/dpp.go` — `buildDPPTraceabilityView`
- `web/src/styles/tokens.css` — `--role-{key}-bg` / `--role-{key}-border`
- `server/templates/process.html`, `action_list.html`, `dpp.html`
- `.gestalt/plans/appwrite-identity.org` — team prefs role catalog
