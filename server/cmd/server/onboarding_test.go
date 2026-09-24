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

func TestHandleOnboardingUnaffiliatedRendersHub(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-onboarding-hub"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-1",
		Email:          "newbie@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	server := &Server{
		identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/my/onboarding", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Get started",
		`href="/my/onboarding/join"`,
		`href="/my/onboarding/request-organization"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in onboarding hub, got:\n%s", want, body)
		}
	}
	for _, gone := range []string{
		"Pending join request",
		"Pending organization creation request",
		`name="intent"`,
		"Back to streams",
	} {
		if strings.Contains(body, gone) {
			t.Fatalf("hub without pending must not contain %q, got:\n%s", gone, body)
		}
	}
}

func TestHandleOnboardingAffiliatedRedirectsHome(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-onboarding-affiliated"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-2",
		Email:          "member@example.com",
		OrgSlug:        "acme",
		Status:         "active",
		CreatedAt:      now,
	}
	server := &Server{
		identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/my/onboarding", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if rec.Header().Get("Location") != "/my" {
		t.Fatalf("location = %q, want /my", rec.Header().Get("Location"))
	}
}

func TestHandleOnboardingHubPendingAndWithdraw(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-onboarding-pending"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-pending",
		Email:          "pending@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	store := NewMemoryStore()
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	identity.getOrganizationBySlugFunc = func(ctx context.Context, slug string) (*IdentityOrg, error) {
		if strings.TrimSpace(slug) != "acme" {
			return nil, ErrIdentityNotFound
		}
		return &IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org"}, nil
	}
	server := &Server{
		identity:    identity,
		store:       store,
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	if _, err := store.InsertJoinRequest(context.Background(), JoinRequest{
		RequesterUserID: "user-pending",
		RequesterEmail:  "pending@example.com",
		OrgSlug:         "acme",
		RoleSlugs:       []string{"viewer"},
		Status:          AffiliationStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("InsertJoinRequest: %v", err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/my/onboarding", nil)
	getReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	getRec := httptest.NewRecorder()
	server.handleMyRoutes(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("pending hub status = %d, want %d body=%q", getRec.Code, http.StatusOK, getRec.Body.String())
	}
	body := getRec.Body.String()
	for _, want := range []string{
		"Pending join request",
		"Acme Org",
		"acme",
		"viewer",
		`name="intent" value="withdraw"`,
		"Undo",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in pending hub, got:\n%s", want, body)
		}
	}
	for _, gone := range []string{
		`href="/my/onboarding/join"`,
		`href="/my/onboarding/request-organization"`,
		"Back to streams",
	} {
		if strings.Contains(body, gone) {
			t.Fatalf("pending hub must not contain %q, got:\n%s", gone, body)
		}
	}

	childReq := httptest.NewRequest(http.MethodGet, "/my/onboarding/join", nil)
	childReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	childRec := httptest.NewRecorder()
	server.handleMyRoutes(childRec, childReq)
	if childRec.Code != http.StatusSeeOther {
		t.Fatalf("child status = %d, want %d", childRec.Code, http.StatusSeeOther)
	}
	if loc := childRec.Header().Get("Location"); loc != "/my/onboarding" {
		t.Fatalf("child location = %q, want /my/onboarding", loc)
	}

	withdrawReq := httptest.NewRequest(http.MethodPost, "/my/onboarding", strings.NewReader("intent=withdraw"))
	withdrawReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withdrawReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	withdrawRec := httptest.NewRecorder()
	server.handleMyRoutes(withdrawRec, withdrawReq)

	if withdrawRec.Code != http.StatusSeeOther {
		t.Fatalf("withdraw status = %d, want %d body=%q", withdrawRec.Code, http.StatusSeeOther, withdrawRec.Body.String())
	}
	if loc := withdrawRec.Header().Get("Location"); loc != "/my/onboarding" {
		t.Fatalf("withdraw location = %q, want /my/onboarding", loc)
	}
	pending, err := server.affiliationService().PendingJoinRequestForUser(context.Background(), "user-pending")
	if err != nil {
		t.Fatalf("PendingJoinRequestForUser: %v", err)
	}
	if pending != nil {
		t.Fatalf("expected join request withdrawn, got %+v", pending)
	}

	afterReq := httptest.NewRequest(http.MethodGet, "/my/onboarding", nil)
	afterReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	afterRec := httptest.NewRecorder()
	server.handleMyRoutes(afterRec, afterReq)
	if afterRec.Code != http.StatusOK {
		t.Fatalf("after withdraw status = %d body=%q", afterRec.Code, afterRec.Body.String())
	}
	afterBody := afterRec.Body.String()
	if !strings.Contains(afterBody, `href="/my/onboarding/join"`) {
		t.Fatalf("expected join CTA after withdraw, got:\n%s", afterBody)
	}
	if strings.Contains(afterBody, "Pending join request") {
		t.Fatalf("expected no pending panel after withdraw, got:\n%s", afterBody)
	}
}

func TestHandleHomeRedirectsUnaffiliatedToOnboarding(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-home-unaffiliated"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-home",
		Email:          "home@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	server := &Server{
		identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/my", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "/my/onboarding" {
		t.Fatalf("location = %q, want /my/onboarding", loc)
	}
}

func TestHandleOnboardingJoinAndRequestPages(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-onboarding-forms"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-3",
		Email:          "joiner@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	identity.listOrganizationsPageFunc = func(ctx context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error) {
		return IdentityOrgPage{}, nil
	}
	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	t.Run("join form", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/my/onboarding/join", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusOK, rec.Body.String())
		}
		body := rec.Body.String()
		for _, want := range []string{
			"Join an organization",
			`class="breadcrumbs"`,
			">Onboarding<",
			`href="/my/onboarding"`,
			`name="q"`,
			`hx-trigger="input changed delay:200ms, search"`,
			"No organizations yet",
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("expected %q, got:\n%s", want, body)
			}
		}
	})

	t.Run("request organization form", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/my/onboarding/request-organization", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusOK, rec.Body.String())
		}
		body := rec.Body.String()
		for _, want := range []string{
			"Request a new organization",
			"organization creation request",
			`name="name"`,
			"Submit organization creation request",
			`class="breadcrumbs"`,
			">Onboarding<",
			`href="/my/onboarding"`,
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("expected %q, got:\n%s", want, body)
			}
		}
	})
}
