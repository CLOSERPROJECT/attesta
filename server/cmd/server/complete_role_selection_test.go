package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleCompleteSubstepUsesSelectedActiveRole(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 2, 26, 16, 0, 0, 0, time.UTC)

	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-session-role",
		Email:          "u-session@example.com",
		RoleSlugs:      []string{"dep1", "dep2"},
		Status:         "active",
		CreatedAt:      now,
	}
	sessionID := "session-role"

	process := Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: now,
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done"},
			"1_2": {State: "done"},
			"1_3": {State: "done"},
			"2_1": {State: "pending"},
			"2_2": {State: "pending"},
			"3_1": {State: "pending"},
			"3_2": {State: "pending"},
		},
	}
	store.SeedProcess(process)

	server := &Server{
		store:       store,
		identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
		tmpl:        testTemplates(),
		sse:         newSSEHub(),
		enforceAuth: true,
		now:         func() time.Time { return now },
		authorizer: fakeAuthorizer{
			decide: func(actor Actor, _ string, _ string, _ WorkflowSub, _ int, _ string, _ bool) (bool, error) {
				return actor.Role == "dep2", nil
			},
		},
		configProvider: func() (RuntimeConfig, error) { return testRuntimeConfig(), nil },
	}

	req := httptest.NewRequest(http.MethodPost, "/w/workflow/process/"+process.ID.Hex()+"/substep/2.1/complete", strings.NewReader("value=42&activeRole=dep2"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()

	server.handleCompleteSubstep(rec, req, process.ID.Hex(), "2.1")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	updated, ok := store.SnapshotProcess(process.ID)
	if !ok {
		t.Fatal("expected updated process")
	}
	step := updated.Progress["2_1"]
	if step.DoneBy == nil || step.DoneBy.Role != "dep2" {
		t.Fatalf("doneBy role = %#v, want dep2", step.DoneBy)
	}
	if step.DoneBy.ID != "appwrite:user-session-role" {
		t.Fatalf("doneBy id = %q, want appwrite:user-session-role", step.DoneBy.ID)
	}
}

func TestHandleCompleteSubstepRejectsInvalidActiveRole(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 2, 26, 16, 0, 0, 0, time.UTC)

	user := AccountUser{
		ID:        primitive.NewObjectID(),
		Email:     "u-session@example.com",
		RoleSlugs: []string{"dep1", "dep2"},
		Status:    "active",
		CreatedAt: now,
	}
	sessionID := "session-role"
	process := Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: now,
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done"},
			"1_2": {State: "done"},
			"1_3": {State: "done"},
			"2_1": {State: "pending"},
		},
	}
	store.SeedProcess(process)

	server := &Server{
		store:       store,
		identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
		tmpl:        testTemplates(),
		sse:         newSSEHub(),
		enforceAuth: true,
		now:         func() time.Time { return now },
		authorizer:  fakeAuthorizer{},
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/w/workflow/process/"+process.ID.Hex()+"/substep/2.1/complete", strings.NewReader("value=42&activeRole=dep3"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()

	server.handleCompleteSubstep(rec, req, process.ID.Hex(), "2.1")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}
