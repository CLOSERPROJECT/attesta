package main

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleProcessPageAndTimelineSuccess(t *testing.T) {
	store := NewMemoryStore()
	id := seedProcessWithPending(store)
	server := &Server{
		store: store,
		tmpl:  testTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	pageReq := httptest.NewRequest(http.MethodGet, "/process/"+id.Hex(), nil)
	pageRec := httptest.NewRecorder()
	server.handleProcessRoutes(pageRec, pageReq)
	if pageRec.Code != http.StatusOK {
		t.Fatalf("expected process page status %d, got %d", http.StatusOK, pageRec.Code)
	}
	if !strings.Contains(pageRec.Body.String(), "PROCESS "+id.Hex()) {
		t.Fatalf("expected process marker in page response, got %q", pageRec.Body.String())
	}

	timelineReq := httptest.NewRequest(http.MethodGet, "/process/"+id.Hex()+"/timeline", nil)
	timelineRec := httptest.NewRecorder()
	server.handleProcessRoutes(timelineRec, timelineReq)
	if timelineRec.Code != http.StatusOK {
		t.Fatalf("expected timeline status %d, got %d", http.StatusOK, timelineRec.Code)
	}
	if !strings.Contains(timelineRec.Body.String(), "TIMELINE") {
		t.Fatalf("expected timeline marker in response, got %q", timelineRec.Body.String())
	}
}

func TestHandleProcessPageNotFoundCases(t *testing.T) {
	server := &Server{
		store: NewMemoryStore(),
		tmpl:  testTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	badIDReq := httptest.NewRequest(http.MethodGet, "/process/not-an-id", nil)
	badIDRec := httptest.NewRecorder()
	server.handleProcessRoutes(badIDRec, badIDReq)
	if badIDRec.Code != http.StatusNotFound {
		t.Fatalf("expected bad id status %d, got %d", http.StatusNotFound, badIDRec.Code)
	}

	missingID := primitive.NewObjectID().Hex()
	missingReq := httptest.NewRequest(http.MethodGet, "/process/"+missingID, nil)
	missingRec := httptest.NewRecorder()
	server.handleProcessRoutes(missingRec, missingReq)
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("expected missing process status %d, got %d", http.StatusNotFound, missingRec.Code)
	}
}

func TestHandleTimelineNotFoundCases(t *testing.T) {
	server := &Server{
		store: NewMemoryStore(),
		tmpl:  testTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	badIDReq := httptest.NewRequest(http.MethodGet, "/process/not-an-id/timeline", nil)
	badIDRec := httptest.NewRecorder()
	server.handleProcessRoutes(badIDRec, badIDReq)
	if badIDRec.Code != http.StatusNotFound {
		t.Fatalf("expected bad id status %d, got %d", http.StatusNotFound, badIDRec.Code)
	}

	missingID := primitive.NewObjectID().Hex()
	missingReq := httptest.NewRequest(http.MethodGet, "/process/"+missingID+"/timeline", nil)
	missingRec := httptest.NewRecorder()
	server.handleProcessRoutes(missingRec, missingReq)
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("expected missing process status %d, got %d", http.StatusNotFound, missingRec.Code)
	}
}

func TestHandleProcessPageTemplateErrorReturns500(t *testing.T) {
	store := NewMemoryStore()
	id := seedProcessWithPending(store)
	tmpl := template.Must(template.New("broken").Parse(`{{define "process.html"}}{{template "missing" .}}{{end}}`))
	server := &Server{
		store: store,
		tmpl:  tmpl,
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/process/"+id.Hex(), nil)
	rr := httptest.NewRecorder()
	server.handleProcessRoutes(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestHandleTimelineTemplateErrorReturns500(t *testing.T) {
	store := NewMemoryStore()
	id := seedProcessWithPending(store)
	tmpl := template.Must(template.New("broken").Parse(`{{define "timeline.html"}}{{template "missing" .}}{{end}}`))
	server := &Server{
		store: store,
		tmpl:  tmpl,
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/process/"+id.Hex()+"/timeline", nil)
	rr := httptest.NewRecorder()
	server.handleProcessRoutes(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestHandleProcessRoutesRejectsWorkflowMismatch(t *testing.T) {
	store := NewMemoryStore()
	processID := store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "other",
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

	req := httptest.NewRequest(http.MethodGet, "/process/"+processID.Hex(), nil)
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

func TestHandleProcessPageIncludesDPPLinkWhenPresent(t *testing.T) {
	store := NewMemoryStore()
	processID := store.SeedProcess(Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().UTC(),
		Status:    "done",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done"},
			"1_2": {State: "done"},
			"1_3": {State: "done"},
			"2_1": {State: "done"},
			"2_2": {State: "done"},
			"3_1": {State: "done"},
			"3_2": {State: "done"},
		},
		DPP: &ProcessDPP{
			GTIN:   "09506000134352",
			Lot:    "LOT-001",
			Serial: "SERIAL-001",
		},
	})
	server := &Server{
		store: store,
		tmpl:  testTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/process/"+processID.Hex(), nil)
	rr := httptest.NewRecorder()
	server.handleProcessRoutes(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "/01/") {
		t.Fatalf("expected DPP URL in response, got %q", rr.Body.String())
	}
}

func TestHandleProcessPageRendersDPPLabel(t *testing.T) {
	store := NewMemoryStore()
	processID := store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   time.Now().UTC(),
		Status:      "done",
		Progress:    map[string]ProcessStep{"1_1": {State: "done"}},
		DPP: &ProcessDPP{
			GTIN:   "09506000134352",
			Lot:    "LOT-001",
			Serial: "SERIAL-001",
		},
	})
	tmpl, err := template.ParseGlob(filepath.Join("..", "..", "templates", "*.html"))
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}
	server := &Server{
		store: store,
		tmpl:  tmpl,
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/process/"+processID.Hex(), nil)
	rr := httptest.NewRecorder()
	server.handleProcessRoutes(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, ">DPP<") {
		t.Fatalf("expected DPP link label in response, got %q", body)
	}
}

func TestHandleProcessDownloadsPartialIncludesDPPLink(t *testing.T) {
	store := NewMemoryStore()
	processID := store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   time.Now().UTC(),
		Status:      "done",
		Progress:    map[string]ProcessStep{"1_1": {State: "done"}},
		DPP: &ProcessDPP{
			GTIN:   "09506000134352",
			Lot:    "LOT-001",
			Serial: "SERIAL-001",
		},
	})
	server := &Server{
		store: store,
		tmpl:  testTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/process/"+processID.Hex()+"/downloads", nil)
	rr := httptest.NewRecorder()
	server.handleProcessRoutes(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "/01/") {
		t.Fatalf("expected DPP URL in downloads response, got %q", rr.Body.String())
	}
}

func seedProcessWithPending(store *MemoryStore) primitive.ObjectID {
	process := Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().UTC(),
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
	return store.SeedProcess(process)
}
