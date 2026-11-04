# 01: Worklist readyWhen → unavailable

**What to build:** Fold Task `readyWhen` into the Go Worklist projector so Unlock true ∧ readyWhen false yields a new wire status `unavailable` (not `available` / not `actionable`). Amend ADR-0005. SPA adds label/tag only; home and mount stay actionable-only. No plant Unlock / YAML change required to land.

**Blocked by:** None (can start immediately).

**Status:** implemented

## Build exactly

1. Add `TaskStatusUnavailable = "unavailable"` in `worklist.go`; extend SPA `TaskStatus` + `taskStatusLabel` (IT tag, e.g. `Non disponibile`) + `TaskStatusTag` styling if needed.
2. After path Unlock says unlocked-not-done, evaluate the leaf Task’s `readyWhen` with the same semantics as Commit (`evaluateReadyWhen` / omit⇒true / fail⇒false). False ⇒ project `unavailable` and **skip** role split.
3. Groups: keep group-node status from Unlock/Close only for this ticket unless a group authors `readyWhen` (none today). Child leaves that author `readyWhen` fold the same way.
4. Amend [`docs/adr/0005-living-worklist-server-projection.md`](../../../docs/adr/0005-living-worklist-server-projection.md): readyWhen is Worklist chrome (not Commit-only); status enum includes `unavailable`; home/mount remain actionable-only; do not overload `available`.
5. Optionally align GLOSSARY Worklist enum line with `unavailable`.
6. Unit tests with **synthetic** Task ids + authored `readyWhen` (true/false/omit/invalid). Assert no plant Task id hardcoding. Commit `ReadyWhenOk` path unchanged (already exists).

## Verify

```bash
cd server && go test ./internal/living/ -count=1 -run 'ReadyWhen|Worklist|Project'
cd app && bun run test:unit -- --run src/lib/living/worklist-overview src/lib/living/types
```

Full living suite still green without plant YAML changes:

```bash
cd server && go test ./internal/living/... -count=1
```

## When done

- [x] Wire + Go `TaskStatus` includes `unavailable`
- [x] Unlock ∧ readyWhen false ⇒ `unavailable`; omit readyWhen ⇒ prior role-split behaviour
- [x] Home Available / mount gates still ignore non-`actionable` (including `unavailable`)
- [x] SPA label/tag paints `unavailable`; no TS readyWhen eval
- [x] ADR-0005 amended; synthetic Go tests cover fold; no `"lineLoad"` special case in Go
- [x] Commit ReadyWhenOk behaviour unchanged
