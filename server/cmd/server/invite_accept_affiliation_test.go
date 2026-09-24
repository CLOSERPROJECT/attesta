package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandleInviteAcceptAffiliationGate(t *testing.T) {
	now := time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)

	t.Run("cross-org affiliated rejects without AcceptInvite", func(t *testing.T) {
		acceptCalled := false
		server := &Server{
			identity: &fakeIdentityStore{
				getUserByIDFunc: func(_ context.Context, userID string) (IdentityUser, error) {
					return IdentityUser{ID: userID, Email: "member@example.com", OrgSlug: "acme"}, nil
				},
				getOrganizationBySlugFunc: func(_ context.Context, slug string) (*IdentityOrg, error) {
					if slug != "acme" {
						return nil, ErrIdentityNotFound
					}
					return &IdentityOrg{ID: "team-acme", Slug: "acme", Name: "Acme"}, nil
				},
				acceptInviteFunc: func(_ context.Context, _, _, _, _ string) (IdentitySession, error) {
					acceptCalled = true
					return IdentitySession{}, errors.New("should not be called")
				},
			},
			now: time.Now,
		}
		req := httptest.NewRequest(http.MethodGet, "/invite/accept?teamId=team-other&membershipId=membership-1&userId=user-1&secret=secret-1", nil)
		rec := httptest.NewRecorder()
		server.handleInvite(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
		if !strings.Contains(rec.Body.String(), "already belongs to another organization") {
			t.Fatalf("body = %q", rec.Body.String())
		}
		if acceptCalled {
			t.Fatal("AcceptInvite must not be called for cross-org invite")
		}
	})

	t.Run("same-org affiliated allows AcceptInvite", func(t *testing.T) {
		acceptCalled := false
		server := &Server{
			identity: &fakeIdentityStore{
				getUserByIDFunc: func(_ context.Context, userID string) (IdentityUser, error) {
					return IdentityUser{ID: userID, Email: "member@example.com", OrgSlug: "acme"}, nil
				},
				getOrganizationBySlugFunc: func(_ context.Context, slug string) (*IdentityOrg, error) {
					if slug != "acme" {
						return nil, ErrIdentityNotFound
					}
					return &IdentityOrg{ID: "team-acme", Slug: "acme", Name: "Acme"}, nil
				},
				acceptInviteFunc: func(_ context.Context, teamID, membershipID, userID, secret string) (IdentitySession, error) {
					acceptCalled = true
					return fakeIdentitySession("invite-session", userID, now.Add(24*time.Hour)), nil
				},
				getCurrentUserFunc: func(_ context.Context, _ string) (IdentityUser, error) {
					return IdentityUser{ID: "user-1", Email: "member@example.com", OrgSlug: "acme", PasswordSet: true}, nil
				},
			},
			now: time.Now,
		}
		req := httptest.NewRequest(http.MethodGet, "/invite/accept?teamId=team-acme&membershipId=membership-1&userId=user-1&secret=secret-1", nil)
		rec := httptest.NewRecorder()
		server.handleInvite(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusSeeOther, rec.Body.String())
		}
		if !acceptCalled {
			t.Fatal("AcceptInvite must be called for same-org re-invite")
		}
	})

	t.Run("unaffiliated allows AcceptInvite", func(t *testing.T) {
		acceptCalled := false
		server := &Server{
			identity: &fakeIdentityStore{
				getUserByIDFunc: func(_ context.Context, userID string) (IdentityUser, error) {
					return IdentityUser{ID: userID, Email: "new@example.com"}, nil
				},
				acceptInviteFunc: func(_ context.Context, teamID, membershipID, userID, secret string) (IdentitySession, error) {
					acceptCalled = true
					return fakeIdentitySession("invite-session", userID, now.Add(24*time.Hour)), nil
				},
				getCurrentUserFunc: func(_ context.Context, _ string) (IdentityUser, error) {
					return IdentityUser{ID: "user-1", Email: "new@example.com", PasswordSet: true}, nil
				},
			},
			now: time.Now,
		}
		req := httptest.NewRequest(http.MethodGet, "/invite/accept?teamId=team-acme&membershipId=membership-1&userId=user-1&secret=secret-1", nil)
		rec := httptest.NewRecorder()
		server.handleInvite(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
		}
		if !acceptCalled {
			t.Fatal("AcceptInvite must be called for unaffiliated invitee")
		}
	})

	t.Run("get organization by slug failure returns 500", func(t *testing.T) {
		acceptCalled := false
		server := &Server{
			identity: &fakeIdentityStore{
				getUserByIDFunc: func(_ context.Context, userID string) (IdentityUser, error) {
					return IdentityUser{ID: userID, Email: "member@example.com", OrgSlug: "acme"}, nil
				},
				getOrganizationBySlugFunc: func(_ context.Context, _ string) (*IdentityOrg, error) {
					return nil, errors.New("get org by slug down")
				},
				acceptInviteFunc: func(_ context.Context, _, _, _, _ string) (IdentitySession, error) {
					acceptCalled = true
					return IdentitySession{}, errors.New("should not be called")
				},
			},
			now: time.Now,
		}
		req := httptest.NewRequest(http.MethodGet, "/invite/accept?teamId=team-other&membershipId=membership-1&userId=user-1&secret=secret-1", nil)
		rec := httptest.NewRecorder()
		server.handleInvite(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
		if acceptCalled {
			t.Fatal("AcceptInvite must not be called when get org by slug fails")
		}
	})
}
