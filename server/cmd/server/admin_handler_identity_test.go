package main

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestHandleOrgAdminUsersCreateOrgWithIdentity(t *testing.T) {
	now := time.Now().UTC()
	currentUser := IdentityUser{
		ID:         "user-1",
		Email:      "owner@example.com",
		Labels:     []string{identityOrgAdminLabel},
		IsOrgAdmin: true,
		Status:     "active",
	}
	createdOrg := IdentityOrg{}
	var createSessionSecret string
	var createName string

	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return currentUser, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				if strings.EqualFold(strings.TrimSpace(slug), strings.TrimSpace(createdOrg.Slug)) && createdOrg.Slug != "" {
					org := createdOrg
					return &org, nil
				}
				return nil, ErrIdentityNotFound
			},
			listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
				if createdOrg.Slug == "" {
					return nil, nil
				}
				return []IdentityUser{currentUser}, nil
			},
			createOrganizationFunc: func(ctx context.Context, sessionSecret, name string) (IdentityOrg, error) {
				createSessionSecret = sessionSecret
				createName = name
				createdOrg = IdentityOrg{ID: "team-1", Slug: "fresh-org", Name: "Fresh Org"}
				currentUser.OrgSlug = createdOrg.Slug
				currentUser.OrgName = createdOrg.Name
				return createdOrg, nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	form := url.Values{}
	form.Set("intent", "create_org")
	form.Set("name", "Fresh Org")
	req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()

	server.handleOrgAdminUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if createSessionSecret != "session-1" || createName != "Fresh Org" {
		t.Fatalf("create args = %q %q", createSessionSecret, createName)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "ORG_ADMIN fresh-org") {
		t.Fatalf("expected org admin body, got %q", body)
	}
	if !strings.Contains(body, "USERS 1") {
		t.Fatalf("expected current user row after bootstrap, got %q", body)
	}
}

