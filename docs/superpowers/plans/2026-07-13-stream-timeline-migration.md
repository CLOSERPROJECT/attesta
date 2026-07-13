# Stream timeline component migration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract the accordion step/substep tree from `process.html` into `components/stream_timeline.html` (define `stream_timeline`), split inner defines, extract step-level CSS to `stream-timeline.css`, and add focused template tests — without Go type renames or CSS class prefix renames.

**Architecture:** Mirror the `substep_body` / `page_header` migration pattern. The public template entry `stream_timeline` continues to accept `ActionListView` (uses `.Timeline` and `.HideStatus` only). Inner defines `stream_timeline_step` and `stream_timeline_substep` live in the same file. Step chrome CSS moves to `components/stream-timeline.css`; substep accordion shell, status pills, override modal, and DPP-shared rules stay in `components/timeline.css`. Class names remain `.timeline-*` / `.substep-*` (no `.stream-timeline-*` rename in this pass).

**Tech Stack:** Go `html/template`, Vite CSS modules (`web/src/styles/`), existing `ActionListView` / `TimelineStep` / `TimelineSubstep` types in `server/cmd/server/main.go`.

**Decisions locked in (2026-07-13):**
- Keep `ActionListView` at call sites (no `StreamTimelineView`)
- Extract `stream-timeline.css` for step-level rules only
- Split inner template defines in one file
- Focused `stream_timeline_test.go` (steps/substeps, HideStatus, org logo fallback)
- Do not commit the untracked substep_body plan doc

---

## File map

| File | Responsibility |
|------|----------------|
| `server/templates/components/stream_timeline.html` | **Create.** Defines `stream_timeline`, `stream_timeline_step`, `stream_timeline_substep` |
| `server/templates/pages/process.html` | **Modify.** Replace `workflow_timeline` calls; delete old define (lines 217–314) |
| `server/templates/pages/stream.html` | **Modify.** Preview dialog: `workflow_timeline` → `stream_timeline` |
| `server/cmd/server/stream_timeline_test.go` | **Create.** Focused template render tests |
| `web/src/styles/components/stream-timeline.css` | **Create.** Step-level `.timeline-list`, `.timeline-step*`, `.timeline-substeps`, step chevron, responsive step rules |
| `web/src/styles/components/timeline.css` | **Modify.** Remove moved rules; keep substep shell, shared chevron group (substep + DPP), override modal |
| `web/src/styles/components.css` | **Modify.** Add `@import` for `stream-timeline.css` before `timeline.css` |
| `docs/css.md` | **Modify.** Template ↔ CSS index |
| `docs/domain-naming-debt.md` | **Modify.** Mark `stream_timeline` resolved |
| `AGENTS.md` | **Modify.** Note `stream_timeline` component |

**Out of scope:** `ActionView`/`ActionListView` renames, route/page renames, DPP `dpp-history-*` convergence, `.timeline-*` → `.stream-timeline-*` class renames, `StreamTimelineView` in `components.go`.

**References before starting:**
- `.agents/skills/attesta-ui-components/SKILL.md`
- `CONTEXT.md` (stream timeline glossary)
- `server/templates/components/substep_body.html` (inner-define pattern)
- `server/cmd/server/page_header_test.go` (test pattern)

---

### Task 1: Failing template tests

**Files:**
- Create: `server/cmd/server/stream_timeline_test.go`

- [ ] **Step 1: Write the failing test file**

