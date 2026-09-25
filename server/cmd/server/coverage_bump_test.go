package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestIdentityMappingEmptySlugEdges(t *testing.T) {
	if got := decodeIdentityRoleLabels([]string{identityRoleLabelPrefix, identityRoleLabelPrefix + "viewer"}); len(got) != 1 || got[0] != "viewer" {
		t.Fatalf("decodeIdentityRoleLabels = %#v", got)
	}
	if got := encodeInviteMembershipRoles([]string{"", "viewer"}, false); len(got) != 2 || got[1] != identityInviteRolePrefix+"viewer" {
		t.Fatalf("encodeInviteMembershipRoles = %#v", got)
	}
	if got := uniqueIdentityStrings([]string{"", "a", "a", " b "}); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("uniqueIdentityStrings = %#v", got)
	}
}

func TestAffiliationAcceptInvitationAndPendingInviteEdges(t *testing.T) {
	aff := NewAffiliation(&fakeIdentityStore{}, NewMemoryStore(), nil, nil, nil)
	if _, err := aff.AcceptInvitation(context.Background(), InvitationAccept{}); !errors.Is(err, ErrAffiliationNotFound) {
		t.Fatalf("empty accept = %v", err)
	}
	if invites, err := aff.ListPendingInvitesForUser(context.Background(), ""); err != nil || invites != nil {
		t.Fatalf("empty user invites = %#v %v", invites, err)
	}
	affNil := NewAffiliation(nil, NewMemoryStore(), nil, nil, nil)
	if invites, err := affNil.ListPendingInvitesForUser(context.Background(), "u"); err != nil || invites != nil {
		t.Fatalf("nil identity invites = %#v %v", invites, err)
	}
}

func TestOnboardingHubOrgAdminInviteRoleLabels(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-invite-admin-labels"
	account := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-invitee",
		Email:          "invitee@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: account})
	identity.listUserMembershipsFunc = func(ctx context.Context, userID string) ([]IdentityMembership, error) {
		return []IdentityMembership{
			{ID: "m-admin-only", TeamID: "t1", OrgSlug: "acme", OrgName: "Acme", UserID: userID, IsOrgAdmin: true, Confirmed: false, InvitedAt: now},
			{ID: "m-admin-roles", TeamID: "t1", OrgSlug: "acme", OrgName: "Acme", UserID: userID, RoleSlugs: []string{"viewer"}, IsOrgAdmin: true, Confirmed: false, InvitedAt: now},
		}, nil
	}
	identity.getOrganizationBySlugFunc = func(ctx context.Context, slug string) (*IdentityOrg, error) {
		return &IdentityOrg{Slug: "acme", Name: "Acme", Roles: []IdentityRole{{Slug: "viewer", Name: "Viewer"}}}, nil
	}
	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	req := httptest.NewRequest(http.MethodGet, onboardingPath(), nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Org admin") || !strings.Contains(body, "Org admin, Viewer") {
		t.Fatalf("expected org-admin invite labels, got:\n%s", body)
	}
}

func TestBuildOnboardingJoinDialogViewEdges(t *testing.T) {
	server := &Server{tmpl: parseTestTemplates(t)}
	req := httptest.NewRequest(http.MethodGet, "/my/onboarding/join", nil)
	rec := httptest.NewRecorder()
	view, ok := server.buildOnboardingJoinDialogView(rec, req, &AccountUser{IdentityUserID: "u"}, "", "", "q", 0)
	if !ok || view.CurrentPage != 1 || view.FormError != "organization not found" {
		t.Fatalf("empty selected = %#v ok=%v", view, ok)
	}
	view, ok = server.buildOnboardingJoinDialogView(rec, req, &AccountUser{IdentityUserID: "u"}, "keep me", "", "q", 2)
	if !ok || view.FormError != "keep me" || view.CurrentPage != 2 {
		t.Fatalf("preserve form error = %#v ok=%v", view, ok)
	}
}
