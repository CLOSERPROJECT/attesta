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
