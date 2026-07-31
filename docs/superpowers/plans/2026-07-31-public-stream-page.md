# Public stream page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a public, shareable `GET /streams/:key` page reachable from landing `public-stream-card` links, with catalog meta, recent completed-run cards (DPP-linked when possible), and a full read-only workflow blueprint.

**Architecture:** Dedicated public handler beside the existing `/streams/public` HTMX partial. Header reuses `buildPublicStreamCardView` fields; recent runs filter `ListRecentProcessesByWorkflow` to `processStatusDone` (cap 8); blueprint reuses `buildWorkflowPreviewProcess` + `buildStreamInstanceDetailView` + `makeStreamInstanceDetailReadOnly` with `HideStatus`. Landing cards gain `Href` via `publicStreamPath`.

**Tech Stack:** Go `net/http` + `html/template`, existing stream timeline/substep components, Vite CSS modules, Go unit tests (`go test`).

**Spec:** `docs/superpowers/specs/2026-07-31-public-stream-page-design.md`

## Global Constraints

- Public URL is `/streams/:key` (no trailing slash in generated links); accept `/streams/:key/`.
- Keep exact `/streams/public` as the HTMX results partial; `:key` must not steal it.
- Catalog gate: key must exist in `workflowCatalog()` / `workflowByKey`; unknown → 404. Category match not required.
- Same page for guests and signed-in users; no redirect to `/my`; no signup/workspace CTA.
- Recent runs: **completed only**, newest first, cap **8**; incomplete/terminated never listed.
- Run card meta: status + completed date only; DPP cue + `<a href="/01/…">` iff `process.DPP` set; else static card.
- Blueprint: full substep preview, `HideStatus`, read-only; no complete/start/terminate forms.
- Light shared tokens only; follow `docs/css.md` + `.agents/skills/attesta-ui-components`.
- Authenticated `/my/streams/:key` behavior unchanged.

## File map

| File | Responsibility |
|------|----------------|
| `server/cmd/server/paths.go` | `publicStreamPath(key string) string` |
| `server/cmd/server/paths_test.go` | Path helper tests |
| `server/cmd/server/components.go` | `Href` on `PublicStreamCardView`; `PublicStreamRunCardView`; `PublicStreamPageView` |
| `server/cmd/server/public_home.go` | Set `Href` in `buildPublicStreamCardView` |
| `server/cmd/server/public_stream.go` | Page handler, recent-run builder, blueprint builder |
| `server/cmd/server/public_stream_test.go` | Handler + builder + template tests |
| `server/cmd/server/public_stream_card_test.go` | Update card link assertions |
| `server/cmd/server/home_handler_test.go` / `public_home_test.go` | Assert homepage cards link to `/streams/{key}` |
| `server/cmd/server/main.go` | Register `/streams/` in `newMux` |
| `server/cmd/server/test_helpers_test.go` | Stub `public_stream_body` / `public_stream.html` in `testTemplates()` if any stub-based tests hit the route |
| `server/templates/components/public_stream_card.html` | Whole-card `<a>` when `Href` set |
| `server/templates/components/public_stream_run_card.html` | Full component: link or static article |
| `server/templates/pages/public_stream.html` | Page body + layout wrapper |
| `server/templates/layout.html` | Dispatch `public_stream_body` |
| `web/src/styles/components/public-stream-card.css` | Anchor reset for linked card |
| `web/src/styles/components/public-stream-run-card.css` | Run card styles |
| `web/src/styles/pages/public-stream.css` | Page layout |
| `web/src/styles/components.css` | Import run-card CSS |
| `web/src/styles/pages.css` | Import public-stream page CSS |

---

### Task 1: Public path helper + landing card link

**Files:**
- Modify: `server/cmd/server/paths.go`
- Modify: `server/cmd/server/paths_test.go`
- Modify: `server/cmd/server/components.go` (`PublicStreamCardView`)
- Modify: `server/cmd/server/public_home.go` (`buildPublicStreamCardView`)
- Modify: `server/templates/components/public_stream_card.html`
- Modify: `web/src/styles/components/public-stream-card.css`
- Modify: `server/cmd/server/public_stream_card_test.go`
- Modify: `server/cmd/server/home_handler_test.go` and/or `public_home_test.go` (assert `href="/streams/…"`)

**Interfaces:**
- Produces:
  - `func publicStreamPath(key string) string` → `"/streams/" + trimmed key` (no trailing slash)
  - `PublicStreamCardView.Href string`
  - `buildPublicStreamCardView` sets `Href: publicStreamPath(key)`

- [ ] **Step 1: Write failing path + card template tests**

Add to `paths_test.go`:

