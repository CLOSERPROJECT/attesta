package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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
	tmpl := parseTestTemplates(t)
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
	if !strings.Contains(body, "GTIN|") || !strings.Contains(body, "Lot|") || !strings.Contains(body, "Serial|") {
		t.Fatalf("expected identifiers in body, got %q", body)
	}
	if !strings.Contains(body, "Merkle root:") {
		t.Fatalf("expected merkle marker in body, got %q", body)
	}
	if !strings.Contains(body, "Organization 1") {
		t.Fatalf("expected step organization in body, got %q", body)
	}
	if !strings.Contains(body, "5 Mar 2026 at 14:30 UTC") {
		t.Fatalf("expected human-readable completion time in body, got %q", body)
	}
	if !strings.Contains(body, `class="stream-timeline-step-summary"`) {
		t.Fatalf("expected stream timeline step summary in body, got %q", body)
	}
	if !strings.Contains(body, `class="dpp-history-item"`) {
		t.Fatalf("expected dpp history rail wrapper in body, got %q", body)
	}
	if !strings.Contains(body, "<dt>value</dt>") || !strings.Contains(body, "<dd>1</dd>") {
		t.Fatalf("expected inline traceability value in body, got %q", body)
	}
	if strings.Contains(body, ">Documents<") {
		t.Fatalf("expected Documents section removed from body, got %q", body)
	}
	if strings.Contains(body, `class="stream-timeline-step-org-mark"`) {
		t.Fatalf("did not expect org mark on DPP page, got %q", body)
	}
}

func TestHandleDigitalLinkDPPHTMLShowsInlineFileLink(t *testing.T) {
	tempDir := t.TempDir()
	writeFileWorkflowConfig(t, tempDir+"/workflow.yaml")

	store := NewMemoryStore()
	process := Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   time.Now().UTC(),
		Status:      "done",
		Progress: map[string]ProcessStep{
			"1_1": {
				State: "done",
				Data: map[string]interface{}{
					"attachment": map[string]interface{}{
						"attachmentId": "65f2a79b8e7f7d8f3c7c99aa",
						"filename":     "cert.pdf",
						"sha256":       "abc123",
					},
				},
			},
		},
		DPP: &ProcessDPP{
			GTIN:        "09506000134352",
			Lot:         "LOT-001",
			Serial:      "SERIAL-001",
			GeneratedAt: time.Now().UTC(),
		},
	}
	store.SeedProcess(process)
	tmpl := parseTestTemplates(t)
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
	if !strings.Contains(body, "cert.pdf") {
		t.Fatalf("expected inline file link in traceability, got %q", body)
	}
	wantPublicURL := digitalLinkURL(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial) + "/attachment/65f2a79b8e7f7d8f3c7c99aa/file"
	if !strings.Contains(body, wantPublicURL) {
		t.Fatalf("expected public dpp attachment URL %q, got %q", wantPublicURL, body)
	}
	if strings.Contains(body, "/w/workflow/process/") {
		t.Fatalf("expected DPP attachment links not to use authenticated workflow route, got %q", body)
	}
	if strings.Contains(body, ">Documents<") {
		t.Fatalf("expected no Documents section, got %q", body)
	}
}

func TestHandleDigitalLinkDPPAttachmentAllowsPublicDownload(t *testing.T) {
	tempDir := t.TempDir()
	writeFileWorkflowConfig(t, tempDir+"/workflow.yaml")

	store := NewMemoryStore()
	processID := primitive.NewObjectID()
	attachment, err := store.SaveAttachment(t.Context(), AttachmentUpload{
		ProcessID:   processID,
		SubstepID:   "1.1",
		Filename:    "cert.pdf",
		ContentType: "application/pdf",
		MaxBytes:    1024,
		UploadedAt:  time.Now().UTC(),
	}, bytes.NewReader([]byte("certificate")))
	if err != nil {
		t.Fatalf("SaveAttachment: %v", err)
	}
	process := Process{
		ID:          processID,
		WorkflowKey: "workflow",
		CreatedAt:   time.Now().UTC(),
		Status:      "done",
		Progress: map[string]ProcessStep{
			"1_1": {
				State: "done",
				Data: map[string]interface{}{
					"attachment": map[string]interface{}{
						"attachmentId": attachment.ID.Hex(),
						"filename":     attachment.Filename,
						"contentType":  attachment.ContentType,
						"size":         attachment.SizeBytes,
						"sha256":       attachment.SHA256,
					},
				},
			},
		},
		DPP: &ProcessDPP{
			GTIN:        "09506000134352",
			Lot:         "LOT-001",
			Serial:      "SERIAL-001",
			GeneratedAt: time.Now().UTC(),
		},
	}
	store.SeedProcess(process)
	server := &Server{
		store:     store,
		tmpl:      testTemplates(),
		configDir: tempDir,
	}

	req := httptest.NewRequest(http.MethodGet, digitalLinkURL(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial)+"/attachment/"+attachment.ID.Hex()+"/file", nil)
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("content type = %q, want application/pdf", got)
	}
	if rr.Body.String() != "certificate" {
		t.Fatalf("body = %q, want certificate", rr.Body.String())
	}
}

