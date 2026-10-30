# 19: Durable DomainStore + OrgConfigStore (Mongo)

**What to build:** Replace Memory-only living persistence with durable Interfaces backed by shared Mongo wiring (same cluster/DB as legacy). Domain by org; Org Config by org + Config version; Active Config pointer. Preserve Memory Adapters for unit tests.

**Blocked by:** 16 recommended (Commit shape stable); not blocked on 14/13

**Status:** queued (do later)

- [ ] Domain survives process restart
- [ ] Org Config versions + Active Config version persist and Load correctly
- [ ] Seed/wipe paths work against durable stores (gate remains non-prod footgun)
- [ ] Do not extend legacy Process Store / progress documents
- [ ] Tests: Memory still used in unit tests; integration or store tests for Mongo Adapter

**Refs:** ticket 15; decisions Persistence; AGENTS.md seed invariant (Go seed, no volume-copy).