```go
func TestPublicStreamPath(t *testing.T) {
	if got := publicStreamPath("  wf-a  "); got != "/streams/wf-a" {
		t.Fatalf("publicStreamPath = %q, want /streams/wf-a", got)
	}
	if got := publicStreamPath(""); got != "/streams/" {
		t.Fatalf("empty key = %q, want /streams/", got)
	}
}
```

In `public_stream_card_test.go`, change `TestPublicStreamCardTemplateRendersCoreFields` so the card **with** `Href` renders a link, and add:

```go
func TestPublicStreamCardTemplateLinksWhenHrefSet(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	card := PublicStreamCardView{
		Name: "Linked Stream",
		Href: "/streams/linked",
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_card", card); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	if !strings.Contains(body, `href="/streams/linked"`) {
		t.Fatalf("expected href, got: %s", body)
	}
	if !strings.Contains(body, `class="public-stream-card"`) {
		t.Fatalf("expected public-stream-card class, got: %s", body)
	}
	if strings.Contains(body, "<article") {
		t.Fatalf("linked card must not use article shell, got: %s", body)
	}
}

func TestPublicStreamCardTemplateStaysArticleWithoutHref(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	card := PublicStreamCardView{Name: "Static"}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_card", card); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	if !strings.Contains(body, `<article class="public-stream-card"`) {
		t.Fatalf("expected article shell, got: %s", body)
	}
	if strings.Contains(body, `href=`) {
		t.Fatalf("empty Href must not link, got: %s", body)
	}
}
```

Remove the old “must not contain `<a` / `href=`” assertions from `TestPublicStreamCardTemplateRendersCoreFields` (that test should pass `Href` empty or move link checks to the new tests).

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && go test ./cmd/server -run 'TestPublicStreamPath|TestPublicStreamCardTemplateLinksWhenHrefSet|TestPublicStreamCardTemplateStaysArticleWithoutHref' -count=1`

Expected: FAIL — `publicStreamPath` undefined and/or template still always `<article>` without href.

- [ ] **Step 3: Implement path, view field, builder, template, CSS**

`paths.go`:

```go
func publicStreamPath(key string) string {
	return "/streams/" + strings.TrimSpace(key)
}
```

`components.go` — add to `PublicStreamCardView`:

```go
Href string // /streams/:key when set; empty keeps a non-link article
```

`buildPublicStreamCardView` return literal — add:

```go
Href: publicStreamPath(key),
```

`public_stream_card.html` — wrap content:

```html
{{ define "public_stream_card" }}
  {{ if .Href }}
  <a class="public-stream-card" href="{{ .Href }}">
  {{ else }}
  <article class="public-stream-card">
  {{ end }}
    {{/* existing header + foot unchanged */}}
  {{ if .Href }}
  </a>
  {{ else }}
  </article>
  {{ end }}
{{ end }}
```

`public-stream-card.css` — add after `.public-stream-card { … }`:

```css
a.public-stream-card {
  color: inherit;
  text-decoration: none;
}

a.public-stream-card:hover {
  border-color: var(--primary);
}
```

Update file header comment: card may be `<a>` or `<article>`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd server && go test ./cmd/server -run 'TestPublicStreamPath|TestPublicStreamCardTemplate|TestBuildPublicStreamCardView|TestHandlePublicHome|TestPublicStreamCardsForPath' -count=1`

Expected: PASS (fix any homepage assertions that assumed no `href=` on cards — require `href="/streams/` + key).

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/paths.go server/cmd/server/paths_test.go \
  server/cmd/server/components.go server/cmd/server/public_home.go \
  server/templates/components/public_stream_card.html \
  web/src/styles/components/public-stream-card.css \
  server/cmd/server/public_stream_card_test.go \
  server/cmd/server/home_handler_test.go server/cmd/server/public_home_test.go
git commit -m "$(cat <<'EOF'
feat(home): link public stream cards to /streams/:key

