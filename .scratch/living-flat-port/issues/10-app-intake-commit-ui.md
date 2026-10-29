# 10: /app — intake Commit UI

**What to build:** Operator walks the flat intake Job in the browser (register through weightCheck, including Leftovers UI for palletWeight) against living Commit + Cerbos. Drafts stay client-side until Commit.

**Blocked by:** 05: Intake — Leftovers completeness all (palletWeight); 09: /app — Load shell + HTTP ports

**Status:** ready-for-agent

- [ ] Browser can complete intake Tasks in Unlock order via living Commit
- [ ] Leftovers UI for palletWeight works against the API
- [ ] Failed Cerbos / Unlock denial surfaces without corrupting Domain
- [ ] No harness impersonation controls
