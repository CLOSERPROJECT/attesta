# 01: Living Load — empty Domain + fixture Org Config

**What to build:** A verified session can Load the org’s living state: empty Domain plus the Active Config version taken from the flat truck-loop Org Config fixture. Unauthenticated callers are rejected. Uses code seam placeholders (`internal/living`, `GET /api/living/domain`, Memory DomainStore / ConfigStore).

**Blocked by:** None (can start immediately).

**Status:** done

- [x] Verified user receives Domain (empty) + Active Config version for the flat fixture
- [x] Unauthenticated / unverified request cannot Load
- [x] Memory DomainStore and ConfigStore are wired for this path
- [x] Behaviour is covered by automated tests
