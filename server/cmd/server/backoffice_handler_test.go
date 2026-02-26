package main

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleBackofficeRoutes(t *testing.T) {
	store := NewMemoryStore()
	activeID, doneID := seedBackofficeFixtures(store)
	_ = doneID
	server := &Server{
		store: store,
		tmpl:  testTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	landingReq := httptest.NewRequest(http.MethodGet, "/backoffice", nil)
	landingReq = landingReq.WithContext(context.WithValue(landingReq.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	landingRec := httptest.NewRecorder()
	server.handleBackoffice(landingRec, landingReq)
	if landingRec.Code != http.StatusOK {
		t.Fatalf("expected landing status %d, got %d", http.StatusOK, landingRec.Code)
	}
	if !strings.Contains(landingRec.Body.String(), "BACKOFFICE") {
		t.Fatalf("expected landing marker, got %q", landingRec.Body.String())
	}

	dashReq := httptest.NewRequest(http.MethodGet, "/backoffice/dep2", nil)
	dashReq = dashReq.WithContext(context.WithValue(dashReq.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	dashRec := httptest.NewRecorder()
	server.handleBackoffice(dashRec, dashReq)
	if dashRec.Code != http.StatusOK {
		t.Fatalf("expected dashboard status %d, got %d", http.StatusOK, dashRec.Code)
	}
	if !strings.Contains(dashRec.Body.String(), "DASHBOARD dep2") {
		t.Fatalf("expected dep2 dashboard marker, got %q", dashRec.Body.String())
	}
	if !strings.Contains(dashRec.Body.String(), "TODO 1") || !strings.Contains(dashRec.Body.String(), "DONE 1") {
		t.Fatalf("expected dashboard counts for dep2, got %q", dashRec.Body.String())
	}

	partialReq := httptest.NewRequest(http.MethodGet, "/backoffice/dep2/partial", nil)
	partialReq = partialReq.WithContext(context.WithValue(partialReq.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	partialRec := httptest.NewRecorder()
	server.handleBackoffice(partialRec, partialReq)
	if partialRec.Code != http.StatusOK {
		t.Fatalf("expected partial status %d, got %d", http.StatusOK, partialRec.Code)
	}
	if !strings.Contains(partialRec.Body.String(), "DASHBOARD dep2") {
		t.Fatalf("expected dep2 partial marker, got %q", partialRec.Body.String())
	}

	processReq := httptest.NewRequest(http.MethodGet, "/backoffice/dep2/process/"+activeID.Hex(), nil)
	processReq = processReq.WithContext(context.WithValue(processReq.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	processRec := httptest.NewRecorder()
	server.handleBackoffice(processRec, processReq)
	if processRec.Code != http.StatusOK {
		t.Fatalf("expected process view status %d, got %d", http.StatusOK, processRec.Code)
	}
	if !strings.Contains(processRec.Body.String(), "PROCESS_PAGE") {
		t.Fatalf("expected process page marker, got %q", processRec.Body.String())
	}
}

func TestHandleBackofficePickerRendersScopedWorkflowLinks(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string", "Main workflow description")
	writeWorkflowConfig(t, filepath.Join(tempDir, "secondary.yaml"), "Secondary workflow", "number")

	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))
	server := &Server{
		tmpl:      tmpl,
		configDir: tempDir,
	}

	req := httptest.NewRequest(http.MethodGet, "/backoffice", nil)
	rec := httptest.NewRecorder()
	server.handleBackoffice(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if location := rec.Header().Get("Location"); location != "" {
		t.Fatalf("expected no implicit redirect, got location %q", location)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `class="panel landing landing-wide"`) {
		t.Fatalf("expected backoffice picker section to include landing-wide class, got %q", body)
	}
	if !strings.Contains(body, `class="workflow-grid"`) || !strings.Contains(body, `class="workflow-card"`) {
		t.Fatalf("expected workflow card grid markup, got %q", body)
	}
	if !strings.Contains(body, "Main workflow") || !strings.Contains(body, "Secondary workflow") {
		t.Fatalf("expected both workflow labels, got %q", body)
	}
	if !strings.Contains(body, "Main workflow description") {
		t.Fatalf("expected optional description in card content, got %q", body)
	}
	if !strings.Contains(body, `href="/w/workflow/backoffice"`) {
		t.Fatalf("expected scoped backoffice href for workflow key, got %q", body)
	}
	if !strings.Contains(body, `href="/w/secondary/backoffice"`) {
		t.Fatalf("expected scoped backoffice href for secondary key, got %q", body)
	}
	if !strings.Contains(body, "Not started") || !strings.Contains(body, "Started") || !strings.Contains(body, "Terminated") {
		t.Fatalf("expected status labels in cards, got %q", body)
	}
}

func TestHandleBackofficeLandingDoesNotUseLandingWideClass(t *testing.T) {
	server := &Server{
		tmpl: template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html"))),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/backoffice", nil)
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	rec := httptest.NewRecorder()
	server.handleBackoffice(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `class="panel landing"`) {
		t.Fatalf("expected backoffice landing to use landing class, got %q", body)
	}
	if strings.Contains(body, `class="panel landing landing-wide"`) {
		t.Fatalf("expected backoffice landing to keep default width, got %q", body)
	}
}

func TestHandleBackofficeRendersWorkflowPicker(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, tempDir+"/workflow.yaml", "Main workflow", "string", "Main workflow description")
	writeWorkflowConfig(t, tempDir+"/secondary.yaml", "Secondary workflow", "number")

	server := &Server{
		tmpl:      testTemplates(),
		configDir: tempDir,
	}

	req := httptest.NewRequest(http.MethodGet, "/backoffice", nil)
	rec := httptest.NewRecorder()
	server.handleBackoffice(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "BACKOFFICE_PICKER") {
		t.Fatalf("expected picker marker, got %q", body)
	}
	if !strings.Contains(body, "workflow:Main workflow:Main workflow description") || !strings.Contains(body, "secondary:Secondary workflow") {
		t.Fatalf("expected workflow options in picker, got %q", body)
	}
	if strings.Contains(body, "secondary:Secondary workflow:Secondary workflow description:") {
		t.Fatalf("expected optional description to be omitted when empty, got %q", body)
	}
}

func TestHandleBackofficeRendersWorkflowPickerCountsByWorkflow(t *testing.T) {
	tempDir := t.TempDir()
	writeTwoSubstepWorkflowConfig(t, tempDir+"/workflow.yaml", "Main workflow")
	writeTwoSubstepWorkflowConfig(t, tempDir+"/secondary.yaml", "Secondary workflow")

	now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-6 * time.Hour),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "pending"},
			"1_2": {State: "pending"},
		},
	})
	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-5 * time.Hour),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done"},
			"1_2": {State: "pending"},
		},
	})
	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-4 * time.Hour),
		Progress: map[string]ProcessStep{
			"1_1": {State: "done"},
			"1_2": {State: "done"},
		},
	})
	store.SeedProcess(Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: now.Add(-3 * time.Hour),
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "pending"},
			"1_2": {State: "pending"},
		},
	})
	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "secondary",
		CreatedAt:   now.Add(-2 * time.Hour),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done"},
			"1_2": {State: "pending"},
		},
	})
	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "secondary",
		CreatedAt:   now.Add(-1 * time.Hour),
		Progress: map[string]ProcessStep{
			"1_1": {State: "done"},
			"1_2": {State: "done"},
		},
	})

	server := &Server{
		tmpl:      testTemplates(),
		configDir: tempDir,
		store:     store,
	}

	req := httptest.NewRequest(http.MethodGet, "/backoffice", nil)
	rec := httptest.NewRecorder()
	server.handleBackoffice(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "workflow:Main workflow:2/1/1") {
		t.Fatalf("expected workflow counts 2/1/1, got %q", body)
	}
	if !strings.Contains(body, "secondary:Secondary workflow:0/1/1") {
		t.Fatalf("expected secondary counts 0/1/1, got %q", body)
	}
}

