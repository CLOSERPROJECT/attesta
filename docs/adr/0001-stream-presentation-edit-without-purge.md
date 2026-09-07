# Stream presentation edits keep instances

Platform-admin Formata saves used to purge all stream instances whenever a started stream was rewritten. That was meant to protect instance integrity when the *executable* blueprint changes (steps, roles, forms, orgs, DPP config). **Stream presentation fields** (name, description, categorization) are discovery/labeling only and do not affect instance identity, auth, or notarization.

**Decision:** If a platform-admin save’s semantic diff is only presentation fields, skip `DeleteWorkflowData` and keep existing instances. Any other change (or a compare that cannot prove presentation-only) still purges after an explicit save-time confirm (`confirmPurge`). Live UI shows the current blueprint labels on old instances.

**Rejected:** Always purge on any YAML change (too blunt for retag/rename); freeze historical names on instances (not needed for v1).
