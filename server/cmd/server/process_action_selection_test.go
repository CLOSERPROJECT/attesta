package main

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestResolveSelectedSubstepIDAndSelectAction(t *testing.T) {
	actions := []ActionView{
		{SubstepID: "1.1", Status: "locked"},
		{SubstepID: "1.2", Status: "available"},
		{SubstepID: "1.3", Status: "done"},
	}

	if got := resolveSelectedSubstepID(actions, "1.3", false); got != "1.3" {
		t.Fatalf("resolveSelectedSubstepID requested = %q, want %q", got, "1.3")
	}
	if got := resolveSelectedSubstepID(actions, "missing", false); got != "1.2" {
		t.Fatalf("resolveSelectedSubstepID fallback available = %q, want %q", got, "1.2")
	}
	if got := resolveSelectedSubstepID(actions, "", true); got != "" {
		t.Fatalf("resolveSelectedSubstepID done = %q, want empty", got)
	}
	if got := resolveSelectedSubstepID([]ActionView{{SubstepID: "9.1", Status: "locked"}}, "", false); got != "9.1" {
		t.Fatalf("resolveSelectedSubstepID first fallback = %q, want %q", got, "9.1")
	}
	if got := resolveSelectedSubstepID(nil, "", false); got != "" {
		t.Fatalf("resolveSelectedSubstepID empty list = %q, want empty", got)
	}

	selected, ok := selectedActionBySubstep(actions, "1.2", false)
	if !ok || selected.SubstepID != "1.2" {
		t.Fatalf("selectedActionBySubstep selected = %#v (ok=%t), want substep 1.2 only", selected, ok)
	}
	selected, ok = selectedActionBySubstep(actions, "", false)
	if !ok || selected.SubstepID != "1.1" {
		t.Fatalf("selectedActionBySubstep empty selected = %#v (ok=%t), want first action", selected, ok)
	}
	if _, ok := selectedActionBySubstep(actions, "404", false); ok {
		t.Fatal("selectedActionBySubstep missing selected should return false")
	}
	if _, ok := selectedActionBySubstep(actions, "1.2", true); ok {
		t.Fatal("selectedActionBySubstep done process should return false")
	}
}

func TestHandleProcessActionsPartialSelectsRequestedSubstep(t *testing.T) {
	store := NewMemoryStore()
	processID := store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   time.Now().UTC(),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "pending"},
			"1_2": {State: "pending"},
			"1_3": {State: "pending"},
			"2_1": {State: "pending"},
			"2_2": {State: "pending"},
			"3_1": {State: "pending"},
			"3_2": {State: "pending"},
		},
	})
	tmpl := template.Must(template.New("test").Parse(`{{define "action_list.html"}}SEL {{.SelectedSubstepID}} DONE {{.ProcessDone}} ACT {{with .Action}}{{.SubstepID}}|{{end}}{{end}}`))
	server := &Server{
		store: store,
		tmpl:  tmpl,
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/process/"+processID.Hex()+"/actions?substep=1.3", nil)
	rec := httptest.NewRecorder()
	server.handleProcessRoutes(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "SEL 1.3") {
		t.Fatalf("expected selected substep 1.3, got %q", body)
	}
	if !strings.Contains(body, "ACT 1.3|") {
		t.Fatalf("expected only substep 1.3 action, got %q", body)
	}
}

func TestHandleProcessActionsPartialShowsDoneResourcesState(t *testing.T) {
	store := NewMemoryStore()
	processID := store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   time.Now().UTC(),
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
		DPP: &ProcessDPP{
			GTIN:   "09506000134352",
			Lot:    "LOT-001",
			Serial: "SERIAL-001",
		},
	})
	tmpl := template.Must(template.New("test").Parse(`{{define "action_list.html"}}SEL {{.SelectedSubstepID}} DONE {{.ProcessDone}} DPP {{.DPPURL}} ACT {{with .Action}}{{.SubstepID}}|{{end}}{{end}}`))
	server := &Server{
		store: store,
		tmpl:  tmpl,
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/process/"+processID.Hex()+"/actions", nil)
	rec := httptest.NewRecorder()
	server.handleProcessRoutes(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "DONE true") {
		t.Fatalf("expected done state in action view, got %q", body)
	}
	if !strings.Contains(body, "DPP /01/09506000134352/10/LOT-001/21/SERIAL-001") {
		t.Fatalf("expected dpp link in action view, got %q", body)
	}
}

func TestHandleProcessActionsPartialErrorBranches(t *testing.T) {
	t.Run("requires authentication when enabled", func(t *testing.T) {
		server := &Server{
			store:       NewMemoryStore(),
			tmpl:        testTemplates(),
			enforceAuth: true,
			configProvider: func() (RuntimeConfig, error) {
				return testRuntimeConfig(), nil
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/process/"+primitive.NewObjectID().Hex()+"/actions", nil)
		rec := httptest.NewRecorder()
		server.handleProcessRoutes(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
		}
	})

	t.Run("config error", func(t *testing.T) {
		server := &Server{
			store: NewMemoryStore(),
			tmpl:  testTemplates(),
			configProvider: func() (RuntimeConfig, error) {
				return RuntimeConfig{}, context.DeadlineExceeded
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/process/"+primitive.NewObjectID().Hex()+"/actions", nil)
		rec := httptest.NewRecorder()
		server.handleProcessRoutes(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("missing process", func(t *testing.T) {
		server := &Server{
			store: NewMemoryStore(),
			tmpl:  testTemplates(),
			configProvider: func() (RuntimeConfig, error) {
				return testRuntimeConfig(), nil
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/process/"+primitive.NewObjectID().Hex()+"/actions", nil)
		rec := httptest.NewRecorder()
		server.handleProcessRoutes(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("workflow mismatch", func(t *testing.T) {
		store := NewMemoryStore()
		processID := store.SeedProcess(Process{
			ID:          primitive.NewObjectID(),
			WorkflowKey: "other",
			CreatedAt:   time.Now().UTC(),
			Status:      "active",
			Progress:    map[string]ProcessStep{"1_1": {State: "pending"}},
		})
		server := &Server{
			store: store,
			tmpl:  testTemplates(),
			configProvider: func() (RuntimeConfig, error) {
				return testRuntimeConfig(), nil
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/process/"+processID.Hex()+"/actions", nil)
		req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
			Key: "workflow",
			Cfg: testRuntimeConfig(),
		}))
		rec := httptest.NewRecorder()
		server.handleProcessRoutes(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("template error", func(t *testing.T) {
		store := NewMemoryStore()
		processID := store.SeedProcess(Process{
			ID:          primitive.NewObjectID(),
			WorkflowKey: "workflow",
			CreatedAt:   time.Now().UTC(),
			Status:      "active",
			Progress:    map[string]ProcessStep{"1_1": {State: "pending"}},
		})
		tmpl := template.Must(template.New("broken").Parse(`{{define "action_list.html"}}{{template "missing" .}}{{end}}`))
		server := &Server{
			store: store,
			tmpl:  tmpl,
			configProvider: func() (RuntimeConfig, error) {
				return testRuntimeConfig(), nil
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/process/"+processID.Hex()+"/actions", nil)
		rec := httptest.NewRecorder()
		server.handleProcessRoutes(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}