func TestHandleDigitalLinkDPPAttachmentRejectsUnlistedAttachment(t *testing.T) {
	tempDir := t.TempDir()
	writeFileWorkflowConfig(t, tempDir+"/workflow.yaml")

	store := NewMemoryStore()
	process := seedDPPFileProcess(store, primitive.NewObjectID(), "65f2a79b8e7f7d8f3c7c99aa")
	server := &Server{
		store:     store,
		tmpl:      testTemplates(),
		configDir: tempDir,
	}

	req := httptest.NewRequest(http.MethodGet, digitalLinkURL(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial)+"/attachment/"+primitive.NewObjectID().Hex()+"/file", nil)
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestHandleDigitalLinkDPPAttachmentRejectsUnknownDigitalLink(t *testing.T) {
	tempDir := t.TempDir()
	writeFileWorkflowConfig(t, tempDir+"/workflow.yaml")

	server := &Server{
		store:     NewMemoryStore(),
		tmpl:      testTemplates(),
		configDir: tempDir,
	}

	req := httptest.NewRequest(http.MethodGet, "/01/09506000134352/10/LOT-001/21/SERIAL-001/attachment/"+primitive.NewObjectID().Hex()+"/file", nil)
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestHandleDigitalLinkDPPAttachmentRejectsBadStoredAttachmentID(t *testing.T) {
	tempDir := t.TempDir()
	writeFileWorkflowConfig(t, tempDir+"/workflow.yaml")

	store := NewMemoryStore()
	process := seedDPPFileProcess(store, primitive.NewObjectID(), "not-an-object-id")
	server := &Server{
		store:     store,
		tmpl:      testTemplates(),
		configDir: tempDir,
	}

	req := httptest.NewRequest(http.MethodGet, digitalLinkURL(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial)+"/attachment/not-an-object-id/file", nil)
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestHandleDigitalLinkDPPAttachmentRejectsMissingStoredAttachment(t *testing.T) {
	tempDir := t.TempDir()
	writeFileWorkflowConfig(t, tempDir+"/workflow.yaml")

	store := NewMemoryStore()
	attachmentID := primitive.NewObjectID().Hex()
	process := seedDPPFileProcess(store, primitive.NewObjectID(), attachmentID)
	server := &Server{
		store:     store,
		tmpl:      testTemplates(),
		configDir: tempDir,
	}

	req := httptest.NewRequest(http.MethodGet, digitalLinkURL(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial)+"/attachment/"+attachmentID+"/file", nil)
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestHandleDigitalLinkDPPAttachmentRejectsOtherProcessAttachment(t *testing.T) {
	tempDir := t.TempDir()
	writeFileWorkflowConfig(t, tempDir+"/workflow.yaml")

	store := NewMemoryStore()
	processID := primitive.NewObjectID()
	attachment, err := store.SaveAttachment(t.Context(), AttachmentUpload{
		ProcessID:   primitive.NewObjectID(),
		SubstepID:   "1.1",
		Filename:    "cert.pdf",
		ContentType: "application/pdf",
		MaxBytes:    1024,
		UploadedAt:  time.Now().UTC(),
	}, bytes.NewReader([]byte("certificate")))
	if err != nil {
		t.Fatalf("SaveAttachment: %v", err)
	}
	process := seedDPPFileProcess(store, processID, attachment.ID.Hex())
	server := &Server{
		store:     store,
		tmpl:      testTemplates(),
		configDir: tempDir,
	}

	req := httptest.NewRequest(http.MethodGet, digitalLinkURL(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial)+"/attachment/"+attachment.ID.Hex()+"/file", nil)
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestHandleDigitalLinkDPPHTMLShowsPrematureTermination(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, tempDir+"/workflow.yaml", "Demo workflow", "string")

	store := NewMemoryStore()
	process := seedDPPProcess(store)
	endedAt := time.Date(2026, 3, 6, 9, 15, 0, 0, time.UTC)
	process.Termination = &ProcessTermination{
		Reason:    "supplier cancelled",
		EndedAt:   endedAt,
		Actor:     &Actor{ID: "appwrite:user-1", Role: "dep1"},
		SubstepID: "1.2",
	}
	store.SeedProcess(process)
	tmpl := parseTestTemplates(t)
	server := &Server{
		store: store,
		tmpl:  tmpl,
		identity: &fakeIdentityStore{
			getUserByIDFunc: func(ctx context.Context, userID string) (IdentityUser, error) {
				return IdentityUser{ID: userID, Email: "ended@example.com", Status: "active"}, nil
			},
		},
		configDir: tempDir,
	}

	req := httptest.NewRequest(http.MethodGet, digitalLinkURL(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial), nil)
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Stream ended early") || !strings.Contains(body, "supplier cancelled") {
		t.Fatalf("expected premature termination in DPP body, got %q", body)
	}
	if strings.Contains(body, "ended@example.com") || !strings.Contains(body, "user-1") {
		t.Fatalf("expected DPP termination to keep user id, got %q", body)
	}
}

func TestHandleDigitalLinkDPPHTMLRendersOverrideSubstepValues(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, tempDir+"/workflow.yaml", "Demo workflow", "string")

	store := NewMemoryStore()
	process := Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   time.Now().UTC(),
		Status:      "done",
		Progress: map[string]ProcessStep{
			"1_1": {
				State: "done",
				Data:  map[string]interface{}{"value": "override-ok"},
			},
		},
		Overrides: map[string]SubstepOverride{
			"1_1": {
				SubstepID: "1.1",
				Schema:    map[string]interface{}{"type": "object"},
				Reason:    "local source shape",
			},
		},
		DPP: &ProcessDPP{
			GTIN:        "09506000134352",
			Lot:         "LOT-001",
			Serial:      "SERIAL-001",
			GeneratedAt: time.Now().UTC(),
		},
	}
	store.SeedProcess(process)

	tmpl := parseTestTemplates(t)
	server := &Server{
		store:     store,
		tmpl:      tmpl,
		configDir: tempDir,
	}

	req := httptest.NewRequest(http.MethodGet, digitalLinkURL(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial), nil)
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	for _, want := range []string{
		"Completed with local form adaptation.",
		"Reason: local source shape",
		"<dt>value</dt>",
		"<dd>override-ok</dd>",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in DPP override body, got: %s", want, body)
		}
	}
}

