# Org Domain change feed invalidates boards by revision

Other operators' `/app` boards learn about Domain writes through a server-sent, stream-scoped feed of **Domain revisions**. Events only say the revision moved; boards re-Load, so the server stays the only projector ([ADR-0005](0005-org-worklist-server-projection.md)). The Domain revision counts writes, exactly +1 each—that lets a client tell its own Commit's echo from foreign writes without actor ids or timing windows, and lets every reconnect catch up by comparing one number. Notifications come from the end of a Commit, after its side records, not from the store. Fanout is in-process: one Attesta process; a second replica would need a shared bus behind the same seam. An open form is never swapped under the operator: it shows a notice and refreshes on accept; dirty drafts survive ([ADR-0009](0009-org-client-draft-persistence.md)).

**Locked:** invalidation-only events; the dense +1-per-write Domain revision; publish at Commit end (not from the store).

**May change without a new ADR:** heartbeat interval, retry/backoff values, UI copy.

**Considered options:** polling the revision (lag and steady load; kept only as a possible fallback behind the same client port); WebSockets (no bidirectional need); Mongo Change Streams (replica-set operations for a single-process app); reusing Formata's `/my` events route (Org stays off it).

**Consequences:** the +1-per-write rule is a store contract every Domain store adapter must keep; deployments must not buffer the change-feed response; running more than one replica breaks live refresh until a bus adapter exists (boards still catch up on reconnect).
