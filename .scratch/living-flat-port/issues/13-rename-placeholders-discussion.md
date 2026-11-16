# 13 rename — discussion notes

**Date:** 2026-10-30 (names locked 2026-11-16; hard-cut amended same day)  
**Ticket:** `.scratch/living-flat-port/issues/13-rename-placeholders.md`  
**Status:** **names agreed — hard cut.** No `/api/living` aliases, no `LIVING_HTTP_*` fallbacks, no dual Cerbos, no Mongo/draft migration.  
**Product copy:** “operating picture” may stay in docs/glossary prose; **code seams use `org`**, not `operation` / `living`.

## Why rename

Flat-port placeholders from decisions dump (`docs/research/attesta-app-living-product-decisions.md`):

| Placeholder | Role |
| --- | --- |
| `server/internal/living` | Go package (Domain / Org Config / Load / Commit HTTP) |
| `app/src/lib/living/` | Client Modules / UI |
| `DomainStore` / `ConfigStore` | Persistence Interfaces |
| `GET /api/living/domain` | Load |
| `POST /api/living/commit` | Commit |
| Cerbos kind `living` | Commit allow/deny |
| `ATTESTA_LIVING_SEED` | Never shipped; demo gate is already `ATTESTA_SEED` |

Rejected forever: `/api/runs`, `RunStore`, `LoadRunPort`, Stream/Process as `/app` product names; bare package `domain` (clashes with Org Config authored `domain` section).

## Settled (2026-11-16)

Scheme A (package-prefixed HTTP). Package is an **owner** noun so `org.DomainStore` reads naturally; vague role types get **glossary objects** so they stay clear under a short package. **Hard cut** — old seams deleted in the same change.

| Seam | Lasting name |
| --- | --- |
| Go package | `server/internal/org` (`package org`) |
| SPA lib | `app/src/lib/org/` |
| Domain persistence | keep `DomainStore` → `org.DomainStore` |
| Config persistence | keep `ConfigStore` → `org.ConfigStore` |
| DPP persistence | `PassportStore` → `DPPStore`; Mongo `org_dpps` |
| Load engine | `DomainLoader` → `org.DomainLoader` |
| Commit engine | `TaskCommitter` → `org.TaskCommitter` |
| HTTP facade | `API` → `org.API` |
| HTTP prefix | `/api/org/…` only (no `/api/living`) |
| Cerbos | kind `task`, action `commit` (delete `living_policy.yaml`) |
| Sidecar | `org-http`; env `ORG_HTTP_*` only |
| SPA session | `OrgSession`, export `orgSession` |
| Demo seed gate | `ATTESTA_SEED` |
| Feragame emails | `admin@feragame.com`, `administration@feragame.com`, `logistic@feragame.com`, `line@feragame.com` |
| Legacy Stream `/my` | **untouched** |

### Target HTTP map

```text
GET  /api/org/domain
GET  /api/org/task-form
GET  /api/org/task-recap
POST /api/org/commit
POST /api/org/close
POST /api/org/qr-resolve
GET  /api/org/artifacts/{id}
```

### Mongo collections

```text
org_domains
org_configs
org_active_config
org_dpps
org_submission_links
```

### How key Go names should read

```text
org.DomainStore
org.ConfigStore
org.DPPStore
org.DomainLoader
org.TaskCommitter
org.API
```

## Hard cut (no compatibility)

- No dual-mount of `/api/living/*`.
- No dual Cerbos kinds; delete `living` policy when flipping to `task`.
- No `LIVING_HTTP_*` env aliases.
- Mongo new names only — reset/re-seed (or drop old `living_*` collections).
- Draft prefix only `attesta:org-draft:` (abandon `attesta:living-draft:`).
- Digital Link order unchanged: org DPP first, else Stream process DPP.

## Explicitly not chosen

- `orgops`, `ops`, `operation` as package
- Glossary-flat HTTP only (`/api/domain`, `/api/commit`)
- Plant-branded package names
- `org_org_configs` (use `org_configs`)

## Resume checklist

- [x] Agree package (+ client lib) name → `org`
- [x] Agree HTTP paths → `/api/org/…` hard cut
- [x] Agree Cerbos → `task` / `commit`
- [x] Agree stores → `DomainStore` / `ConfigStore` / `DPPStore`
- [x] Agree env / seed / emails / Mongo names
- [x] Record lasting table in decisions dump (pointer row)
- [x] Implement ticket 13 (code + docs + smoke green)
