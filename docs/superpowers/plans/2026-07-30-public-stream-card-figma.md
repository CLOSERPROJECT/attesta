# Public stream card Figma layout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restyle the public homepage `public_stream_card` to the Figma shell with product metadata (runs / active|completed / steps / roles), a DPP chip under the description, and a plain bold org count.

**Architecture:** Keep the existing full component (`public_stream_card` template + `public-stream-card.css`). Extend `PublicStreamCardView` with count fields; drop the step-list view type. Builder fills counts via `sortedSteps`, `substepRoles` (distinct), and org total before avatar truncation. Template + CSS rewrite; Lucide `icon-file-text` added in `icons.html` (no Figma asset downloads).

**Tech Stack:** Go `html/template`, existing Lucide SVG templates in `icons.html`, Vite CSS component module, Go unit tests (`go test`).

**Spec:** `docs/superpowers/specs/2026-07-30-public-stream-card-figma-design.md`

## Global Constraints

- Light shared tokens only — no Figma dark palette, no new `--landing-*`.
- Do **not** download assets from Figma MCP.
- Remove Stream + Product Passport badges; DPP chip under description when `PassportEnabled`.
- Zero runs → single metric `no runs yet` (hide active/completed).
- Roles = distinct trimmed slugs from `substepRoles` across all substeps (not YAML role catalog length).
- Org total = plain bold number next to “Organizations” (not a chip); avatars + `+N` overflow unchanged.
- Card remains a non-link `<article>`.
- Follow `.agents/skills/attesta-ui-components` + `docs/css.md` for the existing full component.

## File map

| File | Responsibility |
|------|----------------|
| `server/cmd/server/components.go` | `PublicStreamCardView` fields; remove `PublicStreamCardStepView` |
| `server/cmd/server/public_home.go` | `buildPublicStreamCardView` + `publicStreamCardRoleCount` helper |
| `server/cmd/server/public_stream_card_test.go` | Template + helper unit tests |
| `server/templates/components/public_stream_card.html` | New card markup |
| `server/templates/icons.html` | Add `icon-file-text` |
| `web/src/styles/components/public-stream-card.css` | New layout; delete badges/steps rules |
| `web/src/styles/category-palette.css` | Drop obsolete `.public-stream-card-badge` rule if present |
| `server/cmd/server/home_handler_test.go` | Update public-home assertions for DPP chip / runs copy / no badges |

---

### Task 1: View model + role-count helper

**Files:**
- Modify: `server/cmd/server/components.go` (PublicStreamCardView / remove PublicStreamCardStepView)
- Modify: `server/cmd/server/public_home.go` (add `publicStreamCardRoleCount`)
- Modify: `server/cmd/server/public_stream_card_test.go` (helper tests; fix compile breaks from removed `Steps`)

**Interfaces:**
- Produces:
  - `PublicStreamCardView` with `StepCount`, `RoleCount`, `OrganizationCount` (ints); no `Steps` field
  - `func publicStreamCardRoleCount(def WorkflowDef) int`

- [ ] **Step 1: Write failing helper test**

Add to `server/cmd/server/public_stream_card_test.go`:

```go
func TestPublicStreamCardRoleCountDistinctAcrossSubsteps(t *testing.T) {
	def := WorkflowDef{
		Steps: []WorkflowStep{
			{
				Substep: []WorkflowSub{
					{Roles: []string{"qa", "ops"}},
					{Roles: []string{" qa "}}, // duplicate after trim
				},
			},
			{
				Substep: []WorkflowSub{
					{Role: "reviewer"},
					{Roles: []string{"ops", ""}},
				},
			},
		},
	}
	if got := publicStreamCardRoleCount(def); got != 3 {
		t.Fatalf("publicStreamCardRoleCount = %d, want 3 (qa, ops, reviewer)", got)
	}
	if got := publicStreamCardRoleCount(WorkflowDef{}); got != 0 {
		t.Fatalf("empty def role count = %d, want 0", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd server && go test ./cmd/server -run 'TestPublicStreamCardRoleCountDistinctAcrossSubsteps' -count=1`

Expected: FAIL — `publicStreamCardRoleCount` undefined (and possibly compile errors from `Steps` still referenced in other tests — leave those for Task 3; if the package does not compile, temporarily comment only the `Steps:` literals in existing tests or skip to Step 3’s struct change first so the package builds with `Steps` removed and old tests updated minimally to compile).

