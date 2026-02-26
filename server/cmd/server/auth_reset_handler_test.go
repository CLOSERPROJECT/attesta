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

	"golang.org/x/crypto/bcrypt"
)

type resetFailingStore struct {
	*MemoryStore
	failSetPassword bool
	failMarkUsed    bool
}

func (s *resetFailingStore) SetUserPasswordHash(ctx context.Context, userID, passwordHash string) error {
	if s.failSetPassword {
		return errors.New("set password failed")
	}
	return s.MemoryStore.SetUserPasswordHash(ctx, userID, passwordHash)
}

func (s *resetFailingStore) MarkPasswordResetUsed(ctx context.Context, tokenHash string, usedAt time.Time) error {
	if s.failMarkUsed {
		return errors.New("mark used failed")
	}
	return s.MemoryStore.MarkPasswordResetUsed(ctx, tokenHash, usedAt)
}

func resetTemplates() *template.Template {
	return template.Must(template.New("reset-test").Parse(`
{{define "layout.html"}}{{if eq .Body "reset_request_body"}}{{template "reset_request_body" .}}{{else if eq .Body "reset_set_body"}}{{template "reset_set_body" .}}{{end}}{{end}}
{{define "reset_request_body"}}RESET_REQUEST {{.Confirmation}} {{.ResetLink}}{{end}}
{{define "reset_request.html"}}{{template "layout.html" .}}{{end}}
{{define "reset_set_body"}}RESET_SET {{.Error}}{{end}}
{{define "reset_set.html"}}{{template "layout.html" .}}{{end}}
`))
}

