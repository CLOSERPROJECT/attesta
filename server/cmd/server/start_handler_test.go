package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleStartProcessMethodNotAllowed(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/process/start", nil)
	rr := httptest.NewRecorder()

	server.handleStartProcess(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestHandleStartProcessConfigErrorReturns500(t *testing.T) {
	server := &Server{
		configProvider: func() (RuntimeConfig, error) {
			return RuntimeConfig{}, errors.New("broken config")
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/process/start", nil)
	rr := httptest.NewRecorder()
	server.handleStartProcess(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestHandleStartProcessInsertErrorReturns500(t *testing.T) {
	store := NewMemoryStore()
	store.InsertProcessErr = errors.New("insert failed")
	now := time.Date(2026, 2, 2, 13, 0, 0, 0, time.UTC)
	server := &Server{
		store:         store,
		sse:           newSSEHub(),
		workflowDefID: primitive.NewObjectID(),
		configProvider: func() (RuntimeConfig, error) {
			return startableTestRuntimeConfig(), nil
		},
		now: func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodPost, "/process/start", strings.NewReader("name=Pilot"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	addStartableSession(req, server, now)
	rr := httptest.NewRecorder()
	server.handleStartProcess(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestHandleStartProcessRejectsEmptyName(t *testing.T) {
	store := NewMemoryStore()
	server := &Server{
		store:         store,
		sse:           newSSEHub(),
		workflowDefID: primitive.NewObjectID(),
		configProvider: func() (RuntimeConfig, error) {
			return startableTestRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/process/start", strings.NewReader("name=+++&next=/streams/workflow"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	server.handleStartProcess(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	loc := rr.Header().Get("Location")
	if !strings.Contains(loc, "toast_error=") || !strings.Contains(loc, "Instance+name+is+required") {
		t.Fatalf("location = %q, want toast_error for required name", loc)
	}
	if !strings.HasPrefix(loc, "/streams/workflow") {
		t.Fatalf("location = %q, want return to /streams/workflow", loc)
	}
	processes, err := store.ListRecentProcessesByWorkflow(t.Context(), "workflow", 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(processes) != 0 {
		t.Fatalf("expected no process insert, got %d", len(processes))
	}
}

func TestHandleStartProcessRejectsNonStartableUser(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 2, 2, 13, 0, 0, 0, time.UTC)
	sessionID := "session-non-startable"
	user := AccountUser{
		ID:        primitive.NewObjectID(),
		Email:     "other-role@example.com",
		OrgSlug:   "org1",
		RoleSlugs: []string{"dep2"},
		Status:    "active",
		CreatedAt: now,
	}
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	identity.listOrganizationsFunc = startableTestOrganizations
	server := &Server{
		store:         store,
		sse:           newSSEHub(),
		workflowDefID: primitive.NewObjectID(),
		configProvider: func() (RuntimeConfig, error) {
			return startableTestRuntimeConfig(), nil
		},
		identity:    identity,
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodPost, "/my/streams/workflow/instance/start", strings.NewReader("name=Pilot"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rr := httptest.NewRecorder()
	server.handleStartProcess(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	processes, err := store.ListRecentProcessesByWorkflow(t.Context(), "workflow", 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(processes) != 0 {
		t.Fatalf("expected no process insert, got %d", len(processes))
	}
}

func addStartableSession(req *http.Request, server *Server, now time.Time) {
	sessionID := "session-startable"
	user := AccountUser{
		ID:        primitive.NewObjectID(),
		Email:     "starter@example.com",
		OrgSlug:   "org1",
		RoleSlugs: []string{"dep1"},
		Status:    "active",
		CreatedAt: now,
	}
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	identity.listOrganizationsFunc = startableTestOrganizations
	server.identity = identity
	server.enforceAuth = true
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
}

func startableTestOrganizations(context.Context) ([]IdentityOrg, error) {
	return []IdentityOrg{{
		ID:   "org1",
		Slug: "org1",
		Name: "Organization 1",
		Roles: []IdentityRole{
			{Slug: "dep1", Name: "Department 1"},
			{Slug: "dep2", Name: "Department 2"},
			{Slug: "dep3", Name: "Department 3"},
		},
	}}, nil
}

func startableTestRuntimeConfig() RuntimeConfig {
	cfg := testRuntimeConfig()
	cfg.Workflow.Steps[0].OrganizationSlug = "org1"
	cfg.Organizations = []WorkflowOrganization{{Slug: "org1", Name: "Organization 1"}}
	cfg.Roles = []WorkflowRole{
		{OrgSlug: "org1", Slug: "dep1", Name: "Department 1"},
		{OrgSlug: "org1", Slug: "dep2", Name: "Department 2"},
		{OrgSlug: "org1", Slug: "dep3", Name: "Department 3"},
	}
	return cfg
}
