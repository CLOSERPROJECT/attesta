# UNTP linking of input Passports to outputs

Type: research
Status: resolved
Map: [map.md](../map.md)

## Question

How does UNTP express "an output product made from inputs that carry their own Passports", and what must an Attesta Org Passport emit to be a valid link in that chain?

Specifically:

- Which Digital Traceability Event types apply to the hand-offs we expect, and where does each one belong in a chain? The expected hand-offs are: consuming material (Transformation), grouping lots into containers or pallets (Aggregation), selling between orgs (Transaction) and assembling (Association).
- How do `inputEPCList` / `inputQuantityList` / `outputEPCList` identify inputs? Specifically: whole Passports versus items inside a batch, and partial consumption, such as taking some containers out of a lot whose Passport covers the whole lot (TEV-05).
- How does the DPP point at those events (`traceabilityInformation`, `TraceabilityPerformance.valueChainProcess`, `verifiedRatio`, `SecureLink` hash fields), and does it also carry `materialsProvenance` derived from the inputs?
- Who issues the event: the downstream org (consumer), the upstream org, or both?
- Which UNTP version is current, and does `ModifyEvent` (used by the legacy Stream DTE today) still exist in it?

Primary sources: the [UNTP specification](https://untp.unece.org/docs/specification/Architecture) (source in [`uncefact/spec-untp`](https://github.com/uncefact/spec-untp), `website/docs/specification/`), the UNTP DTE and DPP vocabularies (jargon.sh), and ISO/IEC 19987 (EPCIS 2.0) where UNTP defers to it.

Write findings to `.scratch/passport-chains/research/untp-input-linking.md`, citing a source for each claim.

## Answer

- **Version.** UNTP moved to GitLab (`opensource.unicc.org`). The latest release is v0.7.0 (May 2026), labelled for pilots. 1.0 is pending, and its draft carries DTEs as **GS1 EPCIS 2.0 events used unmodified** inside a VC envelope.
- **`ModifyEvent`** exists only in 0.7.0, where it is meant for post-market actions. In the 1.0 draft it becomes an `ObjectEvent`.
- **Event types.** Consuming material is a `TransformationEvent`. Grouping into containers or pallets is an `AggregationEvent`. The inter-org hand-off is a pair of shipping and receiving `ObjectEvent`s, not a `TransactionEvent`. Assembly is an `AssociationEvent`.
- **Identifying inputs.** `inputEPCList` takes instance-level identifiers. `inputQuantityList` takes a class-level `epcClass` plus a quantity and unit. Listed inputs may be "fully or partially consumed". To move part of a lot, pack it into an identified container first, then name the container.
- **DPP links.** `traceabilityInformation`, `verifiedRatio` and `SecureLink` are 0.6-only. From 0.7.0 the DPP links its events and input DPPs through `relatedDocument` (link type `dte`/`dpp`, optional `digestMultibase`) and through the resolver. `materialProvenance` is optional and issuer-declared. UNTP composes passports rather than nesting upstream data.
- **Who issues.** Each actor issues the events for what it did. The consumer issues the transformation. Supplier and buyer each sign their own half of the transfer.
- **Feragame → Org B.** Feragame issues a lot → container `TransformationEvent`. Org B issues a `TransformationEvent` with the container URIs in `inputEPCList`, and its DPP links that event.

[findings](../research/untp-input-linking.md)
