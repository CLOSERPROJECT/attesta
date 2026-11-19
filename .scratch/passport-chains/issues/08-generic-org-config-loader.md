# Generic Org Config loader for converted Configs

Type: grilling
Status: open
Blocked by: 07
Map: [map.md](../map.md)

## Question

How do converted per-org Config YAML files get into the system, given that today the only loader is the Feragame fixture parser (`ParseFeragameConfigFixture`, `server/internal/org/fixture.go`)?

Decide:

- Where per-org Configs live in the repo.
- How seed discovers them, and how they become each org's Active Config version.
- Whether the `steps` / `substeps` → Jobs / Tasks rename in the fixture parser stays or the YAML adopts Job / Task names.
- How Organizations referenced by a chain get created when seed doesn't already create them.
- How a chain's cross-org references are validated at load time: what happens when the upstream org or its Passport subject kind doesn't exist.
