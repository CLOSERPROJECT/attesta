# 18: Server — HttpPort after successful mutate

**What to build:** After Domain mutate succeeds, run Org Config `onSubmit.http` (side effects / link strip). Failures must not roll back Domain. Commit response (or follow-up shape) exposes links/downloads for the SPA when authored.

**Blocked by:** 16: Server — config-driven Commit (ELT)

**Status:** done — reviewer APPROVE; squash into rework; human smoke

- [x] Fixture Tasks with `http:` produce post-Commit side-effect results without rolling back Domain
- [x] Failure of HttpPort leaves Domain as committed
- [x] Client can surface links/downloads (minimal Adapter; full UI polish can wait for 14)
- [x] Tests cover success + failure-does-not-rollback
- [x] Product `livingHandler` plants FakeHttpPort so register Commit returns `links` (fixture `/fake/*`)

**Refs:** ticket 15; glossary HttpPort; experiment `commit-with-http` / `http-port`.
