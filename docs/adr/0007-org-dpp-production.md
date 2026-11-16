# Org DPP production is a platform capability declared in Org Config

The Org product issues Digital Product Passports as a **core platform** feature
(Config **`passport`** block; Go **`DPPStore`** / **`org_dpps`** persistence):
operator **Passport lock** (a normal Job/Task Commit) evaluates an Org Config
projection, freezes a **Passport snapshot** on a platform Passport record (org +
subject), serves UNTP/GS1 Digital Link credentials from that record, and returns
public links on the Commit response. Org Config supplies *what* to gather and
*how* to identify the product; Go/SPA only execute. One plant Config may author
a concrete subject entity, eligibility predicate, and snapshot graph; that YAML
is payload, not the feature (demo shape:
[`docs/feragame/GLOSSARY.md`](../feragame/GLOSSARY.md)).

**Config shape:** Root `passport` block (sibling of `qr` / `domain`) declares
subject binding, identity (GTIN, lot/serial mapping, product copy), and
`snapshot` projection. A Lock leaf opts in with thin `onSubmit.passport` after
`onSubmit.transform` (transform owns Domain locked/issued facts for eligibility;
producer owns Passport record + links). Job instance id **is** the subject id.
v1: one passport shape per Org Config version; no per-Task projection override;
no re-lock/amend.

**Operator path:** Start the Lock Job → select an eligible subject → Load shows
a recap that is the **same** `passport.snapshot` projection evaluated live
(Chrome / `dataReport`) → one Invia freezes that projection and issues. Recap
and freeze must not be two drifting exprs.

**Boundaries:** Post-Commit Passport links are **not** `onSubmit.http` /
HttpPort. Org DPP issuance does **not** write Stream `Process.DPP` or run
`finalizeProcessIfDone`. Public Digital Link / UNTP HTML+JSON serve may share
extracted helpers with legacy Stream DPP; issue paths stay separate. Org DTE
(per-Submission events) is out of v1 unless later embedded in the snapshot.
Plant terminal / filter statuses used in eligibility remain Config predicates—
not passport closure synonyms.

**Considered options:** (1) Auto-issue when a Domain “fully processed”
predicate flips — rejected: production is an explicit operator approve/lock.
(2) Overload a plant terminal status as passport done — rejected: plants may
reuse the same field for operational filtering. (3) Mint a plant-named Domain
`passport` entity as the only store — rejected: production is platform; plant
Domain stays operational facts. (4) Put the full gather graph only on the Lock
Task with no root block — rejected: mirrors `qr` poorly; duplicates scheme
across Tasks. (5) Job-level “produces DPP” with no Commit leaf — rejected:
Commit is the Seam. (6) Build public URLs via `onSubmit.http` — rejected: links
are producer outputs, not outbound HTTP. (7) Drive Org DPP issue from Stream
process completion — rejected: wrong runtime and storage. (8) Separate
hand-authored Lock recap vs snapshot — rejected: operator must approve what
freezes.

**Consequences:** Need Passport store + producer hooked after successful Lock
Commit when `onSubmit.passport` is set; Org Config validation that passport
opt-in requires a root `passport` block; extract shared Digital Link / UNTP
helpers from Stream code without coupling issue triggers. Demo fixture authors
Lock Job + root projection as payload. Glossary: Passport, Passport lock,
Passport snapshot (platform); plant subject/eligibility nouns in the Feragame
glossary.