func seedDPPProcess(store *MemoryStore) Process {
	doneAt := time.Date(2026, 3, 5, 14, 30, 0, 0, time.UTC)
	process := Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   time.Now().UTC(),
		Status:      "done",
		Progress: map[string]ProcessStep{
			"1_1": {
				State:  "done",
				DoneAt: &doneAt,
				DoneBy: &Actor{ID: "u1", Role: "dep1"},
				Data:   map[string]interface{}{"value": float64(1)},
			},
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

func seedDPPFileProcess(store *MemoryStore, processID primitive.ObjectID, attachmentID string) Process {
	process := Process{
		ID:          processID,
		WorkflowKey: "workflow",
		CreatedAt:   time.Now().UTC(),
		Status:      "done",
		Progress: map[string]ProcessStep{
			"1_1": {
				State: "done",
				Data: map[string]interface{}{
					"attachment": map[string]interface{}{
						"attachmentId": attachmentID,
						"filename":     "cert.pdf",
						"contentType":  "application/pdf",
					},
				},
			},
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

func writeFileWorkflowConfig(t *testing.T, path string) {
	t.Helper()
	content := "workflow:\n" +
		"  name: \"Demo workflow\"\n" +
		"  steps:\n" +
		"    - id: \"1\"\n" +
		"      title: \"Step 1\"\n" +
		"      order: 1\n" +
		"      substeps:\n" +
		"        - id: \"1.1\"\n" +
		"          title: \"Upload\"\n" +
		"          order: 1\n" +
		"          role: \"dep1\"\n" +
		"          inputKey: \"attachment\"\n" +
		"          inputType: \"formata\"\n" +
		"          schema:\n" +
		"            type: object\n" +
		"departments:\n" +
		"  - id: \"dep1\"\n" +
		"    name: \"Department 1\"\n" +
		"users:\n" +
		"  - id: \"u1\"\n" +
		"    name: \"User 1\"\n" +
		"    departmentId: \"dep1\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config %s: %v", path, err)
	}
}
