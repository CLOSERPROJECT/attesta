# Platform Admin Orgs Perf Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `GET /admin/orgs` fast by stopping per-member Appwrite user hydration on the org table, then add early org-admin filtering, parallel per-org fetches, and Appwrite-paged team listing.

**Architecture:** Keep full `ListOrganizationMemberships` (hydrates each member via `GetUserByID`) for flows that need label-accurate roles. Add a lite membership list that decodes Appwrite team membership rows only (`userEmail`, `confirm`, membership roles → `IsOrgAdmin`). Platform admin org rows use lite (+ optional team ID to skip `GetOrganizationBySlug`), filter to org-admins before summarizing, and fetch orgs in parallel. Org catalog paging moves from “list all teams then slice in memory” to `teams.List` with `query.Limit` / `query.Offset` / `OrderAsc("name")` and optional `WithListSearch`.

**Tech Stack:** Go `net/http` server, Appwrite Go SDK `v1.0.0` (`teams`, `query`), existing `IdentityStore` / `fakeIdentityStore`, `httptest` Appwrite mocks in `identity_appwrite_test.go`.

## Global Constraints

- Work only in git worktree `.worktrees/investigate/admin-orgs-slow-load` on branch `investigate/admin-orgs-slow-load`.
- Do **not** add caching.
- Prefer minimal diffs; do not refactor unrelated identity call sites.
- Lite membership `IsOrgAdmin` comes from membership roles (`owner` / invite encoding via `decodeInviteMembershipRoles`) — **not** from user labels. Org-admin invites already set `owner` via `encodeInviteMembershipRoles`. Document this in the lite method comment.
- Full hydrated `ListOrganizationMemberships` behavior must remain unchanged for org-admin member UIs / invite upgrade paths that still call it.
- TDD: failing test → implement → pass → commit per task.
- Run Go tests from `server/`: `go test -count=1 ./cmd/server/ -run <Name>`.

---

## File map

| File | Role |
|------|------|
| `server/cmd/server/identity.go` | Add `IdentityOrgListOptions`, `IdentityOrgPage`, `ListOrganizationsPage`, `ListOrganizationMembershipsLite` |
| `server/cmd/server/identity_appwrite.go` | Implement lite membership decode + paged org list; keep hydrated list as-is |
| `server/cmd/server/identity_mapping.go` | Optional tiny helper if decode is shared; otherwise keep decode next to Appwrite impl |
| `server/cmd/server/identity_test_helpers_test.go` | Fake impls for new interface methods |
| `server/cmd/server/identity_appwrite_test.go` | HTTP-count tests for lite list + paged list |
| `server/cmd/server/main.go` | `platformAdminView`, `platformOrganizations` / page helper, `platformAdminOrganizationRows` |
| `server/cmd/server/admin_handler_identity_test.go` | Existing platform admin status tests; extend for call amplification / parallel safety |
| `server/cmd/server/platform_admin_orgs_perf_test.go` | New focused perf-contract tests (call counts, paging) |

---

### Task 1: Lite membership decode + `ListOrganizationMembershipsLite`

**Files:**
- Modify: `server/cmd/server/identity.go`
- Modify: `server/cmd/server/identity_appwrite.go` (`ListOrganizationMemberships`, `toIdentityMembership`)
- Modify: `server/cmd/server/identity_test_helpers_test.go`
- Modify: `server/cmd/server/identity_appwrite_test.go`

**Interfaces:**
- Consumes: existing `IdentityMembership`, `decodeInviteMembershipRoles`, `identityMembershipOwnerRole`, `GetOrganizationBySlug`, `teams.ListMemberships`
- Produces:
  - `ListOrganizationMembershipsLite(ctx context.Context, orgSlug string) ([]IdentityMembership, error)` on `IdentityStore`
  - unexported `membershipFromAppwrite(membership *models.Membership, org *IdentityOrg) IdentityMembership` (no `GetUserByID`)
  - Hydrated `ListOrganizationMemberships` keeps calling user hydration (can call lite decode then optionally hydrate, or keep `toIdentityMembership` but extract shared decode)

- [ ] **Step 1: Write the failing Appwrite HTTP-count test**

Append to `server/cmd/server/identity_appwrite_test.go`:

