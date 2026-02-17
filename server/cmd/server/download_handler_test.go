package main

import (
	"bytes"
	"context"
	"errors"
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

func TestHandleDownloadProcessAttachmentAllowsAnonymousAccess(t *testing.T) {
	store := NewMemoryStore()
	processID := primitive.NewObjectID()
	fileBytes := []byte("generic-attachment-content")
	attachment, err := store.SaveAttachment(t.Context(), AttachmentUpload{
		ProcessID:   processID,
		SubstepID:   "3.1",
		Filename:    "qa-evidence.txt",
		ContentType: "text/plain",
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

	req := httptest.NewRequest(http.MethodGet, "/process/"+processID.Hex()+"/attachment/"+attachment.ID.Hex()+"/file", nil)
	rr := httptest.NewRecorder()
	server.handleProcessRoutes(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/plain" {
		t.Fatalf("expected content type text/plain, got %q", got)
	}
	if got := rr.Header().Get("Content-Disposition"); !strings.Contains(got, `filename="qa-evidence.txt"`) {
		t.Fatalf("expected content disposition with filename, got %q", got)
	}
	if rr.Body.String() != string(fileBytes) {
		t.Fatalf("expected body %q, got %q", fileBytes, rr.Body.String())
	}
}

func TestHandleDownloadProcessAttachmentReturns404ForProcessMismatch(t *testing.T) {
	store := NewMemoryStore()
	processID := primitive.NewObjectID()
	otherProcessID := primitive.NewObjectID()
	attachment, err := store.SaveAttachment(t.Context(), AttachmentUpload{
		ProcessID:   otherProcessID,
		SubstepID:   "3.1",
		Filename:    "qa-evidence.txt",
		ContentType: "text/plain",
		MaxBytes:    1024,
		UploadedAt:  time.Date(2026, 2, 2, 14, 0, 0, 0, time.UTC),
	}, bytes.NewReader([]byte("wrong process")))
	if err != nil {
		t.Fatalf("save attachment: %v", err)
	}

	store.SeedProcess(Process{
		ID:        processID,
		CreatedAt: time.Now().UTC(),
		Status:    "active",
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

	req := httptest.NewRequest(http.MethodGet, "/process/"+processID.Hex()+"/attachment/"+attachment.ID.Hex()+"/file", nil)
	rr := httptest.NewRecorder()
	server.handleProcessRoutes(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
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

func TestHandleDownloadSubstepFileSanitizesFilenameHeader(t *testing.T) {
	store := NewMemoryStore()
	processID := primitive.NewObjectID()
	attachment, err := store.SaveAttachment(t.Context(), AttachmentUpload{
		ProcessID:   processID,
		SubstepID:   "1.3",
		Filename:    "../evil\"\n.pdf",
		ContentType: "application/pdf",
		MaxBytes:    1024,
		UploadedAt:  time.Date(2026, 2, 2, 14, 0, 0, 0, time.UTC),
	}, bytes.NewReader([]byte("pdf")))
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
	if got := rr.Header().Get("Content-Disposition"); !strings.Contains(got, `filename=".._evil_.pdf"`) {
		t.Fatalf("expected sanitized filename in content disposition, got %q", got)
	}
}

func TestHandleDownloadSubstepFileReturns404ForInvalidOrMissingAttachment(t *testing.T) {
	tests := []struct {
		name         string
		attachmentID string
	}{
		{name: "invalid attachment id", attachmentID: "bad-id"},
		{name: "missing attachment", attachmentID: primitive.NewObjectID().Hex()},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := NewMemoryStore()
			processID := store.SeedProcess(Process{
				ID:        primitive.NewObjectID(),
				CreatedAt: time.Now().UTC(),
				Status:    "active",
				Progress: map[string]ProcessStep{
					"1_3": {
						State: "done",
						Data: map[string]interface{}{
							"attachment": map[string]interface{}{
								"attachmentId": tc.attachmentID,
								"filename":     "missing.pdf",
								"contentType":  "application/pdf",
								"size":         int64(10),
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

			if rr.Code != http.StatusNotFound {
				t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
			}
		})
	}
}

func TestHandleDownloadSubstepFileDefaultsContentTypeWhenMissing(t *testing.T) {
	store := NewMemoryStore()
	processID := primitive.NewObjectID()
	attachment, err := store.SaveAttachment(t.Context(), AttachmentUpload{
		ProcessID:  processID,
		SubstepID:  "1.3",
		Filename:   "payload.bin",
		MaxBytes:   1024,
		UploadedAt: time.Date(2026, 2, 2, 14, 0, 0, 0, time.UTC),
	}, bytes.NewReader([]byte("bin")))
	if err != nil {
		t.Fatalf("save attachment: %v", err)
	}

	store.mu.Lock()
	mem := store.attachments[attachment.ID]
	mem.meta.ContentType = ""
	store.attachments[attachment.ID] = mem
	store.mu.Unlock()

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
	if got := rr.Header().Get("Content-Type"); got != "application/octet-stream" {
		t.Fatalf("content-type = %q, want application/octet-stream", got)
	}
}

func TestHandleDownloadSubstepFileReturns404ForWorkflowMismatch(t *testing.T) {
	store := NewMemoryStore()
	processID := store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "other",
		CreatedAt:   time.Now().UTC(),
		Status:      "active",
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
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	rr := httptest.NewRecorder()
	server.handleProcessRoutes(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestHandleDownloadSubstepFileAdditionalErrorPaths(t *testing.T) {
	t.Run("selected workflow error", func(t *testing.T) {
		server := &Server{
			store: NewMemoryStore(),
			configProvider: func() (RuntimeConfig, error) {
				return RuntimeConfig{}, errors.New("config down")
			},
		}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/process/"+primitive.NewObjectID().Hex()+"/substep/1.3/file", nil)
		server.handleDownloadSubstepFile(rec, req, primitive.NewObjectID().Hex(), "1.3")
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("substep input is not file", func(t *testing.T) {
		store := NewMemoryStore()
		processID := store.SeedProcess(Process{
			ID:          primitive.NewObjectID(),
			WorkflowKey: "workflow",
			CreatedAt:   time.Now().UTC(),
			Status:      "active",
			Progress: map[string]ProcessStep{
				"1_1": {
					State: "done",
					Data:  map[string]interface{}{"value": int64(12)},
				},
			},
		})

		server := &Server{
			store: store,
			configProvider: func() (RuntimeConfig, error) {
				return testRuntimeConfig(), nil
			},
		}
		req := httptest.NewRequest(http.MethodGet, "/process/"+processID.Hex()+"/substep/1.1/file", nil)
		rec := httptest.NewRecorder()
		server.handleDownloadSubstepFile(rec, req, processID.Hex(), "1.1")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("missing attachment payload", func(t *testing.T) {
		store := NewMemoryStore()
		processID := store.SeedProcess(Process{
			ID:          primitive.NewObjectID(),
			WorkflowKey: "workflow",
			CreatedAt:   time.Now().UTC(),
			Status:      "active",
			Progress: map[string]ProcessStep{
				"1_3": {
					State: "done",
					Data:  map[string]interface{}{"other": "value"},
				},
			},
		})

		server := &Server{
			store: store,
			configProvider: func() (RuntimeConfig, error) {
				return testRuntimeConfig(), nil
			},
		}
		req := httptest.NewRequest(http.MethodGet, "/process/"+processID.Hex()+"/substep/1.3/file", nil)
		rec := httptest.NewRecorder()
		server.handleDownloadSubstepFile(rec, req, processID.Hex(), "1.3")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})
}
