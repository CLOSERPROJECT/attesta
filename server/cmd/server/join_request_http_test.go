package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleOnboardingJoinSubmitAndPending(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-join-submit"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-1",
		Email:          "newbie@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	store := NewMemoryStore()
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	identity.getOrganizationBySlugFunc = func(ctx context.Context, slug string) (*IdentityOrg, error) {
		if strings.TrimSpace(slug) != "acme" {
			return nil, ErrIdentityNotFound
		}
		return &IdentityOrg{
			ID:   "team-1",
			Slug: "acme",
			Name: "Acme Org",
			Roles: []IdentityRole{
				{Slug: "viewer", Name: "Viewer"},
				{Slug: "editor", Name: "Editor"},
			},
		}, nil
	}
	identity.listOrganizationsPageFunc = func(ctx context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error) {
		return IdentityOrgPage{
			Organizations: []IdentityOrg{{
				ID:   "team-1",
				Slug: "acme",
				Name: "Acme Org",
				Roles: []IdentityRole{
					{Slug: "viewer", Name: "Viewer"},
					{Slug: "editor", Name: "Editor"},
				},
			}},
			Total: 1,
		}, nil
	}
	identity.listOrganizationUsersFunc = func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
		return []IdentityUser{{
			ID:         "admin-1",
			Email:      "admin@acme.example",
			OrgSlug:    "acme",
			IsOrgAdmin: true,
		}}, nil
	}
	server := &Server{
		identity:    identity,
		store:       store,
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	postReq := httptest.NewRequest(http.MethodPost, "/my/onboarding/join", strings.NewReader("org_slug=acme&roles=viewer&roles=editor"))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	postRec := httptest.NewRecorder()
	server.handleMyRoutes(postRec, postReq)

	if postRec.Code != http.StatusSeeOther {
		t.Fatalf("submit status = %d, want %d body=%q", postRec.Code, http.StatusSeeOther, postRec.Body.String())
	}
	if loc := postRec.Header().Get("Location"); loc != "/my/onboarding" {
		t.Fatalf("submit location = %q, want /my/onboarding", loc)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/my/onboarding", nil)
	getReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	getRec := httptest.NewRecorder()
	server.handleMyRoutes(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("pending status = %d, want %d body=%q", getRec.Code, http.StatusOK, getRec.Body.String())
	}
	body := getRec.Body.String()
	for _, want := range []string{
		"Pending join request",
		"Acme Org",
		"Viewer, Editor",
		`id="undo-request-dialog"`,
		"Undo request",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in pending hub, got:\n%s", want, body)
		}
	}
	if strings.Contains(body, `name="org_slug"`) || strings.Contains(body, `href="/my/onboarding/join"`) {
		t.Fatalf("expected no join form/CTA while pending, got:\n%s", body)
	}

	childReq := httptest.NewRequest(http.MethodGet, "/my/onboarding/join", nil)
	childReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	childRec := httptest.NewRecorder()
	server.handleMyRoutes(childRec, childReq)
	if childRec.Code != http.StatusSeeOther || childRec.Header().Get("Location") != "/my/onboarding" {
		t.Fatalf("pending join child = %d %q, want 303 /my/onboarding", childRec.Code, childRec.Header().Get("Location"))
	}
}

