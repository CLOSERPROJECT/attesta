package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestRenderActionError(t *testing.T) {
	server := &Server{
		tmpl: testTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	process := &Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Date(2026, 2, 4, 10, 0, 0, 0, time.UTC),
		Progress:  map[string]ProcessStep{},
	}
	actor := Actor{UserID: "u1", Role: "dep1"}
	rec := httptest.NewRecorder()
	server.renderActionError(rec, http.StatusBadRequest, "bad payload", process, actor)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "ACTION_LIST") {
		t.Fatalf("expected action list render, got %q", body)
	}
	if !strings.Contains(body, "bad payload") {
		t.Fatalf("expected error message in body, got %q", body)
	}
}

func TestRenderActionErrorForRequest(t *testing.T) {
	server := &Server{
		tmpl: testTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}
	process := &Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Date(2026, 2, 4, 10, 0, 0, 0, time.UTC),
		Progress:  map[string]ProcessStep{},
	}
	actor := Actor{UserID: "u1", Role: "dep1"}

	tests := []struct {
		name   string
		status int
		htmx   bool
	}{
		{name: "bad request htmx", status: http.StatusBadRequest, htmx: true},
		{name: "forbidden htmx", status: http.StatusForbidden, htmx: true},
		{name: "conflict htmx", status: http.StatusConflict, htmx: true},
		{name: "bad gateway htmx", status: http.StatusBadGateway, htmx: true},
		{name: "internal server error htmx", status: http.StatusInternalServerError, htmx: true},
		{name: "bad request full", status: http.StatusBadRequest},
		{name: "forbidden full", status: http.StatusForbidden},
		{name: "conflict full", status: http.StatusConflict},
		{name: "bad gateway full", status: http.StatusBadGateway},
		{name: "internal server error full", status: http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/process/"+process.ID.Hex()+"/substep/1.1/complete", nil)
			if tc.htmx {
				req.Header.Set("HX-Request", "true")
			}
			rec := httptest.NewRecorder()
			server.renderActionErrorForRequest(rec, req, tc.status, "rendered error", process, actor)

			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d", rec.Code, tc.status)
			}
			body := rec.Body.String()
			if tc.htmx {
				if strings.Contains(body, "PROCESS_PAGE") {
					t.Fatalf("expected HTMX partial, got full page body %q", body)
				}
			} else {
				if !strings.Contains(body, "PROCESS_PAGE") {
					t.Fatalf("expected full page render, got %q", body)
				}
			}
			if !strings.Contains(body, "rendered error") {
				t.Fatalf("expected error message in body, got %q", body)
			}
		})
	}
}