```go
func TestAppwriteIdentityListOrganizationMembershipsLiteSkipsUserHydration(t *testing.T) {
	var userGETs int
	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/teams/acme":
			_, _ = w.Write([]byte(`{"$id":"acme","name":"Acme Org","prefs":{"schemaVersion":1,"slug":"acme"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/teams/acme/memberships":
			_, _ = w.Write([]byte(`{"total":2,"memberships":[
				{"$id":"m1","userId":"user-1","userEmail":"owner@example.com","teamId":"acme","teamName":"Acme Org","confirm":true,"roles":["owner"]},
				{"$id":"m2","userId":"user-2","userEmail":"member@example.com","teamId":"acme","teamName":"Acme Org","confirm":true,"roles":["member","iapprover"]}
			]}`))
		case strings.HasPrefix(r.URL.Path, "/v1/users/"):
			userGETs++
			t.Fatalf("lite list must not call users API: %s %s", r.Method, r.URL.Path)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())
	memberships, err := identity.ListOrganizationMembershipsLite(context.Background(), "acme")
	if err != nil {
		t.Fatalf("ListOrganizationMembershipsLite error: %v", err)
	}
	if userGETs != 0 {
		t.Fatalf("userGETs = %d, want 0", userGETs)
	}
	if len(memberships) != 2 {
		t.Fatalf("memberships = %#v", memberships)
	}
	if !memberships[0].IsOrgAdmin || memberships[0].Email != "owner@example.com" || !memberships[0].Confirmed {
		t.Fatalf("memberships[0] = %#v", memberships[0])
	}
	if memberships[1].IsOrgAdmin || memberships[1].Email != "member@example.com" {
		t.Fatalf("memberships[1] = %#v", memberships[1])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd server && go test -count=1 ./cmd/server/ -run TestAppwriteIdentityListOrganizationMembershipsLiteSkipsUserHydration -v`

Expected: FAIL — `IdentityStore` / `*appwriteIdentity` missing `ListOrganizationMembershipsLite`

- [ ] **Step 3: Add interface + fake + Appwrite implementation**

In `identity.go`, add to `IdentityStore`:

```go
ListOrganizationMembershipsLite(ctx context.Context, orgSlug string) ([]IdentityMembership, error)
```

In `identity_test_helpers_test.go`, add field + method:

```go
listOrganizationMembershipsLiteFunc func(ctx context.Context, orgSlug string) ([]IdentityMembership, error)

func (f *fakeIdentityStore) ListOrganizationMembershipsLite(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
	if f.listOrganizationMembershipsLiteFunc != nil {
		return f.listOrganizationMembershipsLiteFunc(ctx, orgSlug)
	}
	// Default: reuse full list hook so existing fakes keep working when only listOrganizationMembershipsFunc is set.
	return f.ListOrganizationMemberships(ctx, orgSlug)
}
```

In `identity_appwrite.go`, extract non-hydrating decode and implement lite list:

```go
func membershipFromAppwrite(membership *models.Membership, org *IdentityOrg) IdentityMembership {
	if membership == nil {
		return IdentityMembership{}
	}
	decodedRoles := decodeInviteMembershipRoles(membership.Roles)
	identity := IdentityMembership{
		ID:              strings.TrimSpace(membership.Id),
		TeamID:          strings.TrimSpace(membership.TeamId),
		UserID:          strings.TrimSpace(membership.UserId),
		Email:           strings.TrimSpace(membership.UserEmail),
		MembershipRoles: append([]string(nil), membership.Roles...),
		RoleSlugs:       append([]string(nil), decodedRoles.BusinessRoles...),
		IsOrgAdmin:      decodedRoles.IsOrgAdmin || hasMembershipRole(membership.Roles, identityMembershipOwnerRole),
		Confirmed:       membership.Confirm,
	}
	if org != nil {
		identity.TeamID = strings.TrimSpace(org.ID)
	}
	if invitedAt, err := parseAppwriteTime(membership.Invited); err == nil {
		identity.InvitedAt = invitedAt
	}
	if joinedAt, err := parseAppwriteTime(membership.Joined); err == nil {
		identity.JoinedAt = joinedAt
	}
	return identity
}

// ListOrganizationMembershipsLite returns memberships decoded from the team membership
// list only. IsOrgAdmin is derived from membership roles (owner / invite encoding), not
// user labels. Callers that need label-accurate roles must use ListOrganizationMemberships.
func (a *appwriteIdentity) ListOrganizationMembershipsLite(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	org, err := a.GetOrganizationBySlug(ctx, orgSlug)
	if err != nil {
		return nil, err
	}
	membershipList, err := teams.New(a.adminClient).ListMemberships(strings.TrimSpace(org.ID))
	if err != nil {
		return nil, normalizeIdentityError(err)
	}
	memberships := make([]IdentityMembership, 0, len(membershipList.Memberships))
	for i := range membershipList.Memberships {
		memberships = append(memberships, membershipFromAppwrite(&membershipList.Memberships[i], org))
	}
	return memberships, nil
}
```

Refactor `toIdentityMembership` to start from `membershipFromAppwrite` then optionally hydrate (preserve current overwrite behavior):

```go
func (a *appwriteIdentity) toIdentityMembership(ctx context.Context, membership *models.Membership, org *IdentityOrg) IdentityMembership {
	identity := membershipFromAppwrite(membership, org)
	if identity.UserID != "" {
		if user, err := a.GetUserByID(ctx, identity.UserID); err == nil {
			identity.Email = user.Email
			identity.RoleSlugs = decodeIdentityRoleLabels(user.Labels)
			identity.IsOrgAdmin = user.IsOrgAdmin
		}
	}
	return identity
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd server && go test -count=1 ./cmd/server/ -run 'TestAppwriteIdentityListOrganizationMembershipsLiteSkipsUserHydration|TestAppwriteIdentityListOrganizationMembershipsPendingMembership' -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/identity.go server/cmd/server/identity_appwrite.go server/cmd/server/identity_test_helpers_test.go server/cmd/server/identity_appwrite_test.go
git commit -m "$(cat <<'EOF'
feat(identity): add lite org membership list without user hydration

EOF
)"
```

---

### Task 2: Platform admin rows use lite list (biggest win)

**Files:**
- Modify: `server/cmd/server/main.go` (`platformAdminOrganizationRows`)
- Create: `server/cmd/server/platform_admin_orgs_perf_test.go`
- Modify: `server/cmd/server/admin_handler_identity_test.go` only if existing status test needs the lite fake hook

**Interfaces:**
- Consumes: `IdentityStore.ListOrganizationMembershipsLite`
- Produces: `platformAdminOrganizationRows` calls lite only (no `ListOrganizationMemberships`)

- [ ] **Step 1: Write failing amplification test**

Create `server/cmd/server/platform_admin_orgs_perf_test.go`:

```go
package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
)

func TestPlatformAdminViewDoesNotCallHydratedMembershipList(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	var hydratedCalls atomic.Int64
	var liteCalls atomic.Int64

	orgs := []IdentityOrg{
		{ID: "team-1", Slug: "accepted", Name: "Accepted Org"},
		{ID: "team-2", Slug: "pending", Name: "Pending Org"},
		{ID: "team-3", Slug: "missing", Name: "Missing Org"},
	}

	identity := &fakeIdentityStore{
		listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
			return orgs, nil
		},
		listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
			hydratedCalls.Add(1)
			t.Fatalf("platform admin view must not call ListOrganizationMemberships (%s)", orgSlug)
			return nil, nil
		},
		listOrganizationMembershipsLiteFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
			liteCalls.Add(1)
			switch orgSlug {
			case "accepted":
				return []IdentityMembership{
					{Email: "admin@example.com", Confirmed: true, IsOrgAdmin: true},
					{Email: "owner@example.com", Confirmed: true, IsOrgAdmin: true},
				}, nil
			case "pending":
				return []IdentityMembership{
					{Email: "pending-owner@example.com", Confirmed: false, IsOrgAdmin: true},
				}, nil
			default:
				return nil, nil
			}
		},
	}

	server := &Server{identity: identity, authorizer: fakeAuthorizer{}}
	view := server.platformAdminView(&AccountUser{Email: "admin@example.com", IsPlatformAdmin: true}, "", PlatformAdminErrors{})

	if hydratedCalls.Load() != 0 {
		t.Fatalf("hydratedCalls = %d", hydratedCalls.Load())
	}
	if liteCalls.Load() != 3 {
		t.Fatalf("liteCalls = %d, want 3", liteCalls.Load())
	}
	if len(view.Organizations) != 3 {
		t.Fatalf("organizations = %#v", view.Organizations)
	}
	if view.Organizations[0].OrgAdminStatus != "At least one org admin accepted" {
		t.Fatalf("accepted status = %q", view.Organizations[0].OrgAdminStatus)
	}
	if view.Organizations[2].OrgAdminStatus != "All org admin invites pending" {
		t.Fatalf("pending status = %q", view.Organizations[2].OrgAdminStatus)
	}
	if view.Organizations[1].OrgAdminStatus != "No org admin" {
		t.Fatalf("missing status = %q", view.Organizations[1].OrgAdminStatus)
	}
	_ = fmt.Sprintf // keep import if needed; remove if unused after edits
}
```

(Remove unused `fmt` if the compiler complains — prefer no dead imports.)

- [ ] **Step 2: Run test to verify it fails**

Run: `cd server && go test -count=1 ./cmd/server/ -run TestPlatformAdminViewDoesNotCallHydratedMembershipList -v`

Expected: FAIL with `platform admin view must not call ListOrganizationMemberships` (current code hits hydrated list; fake default lite → hydrated)

- [ ] **Step 3: Switch `platformAdminOrganizationRows` to lite**

In `main.go`, change the membership fetch:

```go
func platformAdminOrganizationRows(ctx context.Context, organizations []Organization, identity IdentityStore) []PlatformAdminOrganizationRow {
	rows := make([]PlatformAdminOrganizationRow, 0, len(organizations))
	for _, organization := range organizations {
		row := PlatformAdminOrganizationRow{
			Name:             organization.Name,
			Slug:             organization.Slug,
			LogoAttachmentID: organization.LogoAttachmentID,
		}
		if identity != nil && strings.TrimSpace(organization.Slug) != "" {
			memberships, err := identity.ListOrganizationMembershipsLite(ctx, organization.Slug)
			if err != nil {
				log.Printf("failed to list organization memberships for %s: %v", organization.Slug, err)
			} else {
				row.OrgAdminEmails, row.PendingOrgAdminEmails = summarizePlatformOrgAdminMemberships(memberships)
			}
		}
		row.OrgAdminStatus, row.OrgAdminStatusClassName = platformOrgAdminStatus(row.OrgAdminEmails, row.PendingOrgAdminEmails)
		rows = append(rows, row)
	}
	return rows
}
```

- [ ] **Step 4: Run tests**

Run: `cd server && go test -count=1 ./cmd/server/ -run 'TestPlatformAdminViewDoesNotCallHydratedMembershipList|TestPlatformAdminHelpers' -v`

Expected: PASS (include whatever existing test name wraps `platform admin view summarizes org admin status` — currently under `TestPlatformAdminHelpers` / similar in `admin_handler_identity_test.go`)

Also run: `cd server && go test -count=1 ./cmd/server/ -run 'TestPlatformAdmin|TestHandleAdminOrgs' -v` and fix any fakes that relied on hydrated-only hooks (lite defaults to hydrated in fake, so most tests should keep working).

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/main.go server/cmd/server/platform_admin_orgs_perf_test.go server/cmd/server/admin_handler_identity_test.go
git commit -m "$(cat <<'EOF'
perf(admin): load platform org admin status via lite memberships

EOF
)"
```

