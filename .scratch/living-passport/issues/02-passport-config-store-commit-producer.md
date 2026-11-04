# 02: Passport — Org Config, store, Commit producer

**What to build:** Platform Passport capability: parse/validate root `passport` block in Org Config; persist Passport records (org + subject) with frozen **Passport snapshot**; after successful Lock Commit when `onSubmit.passport` is set, run a Commit producer that evaluates snapshot + identity from Config, writes the record once (no re-lock v1), and attaches public Passport links on the Commit response. Links are **not** `onSubmit.http` / HttpPort. Include minimal synthetic Org Config + Commit tests before plant YAML.

**Blocked by:** 01 (shared Digital Link / UNTP helpers available for link shaping).

**Status:** ready-for-agent

- [ ] Root `passport` block validated; Lock opt-in requires it; `onSubmit.passport` is a thin hook after transform (transform owns Domain locked/issued facts for eligibility)
- [ ] Passport store API: get by org+subject; create-once semantics for v1 lock
- [ ] Commit producer runs only on successful Commit with `onSubmit.passport`; snapshot from authored `passport.snapshot` projection
- [ ] Commit `Links` include Passport public URLs; no HttpPort side effects for issuance
- [ ] Synthetic fixture proves end-to-end producer + store without Feragame Task ids
- [ ] Go/SPA remain dumb executors—no plant-named branches
