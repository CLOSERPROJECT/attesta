package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestListUpcomingStreamInstancesOrgInvolvedWaitingOnOther(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	tempDir := t.TempDir()
	writeTwoOrgWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"))

	store := NewMemoryStore()
	// Step 1 done → next available is org2 / dep2.
	processID := store.SeedProcess(Process{
		WorkflowKey: "workflow",
		Name:        "Cross-org batch",
		Status:      processStatusActive,
		CreatedAt:   now,
		Progress: map[string]ProcessStep{
			"1.1": {State: "done", DoneAt: &now},
			"2.1": {State: "pending"},
		},
	})

	server := &Server{
		identity:  &fakeIdentityStore{},
		store:     store,
		configDir: tempDir,
		now:       func() time.Time { return now },
	}

	org1Member := &AccountUser{
		IdentityUserID: "member-1",
		Email:          "member@org1.example",
		OrgSlug:        "org1",
		RoleSlugs:      []string{"dep1"},
		Status:         "active",
	}
	items, err := server.listUpcomingStreamInstances(context.Background(), org1Member)
	if err != nil {
		t.Fatalf("listUpcomingStreamInstances: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Upcoming = %+v, want 1 item", items)
	}
	got := items[0]
	if got.ProcessID != processID.Hex() {
		t.Fatalf("ProcessID = %q, want %q", got.ProcessID, processID.Hex())
	}
	if got.InstanceName != "Cross-org batch" {
		t.Fatalf("InstanceName = %q", got.InstanceName)
	}
	if got.WorkflowName != "Two-org workflow" {
		t.Fatalf("WorkflowName = %q", got.WorkflowName)
	}
	if got.WaitingOn != "Waiting on Organization 2 · Department 2" {
		t.Fatalf("WaitingOn = %q, want waiting on org2 role", got.WaitingOn)
	}
	wantHref := streamInstanceSubstepPath("workflow", processID.Hex(), "2.1")
	if got.Href != wantHref {
		t.Fatalf("Href = %q, want %q", got.Href, wantHref)
	}
}

