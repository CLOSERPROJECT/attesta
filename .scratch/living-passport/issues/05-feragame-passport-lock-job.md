# 05: Feragame Passport lock Job

**What to build:** Feragame plant Org Config: root `passport` projection over truck/load/pallets/containers; Passport Lock **Job** with subject = load; eligibility reads Domain (`fullyProcessed` ∧ not yet locked); Lock Task with transform facts + `onSubmit.passport`; Worklist/Unlock/path wiring consistent with living Job grammar. End-to-end smoke: select eligible load → live recap → Invia → links + public serve.

**Blocked by:** 02, 03, 04 (platform store/producer, public serve, live recap).

**Status:** done (pending SHA at commit)

- [x] Root `passport` block in feragame fixture; identity + snapshot projection match ADR Feragame shape
- [x] Lock Job/Task authored in Org Config only; Job instance id = load subject id
- [x] Eligibility uses `fullyProcessed` for register closure only; separate locked/issued Domain facts for passport
- [x] Worklist shows Lock when eligible; locked loads excluded from re-lock (v1 once)
- [x] Plant tests or documented smoke: Commit links + public Digital Link/UNTP for a seeded load
- [x] Active Config refresh documented (`task seed -- --replace`) when YAML changes

## Active Config refresh

Plant YAML lives in `server/internal/living/fixtures/feragame-config.yaml` and is embedded at build time. Updating the **Active Config** in a running stack requires reseeding:

```bash
task seed -- --replace
```

That is a human-gated Domain wipe — agents must not run it.

## Smoke (human)

1. After `task seed -- --replace` (or fresh empty stack + seed), complete a load through container registration (`fullyProcessed`).
2. Worklist → **Blocca passaporto** → select the load → Lock Load shows live `passportSnapshot` recap → **Invia**.
3. Commit response includes Digital Product Passport link; open public Digital Link / UNTP for that GTIN/lot/serial.
4. Same load must not reappear in select enum; second lock fails closed (passport-exists).
