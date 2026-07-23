package main

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

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

	req := httptest.NewRequest(http.MethodGet, "/instance/"+processID.Hex()+"/attachment/"+attachment.ID.Hex()+"/file", nil)
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

	req := httptest.NewRequest(http.MethodGet, "/instance/"+processID.Hex()+"/attachment/"+attachment.ID.Hex()+"/file", nil)
	rr := httptest.NewRecorder()
	server.handleProcessRoutes(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestHandleDownloadProcessAttachmentSupportsInlinePreview(t *testing.T) {
	store := NewMemoryStore()
	processID := primitive.NewObjectID()
	fileBytes := []byte("%PDF-1.4 preview")
	attachment, err := store.SaveAttachment(t.Context(), AttachmentUpload{
		ProcessID:   processID,
		SubstepID:   "1.3",
		Filename:    "qa-evidence.pdf",
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
			"1_3": {State: "done"},
		},
	})

	server := &Server{
		store: store,
		tmpl:  testTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/instance/"+processID.Hex()+"/attachment/"+attachment.ID.Hex()+"/file?inline=1", nil)
	rr := httptest.NewRecorder()
	server.handleProcessRoutes(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if got := rr.Header().Get("Content-Disposition"); !strings.HasPrefix(got, "inline;") {
		t.Fatalf("expected inline content disposition, got %q", got)
	}
}

func TestHandleDownloadProcessAttachmentErrorBranches(t *testing.T) {
	t.Run("config error", func(t *testing.T) {
		server := &Server{
			store: NewMemoryStore(),
			configProvider: func() (RuntimeConfig, error) {
				return RuntimeConfig{}, errors.New("config down")
			},
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		server.handleDownloadProcessAttachment(rec, req, primitive.NewObjectID().Hex(), primitive.NewObjectID().Hex())
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("invalid process id", func(t *testing.T) {
		server := &Server{
			store: NewMemoryStore(),
			configProvider: func() (RuntimeConfig, error) {
				return testRuntimeConfig(), nil
			},
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		server.handleDownloadProcessAttachment(rec, req, "bad-id", primitive.NewObjectID().Hex())
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("workflow mismatch", func(t *testing.T) {
		store := NewMemoryStore()
		processID := primitive.NewObjectID()
		store.SeedProcess(Process{
			ID:          processID,
			WorkflowKey: "other",
			CreatedAt:   time.Now().UTC(),
			Progress:    map[string]ProcessStep{"1_1": {State: "pending"}},
		})
		server := &Server{
			store: store,
			configProvider: func() (RuntimeConfig, error) {
				return testRuntimeConfig(), nil
			},
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		server.handleDownloadProcessAttachment(rec, req, processID.Hex(), primitive.NewObjectID().Hex())
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("invalid attachment id", func(t *testing.T) {
		store := NewMemoryStore()
		processID := primitive.NewObjectID()
		store.SeedProcess(Process{
			ID:        processID,
			CreatedAt: time.Now().UTC(),
			Progress:  map[string]ProcessStep{"1_1": {State: "pending"}},
		})
		server := &Server{
			store: store,
			configProvider: func() (RuntimeConfig, error) {
				return testRuntimeConfig(), nil
			},
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		server.handleDownloadProcessAttachment(rec, req, processID.Hex(), "bad-id")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("missing attachment metadata", func(t *testing.T) {
		store := NewMemoryStore()
		processID := primitive.NewObjectID()
		store.SeedProcess(Process{
			ID:        processID,
			CreatedAt: time.Now().UTC(),
			Progress:  map[string]ProcessStep{"1_1": {State: "pending"}},
		})
		server := &Server{
			store: store,
			configProvider: func() (RuntimeConfig, error) {
				return testRuntimeConfig(), nil
			},
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		server.handleDownloadProcessAttachment(rec, req, processID.Hex(), primitive.NewObjectID().Hex())
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})
}

func TestHandleDownloadProcessAttachmentDefaultsContentType(t *testing.T) {
	store := NewMemoryStore()
	processID := primitive.NewObjectID()
	attachment, err := store.SaveAttachment(t.Context(), AttachmentUpload{
		ProcessID:   processID,
		SubstepID:   "3.1",
		Filename:    "raw.bin",
		ContentType: "application/octet-stream",
		MaxBytes:    1024,
		UploadedAt:  time.Date(2026, 2, 2, 14, 0, 0, 0, time.UTC),
	}, bytes.NewReader([]byte("raw")))
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
		Progress:  map[string]ProcessStep{"1_1": {State: "pending"}},
	})
	server := &Server{
		store: store,
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.handleDownloadProcessAttachment(rec, req, processID.Hex(), attachment.ID.Hex())
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/octet-stream" {
		t.Fatalf("content-type = %q, want application/octet-stream", got)
	}
}
