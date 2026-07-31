# Platform Admin Routing Rewire Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `/admin` a working platform-admin entry (redirect to organizations), hard-cut rename `/admin/orgs` → `/admin/organizations`, fix breadcrumbs to the org-admin three-crumb shape, and relabel the topbar link to **Admin**.

**Architecture:** Keep today’s flat mux registration. Add `adminPath` (mirror `organizationPath`), a root `/admin` handler that gates then redirects, rename the orgs path prefix everywhere, and make `buildPlatformAdminBreadcrumbs` always emit Dashboard → Platform admin → section.

**Tech Stack:** Go `net/http`, `html/template`, HTMX soft-nav (`admin_console`), existing `requirePlatformAdmin` / Cerbos gate, `go test`, Goa design (`task goa:generate`).

**Spec:** `docs/superpowers/specs/2026-07-31-platform-admin-routing-design.md`

**Worktree:** `.worktrees/feat/platform-admin-routing` on branch `feat/platform-admin-routing`

## Global Constraints

- `/admin` and `/admin/` → auth-gated redirect to `/admin/organizations` (default section)
- Hard cut: `/admin/orgs` and `/admin/orgs/…` return **404** (no compatibility redirect)
- Categories stay at `/admin/categories` (+ nested subcategory POSTs)
- Breadcrumbs always: `Dashboard` → `/my`, `Platform admin` → `/admin`, current section (`Organizations` | `Categories`)
- Topbar: label **Admin**, href `/admin`
- ActivePanel keys: `organizations` | `categories` (replace `orgs`)
- Add `adminPath(rest string)`; use it for constructed admin URLs (nav, redirects, logo, pagination helpers)
- Do **not** register a subtree pattern `/admin/` that would shadow `/admin/organizations` or `/admin/categories`
- No new landing page body; no last-visited memory; no unified `handleAdminRoutes` mux refactor
- Prefer minimal, localized diffs; TDD per task; commit after each task

## File map

| File | Responsibility |
|------|----------------|
| `server/cmd/server/paths.go` | Add `adminPath` |
| `server/cmd/server/paths_test.go` | `adminPath` cases |
| `server/cmd/server/breadcrumbs.go` | Three-crumb platform admin trails + section helpers |
| `server/cmd/server/breadcrumbs_test.go` | Update platform admin breadcrumb expectations |
| `server/cmd/server/admin_root.go` (new) or `main.go` | `handleAdminRoot` redirect |
| `server/cmd/server/admin_root_test.go` (new) | Redirect + auth + hard-cut 404 cases |
| `server/cmd/server/main.go` | Mux registration; path prefix `/admin/organizations`; `ActivePanel`; `platformAdminPath`; `ShowAdminLink` (rename from `ShowOrgsLink`); logo path trim |
| `server/cmd/server/admin_console.go` | Nav hrefs via `adminPath`; ActivePanel `organizations` |
| `server/cmd/server/admin_console_test.go` | Update expected hrefs / active panel |
| `server/templates/layout.html` | Topbar Admin link |
| `server/templates/pages/platform_admin.html` | Replace `/admin/orgs` with `/admin/organizations` |
| `server/cmd/server/*_test.go` | Sweep `/admin/orgs` → `/admin/organizations`; topbar assertions |
| `server/design/design.go` + Goa gen | OpenAPI paths |
| `AGENTS.md` | Document new routes / topbar label |

---

### Task 1: `adminPath` helper

**Files:**
- Modify: `server/cmd/server/paths.go`
- Modify: `server/cmd/server/paths_test.go`

**Interfaces:**
- Consumes: `organizationPath` pattern in `paths.go`
- Produces: `func adminPath(rest string) string` — empty rest → `/admin`; otherwise `/admin/` + trimmed rest (supports `organizations`, `/categories`, `organizations/logo/x`)

- [ ] **Step 1: Write the failing test**

Append to `server/cmd/server/paths_test.go`:

```go
func TestAdminPath(t *testing.T) {
	cases := []struct {
		rest string
		want string
	}{
		{"", "/admin"},
		{"organizations", "/admin/organizations"},
		{"/categories", "/admin/categories"},
		{"organizations/logo/logo-1", "/admin/organizations/logo/logo-1"},
		{"  organizations  ", "/admin/organizations"},
	}
	for _, tc := range cases {
		if got := adminPath(tc.rest); got != tc.want {
			t.Fatalf("adminPath(%q) = %q, want %q", tc.rest, got, tc.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd server && go test ./cmd/server -run TestAdminPath -count=1`

