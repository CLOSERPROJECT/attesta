# Demo fixture seed invoke stays host-specific

Coolify Preview and local worktree Compose both gate on `ATTESTA_SEED` and wait for Appwrite `/v1/health` via shared `deployment/attesta-appwrite-ready.sh`, then run `bootstrap-appwrite` (shared Go seam `BootstrapAppwriteProject`) before the Go demo fixture seed (`attesta seed` / `Seed`). The *invocation* still differs: the Coolify image runs `./attesta bootstrap-appwrite` then `./attesta seed` and hard-fails on bootstrap errors; worktree runs `go run ./cmd/server bootstrap-appwrite` (soft-skips on failure), may soft-skip seed when Appwrite credentials are unset, and may override seed via `ATTESTA_SEED_COMMAND`.

**Decision:** Keep seed *invocation* as host-specific adapters. Do not fold those policies into a shared runner in the ready script (or similar). Shared glue stops at enable-gate + readiness wait; the Go seams remain `BootstrapAppwriteProject` and `Seed(ctx, SeedDeps)`.

**Rejected:** A parameterized shared `run_attesta_seed` that encodes command, credential skip, and `ATTESTA_SEED_COMMAND` — that mostly relocates host divergence into flags and fails the deletion test for a deep module.
