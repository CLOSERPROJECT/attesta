# 12: Close gate Config-only (drop `"lots"` fallback)

**What to build:** Remove plant group-id fallback `group.ID() == "lots"` from Task group Close gating in living Go. Rely solely on authored Config `closeRequiresChildSubmission` (feragame `lots` already sets it `true`). Hard call: engine must not encode Feragame vocabulary.

**Blocked by:** —

**Status:** done

**Audit:** [Audit](9c5fdd0e-e3e8-498e-938a-3f4996773ad7) — high, branch-new (`2f046ac`), `server/internal/living/groups.go` ~164.

- [x] Delete `group.ID() == "lots"` (and any sibling plant-id Close defaults)
- [x] Close behavior for groups with `closeRequiresChildSubmission: true` unchanged; groups without the flag do not get a secret plant default
- [x] Go tests cover Config-true vs absent; no assertion that depends on id `"lots"` as the gate
- [x] Zero `"lots"` string as behavioral vocabulary in production `groups.go` (fixture/test data OK)
