# Demo fixture seed invoke stays host-specific

Coolify Preview and local worktree Compose both gate on `ATTESTA_SEED` and prepare an empty Appwrite + demo fixture before serving (or after local infra is up). Shared orchestration lives in the Go seam `PrepareDemoStack` / CLI `attesta prepare`: enable-gate → wait for Appwrite `/v1/health` → `BootstrapAppwriteProject` → `Seed` (empty-only) with Appwrite open params passed by value. Bootstrap may still write the on-disk runtime env file; that file is only the Server restart adapter (main applies it after Coolify prepare), not an in-process prepare→seed channel.

**Decision:** Keep *host invocation policy* in thin adapters via `--policy`. Adapters do not reimplement wait/bootstrap/seed in shell.

| Host | Adapter | Policy |
|---|---|---|
| Coolify Preview | `deployment/coolify-entrypoint.sh` → `./attesta prepare` then `exec` | Default `--policy=coolify`: hard-fail on bootstrap/seed; server loads `/tmp/attesta-appwrite.env` at startup (overrides stale Coolify-injected keys) |
| Worktree Compose | `scripts/worktree-compose.sh` → `go run ./cmd/server prepare --policy=worktree` | Soft-skip bootstrap failure; soft-skip seed when credentials missing |

Manual/operator re-seed uses `attesta seed` / `task seed` (optionally `--replace`). Partial seed CLIs (`seed-categories`, `seed-instances`, `seed-catalog-streams`, `bootstrap-appwrite`) are not public entrypoints; their Go helpers remain for `Seed` / `PrepareDemoStack`.

**Rejected:** A shared shell runner or `ATTESTA_SEED_COMMAND` escape hatch — both duplicated gate/wait logic outside Go and failed the deletion test for a deep prepare module. Separate soft flags (`--soft-bootstrap`, `--soft-credential-skip`) were collapsed into `--policy`.
