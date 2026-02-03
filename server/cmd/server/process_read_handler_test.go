package main

import (
	"html/template"
	"net/http"
	"net/http/httptest"
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
