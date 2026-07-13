# Domain naming debt

Running log from the UI refactor vocabulary session (2026-07-13). Canonical terms live in [`CONTEXT.md`](../CONTEXT.md). This file tracks **code/template renames** to align later — not glossary definitions.

## Agreed renames (not yet implemented)

| Domain term | Current code / URL / file aliases | Notes |
|-------------|-----------------------------------|-------|
| **Stream** (blueprint) | `Workflow`, `WorkflowDef`, `workflow.yaml`, `workflowKey`, `/w/:key/`, … | Domain retires **workflow**; code keeps aliases until rename pass. |
| **Stream instance** | `Process`, `/process/:id`, `process.html`, `ProcessPageView`, `handleTerminateProcess`, … | User-facing "stream" often means instance; glossary uses **stream instance** for precision. |
| **Step** (blueprint phase) | `ProcessStep` (runtime progress per substep — different concept) | `ProcessStep` overloads "step"; consider rename to `SubstepProgress` or similar in a later pass. |
| **Substep** (completable unit) | `Action`, `ActionView`, `buildActionList`, `action_list.html`, … | Retire **Action** everywhere. See rename targets below. |
| **Stream picker** | `home.html`, `home_picker_body`, `handleHome` | → `stream_picker.html`, `stream_picker_body` |
| **Stream instance detail** (page + payload) | `process.html`, `process_body`, `process_content.html`, `ProcessPageView`, `ActionListView` | → `stream_instance_detail.html`, `stream_instance_detail_body`, `stream_instance_detail_content`; `StreamInstanceDetailView` |

| **Stream timeline** (accordion tree) | `components/stream_timeline.html`, define `stream_timeline` | Migrated 2026-07-13 |
| **Substep body** (inner panel) | `components/substep_body.html`, define `substep_body` | Migrated 2026-07-13 |
| **Substep body modes** | inferred from `Status`, `ReadOnly`, `Disabled`, `DetailMessage` | explicit: `preview`, `actionable`, `result`, `message` |

### Action → Substep rename targets (future pass)

**Go types & functions**
- `ActionView` → `SubstepBodyView` (carries `Mode`: preview | actionable | result | message)
- `ActionListView` → `StreamInstanceDetailView`
- `buildActionList` → `buildSubstepBodies` (or `buildSubstepViews`)
- `buildProcessActionListView` → `buildStreamInstanceDetailView`
- `ActionRoleBadge`, `ActionRoleOption`, `ActionKV`, `ActionAttachmentView` → `SubstepRoleBadge`, etc.
- `selectedActionBySubstep`, `nextAvailableAuthorizedAction` → drop "action" from names
- `TimelineSubstep.Action` → `Body` (`*SubstepBodyView`)
- Test files: `action_list_*_test.go` → `substep_body_*_test.go`

**Templates & defines**

### Resolved in substep_body migration (2026-07-13)

- `action_list.html` → `components/substep_body.html`, define `substep_body`
- `action_detail_content.html` → removed (folded into `substep_body`)

### Resolved in stream_timeline migration (2026-07-13)

- `workflow_timeline` → `components/stream_timeline.html`, define `stream_timeline`
- `timeline.css` step chrome → `components/stream-timeline.css` (class prefix rename deferred)

### Still open (templates & defines)

- DPP history: converge on `substep_body` (result/message modes) over time; `dpp-history-*` markup is debt

**CSS**

### Resolved in substep_body CSS migration (2026-07-13)

- `actions.css` → `components/substep-body.css`
- `.action-*` classes → `.substep-body-*` (carousel root: `.substep-body-attachments-carousel`)

**Docs**
- AGENTS.md "action cards" → substep summary / substep body
- `docs/css.md` index entries

## Open items

- Full Go rename pass per tables above (`ActionView` → `SubstepBodyView`, …)

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
