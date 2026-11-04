# 05: Load/Commit without full Domain + Org Config

**What to build:** Living Load and Commit stop shipping full **Domain** and full **Org Config** to the SPA. The operator UI runs on **Worklist** + **Task form bundle**s plus org / Active Config version metadata. Full Domain and Org Config stay server-side; live Chrome/Derived use each bundle’s bindings bag only.

**Blocked by:** 02: Worklist on Commit + remove SPA gate plants; 04: Task form bundles for all available + provisional

**Status:** ready-for-agent

- [ ] Load/Commit response types used by the SPA no longer require full `domain` + `config` for Jobs chrome or Task forms
- [ ] SPA does not keep a full Domain/Org Config snapshot for gate or FormPlan compile
- [ ] Chrome/Derived still update live from draft + bindings bag
- [ ] Commit request remains taskPath + jobInstanceId + sealed value + mint
- [ ] Truck-loop smoke still completes on `/app`

**Refs:** ADR-0005; GLOSSARY (**Load**, **Worklist**, **Task form bundle**).
