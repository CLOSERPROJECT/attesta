# Affiliation-gated stream authoring (draft/publish later)

Open registration and request-based org join/create leave users unaffiliated until an admin approves. Every Formata save enters the live catalog immediately, so authoring before affiliation—or half-finished publish semantics—would either leak unfinished blueprints or force a large catalog change alongside auth.

**Decision:** No stream authoring until the user is affiliated; keep Formata gates (org admin / platform admin). Draft / unpublished streams come later so affiliated authors can save without publishing to the live catalog.

**Rejected:** Draft/publish in the same change as join/org-creation requests; allowing unaffiliated Formata saves into the live catalog.
