package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleDownloadSubstepFileAllowsAnonymousAccess(t *testing.T) {
	store := NewMemoryStore()
	processID := primitive.NewObjectID()
	fileBytes := []byte("binary-file-content")
	attachment, err := store.SaveAttachment(t.Context(), AttachmentUpload{
		ProcessID:   processID,
		SubstepID:   "1.3",
		Filename:    "cert.pdf",
		ContentType: "application/pdf",
		MaxBytes:    1024,
		UploadedAt:  time.Date(2026, 2, 2, 14, 0, 0, 0, time.UTC),
	}, bytes.NewReader(fileBytes))
	if err != nil {
		t.Fatalf("save attachment: %v", err)
	}

	store.SeedProcess(Process{
		ID:        processID,
		CreatedAt: time.Now().UTC(),
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1_3": {
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
	})

	server := &Server{
		store: store,
		tmpl:  testTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/process/"+processID.Hex()+"/substep/1.3/file", nil)
	rr := httptest.NewRecorder()
	server.handleProcessRoutes(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("expected content type application/pdf, got %q", got)
	}
	if got := rr.Header().Get("Content-Disposition"); !strings.Contains(got, `filename="cert.pdf"`) {
		t.Fatalf("expected content disposition with filename, got %q", got)
	}
	if rr.Body.String() != string(fileBytes) {
		t.Fatalf("expected body %q, got %q", fileBytes, rr.Body.String())
	}
}

func TestHandleDownloadSubstepFileReturns404WhenNotDone(t *testing.T) {
	store := NewMemoryStore()
	processID := store.SeedProcess(Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().UTC(),
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1_3": {State: "pending"},
		},
	})

	server := &Server{
		store: store,
		tmpl:  testTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/process/"+processID.Hex()+"/substep/1.3/file", nil)
	rr := httptest.NewRecorder()
	server.handleProcessRoutes(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}
