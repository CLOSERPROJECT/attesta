package main

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleHomeListsProcesses(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)

	activeID := primitive.NewObjectID()
	active := Process{
		ID:          activeID,
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-2 * time.Hour),
		Status:      "",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(now.Add(-110 * time.Minute))},
			"1_2": {State: "done", DoneAt: ptrTime(now.Add(-100 * time.Minute)), Data: map[string]interface{}{"note": "alpha"}},
			"1_3": {State: "pending"},
			"2_1": {State: "pending"},
			"2_2": {State: "pending"},
			"3_1": {State: "pending"},
			"3_2": {State: "pending"},
		},
	}

	doneID := primitive.NewObjectID()
	done := Process{
		ID:          doneID,
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-1 * time.Hour),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(now.Add(-70 * time.Minute))},
			"1_2": {State: "done", DoneAt: ptrTime(now.Add(-60 * time.Minute))},
			"1_3": {State: "done", DoneAt: ptrTime(now.Add(-50 * time.Minute))},
			"2_1": {State: "done", DoneAt: ptrTime(now.Add(-40 * time.Minute))},
			"2_2": {State: "done", DoneAt: ptrTime(now.Add(-30 * time.Minute))},
			"3_1": {State: "done", DoneAt: ptrTime(now.Add(-20 * time.Minute))},
			"3_2": {State: "done", DoneAt: ptrTime(now.Add(-10 * time.Minute))},
		},
	}

	store.SeedProcess(active)
	store.SeedProcess(done)

	server := &Server{
		authorizer: fakeAuthorizer{},
		store:      store,
		tmpl:       homeTestTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			cfg := testRuntimeConfig()
			cfg.Workflow.Description = "Demo workflow description"
			return cfg, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/w/workflow/", nil)
	cfg := testRuntimeConfig()
	cfg.Workflow.Description = "Demo workflow description"
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: cfg,
	}))
	rec := httptest.NewRecorder()
	server.handleWorkflowHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "PROC 2 SORT time_desc FILTER all") {
		t.Fatalf("expected processes count and default controls, got %q", body)
	}
	if !strings.Contains(body, activeID.Hex()+":active:28") {
		t.Fatalf("expected active process stats, got %q", body)
	}
	if !strings.Contains(body, doneID.Hex()+":done:100") {
		t.Fatalf("expected done process stats, got %q", body)
	}
	if !strings.Contains(body, "SORT time_desc") {
		t.Fatalf("expected default sort, got %q", body)
	}
	if !strings.Contains(body, "DESC Demo workflow description") {
		t.Fatalf("expected workflow description, got %q", body)
	}
}

func TestHandleHomeFiltersProcessesByStatus(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)

	activeID := primitive.NewObjectID()
	store.SeedProcess(Process{
		ID:          activeID,
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-2 * time.Hour),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(now.Add(-110 * time.Minute))},
		},
	})

	doneID := primitive.NewObjectID()
	store.SeedProcess(Process{
		ID:          doneID,
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-1 * time.Hour),
		Status:      "done",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(now.Add(-70 * time.Minute))},
			"1_2": {State: "done", DoneAt: ptrTime(now.Add(-60 * time.Minute))},
			"1_3": {State: "done", DoneAt: ptrTime(now.Add(-50 * time.Minute))},
			"2_1": {State: "done", DoneAt: ptrTime(now.Add(-40 * time.Minute))},
			"2_2": {State: "done", DoneAt: ptrTime(now.Add(-30 * time.Minute))},
			"3_1": {State: "done", DoneAt: ptrTime(now.Add(-20 * time.Minute))},
			"3_2": {State: "done", DoneAt: ptrTime(now.Add(-10 * time.Minute))},
		},
	})

	server := &Server{
		authorizer: fakeAuthorizer{},
		store:      store,
		tmpl:       homeTestTemplates(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/w/workflow/?filter=done", nil)
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	rec := httptest.NewRecorder()
	server.handleWorkflowHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "PROC 1 SORT time_desc FILTER done") {
		t.Fatalf("expected done filter selection, got %q", body)
	}
	if !strings.Contains(body, doneID.Hex()+":done:100") {
		t.Fatalf("expected done process in filtered list, got %q", body)
	}
	if strings.Contains(body, activeID.Hex()+":active") {
		t.Fatalf("did not expect active process in done filter, got %q", body)
	}
}

