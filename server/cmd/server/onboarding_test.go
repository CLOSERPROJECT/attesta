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

func TestHandleOnboardingUnaffiliatedRendersHub(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-onboarding-hub"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-1",
		Email:          "newbie@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	server := &Server{
		identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
		store:       NewMemoryStore(),
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
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Get started",
		`href="/my/onboarding/join"`,
		`href="/my/onboarding/request-organization"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in onboarding hub, got:\n%s", want, body)
		}
	}
	for _, gone := range []string{
		"Pending join request",
		"Pending organization creation request",
		`name="intent"`,
		"Back to streams",
	} {
		if strings.Contains(body, gone) {
			t.Fatalf("hub without pending must not contain %q, got:\n%s", gone, body)
		}
	}
}

func TestHandleOnboardingAffiliatedRedirectsHome(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-onboarding-affiliated"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-2",
		Email:          "member@example.com",
		OrgSlug:        "acme",
		Status:         "active",
		CreatedAt:      now,
	}
	server := &Server{
		identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/my/onboarding", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if rec.Header().Get("Location") != "/my" {
		t.Fatalf("location = %q, want /my", rec.Header().Get("Location"))
	}
}

func TestHandleOnboardingHubPendingAndWithdraw(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-onboarding-pending"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-pending",
		Email:          "pending@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	store := NewMemoryStore()
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	identity.getOrganizationBySlugFunc = func(ctx context.Context, slug string) (*IdentityOrg, error) {
		if strings.TrimSpace(slug) != "acme" {
			return nil, ErrIdentityNotFound
		}
		return &IdentityOrg{
			ID:   "team-1",
			Slug: "acme",
			Name: "Acme Org",
			Roles: []IdentityRole{
				{Slug: "viewer", Name: "Viewer"},
			},
		}, nil
	}
	server := &Server{
		identity:    identity,
		store:       store,
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	if _, err := store.InsertJoinRequest(context.Background(), JoinRequest{
		RequesterUserID: "user-pending",
		RequesterEmail:  "pending@example.com",
		OrgSlug:         "acme",
		RoleSlugs:       []string{"viewer"},
		Status:          AffiliationStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("InsertJoinRequest: %v", err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/my/onboarding", nil)
	getReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	getRec := httptest.NewRecorder()
	server.handleMyRoutes(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("pending hub status = %d, want %d body=%q", getRec.Code, http.StatusOK, getRec.Body.String())
	}
	body := getRec.Body.String()
	for _, want := range []string{
		"Pending join request",
		"Acme Org",
		"Viewer",
		`id="undo-request-dialog"`,
		`onclick="document.getElementById('undo-request-dialog').showModal()"`,
		`name="intent" value="withdraw"`,
		"Undo request",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in pending hub, got:\n%s", want, body)
		}
	}
	for _, gone := range []string{
		`href="/my/onboarding/join"`,
		`href="/my/onboarding/request-organization"`,
		"Back to streams",
	} {
		if strings.Contains(body, gone) {
			t.Fatalf("pending hub must not contain %q, got:\n%s", gone, body)
		}
	}

	childReq := httptest.NewRequest(http.MethodGet, "/my/onboarding/join", nil)
	childReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	childRec := httptest.NewRecorder()
	server.handleMyRoutes(childRec, childReq)
	if childRec.Code != http.StatusSeeOther {
		t.Fatalf("child status = %d, want %d", childRec.Code, http.StatusSeeOther)
	}
	if loc := childRec.Header().Get("Location"); loc != "/my/onboarding" {
		t.Fatalf("child location = %q, want /my/onboarding", loc)
	}

	withdrawReq := httptest.NewRequest(http.MethodPost, "/my/onboarding", strings.NewReader("intent=withdraw"))
	withdrawReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withdrawReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	withdrawRec := httptest.NewRecorder()
	server.handleMyRoutes(withdrawRec, withdrawReq)

	if withdrawRec.Code != http.StatusSeeOther {
		t.Fatalf("withdraw status = %d, want %d body=%q", withdrawRec.Code, http.StatusSeeOther, withdrawRec.Body.String())
	}
	if loc := withdrawRec.Header().Get("Location"); loc != "/my/onboarding" {
		t.Fatalf("withdraw location = %q, want /my/onboarding", loc)
	}
	pending, err := server.affiliationService().PendingJoinRequestForUser(context.Background(), "user-pending")
	if err != nil {
		t.Fatalf("PendingJoinRequestForUser: %v", err)
	}
	if pending != nil {
		t.Fatalf("expected join request withdrawn, got %+v", pending)
	}

	afterReq := httptest.NewRequest(http.MethodGet, "/my/onboarding", nil)
	afterReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	afterRec := httptest.NewRecorder()
	server.handleMyRoutes(afterRec, afterReq)
	if afterRec.Code != http.StatusOK {
		t.Fatalf("after withdraw status = %d body=%q", afterRec.Code, afterRec.Body.String())
	}
	afterBody := afterRec.Body.String()
	if !strings.Contains(afterBody, `href="/my/onboarding/join"`) {
		t.Fatalf("expected join CTA after withdraw, got:\n%s", afterBody)
	}
	if strings.Contains(afterBody, "Pending join request") {
		t.Fatalf("expected no pending panel after withdraw, got:\n%s", afterBody)
	}
}

func TestHandleHomeRedirectsUnaffiliatedToOnboarding(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-home-unaffiliated"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-home",
		Email:          "home@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	server := &Server{
		identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/my", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "/my/onboarding" {
		t.Fatalf("location = %q, want /my/onboarding", loc)
	}
}

func TestHandleOnboardingJoinAndRequestPages(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-onboarding-forms"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-3",
		Email:          "joiner@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	identity.listOrganizationsPageFunc = func(ctx context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error) {
		return IdentityOrgPage{}, nil
	}
	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	t.Run("join form", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/my/onboarding/join", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusOK, rec.Body.String())
		}
		body := rec.Body.String()
		for _, want := range []string{
			"Join an organization",
			`class="back-link"`,
			"Get started",
			`href="/my/onboarding"`,
			`name="q"`,
			`hx-trigger="input changed delay:200ms, search"`,
			"No organizations yet",
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("expected %q, got:\n%s", want, body)
			}
		}
	})

	t.Run("request organization form", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/my/onboarding/request-organization", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusOK, rec.Body.String())
		}
		body := rec.Body.String()
		for _, want := range []string{
			"Request a new organization",
			"organization creation request",
			`name="name"`,
			"Submit organization creation request",
			`class="back-link"`,
			"Get started",
			`href="/my/onboarding"`,
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("expected %q, got:\n%s", want, body)
			}
		}
	})
}

func TestHandleOnboardingHubPendingInviteAcceptAndReject(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-onboarding-invite"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-invitee",
		Email:          "invitee@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	identity.getOrganizationBySlugFunc = func(ctx context.Context, slug string) (*IdentityOrg, error) {
		if strings.TrimSpace(slug) != "acme" {
			return nil, ErrIdentityNotFound
		}
		return &IdentityOrg{
			ID:   "team-acme",
			Slug: "acme",
			Name: "Acme Org",
			Roles: []IdentityRole{
				{Slug: "viewer", Name: "Viewer"},
			},
		}, nil
	}
	pendingMemberships := []IdentityMembership{{
		ID:         "membership-invite-1",
		TeamID:     "team-acme",
		OrgSlug:    "acme",
		OrgName:    "Acme Org",
		UserID:     "user-invitee",
		Email:      "invitee@example.com",
		RoleSlugs:  []string{"viewer"},
		Confirmed:  false,
		InvitedAt:  now,
	}}
	identity.listUserMembershipsFunc = func(ctx context.Context, userID string) ([]IdentityMembership, error) {
		if userID != "user-invitee" {
			return nil, nil
		}
		return append([]IdentityMembership(nil), pendingMemberships...), nil
	}
	var deletedMembershipID string
	var grantedUserID string
	var stampedLabels []string
	identity.deleteOrganizationMembershipAsAdminFunc = func(ctx context.Context, orgSlug, membershipID string) error {
		if orgSlug != "acme" {
			t.Fatalf("delete orgSlug=%q", orgSlug)
		}
		deletedMembershipID = membershipID
		pendingMemberships = nil
		return nil
	}
	identity.addOrganizationUserByIDAsAdminFunc = func(ctx context.Context, orgSlug, userID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
		grantedUserID = userID
		return IdentityMembership{ID: "membership-confirmed", OrgSlug: orgSlug, UserID: userID, RoleSlugs: roleSlugs, Confirmed: true, IsOrgAdmin: isOrgAdmin}, nil
	}
	identity.updateUserLabelsFunc = func(ctx context.Context, userID string, labels []string) (IdentityUser, error) {
		stampedLabels = append([]string(nil), labels...)
		return IdentityUser{ID: userID, Email: "invitee@example.com", Labels: labels, OrgSlug: "acme"}, nil
	}
	identity.getUserByIDFunc = func(ctx context.Context, userID string) (IdentityUser, error) {
		return IdentityUser{ID: userID, Email: "invitee@example.com"}, nil
	}

	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	getReq := httptest.NewRequest(http.MethodGet, "/my/onboarding", nil)
	getReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	getRec := httptest.NewRecorder()
	server.handleMyRoutes(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", getRec.Code, getRec.Body.String())
	}
	body := getRec.Body.String()
	for _, want := range []string{
		"Organization invite",
		"Acme Org",
		"Viewer",
		`class="onboarding-pending onboarding-invite"`,
		`id="accept-invite-dialog-0"`,
		`id="reject-invite-dialog-0"`,
		`name="intent" value="accept_invite"`,
		`name="intent" value="reject_invite"`,
		`name="membership_id" value="membership-invite-1"`,
		`href="/my/onboarding/join"`,
		`href="/my/onboarding/request-organization"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in invite hub, got:\n%s", want, body)
		}
	}

	acceptReq := httptest.NewRequest(http.MethodPost, "/my/onboarding", strings.NewReader("intent=accept_invite&membership_id=membership-invite-1"))
	acceptReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	acceptReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	acceptRec := httptest.NewRecorder()
	pendingMemberships = []IdentityMembership{{
		ID: "membership-invite-1", TeamID: "team-acme", OrgSlug: "acme", OrgName: "Acme Org",
		UserID: "user-invitee", RoleSlugs: []string{"viewer"}, Confirmed: false, InvitedAt: now,
	}}
	server.handleMyRoutes(acceptRec, acceptReq)
	if acceptRec.Code != http.StatusSeeOther || acceptRec.Header().Get("Location") != "/my" {
		t.Fatalf("accept status=%d loc=%q", acceptRec.Code, acceptRec.Header().Get("Location"))
	}
	if deletedMembershipID != "membership-invite-1" || grantedUserID != "user-invitee" {
		t.Fatalf("deleted=%q granted=%q", deletedMembershipID, grantedUserID)
	}
	if len(stampedLabels) == 0 || stampedLabels[0] != encodeIdentityRoleLabel("viewer") {
		t.Fatalf("labels=%#v", stampedLabels)
	}

	pendingMemberships = []IdentityMembership{{
		ID: "membership-invite-1", TeamID: "team-acme", OrgSlug: "acme", OrgName: "Acme Org",
		UserID: "user-invitee", RoleSlugs: []string{"viewer"}, Confirmed: false, InvitedAt: now,
	}}
	deletedMembershipID = ""
	rejectReq := httptest.NewRequest(http.MethodPost, "/my/onboarding", strings.NewReader("intent=reject_invite&membership_id=membership-invite-1"))
	rejectReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rejectReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rejectRec := httptest.NewRecorder()
	server.handleMyRoutes(rejectRec, rejectReq)
	if rejectRec.Code != http.StatusSeeOther || rejectRec.Header().Get("Location") != "/my/onboarding" {
		t.Fatalf("reject status=%d loc=%q", rejectRec.Code, rejectRec.Header().Get("Location"))
	}
	if deletedMembershipID != "membership-invite-1" {
		t.Fatalf("reject deleted=%q", deletedMembershipID)
	}
}

