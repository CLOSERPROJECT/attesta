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
	"go.mongodb.org/mongo-driver/mongo"
)

func TestMemoryStoreUpdateProcessProgressEncodesKey(t *testing.T) {
	store := NewMemoryStore()
	id := store.SeedProcess(Process{Progress: map[string]ProcessStep{}})

	if err := store.UpdateProcessProgress(t.Context(), id, "workflow", "1.1", ProcessStep{State: "done"}); err != nil {
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
	if !strings.HasPrefix(location, "/w/workflow/process/") {
		t.Fatalf("expected redirect location /w/workflow/process/:id, got %q", location)
	}

	processes, err := store.ListRecentProcessesByWorkflow(t.Context(), "workflow", 10)
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
	if processes[0].WorkflowKey != "workflow" {
		t.Fatalf("workflow key = %q, want workflow", processes[0].WorkflowKey)
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

func TestMemoryStoreListRecentProcessesByWorkflowIsolation(t *testing.T) {
	store := NewMemoryStore()
	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "wf-a",
		CreatedAt:   time.Now().UTC(),
		Status:      "active",
	})
	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "wf-b",
		CreatedAt:   time.Now().UTC(),
		Status:      "active",
	})

	a, err := store.ListRecentProcessesByWorkflow(t.Context(), "wf-a", 10)
	if err != nil {
		t.Fatalf("list workflow wf-a: %v", err)
	}
	if len(a) != 1 || a[0].WorkflowKey != "wf-a" {
		t.Fatalf("unexpected wf-a results: %#v", a)
	}

	b, err := store.ListRecentProcessesByWorkflow(t.Context(), "wf-b", 10)
	if err != nil {
		t.Fatalf("list workflow wf-b: %v", err)
	}
	if len(b) != 1 || b[0].WorkflowKey != "wf-b" {
		t.Fatalf("unexpected wf-b results: %#v", b)
	}
}

func TestMemoryStoreDefaultWorkflowFallbackAndWriteBack(t *testing.T) {
	store := NewMemoryStore()
	id := store.SeedProcess(Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().UTC(),
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "pending"},
		},
	})

	processes, err := store.ListRecentProcessesByWorkflow(t.Context(), "workflow", 10)
	if err != nil {
		t.Fatalf("list fallback: %v", err)
	}
	if len(processes) != 1 {
		t.Fatalf("expected fallback process visibility, got %d", len(processes))
	}
	if processes[0].WorkflowKey != "" {
		t.Fatalf("expected missing workflow key before write-back, got %q", processes[0].WorkflowKey)
	}

	if err := store.UpdateProcessProgress(t.Context(), id, "workflow", "1.1", ProcessStep{State: "done"}); err != nil {
		t.Fatalf("write-back update: %v", err)
	}
	updated, ok := store.SnapshotProcess(id)
	if !ok {
		t.Fatal("expected updated process")
	}
	if updated.WorkflowKey != "workflow" {
		t.Fatalf("expected workflow key write-back, got %q", updated.WorkflowKey)
	}
}

func TestMemoryStoreWorkflowQueriesAndMissingProcessErrors(t *testing.T) {
	store := NewMemoryStore()
	id := store.SeedProcess(Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().UTC(),
		Status:    "active",
		Progress:  map[string]ProcessStep{},
	})

	if _, ok := store.SnapshotProcess(primitive.NewObjectID()); ok {
		t.Fatal("expected missing snapshot lookup to return false")
	}

	if _, err := store.LoadLatestProcessByWorkflow(t.Context(), "secondary"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("LoadLatestProcessByWorkflow mismatch err = %v, want %v", err, mongo.ErrNoDocuments)
	}

	missingID := primitive.NewObjectID()
	if err := store.UpdateProcessProgress(t.Context(), missingID, "workflow", "1.1", ProcessStep{State: "done"}); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("UpdateProcessProgress missing err = %v, want %v", err, mongo.ErrNoDocuments)
	}
	if err := store.UpdateProcessStatus(t.Context(), missingID, "workflow", "done"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("UpdateProcessStatus missing err = %v, want %v", err, mongo.ErrNoDocuments)
	}

	if err := store.UpdateProcessStatus(t.Context(), id, "workflow", "done"); err != nil {
		t.Fatalf("UpdateProcessStatus existing err: %v", err)
	}
	if err := store.UpdateProcessProgress(t.Context(), id, "workflow", "1.1", ProcessStep{State: "done"}); err != nil {
		t.Fatalf("UpdateProcessProgress existing err: %v", err)
	}
	if _, err := store.LoadProcessByDigitalLink(t.Context(), "09506000134352", "lot-a", "serial-a"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("LoadProcessByDigitalLink missing err = %v, want %v", err, mongo.ErrNoDocuments)
	}
}

func TestMemoryStoreMissingAttachmentDownload(t *testing.T) {
	store := NewMemoryStore()
	if _, err := store.OpenAttachmentDownload(t.Context(), primitive.NewObjectID()); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("OpenAttachmentDownload missing err = %v, want %v", err, mongo.ErrNoDocuments)
	}
}

func TestMemoryStoreDigitalLinkRoundTrip(t *testing.T) {
	store := NewMemoryStore()
	processID := store.SeedProcess(Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().UTC(),
		Status:    "done",
		Progress:  map[string]ProcessStep{},
	})

	dpp := ProcessDPP{
		GTIN:        "09506000134352",
		Lot:         "LOT-001",
		Serial:      "SERIAL-001",
		GeneratedAt: time.Now().UTC(),
	}
	if err := store.UpdateProcessDPP(t.Context(), processID, "workflow", dpp); err != nil {
		t.Fatalf("UpdateProcessDPP: %v", err)
	}
	process, err := store.LoadProcessByDigitalLink(t.Context(), dpp.GTIN, dpp.Lot, dpp.Serial)
	if err != nil {
		t.Fatalf("LoadProcessByDigitalLink: %v", err)
	}
	if process.ID != processID {
		t.Fatalf("process id = %s, want %s", process.ID.Hex(), processID.Hex())
	}
	if process.DPP == nil {
		t.Fatal("expected process.DPP to be set")
	}
	if process.DPP.GTIN != dpp.GTIN || process.DPP.Lot != dpp.Lot || process.DPP.Serial != dpp.Serial {
		t.Fatalf("unexpected dpp data: %#v", process.DPP)
	}
}
