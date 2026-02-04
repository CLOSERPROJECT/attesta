package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

func TestHandleNotarizedJSONErrors(t *testing.T) {
	store := NewMemoryStore()
	server := &Server{
		store: store,
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	tests := []struct {
		name      string
		processID string
		storeErr  error
	}{
		{name: "invalid object id", processID: "bad-id"},
		{name: "missing process", processID: primitive.NewObjectID().Hex()},
		{name: "store error", processID: primitive.NewObjectID().Hex(), storeErr: errors.New("boom")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store.LoadProcessErr = tc.storeErr
			req := httptest.NewRequest(http.MethodGet, "/process/"+tc.processID+"/notarized.json", nil)
			rec := httptest.NewRecorder()
			server.handleNotarizedJSON(rec, req, tc.processID)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
			}
		})
	}
}

func TestHandleMerkleJSON(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/process/"+processID.Hex()+"/merkle.json", nil)
	rec := httptest.NewRecorder()
	server.handleMerkleJSON(rec, req, processID.Hex())

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want application/json", got)
	}

	var tree MerkleTree
	if err := json.Unmarshal(rec.Body.Bytes(), &tree); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if tree.Root == "" {
		t.Fatal("expected merkle root to be set")
	}
	if len(tree.Leaves) == 0 || len(tree.Levels) == 0 {
		t.Fatalf("expected non-empty merkle tree, got %#v", tree)
	}
}

func TestHandleMerkleJSONErrors(t *testing.T) {
	store := NewMemoryStore()
	server := &Server{
		store: store,
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	tests := []struct {
		name      string
		processID string
		storeErr  error
	}{
		{name: "invalid object id", processID: "bad-id"},
		{name: "missing process", processID: primitive.NewObjectID().Hex()},
		{name: "store error", processID: primitive.NewObjectID().Hex(), storeErr: errors.New("boom")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store.LoadProcessErr = tc.storeErr
			req := httptest.NewRequest(http.MethodGet, "/process/"+tc.processID+"/merkle.json", nil)
			rec := httptest.NewRecorder()
			server.handleMerkleJSON(rec, req, tc.processID)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
			}
		})
	}
}

func TestHandleDownloadAllFilesZip(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 2, 3, 9, 0, 0, 0, time.UTC)
	processID := primitive.NewObjectID()

	attachment, err := store.SaveAttachment(context.Background(), AttachmentUpload{
		ProcessID:   processID,
		SubstepID:   "1.3",
		Filename:    "../alpha.txt",
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
	if got := rec.Header().Get("Content-Type"); got != "application/zip" {
		t.Fatalf("content-type = %q, want application/zip", got)
	}
	if got := rec.Header().Get("Content-Disposition"); got != `attachment; filename="process-`+processID.Hex()+`-files.zip"` {
		t.Fatalf("content-disposition = %q, want process archive filename", got)
	}

	reader, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("read zip: %v", err)
	}
	if len(reader.File) == 0 {
		t.Fatalf("expected zip entries")
	}
	foundManifest := false
	foundExpectedFile := false
	for _, file := range reader.File {
		if file.Name == "manifest.json" {
			foundManifest = true
			continue
		}
		if file.Name == "1_3-.._alpha.txt" {
			foundExpectedFile = true
		}
	}
	if !foundManifest {
		t.Fatalf("expected manifest.json in zip")
	}
	if !foundExpectedFile {
		t.Fatalf("expected sanitized attachment entry name in zip")
	}
}
