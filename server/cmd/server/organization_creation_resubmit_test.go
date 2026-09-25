package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Repro: approve org creation → platform admin deletes the org → founder requests again.
// Platform admin must see the new pending request.
func TestResubmitOrganizationCreationAfterApproveAndDeleteVisibleToAdmin(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-founder-resubmit"
	founder := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-founder",
		Email:          "founder@example.com",
		Status:         "active",
		CreatedAt:      now,
	}

	store := NewMemoryStore()
	identity := newLifecycleIdentityStore(founder)

	server := &Server{
		authorizer:  fakeAuthorizer{},
		store:       store,
		identity:    identity,
		tmpl:        parseTestTemplates(t),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	// 1) Founder requests org.
	postReq := httptest.NewRequest(http.MethodPost, "/my/onboarding/request-organization", strings.NewReader("name=New+Org"))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	postRec := httptest.NewRecorder()
	server.handleMyRoutes(postRec, postReq)
	if postRec.Code != http.StatusSeeOther {
		t.Fatalf("submit status=%d body=%q", postRec.Code, postRec.Body.String())
	}

	pending, err := store.ListPendingOrganizationCreationRequests(context.Background())
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending after submit=%v err=%v", pending, err)
	}
	firstReqID := pending[0].ID

	// 2) Platform admin approves.
	approveHTTP := httptest.NewRequest(http.MethodPost, "/admin/organizations", strings.NewReader("intent=approve_org_creation&request_id="+firstReqID.Hex()))
	approveHTTP.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	approveHTTP.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	approveRec := httptest.NewRecorder()
	server.handleAdminOrgs(approveRec, approveHTTP)
	if approveRec.Code != http.StatusSeeOther {
		t.Fatalf("approve status=%d body=%q", approveRec.Code, approveRec.Body.String())
	}
	if !identity.hasOrg("new-org") {
		t.Fatal("expected org created on approve")
	}
	if !identity.userAffiliated("user-founder") {
		t.Fatal("expected founder affiliated after approve")
	}

	// 3) Platform admin deletes the org (today: DeleteOrganizationAsAdmin only).
	deleteHTTP := httptest.NewRequest(http.MethodPost, "/admin/organizations", strings.NewReader("intent=delete_org&org_slug=new-org"))
	deleteHTTP.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	deleteHTTP.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	deleteRec := httptest.NewRecorder()
	server.handleAdminOrgs(deleteRec, deleteHTTP)
	if deleteRec.Code != http.StatusOK && deleteRec.Code != http.StatusSeeOther {
		t.Fatalf("delete status=%d body=%q", deleteRec.Code, deleteRec.Body.String())
	}
	if identity.hasOrg("new-org") {
		t.Fatal("expected org deleted")
	}

	// After team delete, founder must be able to use onboarding again.
	if identity.userAffiliated("user-founder") {
		t.Fatalf("founder still affiliated after org delete (OrgSlug=%q Labels=%v) — delete_org left membership/labels unclean",
			identity.snapshot("user-founder").OrgSlug, identity.snapshot("user-founder").Labels)
	}

	// 4) Founder requests again — same name (slug reuse after delete) is the common case.
	post2 := httptest.NewRequest(http.MethodPost, "/my/onboarding/request-organization", strings.NewReader("name=New+Org"))
	post2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post2.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	post2Rec := httptest.NewRecorder()
	server.handleMyRoutes(post2Rec, post2)
	if post2Rec.Code != http.StatusSeeOther {
		t.Fatalf("resubmit status=%d body=%q (form error / blocked?)", post2Rec.Code, post2Rec.Body.String())
	}

	pendingAfter, err := store.ListPendingOrganizationCreationRequests(context.Background())
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pendingAfter) != 1 || pendingAfter[0].ProposedSlug != "new-org" {
		t.Fatalf("pending after resubmit=%+v, want one new-org request", pendingAfter)
	}
	if pendingAfter[0].ID == firstReqID {
		t.Fatal("expected a new request document, not the previously approved one")
	}

	// 5) Platform admin must see the new pending request.
	adminGET := httptest.NewRequest(http.MethodGet, "/admin/organizations", nil)
	adminGET.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	adminRec := httptest.NewRecorder()
	server.handleAdminOrgs(adminRec, adminGET)
	if adminRec.Code != http.StatusOK {
		t.Fatalf("admin get status=%d body=%q", adminRec.Code, adminRec.Body.String())
	}
	body := adminRec.Body.String()
	if !strings.Contains(body, "New Org") {
		t.Fatalf("platform admin does not see new pending request; body:\n%s", body)
	}
	if !strings.Contains(body, `name="intent" value="approve_org_creation"`) {
		t.Fatalf("expected approve action for pending request, body:\n%s", body)
	}

	// Residual managed labels must be stripped by DeleteOrganization.
	snap := identity.snapshot("user-founder")
	if hasIdentityLabel(snap.Labels, identityOrgAdminLabel) {
		t.Fatalf("expected org-admin label stripped after delete_org, labels=%v", snap.Labels)
	}
}