**Preferred order if package won’t compile:** apply Step 3 struct change, then make old `Steps`-using tests compile by deleting `Steps:` fields (they will fail assertions until Task 3 — that is OK for TDD on the helper).

- [ ] **Step 3: Update view model + implement helper**

In `components.go`, replace:

```go
// PublicStreamCardStepView is one blueprint step row on a public stream card.
type PublicStreamCardStepView struct {
	Title        string
	SubstepCount int
}

// PublicStreamCardView is the view model for templates/components/public_stream_card.html.
type PublicStreamCardView struct {
	Name                  string
	Description           string
	Steps                 []PublicStreamCardStepView
	PassportEnabled       bool
	InstanceCount         int
	ActiveCount           int  // active instances (dashboard Active: not done, not terminated)
	AllCompleted          bool // true when T>=1 and no active instances (settled)
	Organizations         []PublicStreamCardOrgView
	OrganizationsOverflow int // count beyond the first four avatars; 0 when none
}
```

with:

```go
// PublicStreamCardView is the view model for templates/components/public_stream_card.html.
type PublicStreamCardView struct {
	Name                  string
	Description           string
	PassportEnabled       bool
	InstanceCount         int
	ActiveCount           int  // active instances (dashboard Active: not done, not terminated)
	AllCompleted          bool // true when T>=1 and no active instances (settled)
	StepCount             int
	RoleCount             int
	OrganizationCount     int // total orgs before avatar truncation
	Organizations         []PublicStreamCardOrgView
	OrganizationsOverflow int // count beyond the first four avatars; 0 when none
}
```

In `public_home.go`, add:

```go
func publicStreamCardRoleCount(def WorkflowDef) int {
	seen := map[string]struct{}{}
	for _, step := range def.Steps {
		for _, sub := range step.Substep {
			for _, role := range substepRoles(sub) {
				seen[role] = struct{}{}
			}
		}
	}
	return len(seen)
}
```

(`substepRoles` already trims and handles legacy `Role`.)

- [ ] **Step 4: Run helper test to verify it passes**

Run: `cd server && go test ./cmd/server -run 'TestPublicStreamCardRoleCountDistinctAcrossSubsteps' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/components.go server/cmd/server/public_home.go server/cmd/server/public_stream_card_test.go
git commit -m "$(cat <<'EOF'
feat(public-home): add stream card role count helper and view fields

Prepare PublicStreamCardView for compact Figma metrics by dropping the step-list type and counting distinct substep roles.
EOF
)"
```

---

### Task 2: Fill counts in `buildPublicStreamCardView`

**Files:**
- Modify: `server/cmd/server/public_home.go` (`buildPublicStreamCardView`)
- Modify: `server/cmd/server/public_stream_card_test.go` or add builder-focused assertions (prefer a small unit test that calls the builder with a MemoryStore / nil store)

**Interfaces:**
- Consumes: `publicStreamCardRoleCount`, `sortedSteps`, `publicStreamCardOrganizations`
- Produces: populated `StepCount`, `RoleCount`, `OrganizationCount` on returned `PublicStreamCardView`

- [ ] **Step 1: Write failing builder test**

Add to `public_stream_card_test.go` (or `public_home_test.go` if that file already hosts builder tests):

```go
func TestBuildPublicStreamCardViewFillsStepRoleOrgCounts(t *testing.T) {
	s := &Server{store: NewMemoryStore()}
	cfg := RuntimeConfig{
		Workflow: WorkflowDef{
			Name:        "Counted Stream",
			Description: "counts",
			Steps: []WorkflowStep{
				{
					Title: "One",
					Substep: []WorkflowSub{
						{Roles: []string{"qa"}},
						{Roles: []string{"ops"}},
					},
				},
				{
					Title:   "Two",
					Substep: []WorkflowSub{{Roles: []string{"qa"}}},
				},
			},
		},
		Organizations: []WorkflowOrganization{
			{Slug: "a", Name: "Alpha"},
			{Slug: "b", Name: "Beta"},
			{Slug: "c", Name: "Gamma"},
			{Slug: "d", Name: "Delta"},
			{Slug: "e", Name: "Epsilon"},
		},
		DPP: DPPConfig{Enabled: true},
	}
	card, err := s.buildPublicStreamCardView(context.Background(), "counted", cfg, nil)
	if err != nil {
		t.Fatalf("buildPublicStreamCardView: %v", err)
	}
	if card.StepCount != 2 {
		t.Fatalf("StepCount = %d, want 2", card.StepCount)
	}
	if card.RoleCount != 2 {
		t.Fatalf("RoleCount = %d, want 2", card.RoleCount)
	}
	if card.OrganizationCount != 5 {
		t.Fatalf("OrganizationCount = %d, want 5", card.OrganizationCount)
	}
	if len(card.Organizations) != 4 || card.OrganizationsOverflow != 1 {
		t.Fatalf("avatars=%d overflow=%d, want 4/1", len(card.Organizations), card.OrganizationsOverflow)
	}
	if !card.PassportEnabled {
		t.Fatal("PassportEnabled want true")
	}
}
```

