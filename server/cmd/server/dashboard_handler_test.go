package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandleDashboardRedirectsWhenUnauthenticated(t *testing.T) {
	server := &Server{
		store:       NewMemoryStore(),
		tmpl:        testTemplates(),
		enforceAuth: true,
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	server.handleDashboard(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if rec.Header().Get("Location") != "/login?next=%2Fdashboard" {
		t.Fatalf("location = %q, want /login?next=%%2Fdashboard", rec.Header().Get("Location"))
	}
}

func TestHandleDashboardRendersForAuthenticatedUser(t *testing.T) {
	store := NewMemoryStore()
	_, _ = seedBackofficeFixtures(store)
	now := time.Date(2026, 2, 26, 17, 0, 0, 0, time.UTC)
	user, err := store.CreateUser(t.Context(), AccountUser{
		UserID:    "u-dashboard",
		Email:     "dashboard@example.com",
		RoleSlugs: []string{"dep2"},
		Status:    "active",
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	session, err := store.CreateSession(t.Context(), Session{
		SessionID:   "dash-session",
		UserID:      user.UserID,
		UserMongoID: user.ID,
		CreatedAt:   now,
		LastLoginAt: now,
		ExpiresAt:   now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}

	server := &Server{
		store:       store,
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: session.SessionID})
	rec := httptest.NewRecorder()
	server.handleDashboard(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if body == "" {
		t.Fatal("expected dashboard body")
	}
	if !strings.Contains(body, "DASHBOARD_ME u-dashboard") {
		t.Fatalf("expected dashboard marker, got %q", body)
	}
	if !strings.Contains(body, "TODO 1") {
		t.Fatalf("expected TODO count, got %q", body)
	}
}

func TestHandleBackofficeRedirectsToDashboardWhenAuthEnabled(t *testing.T) {
	server := &Server{
		store:       NewMemoryStore(),
		tmpl:        testTemplates(),
		enforceAuth: true,
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/w/workflow/backoffice", nil)
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	rec := httptest.NewRecorder()
	server.handleBackoffice(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if rec.Header().Get("Location") != "/w/workflow/dashboard" {
		t.Fatalf("location = %q, want /w/workflow/dashboard", rec.Header().Get("Location"))
	}
}

func TestHandleDashboardPartialRendersBodyOnly(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 2, 26, 17, 0, 0, 0, time.UTC)
	user, err := store.CreateUser(t.Context(), AccountUser{
		UserID:    "u-dashboard-partial",
		Email:     "dashboard-partial@example.com",
		RoleSlugs: []string{"dep2"},
		Status:    "active",
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	session, err := store.CreateSession(t.Context(), Session{
		SessionID:   "dash-session-partial",
		UserID:      user.UserID,
		UserMongoID: user.ID,
		CreatedAt:   now,
		LastLoginAt: now,
		ExpiresAt:   now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}

	server := &Server{
		store:       store,
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/dashboard/partial", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: session.SessionID})
	rec := httptest.NewRecorder()
	server.handleDashboard(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "DASHBOARD_ME u-dashboard-partial") {
		t.Fatalf("expected partial body marker, got %q", body)
	}
	if strings.Contains(body, "NAV Home Backoffice") {
		t.Fatalf("did not expect layout nav in partial, got %q", body)
	}
}

func TestHandleDashboardShowsOrgsLinkForPlatformAdmin(t *testing.T) {
	body := renderDashboardForUser(t, "/dashboard", AccountUser{
		UserID:          "platform-admin-user",
		Email:           "platform-admin@example.com",
		IsPlatformAdmin: true,
		Status:          "active",
	})

	if !strings.Contains(body, "NAV Home Backoffice Orgs |") {
		t.Fatalf("expected orgs nav marker, got %q", body)
	}
	if strings.Contains(body, " MyOrg ") {
		t.Fatalf("did not expect MyOrg marker, got %q", body)
	}
}

func TestHandleDashboardShowsMyOrgLinkForOrgAdmin(t *testing.T) {
	body := renderDashboardForUser(t, "/dashboard", AccountUser{
		UserID:    "org-admin-user",
		Email:     "org-admin@example.com",
		RoleSlugs: []string{"org-admin"},
		OrgSlug:   "nav-test-org",
		Status:    "active",
	})

	if !strings.Contains(body, "NAV Home Backoffice MyOrg |") {
		t.Fatalf("expected my org nav marker, got %q", body)
	}
	if strings.Contains(body, " Orgs ") {
		t.Fatalf("did not expect Orgs marker, got %q", body)
	}
}

func TestHandleDashboardHidesAdminLinksForNonAdminUser(t *testing.T) {
	body := renderDashboardForUser(t, "/dashboard", AccountUser{
		UserID:    "non-admin-user",
		Email:     "non-admin@example.com",
		RoleSlugs: []string{"dep1"},
		Status:    "active",
	})

	if !strings.Contains(body, "NAV Home Backoffice |") {
		t.Fatalf("expected base nav marker, got %q", body)
	}
	if strings.Contains(body, " Orgs ") || strings.Contains(body, " MyOrg ") {
		t.Fatalf("did not expect admin nav markers, got %q", body)
	}
}

func TestHandleDashboardScopedShowsMyOrgLinkForOrgAdmin(t *testing.T) {
	body := renderDashboardForUser(t, "/w/workflow/dashboard", AccountUser{
		UserID:    "scoped-org-admin-user",
		Email:     "scoped-org-admin@example.com",
		RoleSlugs: []string{"org-admin"},
		OrgSlug:   "nav-scoped-org",
		Status:    "active",
	})

	if !strings.Contains(body, "NAV Home Backoffice MyOrg |") {
		t.Fatalf("expected scoped my org nav marker, got %q", body)
	}
}

func TestHandleDashboardShowsBothLinksWhenBothFlagsAreTrue(t *testing.T) {
	body := renderDashboardForUser(t, "/dashboard", AccountUser{
		UserID:          "dual-admin-user",
		Email:           "dual-admin@example.com",
		IsPlatformAdmin: true,
		RoleSlugs:       []string{"org-admin"},
		OrgSlug:         "dual-org",
		Status:          "active",
	})

	if !strings.Contains(body, "NAV Home Backoffice Orgs MyOrg |") {
		t.Fatalf("expected both nav markers in order, got %q", body)
	}
}

func renderDashboardForUser(t *testing.T, path string, user AccountUser) string {
	t.Helper()
	store := NewMemoryStore()
	now := time.Date(2026, 2, 26, 17, 0, 0, 0, time.UTC)

	if user.OrgSlug != "" {
		org, err := store.CreateOrganization(t.Context(), Organization{Name: user.OrgSlug, CreatedAt: now})
		if err != nil {
			t.Fatalf("CreateOrganization error: %v", err)
		}
		user.OrgSlug = org.Slug
		user.OrgID = nil
		for _, roleSlug := range user.RoleSlugs {
			if _, err := store.CreateRole(t.Context(), Role{
				OrgID:     org.ID,
				OrgSlug:   org.Slug,
				Slug:      roleSlug,
				Name:      roleSlug,
				CreatedAt: now,
			}); err != nil {
				t.Fatalf("CreateRole error: %v", err)
			}
		}
	}

	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	createdUser, err := store.CreateUser(t.Context(), user)
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	session, err := store.CreateSession(t.Context(), Session{
		SessionID:   "dash-session-" + createdUser.UserID,
		UserID:      createdUser.UserID,
		UserMongoID: createdUser.ID,
		CreatedAt:   now,
		LastLoginAt: now,
		ExpiresAt:   now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}

	server := &Server{
		store:       store,
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, path, nil)
	if strings.HasPrefix(path, "/w/workflow/") {
		req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
			Key: "workflow",
			Cfg: testRuntimeConfig(),
		}))
	}
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: session.SessionID})
	rec := httptest.NewRecorder()
	server.handleDashboard(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	return rec.Body.String()
}
