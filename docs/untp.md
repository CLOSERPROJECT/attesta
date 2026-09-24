# UNTP (UN Transparency Protocol) implementation

Attesta serves a UNTP pilot surface from the existing GS1 Digital Link
identifiers. UNTP is a protocol, not a platform: each supply chain actor
publishes verifiable product data behind identifiers it already controls. The
GS1 Digital Link (`/01/{gtin}/10/{lot}/21/{serial}`) doubles as the UNTP
Identity Resolver URL (ISO/IEC 18975 structured path, no query parameters).

Spec: <https://untp.unece.org/docs/specification/Architecture> (v1.0 expected
after public review — this implementation targets the published v0.8.0 schema
and context artifacts: `untp-dpp` v0.8.0 for the passport, `untp-dte` v0.8.0
— which is GS1 EPCIS 2.0 — for the events, and the UNTP linkset schema for the
resolver. The DPP, DTE, and linkset payloads validate clean against those
three schemas.)

## Endpoints

All endpoints live under the public GS1 Digital Link path registered for a
completed stream instance:

| Request | Response |
|---|---|
| `GET /01/{gtin}/10/{lot}/21/{serial}` | HTML DPP landing page (default link — works for any human scanner) |
| `GET /01/{gtin}/10/{lot}/21/{serial}?format=json` or `Accept: application/json` | UNTP **Digital Product Passport** credential (W3C VC 2.0 envelope, `DigitalProductPassport` type) |
| `GET /01/{gtin}/10/{lot}/21/{serial}?linkType=linkset` (or `all`, or `Accept: application/linkset+json`) | Full **Identity Resolver linkset** (RFC 9264, `application/linkset+json`) |
| `GET /01/{gtin}/10/{lot}/21/{serial}?linkType=dpp\|dte\|pip` | Linkset filtered to one relation (GS1 resolver convention, UNTP IDR-07) |
| `GET /01/{gtin}/10/{lot}/21/{serial}/events` | UNTP **Digital Traceability Event** credential — one GS1 EPCIS 2.0 `ObjectEvent` per completed substep, each carrying that substep's input values |
| `GET /01/{gtin}/10/{lot}/21/{serial}/attachment/{id}/file` | Public attachment download (unchanged; referenced from event `bizTransactionList` entries) |
| `GET /vocabulary/untp/` | Attesta EPCIS extension namespace: JSON-LD context (`Accept: application/ld+json` or `?format=json`), HTML term reference otherwise. Term IRIs such as `/vocabulary/untp/substep` resolve to the same documents; unknown terms 404 |

Unknown `linkType` values return 404. Unknown GTIN/lot/serial return 404 at
every layer (resolver workflow: no data for the identifier).

## UNTP components → Attesta mapping