---

### Task 3: Filter earlier to org-admin memberships only

**Files:**
- Modify: `server/cmd/server/main.go` (`summarizePlatformOrgAdminMemberships` call site or new helper)
- Modify: `server/cmd/server/platform_admin_orgs_perf_test.go`

**Interfaces:**
- Consumes: lite memberships
- Produces: `filterPlatformOrgAdminMemberships(memberships []IdentityMembership) []IdentityMembership` used before summarize

- [ ] **Step 1: Write failing unit test**

Append to `platform_admin_orgs_perf_test.go`:

```go
func TestFilterPlatformOrgAdminMembershipsDropsNonAdmins(t *testing.T) {
	in := []IdentityMembership{
		{Email: "owner@example.com", IsOrgAdmin: true, Confirmed: true},
		{Email: "member@example.com", IsOrgAdmin: false, Confirmed: true},
		{Email: "pending@example.com", IsOrgAdmin: true, Confirmed: false},
	}
	got := filterPlatformOrgAdminMemberships(in)
	if len(got) != 2 {
		t.Fatalf("got = %#v", got)
	}
	if got[0].Email != "owner@example.com" || got[1].Email != "pending@example.com" {
		t.Fatalf("got = %#v", got)
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

Run: `cd server && go test -count=1 ./cmd/server/ -run TestFilterPlatformOrgAdminMembershipsDropsNonAdmins -v`

Expected: FAIL — `filterPlatformOrgAdminMemberships` undefined

- [ ] **Step 3: Implement filter + wire into rows**

```go
func filterPlatformOrgAdminMemberships(memberships []IdentityMembership) []IdentityMembership {
	out := make([]IdentityMembership, 0, len(memberships))
	for _, membership := range memberships {
		if !membership.IsOrgAdmin {
			continue
		}
		out = append(out, membership)
	}
	return out
}
```

In `platformAdminOrganizationRows`, after lite list succeeds:

```go
row.OrgAdminEmails, row.PendingOrgAdminEmails = summarizePlatformOrgAdminMemberships(
	filterPlatformOrgAdminMemberships(memberships),
)
```

Keep `summarizePlatformOrgAdminMemberships`’s platform-admin-email skip (`isPlatformAdminMembership`) as-is.

- [ ] **Step 4: Run tests**

Run: `cd server && go test -count=1 ./cmd/server/ -run 'TestFilterPlatformOrgAdminMembershipsDropsNonAdmins|TestPlatformAdminViewDoesNotCallHydratedMembershipList' -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/main.go server/cmd/server/platform_admin_orgs_perf_test.go
git commit -m "$(cat <<'EOF'
perf(admin): summarize only org-admin memberships on platform org rows

