package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHelperCoverageBranches(t *testing.T) {
	t.Run("user org admin helpers", func(t *testing.T) {
		if userIsOrgAdmin(nil) {
			t.Fatal("nil user should not be org admin")
		}
		if !userIsOrgAdmin(&AccountUser{RoleSlugs: []string{"org_admin"}}) {
			t.Fatal("org_admin role should be accepted")
		}
		if userHasOrganizationContext(nil) {
			t.Fatal("nil user should not have org context")
		}
		if userHasOrganizationContext(&AccountUser{OrgSlug: "acme"}) {
			t.Fatal("user without org id should not have org context")
		}
		orgID := primitive.NewObjectID()
		if userHasOrganizationContext(&AccountUser{OrgID: &orgID}) {
			t.Fatal("user without org slug should not have org context")
		}
	})

	t.Run("selected workflow falls back and errors", func(t *testing.T) {
		server := &Server{
			authorizer: fakeAuthorizer{}, configProvider: func() (RuntimeConfig, error) {
				return RuntimeConfig{}, errors.New("boom")
			}}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, "wrong-type"))
		if _, _, err := server.selectedWorkflow(req); err == nil || !strings.Contains(err.Error(), "boom") {
			t.Fatalf("selectedWorkflow error = %v, want boom", err)
		}
	})

	t.Run("catalog and role helpers", func(t *testing.T) {
		now := time.Now().UTC()
		if sameCatalogModTimes(map[string]time.Time{"a": now}, map[string]time.Time{}) {
			t.Fatal("different lengths should not match")
		}
		if sameCatalogModTimes(map[string]time.Time{"a": now}, map[string]time.Time{"a": now.Add(time.Second)}) {
			t.Fatal("different modtimes should not match")
		}
		server := &Server{
			authorizer: fakeAuthorizer{}}
		cfg := RuntimeConfig{
			Roles:       []WorkflowRole{{Slug: "approver"}},
			Departments: []Department{{ID: "qa"}},
		}
		if !server.isKnownRole(cfg, "approver") {
			t.Fatal("expected role to be known")
		}
		if !server.isKnownRole(cfg, "qa") {
			t.Fatal("expected department role to be known")
		}
		if server.isKnownRole(cfg, "missing") {
			t.Fatal("unexpected known role")
		}
	})

	t.Run("formata and dpp helpers", func(t *testing.T) {
		if got := formataAttachmentFilename("", nil, "image/png"); got != "attachment.png" {
			t.Fatalf("formataAttachmentFilename blank = %q", got)
		}
		if got := formataAttachmentFilename("1.2", []string{"nested", "field"}, "text/plain"); got != "1_2-nested_field.asc" {
			t.Fatalf("formataAttachmentFilename nested = %q", got)
		}
		if got := formataAttachmentFilename("1.1", []string{"payload", "key", "Load 3 files[]", "0"}, "image/png"); got != "1_1-payload_key_Load 3 files_0.png" {
			t.Fatalf("formataAttachmentFilename multifile = %q", got)
		}
		if _, err := dppSerialFromStrategy("unsupported", primitive.NewObjectID()); err == nil {
			t.Fatal("expected unsupported strategy error")
		}
	})
}

func TestRenderPlatformAdminAdditionalBranches(t *testing.T) {
	now := time.Now().UTC()
	server := &Server{
		authorizer: fakeAuthorizer{},
		tmpl:       template.Must(template.New("platform-admin-test").Parse(`{{define "platform_admin.html"}}{{range .Organizations}}{{.Name}}:{{.Slug}}|{{end}}{{.Error}}{{end}}`)),
		identity: &fakeIdentityStore{
			listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
				return []IdentityOrg{
					{ID: "team-2", Slug: "beta", Name: "Acme"},
					{ID: "team-1", Slug: "alpha", Name: "Acme"},
				}, nil
			},
		},
		now: func() time.Time { return now },
	}
	rec := httptest.NewRecorder()

	server.renderPlatformAdmin(rec, &AccountUser{Email: "admin@example.com", IsPlatformAdmin: true}, "", PlatformAdminErrors{Invite: " invite failed "})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "Acme:alpha|Acme:beta|invite failed" {
		t.Fatalf("body = %q", rec.Body.String())
	}

	broken := &Server{
		authorizer: fakeAuthorizer{}, tmpl: template.Must(template.New("broken").Parse(`{{define "platform_admin.html"}}{{template "missing" .}}{{end}}`)), now: func() time.Time { return now }}
	errRec := httptest.NewRecorder()
	broken.renderPlatformAdmin(errRec, &AccountUser{Email: "admin@example.com", IsPlatformAdmin: true}, "", PlatformAdminErrors{})
	if errRec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", errRec.Code, http.StatusInternalServerError)
	}
}

