package main

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func resetTemplates() *template.Template {
	return template.Must(template.New("reset-test").Parse(`
{{define "layout.html"}}{{if eq .Body "reset_request_body"}}{{template "reset_request_body" .}}{{else if eq .Body "reset_set_body"}}{{template "reset_set_body" .}}{{end}}{{end}}
{{define "reset_request_body"}}RESET_REQUEST{{if .Confirmation}} {{.Confirmation}}{{end}}{{end}}
{{define "reset_request.html"}}{{template "layout.html" .}}{{end}}
{{define "reset_set_body"}}RESET_SET{{if .Error}} {{.Error}}{{end}}{{end}}
{{define "reset_set.html"}}{{template "layout.html" .}}{{end}}
`))
}

func TestHandleResetRequestTriggersRecovery(t *testing.T) {
	var recoveryEmail string
	var recoveryURL string
	server := &Server{
		identity: &fakeIdentityStore{
			createRecoveryFunc: func(ctx context.Context, email, redirectURL string) error {
				recoveryEmail = email
				recoveryURL = redirectURL
				return nil
			},
		},
		tmpl: resetTemplates(),
		now:  time.Now,
	}

	req := httptest.NewRequest(http.MethodPost, "/reset", strings.NewReader("email=user%40example.com"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Host = "attesta.local"
	rec := httptest.NewRecorder()
	server.handleResetRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if recoveryEmail != "user@example.com" {
		t.Fatalf("recovery email = %q", recoveryEmail)
	}
	if recoveryURL != "http://attesta.local/reset/confirm" {
		t.Fatalf("recovery url = %q, want http://attesta.local/reset/confirm", recoveryURL)
	}
	if !strings.Contains(rec.Body.String(), "If the account exists") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestHandleResetConfirmCompletesRecovery(t *testing.T) {
	var completedUserID string
	var completedSecret string
	var completedPassword string
	server := &Server{
		identity: &fakeIdentityStore{
			completeRecoveryFunc: func(ctx context.Context, userID, secret, password string) error {
				completedUserID = userID
				completedSecret = secret
				completedPassword = password
				return nil
			},
		},
		tmpl: resetTemplates(),
		now:  time.Now,
	}

	req := httptest.NewRequest(http.MethodPost, "/reset/confirm?userId=user-1&secret=secret-1", strings.NewReader("password=this-is-strong-enough"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.handleResetSet(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if rec.Header().Get("Location") != "/login" {
		t.Fatalf("location = %q, want /login", rec.Header().Get("Location"))
	}
	if completedUserID != "user-1" || completedSecret != "secret-1" || completedPassword != "this-is-strong-enough" {
		t.Fatalf("completed = %q/%q/%q", completedUserID, completedSecret, completedPassword)
	}
}

func TestHandleResetSetBranches(t *testing.T) {
	t.Run("confirm get renders form", func(t *testing.T) {
		server := &Server{identity: &fakeIdentityStore{}, tmpl: resetTemplates(), now: time.Now}
		req := httptest.NewRequest(http.MethodGet, "/reset/confirm?userId=user-1&secret=secret-1", nil)
		rec := httptest.NewRecorder()
		server.handleResetSet(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if !strings.Contains(rec.Body.String(), "RESET_SET") {
			t.Fatalf("body = %q", rec.Body.String())
		}
	})

	t.Run("missing params", func(t *testing.T) {
		server := &Server{identity: &fakeIdentityStore{}, tmpl: resetTemplates(), now: time.Now}
		req := httptest.NewRequest(http.MethodGet, "/reset/confirm?userId=user-1", nil)
		rec := httptest.NewRecorder()
		server.handleResetSet(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("short password", func(t *testing.T) {
		server := &Server{identity: &fakeIdentityStore{}, tmpl: resetTemplates(), now: time.Now}
		req := httptest.NewRequest(http.MethodPost, "/reset/confirm?userId=user-1&secret=secret-1", strings.NewReader("password=short"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		server.handleResetSet(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		server := &Server{identity: &fakeIdentityStore{}, tmpl: resetTemplates(), now: time.Now}
		req := httptest.NewRequest(http.MethodPut, "/reset/confirm?userId=user-1&secret=secret-1", nil)
		rec := httptest.NewRecorder()
		server.handleResetSet(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("completion failure", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				completeRecoveryFunc: func(ctx context.Context, userID, secret, password string) error {
					return errors.New("boom")
				},
			},
			tmpl: resetTemplates(),
			now:  time.Now,
		}
		req := httptest.NewRequest(http.MethodPost, "/reset/confirm?userId=user-1&secret=secret-1", strings.NewReader("password=this-is-strong-enough"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		server.handleResetSet(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("legacy reset path removed", func(t *testing.T) {
		server := &Server{identity: &fakeIdentityStore{}, tmpl: resetTemplates(), now: time.Now}
		req := httptest.NewRequest(http.MethodGet, "/reset/legacy-token", nil)
		rec := httptest.NewRecorder()
		server.handleResetSet(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})
}

func TestHandleResetPageHidesAdminTopbarLinks(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))
	server := &Server{
		store: NewMemoryStore(),
		tmpl:  tmpl,
	}

	req := httptest.NewRequest(http.MethodGet, "/reset", nil)
	rec := httptest.NewRecorder()
	server.handleResetRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if strings.Contains(body, `href="/admin/orgs"`) || strings.Contains(body, `href="/org-admin/users"`) {
		t.Fatalf("expected reset page without admin nav links, got %q", body)
	}
}

func TestHandleResetRequestMethodNotAllowed(t *testing.T) {
	server := &Server{identity: &fakeIdentityStore{}, tmpl: resetTemplates(), now: time.Now}
	req := httptest.NewRequest(http.MethodPut, "/reset", nil)
	rec := httptest.NewRecorder()
	server.handleResetRequest(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
