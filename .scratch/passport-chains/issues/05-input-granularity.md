# Input granularity: whole Passports or items inside them

Type: grilling
Status: open
Blocked by: 01
Map: [map.md](../map.md)

## Question

When a downstream Job takes inputs, what is the unit it binds?

- **(a)** Whole upstream Passports, for example Feragame lots, optionally with a quantity taken from each.
- **(b)** Items listed inside an upstream Passport snapshot, for example the lot Passport plus a container id. Today Feragame's lot snapshot lists `containers` with ids, but containers have no Passport of their own.
- **(c)** Finer-grained upstream Passports, which would require the upstream org to issue per-item Passports, or an aggregation, for what it hands off.

Things the decision must also cover:

- Whether partial consumption is tracked: can one lot feed two downstream Job instances, and does anything stop over-consumption?
- Whether the choice is fixed by the platform or authored per Config.
- How it maps onto UNTP items versus quantities (TEV-05, per ticket 01).
