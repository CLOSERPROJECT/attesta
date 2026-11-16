# HANDOFF — Slice A+B

**Branch:** `feat/living-home-hard-cut`  
**Commit/push:** none (per brief)

## Changed

### Go
- `server/internal/living/commit.go` — authored `overview` → wire `home`; nested-open-group + remaining-work defaults → `home` (no `overview` emit); `continueNavigation` inherits that default
- `server/internal/living/groups_commit_test.go` — expect `home`
- `server/internal/living/task_form_http_test.go` — expect `home` (drop overview accept)
- `server/internal/living/form_bundle_test.go` — expect `home` | `task`

### SPA
- `app/src/lib/living/worklist-nav.ts` — wire `overview` → `{ kind: 'home' }`; `resolveTaskExit` always home; unmountable Task redirect → home
- `app/src/lib/living/nav-place.ts` — Task `backTarget` typed/narrowed to home
- `app/src/lib/living/http-commit-port.ts` — parse maps wire `overview` → `home` (nav + continue)
- tests: `worklist-nav.test.ts`, `nav-place.test.ts`, `http-commit-port.test.ts`

## Not touched
- LivingShell / home bands (Slice C)
- Overview route (kept)
- Task-group product work

## Tests
- `go test ./internal/living/` — pass
- `bun run test:unit -- worklist-nav / nav-place / http-commit-port` — 41 pass

## Blockers / follow-ups
- None for A+B
- `CommitNextOverview` const retained (unused emit); `PostCommitTarget` still has `overview` for route typing
- Slice C may still deep-link home bands → overview until C lands
