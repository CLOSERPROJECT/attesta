# 03: Cerbos on living Commit

**What to build:** Product Commit allow/deny goes only through Cerbos (draft resource kind `living`). Hybrid rule: Go may compute domain/unlock attrs; Cerbos alone EFFECT_ALLOW/DENY. Register Commit works when principal Config roles allow and fails when Cerbos denies. Test stub Authorizer removed from the product path.

**Blocked by:** 02: Commit skeleton — register Task (test Authorizer)

**Status:** done

- [x] Living Commit calls Cerbos; no product bypass that allows without Cerbos
- [x] Allow case: permitted Config role can Commit register
- [x] Deny case: disallowed principal cannot Commit; Domain unchanged
- [x] Policy/attrs stay small (no full DomainSnapshot dumped into Cerbos)