// Ghost membership after delete: team gone but ListMemberships still returns the old membership.
// DeleteOrganization + unresolved-team clearing must still let the founder resubmit.
func TestDeleteOrganizationGhostMembershipStillAllowsResubmit(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-founder-resubmit"
	founder := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "user-founder",
		Email:          "founder@example.com",
		Status:         "active",
		CreatedAt:      now,
	}
	store := NewMemoryStore()
	identity := newLifecycleIdentityStore(founder)
	identity.deleteOrganizationAsAdminFunc = func(_ context.Context, orgSlug string) error {
		identity.mu.Lock()
		defer identity.mu.Unlock()
		slug := strings.TrimSpace(orgSlug)
		if _, ok := identity.orgs[slug]; !ok {
			return ErrIdentityNotFound
		}
		delete(identity.orgs, slug)
		// Intentionally do NOT clear memberships — models a ghost membership row.
		return nil
	}

	server := &Server{
		authorizer:  fakeAuthorizer{},
		store:       store,
		identity:    identity,
		tmpl:        parseTestTemplates(t),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	postReq := httptest.NewRequest(http.MethodPost, "/my/onboarding/request-organization", strings.NewReader("name=New+Org"))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	postRec := httptest.NewRecorder()
	server.handleMyRoutes(postRec, postReq)
	if postRec.Code != http.StatusSeeOther {
		t.Fatalf("submit status=%d", postRec.Code)
	}
	pending, _ := store.ListPendingOrganizationCreationRequests(context.Background())
	approveHTTP := httptest.NewRequest(http.MethodPost, "/admin/organizations", strings.NewReader("intent=approve_org_creation&request_id="+pending[0].ID.Hex()))
	approveHTTP.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	approveHTTP.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	server.handleAdminOrgs(httptest.NewRecorder(), approveHTTP)

	deleteHTTP := httptest.NewRequest(http.MethodPost, "/admin/organizations", strings.NewReader("intent=delete_org&org_slug=new-org"))
	deleteHTTP.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	deleteHTTP.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	deleteRec := httptest.NewRecorder()
	server.handleAdminOrgs(deleteRec, deleteHTTP)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%q", deleteRec.Code, deleteRec.Body.String())
	}

	if identity.userAffiliated("user-founder") {
		t.Fatal("founder still affiliated after delete with ghost membership")
	}
	if hasIdentityLabel(identity.snapshot("user-founder").Labels, identityOrgAdminLabel) {
		t.Fatal("expected labels stripped even when memberships were not cascaded")
	}

	post2 := httptest.NewRequest(http.MethodPost, "/my/onboarding/request-organization", strings.NewReader("name=New+Org"))
	post2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post2.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	post2Rec := httptest.NewRecorder()
	server.handleMyRoutes(post2Rec, post2)
	if post2Rec.Code != http.StatusSeeOther || post2Rec.Header().Get("Location") != "/my/onboarding" {
		t.Fatalf("resubmit status=%d loc=%q body=%q", post2Rec.Code, post2Rec.Header().Get("Location"), post2Rec.Body.String())
	}

	pendingAfter, err := store.ListPendingOrganizationCreationRequests(context.Background())
	if err != nil || len(pendingAfter) != 1 || pendingAfter[0].ProposedSlug != "new-org" {
		t.Fatalf("pending after resubmit=%+v err=%v", pendingAfter, err)
	}

	adminGET := httptest.NewRequest(http.MethodGet, "/admin/organizations", nil)
	adminGET.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	adminRec := httptest.NewRecorder()
	server.handleAdminOrgs(adminRec, adminGET)
	if !strings.Contains(adminRec.Body.String(), "New Org") {
		t.Fatalf("platform admin does not see new pending request; body:\n%s", adminRec.Body.String())
	}
}