Add `"context"` to imports if missing. Confirm `NewMemoryStore` exists (it does in `store.go`); if the constructor name differs, use the same pattern as other server tests.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd server && go test ./cmd/server -run 'TestBuildPublicStreamCardViewFillsStepRoleOrgCounts' -count=1`

Expected: FAIL — StepCount/RoleCount/OrganizationCount still 0 (or compile error if builder still assigns `Steps`).

- [ ] **Step 3: Update builder**

In `buildPublicStreamCardView`, replace stepViews construction and return with:

```go
func (s *Server) buildPublicStreamCardView(ctx context.Context, key string, cfg RuntimeConfig, logoURLs map[string]string) (PublicStreamCardView, error) {
	steps := sortedSteps(cfg.Workflow)
	orgs, overflow := publicStreamCardOrganizations(cfg.Organizations, logoURLs)
	orgCount := len(orgs) + overflow
	instanceCount := 0
	activeCount := 0
	allCompleted := false
	if s.store != nil {
		processes, listErr := s.store.ListRecentProcessesByWorkflow(ctx, key, 0)
		if listErr != nil {
			return PublicStreamCardView{}, listErr
		}
		instanceCount = len(processes)
		if instanceCount > 0 {
			for i := range processes {
				processes[i].Progress = normalizeProgressKeys(processes[i].Progress)
				if deriveProcessStatus(cfg.Workflow, &processes[i]) == processStatusActive {
					activeCount++
				}
			}
			allCompleted = activeCount == 0
		}
	}
	return PublicStreamCardView{
		Name:                  cfg.Workflow.Name,
		Description:           strings.TrimSpace(cfg.Workflow.Description),
		PassportEnabled:       cfg.DPP.Enabled,
		InstanceCount:         instanceCount,
		ActiveCount:           activeCount,
		AllCompleted:          allCompleted,
		StepCount:             len(steps),
		RoleCount:             publicStreamCardRoleCount(cfg.Workflow),
		OrganizationCount:     orgCount,
		Organizations:         orgs,
		OrganizationsOverflow: overflow,
	}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd server && go test ./cmd/server -run 'TestBuildPublicStreamCardViewFillsStepRoleOrgCounts' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/public_home.go server/cmd/server/public_stream_card_test.go
git commit -m "$(cat <<'EOF'
feat(public-home): populate stream card step, role, and org counts

Drive the compact card metrics from workflow structure and org totals instead of a step preview list.
EOF
)"
```

---

### Task 3: Template + DPP icon + unit template tests

**Files:**
- Modify: `server/templates/components/public_stream_card.html`
- Modify: `server/templates/icons.html` (add `icon-file-text`)
- Modify: `server/cmd/server/public_stream_card_test.go` (rewrite assertions for new markup)

**Interfaces:**
- Consumes: `PublicStreamCardView` fields from Tasks 1–2; icon templates `icon-layers-2`, `icon-activity`, `icon-check-circle`, `icon-list`, `icon-users-group`, `icon-file-text`

- [ ] **Step 1: Rewrite failing template tests**

Replace obsolete tests that assert Stream badge / Product Passport / steps list / “instances” copy. Target shapes:

1. `TestPublicStreamCardTemplateRendersCoreFields` — expect name + description; **must not** contain `Stream` as badge text, `public-stream-card-badge`, or `<a`.
2. Replace step-preview test with `TestPublicStreamCardTemplateRendersStepsAndRolesMetrics`:

```go
func TestPublicStreamCardTemplateRendersStepsAndRolesMetrics(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	card := PublicStreamCardView{Name: "PV", StepCount: 3, RoleCount: 4}
	if err := tmpl.ExecuteTemplate(&out, "public_stream_card", card); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`class="public-stream-card-metrics-row"`,
		"3 steps",
		"4 roles",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q, got: %s", want, body)
		}
	}
	if strings.Contains(body, "public-stream-card-steps") {
		t.Fatalf("must not render steps list, got: %s", body)
	}
}
```

3. Passport tests → DPP chip:
   - disabled: no `public-stream-card-dpp`, no `>DPP<`
   - enabled: contains `public-stream-card-dpp` and `DPP`
4. Empty metrics: `no runs yet` (not `no instances yet`); exactly one metric in the **runs** row (second row still has steps/roles — count metrics carefully: assert `no runs yet` present and `active now` / `all completed` absent).
5. Settled / active tests: `1 run` / `2 runs` / `N active now` (not `instance(s)`).
6. Orgs footer test: `<strong>Organizations</strong>` plus `OrganizationCount` rendered (e.g. `class="public-stream-card-orgs-count"` containing `5`).

Keep org avatar / overflow / initials tests (they only need `OrganizationCount` if the template requires it when orgs render — set `OrganizationCount: len(orgs)+overflow` in fixtures).

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && go test ./cmd/server -run 'TestPublicStreamCardTemplate' -count=1`

