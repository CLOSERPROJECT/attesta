# Admin Console HTMX Soft-Nav Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give platform admin and org admin one shared HTMX soft-nav shell (`admin_console`) so sidebar navigation updates page-header + rail without a full reload, while form POSTs stay full-page.

**Architecture:** Extract a full `admin_console` component (header + sidebar + main slot via existing `render` template func). Same panel URLs return the console fragment when `HX-Request` targets `#admin-console`; platform orgs search keeps targeting `#platform-admin-results` and is distinguished via the `HX-Target` header. Org admin drops multi-panel DOM + History JS and renders one panel per route.

**Tech Stack:** Go `html/template`, HTMX (already in layout), existing `isHTMXRequest` / `render` helpers, Vite CSS (reuse `.sidebar-nav` / `.rail-layout` / `.page-header`).

## Global Constraints

- Work only in the git worktree `.worktrees/assess-admin-sidebar-nav` on branch `assess/admin-sidebar-nav` (from current `master`).
- Soft-nav swaps `#admin-console` (page-header including breadcrumbs + title/subtitle + sidebar + main). Layout chrome (topbar) is not swapped.
- Same URLs for full page and soft-nav partial; no new `/content` routes.
- Form POSTs stay full-page redirect/re-render (no HTMX forms in this plan).
- Nested platform orgs search/pagination HTMX must keep targeting `#platform-admin-results`.
- Prefer minimal diffs; match `attesta-ui-components` full-tier tracer (`stream_card`: define stem, DTO in `components.go`, struct literals).
- Read `docs/css.md` before adding CSS; prefer reusing existing classes (no new stylesheet unless required).
- TDD: failing test → implement → pass → commit per task.

---

## File map

| File | Role |
|------|------|
| `server/cmd/server/components.go` | Add `AdminConsoleNavItem`, `AdminConsoleView` |
| `server/templates/components/admin_console.html` | Shared shell template (`admin_console`) |
| `server/cmd/server/admin_console.go` | Nav builders + `hxTarget` helper + render helpers (optional peel from `main.go`) |
| `server/cmd/server/admin_console_test.go` | Unit tests for builders / HX-Target branching helper |
| `server/cmd/server/admin_console_template_test.go` | Template markup contract for `admin_console` |
| `server/templates/pages/platform_admin.html` | Embed `admin_console`; main defines for orgs/categories |
| `server/templates/pages/org_admin.html` | Embed `admin_console`; one panel; remove panel-switcher JS |
| `server/cmd/server/main.go` | HTMX branch for platform/org GETs; wire console views |
| Existing `*_test.go` files listed in tasks | Update markers / add HTMX handler cases |

---

### Task 1: `AdminConsoleView` DTO + `admin_console` template

**Files:**
- Create: `server/templates/components/admin_console.html`
- Create: `server/cmd/server/admin_console_template_test.go`
- Modify: `server/cmd/server/components.go` (append after `BreadcrumbsView`)

**Interfaces:**
- Consumes: existing `BreadcrumbsView`, template func `render(name string, data any) (template.HTML, error)` from `withTemplateFuncs`
- Produces:
  - `type AdminConsoleNavItem struct { Href, Title, Copy string; Active bool }`
  - `type AdminConsoleView struct { ID, NavLabel, Title, Subtitle string; Breadcrumbs BreadcrumbsView; NavItems []AdminConsoleNavItem; MainTemplate string; MainData any }`
  - Template define name `"admin_console"`

- [ ] **Step 1: Write the failing template test**

Create `server/cmd/server/admin_console_template_test.go`:

```go
package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestAdminConsoleTemplateSoftNavContract(t *testing.T) {
	tmpl := parseTestTemplates(t)

	mainCrumbs := BreadcrumbsView{Items: []BreadcrumbItem{
		{Label: "MainSlot", Href: "/main-slot", Current: true},
	}}
	view := AdminConsoleView{
		ID:       "admin-console",
		NavLabel: "Test sections",
		Title:    "Test dashboard",
		Subtitle: "Test subtitle",
		Breadcrumbs: BreadcrumbsView{Items: []BreadcrumbItem{
			{Label: "Dashboard", Href: "/my"},
			{Label: "Test", Href: "/test", Current: true},
		}},
		NavItems: []AdminConsoleNavItem{
			{Href: "/test/a", Title: "Alpha", Copy: "First", Active: true},
			{Href: "/test/b", Title: "Beta", Copy: "Second", Active: false},
		},
		// Reuse existing "breadcrumbs" define as the main slot via render().
		MainTemplate: "breadcrumbs",
		MainData:     mainCrumbs,
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "admin_console", view); err != nil {
		t.Fatalf("render admin_console: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`id="admin-console"`,
		`class="page-header"`,
		`class="breadcrumbs"`,
		"<h1>Test dashboard</h1>",
		"Test subtitle",
		`aria-label="Test sections"`,
		`class="sidebar-nav"`,
		`hx-get="/test/a"`,
		`hx-get="/test/b"`,
		`hx-target="#admin-console"`,
		`hx-select="#admin-console"`,
		`hx-swap="outerHTML"`,
		`hx-push-url="true"`,
		`class="sidebar-nav-link is-active"`,
		`aria-current="page"`,
		"MainSlot",
		`href="/main-slot"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in admin_console, got:\n%s", want, body)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run (from worktree `server/`):

```bash
cd /Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/.worktrees/assess-admin-sidebar-nav/server
go test ./cmd/server/ -count=1 -run 'TestAdminConsoleTemplateSoftNavContract' -v
```

Expected: FAIL (undefined `AdminConsoleView` and/or missing template `admin_console`).

- [ ] **Step 3: Add DTOs to `components.go`**

Append after `BreadcrumbsView`:

```go
// AdminConsoleNavItem is one soft-nav link in templates/components/admin_console.html.
type AdminConsoleNavItem struct {
	Href   string
	Title  string
	Copy   string
	Active bool
}

// AdminConsoleView is the view model for templates/components/admin_console.html.
// MainTemplate is executed via the render func with MainData as the dot.
type AdminConsoleView struct {
	ID           string // default "admin-console" when empty
	NavLabel     string
	Title        string
	Subtitle     string
	Breadcrumbs  BreadcrumbsView
	NavItems     []AdminConsoleNavItem
	MainTemplate string
	MainData     any
}
```

- [ ] **Step 4: Create `admin_console.html`**

Create `server/templates/components/admin_console.html`:

```html
{{/* Shared admin soft-nav shell: page-header + sidebar + main (HTMX swap root). */}}

{{ define "admin_console" }}
  <div
    id="{{ if .ID }}{{ .ID }}{{ else }}admin-console{{ end }}"
    class="admin-console"
  >
    <section class="page-header">
      {{ template "breadcrumbs" .Breadcrumbs }}
      <div class="page-header-body">
        <h1>{{ .Title }}</h1>
        {{ if .Subtitle }}
          <p>{{ .Subtitle }}</p>
        {{ end }}
      </div>
    </section>

    <div class="rail-layout rail-layout-ready">
      <section class="panel panel-sticky">
        <nav
          class="sidebar-nav"
          aria-label="{{ .NavLabel }}"
        >
          {{ range .NavItems }}
            <a
              href="{{ .Href }}"
              class="sidebar-nav-link{{ if .Active }} is-active{{ end }}"
              hx-get="{{ .Href }}"
              hx-target="#{{ if $.ID }}{{ $.ID }}{{ else }}admin-console{{ end }}"
              hx-select="#{{ if $.ID }}{{ $.ID }}{{ else }}admin-console{{ end }}"
              hx-swap="outerHTML"
              hx-push-url="true"
              {{ if .Active }}aria-current="page"{{ end }}
            >
              <span class="sidebar-nav-title">{{ .Title }}</span>
              {{ if .Copy }}
                <span class="sidebar-nav-copy">{{ .Copy }}</span>
              {{ end }}
            </a>
          {{ end }}
        </nav>
      </section>

      <section class="panel rail-layout-main">
        {{ render .MainTemplate .MainData }}
      </section>
    </div>
  </div>
{{ end }}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
go test ./cmd/server/ -count=1 -run 'TestAdminConsoleTemplateSoftNavContract' -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
cd /Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/.worktrees/assess-admin-sidebar-nav
git add server/cmd/server/components.go \
  server/templates/components/admin_console.html \
  server/cmd/server/admin_console_template_test.go
git commit -m "$(cat <<'EOF'
feat(ui): add admin_console soft-nav shell component

EOF
)"
```