func TestHandlePublicCatalogAdditionalOrderingAndStreamBranches(t *testing.T) {
	now := time.Now().UTC()
	identity := catalogAuthIdentity(now, true)
	identity.listOrganizationsFunc = func(ctx context.Context) ([]IdentityOrg, error) {
		return []IdentityOrg{
			{
				ID:   "team-empty",
				Slug: " ",
				Name: "Ignored",
			},
			{
				ID:   "team-b",
				Slug: "beta",
				Name: "Same",
				Roles: []IdentityRole{
					{Slug: "zeta", Name: "Same"},
					{Slug: "alpha", Name: "Same"},
				},
			},
			{
				ID:   "team-a",
				Slug: "alpha",
				Name: "Same",
				Roles: []IdentityRole{
					{Slug: "builder", Name: "Builder"},
				},
			},
		}, nil
	}
	server := catalogServer(now, identity)

	req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()
	server.handlePublicCatalog(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got PublicCatalogResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got.Organizations) != 2 || got.Organizations[0].Slug != "alpha" || got.Organizations[1].Slug != "beta" {
		t.Fatalf("organizations = %#v", got.Organizations)
	}
	if len(got.Roles) != 3 || got.Roles[0].Slug != "builder" || got.Roles[1].Slug != "alpha" || got.Roles[2].Slug != "zeta" {
		t.Fatalf("roles = %#v", got.Roles)
	}
}

func TestOrganizationLogoAdditionalBranches(t *testing.T) {
	server := &Server{
		authorizer: fakeAuthorizer{}}

	t.Run("no multipart form", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", nil)
		upload, errMsg := server.readOrganizationLogoUpload(req)
		if upload != nil || errMsg != "" {
			t.Fatalf("upload=%#v errMsg=%q", upload, errMsg)
		}
	})

	t.Run("multiple files rejected", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		for _, name := range []string{"logo-a.png", "logo-b.png"} {
			part, err := writer.CreateFormFile("logo", name)
			if err != nil {
				t.Fatalf("CreateFormFile error: %v", err)
			}
			if _, err := part.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}); err != nil {
				t.Fatalf("part.Write error: %v", err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("writer.Close error: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		if err := req.ParseMultipartForm(int64(body.Len())); err != nil {
			t.Fatalf("ParseMultipartForm error: %v", err)
		}

		upload, errMsg := server.readOrganizationLogoUpload(req)
		if upload != nil || errMsg != "upload a single logo file" {
			t.Fatalf("upload=%#v errMsg=%q", upload, errMsg)
		}
	})

	t.Run("generic save failure", func(t *testing.T) {
		server := &Server{
			authorizer: fakeAuthorizer{},
			store:      &failingSaveAttachmentStore{Store: NewMemoryStore(), err: errors.New("boom")},
			now:        time.Now,
		}
		req := organizationLogoRequest(t, "logo.png", "", []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
		attachmentID, errMsg := server.parseOrganizationLogoUpload(context.Background(), req, "Acme Org")
		if attachmentID != "" || errMsg != "failed to upload logo" {
			t.Fatalf("attachmentID=%q errMsg=%q", attachmentID, errMsg)
		}
	})
}

type failingSaveAttachmentStore struct {
	Store
	err error
}

func (s *failingSaveAttachmentStore) SaveAttachment(ctx context.Context, upload AttachmentUpload, source io.Reader) (Attachment, error) {
	return Attachment{}, s.err
}