Expected: FAIL on new expected strings / still old markup.

- [ ] **Step 3: Add `icon-file-text`**

Append to `server/templates/icons.html` (Lucide `file-text` paths, same SVG chrome as siblings):

```html
{{ define "icon-file-text" }}
  <svg
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="2"
    stroke-linecap="round"
    stroke-linejoin="round"
    class="icon-svg icon-svg-md"
    aria-hidden="true"
  >
    <path d="M6 22a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h8a2.4 2.4 0 0 1 1.704.706l3.588 3.588A2.4 2.4 0 0 1 20 8v12a2 2 0 0 1-2 2z" />
    <path d="M14 2v5a1 1 0 0 0 1 1h5" />
    <path d="M10 9H8" />
    <path d="M16 13H8" />
    <path d="M16 17H8" />
  </svg>
{{ end }}
```

(If Lucide’s exact path set in this repo’s `@lucide/svelte` differs slightly, match that package’s `FileText` glyph instead — still hand-authored SVG, not a Figma download.)

- [ ] **Step 4: Rewrite `public_stream_card.html`**

Replace the define body with:

```html
{{/* Public homepage stream card: title, description, optional DPP chip, run/step/role metrics, org avatars. Not a link. */}}

{{ define "public_stream_card" }}
  <article class="public-stream-card">
    <div class="public-stream-card-header">
      <h3 class="public-stream-card-title">{{ .Name }}</h3>
      {{ if .Description }}
        <p class="public-stream-card-description">{{ .Description }}</p>
      {{ end }}
      {{ if .PassportEnabled }}
        <span class="public-stream-card-dpp">
          DPP
          {{ template "icon-file-text" . }}
        </span>
      {{ end }}
    </div>
    <div class="public-stream-card-foot">
      <div class="public-stream-card-metrics">
        <div class="public-stream-card-metrics-row">
          {{ if eq .InstanceCount 0 }}
            <span class="public-stream-card-metric">
              {{ template "icon-layers-2" . }}
              no runs yet
            </span>
          {{ else if .AllCompleted }}
            <span class="public-stream-card-metric">
              {{ template "icon-layers-2" . }}
              {{ if eq .InstanceCount 1 }}1 run{{ else }}{{ .InstanceCount }} runs{{ end }}
            </span>
            <span class="public-stream-card-metric">
              {{ template "icon-check-circle" . }}
              all completed
            </span>
          {{ else }}
            <span class="public-stream-card-metric">
              {{ template "icon-layers-2" . }}
              {{ if eq .InstanceCount 1 }}1 run{{ else }}{{ .InstanceCount }} runs{{ end }}
            </span>
            <span class="public-stream-card-metric public-stream-card-metric-active">
              {{ template "icon-activity" . }}
              {{ if eq .ActiveCount 1 }}1 active now{{ else }}{{ .ActiveCount }} active now{{ end }}
            </span>
          {{ end }}
        </div>
        <div class="public-stream-card-metrics-row">
          <span class="public-stream-card-metric">
            {{ template "icon-list" . }}
            {{ if eq .StepCount 1 }}1 step{{ else }}{{ .StepCount }} steps{{ end }}
          </span>
          <span class="public-stream-card-metric">
            {{ template "icon-users-group" . }}
            {{ if eq .RoleCount 1 }}1 role{{ else }}{{ .RoleCount }} roles{{ end }}
          </span>
        </div>
      </div>
      {{ if .Organizations }}
        <hr class="public-stream-card-rule" />
        <div class="public-stream-card-orgs">
          <div class="public-stream-card-orgs-head">
            <strong>Organizations</strong>
            <span class="public-stream-card-orgs-count">{{ .OrganizationCount }}</span>
          </div>
          <div class="public-stream-card-avatars">
            {{ range .Organizations }}
              {{ if .LogoURL }}
                <img src="{{ .LogoURL }}" alt="{{ .Name }}" width="32" height="32" />
              {{ else }}
                <span class="public-stream-card-avatar" role="img" aria-label="{{ .Name }}">{{ .Initials }}</span>
              {{ end }}
            {{ end }}
            {{ if .OrganizationsOverflow }}
              <span
                class="public-stream-card-avatar-more"
                aria-label="{{ .OrganizationsOverflow }} more organizations"
              >+{{ .OrganizationsOverflow }}</span>
            {{ end }}
          </div>
        </div>
      {{ end }}
    </div>
  </article>
{{ end }}
```