func TestHandleResetRequestGenericConfirmation(t *testing.T) {
	store := NewMemoryStore()
	hash, _ := bcrypt.GenerateFromPassword([]byte("old-password-value"), bcrypt.DefaultCost)
	if _, err := store.CreateUser(t.Context(), AccountUser{
		UserID:       "u1",
		Email:        "user@acme.io",
		PasswordHash: string(hash),
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	server := &Server{store: store, tmpl: resetTemplates(), now: time.Now}

	req := httptest.NewRequest(http.MethodPost, "/reset", strings.NewReader("email=missing%40acme.io"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.handleResetRequest(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("missing email status = %d, want %d", rec.Code, http.StatusOK)
	}
	if strings.Contains(rec.Body.String(), "/reset/") {
		t.Fatalf("unexpected reset link for missing email: %q", rec.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/reset", strings.NewReader("email=user%40acme.io"))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec2 := httptest.NewRecorder()
	server.handleResetRequest(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("existing email status = %d, want %d", rec2.Code, http.StatusOK)
	}
	if !strings.Contains(rec2.Body.String(), "/reset/") {
		t.Fatalf("expected reset link for existing email, body=%q", rec2.Body.String())
	}
}

func TestHandleResetSetFlow(t *testing.T) {
	store := NewMemoryStore()
	oldHash, _ := bcrypt.GenerateFromPassword([]byte("old-password-value"), bcrypt.DefaultCost)
	user, err := store.CreateUser(t.Context(), AccountUser{
		UserID:       "u1",
		Email:        "user@acme.io",
		PasswordHash: string(oldHash),
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	if _, err := store.CreatePasswordReset(t.Context(), PasswordReset{
		Email:     "user@acme.io",
		UserID:    user.UserID,
		TokenHash: "reset-token",
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreatePasswordReset error: %v", err)
	}

	server := &Server{store: store, tmpl: resetTemplates(), now: time.Now}
	req := httptest.NewRequest(http.MethodPost, "/reset/reset-token", strings.NewReader("password=new-password-123"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.handleResetSet(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if rec.Header().Get("Location") != "/" {
		t.Fatalf("location = %q, want /", rec.Header().Get("Location"))
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != "attesta_session" {
		t.Fatalf("expected attesta_session cookie, got %#v", cookies)
	}

	updated, err := store.GetUserByUserID(t.Context(), user.UserID)
	if err != nil {
		t.Fatalf("GetUserByUserID error: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte("new-password-123")); err != nil {
		t.Fatalf("password hash mismatch: %v", err)
	}
	reset, err := store.LoadPasswordResetByTokenHash(t.Context(), "reset-token")
	if err != nil {
		t.Fatalf("LoadPasswordResetByTokenHash error: %v", err)
	}
	if reset.UsedAt == nil {
		t.Fatal("expected reset token marked used")
	}
}

func TestHandleResetSetRejectsExpiredToken(t *testing.T) {
	store := NewMemoryStore()
	if _, err := store.CreatePasswordReset(t.Context(), PasswordReset{
		Email:     "user@acme.io",
		UserID:    "u1",
		TokenHash: "expired-reset-token",
		ExpiresAt: time.Now().UTC().Add(-1 * time.Hour),
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreatePasswordReset error: %v", err)
	}

	server := &Server{store: store, tmpl: resetTemplates(), now: time.Now}
	req := httptest.NewRequest(http.MethodGet, "/reset/expired-reset-token", nil)
	rec := httptest.NewRecorder()
	server.handleResetSet(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
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

func TestHandleResetSetGetValidTokenRendersForm(t *testing.T) {
	store := NewMemoryStore()
	if _, err := store.CreatePasswordReset(t.Context(), PasswordReset{
		Email:     "user@acme.io",
		UserID:    "u1",
		TokenHash: "valid-reset-token",
		ExpiresAt: time.Now().UTC().Add(2 * time.Hour),
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreatePasswordReset error: %v", err)
	}

	server := &Server{store: store, tmpl: resetTemplates(), now: time.Now}
	req := httptest.NewRequest(http.MethodGet, "/reset/valid-reset-token", nil)
	rec := httptest.NewRecorder()
	server.handleResetSet(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "RESET_SET") {
		t.Fatalf("expected reset form body, got %q", rec.Body.String())
	}
}

func TestHandleResetRequestMethodNotAllowed(t *testing.T) {
	server := &Server{store: NewMemoryStore(), tmpl: resetTemplates(), now: time.Now}
	req := httptest.NewRequest(http.MethodPut, "/reset", nil)
	rec := httptest.NewRecorder()
	server.handleResetRequest(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleResetSetValidationAndFailurePaths(t *testing.T) {
	base := NewMemoryStore()
	oldHash, _ := bcrypt.GenerateFromPassword([]byte("old-password-value"), bcrypt.DefaultCost)
	user, err := base.CreateUser(t.Context(), AccountUser{
		UserID:       "u-reset-extra",
		Email:        "user-reset-extra@acme.io",
		PasswordHash: string(oldHash),
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}

	t.Run("short password", func(t *testing.T) {
		token := "reset-short-password"
		_, _ = base.CreatePasswordReset(t.Context(), PasswordReset{
			Email:     user.Email,
			UserID:    user.UserID,
			TokenHash: token,
			ExpiresAt: time.Now().UTC().Add(2 * time.Hour),
			CreatedAt: time.Now().UTC(),
		})
		server := &Server{store: base, tmpl: resetTemplates(), now: time.Now}
		req := httptest.NewRequest(http.MethodPost, "/reset/"+token, strings.NewReader("password=short"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		server.handleResetSet(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid reset user", func(t *testing.T) {
		token := "reset-missing-user"
		_, _ = base.CreatePasswordReset(t.Context(), PasswordReset{
			Email:     "missing@acme.io",
			UserID:    "missing-user",
			TokenHash: token,
			ExpiresAt: time.Now().UTC().Add(2 * time.Hour),
			CreatedAt: time.Now().UTC(),
		})
		server := &Server{store: base, tmpl: resetTemplates(), now: time.Now}
		req := httptest.NewRequest(http.MethodPost, "/reset/"+token, strings.NewReader("password=this-is-strong-enough"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		server.handleResetSet(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("set password failure", func(t *testing.T) {
		token := "reset-set-password-failure"
		_, _ = base.CreatePasswordReset(t.Context(), PasswordReset{
			Email:     user.Email,
			UserID:    user.UserID,
			TokenHash: token,
			ExpiresAt: time.Now().UTC().Add(2 * time.Hour),
			CreatedAt: time.Now().UTC(),
		})
		store := &resetFailingStore{MemoryStore: base, failSetPassword: true}
		server := &Server{store: store, tmpl: resetTemplates(), now: time.Now}
		req := httptest.NewRequest(http.MethodPost, "/reset/"+token, strings.NewReader("password=this-is-strong-enough"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		server.handleResetSet(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("mark used failure", func(t *testing.T) {
		token := "reset-mark-used-failure"
		_, _ = base.CreatePasswordReset(t.Context(), PasswordReset{
			Email:     user.Email,
			UserID:    user.UserID,
			TokenHash: token,
			ExpiresAt: time.Now().UTC().Add(2 * time.Hour),
			CreatedAt: time.Now().UTC(),
		})
		store := &resetFailingStore{MemoryStore: base, failMarkUsed: true}
		server := &Server{store: store, tmpl: resetTemplates(), now: time.Now}
		req := httptest.NewRequest(http.MethodPost, "/reset/"+token, strings.NewReader("password=this-is-strong-enough"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		server.handleResetSet(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}