func TestHandleHomeRendersWorkflowPicker(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, tempDir+"/workflow.yaml", "Main workflow", "string", "Main workflow description")
	writeWorkflowConfig(t, tempDir+"/secondary.yaml", "Secondary workflow", "number")

	server := &Server{
		authorizer: fakeAuthorizer{},
		tmpl:       homePickerTemplates(),
		configDir:  tempDir,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.handleHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "PICK 2") {
		t.Fatalf("expected picker marker, got %q", body)
	}
	if !strings.Contains(body, "workflow:Main workflow:Main workflow description") || !strings.Contains(body, "secondary:Secondary workflow") {
		t.Fatalf("expected workflow options in picker, got %q", body)
	}
	if strings.Contains(body, "secondary:Secondary workflow:Secondary workflow description:") {
		t.Fatalf("expected optional description to be omitted when empty, got %q", body)
	}
}

func TestHandleHomePickerRendersWorkflowCardsAndScopedLinks(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string", "Main workflow description")
	writeWorkflowConfig(t, filepath.Join(tempDir, "secondary.yaml"), "Secondary workflow", "number")

	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))
	server := &Server{
		authorizer: fakeAuthorizer{},
		tmpl:       tmpl,
		configDir:  tempDir,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.handleHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `class="panel max-w-7xl mx-auto"`) {
		t.Fatalf("expected home picker panel wrapper, got %q", body)
	}
	if !strings.Contains(body, `class="workflow-grid"`) || !strings.Contains(body, `class="workflow-card"`) {
		t.Fatalf("expected workflow card grid markup, got %q", body)
	}
	if !strings.Contains(body, `href="/w/workflow/"`) {
		t.Fatalf("expected scoped workflow href for workflow key, got %q", body)
	}
	if !strings.Contains(body, `href="/w/secondary/"`) {
		t.Fatalf("expected scoped workflow href for secondary key, got %q", body)
	}
	if !strings.Contains(body, "Main workflow description") {
		t.Fatalf("expected description content in cards, got %q", body)
	}
	if !strings.Contains(body, "Not started") || !strings.Contains(body, "In progress") || !strings.Contains(body, "Completed") {
		t.Fatalf("expected status labels in cards, got %q", body)
	}
}

func TestHandleHomePickerCreateStreamCardVisibility(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string", "Main workflow description")

	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))

	t.Run("visible for org admin", func(t *testing.T) {
		store := NewMemoryStore()
		user := AccountUser{
			ID:        primitive.NewObjectID(),
			Email:     "org-admin-picker@example.com",
			RoleSlugs: []string{"org-admin"},
			Status:    "active",
			CreatedAt: time.Now().UTC(),
		}
		sessionID := "session-org-admin"

		server := &Server{
			authorizer:  fakeAuthorizer{},
			store:       store,
			identity:    testIdentityForSessions(time.Now().UTC(), map[string]AccountUser{sessionID: user}),
			tmpl:        tmpl,
			configDir:   tempDir,
			enforceAuth: true,
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleHome(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if !strings.Contains(body, `href="/org-admin/formata-builder"`) {
			t.Fatalf("expected create stream card for org admin, got %q", body)
		}
		if !strings.Contains(body, "workflow-card-cta") || !strings.Contains(body, "Create new stream") {
			t.Fatalf("expected create stream card cta for org admin, got %q", body)
		}
	})

	t.Run("hidden for non org admin", func(t *testing.T) {
		store := NewMemoryStore()
		user := AccountUser{
			ID:        primitive.NewObjectID(),
			Email:     "member-picker@example.com",
			RoleSlugs: []string{"operator"},
			Status:    "active",
			CreatedAt: time.Now().UTC(),
		}
		sessionID := "session-member"

		server := &Server{
			authorizer:  fakeAuthorizer{},
			store:       store,
			identity:    testIdentityForSessions(time.Now().UTC(), map[string]AccountUser{sessionID: user}),
			tmpl:        tmpl,
			configDir:   tempDir,
			enforceAuth: true,
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleHome(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if strings.Contains(rec.Body.String(), `href="/org-admin/formata-builder"`) {
			t.Fatalf("did not expect create stream card for non org admin, got %q", rec.Body.String())
		}
	})

}

func TestHandleWorkflowHomeRendersValidationState(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))
	cfg := RuntimeConfig{
		Organizations: []WorkflowOrganization{
			{Slug: "org1", Name: "Organization 1"},
		},
		Roles: []WorkflowRole{
			{OrgSlug: "org1", Slug: "dep1", Name: "Department 1"},
		},
		Workflow: WorkflowDef{
			Name: "Workflow with missing refs",
			Steps: []WorkflowStep{
				{
					StepID:           "1",
					Title:            "Step 1",
					Order:            1,
					OrganizationSlug: "org1",
					Substep: []WorkflowSub{
						{SubstepID: "1.1", Title: "Sub 1", Order: 1, Roles: []string{"dep1"}, InputKey: "value", InputType: "string"},
					},
				},
			},
		},
	}

	server := &Server{
		authorizer: fakeAuthorizer{},
		store:      NewMemoryStore(),
		tmpl:       tmpl,
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return IdentitySession{Secret: sessionSecret, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "user@example.com"}, nil
			},
			listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
				return nil, nil
			},
		},
		enforceAuth: true,
	}

	req := httptest.NewRequest(http.MethodGet, "/w/workflow/", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: cfg,
	}))
	rec := httptest.NewRecorder()
	server.handleWorkflowHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	compactBody := strings.Join(strings.Fields(body), " ")
	if !strings.Contains(body, "Stream configuration issue") {
		t.Fatalf("expected validation panel heading, got %q", body)
	}
	if !strings.Contains(body, "workflow references are invalid") {
		t.Fatalf("expected validation error details, got %q", body)
	}
	if !strings.Contains(body, `action="/w/workflow/process/start"`) {
		t.Fatalf("expected new stream form to remain present, got %q", body)
	}
	if !strings.Contains(compactBody, `class="primary"`) || !strings.Contains(compactBody, `type="submit"`) || !strings.Contains(compactBody, `disabled`) || !strings.Contains(compactBody, `New instance`) {
		t.Fatalf("expected new stream button to be disabled for invalid workflow, got %q", body)
	}
}

