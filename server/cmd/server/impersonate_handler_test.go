package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleImpersonateSuccess(t *testing.T) {
	server := &Server{
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/impersonate", strings.NewReader("userId=u1&role=dep1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	server.handleImpersonate(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if got := rr.Header().Get("Location"); got != "/w/workflow/backoffice/dep1" {
		t.Fatalf("expected redirect to /w/workflow/backoffice/dep1, got %q", got)
	}
	cookies := rr.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != "demo_user" || cookies[0].Value != "u1|dep1|workflow" {
		t.Fatalf("expected demo_user cookie u1|dep1|workflow, got %#v", cookies)
	}
}

func TestHandleImpersonateMethodNotAllowed(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/impersonate", nil)
	rr := httptest.NewRecorder()

	server.handleImpersonate(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestHandleImpersonateInvalidFormReturns400(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/impersonate", strings.NewReader("%zz"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	server.handleImpersonate(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleImpersonateMissingFieldsReturns400(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/impersonate", strings.NewReader("userId=u1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	server.handleImpersonate(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleImpersonateUnknownRoleReturns400(t *testing.T) {
	server := &Server{
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/impersonate", strings.NewReader("userId=u1&role=unknown"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	server.handleImpersonate(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleImpersonateBindsCookieToScopedWorkflow(t *testing.T) {
	server := &Server{
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/w/secondary/impersonate", strings.NewReader("userId=u4&role=dep1"))
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "secondary",
		Cfg: testRuntimeConfig(),
	}))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	server.handleImpersonate(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if got := rr.Header().Get("Location"); got != "/w/secondary/backoffice/dep1" {
		t.Fatalf("expected redirect to /w/secondary/backoffice/dep1, got %q", got)
	}
	cookies := rr.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != "demo_user" || cookies[0].Value != "u4|dep1|secondary" {
		t.Fatalf("expected demo_user cookie u4|dep1|secondary, got %#v", cookies)
	}
}
