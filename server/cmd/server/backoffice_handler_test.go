package main

import (
	"net/http"
	"net/http/httptest"
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
	landingRec := httptest.NewRecorder()
	server.handleBackoffice(landingRec, landingReq)
	if landingRec.Code != http.StatusOK {
		t.Fatalf("expected landing status %d, got %d", http.StatusOK, landingRec.Code)
	}
	if !strings.Contains(landingRec.Body.String(), "BACKOFFICE") {
		t.Fatalf("expected landing marker, got %q", landingRec.Body.String())
	}

	dashReq := httptest.NewRequest(http.MethodGet, "/backoffice/dep2", nil)
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

	cookies := dashRec.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != "demo_user" || !strings.Contains(cookies[0].Value, "|dep2") {
		t.Fatalf("expected normalized dep2 cookie, got %#v", cookies)
	}

	partialReq := httptest.NewRequest(http.MethodGet, "/backoffice/dep2/partial", nil)
	partialRec := httptest.NewRecorder()
	server.handleBackoffice(partialRec, partialReq)
	if partialRec.Code != http.StatusOK {
		t.Fatalf("expected partial status %d, got %d", http.StatusOK, partialRec.Code)
	}
	if !strings.Contains(partialRec.Body.String(), "DASHBOARD dep2") {
		t.Fatalf("expected dep2 partial marker, got %q", partialRec.Body.String())
	}

	processReq := httptest.NewRequest(http.MethodGet, "/backoffice/dep2/process/"+activeID.Hex(), nil)
	processRec := httptest.NewRecorder()
	server.handleBackoffice(processRec, processReq)
	if processRec.Code != http.StatusOK {
		t.Fatalf("expected process view status %d, got %d", http.StatusOK, processRec.Code)
	}
	if !strings.Contains(processRec.Body.String(), "PROCESS_PAGE") {
		t.Fatalf("expected process page marker, got %q", processRec.Body.String())
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
	unknownRec := httptest.NewRecorder()
	server.handleBackoffice(unknownRec, unknownReq)
	if unknownRec.Code != http.StatusNotFound {
		t.Fatalf("expected unknown role status %d, got %d", http.StatusNotFound, unknownRec.Code)
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/backoffice/dep1/process/"+primitive.NewObjectID().Hex(), nil)
	missingRec := httptest.NewRecorder()
	server.handleBackoffice(missingRec, missingReq)
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("expected missing process status %d, got %d", http.StatusNotFound, missingRec.Code)
	}
}

func seedBackofficeFixtures(store *MemoryStore) (primitive.ObjectID, primitive.ObjectID) {
	active := Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().UTC().Add(-5 * time.Minute),
		Status:    "active",
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
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().UTC().Add(-10 * time.Minute),
		Status:    "done",
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