Expected: FAIL — `adminPath` undefined

- [ ] **Step 3: Implement `adminPath`**

Append to `server/cmd/server/paths.go` (after `organizationPath`):

```go
// adminPath joins /admin with rest.
// rest may be "organizations", "/categories", or "organizations/logo/{id}".
func adminPath(rest string) string {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "/admin"
	}
	rest = strings.TrimPrefix(rest, "/")
	return "/admin/" + rest
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd server && go test ./cmd/server -run TestAdminPath -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/paths.go server/cmd/server/paths_test.go
git commit -m "$(cat <<'EOF'
feat(admin): add adminPath URL helper

EOF
)"
```

---

### Task 2: Platform admin breadcrumbs (three crumbs)

**Files:**
- Modify: `server/cmd/server/breadcrumbs.go`
- Modify: `server/cmd/server/breadcrumbs_test.go`

**Interfaces:**
- Consumes: `adminPath`, `appHomePath`
- Produces: `buildPlatformAdminBreadcrumbs(activePanel string)` always returns 3 items; helpers `platformAdminSectionLabel` / `platformAdminSectionHref` (or inline switch) for `organizations` (default) and `categories`

- [ ] **Step 1: Rewrite failing breadcrumb tests**

Replace `TestBuildPlatformAdminBreadcrumbsOrgsPanel` and `TestBuildPlatformAdminBreadcrumbsCategoriesPanel` in `breadcrumbs_test.go` with:

```go
func TestBuildPlatformAdminBreadcrumbs(t *testing.T) {
	cases := map[string]struct {
		label string
		href  string
	}{
		"organizations": {label: "Organizations", href: "/admin/organizations"},
		"":              {label: "Organizations", href: "/admin/organizations"},
		"other":         {label: "Organizations", href: "/admin/organizations"},
		"categories":    {label: "Categories", href: "/admin/categories"},
	}
	for panel, want := range cases {
		got := buildPlatformAdminBreadcrumbs(panel)
		if len(got.Items) != 3 {
			t.Fatalf("panel %q: len(Items) = %d, want 3", panel, len(got.Items))
		}
		if got.Items[0].Label != "Dashboard" || got.Items[0].Href != appHomePath {
			t.Fatalf("panel %q: root = %+v", panel, got.Items[0])
		}
		if got.Items[1].Label != "Platform admin" || got.Items[1].Href != adminPath("") || got.Items[1].Current {
			t.Fatalf("panel %q: middle = %+v", panel, got.Items[1])
		}
		if got.Items[2].Label != want.label || got.Items[2].Href != want.href || !got.Items[2].Current {
			t.Fatalf("panel %q: section = %+v, want label=%q href=%q", panel, got.Items[2], want.label, want.href)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && go test ./cmd/server -run 'TestBuildPlatformAdminBreadcrumbs' -count=1`

Expected: FAIL — orgs panel still returns 2 crumbs / wrong hrefs

- [ ] **Step 3: Implement breadcrumbs**

Replace `buildPlatformAdminBreadcrumbs` in `breadcrumbs.go`:

```go
func buildPlatformAdminBreadcrumbs(activePanel string) BreadcrumbsView {
	section := strings.TrimSpace(activePanel)
	return BreadcrumbsView{Items: []BreadcrumbItem{
		{Label: "Dashboard", Href: appHomePath},
		{Label: "Platform admin", Href: adminPath("")},
		{Label: platformAdminSectionLabel(section), Href: platformAdminSectionHref(section), Current: true},
	}}
}

func platformAdminSectionLabel(activePanel string) string {
	switch strings.TrimSpace(activePanel) {
	case "categories":
		return "Categories"
	default:
		return "Organizations"
	}
}

func platformAdminSectionHref(activePanel string) string {
	switch strings.TrimSpace(activePanel) {
	case "categories":
		return adminPath("categories")
	default:
		return adminPath("organizations")
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd server && go test ./cmd/server -run 'TestBuildPlatformAdminBreadcrumbs' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/breadcrumbs.go server/cmd/server/breadcrumbs_test.go
git commit -m "$(cat <<'EOF'
fix(admin): align platform admin breadcrumbs with org-admin shape

EOF
)"
```

---

### Task 3: `/admin` root redirect + mux (no subtree shadow)