func TestHandleHomePickerDeleteButtonVisibility(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))
	now := time.Date(2026, 3, 7, 11, 0, 0, 0, time.UTC)

	t.Run("visible for creator before any process starts", func(t *testing.T) {
		store := NewMemoryStore()
		user := AccountUser{
			ID:             primitive.NewObjectID(),
			IdentityUserID: "creator-home-user",
			Email:          "creator-home@example.com",
			RoleSlugs:      []string{"org-admin"},
			Status:         "active",
		}
		sessionID := "session-home-creator"
		stream, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          workflowStreamYAML("Delete from home"),
			CreatedByUserID: user.IdentityUserID,
			UpdatedByUserID: user.IdentityUserID,
			UpdatedAt:       now,
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream: %v", err)
		}

		server := &Server{
			store:       store,
			identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
			authorizer:  fakeAuthorizer{deleteDecide: workflowDeleteDecision},
			tmpl:        tmpl,
			enforceAuth: true,
			now:         func() time.Time { return now },
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleHome(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if !strings.Contains(rec.Body.String(), `action="/w/`+stream.ID.Hex()+`/delete"`) {
			t.Fatalf("expected delete button for creator, got %q", rec.Body.String())
		}
	})

	t.Run("hidden for creator after process start", func(t *testing.T) {
		store := NewMemoryStore()
		user := AccountUser{
			ID:             primitive.NewObjectID(),
			IdentityUserID: "creator-home-started-user",
			Email:          "creator-home-started@example.com",
			RoleSlugs:      []string{"org-admin"},
			Status:         "active",
		}
		sessionID := "session-home-started"
		stream, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          workflowStreamYAML("Started stream"),
			CreatedByUserID: user.IdentityUserID,
			UpdatedByUserID: user.IdentityUserID,
			UpdatedAt:       now,
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream: %v", err)
		}
		store.SeedProcess(Process{
			ID:          primitive.NewObjectID(),
			WorkflowKey: stream.ID.Hex(),
			CreatedAt:   now,
			Status:      "active",
			Progress:    map[string]ProcessStep{},
		})

		server := &Server{
			store:       store,
			identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
			authorizer:  fakeAuthorizer{deleteDecide: workflowDeleteDecision},
			tmpl:        tmpl,
			enforceAuth: true,
			now:         func() time.Time { return now },
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleHome(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if strings.Contains(rec.Body.String(), `action="/w/`+stream.ID.Hex()+`/delete"`) {
			t.Fatalf("did not expect delete button for started stream, got %q", rec.Body.String())
		}
	})

	t.Run("visible for platform admin even with processes", func(t *testing.T) {
		t.Setenv("ADMIN_EMAIL", "admin@example.com")
		t.Setenv("ADMIN_PASSWORD", "secret")

		store := NewMemoryStore()
		stream, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:          workflowStreamYAML("Platform delete"),
			CreatedByUserID: "someone-else",
			UpdatedByUserID: "someone-else",
			UpdatedAt:       now,
		})
		if err != nil {
			t.Fatalf("SaveFormataBuilderStream: %v", err)
		}
		store.SeedProcess(Process{
			ID:          primitive.NewObjectID(),
			WorkflowKey: stream.ID.Hex(),
			CreatedAt:   now,
			Status:      "done",
			Progress: map[string]ProcessStep{
				"1_1": {State: "done"},
			},
		})

		server := &Server{
			store:       store,
			authorizer:  fakeAuthorizer{deleteDecide: workflowDeleteDecision},
			tmpl:        tmpl,
			enforceAuth: true,
			now:         func() time.Time { return now },
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()
		server.handleHome(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if !strings.Contains(rec.Body.String(), `action="/w/`+stream.ID.Hex()+`/delete"`) {
			t.Fatalf("expected delete button for platform admin, got %q", rec.Body.String())
		}
	})
}

func TestHandleHomeRendersWorkflowPickerCountsByWorkflow(t *testing.T) {
	tempDir := t.TempDir()
	writeTwoSubstepWorkflowConfig(t, tempDir+"/workflow.yaml", "Main workflow")
	writeTwoSubstepWorkflowConfig(t, tempDir+"/secondary.yaml", "Secondary workflow")

	now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-6 * time.Hour),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "pending"},
			"1_2": {State: "pending"},
		},
	})
	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-5 * time.Hour),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(now.Add(-4 * time.Hour))},
			"1_2": {State: "pending"},
		},
	})
	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   now.Add(-4 * time.Hour),
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(now.Add(-3 * time.Hour))},
			"1_2": {State: "done", DoneAt: ptrTime(now.Add(-2 * time.Hour))},
		},
	})
	store.SeedProcess(Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: now.Add(-3 * time.Hour),
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "pending"},
			"1_2": {State: "pending"},
		},
	})
	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "secondary",
		CreatedAt:   now.Add(-2 * time.Hour),
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(now.Add(-90 * time.Minute))},
			"1_2": {State: "pending"},
		},
	})
	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "secondary",
		CreatedAt:   now.Add(-1 * time.Hour),
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(now.Add(-50 * time.Minute))},
			"1_2": {State: "done", DoneAt: ptrTime(now.Add(-40 * time.Minute))},
		},
	})

	server := &Server{
		authorizer: fakeAuthorizer{},
		tmpl:       homePickerTemplates(),
		configDir:  tempDir,
		store:      store,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.handleHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "workflow:Main workflow:2/1/1") {
		t.Fatalf("expected workflow counts 2/1/1, got %q", body)
	}
	if !strings.Contains(body, "secondary:Secondary workflow:0/1/1") {
		t.Fatalf("expected secondary counts 0/1/1, got %q", body)
	}
}

