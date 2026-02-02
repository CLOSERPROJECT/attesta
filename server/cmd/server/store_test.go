package main

import (
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
	if len(processes[0].Progress) != 6 {
		t.Fatalf("expected 6 configured substeps in progress map, got %d", len(processes[0].Progress))
	}
	if !processes[0].CreatedAt.Equal(fixedNow) {
		t.Fatalf("expected deterministic createdAt %s, got %s", fixedNow, processes[0].CreatedAt)
	}
	if _, ok := processes[0].Progress["1_1"]; !ok {
		t.Fatalf("expected encoded key 1_1 in progress, got %#v", processes[0].Progress)
	}
}
