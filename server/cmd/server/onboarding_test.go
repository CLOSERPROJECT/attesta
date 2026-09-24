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
		`href="/my"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in onboarding hub, got:\n%s", want, body)
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
			"Find an organization",
			`name="q"`,
			`href="/my/onboarding"`,
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
			`href="/my/onboarding"`,
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("expected %q, got:\n%s", want, body)
			}
		}
	})
}
