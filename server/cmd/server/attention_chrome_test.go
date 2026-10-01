package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestPopulateAttentionNilSafe(t *testing.T) {
	var server *Server
	server.populateAttention(&PageBase{}, &AccountUser{Email: "a@example.com"})
	(&Server{}).populateAttention(nil, &AccountUser{Email: "a@example.com"})
	(&Server{}).populateAttention(&PageBase{}, nil)
}

func TestPopulateAttentionSwallowsQueueErrors(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")
	wantErr := errors.New("attention store down")

	t.Run("platform admin org creation", func(t *testing.T) {
		server := &Server{
			identity:    &fakeIdentityStore{},
			store:       &failingPendingOrgCreationStore{MemoryStore: NewMemoryStore(), err: wantErr},
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         time.Now,
		}
		pa := platformAdminAccountUser()
		if pa == nil {
			t.Fatal("expected platformAdminAccountUser")
		}
		base := &PageBase{}
		server.populateAttention(base, pa)
		if base.HasAttention || base.HasOrgCreationAttention {
			t.Fatalf("error path must leave Attention unset: %+v", base)
		}
	})

	t.Run("org admin join requests", func(t *testing.T) {
		server := &Server{
			identity:    &fakeIdentityStore{},
			store:       &failingPendingJoinStore{MemoryStore: NewMemoryStore(), err: wantErr},
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         time.Now,
		}
		base := &PageBase{}
		server.populateAttention(base, &AccountUser{
			IdentityUserID: "admin-1",
			Email:          "admin@acme.example",
			OrgSlug:        "acme",
			RoleSlugs:      []string{"org-admin"},
			Status:         "active",
		})
		if base.HasAttention || base.HasJoinRequestAttention {
			t.Fatalf("error path must leave Attention unset: %+v", base)
		}
	})
}

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