func TestListUpcomingStreamInstancesExcludesYourTurn(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")
	store := NewMemoryStore()
	store.SeedProcess(Process{
		WorkflowKey: "workflow",
		Name:        "My turn batch",
		Status:      processStatusActive,
		CreatedAt:   now,
		Progress:    map[string]ProcessStep{"1.1": {State: "pending"}},
	})
	server := &Server{
		identity:  &fakeIdentityStore{},
		store:     store,
		configDir: tempDir,
		now:       func() time.Time { return now },
	}
	items, err := server.listUpcomingStreamInstances(context.Background(), &AccountUser{
		IdentityUserID: "member-1",
		OrgSlug:        "org1",
		RoleSlugs:      []string{"dep1"},
		Status:         "active",
	})
	if err != nil {
		t.Fatalf("listUpcomingStreamInstances: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("your-turn must not appear in Upcoming, got %+v", items)
	}
}

func TestListUpcomingStreamInstancesExcludesPlatformAdminAndUnaffiliated(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	tempDir := t.TempDir()
	writeTwoOrgWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"))
	store := NewMemoryStore()
	store.SeedProcess(Process{
		WorkflowKey: "workflow",
		Name:        "Cross-org batch",
		Status:      processStatusActive,
		CreatedAt:   now,
		Progress: map[string]ProcessStep{
			"1.1": {State: "done", DoneAt: &now},
			"2.1": {State: "pending"},
		},
	})
	server := &Server{
		identity:  &fakeIdentityStore{},
		store:     store,
		configDir: tempDir,
		now:       func() time.Time { return now },
	}

	paItems, err := server.listUpcomingStreamInstances(context.Background(), &AccountUser{
		Email: "admin@example.com", IsPlatformAdmin: true, OrgSlug: "org1", RoleSlugs: []string{"dep1"},
	})
	if err != nil || len(paItems) != 0 {
		t.Fatalf("PA Upcoming = %+v err=%v", paItems, err)
	}

	unaffiliated, err := server.listUpcomingStreamInstances(context.Background(), &AccountUser{
		IdentityUserID: "orphan-1", RoleSlugs: []string{"dep1"}, Status: "active",
	})
	if err != nil || len(unaffiliated) != 0 {
		t.Fatalf("unaffiliated Upcoming = %+v err=%v", unaffiliated, err)
	}
}

func TestListUpcomingStreamInstancesSameOrgDifferentRole(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")
	store := NewMemoryStore()
	store.SeedProcess(Process{
		WorkflowKey: "workflow",
		Name:        "Waiting on dep1",
		Status:      processStatusActive,
		CreatedAt:   now,
		Progress:    map[string]ProcessStep{"1.1": {State: "pending"}},
	})
	server := &Server{
		identity:  &fakeIdentityStore{},
		store:     store,
		configDir: tempDir,
		now:       func() time.Time { return now },
	}
	// Org admin standing → stream access without dep1; next Substep is not theirs.
	items, err := server.listUpcomingStreamInstances(context.Background(), &AccountUser{
		IdentityUserID: "admin-1",
		OrgSlug:        "org1",
		RoleSlugs:      []string{"org-admin"},
		Status:         "active",
	})
	if err != nil {
		t.Fatalf("listUpcomingStreamInstances: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("org-admin without dep1 should see Upcoming, got %+v", items)
	}
	if items[0].WaitingOn != "Waiting on Department 1" {
		t.Fatalf("WaitingOn = %q, want same-org role copy", items[0].WaitingOn)
	}
}

func TestOperatorHomeUpcomingBandBelowYourTurnWithoutDots(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-upcoming"
	tempDir := t.TempDir()
	writeTwoOrgWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"))

	store := NewMemoryStore()
	yourTurnID := store.SeedProcess(Process{
		WorkflowKey: "workflow",
		Name:        "My turn batch",
		Status:      processStatusActive,
		CreatedAt:   now,
		Progress: map[string]ProcessStep{
			"1.1": {State: "pending"},
			"2.1": {State: "pending"},
		},
	})
	upcomingID := store.SeedProcess(Process{
		WorkflowKey: "workflow",
		Name:        "Waiting on org2",
		Status:      processStatusActive,
		CreatedAt:   now.Add(-time.Hour),
		Progress: map[string]ProcessStep{
			"1.1": {State: "done", DoneAt: &now},
			"2.1": {State: "pending"},
		},
	})

	user := AccountUser{
		IdentityUserID: "member-1",
		Email:          "member@org1.example",
		OrgSlug:        "org1",
		RoleSlugs:      []string{"dep1"},
		Status:         "active",
	}
	server := &Server{
		identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
		store:       store,
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		configDir:   tempDir,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/my", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleHome(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("home status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	yourTurnHref := streamInstanceSubstepPath("workflow", yourTurnID.Hex(), "1.1")
	upcomingHref := streamInstanceSubstepPath("workflow", upcomingID.Hex(), "2.1")
	for _, want := range []string{
		"Your turn",
		"My turn batch",
		yourTurnHref,
		`aria-label="Upcoming"`,
		"Waiting on org2",
		"Waiting on Organization 2 · Department 2",
		upcomingHref,
		`attention-dot`,
		`Dashboard (needs attention)`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q on Operator home, got:\n%s", want, body)
		}
	}
	yourTurnIdx := strings.Index(body, "Your turn")
	upcomingIdx := strings.Index(body, `aria-label="Upcoming"`)
	chooseIdx := strings.Index(body, "Choose a stream")
	if yourTurnIdx < 0 || upcomingIdx < 0 || chooseIdx < 0 || !(yourTurnIdx < upcomingIdx && upcomingIdx < chooseIdx) {
		t.Fatalf("expected Your turn above Upcoming above Choose; yourTurn=%d upcoming=%d choose=%d", yourTurnIdx, upcomingIdx, chooseIdx)
	}
}

func TestOperatorHomeUpcomingAloneDoesNotLightAttentionDots(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-upcoming-only"
	tempDir := t.TempDir()
	writeTwoOrgWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"))
	store := NewMemoryStore()
	store.SeedProcess(Process{
		WorkflowKey: "workflow",
		Name:        "Waiting on org2",
		Status:      processStatusActive,
		CreatedAt:   now,
		Progress: map[string]ProcessStep{
			"1.1": {State: "done", DoneAt: &now},
			"2.1": {State: "pending"},
		},
	})
	user := AccountUser{
		IdentityUserID: "member-1",
		Email:          "member@org1.example",
		OrgSlug:        "org1",
		RoleSlugs:      []string{"dep1"},
		Status:         "active",
	}
	server := &Server{
		identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
		store:       store,
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		configDir:   tempDir,
		now:         func() time.Time { return now },
	}
	req := httptest.NewRequest(http.MethodGet, "/my", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleHome(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `aria-label="Upcoming"`) || !strings.Contains(body, "Waiting on org2") {
		t.Fatalf("expected Upcoming band, got:\n%s", body)
	}
	if strings.Contains(body, "Your turn") {
		t.Fatalf("must not show Your turn when only Upcoming, got:\n%s", body)
	}
	if strings.Contains(body, `attention-dot`) || strings.Contains(body, "needs attention") {
		t.Fatalf("Upcoming alone must not light Attention dots, got:\n%s", body)
	}
}

func TestOperatorHomeUpcomingEmptyOmitted(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-no-upcoming"
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")
	user := AccountUser{
		IdentityUserID: "member-1",
		Email:          "member@org1.example",
		OrgSlug:        "org1",
		RoleSlugs:      []string{"dep1"},
		Status:         "active",
	}
	server := &Server{
		identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		configDir:   tempDir,
		now:         func() time.Time { return now },
	}
	req := httptest.NewRequest(http.MethodGet, "/my", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleHome(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, `aria-label="Upcoming"`) || strings.Contains(body, ">Upcoming<") {
		t.Fatalf("empty Upcoming band must be omitted, got:\n%s", body)
	}
}

func TestPlatformAdminHomeOmitsUpcoming(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	tempDir := t.TempDir()
	writeTwoOrgWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"))
	store := NewMemoryStore()
	store.SeedProcess(Process{
		WorkflowKey: "workflow",
		Name:        "Should not appear for PA",
		Status:      processStatusActive,
		CreatedAt:   now,
		Progress: map[string]ProcessStep{
			"1.1": {State: "done", DoneAt: &now},
			"2.1": {State: "pending"},
		},
	})
	server := &Server{
		identity:    &fakeIdentityStore{},
		store:       store,
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		configDir:   tempDir,
		now:         func() time.Time { return now },
	}
	req := httptest.NewRequest(http.MethodGet, "/my", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()
	server.handleHome(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "Upcoming") || strings.Contains(body, "Should not appear for PA") {
		t.Fatalf("platform-admin /my must not show Upcoming, got:\n%s", body)
	}
}

func writeTwoOrgWorkflowConfig(t *testing.T, path string) {
	t.Helper()
	content := `workflow:
  name: "Two-org workflow"
  steps:
    - id: "1"
      title: "Step 1"
      order: 1
      organization: "org1"
      substeps:
        - id: "1.1"
          title: "Org 1 input"
          order: 1
          roles: ["dep1"]
          inputKey: "value"
          inputType: "formata"
          schema:
            type: object
    - id: "2"
      title: "Step 2"
      order: 2
      organization: "org2"
      substeps:
        - id: "2.1"
          title: "Org 2 input"
          order: 1
          roles: ["dep2"]
          inputKey: "value"
          inputType: "formata"
          schema:
            type: object
organizations:
  - slug: "org1"
    name: "Organization 1"
  - slug: "org2"
    name: "Organization 2"
roles:
  - orgSlug: "org1"
    slug: "dep1"
    name: "Department 1"
  - orgSlug: "org2"
    slug: "dep2"
    name: "Department 2"
users:
  - id: "u1"
    name: "User 1"
    departmentId: "dep1"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write two-org config: %v", err)
	}
}
