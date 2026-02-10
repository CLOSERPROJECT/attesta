package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestHandleEventsValidation(t *testing.T) {
	server := &Server{
		sse: newSSEHub(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/events", nil)
	missingRec := httptest.NewRecorder()
	server.handleEvents(missingRec, missingReq)
	if missingRec.Code != http.StatusBadRequest {
		t.Fatalf("expected missing query status %d, got %d", http.StatusBadRequest, missingRec.Code)
	}

	unknownRoleReq := httptest.NewRequest(http.MethodGet, "/events?role=unknown", nil)
	unknownRoleRec := httptest.NewRecorder()
	server.handleEvents(unknownRoleRec, unknownRoleReq)
	if unknownRoleRec.Code != http.StatusBadRequest {
		t.Fatalf("expected unknown role status %d, got %d", http.StatusBadRequest, unknownRoleRec.Code)
	}
}

func TestHandleEventsProcessStream(t *testing.T) {
	server := &Server{
		sse: newSSEHub(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	processID := "p-1"
	req := httptest.NewRequest(http.MethodGet, "/events?processId="+processID, nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		server.handleEvents(rr, req)
		close(done)
	}()

	waitForSSESubscriber(t, server.sse, "process:workflow:"+processID)
	server.sse.Broadcast("process:workflow:"+processID, "process-updated")
	cancel()
	waitForHandlerDone(t, done)

	body := rr.Body.String()
	if !strings.Contains(body, "event: process-updated") {
		t.Fatalf("expected process event marker, got %q", body)
	}
	if !strings.Contains(body, "data: process-updated") {
		t.Fatalf("expected process data payload, got %q", body)
	}
}

func TestHandleEventsRoleStream(t *testing.T) {
	server := &Server{
		sse: newSSEHub(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/events?role=dep1", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		server.handleEvents(rr, req)
		close(done)
	}()

	waitForSSESubscriber(t, server.sse, "role:workflow:dep1")
	server.sse.Broadcast("role:workflow:dep1", "role-updated")
	cancel()
	waitForHandlerDone(t, done)

	body := rr.Body.String()
	if !strings.Contains(body, "event: role-updated") {
		t.Fatalf("expected role event marker, got %q", body)
	}
	if !strings.Contains(body, "data: role-updated") {
		t.Fatalf("expected role data payload, got %q", body)
	}
}

func waitForSSESubscriber(t *testing.T, hub *SSEHub, key string) {
	t.Helper()
	for i := 0; i < 1000; i++ {
		hub.mu.Lock()
		count := len(hub.stream[key])
		hub.mu.Unlock()
		if count > 0 {
			return
		}
		runtime.Gosched()
	}
	t.Fatalf("subscriber for key %q was not established", key)
}

func waitForHandlerDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("event handler did not stop after context cancellation")
	}
}