func TestNormalizeHomeSortKey(t *testing.T) {
	if got := normalizeHomeSortKey("status"); got != "status" {
		t.Fatalf("expected status, got %q", got)
	}
	if got := normalizeHomeSortKey("unknown"); got != "time_desc" {
		t.Fatalf("expected time_desc for unknown, got %q", got)
	}
}

func TestNormalizeHomeStatusFilter(t *testing.T) {
	if got := normalizeHomeStatusFilter("done"); got != "done" {
		t.Fatalf("expected done, got %q", got)
	}
	if got := normalizeHomeStatusFilter("ACTIVE"); got != "active" {
		t.Fatalf("expected active, got %q", got)
	}
	if got := normalizeHomeStatusFilter("unknown"); got != "all" {
		t.Fatalf("expected all for unknown, got %q", got)
	}
}

func TestSortHomeProcessListByStatus(t *testing.T) {
	items := []ProcessListItem{
		{ID: "a", Status: "done", Percent: 100, CreatedAtTime: time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)},
		{ID: "b", Status: "active", Percent: 10, CreatedAtTime: time.Date(2026, 2, 3, 11, 0, 0, 0, time.UTC)},
	}
	sortHomeProcessList(items, "status")
	if items[0].Status != "active" {
		t.Fatalf("expected active first, got %q", items[0].Status)
	}
}

