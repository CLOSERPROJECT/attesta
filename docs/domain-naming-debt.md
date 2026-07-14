# Domain naming debt

Running log from the UI refactor vocabulary session (2026-07-13). Canonical terms live in [`CONTEXT.md`](../CONTEXT.md). This file tracks **code/template renames** to align later — not glossary definitions.

## Agreed renames (not yet implemented)

| Domain term | Current code / URL / file aliases | Notes |
|-------------|-----------------------------------|-------|
| **Stream** (blueprint) | `Workflow`, `WorkflowDef`, `workflow.yaml`, `workflowKey`, `/w/:key/`, … | Domain retires **workflow**; code keeps aliases until rename pass. |
| **Stream instance** | `Process`, `/process/:id`, `process.html`, `ProcessPageView`, `handleTerminateProcess`, … | User-facing "stream" often means instance; glossary uses **stream instance** for precision. |
| **Step** (blueprint phase) | `ProcessStep` (runtime progress per substep — different concept) | `ProcessStep` overloads "step"; consider rename to `SubstepProgress` or similar in a later pass. |
| **Substep** (completable unit) | `Action` (remaining in routes/Cerbos only) | Retire **Action** everywhere else. See rename targets below. |
| **Stream picker** | `home.html`, `home_picker_body`, `handleHome` | → `stream_picker.html`, `stream_picker_body` |
| **Stream instance detail** (page + payload) | `process.html`, `process_body`, `process_content.html`, `ProcessPageView` | → `stream_instance_detail.html`, `stream_instance_detail_body`, `stream_instance_detail_content` |

| **Stream timeline** (accordion tree) | `components/stream_timeline.html`, define `stream_timeline` | Migrated 2026-07-13; substep chrome extracted to `substep_shell` 2026-07-14 |
| **Substep shell** (accordion chrome) | `components/substep_shell.html`, define `substep_shell` | Extracted 2026-07-14 from former `stream_timeline_substep` inner define |
| **Substep body** (inner panel) | `components/substep_body.html`, define `substep_body` | Migrated 2026-07-13 |

### Action → Substep rename targets (future pass)

**Go types & functions** — done in go-domain-vocabulary pass (2026-07-13); see resolved section below.

~~- `ActionView` → `SubstepBodyView` (carries `Mode`: preview | actionable | result | message)~~
~~- `ActionListView` → `StreamInstanceDetailView`~~
~~- `buildActionList` → `buildSubstepBodies` (or `buildSubstepViews`)~~
~~- `buildProcessActionListView` → `buildStreamInstanceDetailView`~~
~~- `ActionRoleBadge`, `ActionRoleOption`, `ActionKV`, `ActionAttachmentView` → `SubstepRoleBadge`, etc.~~
~~- `selectedActionBySubstep`, `nextAvailableAuthorizedAction` → drop "action" from names~~
~~- `TimelineSubstep.Action` → `Body` (`*SubstepBodyView`)~~
~~- Test files: `action_list_*_test.go` → `substep_body_*_test.go`~~

**Templates & defines**

### Resolved in substep_body migration (2026-07-13)

- `action_list.html` → `components/substep_body.html`, define `substep_body`
- `action_detail_content.html` → removed (folded into `substep_body`)

### Resolved in stream_timeline migration (2026-07-13)

- `workflow_timeline` → `components/stream_timeline.html`, define `stream_timeline`
- `timeline.css` step chrome → `components/stream-timeline.css` (class prefix rename deferred)

### Resolved in substep_shell extraction (2026-07-14)

- `stream_timeline_substep` inner define → `components/substep_shell.html`, define `substep_shell`
- `stream_timeline.html` iterates substeps via `(streamTimelineSubstep . $.HideStatus)`; shell dispatches to `substep_body`
- Accordion shell CSS in `components/substep-shell.css` (was split from `timeline.css` in stream-timeline CSS alignment)

### Resolved in go-domain-vocabulary pass (2026-07-13)

- `ActionView` → `SubstepBodyView` (+ satellite types in `components.go`)
- `ActionListView` → `StreamInstanceDetailView`; `ProcessPageView.ActionList` → `Detail`
- `TimelineSubstep.Action` → `Body`
- Builders: `buildSubstepViews` (`substep_views_builder.go`), `buildStreamInstanceDetailView` (`stream_instance_detail.go`), timeline assembly (`timeline_builder.go`)
- Test files: `substep_views_builder_test.go`, `stream_instance_detail_test.go`

### Resolved in DPP / stream_timeline convergence (2026-07-14)

- DPP history substep content → `substep_body` (result/message modes) via `stream_timeline_step`
- Step summaries shared via `stream_step_summary` / `StepSummaryView`
- Digest copy UI in `substep_body` → `.substep-body-digest-*` (was `.dpp-integrity-hash*` in shared partial)

**CSS**

### Resolved in substep_body CSS migration (2026-07-13)

- `actions.css` → `components/substep-body.css`
- `.action-*` classes → `.substep-body-*` (carousel root: `.substep-body-attachments-carousel`)

### Resolved in stream-timeline CSS alignment (2026-07-14)

- `.timeline-*` step chrome → `.stream-timeline-*`
- `timeline.css` → `components/substep-shell.css`
- Substep title: `.substep-title-heading` (no longer reuses step title class)
- Substep meta spans: `.substep-meta-time`, `.substep-meta-actor` (template aligned in `substep_shell.html`, 2026-07-14)
- Removed dead `.data-hash`, `.hash-value`, `.substep-summary-supporting`
- `.process-page` moved to `pages/process.css`

**Docs**
- AGENTS.md "action cards" → substep summary / substep body
- `docs/css.md` index entries

### Resolved in substep body mode pass (2026-07-14)

- `SubstepBodyView.Mode` explicit field (`preview` | `actionable` | `result` | `message`)
- `resolveSubstepBodyMode` centralizes inference; builders set `Mode`; `substep_body.html` dispatches on `Mode`

### Resolved in DPP history step wrapper (2026-07-14)

- DPP history vertical rail wrapper → `components/dpp_history_step.html`, define `dpp_history_step` (`.dpp-history-item`, `.dpp-history-rail`, `.dpp-history-dot`)

## Open items

- Full workflow/process/page renames (`Process` → stream instance, routes, `process.html` filename)

## Resolved in session

- Blueprint: **Stream** (was workflow in code)
- Stream instance detail: `stream_instance_detail` page + `stream_instance_detail_content` HTMX/SSE partial
- Top-level screens: **stream picker** (`/`), **stream dashboard** (`/w/:key/`)
- Component migration order: **`substep_body` first**, then `stream_timeline`
- Mid-level blueprint unit: **Step**
- Completable unit: **Substep** (retire Action)
- Stream instance detail page payload: **StreamInstanceDetailView**
- Accordion tree component: **stream_timeline** (not traceability_timeline)
- Inner expandable content: **substep_body** with modes preview / actionable / result + message edge case
- Preview and locked share the **preview** mode (different reason strings)
- Substep body modes: explicit `SubstepBodyView.Mode` in Go (`substep_views_builder.go`)
