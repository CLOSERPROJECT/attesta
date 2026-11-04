# 04: Task form bundles for all available + provisional

**What to build:** Every Task Worklist marks `available`, plus provisional start when `canStart` (first Task, empty Job instance / mint), has a **Task form bundle** on Load and Commit. Forms no longer mount from Org Config Task documents.

**Blocked by:** 03: Task form bundle for first available Task

**Status:** ready-for-agent

- [x] Load/Commit ship bundles for all `available` Tasks on existing instances
- [x] When `canStart`, a provisional bundle exists for the Job’s first Task with empty instance id (mint path)
- [x] Job chrome mounts forms only from bundles, not from Org Config Task objects
- [x] Truck-loop (or fixture) smoke can walk intake + line Commits using bundles
- [x] Bundle set is replaced on Commit (drop bundles for Tasks that left `available`)

**Refs:** GLOSSARY (**Task form bundle**, **Worklist**).
