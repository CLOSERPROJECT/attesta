package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestLegacyProcessReadRouteRedirectsToWorkflowScopedURL(t *testing.T) {
	store := NewMemoryStore()
	process := Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC),
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "pending"},
		},
	}
	store.SeedProcess(process)

	server := &Server{store: store, configPath: "config/workflow.yaml"}
	req := httptest.NewRequest(http.MethodGet, "/process/"+process.ID.Hex(), nil)
	rec := httptest.NewRecorder()

	server.handleLegacyProcessRoutes(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if got := rec.Header().Get("Location"); got != "/w/workflow/process/"+process.ID.Hex() {
		t.Fatalf("location = %q, want %q", got, "/w/workflow/process/"+process.ID.Hex())
	}
}

func TestLegacyReadRouteReturns404WhenProcessCannotBeResolved(t *testing.T) {
	server := &Server{store: NewMemoryStore(), configPath: "config/workflow.yaml"}
	req := httptest.NewRequest(http.MethodGet, "/process/"+primitive.NewObjectID().Hex(), nil)
	rec := httptest.NewRecorder()

	server.handleLegacyProcessRoutes(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestLegacyBackofficeProcessReadRouteRedirectsToWorkflowScopedURL(t *testing.T) {
	store := NewMemoryStore()
	process := Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC),
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "pending"},
		},
	}
	store.SeedProcess(process)

	server := &Server{store: store, configPath: "config/workflow.yaml"}
	req := httptest.NewRequest(http.MethodGet, "/backoffice/dep1/process/"+process.ID.Hex(), nil)
	rec := httptest.NewRecorder()

	server.handleLegacyBackoffice(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if got := rec.Header().Get("Location"); got != "/w/workflow/backoffice/dep1/process/"+process.ID.Hex() {
		t.Fatalf("location = %q, want %q", got, "/w/workflow/backoffice/dep1/process/"+process.ID.Hex())
	}
}

func TestLegacyMutatingRoutesRequireWorkflowContext(t *testing.T) {
	server := &Server{}

	tests := []struct {
		name string
		path string
		call func(http.ResponseWriter, *http.Request)
	}{
		{
			name: "start process",
			path: "/process/start",
			call: server.handleLegacyStartProcess,
		},
		{
			name: "complete substep",
			path: "/process/" + primitive.NewObjectID().Hex() + "/substep/1.1/complete",
			call: server.handleLegacyProcessRoutes,
		},
		{
			name: "impersonate",
			path: "/impersonate",
			call: server.handleLegacyImpersonate,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, nil)
			rec := httptest.NewRecorder()
			tc.call(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}