```go
package main

import (
	"bytes"
	"strings"
	"testing"
)

func testStreamTimelineView() ActionListView {
	return ActionListView{
		WorkflowKey: "workflow",
		ProcessID:   "process-1",
		Timeline: []TimelineStep{{
			StepID:     "1",
			Title:      "Production",
			OrgName:    "Acme Org",
			OrgLogoURL: "https://example.com/logo.png",
			Expanded:   true,
			Substeps: []TimelineSubstep{{
				SubstepID: "1.1",
				Title:     "Capture batch data",
				Status:    "available",
				Selected:  true,
				Palette:   "blue",
				Action: &ActionView{
					WorkflowKey: "workflow",
					ProcessID:   "process-1",
					SubstepID:   "1.1",
					Title:       "Capture batch data",
					Status:      "available",
					FormSchema:  `{"type":"object"}`,
				},
			}},
		}},
	}
}

func TestStreamTimelineTemplateRendersStepsAndSubsteps(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "stream_timeline", testStreamTimelineView()); err != nil {
		t.Fatalf("render stream_timeline template: %v", err)
	}
	body := out.String()
	compactBody := strings.Join(strings.Fields(body), " ")

	for _, want := range []string{
		`class="timeline-list"`,
		`class="timeline-step"`,
		`class="timeline-step-summary"`,
		`class="timeline-step-org-mark"`,
		`src="https://example.com/logo.png"`,
		`class="timeline-substeps"`,
		`class="substep substep-available"`,
		`class="substep-accordion js-process-substep-panel"`,
		`data-substep-id="1.1"`,
		`class="substep-details"`,
		`<span class="status">available</span>`,
		"Production",
		"Acme Org",
		"Capture batch data",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered timeline, got: %s", want, body)
		}
	}
	if !strings.Contains(compactBody, `class="substep-accordion js-process-substep-panel" data-substep-id="1.1" open`) {
		t.Fatalf("expected selected substep accordion open, got: %s", body)
	}
}

func TestStreamTimelineTemplateHidesStatusWhenHideStatus(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := testStreamTimelineView()
	view.HideStatus = true

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "stream_timeline", view); err != nil {
		t.Fatalf("render stream_timeline template: %v", err)
	}
	body := out.String()

	if strings.Contains(body, `<span class="status">`) {
		t.Fatalf("did not expect status pill when HideStatus is true, got: %s", body)
	}
	if strings.Contains(body, `class="substep substep-available"`) {
		t.Fatalf("did not expect status-colored substep class when HideStatus is true, got: %s", body)
	}
	if !strings.Contains(body, `class="substep"`) {
		t.Fatalf("expected bare substep class, got: %s", body)
	}
}

func TestStreamTimelineTemplateRendersOrgLogoFallback(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := testStreamTimelineView()
	view.Timeline[0].OrgLogoURL = ""

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "stream_timeline", view); err != nil {
		t.Fatalf("render stream_timeline template: %v", err)
	}
	body := out.String()

	if strings.Contains(body, `src="https://example.com/logo.png"`) {
		t.Fatalf("did not expect org logo img when URL empty, got: %s", body)
	}
	if !strings.Contains(body, `class="timeline-step-org-mark"`) {
		t.Fatalf("expected org mark fallback container, got: %s", body)
	}
	if !strings.Contains(body, `class="icon-svg"`) {
		t.Fatalf("expected icon-no-org fallback svg, got: %s", body)
	}
}

func TestStreamTimelineTemplateRendersMissingActionMessage(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := testStreamTimelineView()
	view.Timeline[0].Substeps[0].Action = nil

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "stream_timeline", view); err != nil {
		t.Fatalf("render stream_timeline template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, "No data form configured for this substep.") {
		t.Fatalf("expected missing-action fallback copy, got: %s", body)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd server && go test ./cmd/server/ -run StreamTimeline -count=1
```

Expected: FAIL — `template stream_timeline not defined` (or similar execute error).

- [ ] **Step 3: Commit**

```bash
git add server/cmd/server/stream_timeline_test.go
git commit -m "$(cat <<'EOF'
test(ui): add stream_timeline template tests

EOF
)"
```

---

### Task 2: Extract stream_timeline template

**Files:**
- Create: `server/templates/components/stream_timeline.html`
- Modify: `server/templates/pages/process.html`
- Modify: `server/templates/pages/stream.html`

- [ ] **Step 1: Create the component template**

Create `server/templates/components/stream_timeline.html`:

```html
{{/* Stream timeline: accordion tree of steps and substeps (see CONTEXT.md) */}}

{{ define "stream_timeline" }}
  <div class="timeline-list">
    {{ range .Timeline }}
      {{ template "stream_timeline_step" dict "Step" . "HideStatus" $.HideStatus }}
    {{ end }}
  </div>
{{ end }}

{{ define "stream_timeline_step" }}
  <details
    class="timeline-step"
    {{ if .Step.Expanded }}open{{ end }}
  >
    <summary class="timeline-step-summary" aria-label="Open stream step">
      <div class="timeline-step-summary-main">
        <span class="pill pill-step-number pill-panel">{{ .Step.StepID }}</span>
        {{ if .Step.OrgLogoURL }}
          <img
            src="{{ .Step.OrgLogoURL }}"
            alt=""
            class="timeline-step-org-mark"
          />
        {{ else }}
          <span class="timeline-step-org-mark">
            {{ template "icon-no-org" . }}
          </span>
        {{ end }}
        <span class="timeline-step-copy">
          <span class="timeline-step-title">{{ .Step.Title }}</span>
          <span class="timeline-step-org"
            >{{ if .Step.OrgName }}
              {{ .Step.OrgName }}
            {{ else }}
              No organization
            {{ end }}</span
          >
        </span>
      </div>
      <span class="timeline-step-chevron" aria-hidden="true">
        {{ template "icon-chevron-down" . }}
      </span>
    </summary>
    <ul class="timeline-substeps">
      {{ range .Step.Substeps }}
        {{ template "stream_timeline_substep" dict "Substep" . "HideStatus" $.HideStatus }}
      {{ end }}
    </ul>
  </details>
{{ end }}

{{ define "stream_timeline_substep" }}
  <li
    class="substep{{ if not .HideStatus }}
      substep-{{ .Substep.Status }}
    {{ end }}"
    data-role-palette="{{ .Substep.Palette }}"
  >
    <details
      class="substep-accordion js-process-substep-panel"
      data-substep-id="{{ .Substep.SubstepID }}"
      {{ if .Substep.Selected }}open{{ end }}
    >
      <summary class="substep-accordion-summary">
        <div class="substep-summary-main">
          <span class="pill pill-panel">{{ .Substep.SubstepID }}</span>
          <div class="substep-title">
            <span class="timeline-step-title">{{ .Substep.Title }}</span>
            {{ if eq .Substep.Status "done" }}
              <div class="substep-meta">
                {{ if .Substep.DoneAt }}
                  <span class="time">Completed at {{ .Substep.DoneAt }}</span>
                {{ end }}
                {{ if .Substep.DoneBy }}
                  <span class="actor">by {{ .Substep.DoneBy }}</span>
                {{ else }}
                  <span class="actor">completed</span>
                {{ end }}
              </div>
            {{ end }}
          </div>
        </div>
        {{ if not .HideStatus }}
          <span class="status">
            {{ if .Substep.StatusLabel }}
              {{ .Substep.StatusLabel }}
            {{ else }}
              {{ .Substep.Status }}
            {{ end }}
          </span>
        {{ end }}
        <span class="substep-accordion-chevron" aria-hidden="true">
          {{ template "icon-chevron-down" . }}
        </span>
      </summary>
      <div class="substep-details">
        {{ with .Substep.Action }}
          {{ template "substep_body" . }}
        {{ else }}
          <p class="muted">
            No data form configured for this substep.
          </p>
        {{ end }}
      </div>
    </details>
  </li>
{{ end }}
```

- [ ] **Step 2: Update call sites in process.html**

In `server/templates/pages/process.html`, replace both occurrences:

```html
{{ template "workflow_timeline" .ActionList }}
```

with:

```html
{{ template "stream_timeline" .ActionList }}
```

(lines 23 and 31 in the current file)

- [ ] **Step 3: Delete the old define from process.html**

Remove the entire block from `{{ define "workflow_timeline" }}` through its closing `{{ end }}` (current lines 217–314). Do not leave a stub define.

- [ ] **Step 4: Update stream.html preview dialog**

In `server/templates/pages/stream.html` line 157, replace:

```html
{{ template "workflow_timeline" .Preview }}
```

with:

```html
{{ template "stream_timeline" .Preview }}
```

- [ ] **Step 5: Run template tests**

Run:
```bash
cd server && go test ./cmd/server/ -run StreamTimeline -count=1
```

Expected: PASS (all four `StreamTimeline` tests).

Also run existing integration tests that embed the timeline:

```bash
cd server && go test ./cmd/server/ -run 'ProcessTemplate|HomeTemplateRendersSidebarAndReadOnlyPreview' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add server/templates/components/stream_timeline.html server/templates/pages/process.html server/templates/pages/stream.html
git commit -m "$(cat <<'EOF'
feat(ui): extract stream_timeline component from process page

EOF
)"
```

---

### Task 3: Extract stream-timeline.css

**Files:**
- Create: `web/src/styles/components/stream-timeline.css`
- Modify: `web/src/styles/components/timeline.css`
- Modify: `web/src/styles/components.css`