EOF
)"
```

---

### Task 2: Recent completed-run cards (view + builder + component)

**Files:**
- Modify: `server/cmd/server/components.go`
- Create: `server/cmd/server/public_stream.go` (builders only in this task)
- Create: `server/cmd/server/public_stream_test.go`
- Create: `server/templates/components/public_stream_run_card.html`
- Create: `web/src/styles/components/public-stream-run-card.css`
- Modify: `web/src/styles/components.css`

**Interfaces:**
- Produces:
  - `const publicStreamRecentRunLimit = 8`
  - ```go
    type PublicStreamRunCardView struct {
        StatusLabel   string // "Completed"
        CompletedAt   string // human-readable; may be empty
        DigitalLink   string // /01/… when DPP present; empty = non-link
        PassportChip  bool   // true when DigitalLink set
    }
    ```
  - `func processCompletedAt(process *Process) time.Time`
  - `func buildPublicStreamRunCards(def WorkflowDef, processes []Process) []PublicStreamRunCardView`
  - Template define `public_stream_run_card`

- [ ] **Step 1: Write failing builder + template tests**

Add to `public_stream_test.go`:

```go
package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestBuildPublicStreamRunCardsOnlyCompletedCappedWithDPP(t *testing.T) {
	doneAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	def := WorkflowDef{Steps: []WorkflowStep{{Substep: []WorkflowSub{{SubstepID: "1.1"}}}}}

	processes := []Process{
		{ID: primitive.NewObjectID(), Status: processStatusActive, Progress: map[string]ProcessStep{"1_1": {State: "pending"}}},
		{ID: primitive.NewObjectID(), Status: processStatusTerminated, Termination: &ProcessTermination{EndedAt: doneAt}},
		{
			ID:     primitive.NewObjectID(),
			Status: processStatusDone,
			Progress: map[string]ProcessStep{
				"1_1": {State: "done", DoneAt: &doneAt},
			},
			DPP: &ProcessDPP{GTIN: "09506000134352", Lot: "LOT-1", Serial: "SER-1", GeneratedAt: doneAt},
		},
		{
			ID:     primitive.NewObjectID(),
			Status: processStatusDone,
			Progress: map[string]ProcessStep{
				"1_1": {State: "done", DoneAt: ptrTime(doneAt.Add(-time.Hour))},
			},
		},
	}
	// Newest-first input: builder must preserve ListRecent order among completed only.
	ordered := []Process{processes[2], processes[3], processes[0], processes[1]}
	cards := buildPublicStreamRunCards(def, ordered)
	if len(cards) != 2 {
		t.Fatalf("len = %d, want 2 completed", len(cards))
	}
	if cards[0].DigitalLink != digitalLinkURL("09506000134352", "LOT-1", "SER-1") {
		t.Fatalf("first DigitalLink = %q", cards[0].DigitalLink)
	}
	if !cards[0].PassportChip || cards[0].StatusLabel != "Completed" {
		t.Fatalf("first card = %#v", cards[0])
	}
	if cards[1].DigitalLink != "" || cards[1].PassportChip {
		t.Fatalf("second card must be non-link, got %#v", cards[1])
	}
}

func TestBuildPublicStreamRunCardsRespectsCap(t *testing.T) {
	def := WorkflowDef{}
	doneAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	var processes []Process
	for i := 0; i < publicStreamRecentRunLimit+3; i++ {
		at := doneAt.Add(-time.Duration(i) * time.Hour)
		processes = append(processes, Process{
			ID:       primitive.NewObjectID(),
			Status:   processStatusDone,
			Progress: map[string]ProcessStep{"1_1": {State: "done", DoneAt: &at}},
		})
	}
	cards := buildPublicStreamRunCards(def, processes)
	if len(cards) != publicStreamRecentRunLimit {
		t.Fatalf("len = %d, want %d", len(cards), publicStreamRecentRunLimit)
	}
}

