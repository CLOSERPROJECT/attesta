# 05: Plant — continuation delta + drop dualSources

**What to build:** Next `register` Spawn prefills still-open incomplete containers as rows with hidden `continuesContainerId` (via ticket 01 initialData). Operator enters **full** scale weight; Config transform books **delta** (full − prior booked), supersedes the prior incomplete, and may set incomplete again on the new mint. Drop `dualSources` checkbox and free-picked secondary load enum; previous-load attribution comes from the continued container when relevant. New rows without `continuesContainerId` book entered weight as today.

**Blocked by:** 01 (initialData prefactor), 03 (fullyProcessed), 04 (incomplete freeze).

**Status:** done

- [x] Load after open incompletes exist seeds continuation rows with hidden `continuesContainerId`; incomplete unchecked by default
- [x] Commit: continuation ⇒ Domain booked weight = full − prior; prior incomplete no longer open; optional new incomplete on new mint
- [x] New rows (no continues id) book entered weight; first Spawn of a Job instance has no continuation rows
- [x] dualSources UX and free secondary select removed; tests cover delta + supersede
- [x] No plant-named Go/TS booking logic—Config transform + generic seed only