**Rules to move** from `timeline.css` → `stream-timeline.css`:
- `.timeline-list`
- `.timeline-step`, `.timeline-step:hover`
- `.timeline-step-summary` (+ `::marker` / `::-webkit-details-marker`)
- `.timeline-step-summary-main`
- `.timeline-step-chevron` (standalone block with same properties as today)
- `.timeline-step[open] .timeline-step-chevron`
- `.timeline-step-chevron .icon-svg`
- `.timeline-step-org-mark` (+ `img`)
- `.timeline-step-copy`, `.timeline-step-title`, `.timeline-step-org`
- `.timeline-substeps`
- From `@media (--sm-down)`: `.timeline-step-summary`, `.timeline-step-summary-main`, `.timeline-step-org-mark`, `.timeline-substeps`
- From `@media (--md-down)`: `.timeline-step[open] > summary`, `.timeline-step[open]:has(.substep-accordion[open]) > summary`

**Rules to keep** in `timeline.css`:
- `.process-page`
- `.substep-accordion-chevron, .dpp-accordion-chevron` grouped sizing (remove `.timeline-step-chevron` from this group)
- `.substep-accordion-chevron .icon-svg, .dpp-accordion-chevron .icon-svg` (remove `.timeline-step-chevron` from this group)
- All `.substep*` rules, override modal, `.data-hash`, remaining responsive substep rules

- [ ] **Step 1: Create stream-timeline.css**

Create `web/src/styles/components/stream-timeline.css`:

```css
.timeline-list {
  display: grid;
  gap: var(--space-3);
  min-width: 0;
}

.timeline-step {
  border: 1px solid var(--border);
  border-radius: 4px;
  background: var(--card);
  overflow: visible;
}

.timeline-step:hover {
  border-color: var(--primary);
}

.timeline-step-summary {
  padding: var(--space-3);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  min-width: 0;
}

.timeline-step-summary::marker,
.timeline-step-summary::-webkit-details-marker {
  display: none;
}

.timeline-step-summary-main {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  min-width: 0;
}

.timeline-step-chevron {
  width: 32px;
  height: 32px;
  color: var(--muted-foreground);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition:
    transform 180ms ease,
    color 180ms ease,
    border-color 180ms ease;
}

.timeline-step[open] .timeline-step-chevron {
  transform: rotate(180deg);
  color: var(--foreground);
}

.timeline-step-chevron .icon-svg {
  width: 18px;
  height: 18px;
}

.timeline-step-org-mark {
  width: 44px;
  height: 44px;
  border-radius: 4px;
  border: 1px solid var(--border);
  background: var(--card);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  flex-shrink: 0;
}

.timeline-step-org-mark img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.timeline-step-copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: var(--space-1);
}

.timeline-step-title {
  font-size: var(--text-base);
  line-height: 1.2;
  font-weight: var(--font-bold);
  color: var(--foreground);
  overflow-wrap: anywhere;
}

.timeline-step-org {
  font-size: var(--text-sm);
  line-height: var(--leading-tight);
  color: var(--muted-foreground);
  overflow-wrap: anywhere;
}

.timeline-substeps {
  list-style: none;
  padding: var(--space-4) var(--space-3) var(--space-3);
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  min-width: 0;
}

@media (--sm-down) {
  .timeline-step-summary {
    flex-wrap: wrap;
    gap: var(--space-2);
    padding: var(--space-3);
  }

  .timeline-step-summary-main {
    flex: 1 1 auto;
    min-width: 0;
  }

  .timeline-step-org-mark {
    width: 36px;
    height: 36px;
  }

  .timeline-substeps {
    padding-inline: var(--space-3);
  }
}

@media (--md-down) {
  .timeline-step[open] > summary {
    position: sticky;
    top: 0;
    background: var(--card);
    z-index: 10;
  }

  .timeline-step[open]:has(.substep-accordion[open]) > summary {
    position: static;
  }
}
```

- [ ] **Step 2: Trim moved rules from timeline.css**

At the top of `web/src/styles/components/timeline.css`, remove the blocks listed above. Update the shared chevron selector from:

```css
.timeline-step-chevron,
.substep-accordion-chevron,
.dpp-accordion-chevron {
```

to:

```css
.substep-accordion-chevron,
.dpp-accordion-chevron {
```

Update the icon-svg grouped rule similarly (drop `.timeline-step-chevron`).

Remove step-related blocks from the `@media (--sm-down)` and `@media (--md-down)` sections at the bottom of `timeline.css` (keep only substep/data-hash rules in those media blocks).