EOF
)"
```

---

### Task 4: Parallelize per-org membership fetches

**Files:**
- Modify: `server/cmd/server/main.go` (`platformAdminOrganizationRows`)
- Modify: `server/cmd/server/platform_admin_orgs_perf_test.go`

**Interfaces:**
- Consumes: `ListOrganizationMembershipsLite`, `golang.org/x/sync/errgroup` **or** stdlib `sync.WaitGroup` (prefer stdlib `WaitGroup` + mutex to avoid new dependency debates; `errgroup` is fine if already used in module — check `go.mod`; if absent, use `WaitGroup`)
- Produces: same row order as input `organizations` slice (stable order)

- [ ] **Step 1: Write failing concurrency-order test**

Append:

```go
func TestPlatformAdminOrganizationRowsPreservesOrderUnderParallelFetch(t *testing.T) {
	var calls atomic.Int64
	identity := &fakeIdentityStore{
		listOrganizationMembershipsLiteFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
			n := calls.Add(1)
			// Stagger later orgs so unordered append would scramble results.
			if orgSlug == "a" {
				time.Sleep(30 * time.Millisecond)
			}
			_ = n
			return []IdentityMembership{{Email: orgSlug + "-owner@example.com", IsOrgAdmin: true, Confirmed: true}}, nil
		},
	}
	orgs := []Organization{{Slug: "a", Name: "A"}, {Slug: "b", Name: "B"}, {Slug: "c", Name: "C"}}
	rows := platformAdminOrganizationRows(context.Background(), orgs, identity)
	if len(rows) != 3 || rows[0].Slug != "a" || rows[1].Slug != "b" || rows[2].Slug != "c" {
		t.Fatalf("rows = %#v", rows)
	}
	if rows[0].OrgAdminEmails[0] != "a-owner@example.com" || rows[2].OrgAdminEmails[0] != "c-owner@example.com" {
		t.Fatalf("emails = %#v", rows)
	}
}
```

(Add `"sync/atomic"` / `"time"` imports.)

This test should already pass with sequential code (order-preserving). That is OK — it is a **regression lock** before parallelization. Optionally assert elapsed &lt; sequential lower bound after parallelization in Step 4.

- [ ] **Step 2: Run — expect PASS (baseline lock)**

Run: `cd server && go test -count=1 ./cmd/server/ -run TestPlatformAdminOrganizationRowsPreservesOrderUnderParallelFetch -v`

Expected: PASS

- [ ] **Step 3: Parallelize with stable indexing**

Replace the loop body with pre-sized slice + WaitGroup:

```go
func platformAdminOrganizationRows(ctx context.Context, organizations []Organization, identity IdentityStore) []PlatformAdminOrganizationRow {
	rows := make([]PlatformAdminOrganizationRow, len(organizations))
	var wg sync.WaitGroup
	for i, organization := range organizations {
		wg.Add(1)
		go func(i int, organization Organization) {
			defer wg.Done()
			row := PlatformAdminOrganizationRow{
				Name:             organization.Name,
				Slug:             organization.Slug,
				LogoAttachmentID: organization.LogoAttachmentID,
			}
			if identity != nil && strings.TrimSpace(organization.Slug) != "" {
				memberships, err := identity.ListOrganizationMembershipsLite(ctx, organization.Slug)
				if err != nil {
					log.Printf("failed to list organization memberships for %s: %v", organization.Slug, err)
				} else {
					row.OrgAdminEmails, row.PendingOrgAdminEmails = summarizePlatformOrgAdminMemberships(
						filterPlatformOrgAdminMemberships(memberships),
					)
				}
			}
			row.OrgAdminStatus, row.OrgAdminStatusClassName = platformOrgAdminStatus(row.OrgAdminEmails, row.PendingOrgAdminEmails)
			rows[i] = row
		}(i, organization)
	}
	wg.Wait()
	return rows
}
```

Add `"sync"` to imports in `main.go` if missing.

- [ ] **Step 4: Run tests + optional timing check**

Run: `cd server && go test -count=1 ./cmd/server/ -run 'TestPlatformAdminOrganizationRowsPreservesOrderUnderParallelFetch|TestPlatformAdminViewDoesNotCallHydratedMembershipList' -v`

Expected: PASS

Optional: in the order test, set lite sleep to `20ms` for every org and assert `elapsed < 50ms` so parallelism is proven (3×20ms sequential would be ≥60ms). Only add if stable on CI; skip if flaky.

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/main.go server/cmd/server/platform_admin_orgs_perf_test.go
git commit -m "$(cat <<'EOF'
perf(admin): fetch platform org memberships in parallel

EOF
)"
```

