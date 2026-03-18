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

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type resetFailingStore struct {
	*MemoryStore
	failSetPassword bool
	failMarkUsed    bool
}

func (s *resetFailingStore) SetUserPasswordHash(ctx context.Context, userMongoID primitive.ObjectID, passwordHash string) error {
	if s.failSetPassword {
		return errors.New("set password failed")
	}
	return s.MemoryStore.SetUserPasswordHash(ctx, userMongoID, passwordHash)
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

func TestHandleResetRequestTriggersIdentityRecovery(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPost, "/reset", strings.NewReader("email=user%40acme.io"))
	req.Host = "attesta.local"
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.handleResetRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if recoveryEmail != "user@acme.io" {
		t.Fatalf("recovery email = %q, want user@acme.io", recoveryEmail)
	}
	if recoveryURL != "http://attesta.local/reset/confirm" {
		t.Fatalf("recovery url = %q, want http://attesta.local/reset/confirm", recoveryURL)
	}
	if strings.Contains(rec.Body.String(), "/reset/") {
		t.Fatalf("unexpected local reset link in appwrite mode: %q", rec.Body.String())
	}
}

func TestHandleResetSetFlow(t *testing.T) {
	store := NewMemoryStore()
	oldHash, _ := bcrypt.GenerateFromPassword([]byte("old-password-value"), bcrypt.DefaultCost)
	user, err := store.CreateUser(t.Context(), AccountUser{
		Email:        "user@acme.io",
		PasswordHash: string(oldHash),
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	if _, err := store.CreatePasswordReset(t.Context(), PasswordReset{
		Email:       "user@acme.io",
		UserMongoID: user.ID,
		TokenHash:   "reset-token",
		ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
		CreatedAt:   time.Now().UTC(),
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

	updated, err := store.GetUserByMongoID(t.Context(), user.ID)
	if err != nil {
		t.Fatalf("GetUserByMongoID error: %v", err)
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
		Email:       "user@acme.io",
		UserMongoID: primitive.NewObjectID(),
		TokenHash:   "expired-reset-token",
		ExpiresAt:   time.Now().UTC().Add(-1 * time.Hour),
		CreatedAt:   time.Now().UTC(),
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

func TestHandleResetConfirmCompletesIdentityRecovery(t *testing.T) {
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
		t.Fatalf("completed recovery = %q/%q/%q", completedUserID, completedSecret, completedPassword)
	}
}

func TestHandleResetConfirmBranches(t *testing.T) {
	t.Run("get renders form", func(t *testing.T) {
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

	t.Run("identity missing", func(t *testing.T) {
		server := &Server{tmpl: resetTemplates(), now: time.Now}
		req := httptest.NewRequest(http.MethodGet, "/reset/confirm?userId=user-1&secret=secret-1", nil)
		rec := httptest.NewRecorder()
		server.handleResetSet(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("invalid params", func(t *testing.T) {
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

	t.Run("invalid legacy reset path", func(t *testing.T) {
		server := &Server{store: NewMemoryStore(), tmpl: resetTemplates(), now: time.Now}
		req := httptest.NewRequest(http.MethodGet, "/reset/with/slash", nil)
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

func TestHandleResetSetGetValidTokenRendersForm(t *testing.T) {
	store := NewMemoryStore()
	if _, err := store.CreatePasswordReset(t.Context(), PasswordReset{
		Email:       "user@acme.io",
		UserMongoID: primitive.NewObjectID(),
		TokenHash:   "valid-reset-token",
		ExpiresAt:   time.Now().UTC().Add(2 * time.Hour),
		CreatedAt:   time.Now().UTC(),
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
			Email:       user.Email,
			UserMongoID: user.ID,
			TokenHash:   token,
			ExpiresAt:   time.Now().UTC().Add(2 * time.Hour),
			CreatedAt:   time.Now().UTC(),
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
			Email:       "missing@acme.io",
			UserMongoID: primitive.NewObjectID(),
			TokenHash:   token,
			ExpiresAt:   time.Now().UTC().Add(2 * time.Hour),
			CreatedAt:   time.Now().UTC(),
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
			Email:       user.Email,
			UserMongoID: user.ID,
			TokenHash:   token,
			ExpiresAt:   time.Now().UTC().Add(2 * time.Hour),
			CreatedAt:   time.Now().UTC(),
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
			Email:       user.Email,
			UserMongoID: user.ID,
			TokenHash:   token,
			ExpiresAt:   time.Now().UTC().Add(2 * time.Hour),
			CreatedAt:   time.Now().UTC(),
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

func TestHandleResetRequestAndSetParseAndMethodErrors(t *testing.T) {
	server := &Server{store: NewMemoryStore(), tmpl: resetTemplates(), now: time.Now}

	reqBadResetForm := httptest.NewRequest(http.MethodPost, "/reset?bad=%zz", strings.NewReader("email=user%40acme.io"))
	reqBadResetForm.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recBadResetForm := httptest.NewRecorder()
	server.handleResetRequest(recBadResetForm, reqBadResetForm)
	if recBadResetForm.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recBadResetForm.Code, http.StatusBadRequest)
	}

	recBadPath := httptest.NewRecorder()
	reqBadPath := httptest.NewRequest(http.MethodGet, "/reset/invalid/path", nil)
	server.handleResetSet(recBadPath, reqBadPath)
	if recBadPath.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recBadPath.Code, http.StatusNotFound)
	}

	store := NewMemoryStore()
	if _, err := store.CreatePasswordReset(t.Context(), PasswordReset{
		Email:       "user@acme.io",
		UserMongoID: primitive.NewObjectID(),
		TokenHash:   "method-token",
		ExpiresAt:   time.Now().UTC().Add(2 * time.Hour),
		CreatedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreatePasswordReset error: %v", err)
	}
	serverWithToken := &Server{store: store, tmpl: resetTemplates(), now: time.Now}

	reqMethod := httptest.NewRequest(http.MethodPut, "/reset/method-token", nil)
	recMethod := httptest.NewRecorder()
	serverWithToken.handleResetSet(recMethod, reqMethod)
	if recMethod.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recMethod.Code, http.StatusMethodNotAllowed)
	}

	reqParse := httptest.NewRequest(http.MethodPost, "/reset/method-token?bad=%zz", strings.NewReader("password=this-is-strong-enough"))
	reqParse.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recParse := httptest.NewRecorder()
	serverWithToken.handleResetSet(recParse, reqParse)
	if recParse.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recParse.Code, http.StatusBadRequest)
	}
}
