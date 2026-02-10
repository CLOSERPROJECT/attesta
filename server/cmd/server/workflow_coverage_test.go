package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleHomeNonRootReturns404(t *testing.T) {
	server := &Server{
		tmpl: testTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	rec := httptest.NewRecorder()
	server.handleHome(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestDefaultWorkflowKeyBranches(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "zeta.yaml"), "Zeta workflow", "string")
	writeWorkflowConfig(t, filepath.Join(tempDir, "alpha.yaml"), "Alpha workflow", "string")

	server := &Server{configDir: tempDir}
	if got := server.defaultWorkflowKey(); got != "alpha" {
		t.Fatalf("defaultWorkflowKey without 'workflow' = %q, want alpha", got)
	}

	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")
	server = &Server{configDir: tempDir}
	if got := server.defaultWorkflowKey(); got != "workflow" {
		t.Fatalf("defaultWorkflowKey with 'workflow' = %q, want workflow", got)
	}

	server = &Server{configDir: "/tmp/custom-workflows"}
	if got := server.defaultWorkflowKey(); got != "custom-workflows" {
		t.Fatalf("defaultWorkflowKey fallback = %q, want custom-workflows", got)
	}
}

func TestHandleWorkflowRoutesDispatchFallbacks(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")

	server := &Server{
		store:     NewMemoryStore(),
		tmpl:      testTemplates(),
		sse:       newSSEHub(),
		configDir: tempDir,
	}

	tests := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{name: "missing key", method: http.MethodGet, path: "/w/", want: http.StatusNotFound},
		{name: "unknown tail", method: http.MethodGet, path: "/w/workflow/unknown", want: http.StatusNotFound},
		{name: "events validation", method: http.MethodGet, path: "/w/workflow/events", want: http.StatusBadRequest},
		{name: "impersonate method guard", method: http.MethodGet, path: "/w/workflow/impersonate", want: http.StatusMethodNotAllowed},
		{name: "start method guard", method: http.MethodGet, path: "/w/workflow/process/start", want: http.StatusMethodNotAllowed},
		{name: "backoffice unknown role", method: http.MethodGet, path: "/w/workflow/backoffice/unknown", want: http.StatusNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			server.handleWorkflowRoutes(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
		})
	}
}

func TestHandleLegacyBackofficePassThrough(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")

	server := &Server{
		tmpl:      testTemplates(),
		configDir: tempDir,
	}
	req := httptest.NewRequest(http.MethodGet, "/backoffice", nil)
	rec := httptest.NewRecorder()
	server.handleLegacyBackoffice(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestProcessExportHandlersReturn404OnWorkflowMismatch(t *testing.T) {
	store := NewMemoryStore()
	processID := store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "secondary",
		CreatedAt:   time.Now().UTC(),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "pending"},
		},
	})
	server := &Server{
		store: store,
		tmpl:  testTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/process/"+processID.Hex()+"/notarized.json", nil)
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))

	tests := []struct {
		name string
		call func(*httptest.ResponseRecorder)
	}{
		{
			name: "notarized",
			call: func(rec *httptest.ResponseRecorder) { server.handleNotarizedJSON(rec, req, processID.Hex()) },
		},
		{
			name: "merkle",
			call: func(rec *httptest.ResponseRecorder) { server.handleMerkleJSON(rec, req, processID.Hex()) },
		},
		{
			name: "zip",
			call: func(rec *httptest.ResponseRecorder) { server.handleDownloadAllFiles(rec, req, processID.Hex()) },
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.call(rec)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
			}
		})
	}
}

func TestHandleTimelinePartialReturns404OnWorkflowMismatch(t *testing.T) {
	store := NewMemoryStore()
	processID := store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "secondary",
		CreatedAt:   time.Now().UTC(),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "pending"},
		},
	})
	server := &Server{
		store: store,
		tmpl:  testTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/process/"+processID.Hex()+"/timeline", nil)
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	rec := httptest.NewRecorder()
	server.handleProcessRoutes(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleDepartmentProcessReturns404OnWorkflowMismatch(t *testing.T) {
	store := NewMemoryStore()
	processID := store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "secondary",
		CreatedAt:   time.Now().UTC(),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "pending"},
		},
	})
	server := &Server{
		store: store,
		tmpl:  testTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/backoffice/dep1/process/"+processID.Hex(), nil)
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	rec := httptest.NewRecorder()
	server.handleBackoffice(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleEventsReturns400OnWorkflowQueryMismatch(t *testing.T) {
	server := &Server{
		sse: newSSEHub(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/events?workflow=secondary&processId=p-1", nil)
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	rec := httptest.NewRecorder()
	server.handleEvents(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGetConfigUsesRuntimeProvider(t *testing.T) {
	server := &Server{
		configProvider: func() (RuntimeConfig, error) {
			cfg := testRuntimeConfig()
			cfg.Workflow.Name = "Provided workflow"
			return cfg, nil
		},
	}
	cfg, err := server.getConfig()
	if err != nil {
		t.Fatalf("getConfig() error: %v", err)
	}
	if cfg.Workflow.Name != "Provided workflow" {
		t.Fatalf("workflow name = %q, want Provided workflow", cfg.Workflow.Name)
	}
}
