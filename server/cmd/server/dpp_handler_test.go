package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleDigitalLinkDPPHTML(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, tempDir+"/workflow.yaml", "Demo workflow", "string")

	store := NewMemoryStore()
	process := seedDPPProcess(store)
	server := &Server{
		store:      store,
		tmpl:       testTemplates(),
		configDir:  tempDir,
		authorizer: fakeAuthorizer{},
	}

	req := httptest.NewRequest(http.MethodGet, digitalLinkURL(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial), nil)
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "DPP GTIN") || !strings.Contains(body, process.DPP.GTIN) {
		t.Fatalf("expected DPP HTML marker and GTIN, got %q", body)
	}
}

func TestHandleDigitalLinkDPPJSON(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, tempDir+"/workflow.yaml", "Demo workflow", "string")

	store := NewMemoryStore()
	process := seedDPPProcess(store)
	server := &Server{
		store:     store,
		tmpl:      testTemplates(),
		configDir: tempDir,
	}

	req := httptest.NewRequest(http.MethodGet, digitalLinkURL(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial), nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("expected JSON content type, got %q", got)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response JSON: %v", err)
	}
	if payload["digital_link"] == "" {
		t.Fatalf("expected digital_link in response, got %#v", payload)
	}
}

func TestHandleDigitalLinkDPPNotFound(t *testing.T) {
	server := &Server{
		store: NewMemoryStore(),
		tmpl:  testTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/01/09506000134352/10/LOT-001/21/SERIAL-001", nil)
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestHandleDigitalLinkDPPInvalidPath(t *testing.T) {
	server := &Server{
		store: NewMemoryStore(),
		tmpl:  testTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/01/09506000134352/10/LOT-001", nil)
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestHandleDigitalLinkDPPHTMLTemplateIncludesMarkers(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, tempDir+"/workflow.yaml", "Demo workflow", "string")

	store := NewMemoryStore()
	process := seedDPPProcess(store)
	tmpl, err := template.ParseGlob(filepath.Join("..", "..", "templates", "*.html"))
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}
	server := &Server{
		store:     store,
		tmpl:      tmpl,
		configDir: tempDir,
	}

	req := httptest.NewRequest(http.MethodGet, digitalLinkURL(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial), nil)
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "GTIN:") || !strings.Contains(body, "LOT:") || !strings.Contains(body, "SERIAL:") {
		t.Fatalf("expected identifiers in body, got %q", body)
	}
	if !strings.Contains(body, "Merkle root:") {
		t.Fatalf("expected merkle marker in body, got %q", body)
	}
}

func seedDPPProcess(store *MemoryStore) Process {
	process := Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   time.Now().UTC(),
		Status:      "done",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", Data: map[string]interface{}{"value": float64(1)}},
		},
		DPP: &ProcessDPP{
			GTIN:        "09506000134352",
			Lot:         "LOT-001",
			Serial:      "SERIAL-001",
			GeneratedAt: time.Now().UTC(),
		},
	}
	store.SeedProcess(process)
	return process
}
