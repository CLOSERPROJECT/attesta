# UNTP target version and Passport identifiers

Type: grilling
Status: open
Map: [map.md](../map.md)

## Question

Which UNTP version do Passport chains target, and what identifier form must Passports and events use so that a downstream org's `inputEPCList` matches the upstream Passport exactly?

To settle, using the [UNTP research findings](../research/untp-input-linking.md):

- **Version:** target 0.7.0, the latest release (labelled for pilots, with `relatedDocument` links and `ModifyEvent`), or the 1.0 draft (unmodified EPCIS 2.0 events, with `ObjectEvent` instead of `ModifyEvent`)? Should the legacy Stream emitter move with it, or stay as it is until the Stream runtime retires?
- **Identifier form:** the 1.0 draft recommends canonical `https://id.gs1.org/…` with a 14-digit GTIN. Attesta mints Digital Links on its own deployment host. Which form goes into credentials and events, and which into QR codes?
- **Granularity:** Attesta hardcodes `idGranularity: "item"`, but the Feragame Passport covers a whole lot, with the serial set to the lot id. Is a lot Passport an item (an instance-level id including `/21/`) or a batch (a class-level id, usable in `inputQuantityList`)?
