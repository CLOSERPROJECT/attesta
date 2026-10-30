# 18: Server — HttpPort after successful mutate

**What to build:** After Domain mutate succeeds, run Org Config `onSubmit.http` (side effects / link strip). Failures must not roll back Domain. Commit response (or follow-up shape) exposes links/downloads for the SPA when authored.

**Blocked by:** 16: Server — config-driven Commit (ELT)

**Status:** queued (do later)

- [ ] Fixture Tasks with `http:` produce post-Commit side-effect results without rolling back Domain
- [ ] Failure of HttpPort leaves Domain as committed
- [ ] Client can surface links/downloads (minimal Adapter; full UI polish can wait for 14)
- [ ] Tests cover success + failure-does-not-rollback

**Refs:** ticket 15; glossary HttpPort; experiment `commit-with-http` / `http-port`.