---

### Task 2: HX-Target helper + platform/org console builders

**Files:**
- Create: `server/cmd/server/admin_console.go`
- Create: `server/cmd/server/admin_console_test.go`

**Interfaces:**
- Consumes: `AdminConsoleView`, `AdminConsoleNavItem`, `organizationPath`, `buildOrgAdminBreadcrumbs`, `buildPlatformAdminBreadcrumbs`
- Produces:
  - `func htmxTargetID(r *http.Request) string`
  - `func wantsAdminConsolePartial(r *http.Request) bool` — true when `isHTMXRequest(r)` and target is empty or `admin-console` (not `platform-admin-results`)
  - `func platformAdminConsole(view PlatformAdminView) AdminConsoleView`
  - `func orgAdminConsole(view OrgAdminView) AdminConsoleView`

- [ ] **Step 1: Write failing tests**

Create `server/cmd/server/admin_console_test.go`:

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWantsAdminConsolePartial(t *testing.T) {
	t.Parallel()

	full := httptest.NewRequest(http.MethodGet, "/admin/orgs", nil)
	if wantsAdminConsolePartial(full) {
		t.Fatal("non-HTMX must be false")
	}

	results := httptest.NewRequest(http.MethodGet, "/admin/orgs?q=a", nil)
	results.Header.Set("HX-Request", "true")
	results.Header.Set("HX-Target", "platform-admin-results")
	if wantsAdminConsolePartial(results) {
		t.Fatal("orgs search HTMX must stay on results partial")
	}

	console := httptest.NewRequest(http.MethodGet, "/admin/categories", nil)
	console.Header.Set("HX-Request", "true")
	console.Header.Set("HX-Target", "admin-console")
	if !wantsAdminConsolePartial(console) {
		t.Fatal("sidebar soft-nav must request admin_console partial")
	}
}

func TestPlatformAdminConsoleNav(t *testing.T) {
	t.Parallel()
	view := PlatformAdminView{ActivePanel: "categories", Breadcrumbs: buildPlatformAdminBreadcrumbs("categories")}
	c := platformAdminConsole(view)
	if c.MainTemplate != "platform_admin_main" {
		t.Fatalf("MainTemplate = %q", c.MainTemplate)
	}
	if len(c.NavItems) != 2 || !c.NavItems[1].Active || c.NavItems[1].Href != "/admin/categories" {
		t.Fatalf("unexpected nav: %+v", c.NavItems)
	}
	if c.Subtitle != "Browse stream discovery categories" {
		t.Fatalf("Subtitle = %q", c.Subtitle)
	}
}

func TestOrgAdminConsoleNav(t *testing.T) {
	t.Parallel()
	view := OrgAdminView{ActivePanel: "roles", Breadcrumbs: buildOrgAdminBreadcrumbs("roles")}
	c := orgAdminConsole(view)
	if c.MainTemplate != "org_admin_main" {
		t.Fatalf("MainTemplate = %q", c.MainTemplate)
	}
	if len(c.NavItems) != 3 || !c.NavItems[1].Active || c.NavItems[1].Href != organizationPath("roles") {
		t.Fatalf("unexpected nav: %+v", c.NavItems)
	}
}
```

- [ ] **Step 2: Run tests — expect FAIL**

```bash
go test ./cmd/server/ -count=1 -run 'TestWantsAdminConsolePartial|TestPlatformAdminConsoleNav|TestOrgAdminConsoleNav' -v
```

Expected: FAIL (undefined funcs).

- [ ] **Step 3: Implement `admin_console.go`**

```go
package main

import (
	"net/http"
	"strings"
)

func htmxTargetID(r *http.Request) string {
	if r == nil {
		return ""
	}
	return strings.TrimSpace(r.Header.Get("HX-Target"))
}