| UNTP component | Attesta implementation |
| **DPP** (Digital Product Passport) | Credential subject is a `Product` identified by the GS1 Digital Link URI at item granularity: `batchNumber` = lot, `itemNumber` = serial, `idScheme` = GS1 Digital Link. Name/description/issuer come from `workflow.yaml` `dpp.productName` / `dpp.productDescription` / `dpp.ownerName` (falling back to workflow name/description). `relatedDocument` links the DTE events endpoint. `characteristics` carries workflow key, process id, and the GS1 element string `(01)…(10)…(21)…`. `productCategory`, `producedAtFacility` and `countryOfProduction` are required by the UNTP schema — see the next row for where they come from. |
| **Required subject fields** | The UNTP `Product` requires `productCategory`, `producedAtFacility` and `countryOfProduction`. All three are **optional in `workflow.yaml`**: what the schema constrains is the credential, not the config, and failing config load over presentation data would take down the whole stream catalog, not just the passport. When `dpp.productCategory` / `dpp.producedAtFacility` / `dpp.countryOfProduction` are set they win (CPC scheme URIs default to UN CPC). Otherwise the builders derive: `productCategory` from the stream's taxonomy sub-category under the Attesta scheme `{origin}/vocabulary/untp/categories` (`unclassified` when the stream is uncategorized) — not a fake CPC code; `producedAtFacility` from the organization owning the first step (`{origin}/organization/{slug}`), falling back to the deployment origin; `countryOfProduction` from `DPP_COUNTRY_OF_PRODUCTION` (`"NL"` or `"NL,Netherlands"`), falling back to ISO 3166-1 user-assigned `ZZ` / "Unspecified". A country is never guessed — a passport may state "unspecified", never a wrong origin. |
| **DTE** (Digital Traceability Events) | UNTP 0.8.0 replaced the native event model with **GS1 EPCIS 2.0**, used as published. Every completed workflow substep becomes one EPCIS `ObjectEvent`: `action: OBSERVE`, `bizStep: inspecting` (CBV — the product is observed, identity retained), `eventTime` = substep `DoneAt` with `eventTimeZoneOffset: +00:00` (stored UTC), `epcList` = the product's digital-link URI, `readPoint` / `bizLocation` = the DPP facility, `bizTransactionList` = the substep's attachments as public download URLs (namespaced transaction type, since no CBV type covers a generic evidence upload). Incomplete substeps are not events. **Operator input values** ride in the `attesta:substep` EPCIS user extension — see below. |
| **Substep payloads** (`attesta:substep`) | EPCIS has no slot for arbitrary captured form data, so each event carries an extension object in a namespace this deployment owns (`{origin}/vocabulary/untp/`, declared inline in the DTE `@context`, which UNTP 0.8.0 permits for EPCIS extensions). Fields: `substepId`, `title`, `stepId`, `stepTitle`, `organization`, `role`, `untpRole`, `completedBy`, `input` (the entered values with their JSON types intact, attachment metadata excluded), and `digest` (`sha256:…` over the stored payload, matching the substep notarization and the HTML "Integrity" section). `input` is typed `@json`: a nested payload of undeclared keys would otherwise expand to an empty node and the values would be lost by any JSON-LD processor. The namespace resolves — see the vocabulary endpoint above. |
| **IDR** (Identity Resolver) | The GS1 Digital Link path resolves to an RFC 9264 linkset with `dpp` (credential JSON), `dte` (events JSON), and `pip` (HTML landing page) relations. `linkType` filtering follows the GS1 resolver convention; the bare URL returns the default link (HTML). |
| **VCP** (VC profile) | Credentials use the W3C VC 2.0 envelope (`type: [DigitalProductPassport\|DigitalTraceabilityEvent, VerifiableCredential]`, `issuer`, `validFrom` = DPP generation time). DPP `@context`: W3C credentials/v2 + `https://vocabulary.uncefact.org/untp/0.8.0/context/`. DTE `@context`: those two plus the GS1 EPCIS context `https://ref.gs1.org/standards/epcis/2.0.0/epcis-context.jsonld` plus the inline `attesta` namespace. **Issued unsigned** — see Limitations. |
| **Rendering** ("any maturity") | The HTML DPP page is the human rendering; a "UNTP & machine links" section lists the linkset, credential, and events URLs. |

## Deferred UNTP components (pilot scope)

- **VC proofs** — credentials are not signed (no W3C VC `proof` block). Payload
  integrity is instead anchored by the existing per-substep SHA-256
  notarizations and merkle root (HTML page "Integrity" section, notarized
  export). Signing (DIA + key management) is the main gap before production.
- **DIA** (Digital Identity Anchor) — issuers are identified by deployment
  origin URL + `dpp.ownerName`, not by a DID linked to a register.
- **DCC** (Conformity Credential) — no third-party assessment data exists in
  the demo; no `dcc` linkset relation is emitted.
- **DFR** (Digital Facility Record) — no facility registry in the demo.
- **DAC** (Decentralised Access Control) — all DPP/DTE data is public-by-link,
  matching the existing public attachment behavior. No encrypted targets.
- **DPP claims / material provenance / dimensions / packaging / labels** —
  the demo workflow has no data for them; all are optional in the UNTP schema.
  `productCategory` is emitted (required by the schema) from
  `dpp.productCategory`.

## Absolute URLs

Credentials and linksets must carry absolute URIs. They are derived per
request via `requestBaseURL` (`r.Host` + scheme from `X-Forwarded-Proto` /
TLS / `COOKIE_SECURE`). Behind a proxy, set `X-Forwarded-Proto: https`.

## Code

- `server/cmd/server/untp.go` — credential/linkset builders + events handler.
- `server/cmd/server/untp_vocabulary.go` — the `attesta:` extension term table
  (single source for the inline DTE `@context`, the served JSON-LD context,
  and the HTML reference) + `/vocabulary/untp/` handler.
- `server/cmd/server/dpp.go` — `parseDigitalLinkEventsPath` (path parser).
- `server/cmd/server/main.go` — `handleDigitalLinkDPP` dispatch: `linkType`
  param → linkset, `Accept: application/linkset+json` → linkset,
  `Accept: application/json` / `?format=json` → DPP credential, else HTML.
- Config: `dpp` subject fields (see mapping table) are validated in
  `normalizeDPPConfig` (`main.go`); the optional `untpRole` key on each entry
  in `workflow.yaml` `roles:` maps a workflow role slug to the UNTP
  `PartyRole` vocabulary and is emitted as `attesta:substep.untpRole` —
  roles without a mapping keep `role` (the workflow slug) and omit
  `untpRole`, since the vocabulary only accepts its own values.
- `server/templates/pages/dpp.html` + `web/src/styles/pages/dpp.css` —
  machine-links section.
- `server/templates/pages/untp_vocabulary.html` +
  `web/src/styles/pages/untp-vocabulary.css` — vocabulary reference page.
