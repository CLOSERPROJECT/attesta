# 01: Worklist on Load

**What to build:** Go projects a **Worklist** for the org and returns it on living Load. The operator Jobs UI paints start / available / done / locked from that Worklist only (Unlock order, startsWhen, and Leftovers openness evaluated server-side). Domain and Org Config remain on the wire for this slice.

Wire shape (decision):

```text
worklist: {
  jobs: [
    {
      jobId,
      canStart,
      instances: [
        { id, label, tasks: { [taskId]: "locked" | "available" | "done" } }
      ]
    }
  ]
}
```

Leftover-some exhausted maps to `done`. Living code stays generic; plant Job/Task ids only in Org Config YAML / fixtures.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] Load JSON includes `worklist` with `canStart` and per-instance Task statuses matching Unlock / startsWhen / leftover-open rules
- [ ] JobPanel (or successor chrome) does not call SPA Unlock / startsWhen for painting; it reads Worklist statuses
- [ ] Leftover-open → `available` vs clear → `done` uses config reads / dynamicProperties, not hardcoded plant Task paths
- [ ] Go tests (and/or Load goldens) cover intake Unlock, line startsWhen, and leftover-some openness
- [ ] No plant-specific conditionals in the Worklist projector

**Refs:** ADR-0005; GLOSSARY (**Worklist**, **Load**).
