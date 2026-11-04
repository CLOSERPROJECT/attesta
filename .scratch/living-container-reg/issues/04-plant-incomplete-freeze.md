# 04: Plant — incomplete freeze on register

**What to build:** On `register`, logistic can mark a container row **incomplete** to freeze booked weight for the current load while the physical bin stays open. Domain `container` carries `incomplete` (and booked `weight` for this load). Incomplete defaults unchecked on the form. Cross-load continuation and dropping dualSources are **out of scope** (ticket 05)—dualSources may still exist.

**Blocked by:** 02 (containerReg group + register leaf).

**Status:** done

- [x] Form exposes incomplete (default false) on material rows; Commit persists `incomplete` on minted containers
- [x] Booked weight for this load is the entered weight (no delta math yet)
- [x] Tests cover incomplete true/false Domain facts after Commit
- [x] No continuation seed; no plant-named Go