func TestHandleOrgAdminUsersCreateOrgIdentityValidation(t *testing.T) {
	now := time.Now().UTC()
	createCalls := 0
	server := &Server{
		store: NewMemoryStore(),
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
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				if strings.EqualFold(strings.TrimSpace(slug), "fresh-org") {
					org := IdentityOrg{ID: "team-1", Slug: "fresh-org", Name: "Fresh Org"}
					return &org, nil
				}
				return nil, ErrIdentityNotFound
			},
			createOrganizationFunc: func(ctx context.Context, sessionSecret, name string) (IdentityOrg, error) {
				createCalls++
				return IdentityOrg{}, nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=create_org&name=Fresh+Org"))
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
	if !strings.Contains(rec.Body.String(), "organization slug already exists") {
		t.Fatalf("expected duplicate slug message, got %q", rec.Body.String())
	}
}

func TestHandleOrgAdminUsersUpdateOrgWithIdentityLogo(t *testing.T) {
	now := time.Now().UTC()
	currentUser := IdentityUser{
		ID:         "user-1",
		Email:      "owner@example.com",
		OrgSlug:    "acme",
		OrgName:    "Acme Org",
		Labels:     []string{identityOrgAdminLabel},
		IsOrgAdmin: true,
		Status:     "active",
	}
	org := IdentityOrg{
		ID:         "team-1",
		Slug:       "acme",
		Name:       "Acme Org",
		LogoFileID: "logo-old",
		Roles:      []IdentityRole{{Slug: "qa-reviewer", Name: "QA Reviewer"}},
	}
	var uploaded IdentityFile
	var updateSessionSecret string
	var updateCurrentSlug string
	var updateName string
	var updateLogoFileID string
	var updateRoles []IdentityRole

	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return currentUser, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				if strings.EqualFold(strings.TrimSpace(slug), "updated-name-org") && org.Slug == "updated-name-org" {
					current := org
					return &current, nil
				}
				if strings.EqualFold(strings.TrimSpace(slug), "acme") {
					current := org
					return &current, nil
				}
				return nil, ErrIdentityNotFound
			},
			uploadOrganizationLogoFunc: func(ctx context.Context, orgSlug string, file IdentityFile) (IdentityFile, error) {
				uploaded = file
				return IdentityFile{ID: "logo-new", Filename: file.Filename, ContentType: file.ContentType}, nil
			},
			updateOrganizationFunc: func(ctx context.Context, sessionSecret, currentSlug, name, logoFileID string, roles []IdentityRole) (IdentityOrg, error) {
				updateSessionSecret = sessionSecret
				updateCurrentSlug = currentSlug
				updateName = name
				updateLogoFileID = logoFileID
				updateRoles = append([]IdentityRole(nil), roles...)
				org = IdentityOrg{
					ID:         "team-1",
					Slug:       "updated-name-org",
					Name:       "Updated Name Org",
					LogoFileID: logoFileID,
					Roles:      append([]IdentityRole(nil), roles...),
				}
				currentUser.OrgSlug = org.Slug
				currentUser.OrgName = org.Name
				return org, nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	body := &strings.Builder{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("intent", "update_org"); err != nil {
		t.Fatalf("WriteField intent error: %v", err)
	}
	if err := writer.WriteField("name", "Updated Name Org"); err != nil {
		t.Fatalf("WriteField name error: %v", err)
	}
	part, err := writer.CreateFormFile("logo", "logo.png")
	if err != nil {
		t.Fatalf("CreateFormFile error: %v", err)
	}
	if _, err := io.WriteString(part, "PNG"); err != nil {
		t.Fatalf("WriteString logo error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader(body.String()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()

	server.handleOrgAdminUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if updateSessionSecret != "session-1" || updateCurrentSlug != "acme" || updateName != "Updated Name Org" || updateLogoFileID != "logo-new" {
		t.Fatalf("update args = %q %q %q %q", updateSessionSecret, updateCurrentSlug, updateName, updateLogoFileID)
	}
	if uploaded.Filename != "logo.png" {
		t.Fatalf("uploaded file = %#v", uploaded)
	}
	if len(updateRoles) != 1 || updateRoles[0].Slug != "qa-reviewer" {
		t.Fatalf("update roles = %#v", updateRoles)
	}
	if !strings.Contains(rec.Body.String(), "ORG_ADMIN updated-name-org") {
		t.Fatalf("expected updated org slug in body, got %q", rec.Body.String())
	}
}

func TestHandleOrgAdminLogoWithIdentity(t *testing.T) {
	now := time.Now().UTC()
	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{
					ID:         "user-1",
					Email:      "owner@example.com",
					OrgSlug:    "acme",
					OrgName:    "Acme Org",
					Labels:     []string{identityOrgAdminLabel},
					IsOrgAdmin: true,
					Status:     "active",
				}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org", LogoFileID: "logo-1"}
				return &org, nil
			},
			getOrganizationLogoFunc: func(ctx context.Context, fileID string) (IdentityFile, error) {
				return IdentityFile{
					ID:          fileID,
					Filename:    "logo.svg",
					ContentType: "image/svg+xml",
					Data:        []byte("<svg/>"),
				}, nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/org-admin/logo/logo-1", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()

	server.handleOrgAdminLogo(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get("Content-Type") != "image/svg+xml" {
		t.Fatalf("content type = %q", rec.Header().Get("Content-Type"))
	}
	if body := rec.Body.String(); body != "<svg/>" {
		t.Fatalf("body = %q", body)
	}
}

func TestHandleOrgAdminRolesWithIdentity(t *testing.T) {
	now := time.Now().UTC()
	org := IdentityOrg{
		ID:    "team-1",
		Slug:  "acme",
		Name:  "Acme Org",
		Roles: []IdentityRole{{Slug: "qa-reviewer", Name: "QA Reviewer"}},
	}
	var updateSessionSecret string
	var updatedRoles []IdentityRole

	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{
					ID:         "user-1",
					Email:      "owner@example.com",
					OrgSlug:    "acme",
					OrgName:    "Acme Org",
					Labels:     []string{identityOrgAdminLabel},
					IsOrgAdmin: true,
					Status:     "active",
				}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				current := org
				return &current, nil
			},
			updateOrganizationFunc: func(ctx context.Context, sessionSecret, currentSlug, name, logoFileID string, roles []IdentityRole) (IdentityOrg, error) {
				updateSessionSecret = sessionSecret
				updatedRoles = append([]IdentityRole(nil), roles...)
				org.Roles = append([]IdentityRole(nil), roles...)
				return org, nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodPost, "/org-admin/roles", strings.NewReader("name=Approver&palette=blue"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()

	server.handleOrgAdminRoles(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if updateSessionSecret != "session-1" {
		t.Fatalf("session secret = %q", updateSessionSecret)
	}
	if len(updatedRoles) != 2 || updatedRoles[1].Slug != "approver" {
		t.Fatalf("updated roles = %#v", updatedRoles)
	}
}

func TestHandleOrgAdminUsersSetRolesWithIdentity(t *testing.T) {
	now := time.Now().UTC()
	users := []IdentityUser{
		{
			ID:         "user-1",
			Email:      "owner@example.com",
			OrgSlug:    "acme",
			Labels:     []string{identityOrgAdminLabel, encodeIdentityRoleLabel("qa-reviewer")},
			IsOrgAdmin: true,
			Status:     "active",
		},
		{
			ID:      "user-2",
			Email:   "member@example.com",
			OrgSlug: "acme",
			Labels:  []string{encodeIdentityRoleLabel("qa-reviewer")},
			Status:  "active",
		},
	}
	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return users[0], nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				org := IdentityOrg{
					ID:   "team-1",
					Slug: "acme",
					Name: "Acme Org",
					Roles: []IdentityRole{
						{Slug: "qa-reviewer", Name: "QA Reviewer"},
						{Slug: "approver", Name: "Approver"},
					},
				}
				return &org, nil
			},
			listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
				return append([]IdentityUser(nil), users...), nil
			},
			updateUserLabelsFunc: func(ctx context.Context, userID string, labels []string) (IdentityUser, error) {
				for idx := range users {
					if users[idx].ID != userID {
						continue
					}
					users[idx].Labels = append([]string(nil), labels...)
					users[idx].IsOrgAdmin = hasIdentityLabel(labels, identityOrgAdminLabel)
					return users[idx], nil
				}
				return IdentityUser{}, ErrIdentityNotFound
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	targetID := stableIdentityUserObjectID("user-2").Hex()
	req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=set_roles&userMongoId="+targetID+"&roles=approver&roles=org-admin"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()

	server.handleOrgAdminUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !hasIdentityLabel(users[1].Labels, identityOrgAdminLabel) || !containsRole(decodeIdentityRoleLabels(users[1].Labels), "approver") {
		t.Fatalf("updated user labels = %#v", users[1].Labels)
	}

	selfID := stableIdentityUserObjectID("user-1").Hex()
	selfReq := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=set_roles&userMongoId="+selfID+"&roles=approver"))
	selfReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	selfReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	selfRec := httptest.NewRecorder()

	server.handleOrgAdminUsers(selfRec, selfReq)

	if selfRec.Code != http.StatusOK {
		t.Fatalf("self status = %d, want %d", selfRec.Code, http.StatusOK)
	}
	if !strings.Contains(selfRec.Body.String(), "cannot remove org-admin from your own account") {
		t.Fatalf("expected self-protection message, got %q", selfRec.Body.String())
	}
}

func TestHandleOrgAdminUsersInviteWithIdentity(t *testing.T) {
	now := time.Now().UTC()
	inviteCalls := 0
	var invitedRoles []string
	var invitedAdmin bool
	var invitedRedirect string
	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{
					ID:         "user-1",
					Email:      "owner@example.com",
					OrgSlug:    "acme",
					OrgName:    "Acme Org",
					Labels:     []string{identityOrgAdminLabel},
					IsOrgAdmin: true,
					Status:     "active",
				}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				org := IdentityOrg{
					ID:   "team-1",
					Slug: "acme",
					Name: "Acme Org",
					Roles: []IdentityRole{
						{Slug: "qa-reviewer", Name: "QA Reviewer"},
						{Slug: "approver", Name: "Approver"},
					},
				}
				return &org, nil
			},
			listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
				return nil, nil
			},
			inviteOrganizationUserFunc: func(ctx context.Context, sessionSecret, orgSlug, email, redirectURL string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
				inviteCalls++
				invitedRoles = append([]string(nil), roleSlugs...)
				invitedAdmin = isOrgAdmin
				invitedRedirect = redirectURL
				return IdentityMembership{ID: "membership-1", Email: email}, nil
			},
			listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
				return []IdentityUser{{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"}}, nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=invite&email=new%40example.com&roles=approver&roles=org-admin"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()

	server.handleOrgAdminUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if inviteCalls != 1 || len(invitedRoles) != 1 || invitedRoles[0] != "approver" || !invitedAdmin {
		t.Fatalf("invite call = %d roles=%#v admin=%v", inviteCalls, invitedRoles, invitedAdmin)
	}
	if !strings.Contains(invitedRedirect, "/invite/accept") {
		t.Fatalf("redirect = %q", invitedRedirect)
	}
}

func TestHandleOrgAdminUsersInviteIdentityDuplicatePending(t *testing.T) {
	now := time.Now().UTC()
	inviteCalls := 0
	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org", Roles: []IdentityRole{{Slug: "approver", Name: "Approver"}}}
				return &org, nil
			},
			listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
				return []IdentityMembership{{ID: "membership-1", Email: "pending@example.com", RoleSlugs: []string{"approver"}, Confirmed: false}}, nil
			},
			inviteOrganizationUserFunc: func(ctx context.Context, sessionSecret, orgSlug, email, redirectURL string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
				inviteCalls++
				return IdentityMembership{}, nil
			},
			listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
				return nil, nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=invite&email=pending%40example.com&roles=approver"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()

	server.handleOrgAdminUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if inviteCalls != 0 {
		t.Fatalf("invite calls = %d, want 0", inviteCalls)
	}
}

func TestHandleOrgAdminUsersInviteIdentityExistingAndCrossOrg(t *testing.T) {
	now := time.Now().UTC()
	updatedUsers := make(map[string][]string)
	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org", Roles: []IdentityRole{{Slug: "approver", Name: "Approver"}}}
				return &org, nil
			},
			listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
				return nil, nil
			},
			getUserByEmailFunc: func(ctx context.Context, email string) (IdentityUser, error) {
				switch email {
				case "member@example.com":
					return IdentityUser{ID: "user-2", Email: email, OrgSlug: "acme", Labels: []string{encodeIdentityRoleLabel("approver")}, Status: "active"}, nil
				case "other@example.com":
					return IdentityUser{ID: "user-3", Email: email, OrgSlug: "other-org", Labels: []string{encodeIdentityRoleLabel("approver")}, Status: "active"}, nil
				default:
					return IdentityUser{}, ErrIdentityNotFound
				}
			},
			updateUserLabelsFunc: func(ctx context.Context, userID string, labels []string) (IdentityUser, error) {
				updatedUsers[userID] = append([]string(nil), labels...)
				return IdentityUser{ID: userID, Labels: labels}, nil
			},
			listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
				return []IdentityUser{{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"}}, nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=invite&email=member%40example.com&roles=approver"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()
	server.handleOrgAdminUsers(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("existing member status = %d, want %d", rec.Code, http.StatusOK)
	}
	if len(updatedUsers["user-2"]) != 1 || updatedUsers["user-2"][0] != encodeIdentityRoleLabel("approver") {
		t.Fatalf("updated user labels = %#v", updatedUsers)
	}

	reqOther := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=invite&email=other%40example.com&roles=approver"))
	reqOther.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqOther.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	recOther := httptest.NewRecorder()
	server.handleOrgAdminUsers(recOther, reqOther)
	if recOther.Code != http.StatusOK {
		t.Fatalf("cross-org status = %d, want %d", recOther.Code, http.StatusOK)
	}
	if !strings.Contains(recOther.Body.String(), "email already belongs to another organization") {
		t.Fatalf("expected cross-org error, got %q", recOther.Body.String())
	}
}

func TestHandleOrgAdminUsersDeleteUserWithIdentity(t *testing.T) {
	now := time.Now().UTC()
	deletedMemberships := []string{}
	updatedUsers := map[string][]string{}
	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"}, nil
			},
			listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
				return []IdentityMembership{
					{ID: "membership-1", UserID: "user-1", Email: "owner@example.com", Confirmed: true},
					{ID: "membership-2", UserID: "user-2", Email: "member@example.com", Confirmed: true},
				}, nil
			},
			deleteOrganizationMembershipFunc: func(ctx context.Context, sessionSecret, orgSlug, membershipID string) error {
				deletedMemberships = append(deletedMemberships, membershipID)
				return nil
			},
			getUserByIDFunc: func(ctx context.Context, userID string) (IdentityUser, error) {
				return IdentityUser{ID: userID, Email: "member@example.com", Labels: []string{"custom:keep", encodeIdentityRoleLabel("approver"), identityOrgAdminLabel}, Status: "active"}, nil
			},
			updateUserLabelsFunc: func(ctx context.Context, userID string, labels []string) (IdentityUser, error) {
				updatedUsers[userID] = append([]string(nil), labels...)
				return IdentityUser{ID: userID, Labels: labels}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org"}
				return &org, nil
			},
			listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
				return []IdentityUser{{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"}}, nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	targetID := stableIdentityUserObjectID("user-2").Hex()
	req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=delete_user&userMongoId="+targetID))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()
	server.handleOrgAdminUsers(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if len(deletedMemberships) != 1 || deletedMemberships[0] != "membership-2" {
		t.Fatalf("deleted memberships = %#v", deletedMemberships)
	}
	if labels := updatedUsers["user-2"]; len(labels) != 1 || labels[0] != "custom:keep" {
		t.Fatalf("updated user labels = %#v", updatedUsers)
	}
}
