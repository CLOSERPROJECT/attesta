package main

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type inviteFailingStore struct {
	*MemoryStore
	failSetPassword   bool
	failCreateSession bool
}

func (s *inviteFailingStore) SetUserPasswordHash(ctx context.Context, userID, passwordHash string) error {
	if s.failSetPassword {
		return errors.New("set password failed")
	}
	return s.MemoryStore.SetUserPasswordHash(ctx, userID, passwordHash)
}

func (s *inviteFailingStore) CreateSession(ctx context.Context, session Session) (Session, error) {
	if s.failCreateSession {
		return Session{}, errors.New("create session failed")
	}
	return s.MemoryStore.CreateSession(ctx, session)
}

func inviteTemplates() *template.Template {
	return template.Must(template.New("invite-test").Parse(`
{{define "layout.html"}}{{if eq .Body "invite_body"}}{{template "invite_body" .}}{{else if eq .Body "login_body"}}{{template "login_body" .}}{{end}}{{end}}
{{define "invite_body"}}INVITE{{if .Error}} {{.Error}}{{end}}{{end}}
{{define "invite.html"}}{{template "layout.html" .}}{{end}}
{{define "login_body"}}LOGIN{{if .Error}} {{.Error}}{{end}}{{end}}
{{define "login.html"}}{{template "layout.html" .}}{{end}}
`))
}

