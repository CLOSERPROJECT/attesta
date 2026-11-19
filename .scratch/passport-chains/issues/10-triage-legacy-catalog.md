# Triage the legacy catalog: which Streams get a conversion target

Type: grilling
Status: claimed
Map: [map.md](../map.md)

## Question

Which of the exported production Streams get a conversion target, and for each family, which version is canonical? Everything not kept is dropped, without review.

The [catalog export](02-export-legacy-stream-catalog.md) has 23 Streams; three are byte-identical re-imports, which leaves 20 distinct. Grouped by shape (ids shortened to their last 4 characters; files in `../legacy-streams/`):

- **Multi-org chain over closing-the-loop / due2lab / critical / dockweiler / ifw / atlant-3d:**
  - Indium recycling audit (`4b56`, copy `c194`)
  - Indium stream v2 (`c161`, 1 DPP)
  - Indium stream v2 with a 7th step back at critical (`c174`)
  - Gallium stream v2 (`c164`)
  - Demo LOOPS (`f6db`)
  - Gallium from e-waste, DALP 3D printing (`c195`)
  - Gallium circularity trace (`f345`, copy `c192`). This one mixes in organization-1/2.
- **Demo orgs organization-1/2/3:**
  - Gallium recycling notarization (`4b57`: 14 instances, 5 DPPs, the most used)
  - Gallium shipment audit (`4b55`)
- **Single-org:**
  - PV stream (`d022`)
  - Indium stream (`e6fd`)
  - Terminalista Arrivo da Nave (`e73f`)
  - RAAE temp (`7c8f`)
  - TEST CTL (`c15b`)
  - PV stream (Out Of Use) v3 (`e74c`), v3 test (`0ae5`), v4 (`f630`, copy `c190`)
  - Ecosistem RAEE gate-to-gate v0.2 (`4216`)
  - Feragame PVP v0.2 (`018a`)
- **Not a product flow:** PID/EAA provider onboarding (`e741`), seven steps across trust-framework orgs, with pid-eaa-provider appearing three times.

To settle:

- Drop tests, temporaries and copies, and pick one canonical version per family (Indium v1 vs v2 vs v2+critical; PV Out Of Use v3 vs v4; Gallium circularity vs Gallium stream v2).
- Do Streams built on the demo orgs (organization-1/2/3) convert, or are they superseded by the real-org chains?
- Is Feragame PVP v0.2 already superseded by the Feragame Org Config, or does it carry something the Config lacks?
- Does PID/EAA onboarding belong in Passport chains at all, given there's no product and no Passport? If not, rule it out of scope.
- For single-org Streams, confirm they become a plain Org Config with no chain links.

The answer is the kept list (Stream id → family → reason), which seeds one review ticket per kept Stream.

## Comments

- Paused 2026-11-19 with the triage round asked but not answered. Proposed (agent recommendations, not decisions):
  - **Keep 5:**
    - Indium stream v2 (`c161`: newest, the only member with a DPP)
    - Gallium stream v2 (`c164`: same six orgs as Indium)
    - PV stream (Out Of Use) v4 (`f630`: single-org)
    - Ecosistem RAEE gate-to-gate v0.2 (`4216`: single-org)
    - Feragame PVP v0.2 (`018a`): richer than the current Feragame Config (campaigns, lot↔container genealogy, DPP on output lots). Review it as "what the Feragame Config adopts", ideally before the Feragame → Org B sketch is finalised.
  - **Drop:**
    - Older Indium variants: `4b56`, `c194` (without atlant-3d) and `c174` (June draft with a 7th step at critical)
    - Gallium copies with an organization-1 placeholder: `c195`, `f6db`
    - Gallium circularity trace (`f345`, `c192`) unless it's a planned pilot
    - All demo or placeholder-org Streams: `4b57` (its 5 DPPs keep resolving via the Stream fallback), `4b55`, `e6fd`, `d022`, `0ae5`, `7c8f`, `e73f`, `c15b`
    - PV v3 (`e74c`) and the v4 copy (`c190`)
  - **Out of scope:** PID/EAA provider onboarding (`e741`): a non-product approval flow. Note on the map that some users use Streams as generic multi-party approval workflows.
  - Resume by re-asking these as Q1–Q8 and recording the user's answers.