---

### Task 5: Server-side team pagination (+ search)

**Files:**
- Modify: `server/cmd/server/identity.go`
- Modify: `server/cmd/server/identity_appwrite.go`
- Modify: `server/cmd/server/identity_test_helpers_test.go`
- Modify: `server/cmd/server/identity_appwrite_test.go`
- Modify: `server/cmd/server/main.go` (`platformAdminView`, replace or narrow `platformOrganizations` usage for this page)
- Modify: `server/cmd/server/platform_admin_orgs_perf_test.go`

**Interfaces:**
- Consumes: Appwrite `teams.List` with `WithListQueries([]string{query.Limit(n), query.Offset(o), query.OrderAsc("name")})`, optional `WithListSearch(q)`, `WithListTotal(true)`
- Produces:
  - `type IdentityOrgListOptions struct { Search string; Limit int; Offset int }`
  - `type IdentityOrgPage struct { Organizations []IdentityOrg; Total int }`
  - `ListOrganizationsPage(ctx context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error)`
  - `ListOrganizations` remains for other callers (unchanged behavior in this task)

**Behavior notes:**
- Empty search: page with limit/offset ordered by name.
- Non-empty search: pass `WithListSearch(trimmedQuery)` **and** still apply limit/offset. Accept Appwrite full-text search semantics (may differ slightly from previous `strings.Contains` on name). Do **not** fetch all teams to re-filter in memory on this page path.
- `MatchedOrganizations` / `TotalPages` must use `page.Total`, not `len(page.Organizations)`.
- Default Appwrite list limit is small; this task is also a correctness fix when org count &gt; 25.