func wantsAdminConsolePartial(r *http.Request) bool {
	if !isHTMXRequest(r) {
		return false
	}
	target := htmxTargetID(r)
	return target == "" || target == "admin-console"
}

func platformAdminConsole(view PlatformAdminView) AdminConsoleView {
	active := strings.TrimSpace(view.ActivePanel)
	subtitle := "Create and manage organizations"
	if active == "categories" {
		subtitle = "Browse stream discovery categories"
	}
	return AdminConsoleView{
		ID:          "admin-console",
		NavLabel:    "Platform admin sections",
		Title:       "Platform admin dashboard",
		Subtitle:    subtitle,
		Breadcrumbs: view.Breadcrumbs,
		NavItems: []AdminConsoleNavItem{
			{
				Href:   "/admin/orgs",
				Title:  "Organizations",
				Copy:   "Create and manage organizations",
				Active: active != "categories",
			},
			{
				Href:   "/admin/categories",
				Title:  "Categories",
				Copy:   "Browse stream discovery categories",
				Active: active == "categories",
			},
		},
		MainTemplate: "platform_admin_main",
		MainData:     view,
	}
}

func orgAdminConsole(view OrgAdminView) AdminConsoleView {
	active := strings.TrimSpace(view.ActivePanel)
	if active == "" {
		active = "profile"
	}
	return AdminConsoleView{
		ID:          "admin-console",
		NavLabel:    "Organization admin sections",
		Title:       "Organization admin dashboard",
		Subtitle:    "Manage organization settings, roles, and members",
		Breadcrumbs: view.Breadcrumbs,
		NavItems: []AdminConsoleNavItem{
			{
				Href:   organizationPath("profile"),
				Title:  "Organization profile",
				Copy:   "Update your organization name and logo",
				Active: active == "profile",
			},
			{
				Href:   organizationPath("roles"),
				Title:  "Roles",
				Copy:   "Manage the role catalog for your organization",
				Active: active == "roles",
			},
			{
				Href:   organizationPath("members"),
				Title:  "Members",
				Copy:   "Invite people and update member access",
				Active: active == "members",
			},
		},
		MainTemplate: "org_admin_main",
		MainData:     view,
	}
}
```

- [ ] **Step 4: Run tests — expect PASS**

```bash
go test ./cmd/server/ -count=1 -run 'TestWantsAdminConsolePartial|TestPlatformAdminConsoleNav|TestOrgAdminConsoleNav' -v
```

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/admin_console.go server/cmd/server/admin_console_test.go
git commit -m "$(cat <<'EOF'
feat(admin): add console builders and HX-Target partial gating

EOF
)"
```

---

### Task 3: Wire platform admin templates + HTMX GET branching

**Files:**
- Modify: `server/templates/pages/platform_admin.html`
- Modify: `server/cmd/server/main.go` (`renderPlatformAdmin`, `handleAdminCategories`, `handleAdminOrgs` GET)
- Modify: `server/cmd/server/platform_admin_template_test.go`
- Modify: `server/cmd/server/panel_markup_test.go` (`TestPlatformAdminPanelMarkup`)
- Create or modify: `server/cmd/server/admin_categories_handler_test.go` / new `platform_admin_htmx_test.go`

**Interfaces:**
- Consumes: `platformAdminConsole`, `wantsAdminConsolePartial`, template `"admin_console"`, `"platform_admin_main"`
- Produces: platform body embeds console; HTMX sidebar returns console fragment; search HTMX still returns `platform_admin_results`

- [ ] **Step 1: Write failing HTMX handler test**

Add to new file `server/cmd/server/platform_admin_htmx_test.go` (reuse session/admin setup from `admin_categories_handler_test.go`):

