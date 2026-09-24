package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleOnboardingRequestOrganizationSubmitAndPending(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-org-creation-submit"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-1",
		Email:          "newbie@example.com",
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

	postReq := httptest.NewRequest(http.MethodPost, "/my/onboarding/request-organization", strings.NewReader("name=Fresh+Org"))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	postRec := httptest.NewRecorder()
	server.handleMyRoutes(postRec, postReq)

	if postRec.Code != http.StatusSeeOther {
		t.Fatalf("submit status = %d, want %d body=%q", postRec.Code, http.StatusSeeOther, postRec.Body.String())
	}
	if loc := postRec.Header().Get("Location"); loc != "/my/onboarding/request-organization" {
		t.Fatalf("submit location = %q", loc)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/my/onboarding/request-organization", nil)
	getReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	getRec := httptest.NewRecorder()
	server.handleMyRoutes(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("pending status = %d, want %d body=%q", getRec.Code, http.StatusOK, getRec.Body.String())
	}
	body := getRec.Body.String()
	for _, want := range []string{
		"Pending organization creation request",
		"Fresh Org",
		"fresh-org",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in pending page, got:\n%s", want, body)
		}
	}
	if strings.Contains(body, `name="name"`) {
		t.Fatalf("expected no submit form while pending, got:\n%s", body)
	}
}

