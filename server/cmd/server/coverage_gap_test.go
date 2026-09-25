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

func TestAffiliationJoinDecideFormError(t *testing.T) {
	if got := affiliationJoinDecideFormError(ErrAffiliationAlreadyAffiliated); got != "requester already belongs to an organization" {
		t.Fatalf("got=%q", got)
	}
	if got := affiliationJoinDecideFormError(ErrAffiliationNotFound); got != "join request not found" {
		t.Fatalf("got=%q", got)
	}
	if got := affiliationJoinDecideFormError(ErrAffiliationNotPending); got != "join request is not pending" {
		t.Fatalf("got=%q", got)
	}
	if got := affiliationJoinDecideFormError(ErrAffiliationInvalidRoles); got != "requested roles are no longer valid" {
		t.Fatalf("got=%q", got)
	}
	if got := affiliationJoinDecideFormError(errors.New("x")); got != "failed to process join request" {
		t.Fatalf("got=%q", got)
	}
}

func TestHandleOnboardingJoinPageZeroClamps(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-page-zero"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-page-zero",
		Email:          "zero@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	var seenOffset int
	identity.listOrganizationsPageFunc = func(_ context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error) {
		seenOffset = opts.Offset
		return IdentityOrgPage{Organizations: []IdentityOrg{{Slug: "acme", Name: "Acme"}}, Total: 1}, nil
	}
	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	req := httptest.NewRequest(http.MethodGet, "/my/onboarding/join?q=acme&page=0", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if seenOffset != 0 {
		t.Fatalf("offset=%d, want 0 after clamp", seenOffset)
	}
}

func TestHandleLeaveOrganizationUnauthAndEmptySession(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	server := &Server{
		identity:    testIdentityForSessions(now, map[string]AccountUser{}),
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	req := httptest.NewRequest(http.MethodPost, leaveOrganizationPath(), nil)
	rec := httptest.NewRecorder()
	server.handleLeaveOrganization(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauth status=%d", rec.Code)
	}

	// enforceAuth off + no cookie → sessionSecretFromRequest fails
	open := &Server{
		identity:    &fakeIdentityStore{},
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: false,
		now:         func() time.Time { return now },
	}
	openReq := httptest.NewRequest(http.MethodPost, leaveOrganizationPath(), nil)
	openRec := httptest.NewRecorder()
	open.handleLeaveOrganization(openRec, openReq)
	if openRec.Code != http.StatusUnauthorized {
		t.Fatalf("empty session status=%d body=%q", openRec.Code, openRec.Body.String())
	}
}

func TestMemoryStoreListPendingUnequalTimes(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	older := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	idOld := primitive.NewObjectID()
	idNew := primitive.NewObjectID()
	store.joinRequests[idOld] = JoinRequest{ID: idOld, OrgSlug: "acme", Status: AffiliationStatusPending, CreatedAt: older}
	store.joinRequests[idNew] = JoinRequest{ID: idNew, OrgSlug: "acme", Status: AffiliationStatusPending, CreatedAt: newer}
	items, err := store.ListPendingJoinRequestsByOrg(ctx, "acme")
	if err != nil || len(items) != 2 || items[0].ID != idNew {
		t.Fatalf("join items=%v err=%v", items, err)
	}

	orgOld := primitive.NewObjectID()
	orgNew := primitive.NewObjectID()
	store.organizationCreationRequests[orgOld] = OrganizationCreationRequest{ID: orgOld, Status: AffiliationStatusPending, CreatedAt: older}
	store.organizationCreationRequests[orgNew] = OrganizationCreationRequest{ID: orgNew, Status: AffiliationStatusPending, CreatedAt: newer}
	orgItems, err := store.ListPendingOrganizationCreationRequests(ctx)
	if err != nil || len(orgItems) != 2 || orgItems[0].ID != orgNew {
		t.Fatalf("org items=%v err=%v", orgItems, err)
	}
}

func TestApproveOrganizationCreationSlugUnavailable(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	aff := NewAffiliation(&fakeIdentityStore{}, store, &recordingMailer{}, fixedNow, nil)
	saved, err := aff.SubmitOrganizationCreationRequest(ctx, IdentityUser{ID: "founder", Email: "f@example.com"}, "Taken Org")
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	identity := &fakeIdentityStore{
		getUserByIDFunc: func(_ context.Context, userID string) (IdentityUser, error) {
			return IdentityUser{ID: userID, Email: "f@example.com"}, nil
		},
		getOrganizationBySlugFunc: func(_ context.Context, slug string) (*IdentityOrg, error) {
			return &IdentityOrg{Slug: slug, Name: "Exists"}, nil
		},
	}
	aff2 := NewAffiliation(identity, store, &recordingMailer{}, fixedNow, nil)
	_, _, err = aff2.ApproveOrganizationCreationRequest(ctx, saved.ID, IdentityUser{ID: "admin"})
	if !errors.Is(err, ErrAffiliationOrganizationSlugExists) {
		t.Fatalf("err=%v", err)
	}
}

func TestSMTPMailerSendCanceledAndAuth(t *testing.T) {
	mailer := &smtpMailer{host: "127.0.0.1", port: "9", from: "a@b.c", username: "u", password: "p"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := mailer.Send(ctx, MailMessage{To: []string{"x@y.z"}, Subject: "s", Body: "b"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled err=%v", err)
	}
}

func TestHandleOrganizationHomeUnauthenticated(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	server := &Server{
		identity:    testIdentityForSessions(now, map[string]AccountUser{}),
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	req := httptest.NewRequest(http.MethodGet, organizationPath(""), nil)
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)
	if rec.Code != http.StatusSeeOther && rec.Code != http.StatusUnauthorized && rec.Code != http.StatusFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestMergeStringMapCoverage(t *testing.T) {
	got := mergeStringMap(map[string]interface{}{"a": 1}, map[string]interface{}{"b": "2", "a": "overwrite"})
	if got["b"] != "2" || got["a"] != "overwrite" {
		t.Fatalf("got=%v", got)
	}
	if empty := mergeStringMap(nil, nil); len(empty) != 0 {
		t.Fatalf("empty=%v", empty)
	}
	if onlyExtra := mergeStringMap(nil, map[string]interface{}{"x": 1}); onlyExtra["x"] != 1 {
		t.Fatalf("onlyExtra=%v", onlyExtra)
	}
}
func TestSMTPMailerSecureModesDialErrors(t *testing.T) {
	msg := MailMessage{To: []string{"user@example.com"}, Subject: "s", Body: "b"}
	for _, secure := range []smtpSecureMode{smtpSecureStartTLS, smtpSecureImplicitTLS, smtpSecurePlain} {
		mailer := &smtpMailer{
			host:     "127.0.0.1",
			port:     "1",
			from:     "attesta@localhost",
			username: "user",
			password: "pass",
			secure:   secure,
		}
		err := mailer.Send(context.Background(), msg)
		if err == nil {
			t.Fatalf("secure=%v expected dial error", secure)
		}
	}
}

func TestLeaveOrganizationEmptySessionUsesCookie(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	server := &Server{
		identity:    &fakeIdentityStore{},
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: false,
		now:         func() time.Time { return now },
	}
	req := httptest.NewRequest(http.MethodPost, leaveOrganizationPath(), nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "cookie-secret"})
	rec := httptest.NewRecorder()
	server.handleLeaveOrganization(rec, req)
	// empty AccountUser is unaffiliated → redirect onboarding
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != onboardingPath() {
		t.Fatalf("status=%d loc=%q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestStreamManagementFlagsNilAuthorizer(t *testing.T) {
	server := &Server{}
	clone, edit, purge, del, reason := server.streamManagementFlags(context.Background(), &AccountUser{Email: "a@b.c"}, "k", FormataBuilderStream{}, false, true)
	if clone || edit || purge || del || reason != "" {
		t.Fatalf("expected all false/empty, got %v %v %v %v %q", clone, edit, purge, del, reason)
	}
}

func TestFormataBuilderAuthErrorAndForbidden(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-formata-cov"
	account := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "member-1",
		Email:          "member@example.com",
		OrgSlug:        "acme",
		RoleSlugs:      []string{"viewer"},
		Status:         "active",
		CreatedAt:      now,
	}
	sessions := map[string]AccountUser{sessionID: account}
	identity := testIdentityForSessions(now, sessions)

	t.Run("view cerbos error", func(t *testing.T) {
		server := &Server{
			identity: identity,
			store:    NewMemoryStore(),
			tmpl:     parseTestTemplates(t),
			authorizer: fakeAuthorizer{
				accessDecide: func(_ *AccountUser, _, _ string, _ map[string]interface{}, _ string) (bool, error) {
					return false, errors.New("cerbos down")
				},
			},
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodGet, organizationPath("formata-builder"), nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusBadGateway {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("view forbidden", func(t *testing.T) {
		server := &Server{
			identity: identity,
			store:    NewMemoryStore(),
			tmpl:     parseTestTemplates(t),
			authorizer: fakeAuthorizer{
				accessDecide: func(_ *AccountUser, _, _ string, _ map[string]interface{}, _ string) (bool, error) {
					return false, nil
				},
			},
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodGet, organizationPath("formata-builder"), nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("save cerbos error", func(t *testing.T) {
		server := &Server{
			identity: identity,
			store:    NewMemoryStore(),
			tmpl:     parseTestTemplates(t),
			authorizer: fakeAuthorizer{
				accessDecide: func(_ *AccountUser, _, _ string, _ map[string]interface{}, _ string) (bool, error) {
					return false, errors.New("cerbos save down")
				},
			},
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodPost, organizationPath("formata-builder"), strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusBadGateway {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("save forbidden", func(t *testing.T) {
		server := &Server{
			identity: identity,
			store:    NewMemoryStore(),
			tmpl:     parseTestTemplates(t),
			authorizer: fakeAuthorizer{
				accessDecide: func(_ *AccountUser, _, _ string, _ map[string]interface{}, _ string) (bool, error) {
					return false, nil
				},
			},
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodPost, organizationPath("formata-builder"), strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status=%d", rec.Code)
		}
	})
}

func TestAdminCategoriesParseFormErrors(t *testing.T) {
	admin := &AccountUser{Email: "admin@example.com", IsPlatformAdmin: true}
	server := &Server{store: NewMemoryStore(), tmpl: parseTestTemplates(t)}

	req := httptest.NewRequest(http.MethodPost, "/admin/categories", errReadCloser{})
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.handleAdminCategoriesPost(rec, req, admin)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("categories status=%d", rec.Code)
	}

	subReq := httptest.NewRequest(http.MethodPost, "/admin/categories/acme/subcategories", errReadCloser{})
	subReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	subRec := httptest.NewRecorder()
	server.handleAdminSubcategoriesPost(subRec, subReq, admin, "acme")
	if subRec.Code != http.StatusBadRequest {
		t.Fatalf("subcategories status=%d", subRec.Code)
	}
}

func TestHandleOrganizationHomeDirectUnauth(t *testing.T) {
	server := &Server{
		identity:    &fakeIdentityStore{},
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         time.Now,
	}
	req := httptest.NewRequest(http.MethodGet, organizationPath(""), nil)
	rec := httptest.NewRecorder()
	server.handleOrganizationHome(rec, req)
	if rec.Code != http.StatusSeeOther && rec.Code != http.StatusUnauthorized && rec.Code != http.StatusFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestFormataBuilderInvalidBody(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-formata-body"
	account := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "admin-1",
		Email:          "admin@example.com",
		OrgSlug:        "acme",
		RoleSlugs:      []string{"org-admin"},
		Status:         "active",
		CreatedAt:      now,
	}
	sessions := map[string]AccountUser{sessionID: account}
	identity := testIdentityForSessions(now, sessions)
	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	req := httptest.NewRequest(http.MethodPost, organizationPath("formata-builder"), errReadCloser{})
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}