func TestHandleInviteHappyPath(t *testing.T) {
	store := NewMemoryStore()
	org, err := store.CreateOrganization(t.Context(), Organization{Name: "Acme"})
	if err != nil {
		t.Fatalf("CreateOrganization error: %v", err)
	}
	role, err := store.CreateRole(t.Context(), Role{OrgSlug: org.Slug, Name: "org_admin"})
	if err != nil {
		t.Fatalf("CreateRole error: %v", err)
	}
	user, err := store.CreateUser(t.Context(), AccountUser{
		UserID:    "u1",
		OrgSlug:   org.Slug,
		Email:     "new@acme.io",
		RoleSlugs: []string{role.Slug},
		Status:    "invited",
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	if _, err := store.CreateInvite(t.Context(), Invite{
		OrgID:     org.ID,
		Email:     "new@acme.io",
		UserID:    user.UserID,
		RoleSlugs: []string{role.Slug},
		TokenHash: "invite-token",
		ExpiresAt: time.Now().UTC().Add(48 * time.Hour),
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateInvite error: %v", err)
	}

	server := &Server{store: store, tmpl: inviteTemplates(), now: time.Now}
	req := httptest.NewRequest(http.MethodPost, "/invite/invite-token", strings.NewReader("password=this-is-strong-enough"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	server.handleInvite(rec, req)

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

	updatedUser, err := store.GetUserByUserID(t.Context(), user.UserID)
	if err != nil {
		t.Fatalf("GetUserByUserID error: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updatedUser.PasswordHash), []byte("this-is-strong-enough")); err != nil {
		t.Fatalf("password hash verification failed: %v", err)
	}

	invite, err := store.LoadInviteByTokenHash(t.Context(), "invite-token")
	if err != nil {
		t.Fatalf("LoadInviteByTokenHash error: %v", err)
	}
	if invite.UsedAt == nil {
		t.Fatal("expected invite to be marked used")
	}
}

func TestHandleInviteExpiredToken(t *testing.T) {
	store := NewMemoryStore()
	org, _ := store.CreateOrganization(t.Context(), Organization{Name: "Acme"})
	user, _ := store.CreateUser(t.Context(), AccountUser{
		UserID:  "u1",
		OrgSlug: org.Slug,
		Email:   "expired@acme.io",
		Status:  "invited",
	})
	_, _ = store.CreateInvite(t.Context(), Invite{
		OrgID:     org.ID,
		Email:     "expired@acme.io",
		UserID:    user.UserID,
		TokenHash: "expired-token",
		ExpiresAt: time.Now().UTC().Add(-1 * time.Hour),
		CreatedAt: time.Now().UTC(),
	})

	server := &Server{store: store, tmpl: inviteTemplates(), now: time.Now}
	req := httptest.NewRequest(http.MethodGet, "/invite/expired-token", nil)
	rec := httptest.NewRecorder()
	server.handleInvite(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleInviteReusedToken(t *testing.T) {
	store := NewMemoryStore()
	org, _ := store.CreateOrganization(t.Context(), Organization{Name: "Acme"})
	user, _ := store.CreateUser(t.Context(), AccountUser{
		UserID:  "u1",
		OrgSlug: org.Slug,
		Email:   "used@acme.io",
		Status:  "invited",
	})
	_, _ = store.CreateInvite(t.Context(), Invite{
		OrgID:     org.ID,
		Email:     "used@acme.io",
		UserID:    user.UserID,
		TokenHash: "used-token",
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		CreatedAt: time.Now().UTC(),
	})
	_ = store.MarkInviteUsed(t.Context(), "used-token", time.Now().UTC())

	server := &Server{store: store, tmpl: inviteTemplates(), now: time.Now}
	req := httptest.NewRequest(http.MethodGet, "/invite/used-token", nil)
	rec := httptest.NewRecorder()
	server.handleInvite(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleInviteRejectsMismatchedEmailAndWeakPassword(t *testing.T) {
	store := NewMemoryStore()
	org, _ := store.CreateOrganization(t.Context(), Organization{Name: "Acme"})
	user, _ := store.CreateUser(t.Context(), AccountUser{
		UserID:  "u1",
		OrgSlug: org.Slug,
		Email:   "user@acme.io",
		Status:  "invited",
	})
	_, _ = store.CreateInvite(t.Context(), Invite{
		OrgID:     org.ID,
		Email:     "different@acme.io",
		UserID:    user.UserID,
		TokenHash: "mismatch-token",
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		CreatedAt: time.Now().UTC(),
	})

	server := &Server{store: store, tmpl: inviteTemplates(), now: time.Now}

	weakReq := httptest.NewRequest(http.MethodPost, "/invite/mismatch-token", strings.NewReader("password=short"))
	weakReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	weakRec := httptest.NewRecorder()
	server.handleInvite(weakRec, weakReq)
	if weakRec.Code != http.StatusBadRequest {
		t.Fatalf("weak password status = %d, want %d", weakRec.Code, http.StatusBadRequest)
	}

	req := httptest.NewRequest(http.MethodPost, "/invite/mismatch-token", strings.NewReader("password=this-is-strong-enough"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.handleInvite(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("mismatched email status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleInviteRejectsInvalidTokenPath(t *testing.T) {
	server := &Server{store: NewMemoryStore(), tmpl: inviteTemplates(), now: time.Now}

	req := httptest.NewRequest(http.MethodGet, "/invite/", nil)
	rec := httptest.NewRecorder()
	server.handleInvite(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/invite/with/slash", nil)
	rec2 := httptest.NewRecorder()
	server.handleInvite(rec2, req2)
	if rec2.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec2.Code, http.StatusNotFound)
	}
}

func TestHandleInviteMethodNotAllowed(t *testing.T) {
	store := NewMemoryStore()
	org, _ := store.CreateOrganization(t.Context(), Organization{Name: "Acme"})
	user, _ := store.CreateUser(t.Context(), AccountUser{
		UserID:  "u-method",
		OrgSlug: org.Slug,
		Email:   "method@acme.io",
		Status:  "invited",
	})
	_, _ = store.CreateInvite(t.Context(), Invite{
		OrgID:     org.ID,
		Email:     "method@acme.io",
		UserID:    user.UserID,
		TokenHash: "method-token",
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		CreatedAt: time.Now().UTC(),
	})

	server := &Server{store: store, tmpl: inviteTemplates(), now: time.Now}
	req := httptest.NewRequest(http.MethodPut, "/invite/method-token", nil)
	rec := httptest.NewRecorder()
	server.handleInvite(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleInviteInvalidInviteUser(t *testing.T) {
	store := NewMemoryStore()
	org, _ := store.CreateOrganization(t.Context(), Organization{Name: "Acme"})
	_, _ = store.CreateInvite(t.Context(), Invite{
		OrgID:     org.ID,
		Email:     "ghost@acme.io",
		UserID:    "missing-user",
		TokenHash: "ghost-token",
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		CreatedAt: time.Now().UTC(),
	})

	server := &Server{store: store, tmpl: inviteTemplates(), now: time.Now}
	req := httptest.NewRequest(http.MethodPost, "/invite/ghost-token", strings.NewReader("password=this-is-strong-enough"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.handleInvite(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleInviteSetPasswordAndSessionFailures(t *testing.T) {
	base := NewMemoryStore()
	org, _ := base.CreateOrganization(t.Context(), Organization{Name: "Acme"})
	role, _ := base.CreateRole(t.Context(), Role{OrgSlug: org.Slug, Name: "org_admin"})
	user, _ := base.CreateUser(t.Context(), AccountUser{
		UserID:    "u-fail",
		OrgSlug:   org.Slug,
		Email:     "fail@acme.io",
		RoleSlugs: []string{role.Slug},
		Status:    "invited",
	})
	_, _ = base.CreateInvite(t.Context(), Invite{
		OrgID:     org.ID,
		Email:     "fail@acme.io",
		UserID:    user.UserID,
		RoleSlugs: []string{role.Slug},
		TokenHash: "fail-token",
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		CreatedAt: time.Now().UTC(),
	})

	t.Run("set password failure", func(t *testing.T) {
		store := &inviteFailingStore{MemoryStore: base, failSetPassword: true}
		server := &Server{store: store, tmpl: inviteTemplates(), now: time.Now}
		req := httptest.NewRequest(http.MethodPost, "/invite/fail-token", strings.NewReader("password=this-is-strong-enough"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		server.handleInvite(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("create session failure", func(t *testing.T) {
		store := &inviteFailingStore{MemoryStore: base, failCreateSession: true}
		server := &Server{store: store, tmpl: inviteTemplates(), now: time.Now}
		req := httptest.NewRequest(http.MethodPost, "/invite/fail-token", strings.NewReader("password=this-is-strong-enough"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		server.handleInvite(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}
