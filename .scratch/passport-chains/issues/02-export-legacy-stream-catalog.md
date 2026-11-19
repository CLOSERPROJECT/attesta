# Export the legacy Stream catalog

Type: task (HITL)
Status: resolved
Map: [map.md](../map.md)

## Question

Which legacy Streams actually exist and need a conversion target? The seeded demo Streams are random (20–60 copies of three templates), so the real list lives only in a deployed database, and the agent has no access to it.

Checklist for the human:

1. Pick the source environment (staging or production) whose Streams matter.
2. Export `formata_builder_streams` (the raw `stream` YAML per document plus its `_id`) to `.scratch/passport-chains/legacy-streams/`, one file per Stream named `<_id>.yaml`.
3. For each Stream, record instance counts from `processes` by `workflowKey` and status, and whether any instance has an issued DPP.
4. Note which Streams are seed copies of the three disk templates (`server/config/workflow.yaml`, `stream.yaml`, `secondary.yaml`) and which are hand-authored.

Resolved when the export is in place. The answer records the environment, the Stream count, and a table of Stream → steps → organization slugs → instance and DPP counts.

## Answer

- Environment: production (database `closer_demo`, exported 2026-11-19, Mongo server 7.0.48)
- Streams: 23 documents in `formata_builder_streams`, 23 files in `legacy-streams/`
- Instances: 55 total in `processes` (41 active, 13 done, 1 terminated), 7 with an issued DPP
- Origin (checklist item 4): 0 seed copies of the disk templates, 0 seed copies edited later, 23 hand-authored. Hand-authored Streams: all 23 (see table)
- Process groups with no Stream document: `69bbd2a3077d885f85644b6e` (1 active), `69bbd6cf077d885f85644b72` (1 done), `69ca7f678eba39cf0157d22a` (1 active), `stream` (1 done, 1 DPP), `workflow` (3 active)
- Scrub: no secrets found (17 `opaque-string` hits were schema enums, org/role slugs, and field titles). `yamlSha256` in `index.json` is pre-scrub. `raw-export.json` deleted.

| Stream id | Name | Steps → organization | Instances (by status) | DPP | Origin |
| --- | --- | --- | --- | --- | --- |
| 69b299c8077d885f85644b55 | Gallium shipment audit | 1–3 → org1 | 1 (active 1) | 0 | hand-authored |
| 69b299c8077d885f85644b56 | Indium recycling audit | 1 → closing-the-loop; 2 → due2lab; 3 → critical; 4 → dockweiler; 5 → ifw | 4 (active 4) | 0 | hand-authored |
| 69b299c8077d885f85644b57 | Gallium recycling notarization | 1 → organization-1; 2 → organization-2; 3 → organization-3 | 14 (active 9, done 4, terminated 1) | 5 | hand-authored |
| 69e886a89c9a659739b9f345 | Gallium circularity trace | 1 → organization-1; 2 → organization-2; 3 → critical; 4 → dockweiler; 5 → ifw; 6 → closing-the-loop | 0 | 0 | hand-authored |
| 69ee439a0b481595a079d022 | PV stream | 1–3 → organization-1 | 2 (active 1, done 1) | 0 | hand-authored |
| 69f85193ad5a32b1a615e6fd | Indium stream | 1–6 → organization-1 | 7 (active 5, done 2) | 0 | hand-authored |
| 6a02fa45ad5a32b1a615e73f | Terminalista Arrivo da Nave | 1 → organization-1 | 1 (active 1) | 0 | hand-authored |
| 6a03254cad5a32b1a615e741 | PID/EAA provider onboarding | 1 → pid-eaa-provider; 2 → provider-registration-authority; 3 → pid-eaa-provider; 4 → access-certificate-authority; 5 → pid-eaa-provider; 6 → provider-registration-certificates; 7 → trusted-list-provider | 1 (active 1) | 0 | hand-authored |
| 6a046625ad5a32b1a615e74c | PV stream (Out Of Use) v3 | 1–3 → out-of-use | 3 (active 2, done 1) | 0 | hand-authored |
| 6a29581b6f12c95446e6c15b | TEST CTL | 1–4 → closing-the-loop | 1 (active 1) | 0 | hand-authored |
| 6a2ab9bb6f12c95446e6c161 | Indium stream v2 | 1 → closing-the-loop; 2 → due2lab; 3 → critical; 4 → dockweiler; 5 → ifw; 6 → atlant-3d | 3 (active 2, done 1) | 1 | hand-authored |
| 6a2abda46f12c95446e6c164 | Gallium stream v2 | 1 → closing-the-loop; 2 → due2lab; 3 → critical; 4 → dockweiler; 5 → ifw; 6 → atlant-3d | 0 | 0 | hand-authored |
| 6a2bf7f36f12c95446e6c174 | Indium stream v2 | 1 → closing-the-loop; 2 → due2lab; 3 → critical; 4 → dockweiler; 5 → ifw; 6 → atlant-3d; 7 → critical | 0 | 0 | hand-authored |
| 6a329c87fe7f8941cf040ae5 | PV stream (Out Of Use) v3 test | 1–3 → organization-1 | 3 (active 2, done 1) | 0 | hand-authored |
| 6a422714887c4ed15fd6f630 | PV stream (Out Of Use) v4 | 1–3 → out-of-use | 2 (active 1, done 1) | 0 | hand-authored |
| 6a4ba9d3887c4ed15fd6f6db | Demo LOOPS | 1 → organization-1; 2 → due2lab; 3 → critical; 4 → dockweiler; 5 → ifw; 6 → atlant-3d | 2 (active 2) | 0 | hand-authored |
| 6a6cbab43c9e9e30c4bcc190 | PV stream (Out Of Use) v4 | 1–3 → out-of-use | 0 | 0 | hand-authored |
| 6a6cbba63c9e9e30c4bcc192 | Gallium circularity trace | 1 → organization-1; 2 → organization-2; 3 → critical; 4 → dockweiler; 5 → ifw; 6 → closing-the-loop | 0 | 0 | hand-authored |
| 6a6cbcb73c9e9e30c4bcc194 | Indium recycling audit | 1 → closing-the-loop; 2 → due2lab; 3 → critical; 4 → dockweiler; 5 → ifw | 0 | 0 | hand-authored |
| 6a6cbd573c9e9e30c4bcc195 | Gallium from e-waste, DALP 3D printing  | 1 → organization-1; 2 → due2lab; 3 → critical; 4 → dockweiler; 5 → ifw; 6 → atlant-3d | 0 | 0 | hand-authored |
| 6aad13650e66a0e43cf27c8f | RAAE temp | 1–3 → organization-1 | 0 | 0 | hand-authored |
| 6ab12febae92a46e52014216 | Ecosistem - Trattamento RAEE gate-to-gate v0.2 | 1–6 → ecosistem | 1 (active 1) | 0 | hand-authored |
| 6acd01f61102c6b04c6d018a | Feragame PVP - v0.2 | 1–8 → feragame | 3 (active 3) | 0 | hand-authored |

## Comments

- Handoff guide: [EXPORT-GUIDE.md](../legacy-streams/EXPORT-GUIDE.md)
