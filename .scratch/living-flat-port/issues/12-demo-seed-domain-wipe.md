# 12: Demo seed — Domain wipe + all-roles user

**What to build:** Dev/demo can reset living Domain and reload the Org Config fixture without a harness snapshot-replace API. Optional all-roles (or broad Config role) user so Cerbos demos work after reset.

**Blocked by:** 08: Line — remaining Tasks

**Status:** ready-for-agent

- [ ] Seed/reset clears or replaces org Domain without `/api/runs` harness PUT
- [ ] Active Config version / fixture is available after seed
- [ ] Documented or scripted path for an all-roles (or equivalent) demo user
- [ ] Reset is safe for local/dev (not a production footgun without a gate)