func TestHandleOnboardingRequestOrganizationAffiliatedBlocked(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-org-creation-affiliated"
	user := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-2",
		Email:          "member@example.com",
		OrgSlug:        "acme",
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

	req := httptest.NewRequest(http.MethodPost, "/my/onboarding/request-organization", strings.NewReader("name=Fresh+Org"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if rec.Header().Get("Location") != "/my" {
		t.Fatalf("location = %q, want /my", rec.Header().Get("Location"))
	}
	pending, err := server.affiliationService().PendingOrganizationCreationRequestForUser(context.Background(), "user-2")
	if err != nil {
		t.Fatalf("PendingOrganizationCreationRequestForUser: %v", err)
	}
	if pending != nil {
		t.Fatalf("expected no pending request for affiliated user, got %+v", pending)
	}
}

func TestHandleAdminOrgsPendingOrganizationCreationRequests(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	saved, err := store.InsertOrganizationCreationRequest(context.Background(), OrganizationCreationRequest{
		RequesterUserID: "user-1",
		RequesterEmail:  "newbie@example.com",
		ProposedName:    "Fresh Org",
		ProposedSlug:    "fresh-org",
		Status:          AffiliationStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		t.Fatalf("InsertOrganizationCreationRequest: %v", err)
	}

	server := &Server{
		authorizer: fakeAuthorizer{},
		store:      store,
		identity: &fakeIdentityStore{
			listOrganizationsPageFunc: func(ctx context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error) {
				return IdentityOrgPage{}, nil
			},
		},
		tmpl:        parseTestTemplates(t),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/organizations", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()
	server.handleAdminOrgs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Pending organization requests",
		"Fresh Org",
		"fresh-org",
		"newbie@example.com",
		saved.ID.Hex(),
		`name="intent" value="approve_org_creation"`,
		`name="intent" value="reject_org_creation"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in admin orgs page, got:\n%s", want, body)
		}
	}
}

func TestHandleAdminOrgsApproveAndRejectOrganizationCreation(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()

	approveReq, err := store.InsertOrganizationCreationRequest(context.Background(), OrganizationCreationRequest{
		RequesterUserID: "user-approve",
		RequesterEmail:  "approve@example.com",
		ProposedName:    "Approve Org",
		ProposedSlug:    "approve-org",
		Status:          AffiliationStatusPending,
	})
	if err != nil {
		t.Fatalf("insert approve request: %v", err)
	}
	rejectReq, err := store.InsertOrganizationCreationRequest(context.Background(), OrganizationCreationRequest{
		RequesterUserID: "user-reject",
		RequesterEmail:  "reject@example.com",
		ProposedName:    "Reject Org",
		ProposedSlug:    "reject-org",
		Status:          AffiliationStatusPending,
	})
	if err != nil {
		t.Fatalf("insert reject request: %v", err)
	}

	var createdName string
	var stampedUserID string
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
				switch userID {
				case "user-approve":
					return IdentityUser{ID: "user-approve", Email: "approve@example.com", Status: "active"}, nil
				case "user-reject":
					return IdentityUser{ID: "user-reject", Email: "reject@example.com", Status: "active"}, nil
				default:
					return IdentityUser{ID: userID, Status: "active"}, nil
				}
			},
			createOrganizationAsAdminFunc: func(ctx context.Context, name string) (IdentityOrg, error) {
				createdName = name
				return IdentityOrg{ID: "team-approve", Slug: "approve-org", Name: name}, nil
			},
			addOrganizationUserByIDAsAdminFunc: func(ctx context.Context, orgSlug, userID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
				return IdentityMembership{ID: "membership-" + userID, UserID: userID, IsOrgAdmin: isOrgAdmin}, nil
			},
			updateUserLabelsFunc: func(ctx context.Context, userID string, labels []string) (IdentityUser, error) {
				stampedUserID = userID
				return IdentityUser{ID: userID, Labels: labels, IsOrgAdmin: true}, nil
			},
		},
		tmpl:        parseTestTemplates(t),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	approveHTTP := httptest.NewRequest(http.MethodPost, "/admin/organizations", strings.NewReader("intent=approve_org_creation&request_id="+approveReq.ID.Hex()))
	approveHTTP.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	approveHTTP.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	approveRec := httptest.NewRecorder()
	server.handleAdminOrgs(approveRec, approveHTTP)

	if approveRec.Code != http.StatusSeeOther {
		t.Fatalf("approve status = %d, want %d body=%q", approveRec.Code, http.StatusSeeOther, approveRec.Body.String())
	}
	if loc := approveRec.Header().Get("Location"); !strings.Contains(loc, "confirmation=organization+creation+request+approved") {
		t.Fatalf("approve location = %q", loc)
	}
	if createdName != "Approve Org" || stampedUserID != "user-approve" {
		t.Fatalf("approve wiring createdName=%q stampedUserID=%q", createdName, stampedUserID)
	}
	loadedApprove, err := store.LoadOrganizationCreationRequestByID(context.Background(), approveReq.ID)
	if err != nil || loadedApprove == nil || loadedApprove.Status != AffiliationStatusApproved {
		t.Fatalf("approve persisted=%+v err=%v", loadedApprove, err)
	}

	rejectHTTP := httptest.NewRequest(http.MethodPost, "/admin/organizations", strings.NewReader("intent=reject_org_creation&request_id="+rejectReq.ID.Hex()+"&reason=duplicate+brand"))
	rejectHTTP.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rejectHTTP.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rejectRec := httptest.NewRecorder()
	server.handleAdminOrgs(rejectRec, rejectHTTP)

	if rejectRec.Code != http.StatusSeeOther {
		t.Fatalf("reject status = %d, want %d body=%q", rejectRec.Code, http.StatusSeeOther, rejectRec.Body.String())
	}
	if loc := rejectRec.Header().Get("Location"); !strings.Contains(loc, "confirmation=organization+creation+request+rejected") {
		t.Fatalf("reject location = %q", loc)
	}
	loadedReject, err := store.LoadOrganizationCreationRequestByID(context.Background(), rejectReq.ID)
	if err != nil || loadedReject == nil || loadedReject.Status != AffiliationStatusRejected || loadedReject.RejectReason != "duplicate brand" {
		t.Fatalf("reject persisted=%+v err=%v", loadedReject, err)
	}
}

func TestHandleOrgAdminUsersSelfServeCreateOrgGone(t *testing.T) {
	now := time.Now().UTC()
	createCalls := 0
	server := &Server{
		authorizer: fakeAuthorizer{},
		store:      NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{
					ID:         "user-1",
					Email:      "owner@example.com",
					Labels:     []string{identityOrgAdminLabel},
					IsOrgAdmin: true,
					Status:     "active",
				}, nil
			},
			createOrganizationFunc: func(ctx context.Context, sessionSecret, name string) (IdentityOrg, error) {
				createCalls++
				return IdentityOrg{ID: "team-1", Slug: "fresh-org", Name: name}, nil
			},
		},
		tmpl:        parseTestTemplates(t),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodPost, "/my/organization/users", strings.NewReader("intent=create_org&name=Fresh+Org"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()
	server.handleOrgAdminUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if createCalls != 0 {
		t.Fatalf("create calls = %d, want 0", createCalls)
	}
	if !strings.Contains(rec.Body.String(), "request organization creation via onboarding") {
		t.Fatalf("expected onboarding redirect message, got %q", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `name="intent" value="create_org"`) {
		t.Fatalf("expected self-serve create form gone, got %q", rec.Body.String())
	}
}
