package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestHandleDeleteWorkflow(t *testing.T) {
	now := time.Date(2026, 3, 7, 10, 0, 0, 0, time.UTC)

	t.Run("creator can delete stream with no processes", func(t *testing.T) {
		store := NewMemoryStore()
		user := AccountUser{
			ID:             primitive.NewObjectID(),
			IdentityUserID: "creator-user",
			Email:          "creator@example.com",
			RoleSlugs:      []string{"org-admin"},
			Status:         "active",
		}
		sessionID := "session-creator"
		saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          workflowStreamYAML("Creator stream"),
			CreatedByUserID: user.IdentityUserID,
			UpdatedByUserID: user.IdentityUserID,
			UpdatedAt:       now,
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream: %v", err)
		}

		server := &Server{
			store:       store,
			identity:    workflowDeleteIdentity(now, sessionID, user),
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		if _, err := server.workflowByKey(saved.ID.Hex()); err != nil {
			t.Fatalf("workflowByKey: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/w/"+saved.ID.Hex()+"/delete", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleWorkflowRoutes(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
		}
		if location := rec.Header().Get("Location"); !strings.Contains(location, "confirmation=Creator+stream+was+deleted.") {
			t.Fatalf("location = %q", location)
		}
		if _, err := store.LoadFormataBuilderStreamByID(t.Context(), saved.ID); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("LoadFormataBuilderStreamByID error = %v, want mongo.ErrNoDocuments", err)
		}
	})

	t.Run("creator cannot delete stream after a process started", func(t *testing.T) {
		store := NewMemoryStore()
		user := AccountUser{
			ID:             primitive.NewObjectID(),
			IdentityUserID: "creator-blocked-user",
			Email:          "creator-blocked@example.com",
			RoleSlugs:      []string{"org-admin"},
			Status:         "active",
		}
		sessionID := "session-blocked"
		saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          workflowStreamYAML("Blocked stream"),
			CreatedByUserID: user.IdentityUserID,
			UpdatedByUserID: user.IdentityUserID,
			UpdatedAt:       now,
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream: %v", err)
		}
		store.SeedProcess(Process{
			ID:          primitive.NewObjectID(),
			WorkflowKey: saved.ID.Hex(),
			CreatedAt:   now,
			Status:      "active",
			Progress:    map[string]ProcessStep{},
		})

		server := &Server{
			store:    store,
			identity: workflowDeleteIdentity(now, sessionID, user),
			authorizer: fakeAuthorizer{deleteDecide: func(user *AccountUser, workflowKey string, createdByUserID string, hasProcesses bool) (bool, error) {
				return false, nil
			}},
			enforceAuth: true,
			now:         func() time.Time { return now },
		}

		req := httptest.NewRequest(http.MethodPost, "/w/"+saved.ID.Hex()+"/delete", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleWorkflowRoutes(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
		}
		if location := rec.Header().Get("Location"); !strings.Contains(location, "error=Blocked+stream+cannot+be+deleted+because+one+or+more+processes+have+already+been+started.") {
			t.Fatalf("location = %q", location)
		}
		if _, err := store.LoadFormataBuilderStreamByID(t.Context(), saved.ID); err != nil {
			t.Fatalf("stream should still exist, got error %v", err)
		}
	})

	t.Run("platform admin can delete stream and purge workflow data", func(t *testing.T) {
		t.Setenv("ADMIN_EMAIL", "admin@example.com")
		t.Setenv("ADMIN_PASSWORD", "secret")

		store := NewMemoryStore()
		saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          workflowStreamYAML("Platform stream"),
			CreatedByUserID: "someone-else",
			UpdatedByUserID: "someone-else",
			UpdatedAt:       now,
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream: %v", err)
		}
		processID := store.SeedProcess(Process{
			ID:          primitive.NewObjectID(),
			WorkflowKey: saved.ID.Hex(),
			CreatedAt:   now,
			Status:      "done",
			Progress: map[string]ProcessStep{
				"1_1": {State: "done"},
			},
		})
		if err := store.InsertNotarization(t.Context(), Notarization{
			ProcessID: processID,
			SubstepID: "1.1",
			CreatedAt: now,
		}); err != nil {
			t.Fatalf("InsertNotarization: %v", err)
		}
		if _, err := store.SaveAttachment(t.Context(), AttachmentUpload{
			ProcessID:  processID,
			SubstepID:  "1.1",
			Filename:   "evidence.txt",
			UploadedAt: now,
		}, bytes.NewBufferString("evidence")); err != nil {
			t.Fatalf("SaveAttachment: %v", err)
		}

		server := &Server{
			store:       store,
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         func() time.Time { return now },
		}

		req := httptest.NewRequest(http.MethodPost, "/w/"+saved.ID.Hex()+"/delete", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()
		server.handleWorkflowRoutes(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
		}
		if location := rec.Header().Get("Location"); !strings.Contains(location, "confirmation=Platform+stream+was+deleted.") {
			t.Fatalf("location = %q", location)
		}
		if _, err := store.LoadFormataBuilderStreamByID(t.Context(), saved.ID); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("stream should be deleted, got %v", err)
		}
		if _, ok := store.SnapshotProcess(processID); ok {
			t.Fatal("expected workflow processes to be purged")
		}
		if len(store.Notarizations()) != 0 {
			t.Fatalf("expected notarizations to be purged, got %d", len(store.Notarizations()))
		}
		store.mu.RLock()
		attachmentCount := len(store.attachments)
		store.mu.RUnlock()
		if attachmentCount != 0 {
			t.Fatalf("expected attachments to be purged, got %d", attachmentCount)
		}
	})

	t.Run("cerbos error returns bad gateway", func(t *testing.T) {
		store := NewMemoryStore()
		user := AccountUser{
			ID:             primitive.NewObjectID(),
			IdentityUserID: "creator-error-user",
			Email:          "creator-error@example.com",
			RoleSlugs:      []string{"org-admin"},
			Status:         "active",
		}
		sessionID := "session-creator-error"
		saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          workflowStreamYAML("Error stream"),
			CreatedByUserID: user.IdentityUserID,
			UpdatedByUserID: user.IdentityUserID,
			UpdatedAt:       now,
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream: %v", err)
		}

		server := &Server{
			store:    store,
			identity: workflowDeleteIdentity(now, sessionID, user),
			authorizer: fakeAuthorizer{deleteDecide: func(user *AccountUser, workflowKey string, createdByUserID string, hasProcesses bool) (bool, error) {
				return false, errors.New("cerbos down")
			}},
			enforceAuth: true,
			now:         func() time.Time { return now },
		}

		req := httptest.NewRequest(http.MethodPost, "/w/"+saved.ID.Hex()+"/delete", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleWorkflowRoutes(rec, req)

		if rec.Code != http.StatusBadGateway {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadGateway)
		}
	})

	t.Run("creator can delete stream even when workflow refs are invalid", func(t *testing.T) {
		store := NewMemoryStore()
		user := AccountUser{
			ID:             primitive.NewObjectID(),
			IdentityUserID: "creator-invalid-user",
			Email:          "creator-invalid@example.com",
			RoleSlugs:      []string{"org-admin"},
			Status:         "active",
		}
		sessionID := "session-invalid"
		saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          workflowStreamYAML("Invalid refs stream"),
			CreatedByUserID: user.IdentityUserID,
			UpdatedByUserID: user.IdentityUserID,
			UpdatedAt:       now,
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream: %v", err)
		}

		server := &Server{
			store: store,
			identity: &fakeIdentityStore{
				getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
					return IdentitySession{Secret: sessionSecret, ExpiresAt: now.Add(time.Hour)}, nil
				},
				getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
					return IdentityUser{ID: user.IdentityUserID, Email: user.Email, IsOrgAdmin: true}, nil
				},
				listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
					return nil, nil
				},
			},
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         func() time.Time { return now },
		}

		req := httptest.NewRequest(http.MethodPost, "/w/"+saved.ID.Hex()+"/delete", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleWorkflowRoutes(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
		}
		if location := rec.Header().Get("Location"); !strings.Contains(location, "confirmation=Invalid+refs+stream+was+deleted.") {
			t.Fatalf("location = %q", location)
		}
		if _, err := store.LoadFormataBuilderStreamByID(t.Context(), saved.ID); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("LoadFormataBuilderStreamByID error = %v, want mongo.ErrNoDocuments", err)
		}
	})
}