```go
func TestHandleAdminCategoriesHTMXReturnsAdminConsole(t *testing.T) {
	// arrange platform admin server + taxonomy seed (copy pattern from TestHandleAdminCategories*)
	req := httptest.NewRequest(http.MethodGet, "/admin/categories", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "admin-console")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "admin-session"})
	rec := httptest.NewRecorder()
	server.handleAdminCategories(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `id="admin-console"`) {
		t.Fatalf("expected admin-console fragment, got: %s", body)
	}
	if strings.Contains(body, `class="topbar"`) || strings.Contains(body, "<html") {
		t.Fatalf("HTMX soft-nav must not include layout, got: %s", body)
	}
	if !strings.Contains(body, "Categories") || !strings.Contains(body, "hx-get=\"/admin/orgs\"") {
		t.Fatalf("expected categories console nav, got: %s", body)
	}
}

func TestHandleAdminOrgsHTMXSearchStillReturnsResults(t *testing.T) {
	// GET /admin/orgs with HX-Request + HX-Target: platform-admin-results
	// expect id="platform-admin-results" and NOT require full admin-console wrapper swap root as sole content
	// (results partial may appear without page-header)
}
```

Fill arrange blocks by copying the working platform-admin server setup from existing tests in `admin_categories_handler_test.go` / `admin_handler_identity_test.go` (platform admin session cookie + `ADMIN_EMAIL`/`ADMIN_PASSWORD` env).

- [ ] **Step 2: Run — expect FAIL**

```bash
go test ./cmd/server/ -count=1 -run 'TestHandleAdminCategoriesHTMXReturnsAdminConsole|TestHandleAdminOrgsHTMXSearchStillReturnsResults' -v
```

- [ ] **Step 3: Refactor `platform_admin.html`**

Replace the duplicated page-header + sidebar block in `platform_admin_body` with:

```html
{{ define "platform_admin_body" }}
  <div class="stack u-max-w-7xl u-mx-auto">
    {{ template "admin_console" (platformAdminConsoleDot .) }}
  </div>
  {{/* keep dialogs that today live outside main if any — move dialogs into platform_admin_main or keep after console if they use document.getElementById */}}
{{ end }}
```

Go templates cannot call `platformAdminConsole` unless it is a template func. **Do not add a template func.** Instead build console in the handler / view:

Option A (preferred): add field `Console AdminConsoleView` on `PlatformAdminView`, set it in `platformAdminView` / categories handler before render.

```go
view.Console = platformAdminConsole(view)
```

Template:

```html
{{ define "platform_admin_body" }}
  <div class="stack u-max-w-7xl u-mx-auto">
    {{ template "admin_console" .Console }}
  </div>
{{ end }}

{{ define "platform_admin_main" }}
  {{ if eq .ActivePanel "categories" }}
    {{ template "platform_admin_categories_panel" . }}
  {{ else }}
    {{/* existing orgs main markup: head-actions, search, results — NOT the outer header/sidebar */}}
    ...
  {{ end }}
{{ end }}
```

Ensure create/edit/invite `<dialog>` elements remain in the DOM for orgs (inside `platform_admin_main` or after console inside the stack — keep current click handlers working).

- [ ] **Step 4: Update render paths in `main.go`**

