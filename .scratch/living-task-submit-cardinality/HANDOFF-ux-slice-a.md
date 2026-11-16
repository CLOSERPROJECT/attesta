# HANDOFF — UX Slice A (LinkStore address + timestamp)

## Done

LinkStore Put/Get (memory + Mongo) now require `Submission.Timestamp` in the key.
Commit Put uses that Commit’s `ts` (RFC3339Nano). `LoadTaskRecap` Gets by
`sub.Timestamp`. No address-only fallback.

## Paths changed

- `server/internal/living/link_store.go`
- `server/internal/living/link_store_mongo.go`
- `server/internal/living/commit.go`
- `server/internal/living/task_recap.go`
- `server/internal/living/link_store_test.go`

## Tests

```text
go test ./internal/living/ -count=1
```

**Pass** (includes `TestCommitManySubmitPreservesLinksPerTimestamp`: two
many-submit Commits → two link docs, each Get by timestamp).

## Wipe (pre-prod)

Address-only link docs are orphaned. Wipe local/stack LinkStore (and Domain if
desired) before relying on recap links:

- `task reset` / `task purge` for this checkout’s volumes, **or**
- drop Mongo collection `living_submission_links`

No migration job.

## Slice C follow-ups

- List-recap: for each Domain Submission under the address, call
  `LinkStore.Get(org, addr, sub.Timestamp)` (API already timestamp-required).
- Single-recap still uses `findSubmission` (first match) + that row’s timestamp;
  list endpoint owns N rows / oldest-first sort.
- Fixed `Committer.Now` in tests still collapses timestamps — use advancing
  clocks when asserting multiple link docs.
