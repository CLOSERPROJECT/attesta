# Go domain vocabulary rename — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename Go view types, builders, and template fields from legacy **Action** / **ActionList** vocabulary to **SubstepBody** / **StreamInstanceDetail** per `CONTEXT.md` and `docs/domain-naming-debt.md` — zero behavior change.

**Architecture:** Mechanical rename on branch `refactor/go-domain-vocabulary`, PR target `refactor/domain-vocabulary` (squash merge). Move shared component view DTOs into `components.go` (same pattern as `PageHeaderView`). Update production templates that reference renamed struct fields. Do **not** rename routes, `process.html` file names, `Process`/`Workflow*` types, Cerbos `cerbosActionView`, or HTML form `action=` attributes.

**Tech Stack:** Go 1.25 (`server/cmd/server`), `html/template`, existing test harness (`parseTestTemplates`, `go test ./cmd/server/`).

**Decisions locked in (2026-07-13):**

| Legacy | Target |
|--------|--------|
| `ActionView` | `SubstepBodyView` |
| `ActionRoleBadge` | `SubstepRoleBadge` |
| `ActionRoleOption` | `SubstepRoleOption` |
| `ActionKV` | `SubstepKV` |
| `ActionAttachmentView` | `SubstepAttachmentView` |
| `ActionListView` | `StreamInstanceDetailView` |
| `ActionListView.Action` | `StreamInstanceDetailView.SelectedBody` |
| `ProcessPageView.ActionList` | `ProcessPageView.Detail` |
| `TimelineSubstep.Action` | `TimelineSubstep.Body` |
| `buildActionList` | `buildSubstepViews` |
| `buildProcessActionListView` | `buildStreamInstanceDetailView` |
| `makeActionListReadOnly` | `makeStreamInstanceDetailReadOnly` |
| `selectedActionBySubstep` | `selectedSubstepBody` |
| `nextAvailableAuthorizedAction` | `nextAuthorizedSubstepBody` |
| `decorateTimelineActions` | `decorateTimelineSubstepBodies` |
| `applyDoneByEmailToActions` | `applyDoneByEmailToSubstepViews` |
| `buildActionAttachments` | `buildSubstepAttachments` |
| `findAction` (test helper) | `findSubstepView` |

**Explicitly NOT renamed:** `cerbosActionView`, `authorizeUserAction`, `WorkflowOption.EditAction`, `WorkflowOption.DeleteAction`, `TerminateAction` (URL string), `resolveSelectedSubstepID` (already domain-clean). No `SubstepBodyView.Mode` field in this pass.

**Branch / PR:** Work on `refactor/go-domain-vocabulary` → squash PR into `refactor/domain-vocabulary`.

---

## File map

| File | Responsibility |
|------|----------------|
| `server/cmd/server/components.go` | **Modify.** Add `SubstepBodyView`, satellite types, `TimelineStep`, `TimelineSubstep`, `StreamInstanceDetailView` |
| `server/cmd/server/main.go` | **Modify.** Remove old type defs; rename builders/handlers; `ProcessPageView.Detail` |
| `server/cmd/server/dpp.go` | **Modify.** `buildSubstepAttachments`, `SubstepKV` |
| `server/templates/pages/process.html` | **Modify.** `.ActionList` → `.Detail` |
| `server/templates/components/stream_timeline.html` | **Modify.** `.Substep.Action` → `.Substep.Body` |
| `server/cmd/server/action_list_builder_test.go` | **Rename** → `substep_views_builder_test.go` |
| `server/cmd/server/action_list_render_roles_test.go` | **Rename** → `stream_instance_detail_test.go` |
| ~18 other `*_test.go` files | **Modify.** Type/function/test name updates |
| `docs/domain-naming-debt.md` | **Modify.** Mark Go rename resolved |
| `AGENTS.md` | **Modify.** Replace `ActionView`/`ActionListView` references |

**Out of scope:** `process.html` file rename, `/process/:id` routes, `ProcessPageView` type rename, CSS class renames, DPP `dpp-history-*` markup, explicit `Mode` enum on `SubstepBodyView`.

**References before starting:**
- `CONTEXT.md` (glossary)
- `docs/domain-naming-debt.md` (rename table)
- `.agents/skills/attesta-ui-components/SKILL.md` (`components.go` conventions)
- Prior migration: `docs/superpowers/plans/2026-07-13-stream-timeline-migration.md`

