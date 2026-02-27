package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleProcessRoutesDispatchesEndpoints(t *testing.T) {
	store := NewMemoryStore()
	process := Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Date(2026, 2, 4, 10, 0, 0, 0, time.UTC),
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "pending"},
			"1_2": {State: "pending"},
			"1_3": {State: "pending"},
			"2_1": {State: "pending"},
			"2_2": {State: "pending"},
			"3_1": {State: "pending"},
			"3_2": {State: "pending"},
		},
	}
	store.SeedProcess(process)

	server := &Server{
		store:      store,
		tmpl:       testTemplates(),
		authorizer: fakeAuthorizer{},
		sse:        newSSEHub(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
		now: func() time.Time { return time.Date(2026, 2, 4, 11, 0, 0, 0, time.UTC) },
	}

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{name: "process page", method: http.MethodGet, path: "/process/" + process.ID.Hex(), wantStatus: http.StatusOK, wantBody: "PROCESS"},
		{name: "timeline", method: http.MethodGet, path: "/process/" + process.ID.Hex() + "/timeline", wantStatus: http.StatusOK, wantBody: "TIMELINE"},
		{name: "notarized export", method: http.MethodGet, path: "/process/" + process.ID.Hex() + "/notarized.json", wantStatus: http.StatusOK},
		{name: "merkle export", method: http.MethodGet, path: "/process/" + process.ID.Hex() + "/merkle.json", wantStatus: http.StatusOK},
		{name: "all files zip", method: http.MethodGet, path: "/process/" + process.ID.Hex() + "/files.zip", wantStatus: http.StatusOK},
		{name: "complete substep", method: http.MethodPost, path: "/process/" + process.ID.Hex() + "/substep/1.1/complete", body: "value=10", wantStatus: http.StatusOK, wantBody: "PROCESS_PAGE"},
		{name: "substep file branch", method: http.MethodGet, path: "/process/" + process.ID.Hex() + "/substep/1.3/file", wantStatus: http.StatusNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var bodyReader *strings.Reader
			if tc.body == "" {
				bodyReader = strings.NewReader("")
			} else {
				bodyReader = strings.NewReader(tc.body)
			}
			req := httptest.NewRequest(tc.method, tc.path, bodyReader)
			if tc.method == http.MethodPost {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			rec := httptest.NewRecorder()
			server.handleProcessRoutes(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantBody != "" && !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Fatalf("body = %q, want marker %q", rec.Body.String(), tc.wantBody)
			}
		})
	}
}

func TestHandleProcessRoutesReturns404ForInvalidPaths(t *testing.T) {
	server := &Server{store: NewMemoryStore(), tmpl: testTemplates()}
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "missing process id", method: http.MethodGet, path: "/process/"},
		{name: "wrong method for process page", method: http.MethodPost, path: "/process/" + primitive.NewObjectID().Hex()},
		{name: "wrong method for complete", method: http.MethodGet, path: "/process/" + primitive.NewObjectID().Hex() + "/substep/1.1/complete"},
		{name: "unknown path", method: http.MethodGet, path: "/process/" + primitive.NewObjectID().Hex() + "/unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			server.handleProcessRoutes(rec, req)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
			}
		})
	}
}

func TestHandleWorkflowRoutesDispatchAndUnknownWorkflow(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, tempDir+"/workflow.yaml", "Main workflow", "string")

	store := NewMemoryStore()
	process := Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   time.Date(2026, 2, 4, 10, 0, 0, 0, time.UTC),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "pending"},
		},
	}
	store.SeedProcess(process)

	server := &Server{
		store:      store,
		tmpl:       testTemplates(),
		authorizer: fakeAuthorizer{},
		sse:        newSSEHub(),
		now:        func() time.Time { return time.Date(2026, 2, 4, 11, 0, 0, 0, time.UTC) },
		configDir:  tempDir,
	}

	okReq := httptest.NewRequest(http.MethodGet, "/w/workflow/process/"+process.ID.Hex(), nil)
	okRec := httptest.NewRecorder()
	server.handleWorkflowRoutes(okRec, okReq)
	if okRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", okRec.Code, http.StatusOK)
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/w/missing/process/"+process.ID.Hex(), nil)
	missingRec := httptest.NewRecorder()
	server.handleWorkflowRoutes(missingRec, missingReq)
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", missingRec.Code, http.StatusNotFound)
	}
}

func TestHandleWorkflowRoutesAdditionalBranches(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, tempDir+"/workflow.yaml", "Main workflow", "string")

	server := &Server{
		store:      NewMemoryStore(),
		tmpl:       testTemplates(),
		authorizer: fakeAuthorizer{},
		sse:        newSSEHub(),
		now:        func() time.Time { return time.Date(2026, 2, 4, 11, 0, 0, 0, time.UTC) },
		configDir:  tempDir,
	}

	reqMissingWorkflow := httptest.NewRequest(http.MethodGet, "/w/", nil)
	recMissingWorkflow := httptest.NewRecorder()
	server.handleWorkflowRoutes(recMissingWorkflow, reqMissingWorkflow)
	if recMissingWorkflow.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recMissingWorkflow.Code, http.StatusNotFound)
	}

	reqWorkflowHome := httptest.NewRequest(http.MethodGet, "/w/workflow/", nil)
	recWorkflowHome := httptest.NewRecorder()
	server.handleWorkflowRoutes(recWorkflowHome, reqWorkflowHome)
	if recWorkflowHome.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recWorkflowHome.Code, http.StatusOK)
	}

	reqUnknown := httptest.NewRequest(http.MethodGet, "/w/workflow/unknown", nil)
	recUnknown := httptest.NewRecorder()
	server.handleWorkflowRoutes(recUnknown, reqUnknown)
	if recUnknown.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recUnknown.Code, http.StatusNotFound)
	}

	reqEvents := httptest.NewRequest(http.MethodGet, "/w/workflow/events?role=qa", nil)
	recEvents := httptest.NewRecorder()
	server.handleWorkflowRoutes(recEvents, reqEvents)
	if recEvents.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recEvents.Code, http.StatusBadRequest)
	}
}

func TestHandleWorkflowRoutesRequiresAuthWhenEnabled(t *testing.T) {
	server := &Server{
		store:       NewMemoryStore(),
		tmpl:        testTemplates(),
		enforceAuth: true,
	}

	req := httptest.NewRequest(http.MethodGet, "/w/workflow", nil)
	rec := httptest.NewRecorder()
	server.handleWorkflowRoutes(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
}
