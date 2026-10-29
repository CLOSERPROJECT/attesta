# 10: /app — intake Commit UI

**What to build:** Operator walks the flat intake Job in the browser (register through weightCheck, including Leftovers UI for palletWeight) against living Commit + Cerbos. Drafts stay client-side until Commit.

**Blocked by:** 05: Intake — Leftovers completeness all (palletWeight); 09: /app — Load shell + HTTP ports

**Status:** done

- [x] Browser can complete intake Tasks in Unlock order via living Commit
- [x] Leftovers UI for palletWeight works against the API
- [x] Failed Cerbos / Unlock denial surfaces without corrupting Domain
- [x] No harness impersonation controls
