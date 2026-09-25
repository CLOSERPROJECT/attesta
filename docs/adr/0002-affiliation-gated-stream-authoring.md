# Affiliation-gated stream authoring (draft/publish later)

Open registration and request-based org join/create leave users unaffiliated until an admin approves. Today every Formata save enters the live catalog immediately, so letting anyone author before affiliation (or introducing half-finished publish semantics) would either leak unfinished blueprints or force a large catalog change in the first auth PR.

**Decision:** Ship auth/affiliation first with **no stream authoring until the user is affiliated**, and keep current Formata gates (org admin / platform admin). A follow-up PR introduces **draft / unpublished** streams so affiliated authors can save without publishing to the live catalog.

**Rejected for this PR:** Draft/publish in the same change as join/org-creation requests (too much surface); allowing unaffiliated Formata saves into the live catalog.
