package main

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestResolveSelectedSubstepIDAndFilterActions(t *testing.T) {
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

	filtered := filterActionsBySubstep(actions, "1.2", false)
	if len(filtered) != 1 || filtered[0].SubstepID != "1.2" {
		t.Fatalf("filterActionsBySubstep selected = %#v, want substep 1.2 only", filtered)
	}
	filtered = filterActionsBySubstep(actions, "", false)
	if len(filtered) != len(actions) {
		t.Fatalf("filterActionsBySubstep empty selected len = %d, want %d", len(filtered), len(actions))
	}
	filtered = filterActionsBySubstep(actions, "404", false)
	if len(filtered) != 0 {
		t.Fatalf("filterActionsBySubstep missing selected = %#v, want empty", filtered)
	}
	filtered = filterActionsBySubstep(actions, "1.2", true)
	if len(filtered) != 0 {
		t.Fatalf("filterActionsBySubstep done process = %#v, want empty", filtered)
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
	tmpl := template.Must(template.New("test").Parse(`{{define "action_list.html"}}SEL {{.SelectedSubstepID}} DONE {{.ProcessDone}} ACT {{range .Actions}}{{.SubstepID}}|{{end}}{{end}}`))
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
	tmpl := template.Must(template.New("test").Parse(`{{define "action_list.html"}}SEL {{.SelectedSubstepID}} DONE {{.ProcessDone}} DPP {{.DPPURL}} ACT {{range .Actions}}{{.SubstepID}}|{{end}}{{end}}`))
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