- [ ] **Step 1: Write failing Appwrite paging test**

Append to `identity_appwrite_test.go`:

```go
func TestAppwriteIdentityListOrganizationsPagePassesLimitOffsetSearch(t *testing.T) {
	var gotQueries []string
	var gotSearch string
	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/v1/teams" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotSearch = r.URL.Query().Get("search")
		gotQueries = r.URL.Query()["queries[]"]
		if len(gotQueries) == 0 {
			// SDK may encode as queries — accept either form used by sdk-for-go v1.0.0
			gotQueries = r.URL.Query()["queries"]
		}
		_, _ = w.Write([]byte(`{"total":40,"teams":[{"$id":"acme","name":"Acme Org","prefs":{"schemaVersion":1,"slug":"acme"}}]}`))
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())
	page, err := identity.ListOrganizationsPage(context.Background(), IdentityOrgListOptions{
		Search: "acme",
		Limit:  12,
		Offset: 24,
	})
	if err != nil {
		t.Fatalf("ListOrganizationsPage error: %v", err)
	}
	if page.Total != 40 || len(page.Organizations) != 1 || page.Organizations[0].Slug != "acme" {
		t.Fatalf("page = %#v", page)
	}
	if gotSearch != "acme" {
		t.Fatalf("search = %q, want acme", gotSearch)
	}
	joined := strings.Join(gotQueries, ",")
	if !strings.Contains(joined, "12") || !strings.Contains(joined, "24") {
		t.Fatalf("queries = %#v (raw=%q), want limit 12 and offset 24", gotQueries, joined)
	}
}
```

If the SDK encodes queries as JSON strings in a single param, adjust assertions after observing the failing request dump (`t.Logf("%v", r.URL.RawQuery)`).

- [ ] **Step 2: Run — expect FAIL**

Run: `cd server && go test -count=1 ./cmd/server/ -run TestAppwriteIdentityListOrganizationsPagePassesLimitOffsetSearch -v`

Expected: FAIL — method undefined

- [ ] **Step 3: Implement `ListOrganizationsPage`**

In `identity.go`:

```go
type IdentityOrgListOptions struct {
	Search string
	Limit  int
	Offset int
}

type IdentityOrgPage struct {
	Organizations []IdentityOrg
	Total         int
}

// on IdentityStore:
ListOrganizationsPage(ctx context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error)
```

In `identity_appwrite.go`:

```go
func (a *appwriteIdentity) ListOrganizationsPage(ctx context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error) {
	if err := ctx.Err(); err != nil {
		return IdentityOrgPage{}, err
	}
	limit := opts.Limit
	if limit < 1 {
		limit = platformAdminOrganizationsPerPage // or local default 12; avoid import cycle — use literal 12 here if const lives in main
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}
	queries := []string{
		query.Limit(limit),
		query.Offset(offset),
		query.OrderAsc("name"),
	}
	options := []teams.ListOption{
		teams.New(a.adminClient).WithListQueries(queries),
		teams.New(a.adminClient).WithListTotal(true),
	}
	if search := strings.TrimSpace(opts.Search); search != "" {
		options = append(options, teams.New(a.adminClient).WithListSearch(search))
	}
	teamList, err := teams.New(a.adminClient).List(options...)
	if err != nil {
		return IdentityOrgPage{}, normalizeIdentityError(err)
	}
	if err := ctx.Err(); err != nil {
		return IdentityOrgPage{}, err
	}
	return IdentityOrgPage{
		Organizations: decodeIdentityOrgs(teamList),
		Total:         teamList.Total,
	}, nil
}
```

**Const placement:** do **not** reference `platformAdminOrganizationsPerPage` from `identity_appwrite.go` if that creates awkward coupling — use `if limit < 1 { limit = 12 }`.

Fake:

```go
listOrganizationsPageFunc func(ctx context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error)

func (f *fakeIdentityStore) ListOrganizationsPage(ctx context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error) {
	if f.listOrganizationsPageFunc != nil {
		return f.listOrganizationsPageFunc(ctx, opts)
	}
	orgs, err := f.ListOrganizations(ctx)
	if err != nil {
		return IdentityOrgPage{}, err
	}
	// In-memory page so existing tests that only stub ListOrganizations still work when platformAdminView switches.
	filtered := orgs
	if q := strings.ToLower(strings.TrimSpace(opts.Search)); q != "" {
		filtered = nil
		for _, org := range orgs {
			if strings.Contains(strings.ToLower(org.Name), q) {
				filtered = append(filtered, org)
			}
		}
	}
	total := len(filtered)
	limit := opts.Limit
	if limit < 1 {
		limit = 12
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return IdentityOrgPage{Organizations: append([]IdentityOrg(nil), filtered[offset:end]...), Total: total}, nil
}
```

