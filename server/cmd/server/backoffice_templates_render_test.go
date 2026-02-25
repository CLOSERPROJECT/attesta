package main

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackofficeTemplatesRenderWorkflowAndStepTitles(t *testing.T) {
	store := NewMemoryStore()
	activeID, _ := seedBackofficeFixtures(store)
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))
	server := &Server{
		store: store,
		tmpl:  tmpl,
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	dashboardReq := httptest.NewRequest(http.MethodGet, "/backoffice/dep2", nil)
	dashboardReq = dashboardReq.WithContext(context.WithValue(dashboardReq.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	dashboardRec := httptest.NewRecorder()
	server.handleBackoffice(dashboardRec, dashboardReq)
	if dashboardRec.Code != http.StatusOK {
		t.Fatalf("dashboard status = %d, want %d", dashboardRec.Code, http.StatusOK)
	}
	dashboardBody := dashboardRec.Body.String()
	if !strings.Contains(dashboardBody, "Demo workflow - Step 2") {
		t.Fatalf("expected dashboard to include workflow and active step title, got %q", dashboardBody)
	}
	if !strings.Contains(dashboardBody, "Demo workflow - Step 3") {
		t.Fatalf("expected dashboard to include workflow and done step title, got %q", dashboardBody)
	}

	processReq := httptest.NewRequest(http.MethodGet, "/backoffice/dep2/process/"+activeID.Hex(), nil)
	processReq = processReq.WithContext(context.WithValue(processReq.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	processRec := httptest.NewRecorder()
	server.handleBackoffice(processRec, processReq)
	if processRec.Code != http.StatusOK {
		t.Fatalf("process status = %d, want %d", processRec.Code, http.StatusOK)
	}
	processBody := processRec.Body.String()
	if !strings.Contains(processBody, "Demo workflow - Step 2") {
		t.Fatalf("expected process page to include workflow and current step title, got %q", processBody)
	}
}