func TestHandleBackofficeUnknownRoleAndMissingProcess(t *testing.T) {
	server := &Server{
		store: NewMemoryStore(),
		tmpl:  testTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	unknownReq := httptest.NewRequest(http.MethodGet, "/backoffice/unknown", nil)
	unknownReq = unknownReq.WithContext(context.WithValue(unknownReq.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	unknownRec := httptest.NewRecorder()
	server.handleBackoffice(unknownRec, unknownReq)
	if unknownRec.Code != http.StatusNotFound {
		t.Fatalf("expected unknown role status %d, got %d", http.StatusNotFound, unknownRec.Code)
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/backoffice/dep1/process/"+primitive.NewObjectID().Hex(), nil)
	missingReq = missingReq.WithContext(context.WithValue(missingReq.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	missingRec := httptest.NewRecorder()
	server.handleBackoffice(missingRec, missingReq)
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("expected missing process status %d, got %d", http.StatusNotFound, missingRec.Code)
	}
}

func TestBackofficeHandlersErrorPaths(t *testing.T) {
	t.Run("backoffice selected workflow error", func(t *testing.T) {
		server := &Server{
			tmpl: testTemplates(),
			configProvider: func() (RuntimeConfig, error) {
				return RuntimeConfig{}, errors.New("config down")
			},
		}
		req := httptest.NewRequest(http.MethodGet, "/backoffice", nil)
		rec := httptest.NewRecorder()
		server.handleBackoffice(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("department selected workflow errors", func(t *testing.T) {
		server := &Server{
			tmpl: testTemplates(),
			configProvider: func() (RuntimeConfig, error) {
				return RuntimeConfig{}, errors.New("config down")
			},
		}
		req := httptest.NewRequest(http.MethodGet, "/backoffice/dep1", nil)

		dashRec := httptest.NewRecorder()
		server.handleDepartmentDashboard(dashRec, req, "dep1")
		if dashRec.Code != http.StatusInternalServerError {
			t.Fatalf("dashboard status = %d, want %d", dashRec.Code, http.StatusInternalServerError)
		}

		partialRec := httptest.NewRecorder()
		server.handleDepartmentDashboardPartial(partialRec, req, "dep1")
		if partialRec.Code != http.StatusInternalServerError {
			t.Fatalf("partial status = %d, want %d", partialRec.Code, http.StatusInternalServerError)
		}

		processRec := httptest.NewRecorder()
		server.handleDepartmentProcess(processRec, req, "dep1", primitive.NewObjectID().Hex())
		if processRec.Code != http.StatusInternalServerError {
			t.Fatalf("process status = %d, want %d", processRec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("department template errors", func(t *testing.T) {
		store := NewMemoryStore()
		processID := store.SeedProcess(Process{
			ID:          primitive.NewObjectID(),
			WorkflowKey: "workflow",
			CreatedAt:   time.Now().UTC(),
			Status:      "active",
			Progress: map[string]ProcessStep{
				"1_1": {State: "pending"},
			},
		})
		server := &Server{
			store: store,
			tmpl:  template.Must(template.New("broken").Parse(`{{define "other"}}x{{end}}`)),
		}
		req := httptest.NewRequest(http.MethodGet, "/backoffice/dep1", nil)
		req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
			Key: "workflow",
			Cfg: testRuntimeConfig(),
		}))

		dashRec := httptest.NewRecorder()
		server.handleDepartmentDashboard(dashRec, req, "dep1")
		if dashRec.Code != http.StatusInternalServerError {
			t.Fatalf("dashboard status = %d, want %d", dashRec.Code, http.StatusInternalServerError)
		}

		partialRec := httptest.NewRecorder()
		server.handleDepartmentDashboardPartial(partialRec, req, "dep1")
		if partialRec.Code != http.StatusInternalServerError {
			t.Fatalf("partial status = %d, want %d", partialRec.Code, http.StatusInternalServerError)
		}

		processRec := httptest.NewRecorder()
		server.handleDepartmentProcess(processRec, req, "dep1", processID.Hex())
		if processRec.Code != http.StatusInternalServerError {
			t.Fatalf("process status = %d, want %d", processRec.Code, http.StatusInternalServerError)
		}
	})
}

func seedBackofficeFixtures(store *MemoryStore) (primitive.ObjectID, primitive.ObjectID) {
	active := Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   time.Now().UTC().Add(-5 * time.Minute),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done"},
			"1_2": {State: "done"},
			"1_3": {State: "done"},
			"2_1": {State: "pending"},
			"2_2": {State: "pending"},
			"3_1": {State: "pending"},
			"3_2": {State: "pending"},
		},
	}
	done := Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   time.Now().UTC().Add(-10 * time.Minute),
		Status:      "done",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done"},
			"1_2": {State: "done"},
			"1_3": {State: "done"},
			"2_1": {State: "done"},
			"2_2": {State: "done"},
			"3_1": {State: "done"},
			"3_2": {State: "done"},
		},
	}
	store.SeedProcess(active)
	store.SeedProcess(done)
	return active.ID, done.ID
}
