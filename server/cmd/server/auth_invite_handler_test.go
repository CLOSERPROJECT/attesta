package main

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

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
