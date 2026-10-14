# Research: Stream instance Substep payload addressing

**Ticket:** [#269](https://github.com/CLOSERPROJECT/attesta/issues/269)  
**Map:** [#267](https://github.com/CLOSERPROJECT/attesta/issues/267)  
**Scope:** Facts from this codebase and persistence only. No Derived-field config API is proposed here.

Domain terms follow `CONTEXT.md`: **Stream** (blueprint), **Stream instance** (runtime; code `Process`), **Substep** (completable unit; code `WorkflowSub` / progress entry).

---

## 1. Where completed payloads live

A Stream instance is a Mongo `processes` document (`Process` in `server/cmd/server/main.go`).

| Field | Role |
|---|---|
| `progress` | `map[string]ProcessStep` — one entry per Substep |
| `ProcessStep.data` | Completed Formata form object (`map[string]interface{}`) |
| `ProcessStep.state` | `"pending"` / `"done"` (completion sets `"done"`) |
| `ProcessStep.description` | Copy of blueprint `inputKey` at completion time |
| `ProcessStep.doneAt` / `doneBy` | Completion metadata |

Completion writes progress then a notarization copy (`server/cmd/server/process_completion.go`):

- `UpdateProcessProgress(..., substepID, ProcessStep{Data: payload, ...})`
- `InsertNotarization` into collection `notarizations` with `substepId` (dotted domain id, e.g. `"1.1"`) and the same `payload` map

Runtime readers that care about form values use **`process.Progress[substepID].Data`**, not the notarization collection (`substep_views_builder.go`, `dpp.go`, `timeline_builder.go`).

---

## 2. Progress keys: dots → underscores (write) / normalize (read)

Substep ids in the Stream blueprint use dotted ids (`"1.1"`, `"2.3"` — see `server/config/stream.yaml`).

**Write / persistence** (`encodeProgressKey` in `server/cmd/server/main.go`):

```go
func encodeProgressKey(key string) string {
	return strings.ReplaceAll(key, ".", "_")
}
```

Mongo `$set` path (`MongoStore.UpdateProcessProgress` in `server/cmd/server/store.go`):

```text
progress.<encodedSubstepID>   // e.g. progress.1_1
```

The same encoding applies to `substepOverrides.<encodedSubstepID>` (`SaveSubstepOverride`). Memory store also stores under the encoded key (`MemoryStore.UpdateProcessProgress`).

**Read / in-memory** (`normalizeProgressKeys`):

```go
decoded := strings.ReplaceAll(key, "_", ".")
```

Call sites (non-exhaustive): `ProcessService.reloadProcess`, process page loads in `main.go`, attention/upcoming/public list helpers, UNTP. After normalization, lookups use dotted ids: `process.Progress["1.1"]`.

Tests: `TestProgressKeyEncodingAndNormalization` (`helpers_order_test.go`), `TestMemoryStoreUpdateProcessProgressEncodesKey` (`store_test.go`), `TestMongoStoreUpdateProcessProgress` (`store_mongo_crud_test.go` expects `"progress.1_1"`).

**Invariant (also in `AGENTS.md`):** progress map keys containing `.` are encoded for Mongo (`.` → `_`) and normalized on reads (`_` → `.`). Field names **inside** `ProcessStep.data` are not rewritten by this boundary.

Caveat of the current helpers: encoding/decoding is a global `.` ↔ `_` replace on the progress map key, not a structured escape. Keys that already contain `_` round-trip as if those underscores were dots (`"a_b_c"` → `"a.b.c"` in tests).

---

## 3. Formata `inputKey` boundary — what it is and is not

Blueprint field (`WorkflowSub.InputKey`, YAML `inputKey`):

- Stored on the Substep definition.
- On completion, copied into `ProcessStep.Description` (`process_completion.go`).
- Shown as UI description / label material (`substep_views_builder.go`).
- In shipped Stream YAML (`server/config/stream.yaml`), often a **long human-readable prose string**, not a short JSON root key.

**Live completion does not nest the form under `inputKey`.**

Pipeline:

1. Client (`web/src/main.js` `submitFormataPayload`) JSON-stringifies the Formata form model into hidden input `name="value"` (`server/templates/components/substep_body.html`).
2. `parseFormataScalarPayload` reads `value` (or falls back to other POST fields excluding `value`) (`main.go`).
3. `normalizePayload` unmarshals JSON to a **top-level object** and returns it unchanged — `InputKey` is unused (`main.go`).
4. Attachment data-URLs inside that object are replaced with attachment meta maps (`persistFormataAttachments`).
5. That object becomes `ProcessStep.Data` and the notarization `payload`.

So for a completed Formata Substep, addressing starts at **schema property names** as keys of `Data`, not at `inputKey`.

**Contrast — demo seed only:** `synthesizeSeedStepData` (`seed_instances.go`) for formata/object often wraps under `InputKey`:

```text
Data[inputKey] = { "seed": true, "note": "..." }
```

unless the Substep supplies the DPP lot key. Seed shape is not the live completion shape.

**Legacy DPP lookup:** when `Description` is nil, `legacyDPPDataLookupKeys` (`dpp.go`) also tries slugified title / `inputKey` as top-level `Data` keys. Current completions set `Description`, so lot/serial resolution uses `dpp.lotInputKey` / `serialInputKey` as top-level keys scanned across done Substeps (`dppFirstStringValue`).

`processStepDataValue` returns the entire `progress.Data` map (no `inputKey` peel). Display flattening walks that object (`flattenDisplayValues` / `normalizeFormataPathSegment` — strips a trailing `[]` from segment names; nests with `.` and arrays with `[i]`).

---

## 4. Nesting inside one Substep payload

Within `ProcessStep.Data`:

- Structure mirrors the Formata/JSON Schema form model (objects, arrays, primitives).
- Nested object keys keep their schema property names (including spaces in some Stream YAML schemas).
- File fields become attachment meta objects (`attachmentId`, `filename`, `contentType`, `size`, `sha256`).

There is **no** second addressing layer that keys all Stream-instance form fields into one flat map. Cross-Substep access always goes through the progress entry for that Substep id first.

---

## 5. Natural path model for expressions (addressing only)

Given the storage model above — not a Derived-field language choice — paths that match existing access patterns:

### Same-Substep siblings (intra-form)

Relative to the **current form data object** (in-progress: Formata model; completed: `Progress[<thisSubstepId>].Data`):

```text
<field>
<parent>.<child>
<path>[<index>].<field>
```

Example from `server/config/stream.yaml` Substep `1.1` schema nesting:

```text
Group title.Batch IDs
```

Same-Substep sibling reads stay inside one `Data` object; they do not use another Substep’s progress key.

### Prior Substeps (cross-Substep)

Address the other Substep’s progress entry, then a path inside its `Data`:

```text
<substepId> → Progress[<substepId>].Data → <field path as above>
```

In memory / domain ids use **dots** (`1.1`). Persistence uses **underscores** only on the progress map key (`1_1`). Expressions that mirror runtime code should use dotted Substep ids; any persistence-layer path must apply `encodeProgressKey` at the progress-key segment only.

Illustrative (not an API):

```text
1.1 / Group title.Batch IDs
2.1 / shipmentRef
```

Existing cross-Substep scan used by DPP: ordered Substeps, skip non-done, look up a **top-level** `Data` key (`dppFirstStringValue`) — first non-empty string wins. That is a special-case global key search, not a general nested path language.

### What expressions should not assume

- That `inputKey` is the root of `Data` on live completions.
- That all Substeps share one flat field namespace without Substep id.
- That progress-key encoding applies inside form field names.
- That notarization documents are the primary read path for form values.

---

## 6. Persistence sketch

```text
processes/{_id}
  progress:
    "1_1":                    # encoded Substep id
      state: "done"
      description: "<inputKey string from blueprint>"
      doneAt: ...
      doneBy: { ... }
      data:                   # Formata JSON object (schema props at root)
        "Group title":
          "Batch IDs": "..."
        ...
  substepOverrides:
    "1_1": { ... }            # same key encoding

notarizations/{_id}
  processId: ...
  substepId: "1.1"            # dotted domain id (not encoded)
  payload: { ... }            # same map as progress data
```

After load + `normalizeProgressKeys`, in-memory:

```text
process.Progress["1.1"].Data["Group title"].(map)["Batch IDs"]
```

---

## Sources

| Claim | Source |
|---|---|
| `Process` / `ProcessStep` / `WorkflowSub` shapes | `server/cmd/server/main.go` |
| Completion writes Data + notarization | `server/cmd/server/process_completion.go` |
| encode / normalize progress keys | `server/cmd/server/main.go` (`encodeProgressKey`, `normalizeProgressKeys`); `AGENTS.md` invariant |
| Mongo `$set progress.<encoded>` | `server/cmd/server/store.go` (`MongoStore.UpdateProcessProgress`) |
| Form POST `value` → unwrapped object | `parseFormataScalarPayload`, `normalizePayload` in `main.go`; `web/src/main.js`; `substep_body.html` |
| UI reads `Progress[substepID].Data` | `server/cmd/server/substep_views_builder.go` |
| DPP cross-Substep top-level key scan | `server/cmd/server/dpp.go` (`dppFirstStringValue`) |
| Display path flattening | `flattenDisplayValues` / `normalizeFormataPathSegment` in `main.go` |
| Seed nests under `inputKey` | `server/cmd/server/seed_instances.go` (`synthesizeSeedStepData`) |
| Example Substep ids / schemas / `lotInputKey` | `server/config/stream.yaml` |