func TestAffiliationPendingInviteAcceptReject(t *testing.T) {
	ctx := context.Background()
	identity := &fakeIdentityStore{}
	aff := NewAffiliation(identity, NewMemoryStore(), nil, nil, nil)

	identity.listUserMembershipsFunc = func(ctx context.Context, userID string) ([]IdentityMembership, error) {
		return []IdentityMembership{{
			ID: "m1", TeamID: "team-acme", OrgSlug: "acme", OrgName: "Acme",
			UserID: userID, RoleSlugs: []string{"viewer"}, IsOrgAdmin: true, Confirmed: false,
		}}, nil
	}
	var deleted string
	identity.deleteOrganizationMembershipAsAdminFunc = func(ctx context.Context, orgSlug, membershipID string) error {
		deleted = orgSlug + ":" + membershipID
		return nil
	}
	identity.addOrganizationUserByIDAsAdminFunc = func(ctx context.Context, orgSlug, userID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
		if !isOrgAdmin || len(roleSlugs) != 1 || roleSlugs[0] != "viewer" {
			t.Fatalf("grant roles=%v admin=%v", roleSlugs, isOrgAdmin)
		}
		return IdentityMembership{ID: "m2", Confirmed: true}, nil
	}
	identity.updateUserLabelsFunc = func(ctx context.Context, userID string, labels []string) (IdentityUser, error) {
		return IdentityUser{ID: userID, Labels: labels}, nil
	}
	identity.getUserByIDFunc = func(ctx context.Context, userID string) (IdentityUser, error) {
		return IdentityUser{ID: userID, Email: "a@example.com"}, nil
	}

	user := IdentityUser{ID: "user-1", Email: "a@example.com"}
	invites, err := aff.ListPendingInvitesForUser(ctx, user.ID)
	if err != nil || len(invites) != 1 || !invites[0].IsOrgAdmin {
		t.Fatalf("invites=%#v err=%v", invites, err)
	}
	if err := aff.AcceptPendingInvite(ctx, user, "m1"); err != nil {
		t.Fatalf("AcceptPendingInvite: %v", err)
	}
	if deleted != "acme:m1" {
		t.Fatalf("deleted=%q", deleted)
	}
	if err := aff.RejectPendingInvite(ctx, user, "missing"); !errors.Is(err, ErrAffiliationNotFound) {
		t.Fatalf("RejectPendingInvite missing = %v", err)
	}
}
