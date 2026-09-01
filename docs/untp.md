# UNTP (UN Transparency Protocol) implementation

Attesta serves a UNTP pilot surface from the existing GS1 Digital Link
identifiers. UNTP is a protocol, not a platform: each supply chain actor
publishes verifiable product data behind identifiers it already controls. The
GS1 Digital Link (`/01/{gtin}/10/{lot}/21/{serial}`) doubles as the UNTP
Identity Resolver URL (ISO/IEC 18975 structured path, no query parameters).

Spec: <https://untp.unece.org/docs/specification/Architecture> (v1.0 work in
progress — this implementation targets the published v0.7.0 model shapes).

## Endpoints

All endpoints live under the public GS1 Digital Link path registered for a
completed stream instance:

| Request | Response |
|---|---|
| `GET /01/{gtin}/10/{lot}/21/{serial}` | HTML DPP landing page (default link — works for any human scanner) |
| `GET /01/{gtin}/10/{lot}/21/{serial}?format=json` or `Accept: application/json` | UNTP **Digital Product Passport** credential (W3C VC 2.0 envelope, `DigitalProductPassport` type) |
| `GET /01/{gtin}/10/{lot}/21/{serial}?linkType=linkset` (or `all`, or `Accept: application/linkset+json`) | Full **Identity Resolver linkset** (RFC 9264, `application/linkset+json`) |
| `GET /01/{gtin}/10/{lot}/21/{serial}?linkType=dpp\|dte\|pip` | Linkset filtered to one relation (GS1 resolver convention, UNTP IDR-07) |
| `GET /01/{gtin}/10/{lot}/21/{serial}/events` | UNTP **Digital Traceability Event** credential — one `ModifyEvent` per completed substep |
| `GET /01/{gtin}/10/{lot}/21/{serial}/attachment/{id}/file` | Public attachment download (unchanged; referenced as event `relatedDocument` links) |

Unknown `linkType` values return 404. Unknown GTIN/lot/serial return 404 at
every layer (resolver workflow: no data for the identifier).

## UNTP components → Attesta mapping

| UNTP component | Attesta implementation |
|---|---|
| **DPP** (Digital Product Passport) | Credential subject is a `Product` identified by the GS1 Digital Link URI at item granularity: `batchNumber` = lot, `itemNumber` = serial, `idScheme` = GS1 Digital Link. Name/description/issuer come from `workflow.yaml` `dpp.productName` / `dpp.productDescription` / `dpp.ownerName` (falling back to workflow name/description). `relatedDocument` links the DTE events endpoint. `characteristics` carries workflow key, process id, and the GS1 element string `(01)…(10)…(21)…`. |
| **DTE** (Digital Traceability Events) | Every completed workflow substep becomes one `ModifyEvent` (checkpoint observation, product identity retained): `eventDate` = substep `DoneAt`, `activityType` = substep id/title under the workflow scheme, `relatedParty` = completing actor (role + party URI), `relatedDocument` = the substep's attachments via their public digital-link download URLs, `disposition` = `active`. Incomplete substeps are not events. |
| **IDR** (Identity Resolver) | The GS1 Digital Link path resolves to an RFC 9264 linkset with `dpp` (credential JSON), `dte` (events JSON), and `pip` (HTML landing page) relations. `linkType` filtering follows the GS1 resolver convention; the bare URL returns the default link (HTML). |
| **VCP** (VC profile) | Credentials use the W3C VC 2.0 envelope (`@context`: W3C credentials/v2 + UNTP vocabulary, `type: [DigitalProductPassport|DigitalTraceabilityEvent, VerifiableCredential]`, `issuer`, `validFrom` = DPP generation time). **Issued unsigned** — see Limitations. |
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
- **DPP claims / material provenance / dimensions / classification** — the
  demo workflow has no data for them; all are optional in the UNTP schema.

## Absolute URLs

Credentials and linksets must carry absolute URIs. They are derived per
request via `requestBaseURL` (`r.Host` + scheme from `X-Forwarded-Proto` /
TLS / `COOKIE_SECURE`). Behind a proxy, set `X-Forwarded-Proto: https`.

## Code

- `server/cmd/server/untp.go` — credential/linkset builders + events handler.
- `server/cmd/server/dpp.go` — `parseDigitalLinkEventsPath` (path parser).
- `server/cmd/server/main.go` — `handleDigitalLinkDPP` dispatch: `linkType`
  param → linkset, `Accept: application/linkset+json` → linkset,
  `Accept: application/json` / `?format=json` → DPP credential, else HTML.
- `server/templates/pages/dpp.html` + `web/src/styles/pages/dpp.css` —
  machine-links section.
