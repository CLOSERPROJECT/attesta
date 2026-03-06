package main

import (
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestHandleOrgAdminFormataBuilderGet(t *testing.T) {
	store := NewMemoryStore()
	server := &Server{store: store, enforceAuth: true, now: time.Now}

	org, err := store.CreateOrganization(t.Context(), Organization{Name: "Builder Org"})
	if err != nil {
		t.Fatalf("CreateOrganization error: %v", err)
	}
	if _, err := store.CreateRole(t.Context(), Role{
		OrgID:     org.ID,
		OrgSlug:   org.Slug,
		Slug:      "org-admin",
		Name:      "Org Admin",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateRole error: %v", err)
	}
	orgID := org.ID
	orgAdmin, err := store.CreateUser(t.Context(), AccountUser{
		OrgID:     &orgID,
		OrgSlug:   org.Slug,
		Email:     "org-admin-builder@example.com",
		RoleSlugs: []string{"org-admin"},
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser org-admin error: %v", err)
	}
	orgAdminSession := createSessionForTestUser(t, store, orgAdmin)

	t.Run("unauthenticated redirects to login", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/org-admin/formata-builder", nil)
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
		}
	})

	t.Run("org admin can load builder", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/org-admin/formata-builder", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if !strings.Contains(strings.ToLower(body), "<!doctype html>") {
			t.Fatalf("expected html response body, got %q", body)
		}
		if !strings.Contains(body, "/org-admin/formata-builder/assets/") {
			t.Fatalf("expected rewritten asset prefix in html, got %q", body)
		}
		if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "no-store") {
			t.Fatalf("cache-control = %q, want no-store", got)
		}
	})

	t.Run("org admin can load nested route fallback", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/org-admin/formata-builder/editor/home", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if !strings.Contains(strings.ToLower(body), "<!doctype html>") {
			t.Fatalf("expected html fallback body, got %q", body)
		}
		if !strings.Contains(body, "/org-admin/formata-builder/assets/") {
			t.Fatalf("expected rewritten asset prefix in fallback html, got %q", body)
		}
	})

	t.Run("missing static asset returns not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/org-admin/formata-builder/assets/app.js", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("public icon asset is served", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/org-admin/formata-builder/vite.svg", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if !strings.Contains(rec.Header().Get("Content-Type"), "image/svg+xml") {
			t.Fatalf("content-type = %q, want svg content type", rec.Header().Get("Content-Type"))
		}
	})

	t.Run("js assets are rewritten from legacy absolute prefix", func(t *testing.T) {
		indexReq := httptest.NewRequest(http.MethodGet, "/org-admin/formata-builder", nil)
		indexReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		indexRec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(indexRec, indexReq)
		if indexRec.Code != http.StatusOK {
			t.Fatalf("index status = %d, want %d", indexRec.Code, http.StatusOK)
		}
		jsPath := findFirstBuilderAssetPath(t, indexRec.Body.String(), `src="(/org-admin/formata-builder/assets/[^"]+\.js)"`)

		req := httptest.NewRequest(http.MethodGet, jsPath, nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if strings.Contains(body, "/formata-arch/") {
			t.Fatalf("expected rewritten js body, still contains legacy prefix")
		}
		if !strings.Contains(body, "/org-admin/formata-builder/") {
			t.Fatalf("expected rewritten js body to contain org-admin prefix")
		}
		if !strings.Contains(rec.Header().Get("Content-Type"), "javascript") {
			t.Fatalf("content-type = %q, want javascript content type", rec.Header().Get("Content-Type"))
		}
	})

	t.Run("css assets are served with stylesheet content type", func(t *testing.T) {
		indexReq := httptest.NewRequest(http.MethodGet, "/org-admin/formata-builder", nil)
		indexReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		indexRec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(indexRec, indexReq)
		if indexRec.Code != http.StatusOK {
			t.Fatalf("index status = %d, want %d", indexRec.Code, http.StatusOK)
		}
		cssPath := findFirstBuilderAssetPath(t, indexRec.Body.String(), `href="(/org-admin/formata-builder/assets/[^"]+\.css)"`)

		req := httptest.NewRequest(http.MethodGet, cssPath, nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if !strings.Contains(rec.Header().Get("Content-Type"), "text/css") {
			t.Fatalf("content-type = %q, want text/css", rec.Header().Get("Content-Type"))
		}
	})

	t.Run("platform admin get is forbidden", func(t *testing.T) {
		platformAdmin, err := store.CreateUser(t.Context(), AccountUser{
			Email:           "platform-builder-get@example.com",
			IsPlatformAdmin: true,
			Status:          "active",
			CreatedAt:       time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("CreateUser platform admin error: %v", err)
		}
		sessionID := createSessionForTestUser(t, store, platformAdmin)
		req := httptest.NewRequest(http.MethodGet, "/org-admin/formata-builder", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/org-admin/formata-builder", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})
}

func TestHandleOrgAdminFormataBuilderPost(t *testing.T) {
	store := NewMemoryStore()
	server := &Server{store: store, enforceAuth: true, now: time.Now}

	org, err := store.CreateOrganization(t.Context(), Organization{Name: "Builder Post Org"})
	if err != nil {
		t.Fatalf("CreateOrganization error: %v", err)
	}
	if _, err := store.CreateRole(t.Context(), Role{
		OrgID:     org.ID,
		OrgSlug:   org.Slug,
		Slug:      "org-admin",
		Name:      "Org Admin",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateRole error: %v", err)
	}
	orgID := org.ID
	orgAdmin, err := store.CreateUser(t.Context(), AccountUser{
		OrgID:     &orgID,
		OrgSlug:   org.Slug,
		Email:     "org-admin-builder-post@example.com",
		RoleSlugs: []string{"org-admin"},
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser org-admin error: %v", err)
	}
	orgAdminSession := createSessionForTestUser(t, store, orgAdmin)

	platformAdmin, err := store.CreateUser(t.Context(), AccountUser{
		Email:           "platform-builder-post@example.com",
		IsPlatformAdmin: true,
		Status:          "active",
		CreatedAt:       time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser platform admin error: %v", err)
	}
	platformSession := createSessionForTestUser(t, store, platformAdmin)

	t.Run("unauthenticated is unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/org-admin/formata-builder", strings.NewReader("stream"))
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("forbidden for non admin role", func(t *testing.T) {
		plain, err := store.CreateUser(t.Context(), AccountUser{
			Email:     "plain-builder-post@example.com",
			RoleSlugs: []string{"inspector"},
			Status:    "active",
			CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("CreateUser plain error: %v", err)
		}
		sessionID := createSessionForTestUser(t, store, plain)
		req := httptest.NewRequest(http.MethodPost, "/org-admin/formata-builder", strings.NewReader("stream"))
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("org admin can save stream", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/org-admin/formata-builder", strings.NewReader(`{"nodes":[]}`))
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
		}
		stream, err := store.LoadFormataBuilderStream(t.Context())
		if err != nil {
			t.Fatalf("LoadFormataBuilderStream error: %v", err)
		}
		if stream.Stream != `{"nodes":[]}` {
			t.Fatalf("stream = %q, want %q", stream.Stream, `{"nodes":[]}`)
		}
		if stream.UpdatedByUserMongoID != orgAdmin.ID {
			t.Fatalf("updatedByUserMongoID = %s, want %s", stream.UpdatedByUserMongoID.Hex(), orgAdmin.ID.Hex())
		}
	})

	t.Run("rejects non-root post path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/org-admin/formata-builder/assets/app.js", strings.NewReader(`{"nodes":[]}`))
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("platform admin can save stream", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/org-admin/formata-builder", strings.NewReader(`{"nodes":[1]}`))
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
		}
		stream, err := store.LoadFormataBuilderStream(t.Context())
		if err != nil {
			t.Fatalf("LoadFormataBuilderStream error: %v", err)
		}
		if stream.Stream != `{"nodes":[1]}` {
			t.Fatalf("stream = %q, want %q", stream.Stream, `{"nodes":[1]}`)
		}
		if stream.UpdatedByUserMongoID != platformAdmin.ID {
			t.Fatalf("updatedByUserMongoID = %s, want %s", stream.UpdatedByUserMongoID.Hex(), platformAdmin.ID.Hex())
		}
	})

	t.Run("rejects empty stream body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/org-admin/formata-builder", strings.NewReader("   \n\t"))
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("rejects oversized stream body", func(t *testing.T) {
		t.Setenv("FORMATA_STREAM_MAX_BYTES", "8")
		req := httptest.NewRequest(http.MethodPost, "/org-admin/formata-builder", strings.NewReader("0123456789"))
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
		}
	})
}

func TestFormataBuilderHelpers(t *testing.T) {
	t.Run("default max bytes when env invalid", func(t *testing.T) {
		t.Setenv("FORMATA_STREAM_MAX_BYTES", "invalid")
		if got := formataBuilderStreamMaxBytes(); got != 1<<20 {
			t.Fatalf("max bytes = %d, want %d", got, int64(1<<20))
		}
	})

	t.Run("default max bytes when env <= 0", func(t *testing.T) {
		t.Setenv("FORMATA_STREAM_MAX_BYTES", "0")
		if got := formataBuilderStreamMaxBytes(); got != 1<<20 {
			t.Fatalf("max bytes = %d, want %d", got, int64(1<<20))
		}
	})

	t.Run("read embedded index asset", func(t *testing.T) {
		data, contentType, err := readFormataBuilderAsset("index.html")
		if err != nil {
			t.Fatalf("readFormataBuilderAsset error: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("expected non-empty asset data")
		}
		if contentType == "" {
			t.Fatal("expected detected content type")
		}
	})

	t.Run("read embedded asset rejects traversal path", func(t *testing.T) {
		_, _, err := readFormataBuilderAsset("../secrets.txt")
		if !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("error = %v, want fs.ErrNotExist", err)
		}
	})
}

func findFirstBuilderAssetPath(t *testing.T, body, pattern string) string {
	t.Helper()
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		t.Fatalf("failed to extract path with pattern %q from body %q", pattern, body)
	}
	return matches[1]
}