func TestPublicStreamRunCardTemplateLinkVsStatic(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var linked bytes.Buffer
	if err := tmpl.ExecuteTemplate(&linked, "public_stream_run_card", PublicStreamRunCardView{
		StatusLabel:  "Completed",
		CompletedAt:  "1 Jul 2026 at 12:00 UTC",
		DigitalLink:  "/01/09506000134352/10/LOT-1/21/SER-1",
		PassportChip: true,
	}); err != nil {
		t.Fatalf("linked render: %v", err)
	}
	lb := linked.String()
	if !strings.Contains(lb, `href="/01/09506000134352/10/LOT-1/21/SER-1"`) {
		t.Fatalf("expected DPP href, got: %s", lb)
	}
	if !strings.Contains(lb, "DPP") {
		t.Fatalf("expected DPP cue, got: %s", lb)
	}

	var static bytes.Buffer
	if err := tmpl.ExecuteTemplate(&static, "public_stream_run_card", PublicStreamRunCardView{
		StatusLabel: "Completed",
		CompletedAt: "1 Jul 2026 at 11:00 UTC",
	}); err != nil {
		t.Fatalf("static render: %v", err)
	}
	sb := static.String()
	if strings.Contains(sb, "href=") {
		t.Fatalf("static card must not link, got: %s", sb)
	}
	if !strings.Contains(sb, `<article class="public-stream-run-card"`) {
		t.Fatalf("expected article, got: %s", sb)
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && go test ./cmd/server -run 'TestBuildPublicStreamRunCards|TestPublicStreamRunCardTemplate' -count=1`

Expected: FAIL — undefined types/funcs / missing template.

- [ ] **Step 3: Implement view, builders, template, CSS**

`components.go`:

```go
// PublicStreamRunCardView is one recent completed run on the public stream page.
type PublicStreamRunCardView struct {
	StatusLabel  string
	CompletedAt  string
	DigitalLink  string
	PassportChip bool
}
```

`public_stream.go`:

```go
package main

import (
	"strings"
	"time"
)

const publicStreamRecentRunLimit = 8

func processCompletedAt(process *Process) time.Time {
	if process == nil {
		return time.Time{}
	}
	var latest time.Time
	for _, step := range process.Progress {
		if step.DoneAt != nil && step.DoneAt.After(latest) {
			latest = *step.DoneAt
		}
	}
	if latest.IsZero() && process.DPP != nil && !process.DPP.GeneratedAt.IsZero() {
		return process.DPP.GeneratedAt.UTC()
	}
	return latest
}

func buildPublicStreamRunCards(def WorkflowDef, processes []Process) []PublicStreamRunCardView {
	out := make([]PublicStreamRunCardView, 0, publicStreamRecentRunLimit)
	for i := range processes {
		p := processes[i]
		p.Progress = normalizeProgressKeys(p.Progress)
		if deriveProcessStatus(def, &p) != processStatusDone {
			continue
		}
		card := PublicStreamRunCardView{
			StatusLabel: "Completed",
			CompletedAt: humanReadableTraceabilityTime(processCompletedAt(&p)),
		}
		if p.DPP != nil {
			gtin := strings.TrimSpace(p.DPP.GTIN)
			lot := strings.TrimSpace(p.DPP.Lot)
			serial := strings.TrimSpace(p.DPP.Serial)
			if gtin != "" && lot != "" && serial != "" {
				card.DigitalLink = digitalLinkURL(gtin, lot, serial)
				card.PassportChip = true
			}
		}
		out = append(out, card)
		if len(out) >= publicStreamRecentRunLimit {
			break
		}
	}
	return out
}
```

`public_stream_run_card.html`:

```html
{{ define "public_stream_run_card" }}
  {{ if .DigitalLink }}
  <a class="public-stream-run-card" href="{{ .DigitalLink }}">
  {{ else }}
  <article class="public-stream-run-card">
  {{ end }}
    <div class="public-stream-run-card-meta">
      <span class="public-stream-run-card-status">{{ .StatusLabel }}</span>
      {{ if .CompletedAt }}
      <time class="public-stream-run-card-date">{{ .CompletedAt }}</time>
      {{ end }}
    </div>
    {{ if .PassportChip }}
    <span class="public-stream-run-card-dpp">
      DPP
      {{ template "icon-file-text" . }}
    </span>
    {{ end }}
  {{ if .DigitalLink }}
  </a>
  {{ else }}
  </article>
  {{ end }}
{{ end }}
```

`public-stream-run-card.css` — light card shell (`--card`, `--border`, padding, radius); `a.public-stream-run-card` inherits color / no underline; DPP chip mirrors public stream card chip tokens. Header comment with markup tree.

`components.css` — add:

```css
@import url("./components/public-stream-run-card.css");
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd server && go test ./cmd/server -run 'TestBuildPublicStreamRunCards|TestPublicStreamRunCardTemplate|TestProcessCompletedAt' -count=1`

Expected: PASS. (Optional: add a tiny `TestProcessCompletedAt` if you extract edge cases.)

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/components.go server/cmd/server/public_stream.go \
  server/cmd/server/public_stream_test.go \
  server/templates/components/public_stream_run_card.html \
  web/src/styles/components/public-stream-run-card.css \
  web/src/styles/components.css
git commit -m "$(cat <<'EOF'
feat(public-stream): add completed run cards with optional DPP links

EOF
)"
```

---

### Task 3: Public stream page template + blueprint + CSS

**Files:**
- Modify: `server/cmd/server/components.go` (`PublicStreamPageView`)
- Modify: `server/cmd/server/public_stream.go` (blueprint helper)
- Modify: `server/cmd/server/public_stream_test.go`
- Create: `server/templates/pages/public_stream.html`
- Modify: `server/templates/layout.html`
- Create: `web/src/styles/pages/public-stream.css`
- Modify: `web/src/styles/pages.css`
- Modify: `docs/css.md` exceptions only if needed (stem `public_stream` → `public-stream.css` is normal)

**Interfaces:**
- Produces:
  - ```go
    type PublicStreamPageView struct {
        PageBase
        Header PublicStreamCardView // Href unused on page header; Name/Description/metrics/orgs
        RecentRuns []PublicStreamRunCardView
        Blueprint StreamInstanceDetailView
    }
    ```
  - `func (s *Server) buildPublicStreamBlueprint(ctx context.Context, cfg RuntimeConfig, workflowKey string) StreamInstanceDetailView`
  - Template defines `public_stream_body` + `public_stream.html`
  - Layout dispatches `public_stream_body` (normal `.page`, **not** `page-landing`; keep default `.site-footer`)

- [ ] **Step 1: Write failing template + blueprint tests**

```go
func TestBuildPublicStreamBlueprintIsReadOnly(t *testing.T) {
	server := &Server{tmpl: parseTestTemplates(t), store: NewMemoryStore()}
	cfg, err := parseRuntimeConfigData("wf.yaml", []byte(minimalCategorizedWorkflowYAML("")))
	if err != nil {
		t.Fatal(err)
	}
	view := server.buildPublicStreamBlueprint(t.Context(), cfg, "wf")
	if !view.HideStatus {
		t.Fatal("expected HideStatus")
	}
	if view.CanTerminate || view.TerminateAction != "" {
		t.Fatalf("must not expose terminate, got %#v", view)
	}
	if len(view.Timeline) == 0 {
		t.Fatal("expected timeline steps")
	}
	for _, step := range view.Timeline {
		for _, sub := range step.Substeps {
			if sub.Body == nil || !sub.Body.ReadOnly {
				t.Fatalf("substep body must be read-only, got %#v", sub.Body)
			}
		}
	}
}

func TestPublicStreamBodyTemplateRendersSections(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := PublicStreamPageView{
		PageBase: PageBase{Body: "public_stream_body"},
		Header: PublicStreamCardView{
			Name:        "Pilot Workflow",
			Description: "Gallium batches",
			StepCount:   3,
			RoleCount:   5,
		},
		RecentRuns: []PublicStreamRunCardView{{
			StatusLabel: "Completed",
			CompletedAt: "1 Jul 2026 at 12:00 UTC",
		}},
		Blueprint: StreamInstanceDetailView{
			HideStatus: true,
			Timeline: []TimelineStep{{
				ID: "1", Title: "Step 1",
				Substeps: []TimelineSubstep{{
					ID: "1.1", Title: "Input",
					Body: &SubstepBodyView{Mode: "preview", Title: "Input", ReadOnly: true, Reason: "Public preview."},
				}},
			}},
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_body", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`class="public-stream"`,
		"Pilot Workflow",
		"Gallium batches",
		"Recent completed runs",
		"Workflow",
		`class="public-stream-run-card"`,
		`class="stream-timeline-list"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in body, got: %s", want, body)
		}
	}
	for _, mustNot := range []string{
		`action=`,
		"/instance/start",
		"/terminate",
		"Sign in to run",
		"Open in workspace",
	} {
		if strings.Contains(body, mustNot) {
			t.Fatalf("must not contain %q, got: %s", mustNot, body)
		}
	}
}