---

### Task 1: Baseline verification

**Files:**
- Test: `server/cmd/server/` (full package)

- [ ] **Step 1: Confirm branch**

Run: `git branch --show-current`  
Expected: `refactor/go-domain-vocabulary`

- [ ] **Step 2: Run full server tests**

Run: `cd server && go test ./cmd/server/ -count=1`  
Expected: PASS (all tests green before any rename)

- [ ] **Step 3: Record baseline grep counts** (sanity check for completion)

Run: `rg -c 'ActionView|ActionListView|buildActionList|buildProcessActionListView' server/cmd/server | wc -l`  
Expected: non-zero (rename surface exists)

---

### Task 2: Satellite types in `components.go`

**Files:**
- Modify: `server/cmd/server/components.go`
- Modify: `server/cmd/server/main.go` (remove old defs, fix references)
- Modify: `server/cmd/server/dpp.go`
- Modify: `server/cmd/server/dpp_helpers_test.go`
- Modify: `server/cmd/server/helpers_completed_values_test.go`

- [ ] **Step 1: Extend `components.go` with satellite types**

Replace the file contents with:

```go
package main

// PageHeaderView is the view model for templates/components/page_header.html.
type PageHeaderView struct {
	BackHref    string
	BackLabel   string
	Title       string
	Subtitle    string
	Description string
	Meta        string
}

// SubstepRoleBadge is a role pill on a substep body (preview/result modes).
type SubstepRoleBadge struct {
	ID      string
	Label   string
	Palette string
}

// SubstepRoleOption is a selectable role in termination / multi-role forms.
type SubstepRoleOption struct {
	Slug  string
	Label string
}

// SubstepKV is a flattened submitted value row on a completed substep body.
type SubstepKV struct {
	Key   string
	Value string
}

// SubstepAttachmentView is a file attachment on a substep body.
type SubstepAttachmentView struct {
	AttachmentID string
	Key          string
	Filename     string
	URL          string
	PreviewURL   string
	PreviewKind  string
	SHA256       string
}
```

- [ ] **Step 2: Delete legacy satellite types from `main.go`**

Remove these type blocks from `server/cmd/server/main.go` (lines ~283–307):

- `type ActionRoleBadge struct { ... }`
- `type ActionRoleOption struct { ... }`
- `type ActionKV struct { ... }`
- `type ActionAttachmentView struct { ... }`

- [ ] **Step 3: Global replace satellite type names**

Run from repo root:

```bash
cd server/cmd/server && \
  perl -pi -e 's/ActionRoleBadge/SubstepRoleBadge/g; s/ActionRoleOption/SubstepRoleOption/g; s/ActionKV/SubstepKV/g; s/ActionAttachmentView/SubstepAttachmentView/g' \
  *.go
```

- [ ] **Step 4: Rename attachment builder function**

In `server/cmd/server/main.go`, rename:

```go
func buildSubstepAttachments(workflowKey string, process *Process, data map[string]interface{}) []SubstepAttachmentView {
```

(was `buildActionAttachments`)

Update call sites in `main.go` and `dpp.go` (`buildSubstepAttachments`).

In `server/cmd/server/helpers_completed_values_test.go`, rename:

- `TestBuildActionAttachmentsAndDownloadViews` → `TestBuildSubstepAttachmentsAndDownloadViews`
- `TestActionAttachmentPreviewKindAndURL` → `TestSubstepAttachmentPreviewKindAndURL`
- `buildActionAttachments` → `buildSubstepAttachments` in test bodies

- [ ] **Step 5: Run tests**

