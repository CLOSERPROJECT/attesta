package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestPageBaseForUserHasAttentionOpenInvitation(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	identity := &fakeIdentityStore{
		listUserMembershipsFunc: func(_ context.Context, userID string) ([]IdentityMembership, error) {
			if userID != "invitee-1" {
				return nil, nil
			}
			return []IdentityMembership{{
				ID:        "m1",
				OrgSlug:   "acme",
				OrgName:   "Acme",
				UserID:    "invitee-1",
				Confirmed: false,
				InvitedAt: now,
			}}, nil
		},
	}
	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	base := server.pageBaseForUser(&AccountUser{
		IdentityUserID: "invitee-1",
		Email:          "invitee@example.com",
		Status:         "active",
	}, "onboarding_body", "", "")
	if !base.HasAttention {
		t.Fatal("expected HasAttention for open Invitation")
	}
}

func TestPageBaseForUserHasAttentionWaitingDoesNotLight(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	if _, err := store.InsertJoinRequest(context.Background(), JoinRequest{
		ID:              primitive.NewObjectID(),
		RequesterUserID: "waiter-1",
		RequesterEmail:  "waiter@example.com",
		OrgSlug:         "acme",
		Status:          AffiliationStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("InsertJoinRequest: %v", err)
	}
	server := &Server{
		identity: &fakeIdentityStore{
			listUserMembershipsFunc: func(_ context.Context, _ string) ([]IdentityMembership, error) {
				return nil, nil
			},
		},
		store:       store,
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	base := server.pageBaseForUser(&AccountUser{
		IdentityUserID: "waiter-1",
		Email:          "waiter@example.com",
		Status:         "active",
	}, "onboarding_body", "", "")
	if base.HasAttention {
		t.Fatal("Waiting must not set HasAttention")
	}
}

func TestLayoutAccountMenuShowsAttentionDot(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var withDot bytes.Buffer
	if err := tmpl.ExecuteTemplate(&withDot, "layout.html", PageBase{ShowLogout: true, HasAttention: true}); err != nil {
		t.Fatalf("ExecuteTemplate: %v", err)
	}
	body := withDot.String()
	if !strings.Contains(body, `attention-dot`) {
		t.Fatalf("expected attention-dot when HasAttention, got:\n%s", body)
	}
	if !strings.Contains(body, `has-attention`) {
		t.Fatalf("expected has-attention class on trigger, got:\n%s", body)
	}

	var withoutDot bytes.Buffer
	if err := tmpl.ExecuteTemplate(&withoutDot, "layout.html", PageBase{ShowLogout: true}); err != nil {
		t.Fatalf("ExecuteTemplate: %v", err)
	}
	if strings.Contains(withoutDot.String(), `attention-dot`) {
		t.Fatalf("expected no attention-dot when HasAttention false, got:\n%s", withoutDot.String())
	}
}

func TestOnboardingHubAttentionDotFollowsInvitation(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-attention-invite"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-attention",
		Email:          "attention@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	identity.getOrganizationBySlugFunc = func(ctx context.Context, slug string) (*IdentityOrg, error) {
		if slug != "acme" {
			return nil, ErrIdentityNotFound
		}
		return &IdentityOrg{ID: "team-acme", Slug: "acme", Name: "Acme Org", Roles: []IdentityRole{{Slug: "viewer", Name: "Viewer"}}}, nil
	}
	pendingMemberships := []IdentityMembership{{
		ID: "membership-attention-1", TeamID: "team-acme", OrgSlug: "acme", OrgName: "Acme Org",
		UserID: "user-attention", Email: "attention@example.com", RoleSlugs: []string{"viewer"},
		Confirmed: false, InvitedAt: now,
	}}
	identity.listUserMembershipsFunc = func(ctx context.Context, userID string) ([]IdentityMembership, error) {
		if userID != "user-attention" {
			return nil, nil
		}
		return append([]IdentityMembership(nil), pendingMemberships...), nil
	}
	identity.deleteOrganizationMembershipAsAdminFunc = func(ctx context.Context, orgSlug, membershipID string) error {
		pendingMemberships = nil
		return nil
	}

	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	getWithInvite := httptest.NewRequest(http.MethodGet, "/my/onboarding", nil)
	getWithInvite.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	getRec := httptest.NewRecorder()
	server.handleMyRoutes(getRec, getWithInvite)
	if getRec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", getRec.Code, getRec.Body.String())
	}
	if !strings.Contains(getRec.Body.String(), `attention-dot`) {
		t.Fatalf("expected attention-dot with open Invitation, got:\n%s", getRec.Body.String())
	}

	rejectReq := httptest.NewRequest(http.MethodPost, "/my/onboarding", strings.NewReader("intent=reject_invite&membership_id=membership-attention-1"))
	rejectReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rejectReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rejectRec := httptest.NewRecorder()
	server.handleMyRoutes(rejectRec, rejectReq)
	if rejectRec.Code != http.StatusSeeOther {
		t.Fatalf("reject status=%d loc=%q body=%q", rejectRec.Code, rejectRec.Header().Get("Location"), rejectRec.Body.String())
	}

	getAfter := httptest.NewRequest(http.MethodGet, "/my/onboarding", nil)
	getAfter.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	afterRec := httptest.NewRecorder()
	server.handleMyRoutes(afterRec, getAfter)
	if afterRec.Code != http.StatusOK {
		t.Fatalf("after status=%d body=%q", afterRec.Code, afterRec.Body.String())
	}
	if strings.Contains(afterRec.Body.String(), `attention-dot`) {
		t.Fatalf("expected no attention-dot after declining Invitation, got:\n%s", afterRec.Body.String())
	}
}

func TestOnboardingHubWaitingDoesNotShowAttentionDot(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-attention-waiting"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-waiting",
		Email:          "waiting@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	store := NewMemoryStore()
	if _, err := store.InsertJoinRequest(context.Background(), JoinRequest{
		ID:              primitive.NewObjectID(),
		RequesterUserID: "user-waiting",
		RequesterEmail:  "waiting@example.com",
		OrgSlug:         "acme",
		RoleSlugs:       []string{"viewer"},
		Status:          AffiliationStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("InsertJoinRequest: %v", err)
	}
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	identity.getOrganizationBySlugFunc = func(ctx context.Context, slug string) (*IdentityOrg, error) {
		return &IdentityOrg{ID: "team-acme", Slug: "acme", Name: "Acme Org", Roles: []IdentityRole{{Slug: "viewer", Name: "Viewer"}}}, nil
	}
	identity.listUserMembershipsFunc = func(ctx context.Context, _ string) ([]IdentityMembership, error) {
		return nil, nil
	}

	server := &Server{
		identity:    identity,
		store:       store,
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/my/onboarding", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Waiting for organization admin review") {
		t.Fatalf("expected Waiting join content, got:\n%s", body)
	}
	if strings.Contains(body, `attention-dot`) {
		t.Fatalf("Waiting must not show attention-dot, got:\n%s", body)
	}
}
