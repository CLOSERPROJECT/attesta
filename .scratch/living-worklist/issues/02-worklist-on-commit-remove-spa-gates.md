# 02: Worklist on Commit + remove SPA gate plants

**What to build:** Successful living Commit returns an updated **Worklist**. The SPA session replaces Worklist from the Commit response (same as Domain today). Production SPA Unlock / startsWhen modules are removed; gate truth lives in Go + Load/Commit goldens.

**Blocked by:** 01: Worklist on Load

**Status:** ready-for-agent

- [ ] CommitResult includes `worklist` consistent with post-Commit Domain
- [ ] After successful Commit, session updates Worklist without re-deriving gates in TS
- [ ] Production `unlock` / `starts-when` modules (and chrome imports) are gone
- [ ] Gate coverage remains via Go tests and/or Load/Commit golden fixtures
- [ ] Failed Commit still refreshes so chrome matches server (existing re-Load behaviour or equivalent)

**Refs:** ADR-0005; GLOSSARY (**Worklist**, **Commit**).