Run: `cd server && go test ./cmd/server/ -count=1`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add server/cmd/server/components.go server/cmd/server/main.go server/cmd/server/dpp.go server/cmd/server/*_test.go
git commit -m "refactor(go): rename Action satellite types to Substep*"
```

---

### Task 3: `SubstepBodyView` type rename

**Files:**
- Modify: `server/cmd/server/components.go`
- Modify: `server/cmd/server/main.go`
- Modify: all `*_test.go` referencing `ActionView`

- [ ] **Step 1: Add `SubstepBodyView` to `components.go`**

Append after `SubstepAttachmentView`:

```go
// SubstepBodyView is the view model for templates/components/substep_body.html.
type SubstepBodyView struct {
	WorkflowKey    string
	ProcessID      string
	SubstepID      string
	Title          string
	Description    string
	Role           string
	RoleBadges     []SubstepRoleBadge
	MatchingRoles  []SubstepRoleOption
	RoleLabel      string
	Palette        string
	InputKey       string
	InputType      string
	FormSchema     string
	FormUISchema   string
	Status         string
	DoneAt         string
	DoneBy         string
	DoneRole       string
	Values         []SubstepKV
	Attachments    []SubstepAttachmentView
	Disabled       bool
	ReadOnly       bool
	Reason         string
	DetailMessage  string
	CanAdaptForm   bool
	AdaptURL       string
	FormataArchURL string
	OverrideReason string
	HasOverride    bool
}
```

- [ ] **Step 2: Delete `ActionView` from `main.go`**

Remove `type ActionView struct { ... }` block (~lines 251–281).

Update `TimelineSubstep` temporarily — field still named `Action` but type becomes `*SubstepBodyView`:

```go
type TimelineSubstep struct {
	// ...
	Action *SubstepBodyView
	// ...
}
```

- [ ] **Step 3: Global replace `ActionView` → `SubstepBodyView`**

Run:

```bash
cd server/cmd/server && perl -pi -e 's/\bActionView\b/SubstepBodyView/g' *.go
```

**Verify** `cerbosActionView` was NOT changed:

Run: `rg 'cerbosSubstepBodyView' server/`  
Expected: no matches

- [ ] **Step 4: Rename override test**

In `server/cmd/server/substep_override_test.go`:

- `TestCompletedActionViewExposesLocalAdaptationReason` → `TestCompletedSubstepBodyViewExposesLocalAdaptationReason`

- [ ] **Step 5: Run tests**

Run: `cd server && go test ./cmd/server/ -count=1`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git commit -am "refactor(go): rename ActionView to SubstepBodyView"
```

---

### Task 4: Substep view builders and helpers

**Files:**
- Modify: `server/cmd/server/main.go`
- Modify: `server/cmd/server/home_handler_test.go`
- Modify: `server/cmd/server/process_action_selection_test.go`
- Modify: `server/cmd/server/helpers_doneby_email_test.go`
- Rename: `server/cmd/server/action_list_builder_test.go` → `substep_views_builder_test.go`

- [ ] **Step 1: Rename builder functions in `main.go`**

| Old signature | New signature |
|---------------|---------------|
| `func buildActionList(...) []SubstepBodyView` | `func buildSubstepViews(...) []SubstepBodyView` |
| `func selectedActionBySubstep(actions []SubstepBodyView, ...) (SubstepBodyView, bool)` | `func selectedSubstepBody(actions []SubstepBodyView, ...) (SubstepBodyView, bool)` |
| `func nextAvailableAuthorizedAction(...) (SubstepBodyView, bool)` | `func nextAuthorizedSubstepBody(...) (SubstepBodyView, bool)` |
| `func (s *Server) applyDoneByEmailToActions(...) []SubstepBodyView` | `func (s *Server) applyDoneByEmailToSubstepViews(...) []SubstepBodyView` |
| `func decorateTimelineActions(timeline []TimelineStep, actions []SubstepBodyView)` | `func decorateTimelineSubstepBodies(timeline []TimelineStep, views []SubstepBodyView)` |

Update all call sites in `main.go`:

```go
actions := buildSubstepViews(cfg.Workflow, process, workflowKey, actor, onlyRole, roleMeta, cfg.Roles)
// ...
actions = s.applyDoneByEmailToSubstepViews(ctx, cfg.Workflow, actor, actions)
timeline = decorateTimelineSubstepBodies(timeline, actions)
// ...
if action, ok := nextAuthorizedSubstepBody(...); ok {
// ...
if action, ok := selectedSubstepBody(actions, selected, processDone); ok {
```

Also update `home_handler_test.go` (`nextAuthorizedSubstepBody`).

- [ ] **Step 2: Rename test file and helper**

```bash
git mv server/cmd/server/action_list_builder_test.go server/cmd/server/substep_views_builder_test.go
```

In `substep_views_builder_test.go`:

- `buildActionList` → `buildSubstepViews` (all occurrences)
- `findAction` → `findSubstepView`:

```go
func findSubstepView(t *testing.T, views []SubstepBodyView, substepID string) SubstepBodyView {
	t.Helper()
	for _, view := range views {
		if view.SubstepID == substepID {
			return view
		}
	}
	t.Fatalf("substep view %q not found in %#v", substepID, views)
	return SubstepBodyView{}
}
```

- `TestBuildActionList*` → `TestBuildSubstepViews*` (all 10 test names in file)

In `process_action_selection_test.go`:

- `selectedActionBySubstep` → `selectedSubstepBody`
- `decorateTimelineActions` → `decorateTimelineSubstepBodies`
- Update failure messages that say `selectedActionBySubstep`

In `helpers_doneby_email_test.go`:

- `applyDoneByEmailToActions` → `applyDoneByEmailToSubstepViews`

- [ ] **Step 3: Run tests**

Run: `cd server && go test ./cmd/server/ -count=1`  
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add -A server/cmd/server/
git commit -m "refactor(go): rename substep view builders and helpers"
```

---

### Task 5: `StreamInstanceDetailView` and `ProcessPageView.Detail`

**Files:**
- Modify: `server/cmd/server/components.go`
- Modify: `server/cmd/server/main.go`
- Rename: `server/cmd/server/action_list_render_roles_test.go` → `stream_instance_detail_test.go`
- Modify: `server/cmd/server/process_template_test.go`
- Modify: `server/cmd/server/stream_timeline_test.go`
- Modify: `server/cmd/server/home_template_preview_test.go`
- Modify: `server/cmd/server/complete_handler_test.go`
- Modify: `server/cmd/server/test_helpers_test.go`
- Modify: `server/cmd/server/action_error_test.go`

- [ ] **Step 1: Add types to `components.go`**

Append:

```go
// TimelineSubstep is one row in the stream timeline accordion (summary + optional body).
type TimelineSubstep struct {
	SubstepID    string
	Title        string
	Description  string
	Selected     bool
	Body         *SubstepBodyView
	Palette      string
	Status       string
	StatusLabel  string
	DoneBy       string
	DoneRole     string
	DoneAt       string
	DisplayValue string
	FileName     string
	FileSHA256   string
	FileURL      string
}

// TimelineStep groups substeps under a blueprint step in the stream timeline.
type TimelineStep struct {
	StepID     string
	Title      string
	OrgSlug    string
	OrgName    string
	OrgLogoURL string
	Expanded   bool
	Substeps   []TimelineSubstep
}

// StreamInstanceDetailView is the HTMX/SSE partial payload for stream instance detail content.
type StreamInstanceDetailView struct {
	WorkflowKey       string
	WorkflowPath      string
	ProcessID         string
	CurrentUser       Actor
	SelectedSubstepID string
	ProcessDone       bool
	SelectedBody      *SubstepBodyView
	Error             string
	Timeline          []TimelineStep
	HideStatus        bool
	DPPURL            string
	DPPGS1            string
	Attachments       []ProcessDownloadAttachment
	CanTerminate      bool
	TerminateAction   string
	TerminateSubstep  string
	TerminateRoles    []SubstepRoleOption
	Termination       *ProcessTerminationView
}
```

- [ ] **Step 2: Remove old types from `main.go`**

Delete:

- `type TimelineSubstep struct { ... }`
- `type TimelineStep struct { ... }`
- `type ActionListView struct { ... }`

- [ ] **Step 3: Rename builder methods**

```go
func (s *Server) buildStreamInstanceDetailView(ctx context.Context, cfg RuntimeConfig, workflowKey string, process *Process, actor Actor, selectedSubstepID, message string, onlyRole bool) StreamInstanceDetailView {
	// ...
	view := StreamInstanceDetailView{ ... }
	// ...
	if body, ok := selectedSubstepBody(actions, selected, processDone); ok {
		view.SelectedBody = &body
	}
	// ...
	if view.SelectedBody != nil {
		body := *view.SelectedBody
		view.SelectedBody = &body
	}
	return view
}

func makeStreamInstanceDetailReadOnly(view StreamInstanceDetailView, reason string) StreamInstanceDetailView {
	// ...
	if view.SelectedBody != nil {
		body := *view.SelectedBody
		body.ReadOnly = true
		body.Reason = reason
		view.SelectedBody = &body
	}
	for stepIndex := range view.Timeline {
		for substepIndex := range view.Timeline[stepIndex].Substeps {
			body := view.Timeline[stepIndex].Substeps[substepIndex].Body
			if body == nil {
				continue
			}
			bodyCopy := *body
			bodyCopy.ReadOnly = true
			bodyCopy.Reason = reason
			view.Timeline[stepIndex].Substeps[substepIndex].Body = &bodyCopy
		}
	}
	return view
}
```

Update `buildProcessPageView`:

```go
detail := s.buildStreamInstanceDetailView(ctx, cfg, workflowKey, process, actor, selectedSubstepID, message, onlyRole)
// ...
return ProcessPageView{
	// ...
	Detail:      detail,
	DPPURL:      detail.DPPURL,
	DPPGS1:      detail.DPPGS1,
	Attachments: detail.Attachments,
}
```

Update `ProcessPageView`:

```go
type ProcessPageView struct {
	PageBase
	Header       PageHeaderView
	ProcessID    string
	InstanceName string
	Status       string
	StatusLabel  string
	Detail       StreamInstanceDetailView
	DPPURL       string
	DPPGS1       string
	Attachments  []ProcessDownloadAttachment
}
```

Update `StreamDashboardView.Preview` field type to `StreamInstanceDetailView` (field name stays `Preview`).

Update `decorateTimelineSubstepBodies` to set `.Body`:

```go
timeline[stepIndex].Substeps[substepIndex].Body = &bodyCopy
```

- [ ] **Step 4: Global replace remaining identifiers**

Run:

```bash
cd server/cmd/server && perl -pi -e '
  s/\bActionListView\b/StreamInstanceDetailView/g;
  s/\bbuildProcessActionListView\b/buildStreamInstanceDetailView/g;
  s/\bmakeActionListReadOnly\b/makeStreamInstanceDetailReadOnly/g;
' *.go
```

Fix `ProcessPageView` field: `ActionList` → `Detail` manually (not a global replace — would hit unrelated symbols).

In test stub `test_helpers_test.go`, update `process_content.html` define:

```go
{{define "process_content.html"}}PROCESS_CONTENT {{.ProcessID}} {{.DPPURL}} {{.Detail.Error}}{{with .Detail.SelectedBody}}{{.SubstepID}}{{end}}{{end}}
```

In `complete_handler_test.go`:

```go
`{{define "process_content.html"}}SEL {{.Detail.SelectedSubstepID}}{{end}}`
`{{define "process_content.html"}}DONE {{.Detail.ProcessDone}} {{.DPPURL}}{{end}}`
```

- [ ] **Step 5: Rename render test file**

```bash
git mv server/cmd/server/action_list_render_roles_test.go server/cmd/server/stream_instance_detail_test.go
```

In `stream_instance_detail_test.go`:

- `TestBuildProcessActionListViewDoesNotFilterByCurrentActiveRole` → `TestBuildStreamInstanceDetailViewDoesNotFilterByCurrentActiveRole`
- `buildProcessActionListView` → `buildStreamInstanceDetailView`
- `view.Action` → `view.SelectedBody`

In `stream_timeline_test.go`:

- `testStreamTimelineView() StreamInstanceDetailView`
- `Action: &SubstepBodyView{` → `Body: &SubstepBodyView{`
- `view.Timeline[0].Substeps[0].Action` → `view.Timeline[0].Substeps[0].Body`

In `process_template_test.go`:

- `ActionList: StreamInstanceDetailView{` → `Detail: StreamInstanceDetailView{`
- `Action: &SubstepBodyView{` → `SelectedBody: &SubstepBodyView{` (top-level)
- Timeline substep: `Action:` → `Body:`

In `home_template_preview_test.go`:

- `Preview: StreamInstanceDetailView{`
- `Action: &SubstepBodyView{` → `Body: &SubstepBodyView{`

In `action_error_test.go`:

- `TestRenderActionViewsReturn500WhenConfigFails` → `TestRenderSubstepBodyViewsReturn500WhenConfigFails` (if test body references old names)

- [ ] **Step 6: Run tests**

Run: `cd server && go test ./cmd/server/ -count=1`  
Expected: FAIL until templates updated in Task 6 — if templates still use `.ActionList`, only template execution tests fail. Proceed to Task 6 if compile passes.

- [ ] **Step 7: Commit**

```bash
git add -A server/cmd/server/
git commit -m "refactor(go): rename ActionListView to StreamInstanceDetailView"
```

---

### Task 6: Production templates

**Files:**
- Modify: `server/templates/pages/process.html`
- Modify: `server/templates/components/stream_timeline.html`

- [ ] **Step 1: Update `process.html`**

Replace every `.ActionList` with `.Detail` (13 occurrences). Example:

```html
data-selected-substep="{{ .Detail.SelectedSubstepID }}"
```

```html
{{ template "error_banner.html" .Detail }}
```

```html
{{ if .Detail.ProcessDone }}
  {{ template "stream_timeline" .Detail }}
```

```html
{{ if .Detail.CanTerminate }}
```

```html
{{ .Detail.TerminateSubstep }}
```

```html
action="{{ .Detail.TerminateAction }}"
```

```html
{{ if .Detail.TerminateRoles }}
```

(Keep `action=` attribute name — HTML form attribute, not Go field rename.)

- [ ] **Step 2: Update `stream_timeline.html`**

Line ~90, change:

```html
{{ with .Substep.Body }}
  {{ template "substep_body" . }}
{{ end }}
```

(was `.Substep.Action`)

- [ ] **Step 3: Run tests**

Run: `cd server && go test ./cmd/server/ -count=1`  
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add server/templates/pages/process.html server/templates/components/stream_timeline.html
git commit -m "refactor(ui): align templates with StreamInstanceDetailView fields"
```

---

### Task 7: Documentation and final verification

**Files:**
- Modify: `docs/domain-naming-debt.md`
- Modify: `AGENTS.md`

- [ ] **Step 1: Update `docs/domain-naming-debt.md`**

Under "## Agreed renames", move Go items to a new "## Resolved in go-domain-vocabulary pass (2026-07-13)" section:

```markdown
## Resolved in go-domain-vocabulary pass (2026-07-13)

- `ActionView` → `SubstepBodyView` (+ satellite types in `components.go`)
- `ActionListView` → `StreamInstanceDetailView`; `ProcessPageView.ActionList` → `Detail`
- `TimelineSubstep.Action` → `Body`
- Builders: `buildSubstepViews`, `buildStreamInstanceDetailView`, etc.
- Test files: `substep_views_builder_test.go`, `stream_instance_detail_test.go`

## Open items

- Full workflow/process/page renames (`Process` → stream instance, routes, `process.html` filename)
- DPP history: converge `dpp-history-*` onto `substep_body` partials
- CSS class prefix: `.timeline-*` → `.stream-timeline-*` (deferred)
- `SubstepBodyView.Mode` explicit field
```

Remove or strike through the old "Action → Substep rename targets" Go table rows that are now done.

- [ ] **Step 2: Update `AGENTS.md`**

Replace references to `ActionView` / `ActionListView` / `action_list` with `SubstepBodyView` / `StreamInstanceDetailView` / `substep_body` where they describe current code.

- [ ] **Step 3: Zero-legacy grep check**

Run:

```bash
rg '\bActionView\b|\bActionListView\b|buildActionList|buildProcessActionListView|selectedActionBySubstep|nextAvailableAuthorizedAction|decorateTimelineActions|applyDoneByEmailToActions|makeActionListReadOnly|buildActionAttachments' server/
```

Expected: no matches (except `docs/superpowers/plans/` historical plans if not updated — optional)

Run:

```bash
rg '\.ActionList\b' server/templates/
```

Expected: no matches

- [ ] **Step 4: Full test suite**

Run: `cd server && go test ./cmd/server/ -count=1`  
Expected: PASS

Run: `task css:lint`  
Expected: PASS (no CSS changes in this pass)

- [ ] **Step 5: Commit**

```bash
git add docs/domain-naming-debt.md AGENTS.md
git commit -m "docs: mark Go domain vocabulary rename complete"
```

---

## Self-review

**Spec coverage:**
- All debt-doc Go rename targets → Tasks 2–5
- `ProcessPageView.ActionList` → `Detail` → Task 5 + Task 6
- `TimelineSubstep.Action` → `Body` → Task 5 + Task 6
- Test file renames → Tasks 4–5
- Docs → Task 7
- Explicitly deferred items listed in header and file map

**Placeholder scan:** No TBD/TODO steps.

**Type consistency:** `SubstepBodyView` defined Task 3; `StreamInstanceDetailView.SelectedBody` and `TimelineSubstep.Body` both `*SubstepBodyView`; `buildSubstepViews` returns `[]SubstepBodyView` consumed by `decorateTimelineSubstepBodies`.

---

## Execution handoff

**Plan complete and saved to `docs/superpowers/plans/2026-07-13-go-domain-vocabulary-rename.md`. Two execution options:**

**1. Subagent-Driven (recommended)** — dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** — execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