func TestPublicStreamBodyTemplateEmptyRuns(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := PublicStreamPageView{
		Header: PublicStreamCardView{Name: "Empty Runs"},
		Blueprint: StreamInstanceDetailView{HideStatus: true},
	}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_body", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(out.String(), "No completed runs yet.") {
		t.Fatalf("expected empty copy, got: %s", out.String())
	}
}
```

Check `stream_timeline.html` for the real root class name and use that string in the assertion.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && go test ./cmd/server -run 'TestBuildPublicStreamBlueprint|TestPublicStreamBodyTemplate' -count=1`

Expected: FAIL — missing types/templates/helpers.

- [ ] **Step 3: Implement page view, blueprint builder, templates, layout, CSS**

`components.go`:

```go
// PublicStreamPageView is the view model for templates/pages/public_stream.html.
type PublicStreamPageView struct {
	PageBase
	Header     PublicStreamCardView
	RecentRuns []PublicStreamRunCardView
	Blueprint  StreamInstanceDetailView
}
```

`public_stream.go` — add:

```go
func (s *Server) buildPublicStreamBlueprint(ctx context.Context, cfg RuntimeConfig, workflowKey string) StreamInstanceDetailView {
	preview := makeStreamInstanceDetailReadOnly(
		s.buildStreamInstanceDetailView(
			ctx,
			cfg,
			workflowKey,
			buildWorkflowPreviewProcess(cfg.Workflow, workflowKey),
			Actor{},
			"",
			"",
			false,
		),
		"Public preview.",
	)
	preview.HideStatus = true
	preview.WorkflowPath = publicStreamPath(workflowKey) // avoid leaking /my paths in markup if referenced
	return preview
}
```

`public_stream.html` (page-local header — do **not** nest `public_stream_card`):

