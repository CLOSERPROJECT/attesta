# Spec: living Passport production

**Branch:** `feat/living-passport` (from `feat/living-container-reg`; rebase onto integration before merge when container-reg lands)  
**ADR:** [`docs/adr/0007-living-passport-production.md`](../../docs/adr/0007-living-passport-production.md)  
**Status:** ready-for-agent  
**Hard constraint:** Go and SPA are dumb executors of Org Config. No Feragame- or plant Task-id special-case logic in the backend or SPA for Passport gather, lock, or serve.

## Problem

Operators must explicitly **Passport lock** (a normal Job/Task Commit) after reviewing what will freeze. Org Config declares subject binding, identity (GTIN, lot/serial, product copy), and a `passport.snapshot` projection; the platform stores a **Passport snapshot** on a Passport record (org + subject) and serves UNTP/GS1 Digital Link credentials from that record. Post-Commit Passport links are producer outputs—not `onSubmit.http` / HttpPort. Living issuance does not write Stream `Process.DPP` or run `finalizeProcessIfDone`.

## Solution (v1)

- Root `passport` block (sibling of `qr` / `domain`) in Org Config; Lock leaf opts in with thin `onSubmit.passport` after `onSubmit.transform`.
- Job instance id **is** the subject id (Feragame: subject = **load**, eligibility = `fullyProcessed` ∧ not yet locked).
- Lock **Load** recap evaluates the **same** `passport.snapshot` projection live (Chrome / `dataReport`) as the freeze; one Invia freezes that projection and issues.
- Extract shared Digital Link / UNTP helpers from Stream code for public serve; issue paths stay separate from Stream DPP triggers.
- `fullyProcessed` remains container-registration eligibility only—not passport closure.

## Out of scope (v1)

- Re-lock / amend; per-Task projection override; multiple passport shapes per Org Config version.
- Living DTE per-Submission events unless later embedded in snapshot.
- Driving living issue from Stream process completion.

## Glossary (living)

**Passport**, **Passport lock**, **Passport snapshot**, **Org Config**, **Domain**, **Job**, **Task**, **fullyProcessed** — see ADR and repo glossary.

## Testing

- Prefer external behavior: Commit produces Passport record + Commit links; public Digital Link/UNTP routes serve frozen snapshot; Load recap matches freeze input.
- Synthetic Org Config fixtures before Feragame plant YAML; no assertions on internal helper names unless guarding Stream regression.
