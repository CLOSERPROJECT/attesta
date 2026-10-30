# 03: Task form bundle for first available Task

**What to build:** For at least one **available** Task, Load returns a **Task form bundle**: the minimum payload to mount that form (compiled schema + uiSchema, initial/seed data, leftover ids/columns when needed, Chrome and Derived *rules*, seal/chrome field ids, title, minimal bindings bag). That Task mounts from the bundle — not by compiling from a full authored Task document plus full Domain.

Chrome/Derived stay live on the client using the bindings bag. Living code stays generic.

**Blocked by:** 01: Worklist on Load

**Status:** ready-for-agent

- [ ] Load includes a Task form bundle for ≥1 Task that Worklist marks `available`
- [ ] TaskForm (or equivalent) mounts that Task from the bundle without client FormPlan compile from full authored Task + full Domain
- [ ] Bundle includes schema/uiSchema, initial data, dynamic leftover projection fields when needed, Chrome/Derived rules, seal/chrome ids, title, bindings bag
- [ ] Commit of that Task still works end-to-end in smoke/tests
- [ ] No plant-specific Task branches in bundle builders

**Refs:** ADR-0005; GLOSSARY (**Task form bundle**).
