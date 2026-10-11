# Demo fixture seed invoke stays host-specific

Coolify Preview and local worktree Compose both gate on `ATTESTA_SEED` and prepare an empty Appwrite + demo fixture before serving (or after local infra is up). Shared orchestration now lives in the Go seam `PrepareDemoStack` / CLI `attesta prepare`: enable-gate → wait for Appwrite `/v1/health` → `BootstrapAppwriteProject` → apply runtime env file in-process → `Seed` (empty-only).

**Decision:** Keep *host invocation policy* in thin adapters. Do not fold Coolify hard-fail vs worktree soft-skip (or `ATTESTA_SEED_COMMAND`) into one shared shell runner. Adapters only choose soft vs hard flags and the worktree escape hatch:

| Host | Adapter | Policy |
|---|---|---|
| Coolify Preview | `deployment/coolify-entrypoint.sh` → `./attesta prepare` | Hard-fail on bootstrap/seed; sources env file once before `exec` so the server inherits keys |
| Worktree Compose | `scripts/worktree-compose.sh` → `go run ./cmd/server prepare --soft-bootstrap --soft-credential-skip` | Soft-skip bootstrap failure; soft-skip seed when credentials missing; `ATTESTA_SEED_COMMAND` override stays shell-only |

**Rejected:** A parameterized shared `run_attesta_seed` shell helper that encodes command, credential skip, and `ATTESTA_SEED_COMMAND` — that mostly relocates host divergence into flags and fails the deletion test for a deep module. Deepen the Go prepare seam instead.