- [ ] **Step 5: Run template tests to verify they pass**

Run: `cd server && go test ./cmd/server -run 'TestPublicStreamCardTemplate|TestPublicStreamCardRoleCount|TestBuildPublicStreamCardView|TestOrganizationInitials|TestPublicStreamCardOrganizations' -count=1`

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add server/templates/components/public_stream_card.html server/templates/icons.html server/cmd/server/public_stream_card_test.go
git commit -m "$(cat <<'EOF'
feat(public-home): restyle public stream card markup for Figma metrics

Replace badges and the steps list with a DPP chip, runs/active rows, steps/roles counts, and a plain org total.
EOF
)"
```

---

### Task 4: Component CSS

**Files:**
- Modify: `web/src/styles/components/public-stream-card.css`
- Modify: `web/src/styles/category-palette.css` (remove unused `.public-stream-card-badge` rule)
- Read: `docs/css.md` (layer / component contract)

**Interfaces:**
- Consumes: class names from Task 3 template
- Produces: light-token styles for DPP chip, two metric rows, org count text

- [ ] **Step 1: Update CSS contract comment + rules**

Rewrite the file header contract to:

```css
/*
 * Public stream card — public homepage catalog tile (full component).
 *
 * article.public-stream-card
 *   .public-stream-card-header
 *     h3.public-stream-card-title
 *     p.public-stream-card-description
 *     span.public-stream-card-dpp (+ icon-file-text)
 *   .public-stream-card-foot
 *     .public-stream-card-metrics
 *       .public-stream-card-metrics-row > span.public-stream-card-metric[+icon…]
 *         .public-stream-card-metric-active → activity icon uses --primary
 *     hr.public-stream-card-rule
 *     .public-stream-card-orgs
 *       .public-stream-card-orgs-head > strong + span.public-stream-card-orgs-count
 *       .public-stream-card-avatars > img | span.public-stream-card-avatar | span.public-stream-card-avatar-more
 */
```

Keep existing shell (`.public-stream-card`, title, description, foot, rule, avatars).

**Delete** rules for: `.public-stream-card-badges`, `.public-stream-card-badge`, `.public-stream-card-badge[data-category]`, and the entire `.public-stream-card-steps*` / `.public-stream-card-step*` block.

**Change** `.public-stream-card-metrics` to a vertical stack of rows:

```css
.public-stream-card-metrics {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  width: 100%;
  color: var(--muted-foreground);
  font-size: var(--text-sm);
  line-height: 1.25rem;
}