func TestHandleOnboardingJoinAffiliatedBlocked(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-join-affiliated"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-2",
		Email:          "member@example.com",
		OrgSlug:        "acme",
		Status:         "active",
		CreatedAt:      now,
	}
	store := NewMemoryStore()
	server := &Server{
		identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
		store:       store,
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodPost, "/my/onboarding/join", strings.NewReader("org_slug=other&roles=viewer"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if rec.Header().Get("Location") != "/my" {
		t.Fatalf("location = %q, want /my", rec.Header().Get("Location"))
	}
	pending, err := server.affiliationService().PendingJoinRequestForUser(context.Background(), "user-2")
	if err != nil {
		t.Fatalf("PendingJoinRequestForUser: %v", err)
	}
	if pending != nil {
		t.Fatalf("expected no pending request for affiliated user, got %+v", pending)
	}
}

func TestHandleOnboardingJoinSearchOmitsOrgAdminRole(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-join-search"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-3",
		Email:          "searcher@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	identity.listOrganizationsPageFunc = func(ctx context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error) {
		if strings.TrimSpace(opts.Search) == "" {
			t.Fatal("expected non-empty search for this test")
		}
		return IdentityOrgPage{
			Organizations: []IdentityOrg{{ID: "team-1", Slug: "acme", Name: "Acme Org"}},
			Total:         1,
		}, nil
	}
	identity.getOrganizationBySlugFunc = func(ctx context.Context, slug string) (*IdentityOrg, error) {
		return &IdentityOrg{
			ID:   "team-1",
			Slug: "acme",
			Name: "Acme Org",
			Roles: []IdentityRole{
				{Slug: "viewer", Name: "Viewer"},
				{Slug: "org-admin", Name: "Org Admin"},
			},
		}, nil
	}
	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	listReq := httptest.NewRequest(http.MethodGet, "/my/onboarding/join?q=acme", nil)
	listReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	listRec := httptest.NewRecorder()
	server.handleMyRoutes(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d body=%q", listRec.Code, http.StatusOK, listRec.Body.String())
	}
	listBody := listRec.Body.String()
	for _, want := range []string{
		"Get started",
		`href="/my/onboarding"`,
		"Acme Org",
		`class="list-row"`,
		`hx-get="/my/onboarding/join?org=acme&amp;q=acme"`,
		`hx-target="#join-org-dialog-body"`,
		`id="join-org-dialog"`,
		`id="join-org-dialog-body"`,
	} {
		if !strings.Contains(listBody, want) {
			t.Fatalf("expected %q in join list page, got:\n%s", want, listBody)
		}
	}
	if strings.Contains(listBody, `<span class="muted">acme</span>`) {
		t.Fatalf("join results must not show org slug, got:\n%s", listBody)
	}

	dialogReq := httptest.NewRequest(http.MethodGet, "/my/onboarding/join?q=acme&org=acme", nil)
	dialogReq.Header.Set("HX-Request", "true")
	dialogReq.Header.Set("HX-Target", "join-org-dialog-body")
	dialogReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	dialogRec := httptest.NewRecorder()
	server.handleMyRoutes(dialogRec, dialogReq)
	if dialogRec.Code != http.StatusOK {
		t.Fatalf("dialog status = %d, want %d body=%q", dialogRec.Code, http.StatusOK, dialogRec.Body.String())
	}
	dialogBody := dialogRec.Body.String()
	for _, want := range []string{
		`data-join-dialog-card`,
		`data-role-picker`,
		`data-role-picker-option`,
		`data-role-picker-require-selection`,
		`data-role-picker-submit`,
		`data-value="viewer"`,
		`data-label="Viewer"`,
		"Submit join request",
		"Acme Org",
	} {
		if !strings.Contains(dialogBody, want) {
			t.Fatalf("expected %q in join dialog partial, got:\n%s", want, dialogBody)
		}
	}
	if strings.Contains(dialogBody, "Viewer (viewer)") {
		t.Fatalf("join role picker must show pills, not name+slug, got:\n%s", dialogBody)
	}
	if strings.Contains(dialogBody, `data-value="org-admin"`) {
		t.Fatalf("join role picker must omit org-admin, got:\n%s", dialogBody)
	}
	if strings.Contains(dialogBody, `type="checkbox" name="roles"`) {
		t.Fatalf("expected roles-picker dialog, not inline checkboxes, got:\n%s", dialogBody)
	}
	if strings.Contains(dialogBody, "Join an organization") {
		t.Fatalf("dialog partial must not include full page, got:\n%s", dialogBody)
	}
}

