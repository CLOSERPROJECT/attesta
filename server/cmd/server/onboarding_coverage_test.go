package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleOnboardingHubPendingOrgCreationAndWithdraw(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-onboarding-org-pending"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-org-pending",
		Email:          "founder@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	store := NewMemoryStore()
	server := &Server{
		identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
		store:       store,
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	if _, err := store.InsertOrganizationCreationRequest(context.Background(), OrganizationCreationRequest{
		RequesterUserID: "user-org-pending",
		RequesterEmail:  "founder@example.com",
		ProposedName:    "Fresh Co",
		ProposedSlug:    "fresh-co",
		Status:          AffiliationStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("InsertOrganizationCreationRequest: %v", err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/my/onboarding", nil)
	getReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	getRec := httptest.NewRecorder()
	server.handleMyRoutes(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%q", getRec.Code, getRec.Body.String())
	}
	body := getRec.Body.String()
	for _, want := range []string{
		"Pending organization creation request",
		"Fresh Co",
		`id="undo-request-dialog"`,
		`name="intent" value="withdraw"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in pending org hub, got:\n%s", want, body)
		}
	}

	childReq := httptest.NewRequest(http.MethodGet, "/my/onboarding/request-organization", nil)
	childReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	childRec := httptest.NewRecorder()
	server.handleMyRoutes(childRec, childReq)
	if childRec.Code != http.StatusSeeOther || childRec.Header().Get("Location") != "/my/onboarding" {
		t.Fatalf("child redirect status=%d loc=%q", childRec.Code, childRec.Header().Get("Location"))
	}

	withdrawReq := httptest.NewRequest(http.MethodPost, "/my/onboarding", strings.NewReader("intent=withdraw"))
	withdrawReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withdrawReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	withdrawRec := httptest.NewRecorder()
	server.handleMyRoutes(withdrawRec, withdrawReq)
	if withdrawRec.Code != http.StatusSeeOther {
		t.Fatalf("withdraw status=%d body=%q", withdrawRec.Code, withdrawRec.Body.String())
	}
	pending, err := server.affiliationService().PendingOrganizationCreationRequestForUser(context.Background(), "user-org-pending")
	if err != nil || pending != nil {
		t.Fatalf("expected withdrawn, pending=%+v err=%v", pending, err)
	}
}

func TestHandleOnboardingHubEdgePaths(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-onboarding-edges"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-edges",
		Email:          "edges@example.com",
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

	t.Run("unknown route", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/my/onboarding/nope", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/my/onboarding", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("unsupported intent", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/my/onboarding", strings.NewReader("intent=noop"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("withdraw with nothing pending redirects", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/my/onboarding", strings.NewReader("intent=withdraw"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/my/onboarding" {
			t.Fatalf("status=%d loc=%q", rec.Code, rec.Header().Get("Location"))
		}
	})

	t.Run("pending join without org name falls back to slug", func(t *testing.T) {
		store := NewMemoryStore()
		identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
		identity.getOrganizationBySlugFunc = func(_ context.Context, _ string) (*IdentityOrg, error) {
			return nil, ErrIdentityNotFound
		}
		srv := &Server{
			identity:    identity,
			store:       store,
			tmpl:        parseTestTemplates(t),
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		if _, err := store.InsertJoinRequest(context.Background(), JoinRequest{
			RequesterUserID: "user-edges",
			RequesterEmail:  "edges@example.com",
			OrgSlug:         "ghost-org",
			RoleSlugs:       []string{"viewer"},
			Status:          AffiliationStatusPending,
			CreatedAt:       now,
			UpdatedAt:       now,
		}); err != nil {
			t.Fatalf("InsertJoinRequest: %v", err)
		}
		req := httptest.NewRequest(http.MethodGet, "/my/onboarding", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		srv.handleMyRoutes(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "ghost-org") {
			t.Fatalf("expected slug fallback, got:\n%s", rec.Body.String())
		}
	})
}

func TestHandleOnboardingJoinPOSTValidationErrors(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-onboarding-join-post"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-join-post",
		Email:          "joiner@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	identity.getOrganizationBySlugFunc = func(_ context.Context, slug string) (*IdentityOrg, error) {
		if strings.TrimSpace(slug) != "acme" {
			return nil, ErrIdentityNotFound
		}
		return &IdentityOrg{
			ID:   "team-acme",
			Slug: "acme",
			Name: "Acme Org",
			Roles: []IdentityRole{
				{Slug: "viewer", Name: "Viewer"},
				{Slug: "editor", Name: "Editor"},
			},
		}, nil
	}
	identity.listOrganizationsPageFunc = func(_ context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error) {
		return IdentityOrgPage{
			Organizations: []IdentityOrg{{Slug: "acme", Name: "Acme Org"}},
			Total:         1,
		}, nil
	}
	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	t.Run("invalid roles render form error", func(t *testing.T) {
		form := url.Values{
			"org_slug": {"acme"},
			"roles":    {"not-a-role"},
			"q":        {"acme"},
			"page":     {"1"},
		}
		req := httptest.NewRequest(http.MethodPost, "/my/onboarding/join", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		if !strings.Contains(body, "select one or more organization roles") {
			t.Fatalf("expected invalid roles message, got:\n%s", body)
		}
	})

	t.Run("org not found render form error", func(t *testing.T) {
		form := url.Values{
			"org_slug": {"missing"},
			"roles":    {"viewer"},
		}
		req := httptest.NewRequest(http.MethodPost, "/my/onboarding/join", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "organization not found") {
			t.Fatalf("expected not found message, got:\n%s", rec.Body.String())
		}
	})

	t.Run("join method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/my/onboarding/join", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status=%d", rec.Code)
		}
	})
}

func TestHandleOnboardingJoinPaginationClamp(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-onboarding-join-page"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-join-page",
		Email:          "pager@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	var offsets []int
	identity.listOrganizationsPageFunc = func(_ context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error) {
		offsets = append(offsets, opts.Offset)
		orgs := make([]IdentityOrg, 0)
		if opts.Offset < onboardingJoinSearchLimit {
			orgs = append(orgs, IdentityOrg{Slug: "acme", Name: "Acme"})
		}
		return IdentityOrgPage{Organizations: orgs, Total: onboardingJoinSearchLimit + 3}, nil
	}
	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/my/onboarding/join?q=acme&page=99", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if len(offsets) < 2 {
		t.Fatalf("expected clamp refetch, offsets=%v", offsets)
	}
	if offsets[0] != 98*onboardingJoinSearchLimit {
		t.Fatalf("first offset=%d", offsets[0])
	}
	if offsets[1] != onboardingJoinSearchLimit {
		t.Fatalf("clamped offset=%d, want %d", offsets[1], onboardingJoinSearchLimit)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "page=2") && !strings.Contains(body, ">2<") {
		// page numbers rendered; ensure we landed on last page content path
		if !strings.Contains(body, "Acme") && !strings.Contains(body, "Join an organization") {
			t.Fatalf("unexpected join page body:\n%s", body)
		}
	}
}

func TestHandleOnboardingJoinSelectedOrgNotFound(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-onboarding-join-selected"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-join-selected",
		Email:          "selected@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	identity.getOrganizationBySlugFunc = func(_ context.Context, _ string) (*IdentityOrg, error) {
		return nil, ErrIdentityNotFound
	}
	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/my/onboarding/join?org=missing", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "join-org-dialog-body")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "organization not found") {
		t.Fatalf("expected org not found form error, got:\n%s", rec.Body.String())
	}
}

func TestHandleOnboardingRequestOrganizationPOSTValidation(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-onboarding-req-org"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-req-org",
		Email:          "req@example.com",
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

	t.Run("empty name form error", func(t *testing.T) {
		form := url.Values{"name": {"   "}}
		req := httptest.NewRequest(http.MethodPost, "/my/onboarding/request-organization", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "organization name is required") {
			t.Fatalf("expected invalid name message, got:\n%s", rec.Body.String())
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/my/onboarding/request-organization", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status=%d", rec.Code)
		}
	})
}

func TestNormalizeOnboardingJoinPage(t *testing.T) {
	if got := normalizeOnboardingJoinPage(0, 25); got != 1 {
		t.Fatalf("raw<1 got=%d", got)
	}
	if got := normalizeOnboardingJoinPage(-3, 25); got != 1 {
		t.Fatalf("negative got=%d", got)
	}
	if got := normalizeOnboardingJoinPage(99, 25); got != 3 {
		t.Fatalf("clamp high got=%d want 3", got)
	}
	if got := normalizeOnboardingJoinPage(2, 0); got != 1 {
		t.Fatalf("empty total got=%d", got)
	}
	if got := normalizeOnboardingJoinPage(2, 25); got != 2 {
		t.Fatalf("in range got=%d", got)
	}
}

func TestIdentityUserForAffiliationNil(t *testing.T) {
	if got := identityUserForAffiliation(nil); got.ID != "" || got.Email != "" {
		t.Fatalf("nil user = %+v", got)
	}
}

func TestRedirectOnboardingIfPendingStoreError(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-onboarding-pending-err"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-pending-err",
		Email:          "err@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	hook := &affiliationStoreHook{
		MemoryStore: NewMemoryStore(),
		findPendingJoinErr: errors.New("store down"),
	}
	server := &Server{
		identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
		affiliation: NewAffiliation(testIdentityForSessions(now, map[string]AccountUser{sessionID: user}), hook, &recordingMailer{}, func() time.Time { return now }, nil),
	}

	req := httptest.NewRequest(http.MethodGet, "/my/onboarding/join", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}

type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) { return 0, errors.New("boom body") }
func (errReadCloser) Close() error             { return nil }

func TestHandleOnboardingParseFormAndWithdrawErrors(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-onboarding-form-err"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-form-err",
		Email:          "form@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})

	t.Run("hub parse form error", func(t *testing.T) {
		server := &Server{
			identity:    identity,
			store:       NewMemoryStore(),
			tmpl:        parseTestTemplates(t),
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodPost, "/my/onboarding", errReadCloser{})
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("join parse form error", func(t *testing.T) {
		server := &Server{
			identity:    identity,
			store:       NewMemoryStore(),
			tmpl:        parseTestTemplates(t),
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodPost, "/my/onboarding/join", errReadCloser{})
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("request org parse form error", func(t *testing.T) {
		server := &Server{
			identity:    identity,
			store:       NewMemoryStore(),
			tmpl:        parseTestTemplates(t),
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodPost, "/my/onboarding/request-organization", errReadCloser{})
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("withdraw join load error", func(t *testing.T) {
		hook := &affiliationStoreHook{MemoryStore: NewMemoryStore(), findPendingJoinErr: errors.New("join load failed")}
		server := &Server{
			identity:    identity,
			store:       NewMemoryStore(),
			tmpl:        parseTestTemplates(t),
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         func() time.Time { return now },
			affiliation: NewAffiliation(identity, hook, &recordingMailer{}, func() time.Time { return now }, nil),
		}
		req := httptest.NewRequest(http.MethodPost, "/my/onboarding", strings.NewReader("intent=withdraw"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("withdraw join fails renders form error", func(t *testing.T) {
		reqID := primitive.NewObjectID()
		hook := &affiliationStoreHook{
			MemoryStore:             NewMemoryStore(),
			findPendingJoinOverride: &JoinRequest{ID: reqID, RequesterUserID: "user-form-err", Status: AffiliationStatusPending},
			deleteJoinErr:           errors.New("cannot delete"),
		}
		server := &Server{
			identity:    identity,
			store:       NewMemoryStore(),
			tmpl:        parseTestTemplates(t),
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         func() time.Time { return now },
			affiliation: NewAffiliation(identity, hook, &recordingMailer{}, func() time.Time { return now }, nil),
		}
		req := httptest.NewRequest(http.MethodPost, "/my/onboarding", strings.NewReader("intent=withdraw"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "failed to undo request") {
			t.Fatalf("expected withdraw form error, got:\n%s", rec.Body.String())
		}
	})

	t.Run("withdraw org load error", func(t *testing.T) {
		hook := &affiliationStoreHook{MemoryStore: NewMemoryStore(), findPendingOrgErr: errors.New("org load failed")}
		server := &Server{
			identity:    identity,
			store:       NewMemoryStore(),
			tmpl:        parseTestTemplates(t),
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         func() time.Time { return now },
			affiliation: NewAffiliation(identity, hook, &recordingMailer{}, func() time.Time { return now }, nil),
		}
		req := httptest.NewRequest(http.MethodPost, "/my/onboarding", strings.NewReader("intent=withdraw"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("withdraw org fails renders form error", func(t *testing.T) {
		reqID := primitive.NewObjectID()
		hook := &affiliationStoreHook{
			MemoryStore:            NewMemoryStore(),
			findPendingOrgOverride: &OrganizationCreationRequest{ID: reqID, RequesterUserID: "user-form-err", Status: AffiliationStatusPending, ProposedName: "X", ProposedSlug: "x"},
			deleteOrgErr:           errors.New("cannot delete org"),
		}
		server := &Server{
			identity:    identity,
			store:       NewMemoryStore(),
			tmpl:        parseTestTemplates(t),
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         func() time.Time { return now },
			affiliation: NewAffiliation(identity, hook, &recordingMailer{}, func() time.Time { return now }, nil),
		}
		req := httptest.NewRequest(http.MethodPost, "/my/onboarding", strings.NewReader("intent=withdraw"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "failed to undo request") {
			t.Fatalf("expected withdraw form error, got:\n%s", rec.Body.String())
		}
	})

	t.Run("hub render join pending load error", func(t *testing.T) {
		hook := &affiliationStoreHook{MemoryStore: NewMemoryStore(), findPendingJoinErr: errors.New("hub join failed")}
		server := &Server{
			identity:    identity,
			store:       NewMemoryStore(),
			tmpl:        parseTestTemplates(t),
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         func() time.Time { return now },
			affiliation: NewAffiliation(identity, hook, &recordingMailer{}, func() time.Time { return now }, nil),
		}
		req := httptest.NewRequest(http.MethodGet, "/my/onboarding", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("hub render org pending load error", func(t *testing.T) {
		hook := &affiliationStoreHook{MemoryStore: NewMemoryStore(), findPendingOrgErr: errors.New("hub org failed")}
		server := &Server{
			identity:    identity,
			store:       NewMemoryStore(),
			tmpl:        parseTestTemplates(t),
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         func() time.Time { return now },
			affiliation: NewAffiliation(identity, hook, &recordingMailer{}, func() time.Time { return now }, nil),
		}
		req := httptest.NewRequest(http.MethodGet, "/my/onboarding", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rec.Code)
		}
	})
}

func TestHandleOnboardingJoinSearchAndOrgLoadErrors(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-onboarding-search-err"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-search-err",
		Email:          "search@example.com",
		Status:         "active",
		CreatedAt:      now,
	}

	t.Run("search error", func(t *testing.T) {
		identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
		identity.listOrganizationsPageFunc = func(_ context.Context, _ IdentityOrgListOptions) (IdentityOrgPage, error) {
			return IdentityOrgPage{}, errors.New("search failed")
		}
		server := &Server{
			identity:    identity,
			store:       NewMemoryStore(),
			tmpl:        parseTestTemplates(t),
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodGet, "/my/onboarding/join?q=acme", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("selected org load error", func(t *testing.T) {
		identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
		identity.getOrganizationBySlugFunc = func(_ context.Context, _ string) (*IdentityOrg, error) {
			return nil, errors.New("org lookup failed")
		}
		server := &Server{
			identity:    identity,
			store:       NewMemoryStore(),
			tmpl:        parseTestTemplates(t),
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodGet, "/my/onboarding/join?org=acme", nil)
		req.Header.Set("HX-Request", "true")
		req.Header.Set("HX-Target", "join-org-dialog-body")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("page clamp refetch error", func(t *testing.T) {
		identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
		calls := 0
		identity.listOrganizationsPageFunc = func(_ context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error) {
			calls++
			if calls == 1 {
				return IdentityOrgPage{Total: onboardingJoinSearchLimit + 1}, nil
			}
			return IdentityOrgPage{}, errors.New("refetch failed")
		}
		server := &Server{
			identity:    identity,
			store:       NewMemoryStore(),
			tmpl:        parseTestTemplates(t),
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodGet, "/my/onboarding/join?q=acme&page=99", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("unauthenticated onboarding", func(t *testing.T) {
		server := &Server{
			identity:    testIdentityForSessions(now, map[string]AccountUser{}),
			store:       NewMemoryStore(),
			tmpl:        parseTestTemplates(t),
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodGet, "/my/onboarding", nil)
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusSeeOther && rec.Code != http.StatusUnauthorized && rec.Code != http.StatusFound {
			t.Fatalf("status=%d", rec.Code)
		}
	})
}

func TestMapAffiliationFormErrorBranches(t *testing.T) {
	msgs := affiliationFormErrorMessages{
		AlreadyAffiliated:      "a",
		PendingExists:          "p",
		NotFound:               "n",
		NotPending:             "np",
		InvalidRoles:           "ir",
		InvalidName:            "in",
		OrganizationSlugExists: "ose",
		Default:                "d",
	}
	if got := mapAffiliationFormError(nil, msgs); got != "" {
		t.Fatalf("nil=%q", got)
	}
	cases := []struct {
		err  error
		want string
	}{
		{ErrAffiliationAlreadyAffiliated, "a"},
		{ErrAffiliationPendingExists, "p"},
		{ErrAffiliationNotFound, "n"},
		{ErrAffiliationNotPending, "np"},
		{ErrAffiliationInvalidRoles, "ir"},
		{ErrAffiliationInvalidName, "in"},
		{ErrAffiliationOrganizationSlugExists, "ose"},
		{errors.New("other"), "d"},
	}
	for _, tc := range cases {
		if got := mapAffiliationFormError(tc.err, msgs); got != tc.want {
			t.Fatalf("err=%v got=%q want=%q", tc.err, got, tc.want)
		}
	}
	if got := affiliationWithdrawFormError(ErrAffiliationNotPending); got != "request is not pending" {
		t.Fatalf("withdraw not pending=%q", got)
	}
}

func TestRoleMetaIndexEdges(t *testing.T) {
	if got := (*Server)(nil).roleMetaIndex(context.Background()); len(got) != 0 {
		t.Fatalf("nil server=%v", got)
	}
	server := &Server{identity: &fakeIdentityStore{
		listOrganizationsFunc: func(_ context.Context) ([]IdentityOrg, error) {
			return nil, errors.New("list failed")
		},
	}}
	if got := server.roleMetaIndex(context.Background()); len(got) != 0 {
		t.Fatalf("list err=%v", got)
	}
	server.identity = &fakeIdentityStore{
		listOrganizationsFunc: func(_ context.Context) ([]IdentityOrg, error) {
			return []IdentityOrg{
				{Slug: "", Roles: []IdentityRole{{Slug: "x", Name: "X"}}},
				{Slug: "acme", Roles: []IdentityRole{
					{Slug: "", Name: "empty"},
					{Slug: "viewer", Name: ""},
					{Slug: "editor", Name: "Editor"},
				}},
			}, nil
		},
	}
	idx := server.roleMetaIndex(context.Background())
	if _, ok := idx[roleMetaKey{OrgSlug: "acme", RoleSlug: "viewer"}]; !ok {
		t.Fatalf("idx=%v", idx)
	}
	if idx[roleMetaKey{OrgSlug: "acme", RoleSlug: "viewer"}].Label != "viewer" {
		t.Fatalf("empty name should fall back to slug: %#v", idx[roleMetaKey{OrgSlug: "acme", RoleSlug: "viewer"}])
	}
	if meta := roleMetaForOrg("", "", idx, nil); meta.Palette != "fallback" {
		t.Fatalf("empty role=%#v", meta)
	}
}
