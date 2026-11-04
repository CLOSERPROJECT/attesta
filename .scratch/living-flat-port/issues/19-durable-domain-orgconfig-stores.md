# 19: Durable DomainStore + OrgConfigStore (Mongo)

**What to build:** Replace Memory-only living persistence with durable Interfaces backed by shared Mongo wiring (same cluster/DB as legacy). Domain by org; Org Config by org + Config version; Active Config pointer. Preserve Memory Adapters for unit tests.

**Blocked by:** 16 recommended (Commit shape stable); not blocked on 14/13

**Status:** done — reviewer APPROVE; squash into rework; human smoke

- [x] Domain survives process restart
- [x] Org Config versions + Active Config version persist and Load correctly
- [x] Seed/wipe paths work against durable stores (gate remains non-prod footgun)
- [x] Do not extend legacy Process Store / progress documents
- [x] Tests: Memory still used in unit tests; integration or store tests for Mongo Adapter

**Refs:** ticket 15; decisions Persistence; AGENTS.md seed invariant (Go seed, no volume-copy).

**Collections:** `living_domains` (`_id`=org), `living_org_configs` (`_id`=org+NUL+version, JSON payload), `living_active_config` (`_id`=org). Entity-map / fact keys encode `.`→`_` at Mongo boundary (same as Process progress). Product path: `ensureLivingStores` uses Mongo when `store` is `*MongoStore`; Memory otherwise / unit tests.