func TestHandleOnboardingJoinSearchPagination(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-join-pagination"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-page",
		Email:          "pager@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	var sawOpts IdentityOrgListOptions
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	identity.listOrganizationsPageFunc = func(ctx context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error) {
		sawOpts = opts
		orgs := make([]IdentityOrg, 0, onboardingJoinSearchLimit)
		for i := 0; i < onboardingJoinSearchLimit; i++ {
			n := opts.Offset + i + 1
			orgs = append(orgs, IdentityOrg{
				ID:   "team-" + strconv.Itoa(n),
				Slug: "org-" + strconv.Itoa(n),
				Name: "Org " + strconv.Itoa(n),
			})
		}
		return IdentityOrgPage{Organizations: orgs, Total: 25}, nil
	}
	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/my/onboarding/join?q=org&page=2", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if sawOpts.Search != "org" || sawOpts.Limit != onboardingJoinSearchLimit || sawOpts.Offset != onboardingJoinSearchLimit {
		t.Fatalf("list opts = %+v, want search=org limit=%d offset=%d", sawOpts, onboardingJoinSearchLimit, onboardingJoinSearchLimit)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`aria-label="Organizations pagination"`,
		`href="/my/onboarding/join?q=org"`,
		`href="/my/onboarding/join?q=org&amp;page=3"`,
		"Org 13",
		`hx-get="/my/onboarding/join?org=org-13&amp;page=2&amp;q=org"`,
		`hx-target="#join-org-dialog-body"`,
		`class="empty-state"`,
	} {
		if want == `class="empty-state"` {
			if strings.Contains(body, want) {
				t.Fatalf("did not expect empty-state when results exist, got:\n%s", body)
			}
			continue
		}
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in paginated join page, got:\n%s", want, body)
		}
	}
	if strings.Contains(body, `>Select</a>`) {
		t.Fatalf("expected whole-row links, not Select buttons, got:\n%s", body)
	}
}

func TestHandleOnboardingJoinBrowseAllPagination(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-join-browse"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-browse",
		Email:          "browser@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	var sawOpts IdentityOrgListOptions
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	identity.listOrganizationsPageFunc = func(ctx context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error) {
		sawOpts = opts
		orgs := make([]IdentityOrg, 0, onboardingJoinSearchLimit)
		for i := 0; i < onboardingJoinSearchLimit; i++ {
			n := opts.Offset + i + 1
			orgs = append(orgs, IdentityOrg{
				ID:   "team-" + strconv.Itoa(n),
				Slug: "org-" + strconv.Itoa(n),
				Name: "Org " + strconv.Itoa(n),
			})
		}
		return IdentityOrgPage{Organizations: orgs, Total: 25}, nil
	}
	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/my/onboarding/join?page=2", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if sawOpts.Search != "" || sawOpts.Limit != onboardingJoinSearchLimit || sawOpts.Offset != onboardingJoinSearchLimit {
		t.Fatalf("list opts = %+v, want empty search limit=%d offset=%d", sawOpts, onboardingJoinSearchLimit, onboardingJoinSearchLimit)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Get started",
		`href="/my/onboarding"`,
		`aria-label="Organizations pagination"`,
		`href="/my/onboarding/join"`,
		`href="/my/onboarding/join?page=3"`,
		"Org 13",
		`hx-get="/my/onboarding/join?org=org-13&amp;page=2"`,
		`hx-target="#join-org-dialog-body"`,
		`hx-trigger="input changed delay:200ms, search"`,
		`id="onboarding-join-results"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in browse-all join page, got:\n%s", want, body)
		}
	}
}

func TestHandleOnboardingJoinHTMXResultsPartial(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-join-htmx"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-htmx",
		Email:          "htmx@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	identity.listOrganizationsPageFunc = func(ctx context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error) {
		return IdentityOrgPage{
			Organizations: []IdentityOrg{{
				ID:         "team-1",
				Slug:       "acme",
				Name:       "Acme Org",
				LogoFileID: "logo-1",
			}},
			Total: 1,
		}, nil
	}
	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/my/onboarding/join?q=acme", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "onboarding-join-results")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`id="onboarding-join-results"`,
		"Acme Org",
		`src="/organization/logo/acme"`,
		`class="list-row"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in HTMX partial, got:\n%s", want, body)
		}
	}
	if strings.Contains(body, "Join an organization") || strings.Contains(body, `id="join-org-search"`) {
		t.Fatalf("HTMX results partial must not include full page chrome, got:\n%s", body)
	}
}

