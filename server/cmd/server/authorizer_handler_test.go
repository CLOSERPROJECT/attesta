package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleCompleteSubstepAuthorizerAllow(t *testing.T) {
	store := NewMemoryStore()
	server, processID, fixedNow := newServerForCompleteTests(t, store, fakeAuthorizer{
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
	if step.DoneAt == nil || !step.DoneAt.Equal(fixedNow) {
		t.Fatalf("expected deterministic doneAt %s, got %#v", fixedNow, step.DoneAt)
	}
	if len(store.Notarizations()) != 1 {
		t.Fatalf("expected 1 notarization, got %d", len(store.Notarizations()))
	}
	notary := store.Notarizations()[0]
	if !notary.CreatedAt.Equal(fixedNow) {
		t.Fatalf("expected deterministic notarization time %s, got %s", fixedNow, notary.CreatedAt)
	}
}

func TestHandleCompleteSubstepAuthorizerDenyReturns403(t *testing.T) {
	store := NewMemoryStore()
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{
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
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{
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

func newServerForCompleteTests(t *testing.T, store *MemoryStore, authorizer Authorizer) (*Server, string, time.Time) {
	t.Helper()
	fixedNow := time.Date(2026, 2, 2, 14, 0, 0, 0, time.UTC)

	process := Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().UTC(),
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "pending"},
			"1_2": {State: "pending"},
			"1_3": {State: "pending"},
			"2_1": {State: "pending"},
			"2_2": {State: "pending"},
			"3_1": {State: "pending"},
			"3_2": {State: "pending"},
		},
	}
	store.SeedProcess(process)

	server := &Server{
		store:      store,
		tmpl:       testTemplates(),
		authorizer: authorizer,
		sse:        newSSEHub(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
		now: func() time.Time { return fixedNow },
	}
	return server, process.ID.Hex(), fixedNow
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