func workflowStreamYAML(name string) string {
	return "workflow:\n" +
		"  name: \"" + name + "\"\n" +
		"  description: \"demo\"\n" +
		"  steps:\n" +
		"    - id: \"1\"\n" +
		"      title: \"Step 1\"\n" +
		"      order: 1\n" +
		"      organization: \"org1\"\n" +
		"      substeps:\n" +
		"        - id: \"1.1\"\n" +
		"          title: \"Input\"\n" +
		"          order: 1\n" +
		"          roles: [\"dep1\"]\n" +
		"          inputKey: \"value\"\n" +
		"          inputType: \"string\"\n" +
		"organizations:\n" +
		"  - slug: \"org1\"\n" +
		"    name: \"Org\"\n" +
		"roles:\n" +
		"  - orgSlug: \"org1\"\n" +
		"    slug: \"dep1\"\n" +
		"    name: \"Dep\"\n" +
		"users:\n" +
		"  - id: \"u1\"\n" +
		"    name: \"User 1\"\n" +
		"    departmentId: \"dep1\"\n"
}

func workflowDeleteIdentity(now time.Time, sessionID string, user AccountUser) *fakeIdentityStore {
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	identity.listOrganizationsFunc = func(ctx context.Context) ([]IdentityOrg, error) {
		return []IdentityOrg{
			{
				ID:   "org-1",
				Slug: "org1",
				Name: "Org",
				Roles: []IdentityRole{
					{Slug: "dep1", Name: "Dep"},
				},
			},
		}, nil
	}
	return identity
}