func TestHandleOnboardingJoinSearchEmptyState(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-join-empty"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-empty",
		Email:          "empty@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	identity.listOrganizationsPageFunc = func(ctx context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error) {
		return IdentityOrgPage{Organizations: nil, Total: 0}, nil
	}
	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/my/onboarding/join?q=zzz", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`class="empty-state"`,
		`class="empty-state-title">No organizations match your search<`,
		`class="empty-state-hint">Try a different name.<`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in empty join search, got:\n%s", want, body)
		}
	}
}

func TestHandleOrgAdminMembersPendingJoinRequests(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	saved, err := store.InsertJoinRequest(context.Background(), JoinRequest{
		RequesterUserID: "user-1",
		RequesterEmail:  "newbie@example.com",
		OrgSlug:         "acme",
		RoleSlugs:       []string{"viewer"},
		Status:          AffiliationStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		t.Fatalf("InsertJoinRequest: %v", err)
	}

	server := &Server{
		authorizer: fakeAuthorizer{},
		store:      store,
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "admin-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{
					ID:         "admin-1",
					Email:      "admin@acme.example",
					OrgSlug:    "acme",
					Labels:     []string{identityOrgAdminLabel},
					IsOrgAdmin: true,
					Status:     "active",
				}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				return &IdentityOrg{
					ID:    "team-1",
					Slug:  "acme",
					Name:  "Acme Org",
					Roles: []IdentityRole{{Slug: "viewer", Name: "Viewer"}},
				}, nil
			},
			listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
				return []IdentityUser{{
					ID:         "admin-1",
					Email:      "admin@acme.example",
					OrgSlug:    "acme",
					IsOrgAdmin: true,
					Status:     "active",
				}}, nil
			},
			listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
				return nil, nil
			},
		},
		tmpl:        parseTestTemplates(t),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/my/organization/members", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()
	server.handleOrgAdminPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Pending join requests",
		"newbie@example.com",
		"Viewer",
		`class="pill pill-sm role-pill"`,
		`class="role-pill-row"`,
		saved.ID.Hex(),
		`name="intent" value="approve_join"`,
		`name="intent" value="reject_join"`,
		"<h2>Users</h2>",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in members page, got:\n%s", want, body)
		}
	}
}