```html
{{ define "public_stream_body" }}
  <div class="public-stream u-max-w-7xl u-mx-auto stack">
    <header class="public-stream-header">
      <h1 class="public-stream-title">{{ .Header.Name }}</h1>
      {{ if .Header.Description }}
      <p class="public-stream-description">{{ .Header.Description }}</p>
      {{ end }}
      {{ if .Header.PassportEnabled }}
      <span class="public-stream-dpp">
        DPP
        {{ template "icon-file-text" . }}
      </span>
      {{ end }}
      <div class="public-stream-metrics">
        <div class="public-stream-metrics-row">
          {{ if eq .Header.InstanceCount 0 }}
          <span class="public-stream-metric">
            {{ template "icon-layers-2" . }}
            no runs yet
          </span>
          {{ else if .Header.AllCompleted }}
          <span class="public-stream-metric">
            {{ template "icon-layers-2" . }}
            {{ if eq .Header.InstanceCount 1 }}1 run{{ else }}{{ .Header.InstanceCount }} runs{{ end }}
          </span>
          <span class="public-stream-metric">
            {{ template "icon-check-circle" . }}
            all completed
          </span>
          {{ else }}
          <span class="public-stream-metric">
            {{ template "icon-layers-2" . }}
            {{ if eq .Header.InstanceCount 1 }}1 run{{ else }}{{ .Header.InstanceCount }} runs{{ end }}
          </span>
          <span class="public-stream-metric public-stream-metric-active">
            {{ template "icon-activity" . }}
            {{ if eq .Header.ActiveCount 1 }}1 active now{{ else }}{{ .Header.ActiveCount }} active now{{ end }}
          </span>
          {{ end }}
        </div>
        <div class="public-stream-metrics-row">
          <span class="public-stream-metric">
            {{ template "icon-list" . }}
            {{ if eq .Header.StepCount 1 }}1 step{{ else }}{{ .Header.StepCount }} steps{{ end }}
          </span>
          <span class="public-stream-metric">
            {{ template "icon-users-group" . }}
            {{ if eq .Header.RoleCount 1 }}1 role{{ else }}{{ .Header.RoleCount }} roles{{ end }}
          </span>
        </div>
      </div>
      {{ if .Header.Organizations }}
      <hr class="public-stream-rule" />
      <div class="public-stream-orgs">
        <div class="public-stream-orgs-head">
          <strong>Organizations</strong>
          <span class="public-stream-orgs-count">{{ .Header.OrganizationCount }}</span>
        </div>
        <div class="public-stream-avatars">
          {{ range .Header.Organizations }}
            {{ if .LogoURL }}
            <img src="{{ .LogoURL }}" alt="{{ .Name }}" width="32" height="32" />
            {{ else }}
            <span class="public-stream-avatar" role="img" aria-label="{{ .Name }}">{{ .Initials }}</span>
            {{ end }}
          {{ end }}
          {{ if .Header.OrganizationsOverflow }}
          <span
            class="public-stream-avatar-more"
            aria-label="{{ .Header.OrganizationsOverflow }} more organizations"
          >+{{ .Header.OrganizationsOverflow }}</span>
          {{ end }}
        </div>
      </div>
      {{ end }}
    </header>

    <section class="public-stream-runs" aria-labelledby="public-stream-runs-heading">
      <h2 id="public-stream-runs-heading">Recent completed runs</h2>
      {{ if .RecentRuns }}
      <div class="public-stream-runs-grid">
        {{ range .RecentRuns }}
          {{ template "public_stream_run_card" . }}
        {{ end }}
      </div>
      {{ else }}
      <p class="public-stream-runs-empty">No completed runs yet.</p>
      {{ end }}
    </section>

    <section class="public-stream-blueprint" aria-labelledby="public-stream-blueprint-heading">
      <h2 id="public-stream-blueprint-heading">Workflow</h2>
      {{ template "stream_timeline" .Blueprint.StreamTimeline }}
    </section>
  </div>
{{ end }}

{{ define "public_stream.html" }}
  {{ template "layout.html" . }}
{{ end }}
```

`layout.html` — inside `<main>`, add:

```html
{{ else if eq .Body "public_stream_body" }}
  {{ template "public_stream_body" . }}
```

`public-stream.css` — stack layout, section gaps, runs grid; reuse tokens; no new palette. Import from `pages.css`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd server && go test ./cmd/server -run 'TestBuildPublicStreamBlueprint|TestPublicStreamBodyTemplate' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/components.go server/cmd/server/public_stream.go \
  server/cmd/server/public_stream_test.go \
  server/templates/pages/public_stream.html server/templates/layout.html \
  web/src/styles/pages/public-stream.css web/src/styles/pages.css
git commit -m "$(cat <<'EOF'
feat(public-stream): add page template with read-only blueprint

