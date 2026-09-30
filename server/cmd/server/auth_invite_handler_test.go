package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandleInviteAcceptCreatesSessionCookie(t *testing.T) {
	now := time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)
	var acceptedTeamID string
	var acceptedMembershipID string
	var acceptedUserID string
	var acceptedSecret string
	var establishedUserID string
	var establishedVerified bool
	server := &Server{
		identity: &fakeIdentityStore{
			acceptInviteFunc: func(ctx context.Context, teamID, membershipID, userID, secret string) (IdentitySession, error) {
				acceptedTeamID = teamID
				acceptedMembershipID = membershipID
				acceptedUserID = userID
				acceptedSecret = secret
				return fakeIdentitySession("invite-session", userID, now.Add(24*time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "invitee@example.com", PasswordSet: true}, nil
			},
			updateEmailVerificationFunc: func(ctx context.Context, userID string, verified bool) error {
				establishedUserID = userID
				establishedVerified = verified
				return nil
			},
		},
		store: NewMemoryStore(),
		now:   time.Now,
	}

	req := httptest.NewRequest(http.MethodGet, "/invite/accept?teamId=acme&membershipId=membership-1&userId=user-1&secret=secret-1", nil)
	rec := httptest.NewRecorder()
	server.handleInvite(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if rec.Header().Get("Location") != appHomePath {
		t.Fatalf("location = %q, want %s", rec.Header().Get("Location"), appHomePath)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != "attesta_session" || cookies[0].Value != "invite-session" {
		t.Fatalf("cookies = %#v", cookies)
	}
	if acceptedTeamID != "acme" || acceptedMembershipID != "membership-1" || acceptedUserID != "user-1" || acceptedSecret != "secret-1" {
		t.Fatalf("accepted params = %q/%q/%q/%q", acceptedTeamID, acceptedMembershipID, acceptedUserID, acceptedSecret)
	}
	if establishedUserID != "user-1" || !establishedVerified {
		t.Fatalf("EstablishFromMailboxProof = %q/%v, want user-1/true", establishedUserID, establishedVerified)
	}
}

func TestHandleInviteAcceptRedirectsToInvitePasswordWhenUnset(t *testing.T) {
	now := time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)
	establishCalled := false
	server := &Server{
		identity: &fakeIdentityStore{
			acceptInviteFunc: func(ctx context.Context, teamID, membershipID, userID, secret string) (IdentitySession, error) {
				return fakeIdentitySession("invite-session", userID, now.Add(24*time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "invitee@example.com", PasswordSet: false}, nil
			},
			updateEmailVerificationFunc: func(ctx context.Context, userID string, verified bool) error {
				establishCalled = true
				return nil
			},
		},
		store: NewMemoryStore(),
		now:   time.Now,
	}

	req := httptest.NewRequest(http.MethodGet, "/invite/accept?teamId=acme&membershipId=membership-1&userId=user-1&secret=secret-1", nil)
	rec := httptest.NewRecorder()
	server.handleInvite(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if rec.Header().Get("Location") != "/invite/password" {
		t.Fatalf("location = %q, want /invite/password", rec.Header().Get("Location"))
	}
	if !establishCalled {
		t.Fatal("expected EstablishFromMailboxProof to be called")
	}
}

func TestHandleInviteAcceptEstablishFailureDoesNotRedirectHome(t *testing.T) {
	now := time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)
	server := &Server{
		identity: &fakeIdentityStore{
			acceptInviteFunc: func(ctx context.Context, teamID, membershipID, userID, secret string) (IdentitySession, error) {
				return fakeIdentitySession("invite-session", userID, now.Add(24*time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "invitee@example.com", PasswordSet: true}, nil
			},
			updateEmailVerificationFunc: func(ctx context.Context, userID string, verified bool) error {
				return errors.New("establish boom")
			},
		},
		store: NewMemoryStore(),
		now:   time.Now,
	}

	req := httptest.NewRequest(http.MethodGet, "/invite/accept?teamId=acme&membershipId=membership-1&userId=user-1&secret=secret-1", nil)
	rec := httptest.NewRecorder()
	server.handleInvite(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Fatalf("location = %q, want no redirect to %s on establish failure", loc, appHomePath)
	}
	cookies := rec.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "attesta_session" {
			t.Fatalf("session cookie written on establish failure: %#v", c)
		}
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
			store: NewMemoryStore(),
			now:   time.Now,
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
			store: NewMemoryStore(),
			now:   time.Now,
		}
		req := httptest.NewRequest(http.MethodGet, "/invite/accept?teamId=acme&membershipId=membership-1&userId=user-1&secret=secret-1", nil)
		rec := httptest.NewRecorder()
		server.handleInvite(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("legacy path removed", func(t *testing.T) {
		server := &Server{identity: &fakeIdentityStore{}, now: time.Now}
		req := httptest.NewRequest(http.MethodGet, "/invite/legacy-token", nil)
		rec := httptest.NewRecorder()
		server.handleInvite(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})
}