In `platformAdminView` / categories view construction, set `view.Console = platformAdminConsole(view)` (Console.MainData must be the `PlatformAdminView` value **without** infinite nest — set MainData to a copy or set Console after filling other fields:

```go
view := PlatformAdminView{ ... }
view.Console = platformAdminConsole(view)
// platformAdminConsole sets MainData: view — at that moment Console is zero; OK because main template only needs ActivePanel/Orgs/Categories/etc.
```

`renderPlatformAdmin`:

```go
func (s *Server) renderPlatformAdmin(w http.ResponseWriter, r *http.Request, user *AccountUser, confirmation string, errs PlatformAdminErrors) {
	view := s.platformAdminView(user, confirmation, errs)
	view.Console = platformAdminConsole(view)
	if wantsAdminConsolePartial(r) {
		if err := s.tmpl.ExecuteTemplate(w, "admin_console", view.Console); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	if err := s.tmpl.ExecuteTemplate(w, "platform_admin.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
```

Update all `renderPlatformAdmin` call sites to pass `r` (or keep a wrapper). For `handleAdminOrgs` GET:

```go
if isHTMXRequest(r) && htmxTargetID(r) == "platform-admin-results" {
	s.renderPlatformAdminResults(...)
	return
}
if wantsAdminConsolePartial(r) {
	s.renderPlatformAdmin(w, r, ...)
	return
}
s.renderPlatformAdmin(w, r, ...)
```

(Same function handles both full and console partial via `wantsAdminConsolePartial`.)

`handleAdminCategories`: build view, set Console, then if `wantsAdminConsolePartial(r)` execute `admin_console`, else `platform_admin.html`.

- [ ] **Step 5: Update template tests**

- `TestPlatformAdminCategoriesPanelMarkup` / `TestPlatformAdminPanelMarkup`: expect `id="admin-console"`, `hx-get="/admin/categories"`, `hx-target="#admin-console"`; keep categories/orgs content assertions.
- Remove expectations that conflict with moved markup.

- [ ] **Step 6: Run focused tests**

```bash
go test ./cmd/server/ -count=1 -run 'TestAdminConsole|TestPlatformAdmin|TestHandleAdminCategories|TestHandleAdminOrgsHTMX|TestPanelMarkup' -v
```

Expected: PASS (fix any remaining marker failures).

- [ ] **Step 7: Commit**

```bash
git add server/templates/pages/platform_admin.html server/cmd/server/main.go \
  server/cmd/server/platform_admin_template_test.go server/cmd/server/panel_markup_test.go \
  server/cmd/server/platform_admin_htmx_test.go
git commit -m "$(cat <<'EOF'
feat(admin): platform admin soft-nav via shared admin_console

EOF
)"
```

---

### Task 4: Rewrite org admin to single-panel + soft-nav

**Files:**
- Modify: `server/templates/pages/org_admin.html`
- Modify: `server/cmd/server/main.go` (`renderOrgAdminWithErrors` signature to accept `r`, HTMX branch)
- Modify: `server/cmd/server/org_admin_template_sidebar_test.go`
- Modify: `server/cmd/server/org_admin_section_urls_test.go`
- Modify: `server/cmd/server/panel_markup_test.go` (`TestOrgAdminRolesPanelMarkup`)
- Modify: other org admin template tests if they assume all three panel IDs

**Interfaces:**
- Consumes: `orgAdminConsole`, `wantsAdminConsolePartial`
- Produces: one panel HTML per `ActivePanel`; no `data-org-admin-*` panel switcher; soft-nav HTMX on console

- [ ] **Step 1: Update sidebar test to new contract (fails on old markup)**

Rewrite `TestOrgAdminTemplateRendersSidebarPanels` expectations:

```go
// ActivePanel profile only
view.ActivePanel = "profile"
// expect:
// id="admin-console", hx-get organizationPath links, org profile form
// must NOT contain: data-org-admin-nav, data-org-admin-shell, id="org-admin-panel-roles", history.pushState
// must NOT contain members invite list markers if those only lived in members panel
```

Add sibling tests or table cases for `roles` and `members` ActivePanel ensuring only that panel’s main content appears.

- [ ] **Step 2: Run — expect FAIL**

```bash
go test ./cmd/server/ -count=1 -run 'TestOrgAdminTemplateRendersSidebarPanels' -v
```

- [ ] **Step 3: Restructure `org_admin.html`**

1. `NeedsOrganizationSetup`: keep existing setup form UI (no `admin_console` soft-nav).
2. Otherwise:

```html
{{ define "org_admin_body" }}
  <div class="stack u-max-w-7xl u-mx-auto">
    {{ if .NeedsOrganizationSetup }}
      {{/* existing setup header + form */}}
    {{ else }}
      {{ template "admin_console" .Console }}
    {{ end }}
  </div>
  {{/* keep role/member dialogs required by the active panel; drop inactive panel dialogs if unused */}}
{{ end }}

{{ define "org_admin_main" }}
  {{ if eq .ActivePanel "roles" }}
    {{ template "org_admin_roles_panel" . }}
  {{ else if eq .ActivePanel "members" }}
    {{ template "org_admin_members_panel" . }}
  {{ else }}
    {{ template "org_admin_profile_panel" . }}
  {{ end }}
{{ end }}
```

Split existing panel sections into `org_admin_profile_panel` / `org_admin_roles_panel` / `org_admin_members_panel` defines (move markup out of the triple-`hidden` sections).

3. Delete the panel-switcher IIFE (`data-org-admin-shell`, `history.pushState`, `preventDefault` on nav). **Keep** role-palette picker / invite-copy / other scripts that do not depend on multi-panel shell — if they are intertwined, split carefully so palette pickers still work on roles/members panels.

- [ ] **Step 4: Wire `Console` + HTMX in `renderOrgAdminWithErrors`**

```go
view.Console = orgAdminConsole(view)
if !view.NeedsOrganizationSetup && wantsAdminConsolePartial(r) {
	_ = s.tmpl.ExecuteTemplate(w, "admin_console", view.Console)
	return
}
_ = s.tmpl.ExecuteTemplate(w, "org_admin.html", view)
```

Thread `*http.Request` into `renderOrgAdminWithErrors` (update all call sites).

- [ ] **Step 5: Fix section URL / panel markup tests**

- `TestOrgAdminSectionURLHandlers`: stop asserting `data-org-admin-default-panel` and sibling panel IDs; assert active panel content + `aria-current` / `is-active` on the matching nav href; assert other panels’ unique strings absent.
- `TestOrgAdminRolesPanelMarkup`: set `ActivePanel: "roles"`; assert roles markup without requiring `org-admin-panel-members` boundary.

- [ ] **Step 6: Run org-admin + related tests**

```bash
go test ./cmd/server/ -count=1 -run 'TestOrgAdmin|TestAdminConsole|TestListRow|TestDialogMarkup' -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add server/templates/pages/org_admin.html server/cmd/server/main.go \
  server/cmd/server/org_admin_*.go server/cmd/server/panel_markup_test.go \
  server/cmd/server/list_row_markup_test.go server/cmd/server/dialog_markup_test.go
git commit -m "$(cat <<'EOF'
feat(admin): org admin single-panel soft-nav via admin_console

EOF
)"
```

---

### Task 5: Regression sweep + manual checklist

**Files:**
- Test only (fix stragglers)

- [ ] **Step 1: Run broader test suite**

```bash
cd /Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/.worktrees/assess-admin-sidebar-nav/server
go test ./cmd/server/ -count=1 -timeout 120s
```

Expected: PASS. Fix any leftover assertions referencing `data-org-admin-*` or missing `hx-` attrs.

- [ ] **Step 2: Manual smoke (if `task dev` available in worktree)**

From worktree:

```bash
task dev
```

Open printed URL:

1. Platform admin: click Organizations ↔ Categories — no full reload (Network: document not refetched; `HX-Request` fragment); breadcrumbs update; topbar persists.
2. Platform admin: search orgs — still updates results list only.
3. Org admin: Profile ↔ Roles ↔ Members — soft-nav; only one panel’s content; forms POST still full reload.
4. Org admin without org: setup form still works.
5. Disable JS: sidebar links still navigate full page.

- [ ] **Step 3: Commit any test fixes** (if needed)

```bash
git commit -m "$(cat <<'EOF'
test(admin): finish soft-nav regression fixes

EOF
)"
```

---

## Spec coverage (self-review)

| Decision | Task |
|----------|------|
| Shared full `admin_console` component | Task 1 |
| Swap page-header + nav + main | Task 1 template |
| Same URL + `HX-Request` partial | Tasks 3–4 |
| Distinguish orgs search via `HX-Target` | Task 2–3 |
| POSTs stay full-page | Tasks 3–4 (no form hx attrs) |
| Org admin single panel, remove History JS | Task 4 |
| Org setup without soft-nav | Task 4 |
| Nested results HTMX preserved | Task 3 |
| Tests for fragment vs layout | Tasks 1, 3, 4, 5 |

## Placeholder scan

No TBD/TODO left in tasks; handler arrange blocks point at concrete existing test files to copy.

## Type consistency

- `AdminConsoleView` / `AdminConsoleNavItem` names stable across tasks
- `wantsAdminConsolePartial` / `htmxTargetID` / `platformAdminConsole` / `orgAdminConsole` names stable
- Main template names: `platform_admin_main`, `org_admin_main`
- DOM id: `admin-console`; results id unchanged: `platform-admin-results`