func TestLayoutAccountMenuSectionDotsOnlyOnDashboard(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "layout.html", PageBase{
		ShowLogout:              true,
		ShowAdminLink:           true,
		ShowMyOrgLink:           true,
		HasAttention:            true,
		HasJoinRequestAttention: true,
		HasOrgCreationAttention: true,
	}); err != nil {
		t.Fatalf("ExecuteTemplate: %v", err)
	}
	body := buf.String()
	if !strings.Contains(body, `Dashboard (needs attention)`) {
		t.Fatalf("expected Dashboard section Attention chrome, got:\n%s", body)
	}
	if strings.Contains(body, `Admin (needs attention)`) || strings.Contains(body, `My organization (needs attention)`) {
		t.Fatalf("account-menu Admin/My organization must not carry section Attention dots, got:\n%s", body)
	}
	if strings.Count(body, `attention-dot`) != 2 {
		t.Fatalf("expected trigger + Dashboard dots only, got %d:\n%s", strings.Count(body, `attention-dot`), body)
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

func TestPageBaseForUserHasOrgCreationAttention(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	if _, err := store.InsertOrganizationCreationRequest(context.Background(), OrganizationCreationRequest{
		ID:              primitive.NewObjectID(),
		RequesterUserID: "founder-1",
		RequesterEmail:  "founder@example.com",
		ProposedName:    "New Co",
		ProposedSlug:    "new-co",
		Status:          AffiliationStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("InsertOrganizationCreationRequest: %v", err)
	}
	server := &Server{
		identity:    &fakeIdentityStore{},
		store:       store,
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	// Platform admin has no Appwrite IdentityUserID — Attention must still resolve.
	pa := platformAdminAccountUser()
	if pa == nil {
		t.Fatal("expected platformAdminAccountUser")
	}
	if strings.TrimSpace(pa.IdentityUserID) != "" {
		t.Fatalf("platform admin must have empty IdentityUserID for this regression, got %q", pa.IdentityUserID)
	}
	base := server.pageBaseForUser(pa, "home_picker_body", "", "")
	if !base.HasAttention || !base.HasOrgCreationAttention {
		t.Fatalf("platform admin with pending org creation must light Attention chrome: %+v", base)
	}
	if base.HasJoinRequestAttention {
		t.Fatalf("platform admin must not get Join-request Attention: %+v", base)
	}

	emptyStore := &Server{
		identity:    &fakeIdentityStore{},
		store:       NewMemoryStore(),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	cleared := emptyStore.pageBaseForUser(pa, "home_picker_body", "", "")
	if cleared.HasAttention || cleared.HasOrgCreationAttention {
		t.Fatalf("empty org-creation queue must clear Attention chrome: %+v", cleared)
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
		`Dashboard (needs attention)`,
		`name="intent" value="approve_join"`,
		`name="next" value="/my"`,
	} {
		if !strings.Contains(homeBody, want) {
			t.Fatalf("expected %q on Operator home Attention band, got:\n%s", want, homeBody)
		}
	}
	if strings.Contains(homeBody, `My organization (needs attention)`) {
		t.Fatalf("account-menu My organization must not carry section Attention, got:\n%s", homeBody)
	}
	joinIdx := strings.Index(homeBody, "Join requests")
	chooseIdx := strings.Index(homeBody, "Choose a stream")
	if joinIdx < 0 || chooseIdx < 0 || joinIdx > chooseIdx {
		t.Fatalf("expected Join requests above Choose a stream, join=%d choose=%d body:\n%s", joinIdx, chooseIdx, homeBody)
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

func TestPageBaseForUserHasAttentionYourTurnStream(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")
	store := NewMemoryStore()
	store.SeedProcess(Process{
		WorkflowKey: "workflow",
		Name:        "Batch needing action",
		Status:      processStatusActive,
		CreatedAt:   now,
		Progress: map[string]ProcessStep{
			"1.1": {State: "pending"},
		},
	})

	server := &Server{
		identity:    &fakeIdentityStore{},
		store:       store,
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		configDir:   tempDir,
		now:         func() time.Time { return now },
	}

	memberBase := server.pageBaseForUser(&AccountUser{
		IdentityUserID: "member-1",
		Email:          "member@org1.example",
		OrgSlug:        "org1",
		RoleSlugs:      []string{"dep1"},
		Status:         "active",
	}, "home_picker_body", "", "")
	if !memberBase.HasAttention {
		t.Fatal("affiliated Member with your-turn Stream must light HasAttention")
	}

	// Upcoming: another role's turn — Member with no matching role must not light.
	otherBase := server.pageBaseForUser(&AccountUser{
		IdentityUserID: "other-1",
		Email:          "other@org1.example",
		OrgSlug:        "org1",
		RoleSlugs:      []string{"viewer"},
		Status:         "active",
	}, "home_picker_body", "", "")
	if otherBase.HasAttention {
		t.Fatal("Upcoming (no actionable Substep) must not light HasAttention")
	}

	pa := platformAdminAccountUser()
	if pa == nil {
		t.Fatal("expected platformAdminAccountUser")
	}
	paBase := server.pageBaseForUser(pa, "home_picker_body", "", "")
	if paBase.HasAttention {
		t.Fatal("platform admin must not get stream Attention on HasAttention")
	}
}

func TestOperatorHomeYourTurnStreamAttentionBandAndResolve(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-your-turn"
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")

	store := NewMemoryStore()
	processID := store.SeedProcess(Process{
		WorkflowKey: "workflow",
		Name:        "Batch needing action",
		Status:      processStatusActive,
		CreatedAt:   now,
		Progress: map[string]ProcessStep{
			"1.1": {State: "pending"},
		},
	})
	instanceHref := streamInstanceSubstepPath("workflow", processID.Hex(), "1.1")

	if _, err := store.InsertJoinRequest(context.Background(), JoinRequest{
		RequesterUserID: "joiner-1",
		RequesterEmail:  "joiner@example.com",
		OrgSlug:         "org1",
		RoleSlugs:       []string{"dep1"},
		Status:          AffiliationStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("InsertJoinRequest: %v", err)
	}

	user := AccountUser{
		IdentityUserID: "admin-1",
		Email:          "admin@org1.example",
		OrgSlug:        "org1",
		RoleSlugs:      []string{"org-admin", "dep1"},
		Status:         "active",
	}
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	identity.getOrganizationBySlugFunc = func(ctx context.Context, slug string) (*IdentityOrg, error) {
		return &IdentityOrg{
			ID: "team-1", Slug: "org1", Name: "Org 1",
			Roles: []IdentityRole{{Slug: "dep1", Name: "Department 1"}},
		}, nil
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
		"Your turn",
		"Batch needing action",
		instanceHref,
		`attention-dot`,
		`Dashboard (needs attention)`,
	} {
		if !strings.Contains(homeBody, want) {
			t.Fatalf("expected %q on Operator home, got:\n%s", want, homeBody)
		}
	}
	joinIdx := strings.Index(homeBody, "Join requests")
	yourTurnIdx := strings.Index(homeBody, "Your turn")
	chooseIdx := strings.Index(homeBody, "Choose a stream")
	if joinIdx < 0 || yourTurnIdx < 0 || chooseIdx < 0 || !(joinIdx < yourTurnIdx && yourTurnIdx < chooseIdx) {
		t.Fatalf("expected Join above Your turn above Choose a stream; join=%d yourTurn=%d choose=%d", joinIdx, yourTurnIdx, chooseIdx)
	}

	// Resolve your-turn by completing the actionable Substep (process becomes done).
	doneAt := now
	if err := store.UpdateProcessProgress(context.Background(), processID, "workflow", "1.1", ProcessStep{
		State:  "done",
		DoneAt: &doneAt,
		Data:   map[string]interface{}{"status": "ok"},
	}); err != nil {
		t.Fatalf("UpdateProcessProgress: %v", err)
	}
	if err := store.UpdateProcessStatus(context.Background(), processID, "workflow", processStatusDone); err != nil {
		t.Fatalf("UpdateProcessStatus: %v", err)
	}

	afterHome := httptest.NewRequest(http.MethodGet, "/my", nil)
	afterHome.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	afterRec := httptest.NewRecorder()
	server.handleHome(afterRec, afterHome)
	if afterRec.Code != http.StatusOK {
		t.Fatalf("after home status=%d", afterRec.Code)
	}
	afterBody := afterRec.Body.String()
	if strings.Contains(afterBody, "Your turn") || strings.Contains(afterBody, "Batch needing action") {
		t.Fatalf("expected your-turn band cleared after resolve, got:\n%s", afterBody)
	}
	// Join Attention remains — dots stay lit from Join alone.
	if !strings.Contains(afterBody, "Join requests") || !strings.Contains(afterBody, `attention-dot`) {
		t.Fatalf("Join Attention should remain after stream resolve, got:\n%s", afterBody)
	}
}

func TestOperatorHomeYourTurnStreamAttentionEmptyOmitted(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-no-your-turn"
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")
	user := AccountUser{
		IdentityUserID: "member-1",
		Email:          "member@org1.example",
		OrgSlug:        "org1",
		RoleSlugs:      []string{"dep1"},
		Status:         "active",
	}
	server := &Server{
		identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		configDir:   tempDir,
		now:         func() time.Time { return now },
	}
	req := httptest.NewRequest(http.MethodGet, "/my", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleHome(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "Your turn") || strings.Contains(body, `aria-label="Attention"`) || strings.Contains(body, `attention-dot`) {
		t.Fatalf("empty your-turn Attention band must be omitted, got:\n%s", body)
	}
}

func TestPlatformAdminHomeOmitsStreamAttention(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")
	store := NewMemoryStore()
	store.SeedProcess(Process{
		WorkflowKey: "workflow",
		Name:        "Should not appear for PA",
		Status:      processStatusActive,
		CreatedAt:   now,
		Progress:    map[string]ProcessStep{"1.1": {State: "pending"}},
	})

	server := &Server{
		identity:    &fakeIdentityStore{},
		store:       store,
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		configDir:   tempDir,
		now:         func() time.Time { return now },
	}
	req := httptest.NewRequest(http.MethodGet, "/my", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()
	server.handleHome(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "Your turn") || strings.Contains(body, "Should not appear for PA") {
		t.Fatalf("platform-admin /my must not show stream Attention, got:\n%s", body)
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

func TestPlatformAdminHomeOrgCreationAttentionBandAndResolve(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	saved, err := store.InsertOrganizationCreationRequest(context.Background(), OrganizationCreationRequest{
		RequesterUserID: "founder-1",
		RequesterEmail:  "founder@example.com",
		ProposedName:    "Fresh Org",
		ProposedSlug:    "fresh-org",
		Status:          AffiliationStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		t.Fatalf("InsertOrganizationCreationRequest: %v", err)
	}

	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")

	server := &Server{
		authorizer: fakeAuthorizer{},
		store:      store,
		identity: &fakeIdentityStore{
			listOrganizationsPageFunc: func(ctx context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error) {
				return IdentityOrgPage{}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				return nil, ErrIdentityNotFound
			},
			getUserByIDFunc: func(ctx context.Context, userID string) (IdentityUser, error) {
				if userID != "founder-1" {
					return IdentityUser{}, ErrIdentityNotFound
				}
				return IdentityUser{ID: "founder-1", Email: "founder@example.com", Status: "active"}, nil
			},
			createOrganizationAsAdminFunc: func(ctx context.Context, name string) (IdentityOrg, error) {
				return IdentityOrg{ID: "team-fresh", Slug: "fresh-org", Name: name}, nil
			},
			addOrganizationUserByIDAsAdminFunc: func(ctx context.Context, orgSlug, userID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
				return IdentityMembership{ID: "membership-" + userID, UserID: userID, IsOrgAdmin: isOrgAdmin}, nil
			},
			updateUserLabelsFunc: func(ctx context.Context, userID string, labels []string) (IdentityUser, error) {
				return IdentityUser{ID: userID, Labels: labels, IsOrgAdmin: true}, nil
			},
		},
		tmpl:        parseTestTemplates(t),
		enforceAuth: true,
		configDir:   tempDir,
		now:         func() time.Time { return now },
	}

	getHome := httptest.NewRequest(http.MethodGet, "/my", nil)
	getHome.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	homeRec := httptest.NewRecorder()
	server.handleHome(homeRec, getHome)
	if homeRec.Code != http.StatusOK {
		t.Fatalf("home status=%d body=%q", homeRec.Code, homeRec.Body.String())
	}
	homeBody := homeRec.Body.String()
	for _, want := range []string{
		`aria-label="Attention"`,
		"Organization requests",
		"Fresh Org",
		"founder@example.com",
		`attention-dot`,
		`Dashboard (needs attention)`,
		`name="intent" value="approve_org_creation"`,
		`name="next" value="/my"`,
		`action="/admin/organizations"`,
	} {
		if !strings.Contains(homeBody, want) {
			t.Fatalf("expected %q on platform-admin home Attention band, got:\n%s", want, homeBody)
		}
	}
	if strings.Contains(homeBody, `Admin (needs attention)`) {
		t.Fatalf("account-menu Admin must not carry section Attention, got:\n%s", homeBody)
	}
	orgIdx := strings.Index(homeBody, "Organization requests")
	quickIdx := strings.Index(homeBody, "Manage streams")
	if orgIdx < 0 || quickIdx < 0 || orgIdx > quickIdx {
		t.Fatalf("expected Organization requests above Manage streams, org=%d quick=%d body:\n%s", orgIdx, quickIdx, homeBody)
	}
	if strings.Contains(homeBody, "Choose a stream") {
		t.Fatalf("platform-admin home must not show Choose a stream catalog, got:\n%s", homeBody)
	}

	adminReq := httptest.NewRequest(http.MethodGet, "/admin/organizations", nil)
	adminReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	adminRec := httptest.NewRecorder()
	server.handleAdminOrgs(adminRec, adminReq)
	if adminRec.Code != http.StatusOK {
		t.Fatalf("admin orgs status=%d", adminRec.Code)
	}
	if !strings.Contains(adminRec.Body.String(), `Organizations (needs attention)`) {
		t.Fatalf("expected Organizations soft-nav Attention chrome, got:\n%s", adminRec.Body.String())
	}

	approve := httptest.NewRequest(http.MethodPost, "/admin/organizations", strings.NewReader(
		"intent=approve_org_creation&request_id="+saved.ID.Hex()+"&next=/my",
	))
	approve.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	approve.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	approveRec := httptest.NewRecorder()
	server.handleAdminOrgs(approveRec, approve)
	if approveRec.Code != http.StatusSeeOther || approveRec.Header().Get("Location") != "/my" {
		t.Fatalf("approve redirect status=%d loc=%q", approveRec.Code, approveRec.Header().Get("Location"))
	}

	afterHome := httptest.NewRequest(http.MethodGet, "/my", nil)
	afterHome.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	afterRec := httptest.NewRecorder()
	server.handleHome(afterRec, afterHome)
	if afterRec.Code != http.StatusOK {
		t.Fatalf("after home status=%d", afterRec.Code)
	}
	afterBody := afterRec.Body.String()
	if strings.Contains(afterBody, "founder@example.com") || strings.Contains(afterBody, `aria-label="Attention"`) {
		t.Fatalf("expected org-creation Attention band cleared after approve, got:\n%s", afterBody)
	}
	if strings.Contains(afterBody, `attention-dot`) {
		t.Fatalf("expected Attention dots cleared after approve, got:\n%s", afterBody)
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
