# Feragame → Org B chain sketch

Type: prototype
Status: open
Blocked by: 01
Map: [map.md](../map.md)

## Question

How does the chain look concretely when Org B takes, as input for one of its Jobs, containers that Feragame produced across N lots?

The prototype is a rough artifact to react to, not product code. It should cover:

- **Org B Config:** the type-level declaration of the upstream source (Feragame lot Passports), and the Task that binds concrete inputs to an Org B Job instance (scan a Digital Link, or pick from a list?).
- **What Org B's Domain and expressions see:** how the bound upstream snapshots appear (for example a new `$inputs` binding next to `$domain` / `$instance`), and what gets stored in Org B's Domain at Commit.
- **Org B's issued Passport:** the snapshot and the UNTP links back to the Feragame Passports (event type per ticket 01).
- **One failure path:** an input that is already fully consumed, or a Digital Link that doesn't resolve.

Use a real Feragame customer as Org B if one exists; otherwise use a made-up Org B. Use the production Indium stream v2 (`../legacy-streams/6a2ab9bb6f12c95446e6c161.yaml`: closing-the-loop → due2lab → critical → dockweiler → ifw → atlant-3d, the one with an issued DPP) as the multi-hop check: does the sketch still read well over five hops?

Save the sketch under `.scratch/passport-chains/prototypes/` and link it from the answer.
