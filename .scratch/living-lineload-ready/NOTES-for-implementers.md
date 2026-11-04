# Notes for implementers

Read before coding. Product + Config grammar are frozen in [`LOCKED.md`](./LOCKED.md).

## Repo orientation

- [`AGENTS.md`](../../AGENTS.md) — working rules, invariants, where docs live.
- [`GLOSSARY.md`](../../GLOSSARY.md) — **Unlock order**, **startsWhen**, **readyWhen**, **Submission**, **Domain reads**, **Worklist**. Glossary already says readyWhen false ⇒ “visible but unavailable”; ticket 01 makes that a real Worklist status.
- [`docs/adr/0005-living-worklist-server-projection.md`](../../docs/adr/0005-living-worklist-server-projection.md) — amend in **01** (`readyWhen` in Worklist; `unavailable`). Do not revive leftovers/completeness grammar.

## Key Go seams

| Concern | File |
| --- | --- |
| Unlock / path | `server/internal/living/unlock.go` (+ `unlock_*_test.go`) |
| readyWhen eval / Commit attr | `server/internal/living/ready_when.go` |
| Worklist projection / `TaskStatus` | `server/internal/living/worklist.go` |
| Task parse / validate | `server/internal/living/tasks.go`, `config.go` |
| Plant fixture | `server/internal/living/fixtures/feragame-config.yaml` |
| Plant flow / weigh / lineLoad tests | `plant_flow_test.go`, `pallet_weight_test.go`, `line_remaining_test.go`, `seed*.go` |

## SPA (label/tag only in 01)

- `app/src/lib/living/types.ts` — `TaskStatus` union
- `app/src/lib/living/worklist-overview.ts` — `taskStatusLabel`
- `app/src/lib/living/TaskStatusTag.svelte` — paint
- Home / mount already actionable-only (`worklist-home.ts`, `worklist-nav.ts`) — do not treat `unavailable` as mountable

## Pit of success

- Generic engine: synthetic Task ids in unit tests; plant YAML only in **03**.
- Fail closed if `"lineLoad"` (or any plant id) appears in Go conditionals.
- Commit already has `ReadyWhenOk`; reuse readyWhen evaluation for Worklist — one law.
