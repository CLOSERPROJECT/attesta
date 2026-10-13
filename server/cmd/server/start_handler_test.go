package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleStartProcessMethodNotAllowed(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/process/start", nil)
	rr := httptest.NewRecorder()

	server.handleStartProcess(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestHandleStartProcessConfigErrorReturns500(t *testing.T) {
	server := &Server{
		configProvider: func() (RuntimeConfig, error) {
			return RuntimeConfig{}, errors.New("broken config")
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/process/start", nil)
	rr := httptest.NewRecorder()
	server.handleStartProcess(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestHandleStartProcessInsertErrorReturns500(t *testing.T) {
	store := NewMemoryStore()
	store.InsertProcessErr = errors.New("insert failed")
	server := &Server{
		store:         store,
		sse:           newSSEHub(),
		workflowDefID: primitive.NewObjectID(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/process/start", strings.NewReader("name=Pilot"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	server.handleStartProcess(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestHandleStartProcessRejectsEmptyName(t *testing.T) {
	store := NewMemoryStore()
	server := &Server{
		store:         store,
		sse:           newSSEHub(),
		workflowDefID: primitive.NewObjectID(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/process/start", strings.NewReader("name=+++&next=/streams/workflow"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	server.handleStartProcess(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	loc := rr.Header().Get("Location")
	if !strings.Contains(loc, "toast_error=") || !strings.Contains(loc, "Instance+name+is+required") {
		t.Fatalf("location = %q, want toast_error for required name", loc)
	}
	if !strings.HasPrefix(loc, "/streams/workflow") {
		t.Fatalf("location = %q, want return to /streams/workflow", loc)
	}
	processes, err := store.ListRecentProcessesByWorkflow(t.Context(), "workflow", 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(processes) != 0 {
		t.Fatalf("expected no process insert, got %d", len(processes))
	}
}