**Files:**
- Create: `server/cmd/server/admin_root.go`
- Create: `server/cmd/server/admin_root_test.go`
- Modify: `server/cmd/server/main.go` (`newMux` registrations only in this task — add `/admin`; do **not** rename orgs yet)

**Interfaces:**
- Consumes: `requirePlatformAdmin`, `adminPath`
- Produces: `func (s *Server) handleAdminRoot(w http.ResponseWriter, r *http.Request)` — GET only; path must be exactly `/admin` (ServeMux may redirect `/admin/` → `/admin`); then `http.Redirect` 303/302 to `adminPath("organizations")`

**Critical:** Register `mux.HandleFunc("/admin", s.handleAdminRoot)` only. Do **not** register `/admin/` as a trailing-slash subtree pattern (it would steal `/admin/organizations` and `/admin/categories`). Rely on Go 1.22+ ServeMux trailing-slash redirect between `/admin` and `/admin/`, or assert `/admin/` behavior in the test and adjust with an exact `{$}` pattern only if needed.

- [ ] **Step 1: Write failing handler tests**

Create `server/cmd/server/admin_root_test.go` (same harness as `TestHandleAdminOrgsPlatformAdminHTMXGetAndLogo`):

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandleAdminRootRedirectsToOrganizations(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()
	server := &Server{
		authorizer:  fakeAuthorizer{},
		identity:    &fakeIdentityStore{},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()
	server.handleAdminRoot(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); loc != adminPath("organizations") {
		t.Fatalf("Location = %q, want %q", loc, adminPath("organizations"))
	}
}

func TestHandleAdminRootRejectsUnauthenticated(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	server := &Server{
		authorizer:  fakeAuthorizer{},
		identity:    &fakeIdentityStore{},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return time.Now().UTC() },
	}

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	server.handleAdminRoot(rec, req)

	if rec.Code == http.StatusSeeOther {
		t.Fatalf("must not redirect unauthenticated user, status=%d location=%q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestMuxAdminRootRedirects(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()
	server := &Server{
		authorizer:  fakeAuthorizer{},
		identity:    &fakeIdentityStore{},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()
	server.newMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); loc != "/admin/organizations" {
		t.Fatalf("Location = %q, want /admin/organizations", loc)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && go test ./cmd/server -run 'TestHandleAdminRoot|TestMuxAdminRoot' -count=1`

Expected: FAIL — `handleAdminRoot` undefined / mux 404

- [ ] **Step 3: Implement handler + register**

Create `server/cmd/server/admin_root.go`:

```go
package main

import "net/http"

func (s *Server) handleAdminRoot(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requirePlatformAdmin(w, r); !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.Redirect(w, r, adminPath("organizations"), http.StatusSeeOther)
}
```

In `newMux()` near the other `/admin/…` registrations, add:

```go
mux.HandleFunc("/admin", s.handleAdminRoot)
```

Do **not** add `mux.HandleFunc("/admin/", …)` (subtree would shadow section routes).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd server && go test ./cmd/server -run 'TestHandleAdminRoot|TestMuxAdminRoot' -count=1`

Expected: PASS

Also smoke: `cd server && go test ./cmd/server -run 'TestHandleAdminCategories|TestHandleAdminOrgs' -count=1` — categories/orgs must still work (orgs still at old path until Task 4).

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/admin_root.go server/cmd/server/admin_root_test.go server/cmd/server/main.go
git commit -m "$(cat <<'EOF'
feat(admin): redirect /admin to organizations console

EOF
)"
```

---

### Task 4: Hard-cut rename `/admin/orgs` → `/admin/organizations`

**Files:**
- Modify: `server/cmd/server/main.go` — mux paths, `handleAdminOrgs` path trim prefix, `platformAdminPath`, `platformAdminView` ActivePanel/`buildPlatformAdminBreadcrumbs("organizations")`, `handlePlatformAdminLogo` path trim
- Modify: `server/cmd/server/admin_console.go` — nav href `adminPath("organizations")`; Active `active == "organizations" || (active != "categories" && …)` prefer `active != "categories"` still OK if default panel is organizations
- Modify: all Go tests that hit `/admin/orgs` (especially `admin_handler_identity_test.go`, `platform_admin_htmx_test.go`, `auth_session_handler_test.go`, `panel_markup_test.go`, `helpers_misc_coverage_test.go`, `auth_helpers_test.go`, `auth_bootstrap_test.go`, `auth_reset_handler_test.go`, `admin_console_test.go`)
- Modify: `server/templates/pages/platform_admin.html` — every `/admin/orgs` → `/admin/organizations` (forms, HTMX, logo `src`, pagination)
- Modify: `server/design/design.go` — OpenAPI HTTP paths for platform orgs methods

**Interfaces:**
- Consumes: `adminPath`
- Produces: live console at `/admin/organizations` (+ logo); legacy `/admin/orgs*` → 404 via mux (no handler registered)

- [ ] **Step 1: Write / update failing tests for the new path and hard cut**

Append to `server/cmd/server/admin_root_test.go`:

```go
func TestLegacyAdminOrgsPathGone(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()
	server := &Server{
		authorizer:  fakeAuthorizer{},
		identity:    &fakeIdentityStore{},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	for _, path := range []string{"/admin/orgs", "/admin/orgs/", "/admin/orgs/logo/x"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()
		server.newMux().ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want 404", path, rec.Code)
		}
	}
}

func TestMuxAdminOrganizationsPathOK(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()
	server := &Server{
		authorizer: fakeAuthorizer{},
		identity: &fakeIdentityStore{
			listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
				return nil, nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/organizations", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()
	server.newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}
}
```

Bulk-update existing tests: replace string `/admin/orgs` with `/admin/organizations` in request URLs and assertions. Keep the Go handler name `handleAdminOrgs` (URL rename only).

Add `"context"` to the import block in `admin_root_test.go` for `TestMuxAdminOrganizationsPathOK`.

- [ ] **Step 2: Run a focused subset to see failures**

Run: `cd server && go test ./cmd/server -run 'TestLegacyAdminOrgs|TestAdminOrganizations|TestHandleAdminOrgs|TestPlatformAdmin|platformAdminPath' -count=1`

Expected: FAIL until implementation

- [ ] **Step 3: Implement rename**

In `newMux()`:

```go
mux.HandleFunc("/admin", s.handleAdminRoot)
mux.HandleFunc("/admin/organizations", s.handleAdminOrgs)
mux.HandleFunc("/admin/organizations/", s.handleAdminOrgs)
mux.HandleFunc("/admin/categories", s.handleAdminCategories)
mux.HandleFunc("/admin/categories/", s.handleAdminCategoriesPath)
// remove /admin/orgs registrations
```

In `handleAdminOrgs`, change prefix trim to `/admin/organizations`.

In `handlePlatformAdminLogo`, trim `/admin/organizations/logo/`.

Update `platformAdminPath`:

```go
func platformAdminPath(query string, page int) string {
	// … build query …
	base := adminPath("organizations")
	if encoded != "" {
		return base + "?" + encoded
	}
	return base
}
```

In `platformAdminView`:

```go
ActivePanel: "organizations",
Breadcrumbs: buildPlatformAdminBreadcrumbs("organizations"),
```

In `platformAdminConsole`:

```go
{
	Href:   adminPath("organizations"),
	Title:  "Organizations",
	Copy:   "Create and manage organizations",
	Active: active != "categories",
},
{
	Href:   adminPath("categories"),
	Title:  "Categories",
	Copy:   "Manage stream discovery taxonomy",
	Active: active == "categories",
},
```

Templates: replace hardcoded `/admin/orgs` with `/admin/organizations` (or `{{/* keep literal paths matching adminPath */}}`).

`design.go`: change GET/POST `/admin/orgs` → `/admin/organizations`, logo `/admin/organizations/logo/{logo_id}`.

- [ ] **Step 4: Run tests**

Run: `cd server && go test ./cmd/server -count=1`

Expected: PASS (fix any remaining `/admin/orgs` string assertions)

Then regenerate OpenAPI:

Run: `task goa:generate` (from repo/worktree root)

Commit generated files if they change.

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server server/templates/pages/platform_admin.html server/design/design.go
# include any goa gen output under server/gen if changed
git commit -m "$(cat <<'EOF'
feat(admin): rename /admin/orgs to /admin/organizations

Hard-cut legacy orgs paths to 404; keep categories URLs unchanged.
EOF
)"
```

---

### Task 5: Topbar **Admin** link + docs

**Files:**
- Modify: `server/templates/layout.html` — label/href
- Modify: `server/cmd/server/main.go` — rename `ShowOrgsLink` → `ShowAdminLink` on `PageBase` and assignment site (~1433)
- Modify: `server/cmd/server/test_helpers_test.go` — stub layout marker `Admin` instead of `Orgs`
- Modify: any tests asserting `ShowOrgsLink` / `href="/admin/orgs"` / visible `Orgs` (e.g. `auth_session_handler_test.go`)
- Modify: `AGENTS.md` — platform admin routes and topbar bullet

**Interfaces:**
- Consumes: `adminPath("")` → `/admin`
- Produces: account menu item visible when `ShowAdminLink`; text `Admin`; `href="/admin"`

- [ ] **Step 1: Write / update failing assertion**

In the session/layout test that currently forbids or expects `href="/admin/orgs"`, assert:

```go
if !strings.Contains(body, `href="/admin"`) || !strings.Contains(body, "Admin") {
	t.Fatalf("expected Admin link to /admin, got:\n%s", body)
}
if strings.Contains(body, `href="/admin/orgs"`) || strings.Contains(body, ">Orgs<") {
	t.Fatalf("legacy Orgs link still present")
}
```

Update `test_helpers_test.go` stub:

```go
{{if .ShowAdminLink}} Admin{{end}}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && go test ./cmd/server -run 'ShowOrgs|ShowAdmin|Orgs|Login|Session' -count=1`

Expected: FAIL on renamed field / old copy

- [ ] **Step 3: Implement**

`layout.html`:

```html
{{ if .ShowAdminLink }}
  <a href="/admin" class="account-menu-item">
    …
    Admin
  </a>
{{ end }}
```

Rename struct field and all references `ShowOrgsLink` → `ShowAdminLink`.

Update `AGENTS.md` bullets:

- Platform admin: `/admin` (redirects to `/admin/organizations`); sections `/admin/organizations`, `/admin/categories`; logo `/admin/organizations/logo/:id`
- Topbar: platform admin sees `Admin` (`/admin`)
- Remove legacy `/admin/orgs` citations

- [ ] **Step 4: Full package test**

Run: `cd server && go test ./cmd/server -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/templates/layout.html server/cmd/server/main.go server/cmd/server/test_helpers_test.go server/cmd/server/*_test.go AGENTS.md
git commit -m "$(cat <<'EOF'
feat(admin): topbar Admin entry points at /admin

EOF
)"
```

---

### Task 6: Final sweep + verification

**Files:**
- Any remaining `/admin/orgs` hits under the worktree (docs plans/specs historical mentions may stay; **code + AGENTS.md + live design** must be clean)

- [ ] **Step 1: Search for leftovers**

Run:

```bash
rg '/admin/orgs|ShowOrgsLink|ActivePanel:.*"orgs"|buildPlatformAdminBreadcrumbs\("orgs"\)' \
  server AGENTS.md README.md QUICKSTART.md DOCKER.md \
  --glob '!docs/superpowers/**'
```

Expected: no matches in live code/docs (historical specs/plans under `docs/superpowers/` may still mention old paths — leave them).

- [ ] **Step 2: Manual checklist (against `task dev` in this worktree)**

1. Platform admin login → account menu shows **Admin** → lands on organizations console via `/admin` redirect  
2. Breadcrumbs on organizations: Dashboard → Platform admin → Organizations  
3. Soft-nav to Categories → breadcrumbs end with Categories; Platform admin crumb goes to `/admin`  
4. `/admin/orgs` → 404  
5. Org logo still loads from `/admin/organizations/logo/…`  
6. Org search/pagination HTMX still targets `#platform-admin-results`

- [ ] **Step 3: Commit only if sweep produced fixes**

If `git status` is clean, skip. Otherwise:

```bash
git add -u
git commit -m "$(cat <<'EOF'
chore(admin): finish platform admin routing path sweep

EOF
)"
```

---

## Spec coverage (self-check)

| Spec requirement | Task |
|------------------|------|
| `/admin` → redirect to organizations | 3 |
| `/admin/` behaves as entry (via ServeMux or exact handler) | 3 |
| `/admin/organizations` (+ logo) | 4 |
| Categories unchanged | 4 (no URL change) |
| Hard-cut `/admin/orgs*` 404 | 4 |
| `adminPath` helper | 1 |
| Three-crumb breadcrumbs; middle → `/admin` | 2 |
| ActivePanel `organizations` \| `categories` | 4 |
| Topbar Admin → `/admin` | 5 |
| AGENTS.md / OpenAPI design update | 4–5 |
| Tests for redirect, 404, breadcrumbs, paths | 1–5 |
| No landing page / no unified mux / no categories rename | honored (out of scope) |