Add import for `github.com/appwrite/sdk-for-go/query` in `identity_appwrite.go`.

- [ ] **Step 4: Wire `platformAdminView` to paged list**

Replace the list/filter/slice block in `platformAdminView` with:

```go
limit := platformAdminOrganizationsPerPage
requestedPage := errs.Page
if requestedPage < 1 {
	requestedPage = 1
}
offset := (requestedPage - 1) * limit

orgPage, err := s.identity.ListOrganizationsPage(context.Background(), IdentityOrgListOptions{
	Search: errs.SearchQuery,
	Limit:  limit,
	Offset: offset,
})
if err != nil {
	log.Printf("failed to list platform organizations page: %v", err)
	orgPage = IdentityOrgPage{}
}

currentPage := normalizePlatformAdminPage(requestedPage, orgPage.Total)
if currentPage != requestedPage {
	// Re-fetch if page was clamped (e.g. page=99 when only 2 pages).
	offset = (currentPage - 1) * limit
	orgPage, err = s.identity.ListOrganizationsPage(context.Background(), IdentityOrgListOptions{
		Search: errs.SearchQuery,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		log.Printf("failed to list platform organizations page: %v", err)
		orgPage = IdentityOrgPage{}
	}
}

organizations := make([]Organization, 0, len(orgPage.Organizations))
for _, org := range orgPage.Organizations {
	organizations = append(organizations, organizationFromIdentityOrg(org))
}

totalPages := 1
if orgPage.Total > 0 {
	totalPages = (orgPage.Total + limit - 1) / limit
}
pageNumbers := make([]int, 0, totalPages)
for page := 1; page <= totalPages; page++ {
	pageNumbers = append(pageNumbers, page)
}
rows := platformAdminOrganizationRows(context.Background(), organizations, s.identity)
```

Set view fields:

```go
MatchedOrganizations: orgPage.Total,
Organizations:        rows,
CurrentPage:          currentPage,
TotalPages:           totalPages,
PageNumbers:          pageNumbers,
HasPreviousPage:      currentPage > 1,
HasNextPage:          currentPage < totalPages,
PreviousPage:         max(currentPage-1, 1),
NextPage:             min(currentPage+1, totalPages),
SearchQuery:          errs.SearchQuery,
```

Leave `platformOrganizations` / `filterPlatformOrganizations` in place for any non-view callers/tests; if `platformAdminView` is the only consumer, do not delete them in this task unless tests force it — YAGNI cleanup can be a tiny follow-up commit inside this task only if unused and tests agree.

Update `TestPlatformAdminViewDoesNotCallHydratedMembershipList` fake: either stub `listOrganizationsPageFunc` **or** rely on fake default that pages `listOrganizationsFunc` (preferred — then existing stub keeps working).

Add paging contract test:

```go
func TestPlatformAdminViewUsesOrganizationPageTotal(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")
	identity := &fakeIdentityStore{
		listOrganizationsPageFunc: func(ctx context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error) {
			if opts.Limit != 12 || opts.Offset != 12 || opts.Search != "ac" {
				t.Fatalf("opts = %#v", opts)
			}
			return IdentityOrgPage{
				Total: 25,
				Organizations: []IdentityOrg{
					{ID: "t13", Slug: "org-13", Name: "Org 13"},
				},
			}, nil
		},
		listOrganizationMembershipsLiteFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
			return nil, nil
		},
	}
	server := &Server{identity: identity, authorizer: fakeAuthorizer{}}
	view := server.platformAdminView(&AccountUser{Email: "admin@example.com", IsPlatformAdmin: true}, "", PlatformAdminErrors{SearchQuery: "ac", Page: 2})
	if view.MatchedOrganizations != 25 || view.TotalPages != 3 || view.CurrentPage != 2 || len(view.Organizations) != 1 {
		t.Fatalf("view paging = matched=%d pages=%d page=%d rows=%d", view.MatchedOrganizations, view.TotalPages, view.CurrentPage, len(view.Organizations))
	}
}
```

- [ ] **Step 5: Run tests**

Run:

```bash
cd server && go test -count=1 ./cmd/server/ -run 'TestAppwriteIdentityListOrganizationsPagePassesLimitOffsetSearch|TestPlatformAdminViewUsesOrganizationPageTotal|TestPlatformAdminViewDoesNotCallHydratedMembershipList|TestPlatformAdmin' -v
```

Expected: PASS

Then broader identity compile check:

```bash
cd server && go test -count=1 ./cmd/server/ -count=1 2>&1 | tail -30
```

If full package is too slow, at minimum ensure all files compile: `go test -count=1 ./cmd/server/ -run TestDoesNotExist` should compile and print `ok` with no tests.

Fix any `publicCatalogIdentity` / other embedders only if they re-implement the whole interface (they embed `fakeIdentityStore` and should pick up new methods automatically).

- [ ] **Step 6: Commit**

