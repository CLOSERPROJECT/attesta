# 15: Persist Commit links after HttpPort

**What to build:** After Domain Commit Put and HttpPort, persist returned `links[]` in a **side record** keyed like the Submission (job + Job instance + Task path [+ group occurrence]). Store failure must not roll back Domain. No HTTP replay. Commit response still returns links for the stay UI.

**Blocked by:** None (can start immediately)

**Status:** done (local; not committed)

- [x] Side store (not mutating Submission facts) written after HttpPort when links non-empty
- [x] Keyed by Submission address including nested group occurrence when present
- [x] Store error does not fail Commit / does not roll back Domain
- [x] Readable later by address (test or store API)
- [x] No re-run of `onSubmit.http` to rebuild links