EOF
)"
```

---

### Task 4: Handler + mux registration + end-to-end tests

**Files:**
- Modify: `server/cmd/server/public_stream.go` (handler)
- Modify: `server/cmd/server/main.go` (`newMux`)
- Modify: `server/cmd/server/public_stream_test.go`
- Modify: `server/cmd/server/test_helpers_test.go` (optional stub for `public_stream_body` if stub layout is used)
- Modify: `server/cmd/server/public_home_test.go` — ensure `/streams/public` still works after mux change

**Interfaces:**
- Produces:
  - `func (s *Server) handlePublicStream(w http.ResponseWriter, r *http.Request)`
  - `mux.HandleFunc("/streams/", s.handlePublicStream)` registered **in addition to** existing `/streams/public`
  - Handler rules:
    - GET only
    - Path must be `/streams/{key}` or `/streams/{key}/` (no further segments → 404)
    - If `key == "public"` → 404 (exact `/streams/public` is the partial; trailing-slash oddities must not render a stream page)
    - `workflowByKey` miss → 404
    - Build `Header` via `buildPublicStreamCardView` (clear `Href` on page header or leave unused)
    - `RecentRuns` via `ListRecentProcessesByWorkflow(ctx, key, 0)` then `buildPublicStreamRunCards`
    - `Blueprint` via `buildPublicStreamBlueprint`
    - Optional signed-in `pageBaseForUser`; body `"public_stream_body"`
    - Execute `public_stream.html`

- [ ] **Step 1: Write failing handler tests**

```go
func TestHandlePublicStreamOK(t *testing.T) {
	tempDir := t.TempDir()
	yaml := strings.Replace(
		minimalCategorizedWorkflowYAML("  categorySlug: supply-chain\n  subCategorySlug: procurement\n"),
		`name: "Workflow"`,
		`name: "Pilot Workflow"`,
		1,
	)
	if err := os.WriteFile(filepath.Join(tempDir, "pilot.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewMemoryStore()
	doneAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "pilot",
		Status:      processStatusDone,
		CreatedAt:   doneAt,
		Progress:    map[string]ProcessStep{"1_1": {State: "done", DoneAt: &doneAt}},
		DPP:         &ProcessDPP{GTIN: "09506000134352", Lot: "LOT-1", Serial: "SER-1", GeneratedAt: doneAt},
	})
	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "pilot",
		Status:      processStatusActive,
		CreatedAt:   doneAt,
		Progress:    map[string]ProcessStep{"1_1": {State: "pending"}},
	})
	server := &Server{store: store, configDir: tempDir, tmpl: parseTestTemplates(t)}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/streams/pilot", nil)
	server.handlePublicStream(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Pilot Workflow",
		"Recent completed runs",
		`href="/01/09506000134352/10/LOT-1/21/SER-1"`,
		"Workflow",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q, got: %s", want, body)
		}
	}
	if strings.Contains(body, "/instance/start") || strings.Contains(body, "Sign in to run") {
		t.Fatalf("unexpected CTA/actions in body")
	}
}

func TestHandlePublicStreamUnknownKey404(t *testing.T) {
	server := &Server{store: NewMemoryStore(), configDir: t.TempDir(), tmpl: parseTestTemplates(t)}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/streams/missing", nil)
	server.handlePublicStream(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestHandlePublicStreamRejectsNestedPath(t *testing.T) {
	tempDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tempDir, "pilot.yaml"), []byte(minimalCategorizedWorkflowYAML("")), 0o644)
	server := &Server{store: NewMemoryStore(), configDir: tempDir, tmpl: parseTestTemplates(t)}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/streams/pilot/instance/abc", nil)
	server.handlePublicStream(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestHandlePublicStreamTrailingSlashOK(t *testing.T) {
	tempDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tempDir, "pilot.yaml"), []byte(minimalCategorizedWorkflowYAML("")), 0o644)
	server := &Server{store: NewMemoryStore(), configDir: tempDir, tmpl: parseTestTemplates(t)}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/streams/pilot/", nil)
	server.handlePublicStream(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestNewMuxPublicStreamAndPublicPartialCoexist(t *testing.T) {
	tempDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tempDir, "pilot.yaml"), []byte(minimalCategorizedWorkflowYAML(
		"  categorySlug: supply-chain\n  subCategorySlug: procurement\n",
	)), 0o644)
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := &Server{store: store, configDir: tempDir, tmpl: parseTestTemplates(t)}
	mux := server.newMux()

	recPage := httptest.NewRecorder()
	mux.ServeHTTP(recPage, httptest.NewRequest(http.MethodGet, "/streams/pilot", nil))
	if recPage.Code != http.StatusOK {
		t.Fatalf("page status = %d", recPage.Code)
	}

	recPartial := httptest.NewRecorder()
	mux.ServeHTTP(recPartial, httptest.NewRequest(http.MethodGet, "/streams/public?category=supply-chain&subCategory=procurement", nil))
	if recPartial.Code != http.StatusOK {
		t.Fatalf("partial status = %d", recPartial.Code)
	}
	if !strings.Contains(recPartial.Body.String(), `id="public-home-stream-results"`) {
		t.Fatalf("partial body missing results root: %s", recPartial.Body.String())
	}
}
```

Note: workflow key from filename stem — confirm `pilot.yaml` → key `pilot` via existing catalog loader (same pattern as other `configDir` tests). If the loader uses a different key, adjust paths to match.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && go test ./cmd/server -run 'TestHandlePublicStream|TestNewMuxPublicStream' -count=1`

