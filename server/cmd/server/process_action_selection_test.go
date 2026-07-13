package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResolveSelectedSubstepIDAndSelectAction(t *testing.T) {
	actions := []SubstepBodyView{
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
	if got := resolveSelectedSubstepID([]SubstepBodyView{{SubstepID: "9.1", Status: "locked"}}, "", false); got != "9.1" {
		t.Fatalf("resolveSelectedSubstepID first fallback = %q, want %q", got, "9.1")
	}
	if got := resolveSelectedSubstepID(nil, "", false); got != "" {
		t.Fatalf("resolveSelectedSubstepID empty list = %q, want empty", got)
	}

	selected, ok := selectedSubstepBody(actions, "1.2", false)
	if !ok || selected.SubstepID != "1.2" {
		t.Fatalf("selectedSubstepBody selected = %#v (ok=%t), want substep 1.2 only", selected, ok)
	}
	selected, ok = selectedSubstepBody(actions, "", false)
	if !ok || selected.SubstepID != "1.1" {
		t.Fatalf("selectedSubstepBody empty selected = %#v (ok=%t), want first action", selected, ok)
	}
	if _, ok := selectedSubstepBody(actions, "404", false); ok {
		t.Fatal("selectedSubstepBody missing selected should return false")
	}
	if _, ok := selectedSubstepBody(actions, "1.2", true); ok {
		t.Fatal("selectedSubstepBody done process should return false")
	}
}

func TestCloneRequestWithSelectedSubstepUpdatesQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/process/123/content?substep=1.1&filter=active", nil)

	cleared := cloneRequestWithSelectedSubstep(req, "")
	if got := cleared.URL.Query().Get("substep"); got != "" {
		t.Fatalf("cleared substep = %q, want empty", got)
	}
	if got := cleared.URL.Query().Get("filter"); got != "active" {
		t.Fatalf("filter query = %q, want %q", got, "active")
	}
	if strings.Contains(cleared.RequestURI, "substep=") {
		t.Fatalf("request uri = %q, want substep removed", cleared.RequestURI)
	}

	selected := cloneRequestWithSelectedSubstep(req, "2.1")
	if got := selected.URL.Query().Get("substep"); got != "2.1" {
		t.Fatalf("selected substep = %q, want %q", got, "2.1")
	}
	if !strings.Contains(selected.RequestURI, "substep=2.1") {
		t.Fatalf("request uri = %q, want updated substep", selected.RequestURI)
	}
}

func TestCloneRequestWithSelectedSubstepHandlesNilURL(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/process/123/content", nil)
	req.URL = nil
	req.RequestURI = ""

	clone := cloneRequestWithSelectedSubstep(req, "1.2")
	if clone.URL != nil {
		t.Fatalf("clone url = %#v, want nil", clone.URL)
	}
	if clone.RequestURI != "" {
		t.Fatalf("request uri = %q, want empty", clone.RequestURI)
	}
}

func TestDecorateTimelineActionsAttachesMatchingSubstepAction(t *testing.T) {
	timeline := []TimelineStep{{
		StepID: "1",
		Substeps: []TimelineSubstep{
			{SubstepID: "1.1", Status: "available"},
			{SubstepID: "1.2", Status: "available"},
		},
	}}
	actions := []SubstepBodyView{
		{SubstepID: "1.2", Title: "Inspect lot", WorkflowKey: "workflow", Status: "available"},
	}

	got := decorateTimelineSubstepBodies(timeline, actions)
	if got[0].Substeps[0].Body != nil {
		t.Fatal("expected unrelated substep action to stay nil")
	}
	if got[0].Substeps[1].Body == nil {
		t.Fatal("expected matching substep action to be attached")
	}
	if got[0].Substeps[1].Body.SubstepID != "1.2" || got[0].Substeps[1].Body.Title != "Inspect lot" {
		t.Fatalf("attached action = %#v", got[0].Substeps[1].Body)
	}
	if got[0].Substeps[1].Status != "available" {
		t.Fatalf("status = %q, want available", got[0].Substeps[1].Status)
	}
}

func TestDecorateTimelineActionsMapsUnauthorizedAvailableToActive(t *testing.T) {
	timeline := []TimelineStep{{
		StepID: "1",
		Substeps: []TimelineSubstep{
			{SubstepID: "1.1", Status: "available"},
		},
	}}
	actions := []SubstepBodyView{
		{SubstepID: "1.1", Status: "available", Disabled: true},
	}

	got := decorateTimelineSubstepBodies(timeline, actions)
	if got[0].Substeps[0].Status != "active" {
		t.Fatalf("status = %q, want active", got[0].Substeps[0].Status)
	}
}

func TestHandleProcessActionsRouteRemoved(t *testing.T) {
	server := &Server{store: NewMemoryStore(), tmpl: testTemplates()}
	req := httptest.NewRequest(http.MethodGet, "/process/507f1f77bcf86cd799439011/actions", nil)
	rec := httptest.NewRecorder()

	server.handleProcessRoutes(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
