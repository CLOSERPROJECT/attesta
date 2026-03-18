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