Expected: FAIL — handler missing / mux not registered.

- [ ] **Step 3: Implement handler + mux**

`public_stream.go`:

```go
func (s *Server) handlePublicStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimSuffix(r.URL.Path, "/")
	const prefix = "/streams/"
	if path == "/streams" || !strings.HasPrefix(path, prefix) {
		http.NotFound(w, r)
		return
	}
	rest := strings.TrimPrefix(path, prefix)
	if rest == "" || strings.Contains(rest, "/") || rest == "public" {
		http.NotFound(w, r)
		return
	}
	key := strings.TrimSpace(rest)
	cfg, err := s.workflowByKey(key)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ctx := r.Context()
	logoURLs := organizationLogoURLMap(ctx, s.identity)
	header, err := s.buildPublicStreamCardView(ctx, key, cfg, logoURLs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	header.Href = "" // page header is not a link back to itself

	var recent []PublicStreamRunCardView
	if s.store != nil {
		processes, listErr := s.store.ListRecentProcessesByWorkflow(ctx, key, 0)
		if listErr != nil {
			http.Error(w, listErr.Error(), http.StatusInternalServerError)
			return
		}
		recent = buildPublicStreamRunCards(cfg.Workflow, processes)
	}

	base := s.pageBase("public_stream_body", key, cfg.Workflow.Name)
	if user, _, err := s.currentUser(r); err == nil {
		base = s.pageBaseForUser(user, "public_stream_body", key, cfg.Workflow.Name)
	}

	view := PublicStreamPageView{
		PageBase:   base,
		Header:     header,
		RecentRuns: recent,
		Blueprint:  s.buildPublicStreamBlueprint(ctx, cfg, key),
	}
	if err := s.tmpl.ExecuteTemplate(w, "public_stream.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
```

`newMux` — register **after** `/streams/public`:

```go
mux.HandleFunc("/streams/public", s.handlePublicStreamsPartial)
mux.HandleFunc("/streams/", s.handlePublicStream)
```

(Go ServeMux: longer / more specific `/streams/public` wins over `/streams/`.)

- [ ] **Step 4: Run full related suite**

Run:

```bash
cd server && go test ./cmd/server -run 'TestHandlePublicStream|TestNewMuxPublicStream|TestHandlePublicStreamsPartial|TestPublicStream|TestPublicStreamCard|TestPublicStreamPath|TestHandlePublicHome' -count=1
```

Expected: PASS.

Also run CSS lint if the project has it: `task css:lint` (or skip if unavailable).

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/public_stream.go server/cmd/server/public_stream_test.go \
  server/cmd/server/main.go server/cmd/server/test_helpers_test.go
git commit -m "$(cat <<'EOF'
feat(public-stream): serve GET /streams/:key public page

EOF
)"
```

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| `/streams/:key` public page | 4 |
| Landing card → page link | 1 |
| Header meta (desc, DPP, metrics, orgs) | 3 |
| Recent completed runs only, cap 8 | 2 |
| DPP link cards vs static non-DPP | 2 |
| Full read-only blueprint | 3 |
| No CTA / no actions | 3, 4 |
| `/streams/public` preserved | 4 |
| Unknown key 404 | 4 |
| Shared topbar (layout) | 3 |
| Light tokens / component CSS | 2, 3 |

## Self-review notes

- No TBD placeholders; concrete signatures and test code included.
- `digitalLinkURL` (not a fictional `digitalLinkPath`) matches `dpp.go`.
- Workflow catalog key = YAML filename stem — tests must use matching `/streams/{stem}` URLs.
- Header metrics markup is fully specified in Task 3 (mirrors card copy).
- Timeline root class is `stream-timeline-list`.
- File-backed catalog keys use YAML basename stem (`pilot.yaml` → `pilot`) when `MemoryStore` has no Formata streams.