func TestHandleOrgAdminUsersApproveAndRejectJoin(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()

	approveReq, err := store.InsertJoinRequest(context.Background(), JoinRequest{
		RequesterUserID: "user-approve",
		RequesterEmail:  "approve@example.com",
		OrgSlug:         "acme",
		RoleSlugs:       []string{"viewer"},
		Status:          AffiliationStatusPending,
	})
	if err != nil {
		t.Fatalf("insert approve request: %v", err)
	}
	rejectReq, err := store.InsertJoinRequest(context.Background(), JoinRequest{
		RequesterUserID: "user-reject",
		RequesterEmail:  "reject@example.com",
		OrgSlug:         "acme",
		RoleSlugs:       []string{"editor"},
		Status:          AffiliationStatusPending,
	})
	if err != nil {
		t.Fatalf("insert reject request: %v", err)
	}

	var addedUserID string
	var stampedUserID string
	var stampedLabels []string
	server := &Server{
		authorizer: fakeAuthorizer{},
		store:      store,
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "admin-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{
					ID:         "admin-1",
					Email:      "admin@acme.example",
					OrgSlug:    "acme",
					Labels:     []string{identityOrgAdminLabel},
					IsOrgAdmin: true,
					Status:     "active",
				}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				return &IdentityOrg{
					ID:   "team-1",
					Slug: "acme",
					Name: "Acme Org",
					Roles: []IdentityRole{
						{Slug: "viewer", Name: "Viewer"},
						{Slug: "editor", Name: "Editor"},
					},
				}, nil
			},
			getUserByIDFunc: func(ctx context.Context, userID string) (IdentityUser, error) {
				switch userID {
				case "user-approve":
					return IdentityUser{ID: "user-approve", Email: "approve@example.com", Status: "active"}, nil
				case "user-reject":
					return IdentityUser{ID: "user-reject", Email: "reject@example.com", Status: "active"}, nil
				default:
					return IdentityUser{ID: userID, Status: "active"}, nil
				}
			},
			addOrganizationUserByIDAsAdminFunc: func(ctx context.Context, orgSlug, userID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
				addedUserID = userID
				if isOrgAdmin {
					t.Fatalf("approve must not grant org-admin membership")
				}
				return IdentityMembership{ID: "membership-" + userID, UserID: userID, RoleSlugs: append([]string(nil), roleSlugs...)}, nil
			},
			updateUserLabelsFunc: func(ctx context.Context, userID string, labels []string) (IdentityUser, error) {
				stampedUserID = userID
				stampedLabels = append([]string(nil), labels...)
				return IdentityUser{ID: userID, Labels: labels}, nil
			},
			listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
				return nil, nil
			},
			listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
				return nil, nil
			},
		},
		tmpl:        parseTestTemplates(t),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	approveHTTP := httptest.NewRequest(http.MethodPost, "/my/organization/users", strings.NewReader("intent=approve_join&request_id="+approveReq.ID.Hex()))
	approveHTTP.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	approveHTTP.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	approveRec := httptest.NewRecorder()
	server.handleOrgAdminUsers(approveRec, approveHTTP)

	if approveRec.Code != http.StatusSeeOther {
		t.Fatalf("approve status = %d, want %d body=%q", approveRec.Code, http.StatusSeeOther, approveRec.Body.String())
	}
	if loc := approveRec.Header().Get("Location"); loc != "/my/organization/members" {
		t.Fatalf("approve location = %q", loc)
	}
	if addedUserID != "user-approve" || stampedUserID != "user-approve" {
		t.Fatalf("approve wiring addedUserID=%q stampedUserID=%q", addedUserID, stampedUserID)
	}
	if !containsRole(decodeIdentityRoleLabels(stampedLabels), "viewer") || hasIdentityLabel(stampedLabels, identityOrgAdminLabel) {
		t.Fatalf("approve labels = %#v", stampedLabels)
	}
	loadedApprove, err := store.LoadJoinRequestByID(context.Background(), approveReq.ID)
	if err != nil || loadedApprove == nil || loadedApprove.Status != AffiliationStatusApproved {
		t.Fatalf("approve persisted=%+v err=%v", loadedApprove, err)
	}

	rejectHTTP := httptest.NewRequest(http.MethodPost, "/my/organization/users", strings.NewReader("intent=reject_join&request_id="+rejectReq.ID.Hex()+"&reason=not+a+fit"))
	rejectHTTP.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rejectHTTP.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rejectRec := httptest.NewRecorder()
	server.handleOrgAdminUsers(rejectRec, rejectHTTP)

	if rejectRec.Code != http.StatusSeeOther {
		t.Fatalf("reject status = %d, want %d body=%q", rejectRec.Code, http.StatusSeeOther, rejectRec.Body.String())
	}
	if loc := rejectRec.Header().Get("Location"); loc != "/my/organization/members" {
		t.Fatalf("reject location = %q", loc)
	}
	loadedReject, err := store.LoadJoinRequestByID(context.Background(), rejectReq.ID)
	if err != nil || loadedReject == nil || loadedReject.Status != AffiliationStatusRejected || loadedReject.RejectReason != "not a fit" {
		t.Fatalf("reject persisted=%+v err=%v", loadedReject, err)
	}
}
