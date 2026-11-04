# Living HTTP — coordinator ledger

**Integration branch:** `feat/attesta-rework`  
**Feature branch:** `feat/http-port-sidecar`  
**Worktree:** `/Users/giovanniabbatepaolo/.cursor/worktrees/feat-http-port-sidecar-9d1c3350/attesta-714729698316`  
**Tracker:** `.scratch/living-http/issues/` (not GitHub)  
**Role:** chief-of-staff DRI — **all implement/review/fix in subagents**; sparse briefs via pointers  
**Pipeline:** ticket → implementer (A→B→C) → code reviewer → fixer → squash into integration (human)  
**Grill:** 2026-11-02 — FakeHttpPort → Compose `living-http` behind HttpPort  

## Active

| Ticket | Stage | Notes |
| --- | --- | --- |
| 01 | **awaiting human** | Review APPROVE; squash → `feat/attesta-rework` + plant smoke; then `/apply-worktree` or merge as you prefer |

## Done

| Ticket | Notes |
| --- | --- |
| 01A | `f35fdbe` — sidecar + Compose |
| 01B | `670c8ed` — SidecarHttpPort + artifact proxy |
| 01C | `f583210` — fpdf labels PDF |
| review | APPROVE (Standards+Spec); nits only; no fixer |

## Frontier

Human: squash `feat/http-port-sidecar` → `feat/attesta-rework`, smoke per ticket. Tranche B still not built.
Optional follow-ups (not blocking): dedupe CSV helpers; require token when URL set; Attesta info logs; HttpPort standards snip; README “until 01B” stale line.

## Coverage (2026-11-02)

- Measure: living-http pkg 73.6% (outside `task cover`); living was 90.8% with weak `handleArtifact` / `HttpPortFromEnv`.
- Gate raise: `1bb11fe` — unit tests only (no `-tags=integration`). Server total **90.5% → 90.8%**; `COVER_MIN=90.0` **PASS** (go-test cover step; full `task cover` css/worktree not re-run).
- Human smoke green. Still awaiting squash → `feat/attesta-rework`.

## Hard calls (settled — do not re-litigate)

See `.scratch/living-http/issues/01-living-http-sidecar-httpport.md` grill table A–N.  
Artifacts = **local disk**. Product drops in-process FakeHttpPort.

## Env / pit-of-success notes

- Match Compose patterns in `deployment/docker-compose.local.yaml` + `scripts/worktree-env.sh` (`PORT_KEYS`).
- Conventional commits; comments = why only (`AGENTS.md`).
- Main checkout may run `task dev` — do not disturb; use this worktree’s compose project.
- After review: short HttpPort standards note (path-only URL, timeout, no body logs) for future reviewers.
- **01C smoke:** ticket smoke section — `LIVING_HTTP_URL` + `LIVING_HTTP_TOKEN` via Compose; feragame register Commit → labels PDF + CSV links.