```bash
git add server/cmd/server/identity.go server/cmd/server/identity_appwrite.go server/cmd/server/identity_test_helpers_test.go server/cmd/server/identity_appwrite_test.go server/cmd/server/main.go server/cmd/server/platform_admin_orgs_perf_test.go
git commit -m "$(cat <<'EOF'
perf(admin): page platform organizations via Appwrite teams.List

EOF
)"
```

---

### Task 6: End-to-end amplification regression (Appwrite mock)

**Files:**
- Modify: `server/cmd/server/platform_admin_orgs_perf_test.go`

**Interfaces:**
- Consumes: real `NewAppwriteIdentity` + `platformAdminView` after Tasks 1–5
- Produces: hard ceiling on HTTP calls for 12 orgs × 8 members

- [ ] **Step 1: Write failing ceiling test (if current code still hot) / lock test**

```go
func TestPlatformAdminViewAppwriteCallCeiling(t *testing.T) {
	const orgCount = 12
	const membersPerOrg = 8
	var totalRequests atomic.Int64
	var userRequests atomic.Int64

	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		totalRequests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		switch {
		case path == "/v1/teams":
			teamsPayload := make([]map[string]any, 0, orgCount)
			for i := 0; i < orgCount; i++ {
				id := fmt.Sprintf("org-%d", i)
				teamsPayload = append(teamsPayload, map[string]any{
					"$id": id, "name": fmt.Sprintf("Org %d", i),
					"prefs": map[string]any{"schemaVersion": 1, "slug": id},
				})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"total": orgCount, "teams": teamsPayload})
		case strings.HasSuffix(path, "/memberships") && strings.HasPrefix(path, "/v1/teams/"):
			teamID := strings.TrimSuffix(strings.TrimPrefix(path, "/v1/teams/"), "/memberships")
			teamID = strings.TrimSuffix(teamID, "/")
			memberships := make([]map[string]any, 0, membersPerOrg)
			for m := 0; m < membersPerOrg; m++ {
				memberships = append(memberships, map[string]any{
					"$id": fmt.Sprintf("%s-m-%d", teamID, m),
					"userId": fmt.Sprintf("%s-u-%d", teamID, m),
					"userEmail": fmt.Sprintf("u%d@%s.example", m, teamID),
					"teamId": teamID, "teamName": teamID, "confirm": true,
					"roles": []string{"member"},
				})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"total": membersPerOrg, "memberships": memberships})
		case strings.HasPrefix(path, "/v1/teams/"):
			teamID := strings.TrimPrefix(path, "/v1/teams/")
			_ = json.NewEncoder(w).Encode(map[string]any{"$id": teamID, "name": teamID, "prefs": map[string]any{"schemaVersion": 1, "slug": teamID}})
		case strings.HasPrefix(path, "/v1/users/"):
			userRequests.Add(1)
			t.Fatalf("unexpected user hydration: %s", path)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, path)
		}
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())
	server := &Server{identity: identity, authorizer: fakeAuthorizer{}}
	_ = server.platformAdminView(&AccountUser{Email: "admin@example.com", IsPlatformAdmin: true}, "", PlatformAdminErrors{})

	// 1 list page + ≤12 get team + 12 list memberships = ≤25 (no user calls)
	if userRequests.Load() != 0 {
		t.Fatalf("userRequests = %d", userRequests.Load())
	}
	if total := totalRequests.Load(); total > 25 {
		t.Fatalf("totalRequests = %d, want ≤ 25", total)
	}
}
```

- [ ] **Step 2: Run — expect PASS after Tasks 1–5**

Run: `cd server && go test -count=1 ./cmd/server/ -run TestPlatformAdminViewAppwriteCallCeiling -v`

Expected: PASS (`totalRequests ≤ 25`, `userRequests = 0`). Pre-fix baseline was **313** with user hydration.

- [ ] **Step 3: Commit**

```bash
git add server/cmd/server/platform_admin_orgs_perf_test.go
git commit -m "$(cat <<'EOF'
test(admin): lock platform orgs Appwrite call ceiling

EOF
)"
```

---

## Self-review

1. **Spec coverage**
   - Biggest win / lite list: Tasks 1–2
   - Further (1) lightweight API: Task 1
   - Further (2) filter earlier: Task 3
   - Further (3) parallelize: Task 4
   - Further (5) server-side pagination: Task 5
   - Cache explicitly omitted
   - Call-ceiling proof: Task 6

2. **Placeholder scan:** none intentional; Appwrite query encoding assertion in Task 5 may need one adjustment after observing `RawQuery` — that is an allowed empirical tweak, not a TBD.

3. **Type consistency:** `IdentityOrgListOptions` / `IdentityOrgPage` / `ListOrganizationsPage` / `ListOrganizationMembershipsLite` names are stable across tasks; fake defaults preserve older tests.

---

## Out of scope

- Caching org-admin summaries
- Speeding org-admin **members** page (`ListOrganizationUsers` still hydrates users)
- Changing invite / label reconciliation semantics for hydrated memberships
- Deleting unused `platformOrganizations` helpers unless proven dead in Task 5
