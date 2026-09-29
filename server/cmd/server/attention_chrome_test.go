package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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
	if !strings.Contains(body, `href="/my"`) || !strings.Contains(body, `Dashboard (needs attention)`) {
		t.Fatalf("expected Dashboard menu item Attention chrome, got:\n%s", body)
	}
	if strings.Count(body, `attention-dot`) < 2 {
		t.Fatalf("expected attention-dot on account trigger and Dashboard item, got %d:\n%s", strings.Count(body, `attention-dot`), body)
	}

	var withoutDot bytes.Buffer
	if err := tmpl.ExecuteTemplate(&withoutDot, "layout.html", PageBase{ShowLogout: true}); err != nil {
		t.Fatalf("ExecuteTemplate: %v", err)
	}
	if strings.Contains(withoutDot.String(), `attention-dot`) {
		t.Fatalf("expected no attention-dot when HasAttention false, got:\n%s", withoutDot.String())
	}
}

func TestLayoutAccountMenuJoinRequestSectionDots(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "layout.html", PageBase{
		ShowLogout:              true,
		ShowMyOrgLink:           true,
		HasAttention:            true,
		HasJoinRequestAttention: true,
	}); err != nil {
		t.Fatalf("ExecuteTemplate: %v", err)
	}
	body := buf.String()
	if !strings.Contains(body, `My organization (needs attention)`) {
		t.Fatalf("expected My organization section Attention chrome, got:\n%s", body)
	}
	if strings.Count(body, `attention-dot`) < 3 {
		t.Fatalf("expected trigger + Dashboard + My organization dots, got %d:\n%s", strings.Count(body, `attention-dot`), body)
	}
}