.public-stream-card-metrics-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-4);
}
```

Keep `.public-stream-card-metric` / icon size / active accent as today (14px icons).

**Add** DPP + org count:

```css
.public-stream-card-dpp {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px var(--space-2);
  border: 1px solid var(--border);
  border-radius: 4px;
  background: var(--muted);
  color: var(--muted-foreground);
  font-size: var(--text-xs);
  line-height: 1rem;
}

.public-stream-card-dpp .icon-svg {
  width: 13px;
  height: 16px;
  flex-shrink: 0;
}

.public-stream-card-orgs-count {
  color: var(--foreground);
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  line-height: 1rem;
}
```

Optional: allow description to wrap 2–3 lines if Figma reads multi-line — prefer keeping current single-line ellipsis unless visual QA shows wrapping is required; default = keep existing description rule.

In `category-palette.css`, remove the `.public-stream-card-badge:not([data-category])` rule (obsolete).

- [ ] **Step 2: Lint CSS if available**

Run: `task css:lint` (or the project’s CSS lint task from Taskfile). If no such task, skip and rely on visual check.

Expected: pass / no new errors on touched files.

- [ ] **Step 3: Commit**

```bash
git add web/src/styles/components/public-stream-card.css web/src/styles/category-palette.css
git commit -m "$(cat <<'EOF'
style(public-home): restyle public stream card for compact Figma layout

Drop badge/steps styles; add DPP chip, metric rows, and plain org count typography on light tokens.
EOF
)"
```

---

### Task 5: Public home handler tests

**Files:**
- Modify: `server/cmd/server/home_handler_test.go` (assertions for badges / passport / instances / steps list)

**Interfaces:**
- Consumes: rendered HTML from `handlePublicHome` using Task 2–3 builders/templates

- [ ] **Step 1: Update failing assertions**

Find and update tests that still expect:

| Old | New |
|-----|-----|
| `class="public-stream-card-badge">Stream</span>` | absent; assert no `public-stream-card-badge` |
| `Product Passport` / `data-category="passport"` | `public-stream-card-dpp` + `DPP` when DPP enabled; absent when not |
| `public-stream-card-steps-head` / step titles list | `N steps` / `N roles` metrics instead |
| `no instances yet` / `N instance(s)` | `no runs yet` / `N run(s)` |

Example passport test body checks:

```go
if strings.Contains(alphaCard, "public-stream-card-dpp") || strings.Contains(alphaCard, ">DPP<") {
	t.Fatalf("plain stream must not show DPP chip, got %q", alphaCard)
}
if !strings.Contains(betaCard, `class="public-stream-card-dpp"`) {
	t.Fatalf("DPP-enabled stream must show DPP chip, got %q", betaCard)
}
if strings.Contains(body, `public-stream-card-badge`) {
	t.Fatalf("public home cards must not render legacy badges, got %q", body)
}
```

- [ ] **Step 2: Run handler tests**

Run: `cd server && go test ./cmd/server -run 'TestHandlePublicHome' -count=1`

Expected: PASS

- [ ] **Step 3: Broader regression on card + public home**

Run: `cd server && go test ./cmd/server -run 'TestPublicStreamCard|TestHandlePublicHome|TestBuildPublicStreamCardView' -count=1`

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add server/cmd/server/home_handler_test.go
git commit -m "$(cat <<'EOF'
test(public-home): align home handler expectations with stream card restyle

Assert DPP chip and runs/steps/roles metrics instead of Stream badges and the steps list.
EOF
)"
```

---

## Self-review

| Spec requirement | Task |
|------------------|------|
| Restyle in place (no v2) | Tasks 3–4 |
| Remove Stream / Product Passport; DPP under description | Tasks 3, 5 |
| Zero runs → `no runs yet` only | Task 3 |
| Distinct substep role count | Tasks 1–2 |
| Org bold number, not chip | Tasks 3–4 |
| No Figma downloads; Lucide icons | Tasks 3–4 |
| Light tokens | Task 4 |
| Drop `Steps` / `PublicStreamCardStepView` | Tasks 1–2 |
| Builder fills Step/Role/Org counts | Task 2 |
| Update unit + home handler tests | Tasks 1–3, 5 |
| Sidebar/HTMX unchanged | no tasks touch those paths |

No placeholders remaining. Names (`publicStreamCardRoleCount`, `OrganizationCount`, `public-stream-card-dpp`, `public-stream-card-metrics-row`, `public-stream-card-orgs-count`) are consistent across tasks.
