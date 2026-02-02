package main

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleCompleteSubstepAuthorizerAllow(t *testing.T) {
	store := NewMemoryStore()
	server, processID := newServerForCompleteTests(t, store, fakeAuthorizer{
		decide: func(Actor, string, WorkflowSub, int, bool) (bool, error) {
			return true, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value=10"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u1|dep1"})

	rr := httptest.NewRecorder()
	server.handleCompleteSubstep(rr, req, processID, "1.1")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	id, _ := primitive.ObjectIDFromHex(processID)
	process, ok := store.SnapshotProcess(id)
	if !ok {
		t.Fatal("expected process in store")
	}
	step := process.Progress["1_1"]
	if step.State != "done" {
		t.Fatalf("expected substep state done, got %q", step.State)
	}
	if len(store.Notarizations()) != 1 {
		t.Fatalf("expected 1 notarization, got %d", len(store.Notarizations()))
	}
}

func TestHandleCompleteSubstepAuthorizerDenyReturns403(t *testing.T) {
	store := NewMemoryStore()
	server, processID := newServerForCompleteTests(t, store, fakeAuthorizer{
		decide: func(Actor, string, WorkflowSub, int, bool) (bool, error) {
			return false, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value=10"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u1|dep1"})

	rr := httptest.NewRecorder()
	server.handleCompleteSubstep(rr, req, processID, "1.1")

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestHandleCompleteSubstepAuthorizerErrorReturns502(t *testing.T) {
	store := NewMemoryStore()
	server, processID := newServerForCompleteTests(t, store, fakeAuthorizer{
		decide: func(Actor, string, WorkflowSub, int, bool) (bool, error) {
			return false, errors.New("cerbos down")
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value=10"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u1|dep1"})

	rr := httptest.NewRecorder()
	server.handleCompleteSubstep(rr, req, processID, "1.1")

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, rr.Code)
	}
}

func newServerForCompleteTests(t *testing.T, store *MemoryStore, authorizer Authorizer) (*Server, string) {
	t.Helper()
	cfgPath := writeTestConfig(t)
	tmpl := template.Must(template.New("test").Parse(`
{{define "action_list.html"}}{{.Error}}{{end}}
{{define "backoffice_process.html"}}{{.Error}}{{end}}
`))

	process := Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().UTC(),
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "pending"},
			"1_2": {State: "pending"},
			"2_1": {State: "pending"},
			"2_2": {State: "pending"},
			"3_1": {State: "pending"},
			"3_2": {State: "pending"},
		},
	}
	store.SeedProcess(process)

	server := &Server{
		store:      store,
		tmpl:       tmpl,
		authorizer: authorizer,
		sse:        newSSEHub(),
		configPath: cfgPath,
	}
	return server, process.ID.Hex()
}

type fakeAuthorizer struct {
	decide func(actor Actor, processID string, sub WorkflowSub, stepOrder int, sequenceOK bool) (bool, error)
}

func (f fakeAuthorizer) CanComplete(ctx context.Context, actor Actor, processID string, sub WorkflowSub, stepOrder int, sequenceOK bool) (bool, error) {
	if f.decide == nil {
		return true, nil
	}
	return f.decide(actor, processID, sub, stepOrder, sequenceOK)
}
