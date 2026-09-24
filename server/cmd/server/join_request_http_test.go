package main

import (
	"context"
	"net/http"
	"net/http/httptest"
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
	if loc := postRec.Header().Get("Location"); loc != "/my/onboarding/join" {
		t.Fatalf("submit location = %q", loc)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/my/onboarding/join", nil)
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
		"acme",
		"viewer, editor",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in pending page, got:\n%s", want, body)
		}
	}
	if strings.Contains(body, `name="org_slug"`) || strings.Contains(body, `name="q"`) {
		t.Fatalf("expected no submit/search form while pending, got:\n%s", body)
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
	pending, err := store.FindPendingJoinRequestByUser(context.Background(), "user-2")
	if err != nil {
		t.Fatalf("FindPendingJoinRequestByUser: %v", err)
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
			t.Fatal("expected non-empty search")
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

	req := httptest.NewRequest(http.MethodGet, "/my/onboarding/join?q=acme&org=acme", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Find an organization",
		"Acme Org",
		`name="roles" value="viewer"`,
		"Submit join request",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in join page, got:\n%s", want, body)
		}
	}
	if strings.Contains(body, `name="roles" value="org-admin"`) {
		t.Fatalf("join role picker must omit org-admin, got:\n%s", body)
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
		"viewer",
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
