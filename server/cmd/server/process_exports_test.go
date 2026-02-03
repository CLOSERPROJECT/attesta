package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleNotarizedJSON(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 2, 3, 9, 0, 0, 0, time.UTC)
	processID := primitive.NewObjectID()
	process := Process{
		ID:        processID,
		CreatedAt: now,
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1_1": {
				State:  "done",
				DoneAt: ptrTime(now.Add(-10 * time.Minute)),
				DoneBy: &Actor{UserID: "u1", Role: "dep1"},
				Data:   map[string]interface{}{"value": 42},
			},
		},
	}
	store.SeedProcess(process)

	server := &Server{
		store: store,
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/process/"+processID.Hex()+"/notarized.json", nil)
	rec := httptest.NewRecorder()
	server.handleNotarizedJSON(rec, req, processID.Hex())

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var export NotarizedProcessExport
	if err := json.Unmarshal(rec.Body.Bytes(), &export); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if export.ProcessID != processID.Hex() {
		t.Fatalf("expected process id %s, got %s", processID.Hex(), export.ProcessID)
	}
	if export.Merkle.Root == "" {
		t.Fatalf("expected merkle root to be set")
	}
	if len(export.Steps) == 0 {
		t.Fatalf("expected steps in export")
	}
}

func TestHandleDownloadAllFilesZip(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 2, 3, 9, 0, 0, 0, time.UTC)
	processID := primitive.NewObjectID()

	attachment, err := store.SaveAttachment(context.Background(), AttachmentUpload{
		ProcessID:   processID,
		SubstepID:   "1.3",
		Filename:    "alpha.txt",
		ContentType: "text/plain",
		MaxBytes:    1 << 20,
		UploadedAt:  now,
	}, bytes.NewReader([]byte("hello world")))
	if err != nil {
		t.Fatalf("save attachment: %v", err)
	}

	process := Process{
		ID:        processID,
		CreatedAt: now,
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1_3": {
				State:  "done",
				DoneAt: ptrTime(now.Add(-5 * time.Minute)),
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
	}
	store.SeedProcess(process)

	server := &Server{
		store: store,
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/process/"+processID.Hex()+"/files.zip", nil)
	rec := httptest.NewRecorder()
	server.handleDownloadAllFiles(rec, req, processID.Hex())

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	reader, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("read zip: %v", err)
	}
	if len(reader.File) == 0 {
		t.Fatalf("expected zip entries")
	}
	foundManifest := false
	foundFile := false
	for _, file := range reader.File {
		if file.Name == "manifest.json" {
			foundManifest = true
			continue
		}
		foundFile = true
	}
	if !foundManifest {
		t.Fatalf("expected manifest.json in zip")
	}
	if !foundFile {
		t.Fatalf("expected attachment file in zip")
	}
}
