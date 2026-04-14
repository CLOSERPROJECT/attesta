package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestHandleOrgAdminFormataBuilderGet(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "platform-admin-builder@example.com")
	t.Setenv("ADMIN_PASSWORD", "secret")

	store := NewMemoryStore()
	orgID := stableOrgObjectID("builder-org")
	orgAdmin := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "builder-org-admin",
		OrgID:          &orgID,
		OrgSlug:        "builder-org",
		Email:          "org-admin-builder@example.com",
		RoleSlugs:      []string{"org-admin"},
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
	}
	orgAdminSession := "session-builder-org-admin"
	server := &Server{
		authorizer:  fakeAuthorizer{},
		store:       store,
		identity:    testIdentityForSessions(time.Now().UTC(), map[string]AccountUser{orgAdminSession: orgAdmin}),
		enforceAuth: true,
		now:         time.Now,
	}

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

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/org-admin/formata-builder", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("stream json omits edit state fields", func(t *testing.T) {
		saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          workflowStreamYAML("Editable stream"),
			CreatedByUserID: formataStreamUserID(&orgAdmin),
			UpdatedByUserID: formataStreamUserID(&orgAdmin),
			UpdatedAt:       time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream error: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/org-admin/formata-builder/stream/"+saved.ID.Hex(), nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
			t.Fatalf("content-type = %q, want application/json", got)
		}

		var payload map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("json.Unmarshal error: %v", err)
		}
		if _, ok := payload["editable"]; ok {
			t.Fatalf("editable present in payload: %#v", payload["editable"])
		}
		if _, ok := payload["editableRequiresPurge"]; ok {
			t.Fatalf("editableRequiresPurge present in payload: %#v", payload["editableRequiresPurge"])
		}
		workflow, ok := payload["workflow"].(map[string]interface{})
		if !ok || workflow["name"] != "Editable stream" {
			t.Fatalf("workflow payload = %#v, want stream name", payload["workflow"])
		}
	})

	t.Run("stream json marks non owner or started stream as not editable", func(t *testing.T) {
		saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          workflowStreamYAML("Started stream"),
			CreatedByUserID: "another-owner",
			UpdatedByUserID: "another-owner",
			UpdatedAt:       time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream error: %v", err)
		}
		store.SeedProcess(Process{
			ID:          primitive.NewObjectID(),
			WorkflowKey: saved.ID.Hex(),
			CreatedAt:   time.Now().UTC(),
			Status:      "active",
			Progress:    map[string]ProcessStep{},
		})

		req := httptest.NewRequest(http.MethodGet, "/org-admin/formata-builder/stream/"+saved.ID.Hex(), nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var payload map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("json.Unmarshal error: %v", err)
		}
		if _, ok := payload["editable"]; ok {
			t.Fatalf("editable present in payload: %#v", payload["editable"])
		}
		if _, ok := payload["editableRequiresPurge"]; ok {
			t.Fatalf("editableRequiresPurge present in payload: %#v", payload["editableRequiresPurge"])
		}
	})

	t.Run("stream json marks platform admin started stream as editable with purge warning", func(t *testing.T) {
		saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          workflowStreamYAML("Platform editable"),
			CreatedByUserID: "another-owner",
			UpdatedByUserID: "another-owner",
			UpdatedAt:       time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream error: %v", err)
		}
		store.SeedProcess(Process{
			ID:          primitive.NewObjectID(),
			WorkflowKey: saved.ID.Hex(),
			CreatedAt:   time.Now().UTC(),
			Status:      "done",
			Progress:    map[string]ProcessStep{"1_1": {State: "done"}},
		})

		req := httptest.NewRequest(http.MethodGet, "/org-admin/formata-builder/stream/"+saved.ID.Hex(), nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var payload map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("json.Unmarshal error: %v", err)
		}
		if _, ok := payload["editable"]; ok {
			t.Fatalf("editable present in payload: %#v", payload["editable"])
		}
		if _, ok := payload["editableRequiresPurge"]; ok {
			t.Fatalf("editableRequiresPurge present in payload: %#v", payload["editableRequiresPurge"])
		}
	})

	t.Run("stream json rejects invalid id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/org-admin/formata-builder/stream/bad-id", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("stream json returns not found for unknown stream", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/org-admin/formata-builder/stream/"+primitive.NewObjectID().Hex(), nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("stream json returns error for invalid yaml", func(t *testing.T) {
		saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          "workflow: [",
			CreatedByUserID: formataStreamUserID(&orgAdmin),
			UpdatedByUserID: formataStreamUserID(&orgAdmin),
			UpdatedAt:       time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream error: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/org-admin/formata-builder/stream/"+saved.ID.Hex(), nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("stream json returns error when store is missing", func(t *testing.T) {
		serverWithoutStore := &Server{
			authorizer:  fakeAuthorizer{},
			identity:    testIdentityForSessions(time.Now().UTC(), map[string]AccountUser{orgAdminSession: orgAdmin}),
			enforceAuth: true,
			now:         time.Now,
		}

		req := httptest.NewRequest(http.MethodGet, "/org-admin/formata-builder/stream/"+primitive.NewObjectID().Hex(), nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		serverWithoutStore.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}

func TestHandleOrgAdminFormataBuilderPost(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "platform-admin-builder-post@example.com")
	t.Setenv("ADMIN_PASSWORD", "secret")

	store := NewMemoryStore()
	orgID := stableOrgObjectID("builder-post-org")
	orgAdmin := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "builder-post-org-admin",
		OrgID:          &orgID,
		OrgSlug:        "builder-post-org",
		Email:          "org-admin-builder-post@example.com",
		RoleSlugs:      []string{"org-admin"},
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
	}
	orgAdminSession := "session-builder-post-org-admin"
	plain := AccountUser{
		ID:        primitive.NewObjectID(),
		Email:     "plain-builder-post@example.com",
		RoleSlugs: []string{"inspector"},
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}
	server := &Server{
		authorizer: fakeAuthorizer{},
		store:      store,
		identity: testIdentityForSessions(time.Now().UTC(), map[string]AccountUser{
			orgAdminSession: orgAdmin,
			"session-plain": plain,
		}),
		enforceAuth: true,
		now:         time.Now,
	}

	t.Run("unauthenticated is unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/org-admin/formata-builder", strings.NewReader("stream"))
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("forbidden for non admin role", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/org-admin/formata-builder", strings.NewReader("stream"))
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-plain"})
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
		expectedID := orgAdmin.IdentityUserID
		if stream.CreatedByUserID != expectedID {
			t.Fatalf("createdByUserID = %q, want %q", stream.CreatedByUserID, expectedID)
		}
		if stream.UpdatedByUserID != expectedID {
			t.Fatalf("updatedByUserID = %q, want %q", stream.UpdatedByUserID, expectedID)
		}
	})

	t.Run("owner can rewrite editable stream", func(t *testing.T) {
		saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          workflowStreamYAML("Original stream"),
			CreatedByUserID: formataStreamUserID(&orgAdmin),
			UpdatedByUserID: formataStreamUserID(&orgAdmin),
			UpdatedAt:       time.Now().UTC().Add(-time.Hour),
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream error: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/org-admin/formata-builder?stream="+saved.ID.Hex(), strings.NewReader(workflowStreamYAML("Updated stream")))
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
		}

		got, err := store.LoadFormataBuilderStreamByID(t.Context(), saved.ID)
		if err != nil {
			t.Fatalf("LoadFormataBuilderStreamByID error: %v", err)
		}
		if !strings.Contains(got.Stream, "Updated stream") {
			t.Fatalf("stream = %q, want updated yaml", got.Stream)
		}
	})

	t.Run("new=true creates new stream instead of rewriting existing one", func(t *testing.T) {
		saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          workflowStreamYAML("Template stream"),
			CreatedByUserID: formataStreamUserID(&orgAdmin),
			UpdatedByUserID: formataStreamUserID(&orgAdmin),
			UpdatedAt:       time.Now().UTC().Add(-time.Hour),
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream error: %v", err)
		}
		store.SeedProcess(Process{
			ID:          primitive.NewObjectID(),
			WorkflowKey: saved.ID.Hex(),
			CreatedAt:   time.Now().UTC(),
			Status:      "active",
			Progress:    map[string]ProcessStep{},
		})

		req := httptest.NewRequest(http.MethodPost, "/org-admin/formata-builder?stream="+saved.ID.Hex()+"&new=true", strings.NewReader(workflowStreamYAML("Cloned stream")))
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
		}

		original, err := store.LoadFormataBuilderStreamByID(t.Context(), saved.ID)
		if err != nil {
			t.Fatalf("LoadFormataBuilderStreamByID error: %v", err)
		}
		if !strings.Contains(original.Stream, "Template stream") {
			t.Fatalf("original stream = %q, want unchanged yaml", original.Stream)
		}

		streams, err := store.ListFormataBuilderStreams(t.Context())
		if err != nil {
			t.Fatalf("ListFormataBuilderStreams error: %v", err)
		}
		if len(streams) < 2 {
			t.Fatalf("stream count = %d, want at least 2", len(streams))
		}

		var created *FormataBuilderStream
		for i := range streams {
			if streams[i].ID == saved.ID {
				continue
			}
			if strings.Contains(streams[i].Stream, "Cloned stream") {
				created = &streams[i]
				break
			}
		}
		if created == nil {
			t.Fatalf("did not find created stream in %#v", streams)
		}
		if created.CreatedByUserID != formataStreamUserID(&orgAdmin) {
			t.Fatalf("createdByUserID = %q, want %q", created.CreatedByUserID, formataStreamUserID(&orgAdmin))
		}
		if created.ID == saved.ID {
			t.Fatal("expected created stream to have a new id")
		}
	})

	t.Run("rewrite fails when stream has started instances", func(t *testing.T) {
		saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          workflowStreamYAML("Stale stream"),
			CreatedByUserID: formataStreamUserID(&orgAdmin),
			UpdatedByUserID: formataStreamUserID(&orgAdmin),
			UpdatedAt:       time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream error: %v", err)
		}
		store.SeedProcess(Process{
			ID:          primitive.NewObjectID(),
			WorkflowKey: saved.ID.Hex(),
			CreatedAt:   time.Now().UTC(),
			Status:      "active",
			Progress:    map[string]ProcessStep{},
		})

		req := httptest.NewRequest(http.MethodPost, "/org-admin/formata-builder?stream="+saved.ID.Hex(), strings.NewReader(workflowStreamYAML("Should fail")))
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusConflict)
		}

		got, err := store.LoadFormataBuilderStreamByID(t.Context(), saved.ID)
		if err != nil {
			t.Fatalf("LoadFormataBuilderStreamByID error: %v", err)
		}
		if !strings.Contains(got.Stream, "Stale stream") {
			t.Fatalf("stream = %q, want unchanged yaml", got.Stream)
		}
	})

	t.Run("rewrite fails for non owner", func(t *testing.T) {
		saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          workflowStreamYAML("Protected stream"),
			CreatedByUserID: "another-owner",
			UpdatedByUserID: "another-owner",
			UpdatedAt:       time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream error: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/org-admin/formata-builder?stream="+saved.ID.Hex(), strings.NewReader(workflowStreamYAML("Should fail")))
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("platform admin can rewrite started stream and purge workflow data", func(t *testing.T) {
		saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          workflowStreamYAML("Platform stale stream"),
			CreatedByUserID: "another-owner",
			UpdatedByUserID: "another-owner",
			UpdatedAt:       time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream error: %v", err)
		}
		store.SeedProcess(Process{
			ID:          primitive.NewObjectID(),
			WorkflowKey: saved.ID.Hex(),
			CreatedAt:   time.Now().UTC(),
			Status:      "active",
			Progress:    map[string]ProcessStep{},
		})

		req := httptest.NewRequest(http.MethodPost, "/org-admin/formata-builder?stream="+saved.ID.Hex(), strings.NewReader(workflowStreamYAML("Platform updated stream")))
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
		}

		got, err := store.LoadFormataBuilderStreamByID(t.Context(), saved.ID)
		if err != nil {
			t.Fatalf("LoadFormataBuilderStreamByID error: %v", err)
		}
		if !strings.Contains(got.Stream, "Platform updated stream") {
			t.Fatalf("stream = %q, want updated yaml", got.Stream)
		}
		hasProcesses, err := store.HasProcessesByWorkflow(t.Context(), saved.ID.Hex())
		if err != nil {
			t.Fatalf("HasProcessesByWorkflow error: %v", err)
		}
		if hasProcesses {
			t.Fatal("expected workflow data to be purged")
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

	t.Run("rejects invalid stream id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/org-admin/formata-builder?stream=bad-id", strings.NewReader(workflowStreamYAML("Invalid stream id")))
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("rejects missing stream id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/org-admin/formata-builder?stream="+primitive.NewObjectID().Hex(), strings.NewReader(workflowStreamYAML("Missing stream")))
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		server.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("returns error when store is missing", func(t *testing.T) {
		serverWithoutStore := &Server{
			authorizer:  fakeAuthorizer{},
			identity:    testIdentityForSessions(time.Now().UTC(), map[string]AccountUser{orgAdminSession: orgAdmin}),
			enforceAuth: true,
			now:         time.Now,
		}

		req := httptest.NewRequest(http.MethodPost, "/org-admin/formata-builder", strings.NewReader(workflowStreamYAML("No store")))
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		rec := httptest.NewRecorder()
		serverWithoutStore.handleOrgAdminFormataBuilder(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
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

	t.Run("stream yaml converts to json payload", func(t *testing.T) {
		payload, err := formataBuilderStreamJSON(workflowStreamYAML("JSON stream"))
		if err != nil {
			t.Fatalf("formataBuilderStreamJSON error: %v", err)
		}
		root, ok := payload.(map[string]interface{})
		if !ok {
			t.Fatalf("payload type = %T, want map[string]interface{}", payload)
		}
		workflow, ok := root["workflow"].(map[string]interface{})
		if !ok || workflow["name"] != "JSON stream" {
			t.Fatalf("workflow payload = %#v, want stream name", root["workflow"])
		}
	})

	t.Run("invalid stream yaml returns error", func(t *testing.T) {
		if _, err := formataBuilderStreamJSON("workflow: ["); err == nil {
			t.Fatal("expected yaml parse error")
		}
	})

	t.Run("normalize yaml json handles interface keys", func(t *testing.T) {
		got := normalizeYAMLJSONValue(map[interface{}]interface{}{
			"workflow": map[interface{}]interface{}{"name": "Example"},
			1:          []interface{}{map[interface{}]interface{}{"nested": true}},
		})
		root, ok := got.(map[string]interface{})
		if !ok {
			t.Fatalf("normalized root type = %T, want map[string]interface{}", got)
		}
		if _, ok := root["1"]; !ok {
			t.Fatalf("expected numeric key to be stringified, got %#v", root)
		}
	})

	t.Run("memory store updates existing stream", func(t *testing.T) {
		store := NewMemoryStore()
		saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          "before",
			CreatedByUserID: "creator",
			UpdatedByUserID: "creator",
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream error: %v", err)
		}
		updated, err := store.UpdateFormataBuilderStream(t.Context(), FormataBuilderStream{
			ID:              saved.ID,
			Stream:          "after",
			CreatedByUserID: "creator",
			UpdatedByUserID: "updater",
		})
		if err != nil {
			t.Fatalf("UpdateFormataBuilderStream error: %v", err)
		}
		if updated.Stream != "after" {
			t.Fatalf("updated stream = %q, want %q", updated.Stream, "after")
		}
	})

	t.Run("memory store update missing stream returns not found", func(t *testing.T) {
		store := NewMemoryStore()
		if _, err := store.UpdateFormataBuilderStream(t.Context(), FormataBuilderStream{ID: primitive.NewObjectID(), Stream: "x"}); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("UpdateFormataBuilderStream error = %v, want mongo.ErrNoDocuments", err)
		}
	})

	t.Run("memory store update missing id returns not found", func(t *testing.T) {
		store := NewMemoryStore()
		if _, err := store.UpdateFormataBuilderStream(t.Context(), FormataBuilderStream{Stream: "x"}); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("UpdateFormataBuilderStream error = %v, want mongo.ErrNoDocuments", err)
		}
	})

	t.Run("stream edit state is false without user", func(t *testing.T) {
		server := &Server{store: NewMemoryStore()}
		editable, requiresPurge, err := server.formataBuilderStreamEditState(t.Context(), nil, FormataBuilderStream{ID: primitive.NewObjectID()})
		if err != nil {
			t.Fatalf("formataBuilderStreamEditState error: %v", err)
		}
		if editable {
			t.Fatal("expected editable to be false")
		}
		if requiresPurge {
			t.Fatal("expected requiresPurge to be false")
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
