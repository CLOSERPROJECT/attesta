package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"html/template"
)

func TestHandleHomeListsProcessesAndHistory(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)

	activeID := primitive.NewObjectID()
	active := Process{
		ID:          activeID,
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-2 * time.Hour),
		Status:      "",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(now.Add(-110 * time.Minute))},
			"1_2": {State: "done", DoneAt: ptrTime(now.Add(-100 * time.Minute)), Data: map[string]interface{}{"note": "alpha"}},
			"1_3": {State: "pending"},
			"2_1": {State: "pending"},
			"2_2": {State: "pending"},
			"3_1": {State: "pending"},
			"3_2": {State: "pending"},
		},
	}

	doneID := primitive.NewObjectID()
	done := Process{
		ID:          doneID,
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-1 * time.Hour),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(now.Add(-70 * time.Minute))},
			"1_2": {State: "done", DoneAt: ptrTime(now.Add(-60 * time.Minute))},
			"1_3": {State: "done", DoneAt: ptrTime(now.Add(-50 * time.Minute))},
			"2_1": {State: "done", DoneAt: ptrTime(now.Add(-40 * time.Minute))},
			"2_2": {State: "done", DoneAt: ptrTime(now.Add(-30 * time.Minute))},
			"3_1": {State: "done", DoneAt: ptrTime(now.Add(-20 * time.Minute))},
			"3_2": {State: "done", DoneAt: ptrTime(now.Add(-10 * time.Minute))},
		},
	}

	store.SeedProcess(active)
	store.SeedProcess(done)

	server := &Server{
		store: store,
		tmpl:  homeTestTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/w/workflow/", nil)
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	rec := httptest.NewRecorder()
	server.handleWorkflowHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "PROC 2 HIST 1") {
		t.Fatalf("expected processes and history counts, got %q", body)
	}
	if !strings.Contains(body, activeID.Hex()+":active:28") {
		t.Fatalf("expected active process stats, got %q", body)
	}
	if !strings.Contains(body, doneID.Hex()+":done:100") {
		t.Fatalf("expected done process stats, got %q", body)
	}
	if !strings.Contains(body, "SORT time_desc") {
		t.Fatalf("expected default sort, got %q", body)
	}
	if !strings.Contains(body, "HISTORY "+doneID.Hex()+":done") {
		t.Fatalf("expected history to include only done process, got %q", body)
	}
}

func TestHandleHomeRendersWorkflowPicker(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, tempDir+"/workflow.yaml", "Main workflow", "string", "Main workflow description")
	writeWorkflowConfig(t, tempDir+"/secondary.yaml", "Secondary workflow", "number")

	server := &Server{
		tmpl:      homePickerTemplates(),
		configDir: tempDir,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.handleHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "PICK 2") {
		t.Fatalf("expected picker marker, got %q", body)
	}
	if !strings.Contains(body, "workflow:Main workflow:Main workflow description") || !strings.Contains(body, "secondary:Secondary workflow") {
		t.Fatalf("expected workflow options in picker, got %q", body)
	}
	if strings.Contains(body, "secondary:Secondary workflow:") {
		t.Fatalf("expected optional description to be omitted when empty, got %q", body)
	}
}

func TestNormalizeHomeSortKey(t *testing.T) {
	if got := normalizeHomeSortKey("status"); got != "status" {
		t.Fatalf("expected status, got %q", got)
	}
	if got := normalizeHomeSortKey("unknown"); got != "time_desc" {
		t.Fatalf("expected time_desc for unknown, got %q", got)
	}
}

func TestSortHomeProcessListByStatus(t *testing.T) {
	items := []ProcessListItem{
		{ID: "a", Status: "done", Percent: 100, CreatedAtTime: time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)},
		{ID: "b", Status: "active", Percent: 10, CreatedAtTime: time.Date(2026, 2, 3, 11, 0, 0, 0, time.UTC)},
	}
	sortHomeProcessList(items, "status")
	if items[0].Status != "active" {
		t.Fatalf("expected active first, got %q", items[0].Status)
	}
}

func homeTestTemplates() *template.Template {
	return template.Must(template.New("test").Parse(`
{{define "layout.html"}}{{template "home_body" .}}{{end}}
{{define "home_body"}}
PROC {{len .Processes}} HIST {{len .History}} SORT {{.Sort}}
PROCESSES {{range .Processes}}{{.ID}}:{{.Status}}:{{.Percent}}|{{end}}
HISTORY {{range .History}}{{.ID}}:{{.Status}}|{{end}}
{{end}}
{{define "home.html"}}{{template "layout.html" .}}{{end}}
`))
}

func homePickerTemplates() *template.Template {
	return template.Must(template.New("test").Parse(`
{{define "layout.html"}}{{template "home_picker_body" .}}{{end}}
{{define "home_picker_body"}}PICK {{len .Workflows}} {{range .Workflows}}{{.Key}}:{{.Name}}{{if .Description}}:{{.Description}}{{end}}|{{end}}{{end}}
{{define "home.html"}}{{template "layout.html" .}}{{end}}
`))
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
