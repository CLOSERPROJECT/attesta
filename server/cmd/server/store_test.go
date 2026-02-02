package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	cfgPath := writeTestConfig(t)
	store := NewMemoryStore()
	server := &Server{
		store:         store,
		sse:           newSSEHub(),
		workflowDefID: primitive.NewObjectID(),
		configPath:    cfgPath,
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
	if _, ok := processes[0].Progress["1_1"]; !ok {
		t.Fatalf("expected encoded key 1_1 in progress, got %#v", processes[0].Progress)
	}
}

func writeTestConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "workflow.yaml")
	content := `workflow:
  name: Demo workflow
  steps:
    - id: "1"
      title: Step 1
      order: 1
      substeps:
        - id: "1.1"
          title: A
          order: 1
          role: dep1
          inputKey: value
          inputType: number
        - id: "1.2"
          title: B
          order: 2
          role: dep1
          inputKey: note
          inputType: text
    - id: "2"
      title: Step 2
      order: 2
      substeps:
        - id: "2.1"
          title: C
          order: 1
          role: dep2
          inputKey: value
          inputType: number
        - id: "2.2"
          title: D
          order: 2
          role: dep2
          inputKey: note
          inputType: text
    - id: "3"
      title: Step 3
      order: 3
      substeps:
        - id: "3.1"
          title: E
          order: 1
          role: dep3
          inputKey: value
          inputType: number
        - id: "3.2"
          title: F
          order: 2
          role: dep3
          inputKey: note
          inputType: text
departments:
  - id: dep1
    name: Department 1
  - id: dep2
    name: Department 2
  - id: dep3
    name: Department 3
users:
  - id: u1
    name: User 1
    departmentId: dep1
  - id: u2
    name: User 2
    departmentId: dep2
  - id: u3
    name: User 3
    departmentId: dep3
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