func TestHandleHomeErrorPaths(t *testing.T) {
	t.Run("workflow options error", func(t *testing.T) {
		server := &Server{
			authorizer: fakeAuthorizer{},
			tmpl:       homePickerTemplates(),
			configDir:  t.TempDir(),
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		server.handleHome(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("template error", func(t *testing.T) {
		tempDir := t.TempDir()
		writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")

		server := &Server{
			authorizer: fakeAuthorizer{},
			tmpl:       template.Must(template.New("broken").Parse(`{{define "other"}}x{{end}}`)),
			configDir:  tempDir,
			configProvider: func() (RuntimeConfig, error) {
				return RuntimeConfig{}, errors.New("not used")
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		server.handleHome(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}

func TestHandleWorkflowHomeErrorPaths(t *testing.T) {
	t.Run("selected workflow error", func(t *testing.T) {
		server := &Server{
			authorizer: fakeAuthorizer{},
			tmpl:       testTemplates(),
			configProvider: func() (RuntimeConfig, error) {
				return RuntimeConfig{}, errors.New("config down")
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/w/workflow/", nil)
		rec := httptest.NewRecorder()
		server.handleWorkflowHome(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("template error", func(t *testing.T) {
		server := &Server{
			authorizer: fakeAuthorizer{},
			store:      NewMemoryStore(),
			tmpl:       template.Must(template.New("broken").Parse(`{{define "other"}}x{{end}}`)),
			configProvider: func() (RuntimeConfig, error) {
				return testRuntimeConfig(), nil
			},
		}
		req := httptest.NewRequest(http.MethodGet, "/w/workflow/", nil)
		req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
			Key: "workflow",
			Cfg: testRuntimeConfig(),
		}))

		rec := httptest.NewRecorder()
		server.handleWorkflowHome(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}

func TestHandleWorkflowHomeUsesHumanReadableProcessDates(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))
	store := NewMemoryStore()
	createdAt := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	doneAt := time.Date(2026, 2, 3, 11, 30, 0, 0, time.UTC)

	store.SeedProcess(Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		CreatedAt:   createdAt,
		Status:      "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", DoneAt: ptrTime(doneAt), Data: map[string]interface{}{"note": "alpha"}},
		},
	})

	server := &Server{
		authorizer: fakeAuthorizer{},
		store:      store,
		tmpl:       tmpl,
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/w/workflow/", nil)
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	rec := httptest.NewRecorder()
	server.handleWorkflowHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Created: 3 Feb 2026 at 10:00 UTC") {
		t.Fatalf("expected human readable created date, got %q", body)
	}
	if !strings.Contains(body, "Last notarized: 3 Feb 2026 at 11:30 UTC") {
		t.Fatalf("expected human readable last notarized date, got %q", body)
	}
}

func homeTestTemplates() *template.Template {
	return template.Must(template.New("test").Parse(`
{{define "layout.html"}}{{template "home_body" .}}{{end}}
{{define "home_body"}}
PROC {{len .Processes}} SORT {{.Sort}} FILTER {{.StatusFilter}} DESC {{.WorkflowDescription}}
PROCESSES {{range .Processes}}{{.ID}}:{{.Status}}:{{.Percent}}|{{end}}
{{end}}
{{define "home.html"}}{{template "layout.html" .}}{{end}}
`))
}

func homePickerTemplates() *template.Template {
	return template.Must(template.New("test").Parse(`
{{define "layout.html"}}{{template "home_picker_body" .}}{{end}}
{{define "home_picker_body"}}PICK {{len .Workflows}} {{range .Workflows}}{{.Key}}:{{.Name}}{{if .Description}}:{{.Description}}{{end}}:{{.Counts.NotStarted}}/{{.Counts.Started}}/{{.Counts.Terminated}}|{{end}}{{end}}
{{define "home.html"}}{{template "layout.html" .}}{{end}}
`))
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func workflowDeleteDecision(user *AccountUser, workflowKey string, createdByUserID string, hasProcesses bool) (bool, error) {
	if user == nil {
		return false, nil
	}
	if user.IsPlatformAdmin {
		return true, nil
	}
	return !hasProcesses && formataStreamUserID(user) == createdByUserID, nil
}