func TestPageBaseForUserHasJoinRequestAttention(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	if _, err := store.InsertJoinRequest(context.Background(), JoinRequest{
		ID:              primitive.NewObjectID(),
		RequesterUserID: "joiner-1",
		RequesterEmail:  "joiner@example.com",
		OrgSlug:         "acme",
		RoleSlugs:       []string{"viewer"},
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
	adminBase := server.pageBaseForUser(&AccountUser{
		IdentityUserID: "admin-1",
		Email:          "admin@example.com",
		OrgSlug:        "acme",
		RoleSlugs:      []string{"org-admin"},
		Status:         "active",
	}, "home_picker_body", "", "")
	if !adminBase.HasAttention || !adminBase.HasJoinRequestAttention {
		t.Fatalf("Org admin with pending Join must light Attention chrome: %+v", adminBase)
	}

	memberBase := server.pageBaseForUser(&AccountUser{
		IdentityUserID: "member-1",
		Email:          "member@example.com",
		OrgSlug:        "acme",
		RoleSlugs:      []string{"viewer"},
		Status:         "active",
	}, "home_picker_body", "", "")
	if memberBase.HasAttention || memberBase.HasJoinRequestAttention {
		t.Fatalf("Member must not see Join-request Attention: %+v", memberBase)
	}
}

func TestOperatorHomeJoinRequestAttentionBandAndResolve(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-join-attention"
	store := NewMemoryStore()
	saved, err := store.InsertJoinRequest(context.Background(), JoinRequest{
		RequesterUserID: "joiner-1",
		RequesterEmail:  "joiner@example.com",
		OrgSlug:         "acme",
		RoleSlugs:       []string{"viewer"},
		Status:          AffiliationStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		t.Fatalf("InsertJoinRequest: %v", err)
	}

	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")

	identity := &fakeIdentityStore{
		getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
			return fakeIdentitySession(sessionSecret, "admin-1", now.Add(time.Hour)), nil
		},
		getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
			return IdentityUser{
				ID: "admin-1", Email: "admin@acme.example", OrgSlug: "acme",
				Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active",
			}, nil
		},
		getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
			return &IdentityOrg{
				ID: "team-1", Slug: "acme", Name: "Acme Org",
				Roles: []IdentityRole{{Slug: "viewer", Name: "Viewer"}},
			}, nil
		},
		listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
			return []IdentityUser{{
				ID: "admin-1", Email: "admin@acme.example", OrgSlug: "acme", IsOrgAdmin: true, Status: "active",
			}}, nil
		},
		listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
			return nil, nil
		},
		getUserByIDFunc: func(ctx context.Context, userID string) (IdentityUser, error) {
			if userID != "joiner-1" {
				return IdentityUser{}, ErrIdentityNotFound
			}
			return IdentityUser{ID: "joiner-1", Email: "joiner@example.com", Status: "active"}, nil
		},
		addOrganizationUserByIDAsAdminFunc: func(ctx context.Context, orgSlug, userID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
			return IdentityMembership{ID: "mem-joiner", UserID: userID, OrgSlug: orgSlug, RoleSlugs: roleSlugs, Confirmed: true}, nil
		},
		updateUserLabelsFunc: func(ctx context.Context, userID string, labels []string) (IdentityUser, error) {
			return IdentityUser{ID: userID, Labels: labels, Status: "active"}, nil
		},
	}

	server := &Server{
		identity:    identity,
		store:       store,
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		configDir:   tempDir,
		now:         func() time.Time { return now },
	}

	getHome := httptest.NewRequest(http.MethodGet, "/my", nil)
	getHome.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	homeRec := httptest.NewRecorder()
	server.handleHome(homeRec, getHome)
	if homeRec.Code != http.StatusOK {
		t.Fatalf("home status=%d body=%q", homeRec.Code, homeRec.Body.String())
	}
	homeBody := homeRec.Body.String()
	for _, want := range []string{
		`aria-label="Attention"`,
		"Join requests",
		"joiner@example.com",
		`attention-dot`,
		`My organization (needs attention)`,
		`name="intent" value="approve_join"`,
		`name="next" value="/my"`,
	} {
		if !strings.Contains(homeBody, want) {
			t.Fatalf("expected %q on Operator home Attention band, got:\n%s", want, homeBody)
		}
	}

	membersReq := httptest.NewRequest(http.MethodGet, "/my/organization/members", nil)
	membersReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	membersRec := httptest.NewRecorder()
	server.handleOrgAdminPage(membersRec, membersReq)
	if membersRec.Code != http.StatusOK {
		t.Fatalf("members status=%d", membersRec.Code)
	}
	if !strings.Contains(membersRec.Body.String(), `Members (needs attention)`) {
		t.Fatalf("expected Members soft-nav Attention chrome, got:\n%s", membersRec.Body.String())
	}

	approve := httptest.NewRequest(http.MethodPost, "/my/organization/users", strings.NewReader(
		"intent=approve_join&request_id="+saved.ID.Hex()+"&next=/my",
	))
	approve.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	approve.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	approveRec := httptest.NewRecorder()
	server.handleOrgAdminUsers(approveRec, approve)
	if approveRec.Code != http.StatusSeeOther || approveRec.Header().Get("Location") != "/my" {
		t.Fatalf("approve redirect status=%d loc=%q", approveRec.Code, approveRec.Header().Get("Location"))
	}

	afterHome := httptest.NewRequest(http.MethodGet, "/my", nil)
	afterHome.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	afterRec := httptest.NewRecorder()
	server.handleHome(afterRec, afterHome)
	if afterRec.Code != http.StatusOK {
		t.Fatalf("after home status=%d", afterRec.Code)
	}
	afterBody := afterRec.Body.String()
	if strings.Contains(afterBody, "joiner@example.com") || strings.Contains(afterBody, `aria-label="Attention"`) {
		t.Fatalf("expected Join Attention band cleared after approve, got:\n%s", afterBody)
	}
	if strings.Contains(afterBody, `attention-dot`) {
		t.Fatalf("expected Attention dots cleared after approve, got:\n%s", afterBody)
	}
}

func TestOperatorHomeMemberNeverSeesJoinAttention(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	if _, err := store.InsertJoinRequest(context.Background(), JoinRequest{
		RequesterUserID: "joiner-1",
		RequesterEmail:  "joiner@example.com",
		OrgSlug:         "acme",
		RoleSlugs:       []string{"viewer"},
		Status:          AffiliationStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("InsertJoinRequest: %v", err)
	}
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")
	server := &Server{
		store: store,
		tmpl:  parseTestTemplates(t),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "member-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{
					ID: "member-1", Email: "member@acme.example", OrgSlug: "acme",
					Labels: []string{encodeIdentityRoleLabel("viewer")}, Status: "active",
				}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				return &IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme", Roles: []IdentityRole{{Slug: "viewer", Name: "Viewer"}}}, nil
			},
		},
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		configDir:   tempDir,
		now:         func() time.Time { return now },
	}
	req := httptest.NewRequest(http.MethodGet, "/my", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-member"})
	rec := httptest.NewRecorder()
	server.handleHome(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "Join requests") || strings.Contains(body, "joiner@example.com") || strings.Contains(body, `attention-dot`) {
		t.Fatalf("Member must not see Join-request Attention, got:\n%s", body)
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