// lifecycleIdentityStore models Appwrite org + membership + labels for the approve/delete/resubmit path.
type lifecycleIdentityStore struct {
	fakeIdentityStore
	mu           sync.Mutex
	sessionUsers map[string]AccountUser
	orgs         map[string]IdentityOrg
	memberships  map[string][]IdentityMembership // userID → memberships
	users        map[string]IdentityUser
}

func newLifecycleIdentityStore(founder AccountUser) *lifecycleIdentityStore {
	s := &lifecycleIdentityStore{
		sessionUsers: map[string]AccountUser{"session-founder-resubmit": founder},
		orgs:         map[string]IdentityOrg{},
		memberships:  map[string][]IdentityMembership{},
		users: map[string]IdentityUser{
			"user-founder": {
				ID:     "user-founder",
				Email:  "founder@example.com",
				Status: "active",
			},
		},
	}
	s.getSessionFunc = func(_ context.Context, secret string) (IdentitySession, error) {
		if secret == platformAdminSessionValue() {
			return IdentitySession{Secret: secret, UserID: "platform-admin", ExpiresAt: time.Now().Add(time.Hour)}, nil
		}
		if _, ok := s.sessionUsers[secret]; ok {
			return IdentitySession{Secret: secret, UserID: "user-founder", ExpiresAt: time.Now().Add(time.Hour)}, nil
		}
		return IdentitySession{}, ErrIdentityUnauthorized
	}
	s.getCurrentUserFunc = func(ctx context.Context, secret string) (IdentityUser, error) {
		if secret == platformAdminSessionValue() {
			return IdentityUser{ID: "platform-admin", Email: "admin@example.com", Status: "active"}, nil
		}
		return s.GetUserByID(ctx, "user-founder")
	}
	s.getUserByIDFunc = func(_ context.Context, userID string) (IdentityUser, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		u, ok := s.users[userID]
		if !ok {
			return IdentityUser{}, ErrIdentityNotFound
		}
		out := u
		if mems := s.memberships[userID]; len(mems) > 0 {
			m := mems[0]
			if _, orgOK := s.orgs[m.OrgSlug]; orgOK {
				out.OrgSlug = m.OrgSlug
				out.MembershipID = m.ID
				out.IsOrgAdmin = m.IsOrgAdmin || hasIdentityLabel(out.Labels, identityOrgAdminLabel)
			} else {
				// Mirrors appwriteIdentity: unresolved team → not affiliated.
				out.OrgSlug = ""
				out.MembershipID = ""
				out.IsOrgAdmin = hasIdentityLabel(out.Labels, identityOrgAdminLabel)
			}
		} else {
			out.OrgSlug = ""
			out.MembershipID = ""
			out.IsOrgAdmin = hasIdentityLabel(out.Labels, identityOrgAdminLabel)
		}
		return out, nil
	}
	s.listOrganizationMembershipsFunc = func(_ context.Context, orgSlug string) ([]IdentityMembership, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		out := make([]IdentityMembership, 0)
		for _, mems := range s.memberships {
			for _, m := range mems {
				if m.OrgSlug == orgSlug {
					out = append(out, m)
				}
			}
		}
		return out, nil
	}
	s.getOrganizationBySlugFunc = func(_ context.Context, slug string) (*IdentityOrg, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		org, ok := s.orgs[strings.TrimSpace(slug)]
		if !ok {
			return nil, ErrIdentityNotFound
		}
		cp := org
		return &cp, nil
	}
	s.listOrganizationsPageFunc = func(_ context.Context, _ IdentityOrgListOptions) (IdentityOrgPage, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		orgs := make([]IdentityOrg, 0, len(s.orgs))
		for _, org := range s.orgs {
			orgs = append(orgs, org)
		}
		return IdentityOrgPage{Organizations: orgs, Total: len(orgs)}, nil
	}
	s.listOrganizationsFunc = func(_ context.Context) ([]IdentityOrg, error) {
		page, err := s.listOrganizationsPageFunc(context.Background(), IdentityOrgListOptions{})
		return page.Organizations, err
	}
	s.createOrganizationAsAdminFunc = func(_ context.Context, name string) (IdentityOrg, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		slug := canonifySlug(name)
		org := IdentityOrg{ID: slug, Slug: slug, Name: strings.TrimSpace(name)}
		s.orgs[slug] = org
		return org, nil
	}
	s.deleteOrganizationAsAdminFunc = func(_ context.Context, orgSlug string) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		slug := strings.TrimSpace(orgSlug)
		if _, ok := s.orgs[slug]; !ok {
			return ErrIdentityNotFound
		}
		delete(s.orgs, slug)
		// Appwrite cascades team memberships on team delete.
		for userID, mems := range s.memberships {
			kept := mems[:0]
			for _, m := range mems {
				if m.OrgSlug != slug {
					kept = append(kept, m)
				}
			}
			s.memberships[userID] = kept
		}
		// Labels are NOT cleared by Appwrite team delete — matches production.
		return nil
	}
	s.addOrganizationUserByIDAsAdminFunc = func(_ context.Context, orgSlug, userID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		m := IdentityMembership{
			ID:         "mem-" + userID,
			UserID:     userID,
			OrgSlug:    orgSlug,
			RoleSlugs:  append([]string(nil), roleSlugs...),
			IsOrgAdmin: isOrgAdmin,
			Confirmed:  true,
		}
		s.memberships[userID] = []IdentityMembership{m}
		return m, nil
	}
	s.updateUserLabelsFunc = func(_ context.Context, userID string, labels []string) (IdentityUser, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		u, ok := s.users[userID]
		if !ok {
			return IdentityUser{}, ErrIdentityNotFound
		}
		u.Labels = append([]string(nil), labels...)
		u.IsOrgAdmin = hasIdentityLabel(labels, identityOrgAdminLabel)
		s.users[userID] = u
		return u, nil
	}
	s.listOrganizationUsersFunc = func(_ context.Context, orgSlug string) ([]IdentityUser, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		out := make([]IdentityUser, 0)
		for userID, mems := range s.memberships {
			for _, m := range mems {
				if m.OrgSlug != orgSlug {
					continue
				}
				u := s.users[userID]
				u.OrgSlug = orgSlug
				u.IsOrgAdmin = m.IsOrgAdmin || hasIdentityLabel(u.Labels, identityOrgAdminLabel)
				out = append(out, u)
			}
		}
		return out, nil
	}
	return s
}

func (s *lifecycleIdentityStore) hasOrg(slug string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.orgs[slug]
	return ok
}

func (s *lifecycleIdentityStore) userAffiliated(userID string) bool {
	u, err := s.GetUserByID(context.Background(), userID)
	if err != nil {
		return false
	}
	return strings.TrimSpace(u.OrgSlug) != ""
}

func (s *lifecycleIdentityStore) snapshot(userID string) IdentityUser {
	u, _ := s.GetUserByID(context.Background(), userID)
	return u
}
