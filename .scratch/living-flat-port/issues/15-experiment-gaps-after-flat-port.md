# 15: Experiment gaps after flat port (scout)

**What:** Inventory of attesta-experiment capabilities still missing from the flat port on `feat/attesta-rework`. Spawn implement slices from here — not a single mega-ticket.

**Status:** revised 2026-10-30 (server vs client ownership clarified; Feragame hardcoding called out)  
**Scouts:** [Scout experiment living gaps](c89680ee-85e4-4728-9346-2fd21b7fe876), [Find Feragame hardcoding](439d3054-47f1-4df3-819b-de317a82141e)

**Already tracked / deferred:**
- FormPlan / sjsf / custom / derived-leaf UI → ticket **14**
- Placeholder rename (drop `living`) → ticket **13** (paused)
- Task groups / lot intake → product deferred

**Cross-cutting problem:** Commit/Load path is **not Org-Config-driven**. Go `switch` + `apply*Commit` and bespoke `IntakeJob`/`LineJob` hardcode Feragame truck-loop (Task paths, entity slots, leftovers `where`, mint ids `NIR-`/`C-n`, materials, shift enums, Italian copy). Fixture YAML still carries ELT / `http` / `chrome` / `readyWhen` / `reads` intent that is largely **not executed**. See ticket **16**.

---

## How to read this list

- **Server authority** = production Commit/Load/persist/Cerbos predicates in Go. Experiment’s Kit `+server` / Local Commit are *behaviour reference*; attesta’s authority is Go.
- **Client** = SPA Adapters, FormPlan/chrome UI, Drafts, Vitest mirrors. Must not become a second plant.
- Items wrongly parked under “client” in the first scout are moved to **Server** when they are Commit-path law (ELT, HttpPort after mutate, readyWhen→attrs, leftover projection for gates, instanceId/label exprs, durable stores).

---

## Server authority (Go)

| Gap | Experiment | Attesta today |
| --- | --- | --- |
| **ELT / transform runner** from Task `onSubmit.transform` | YAML mint/mintMany/patch + JSONata via transform-runner | **Hand `apply*Commit` switch** per Task path |
| **Leftover entity projection** from `reads.leftovers.where` | Config-driven | **Hardcoded** palletWeight / lineLoad filters |
| **instanceId / label exprs** from Org Config | JSONata on Job/Task | **Hardcoded** (`Turno: `, NIR-/client mint, `C-%d`) |
| **Domain entity map** from config `domain.entities` | Config-shaped | **Fixed** slots `load` / `pallet` / `container` |
| **readyWhen → Cerbos boolean attr** | ready-when + authorizer | **None** (roles + `sequenceOk` only) |
| **HttpPort after successful mutate** | `onSubmit.http` → links/downloads | Fixture has `http:`; **server returns none** |
| **Durable DomainStore / OrgConfigStore** | SQLite RunStore (hist.); decisions: Mongo | **Memory only** |
| Demo seed org content | scenarios | Seeds truck-loop / `feragame` only (OK as fixture; product path must not require it) |

## Client (SPA / Vitest)

| Gap | Experiment | Attesta today |
| --- | --- | --- |
| **FormPlan + custom fields + derived-leaf** | FormPlan → sjsf → chrome | Ticket **14**; hand IntakeJob/LineJob today |
| **Live chrome** (readonly / valuePrint / recap / …) | live-substep-chrome | Fixture present; SPA **ignores** (part of 14) |
| **JobChrome path switches** (`intake`/`line` bespoke) | Generic form from plan | **Hardcoded** jobId routing + plant field lists |
| **HttpPort link strip in UI** | post-Commit downloads/links | **None** (needs server HttpPort first) |
| **Draft persistence** | debounced localStorage DraftSession | **In-memory only** |
| **TS AuthPort** | Interface + harness actors | **None** — session is Go gate; client only needs OperatorIdentity projection if UI role chrome needs it |
| **Local Commit/Load Vitest adapters** | Full accept path mirror | HTTP mocks; **no** in-process ELT mirror of server |
| **Plant e2e smokes** | container-reg / progress-paint scripts | Generic Playwright; **no** living plant e2e |
| **sjsf / jsonata deps in `app/`** | present | **none** (blocked on 14 / server ELT choice) |

## Docs / ops

| Gap | Notes |
| --- | --- |
| Living ADRs 0016–0025 copied into attesta | **none** (glossary/decisions only) |
| ADR-0025 amend (Cerbos + Domain Load/Commit) | deferred in decisions dump |
| Demo user seed ops | `living@` needs `task seed -- --replace` on pre-12 stacks |

## Suggested slice order (not locked)

1. **16** — server config-driven Commit (ELT + reads + instanceId/label; delete Task `switch`)  
2. **17** — readyWhen → Cerbos attr (queued)  
3. **18** — HttpPort after mutate (queued)  
4. **14** — FormPlan UI (once server seals match config)  
5. **19** — durable Domain/OrgConfig stores (queued)  
6. ADR copy / plant e2e / draft persistence — still inventory-only until spawned  
7. **13** rename when names agreed
