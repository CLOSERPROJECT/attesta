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

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type inviteFailingStore struct {
	*MemoryStore
	failSetPassword   bool
	failCreateSession bool
}

func (s *inviteFailingStore) SetUserPasswordHash(ctx context.Context, userMongoID primitive.ObjectID, passwordHash string) error {
	if s.failSetPassword {
		return errors.New("set password failed")
	}
	return s.MemoryStore.SetUserPasswordHash(ctx, userMongoID, passwordHash)
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
		OrgSlug:   org.Slug,
		Email:     "new@acme.io",
		RoleSlugs: []string{role.Slug},
		Status:    "invited",
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	if _, err := store.CreateInvite(t.Context(), Invite{
		OrgID:       org.ID,
		Email:       "new@acme.io",
		UserMongoID: user.ID,
		RoleSlugs:   []string{role.Slug},
		TokenHash:   "invite-token",
		ExpiresAt:   time.Now().UTC().Add(48 * time.Hour),
		CreatedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateInvite error: %v", err)
	}

	server := &Server{store: store, tmpl: inviteTemplates(), now: time.Now}
	req := httptest.NewRequest(http.MethodPost, "/invite/invite-token", strings.NewReader("password=this-is-strong-enough&confirm_password=this-is-strong-enough"))
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

	updatedUser, err := store.GetUserByMongoID(t.Context(), user.ID)
	if err != nil {
		t.Fatalf("GetUserByMongoID error: %v", err)
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

func TestHandleInviteGetRendersInvite(t *testing.T) {
	store := NewMemoryStore()
	org, err := store.CreateOrganization(t.Context(), Organization{Name: "Acme"})
	if err != nil {
		t.Fatalf("CreateOrganization error: %v", err)
	}
	if _, err := store.CreateInvite(t.Context(), Invite{
		OrgID:       org.ID,
		Email:       "new@acme.io",
		UserMongoID: primitive.NewObjectID(),
		RoleSlugs:   []string{"approver"},
		TokenHash:   "invite-token",
		ExpiresAt:   time.Now().UTC().Add(48 * time.Hour),
		CreatedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateInvite error: %v", err)
	}

	server := &Server{store: store, tmpl: inviteTemplates(), now: time.Now}
	req := httptest.NewRequest(http.MethodGet, "/invite/invite-token", nil)
	rec := httptest.NewRecorder()

	server.handleInvite(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); !strings.Contains(body, "INVITE") {
		t.Fatalf("body = %q", body)
	}
}

func TestHandleInviteExpiredToken(t *testing.T) {
	store := NewMemoryStore()
	org, _ := store.CreateOrganization(t.Context(), Organization{Name: "Acme"})
	user, _ := store.CreateUser(t.Context(), AccountUser{
		OrgSlug: org.Slug,
		Email:   "expired@acme.io",
		Status:  "invited",
	})
	_, _ = store.CreateInvite(t.Context(), Invite{
		OrgID:       org.ID,
		Email:       "expired@acme.io",
		UserMongoID: user.ID,
		TokenHash:   "expired-token",
		ExpiresAt:   time.Now().UTC().Add(-1 * time.Hour),
		CreatedAt:   time.Now().UTC(),
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
		OrgSlug: org.Slug,
		Email:   "used@acme.io",
		Status:  "invited",
	})
	_, _ = store.CreateInvite(t.Context(), Invite{
		OrgID:       org.ID,
		Email:       "used@acme.io",
		UserMongoID: user.ID,
		TokenHash:   "used-token",
		ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
		CreatedAt:   time.Now().UTC(),
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
		OrgSlug: org.Slug,
		Email:   "user@acme.io",
		Status:  "invited",
	})
	_, _ = store.CreateInvite(t.Context(), Invite{
		OrgID:       org.ID,
		Email:       "different@acme.io",
		UserMongoID: user.ID,
		TokenHash:   "mismatch-token",
		ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
		CreatedAt:   time.Now().UTC(),
	})

	server := &Server{store: store, tmpl: inviteTemplates(), now: time.Now}

	weakReq := httptest.NewRequest(http.MethodPost, "/invite/mismatch-token", strings.NewReader("password=short&confirm_password=short"))
	weakReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	weakRec := httptest.NewRecorder()
	server.handleInvite(weakRec, weakReq)
	if weakRec.Code != http.StatusBadRequest {
		t.Fatalf("weak password status = %d, want %d", weakRec.Code, http.StatusBadRequest)
	}

	req := httptest.NewRequest(http.MethodPost, "/invite/mismatch-token", strings.NewReader("password=this-is-strong-enough&confirm_password=this-is-strong-enough"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.handleInvite(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("mismatched email status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleInviteRejectsMismatchedConfirmation(t *testing.T) {
	store := NewMemoryStore()
	org, _ := store.CreateOrganization(t.Context(), Organization{Name: "Acme"})
	user, _ := store.CreateUser(t.Context(), AccountUser{
		OrgSlug: org.Slug,
		Email:   "confirm@acme.io",
		Status:  "invited",
	})
	_, _ = store.CreateInvite(t.Context(), Invite{
		OrgID:       org.ID,
		Email:       "confirm@acme.io",
		UserMongoID: user.ID,
		TokenHash:   "confirm-token",
		ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
		CreatedAt:   time.Now().UTC(),
	})

	server := &Server{store: store, tmpl: inviteTemplates(), now: time.Now}
	req := httptest.NewRequest(http.MethodPost, "/invite/confirm-token", strings.NewReader("password=this-is-strong-enough&confirm_password=totally-different"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.handleInvite(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rec.Body.String(), "passwords do not match") {
		t.Fatalf("expected mismatch message, got %q", rec.Body.String())
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
		OrgSlug: org.Slug,
		Email:   "method@acme.io",
		Status:  "invited",
	})
	_, _ = store.CreateInvite(t.Context(), Invite{
		OrgID:       org.ID,
		Email:       "method@acme.io",
		UserMongoID: user.ID,
		TokenHash:   "method-token",
		ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
		CreatedAt:   time.Now().UTC(),
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
		OrgID:       org.ID,
		Email:       "ghost@acme.io",
		UserMongoID: primitive.NewObjectID(),
		TokenHash:   "ghost-token",
		ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
		CreatedAt:   time.Now().UTC(),
	})

	server := &Server{store: store, tmpl: inviteTemplates(), now: time.Now}
	req := httptest.NewRequest(http.MethodPost, "/invite/ghost-token", strings.NewReader("password=this-is-strong-enough&confirm_password=this-is-strong-enough"))
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
		OrgSlug:   org.Slug,
		Email:     "fail@acme.io",
		RoleSlugs: []string{role.Slug},
		Status:    "invited",
	})
	_, _ = base.CreateInvite(t.Context(), Invite{
		OrgID:       org.ID,
		Email:       "fail@acme.io",
		UserMongoID: user.ID,
		RoleSlugs:   []string{role.Slug},
		TokenHash:   "fail-token",
		ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
		CreatedAt:   time.Now().UTC(),
	})

	t.Run("set password failure", func(t *testing.T) {
		store := &inviteFailingStore{MemoryStore: base, failSetPassword: true}
		server := &Server{store: store, tmpl: inviteTemplates(), now: time.Now}
		req := httptest.NewRequest(http.MethodPost, "/invite/fail-token", strings.NewReader("password=this-is-strong-enough&confirm_password=this-is-strong-enough"))
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
		req := httptest.NewRequest(http.MethodPost, "/invite/fail-token", strings.NewReader("password=this-is-strong-enough&confirm_password=this-is-strong-enough"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		server.handleInvite(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}

func TestHandleInviteAcceptCreatesSessionCookie(t *testing.T) {
	now := time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)
	var acceptedTeamID string
	var acceptedMembershipID string
	var acceptedUserID string
	var acceptedSecret string
	server := &Server{
		identity: &fakeIdentityStore{
			acceptInviteFunc: func(ctx context.Context, teamID, membershipID, userID, secret string) (IdentitySession, error) {
				acceptedTeamID = teamID
				acceptedMembershipID = membershipID
				acceptedUserID = userID
				acceptedSecret = secret
				return fakeIdentitySession("invite-session", userID, now.Add(24*time.Hour)), nil
			},
		},
		now: time.Now,
	}

	req := httptest.NewRequest(http.MethodGet, "/invite/accept?teamId=acme&membershipId=membership-1&userId=user-1&secret=secret-1", nil)
	rec := httptest.NewRecorder()
	server.handleInvite(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if rec.Header().Get("Location") != "/" {
		t.Fatalf("location = %q, want /", rec.Header().Get("Location"))
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != "attesta_session" || cookies[0].Value != "invite-session" {
		t.Fatalf("cookies = %#v", cookies)
	}
	if acceptedTeamID != "acme" || acceptedMembershipID != "membership-1" || acceptedUserID != "user-1" || acceptedSecret != "secret-1" {
		t.Fatalf("accepted params = %q/%q/%q/%q", acceptedTeamID, acceptedMembershipID, acceptedUserID, acceptedSecret)
	}
}

func TestHandleInviteAcceptBranches(t *testing.T) {
	t.Run("identity missing", func(t *testing.T) {
		server := &Server{now: time.Now}
		req := httptest.NewRequest(http.MethodGet, "/invite/accept?teamId=acme&membershipId=membership-1&userId=user-1&secret=secret-1", nil)
		rec := httptest.NewRecorder()
		server.handleInvite(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("invalid params", func(t *testing.T) {
		server := &Server{identity: &fakeIdentityStore{}, now: time.Now}
		req := httptest.NewRequest(http.MethodGet, "/invite/accept?teamId=acme&membershipId=membership-1&userId=user-1", nil)
		rec := httptest.NewRecorder()
		server.handleInvite(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		server := &Server{identity: &fakeIdentityStore{}, now: time.Now}
		req := httptest.NewRequest(http.MethodPost, "/invite/accept?teamId=acme&membershipId=membership-1&userId=user-1&secret=secret-1", nil)
		rec := httptest.NewRecorder()
		server.handleInvite(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("accept failure", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				acceptInviteFunc: func(ctx context.Context, teamID, membershipID, userID, secret string) (IdentitySession, error) {
					return IdentitySession{}, errors.New("boom")
				},
			},
			now: time.Now,
		}
		req := httptest.NewRequest(http.MethodGet, "/invite/accept?teamId=acme&membershipId=membership-1&userId=user-1&secret=secret-1", nil)
		rec := httptest.NewRecorder()
		server.handleInvite(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("write session cookie failure", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				acceptInviteFunc: func(ctx context.Context, teamID, membershipID, userID, secret string) (IdentitySession, error) {
					return IdentitySession{UserID: userID}, nil
				},
			},
			now: time.Now,
		}
		req := httptest.NewRequest(http.MethodGet, "/invite/accept?teamId=acme&membershipId=membership-1&userId=user-1&secret=secret-1", nil)
		rec := httptest.NewRecorder()
		server.handleInvite(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}
