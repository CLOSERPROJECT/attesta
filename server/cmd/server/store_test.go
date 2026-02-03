package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestMemoryStoreUpdateProcessProgressEncodesKey(t *testing.T) {
	store := NewMemoryStore()
	id := store.SeedProcess(Process{Progress: map[string]ProcessStep{}})

	if err := store.UpdateProcessProgress(t.Context(), id, "1.1", ProcessStep{State: "done"}); err != nil {
		t.Fatalf("update progress: %v", err)
	}

	process, ok := store.SnapshotProcess(id)
	if !ok {
		t.Fatal("expected process in memory store")
	}
	if _, ok := process.Progress["1_1"]; !ok {
		t.Fatalf("expected encoded progress key 1_1, got %#v", process.Progress)
	}
}

func TestHandleStartProcessWithMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	fixedNow := time.Date(2026, 2, 2, 13, 0, 0, 0, time.UTC)
	server := &Server{
		store:         store,
		sse:           newSSEHub(),
		now:           func() time.Time { return fixedNow },
		workflowDefID: primitive.NewObjectID(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/process/start", nil)
	rr := httptest.NewRecorder()
	server.handleStartProcess(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}
	location := rr.Header().Get("Location")
	if !strings.HasPrefix(location, "/process/") {
		t.Fatalf("expected redirect location /process/:id, got %q", location)
	}

	processes, err := store.ListRecentProcesses(t.Context(), 10)
	if err != nil {
		t.Fatalf("list processes: %v", err)
	}
	if len(processes) != 1 {
		t.Fatalf("expected one process, got %d", len(processes))
	}
	if len(processes[0].Progress) != 7 {
		t.Fatalf("expected 7 configured substeps in progress map, got %d", len(processes[0].Progress))
	}
	if !processes[0].CreatedAt.Equal(fixedNow) {
		t.Fatalf("expected deterministic createdAt %s, got %s", fixedNow, processes[0].CreatedAt)
	}
	if _, ok := processes[0].Progress["1_1"]; !ok {
		t.Fatalf("expected encoded key 1_1 in progress, got %#v", processes[0].Progress)
	}
}

func TestMemoryStoreAttachmentRoundTrip(t *testing.T) {
	store := NewMemoryStore()
	processID := primitive.NewObjectID()
	payload := []byte("hello attachment")

	attachment, err := store.SaveAttachment(t.Context(), AttachmentUpload{
		ProcessID:   processID,
		SubstepID:   "1.3",
		Filename:    "cert.pdf",
		ContentType: "application/pdf",
		MaxBytes:    1024,
		UploadedAt:  time.Date(2026, 2, 2, 10, 0, 0, 0, time.UTC),
	}, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("save attachment: %v", err)
	}
	if attachment.ID.IsZero() {
		t.Fatal("expected non-empty attachment id")
	}
	if attachment.SizeBytes != int64(len(payload)) {
		t.Fatalf("expected size %d, got %d", len(payload), attachment.SizeBytes)
	}
	if attachment.SHA256 == "" {
		t.Fatal("expected sha256 digest")
	}

	meta, err := store.LoadAttachmentByID(t.Context(), attachment.ID)
	if err != nil {
		t.Fatalf("load attachment metadata: %v", err)
	}
	if meta.Filename != "cert.pdf" {
		t.Fatalf("expected filename cert.pdf, got %q", meta.Filename)
	}

	reader, err := store.OpenAttachmentDownload(t.Context(), attachment.ID)
	if err != nil {
		t.Fatalf("open attachment download: %v", err)
	}
	defer reader.Close()
	downloaded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read attachment content: %v", err)
	}
	if string(downloaded) != string(payload) {
		t.Fatalf("expected content %q, got %q", payload, downloaded)
	}
}

func TestMemoryStoreAttachmentTooLarge(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.SaveAttachment(t.Context(), AttachmentUpload{
		ProcessID: primitive.NewObjectID(),
		SubstepID: "1.3",
		MaxBytes:  4,
	}, bytes.NewReader([]byte("toolarge")))
	if !errors.Is(err, ErrAttachmentTooLarge) {
		t.Fatalf("expected ErrAttachmentTooLarge, got %v", err)
	}
}
