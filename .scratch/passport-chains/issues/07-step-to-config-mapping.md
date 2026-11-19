# Mapping rules from legacy Steps to Org Config Jobs

Type: grilling
Status: open
Blocked by: 02, 06, 10
Map: [map.md](../map.md)

## Question

What are the rules for turning one legacy Stream into per-org Org Configs plus chain links, so that each per-Stream review only has to apply them?

Cases the rules must cover, checked against the real catalog from ticket 02:

- **One org owns several consecutive Steps** (as in `secondary.yaml`): one Job, several Jobs, or Tasks inside one Job?
- **An org appears twice, not consecutively** (A → B → A): is that two hand-offs, with two Passports issued?
- **The same org appears in several Streams:** how its Jobs merge into its single Config, and what happens when two Streams give it conflicting definitions.
- **Substep features:** how roles, `inputType`, schema/uiSchema, attachments, and Stream-level `dpp` settings (GTIN, lot input key, serial strategy) map to Tasks, Config roles and the per-org `dpp` / `passport` block.
- **Steps with no substeps** (step 6 of `stream.yaml`), and step organizations that have no Organization record yet.
- **Where each hand-off's Passport lock sits** in the upstream org's Config.
