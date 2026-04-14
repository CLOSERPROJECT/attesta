package main

import (
	"context"
	"errors"
	"fmt"
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
		decide: func(Actor, string, string, WorkflowSub, int, string, bool) (bool, error) {
			return true, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value=%7B%22status%22%3A%22ok%22%7D"))
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
		decide: func(Actor, string, string, WorkflowSub, int, string, bool) (bool, error) {
			return false, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value=%7B%22status%22%3A%22ok%22%7D"))
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
		decide: func(Actor, string, string, WorkflowSub, int, string, bool) (bool, error) {
			return false, errors.New("cerbos down")
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value=%7B%22status%22%3A%22ok%22%7D"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u1|dep1"})

	rr := httptest.NewRecorder()
	server.handleCompleteSubstep(rr, req, processID, "1.1")

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, rr.Code)
	}
}

func TestHandleCompleteSubstepAuthorizerDeniesInvalidActiveRole(t *testing.T) {
	store := NewMemoryStore()
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{
		decide: func(_ Actor, _ string, _ string, _ WorkflowSub, _ int, _ string, _ bool) (bool, error) {
			return true, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value=%7B%22status%22%3A%22ok%22%7D&activeRole=dep2"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	server.handleCompleteSubstep(rr, req, processID, "1.1")

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
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
			return testFormataRuntimeConfig(), nil
		},
		now: func() time.Time { return fixedNow },
	}
	return server, process.ID.Hex(), fixedNow
}

type fakeAuthorizer struct {
	decide       func(actor Actor, processID string, workflowKey string, sub WorkflowSub, stepOrder int, stepOrgSlug string, sequenceOK bool) (bool, error)
	deleteDecide func(user *AccountUser, workflowKey string, createdByUserID string, hasProcesses bool) (bool, error)
	accessDecide func(user *AccountUser, resourceKind, resourceID string, resourceAttr map[string]interface{}, action string) (bool, error)
}

func (f fakeAuthorizer) CanComplete(ctx context.Context, actor Actor, processID string, workflowKey string, sub WorkflowSub, stepOrder int, stepOrgSlug string, sequenceOK bool) (bool, error) {
	if f.decide == nil {
		return true, nil
	}
	return f.decide(actor, processID, workflowKey, sub, stepOrder, stepOrgSlug, sequenceOK)
}

func (f fakeAuthorizer) CanDeleteStream(ctx context.Context, user *AccountUser, workflowKey string, createdByUserID string, hasProcesses bool) (bool, error) {
	if f.deleteDecide == nil {
		return true, nil
	}
	return f.deleteDecide(user, workflowKey, createdByUserID, hasProcesses)
}

func (f fakeAuthorizer) CanAccess(ctx context.Context, user *AccountUser, resourceKind, resourceID string, resourceAttr map[string]interface{}, action string) (bool, error) {
	if f.accessDecide == nil {
		return fakeCanAccessDecision(user, resourceKind, resourceAttr, action), nil
	}
	return f.accessDecide(user, resourceKind, resourceID, resourceAttr, action)
}

func fakeCanAccessDecision(user *AccountUser, resourceKind string, resourceAttr map[string]interface{}, action string) bool {
	if user == nil {
		return false
	}
	switch {
	case resourceKind == cerbosResourcePlatformAdminConsole && action == cerbosActionAccess:
		return user.IsPlatformAdmin
	case resourceKind == cerbosResourceOrgAdminConsole && action == cerbosActionAccess:
		return userIsOrgAdmin(user)
	case resourceKind == cerbosResourceCatalog && action == cerbosActionView:
		return user.IsPlatformAdmin || userIsOrgAdmin(user)
	case resourceKind == cerbosResourceFormataBuilder && action == cerbosActionView:
		return user.IsPlatformAdmin || userIsOrgAdmin(user)
	case resourceKind == cerbosResourceFormataBuilder && action == cerbosActionSave:
		return user.IsPlatformAdmin || userIsOrgAdmin(user)
	case resourceKind == "stream" && action == cerbosActionEdit:
		if user.IsPlatformAdmin {
			return true
		}
		if strings.TrimSpace(formataStreamUserID(user)) != strings.TrimSpace(fmt.Sprint(resourceAttr["createdByUserId"])) {
			return false
		}
		hasProcesses, _ := resourceAttr["hasProcesses"].(bool)
		return !hasProcesses
	case resourceKind == "stream" && action == cerbosActionPurge:
		return user.IsPlatformAdmin
	default:
		return false
	}
}