- [ ] **Step 3: Wire import in components.css**

In `web/src/styles/components.css`, add before the timeline import:

```css
@import url("./components/stream-timeline.css");
```

Resulting order at top of file:

```css
@import url("./components/page-header.css");
@import url("./components/shared.css");
@import url("./components/stream-timeline.css");
@import url("./components/timeline.css");
@import url("./components/substep-body.css");
```

- [ ] **Step 4: Run CSS lint and Go tests**

Run:
```bash
task css:lint
cd server && go test ./cmd/server/ -count=1
```

Expected: both pass.

- [ ] **Step 5: Commit**

```bash
git add web/src/styles/components/stream-timeline.css web/src/styles/components/timeline.css web/src/styles/components.css
git commit -m "$(cat <<'EOF'
refactor(ui): extract stream-timeline step chrome CSS

EOF
)"
```

---

### Task 4: Update documentation

**Files:**
- Modify: `docs/css.md`
- Modify: `docs/domain-naming-debt.md`
- Modify: `AGENTS.md`

- [ ] **Step 1: Update docs/css.md**

In the **Component modules** table, add:

```markdown
| `components/stream-timeline.css` | `.timeline-list`, `.timeline-step*` | `components/stream_timeline.html`, `pages/dpp.html` (`.timeline-step-title` reuse) |
```

In the **Template ↔ CSS index** table:
- Add row: `components/stream_timeline.html` | `components/stream-timeline.css` | `components/timeline.css`, `components/substep-body.css`, `role-palette.css`
- Update `pages/process.html` row: add `components/stream-timeline.css` to "Also uses"
- Update `pages/stream.html` row: add `components/stream-timeline.css` to "Also uses"

In the components barrel line (~19), mention `stream-timeline` alongside timeline.

- [ ] **Step 2: Update docs/domain-naming-debt.md**

Under **Resolved**, add (mirror substep_body block):

```markdown
### Resolved in stream_timeline migration (2026-07-13)

- `workflow_timeline` → `components/stream_timeline.html`, define `stream_timeline`
- `timeline.css` step chrome → `components/stream-timeline.css` (class prefix rename deferred)
```

Remove `workflow_timeline` from **Still open (templates & defines)** and the `timeline.css` → `stream-timeline.css` line from **Still open (CSS)**.

Update **Open items** — remove `stream_timeline` extraction; next work becomes Go rename pass or other debt.

- [ ] **Step 3: Update AGENTS.md**

In the Templates section (near substep_body note), add:

```markdown
- Stream timeline (`server/templates/components/stream_timeline.html`) renders the step/substep accordion tree on stream instance detail and stream dashboard preview; inner defines `stream_timeline_step` and `stream_timeline_substep` dispatch to `substep_body`.
```

- [ ] **Step 4: Commit**

```bash
git add docs/css.md docs/domain-naming-debt.md AGENTS.md docs/superpowers/plans/2026-07-13-stream-timeline-migration.md
git commit -m "$(cat <<'EOF'
docs(ui): document stream_timeline component migration

EOF
)"
```

---

### Task 5: Final verification

- [ ] **Step 1: Run full server test suite**

```bash
cd server && go test ./cmd/server/ -count=1
```

Expected: all tests PASS.

- [ ] **Step 2: Run CSS lint**

```bash
task css:lint
```

Expected: PASS.

- [ ] **Step 3: Grep for stale references**

```bash
rg 'workflow_timeline' server/templates docs
```

Expected: no matches (only historical mentions in git log / this plan file are OK).

- [ ] **Step 4: Optional manual smoke check**

If Docker stack is running locally:

1. Open `/w/:key/process/:id` — timeline renders, substep accordion opens, completion form works.
2. Open `/w/:key/` → Preview dialog — timeline renders without status pills (`HideStatus`).

---

## Self-review checklist

| Requirement | Task |
|-------------|------|
| Extract to `components/stream_timeline.html`, define `stream_timeline` | Task 2 |
| Split inner defines | Task 2 |
| Keep `ActionListView` at call sites | Task 2 (no Go changes) |
| Update process + stream call sites | Task 2 |
| Extract step-level CSS only | Task 3 |
| No `.timeline-*` class renames | Task 3 (explicit) |
| Focused template tests | Task 1 |
| Update docs/css.md, AGENTS.md, debt doc | Task 4 |
| No Go type renames | Out of scope |
| No DPP convergence | Out of scope |
