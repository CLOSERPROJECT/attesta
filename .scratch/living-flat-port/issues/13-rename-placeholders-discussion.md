# 13 rename — discussion notes (paused)

**Date:** 2026-10-30  
**Ticket:** `.scratch/living-flat-port/issues/13-rename-placeholders.md`  
**Status:** discussion paused — more pressing work first; **do not implement rename** until open decisions below are settled.  
**Product smoke** on `feat/attesta-rework` may still be pending; rename was gated on smoke + agreed names.

## Why rename

Flat-port placeholders from decisions dump (`docs/research/attesta-app-living-product-decisions.md`):

| Placeholder | Role |
| --- | --- |
| `server/internal/living` | Go package (Domain / Org Config / Load / Commit HTTP) |
| `app/src/lib/living/` | Client living Modules / UI |
| `DomainStore` / `ConfigStore` | Persistence Interfaces |
| `GET /api/living/domain` | Load |
| `POST /api/living/commit` | Commit |
| Cerbos kind `living` | Commit allow/deny |
| `ATTESTA_LIVING_SEED` | Domain wipe gate (ticket 12) |

Rejected forever: `/api/runs`, `RunStore`, `LoadRunPort`, Stream/Process as living product names.

## Settled in this discussion

1. **`living` is out** — bad lasting product/engineering label; rename away from package, HTTP prefix, Cerbos kind, env, and client lib path.
2. **Package must not be named `domain` alone** — clashes with Org Config’s authored **`domain`** section (entity shape). Org Config is more than Domain: **roles + jobs/tasks + domain**. Runtime **Domain** store is a different noun. A package called `domain` collapses authored config vs accumulating Domain and undersells Jobs/Commit.

## Open decisions (need human)

1. **Go package name** (and matching `app/src/lib/…` if we keep them aligned)  
   Candidates floated: `orgops`, `ops`, `operation`, `work`, `runtime`, plant-flavoured names (too demo-specific), or a short product brand word.  
   Constraint: means “org operating picture” (Config + Domain + Load/Commit), not Domain-the-noun and not `living`.

2. **HTTP paths**  
   Options:  
   - Prefix matches package (`/api/<pkg>/domain` + `/api/<pkg>/commit`)  
   - Glossary-short without package echo (`/api/domain` + `/api/commit`, org from session)  
   Load response is still Domain + Active Config either way.

3. **Cerbos resource kind**  
   Lean from discussion: kind `task`, action `commit` (glossary: Cerbos targets leaf Tasks). Alternatives: keep a product-prefixed kind once package name exists.

4. **Store Interface names**  
   - `DomainStore` — likely keep (glossary “Domain store”)  
   - `ConfigStore` → `OrgConfigStore`? undecided

5. **Env wipe flag**  
   Replace `ATTESTA_LIVING_SEED` with something that follows the new adjective (e.g. `ATTESTA_<PKG>_SEED` or `ATTESTA_DOMAIN_SEED` — latter risks the same Domain clash in ops docs).

6. **When to rename vs ticket 14 (FormPlan)**  
   Coordinate so FormPlan port doesn’t land on paths that immediately rename; order undecided.

## Soft lean (not agreed)

- Drop `living` everywhere.  
- Cerbos → kind `task` / action `commit`.  
- Keep `DomainStore`; consider `OrgConfigStore`.  
- Package: waiting on a name that is not `living` / not bare `domain`.

## Resume checklist

- [ ] Agree package (+ client lib) name  
- [ ] Agree HTTP paths  
- [ ] Agree Cerbos kind (+ action)  
- [ ] Agree store Interface renames (if any)  
- [ ] Agree env flag name  
- [ ] Record lasting table in decisions dump (and/or glossary-adjacent note)  
- [ ] Then implement ticket 13 (code + docs + smoke green)
