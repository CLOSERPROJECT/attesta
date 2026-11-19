# Lock the Passport chain mechanism

Type: grilling
Status: open
Blocked by: 04, 05, 09
Map: [map.md](../map.md)

## Question

Given the Feragame → Org B sketch and the granularity decision, what is the Passport chain mechanism we spec for implementation?

Lock:

- The Config shape for declaring an upstream source, and where it lives (Job, Task, or the root `dpp` / `passport` block).
- How a Task binds inputs and what the executor validates at Commit: whether the Passport exists, whether it was issued by the declared org, the subject kind, and consumption limits.
- What the bound inputs expose to expressions, and what Org B's Domain persists.
- What the downstream Passport records about its inputs. UNTP composes Passports by linking upstream Passports (optionally with a digest) rather than copying their data, while we settled that downstream orgs *read* upstream snapshots. Decide whether any upstream facts are copied into the downstream snapshot, or whether they're only read at Commit and then linked.
- How the mechanism stays Config-declared, with no plant- or org-named executor branches.

Decide whether this warrants an ADR (likely: hard to reverse, and a real trade-off), and whether it supersedes any part of ADR-0007.
