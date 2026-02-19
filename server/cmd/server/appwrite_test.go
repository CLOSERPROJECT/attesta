package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeIDs(t *testing.T) {
	got := normalizeIDs([]string{" team-b ", "", "team-a", "team-b"})
	want := []string{"team-a", "team-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeIDs() = %#v, want %#v", got, want)
	}
}

func TestTeamResolverEnabled(t *testing.T) {
	if teamResolverEnabled(nil) {
		t.Fatal("expected nil resolver to be disabled")
	}
	if teamResolverEnabled(NoopTeamResolver{}) {
		t.Fatal("expected noop resolver to be disabled")
	}
	if teamResolverEnabled(&NoopTeamResolver{}) {
		t.Fatal("expected noop pointer resolver to be disabled")
	}
	if !teamResolverEnabled(fakeTeamResolver{teamIDs: []string{"team-a"}}) {
		t.Fatal("expected active resolver to be enabled")
	}
}

func TestAppwriteTeamResolverTeamIDsForUser(t *testing.T) {
	var path string
	var projectHeader string
	var keyHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.String()
		projectHeader = r.Header.Get("X-Appwrite-Project")
		keyHeader = r.Header.Get("X-Appwrite-Key")
		_, _ = w.Write([]byte(`{
			"total": 3,
			"memberships": [
				{"teamId": "team-b"},
				{"teamId": "team-a"},
				{"teamId": "team-b"}
			]
		}`))
	}))
	defer server.Close()

	resolver := NewAppwriteTeamResolver(server.URL, "project-1", "secret-1", server.Client())
	got, err := resolver.TeamIDsForUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("TeamIDsForUser returned error: %v", err)
	}
	want := []string{"team-a", "team-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("TeamIDsForUser() = %#v, want %#v", got, want)
	}
	if !strings.Contains(path, "/v1/users/user-1/memberships") {
		t.Fatalf("request path = %q, expected /v1/users/user-1/memberships", path)
	}
	if projectHeader != "project-1" {
		t.Fatalf("X-Appwrite-Project = %q, want project-1", projectHeader)
	}
	if keyHeader != "secret-1" {
		t.Fatalf("X-Appwrite-Key = %q, want secret-1", keyHeader)
	}
}

func TestAppwriteTeamResolverNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer server.Close()

	resolver := NewAppwriteTeamResolver(server.URL, "project-1", "secret-1", server.Client())
	_, err := resolver.TeamIDsForUser(context.Background(), "user-1")
	if err == nil {
		t.Fatal("expected non-200 response to return an error")
	}
}

func TestAppwriteTeamResolverMembership404ReturnsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	resolver := NewAppwriteTeamResolver(server.URL, "project-1", "secret-1", server.Client())
	teamIDs, err := resolver.TeamIDsForUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("expected 404 to map to empty memberships, got %v", err)
	}
	if len(teamIDs) != 0 {
		t.Fatalf("expected empty team IDs, got %#v", teamIDs)
	}
}

func TestDesiredTeamIDsForUser(t *testing.T) {
	user := User{
		ID:              "u1",
		DepartmentID:    "dep1",
		AppwriteTeamIDs: []string{"team-extra", "team-intake", "team-extra"},
	}
	departmentTeam := map[string]string{"dep1": "team-intake"}
	got := desiredTeamIDsForUser(user, departmentTeam)
	want := []string{"team-extra", "team-intake"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("desiredTeamIDsForUser() = %#v, want %#v", got, want)
	}
}

func TestAppwriteTeamResolverSyncRuntimeConfig(t *testing.T) {
	type teamPayload struct {
		TeamID string `json:"teamId"`
		Name   string `json:"name"`
	}
	type userPayload struct {
		UserID   string `json:"userId"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	type membershipPayload struct {
		Email  string   `json:"email"`
		UserID string   `json:"userId"`
		Roles  []string `json:"roles"`
		URL    string   `json:"url"`
		Name   string   `json:"name"`
	}

	teams := map[string]string{}
	users := map[string]userPayload{}
	memberships := map[string]map[string]bool{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/teams":
			var payload teamPayload
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode team payload: %v", err)
			}
			if _, exists := teams[payload.TeamID]; exists {
				http.Error(w, "exists", http.StatusConflict)
				return
			}
			teams[payload.TeamID] = payload.Name
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/users":
			var payload userPayload
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode user payload: %v", err)
			}
			if _, exists := users[payload.UserID]; exists {
				http.Error(w, "exists", http.StatusConflict)
				return
			}
			users[payload.UserID] = payload
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/teams/") && strings.HasSuffix(r.URL.Path, "/memberships"):
			var payload membershipPayload
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode membership payload: %v", err)
			}
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			if len(parts) < 4 {
				t.Fatalf("unexpected membership path: %s", r.URL.Path)
			}
			teamID := parts[2]
			if memberships[teamID] == nil {
				memberships[teamID] = map[string]bool{}
			}
			if memberships[teamID][payload.UserID] {
				http.Error(w, "exists", http.StatusConflict)
				return
			}
			memberships[teamID][payload.UserID] = true
			w.WriteHeader(http.StatusCreated)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resolver := NewAppwriteTeamResolver(server.URL, "project-1", "secret-1", server.Client())
	cfg := RuntimeConfig{
		Workflow: WorkflowDef{
			Steps: []WorkflowStep{
				{
					StepID: "1",
					Order:  1,
					Substep: []WorkflowSub{
						{SubstepID: "1.1", AppwriteTeamIDs: []string{"team-extra"}},
					},
				},
			},
		},
		Departments: []Department{
			{ID: "dep1", Name: "Intake", AppwriteTeamID: "team-intake"},
		},
		Users: []User{
			{ID: "u1", Name: "User 1", DepartmentID: "dep1", Email: "u1@example.local"},
		},
	}

	report, err := resolver.SyncRuntimeConfig(context.Background(), cfg, AppwriteSyncOptions{
		DefaultPassword: "TempPassw0rd!",
		MembershipURL:   "http://localhost:3030",
	})
	if err != nil {
		t.Fatalf("SyncRuntimeConfig returned error: %v", err)
	}
	if report.TeamsCreated != 2 || report.UsersCreated != 1 || report.MembershipsCreated != 1 {
		t.Fatalf("unexpected report: %#v", report)
	}

	second, err := resolver.SyncRuntimeConfig(context.Background(), cfg, AppwriteSyncOptions{
		DefaultPassword: "TempPassw0rd!",
		MembershipURL:   "http://localhost:3030",
	})
	if err != nil {
		t.Fatalf("second SyncRuntimeConfig returned error: %v", err)
	}
	if second.TeamsCreated != 0 || second.UsersCreated != 0 || second.MembershipsCreated != 0 {
		t.Fatalf("expected idempotent second sync, got %#v", second)
	}
}

func TestSyncAppwriteFromWorkflowsRequiresConfiguredResolver(t *testing.T) {
	server := &Server{
		authorizer: NewCerbosAuthorizer("http://localhost:3592", nil, nil),
	}
	err := server.syncAppwriteFromWorkflows(context.Background())
	if err == nil {
		t.Fatal("expected sync to fail without Appwrite resolver")
	}
}

func TestSyncAppwriteFromWorkflows(t *testing.T) {
	state := struct {
		teams       map[string]bool
		users       map[string]bool
		memberships map[string]map[string]bool
	}{
		teams:       map[string]bool{},
		users:       map[string]bool{},
		memberships: map[string]map[string]bool{},
	}

	appwrite := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/teams":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode team payload: %v", err)
			}
			teamID := payload["teamId"].(string)
			if state.teams[teamID] {
				http.Error(w, "exists", http.StatusConflict)
				return
			}
			state.teams[teamID] = true
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/users":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode user payload: %v", err)
			}
			userID := payload["userId"].(string)
			if state.users[userID] {
				http.Error(w, "exists", http.StatusConflict)
				return
			}
			state.users[userID] = true
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/teams/") && strings.HasSuffix(r.URL.Path, "/memberships"):
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			teamID := parts[2]
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode membership payload: %v", err)
			}
			userID := payload["userId"].(string)
			if state.memberships[teamID] == nil {
				state.memberships[teamID] = map[string]bool{}
			}
			if state.memberships[teamID][userID] {
				http.Error(w, "exists", http.StatusConflict)
				return
			}
			state.memberships[teamID][userID] = true
			w.WriteHeader(http.StatusCreated)
		default:
			http.NotFound(w, r)
		}
	}))
	defer appwrite.Close()

	tempDir := t.TempDir()
	config := `workflow:
  name: "Main"
  steps:
    - id: "1"
      title: "step"
      order: 1
      substeps:
        - id: "1.1"
          title: "sub"
          order: 1
          role: "dep1"
          appwriteTeamIds: ["team-extra"]
          inputKey: "value"
          inputType: "string"
departments:
  - id: "dep1"
    name: "Department 1"
    color: "#fff"
    border: "#000"
    appwriteTeamId: "team-dep1"
users:
  - id: "u1"
    name: "User 1"
    departmentId: "dep1"
    email: "u1@example.local"
`
	if err := os.WriteFile(filepath.Join(tempDir, "workflow.yaml"), []byte(config), 0o644); err != nil {
		t.Fatalf("write workflow config: %v", err)
	}

	authorizer := NewCerbosAuthorizer("http://localhost:3592", nil, nil).WithTeamResolver(
		NewAppwriteTeamResolver(appwrite.URL, "project-1", "key-1", appwrite.Client()),
		false,
	)
	server := &Server{
		authorizer: authorizer,
		configDir:  tempDir,
	}

	t.Setenv("APPWRITE_SYNC_DEFAULT_PASSWORD", "TempPassw0rd!")
	t.Setenv("APPWRITE_SYNC_MEMBERSHIP_URL", "http://localhost:3030")

	if err := server.syncAppwriteFromWorkflows(context.Background()); err != nil {
		t.Fatalf("syncAppwriteFromWorkflows returned error: %v", err)
	}
	if len(state.teams) != 2 {
		t.Fatalf("teams created = %d, want 2", len(state.teams))
	}
	if len(state.users) != 1 {
		t.Fatalf("users created = %d, want 1", len(state.users))
	}
	if len(state.memberships["team-dep1"]) != 1 {
		t.Fatalf("memberships for team-dep1 = %d, want 1", len(state.memberships["team-dep1"]))
	}
}

func TestNewTeamResolverFromEnv(t *testing.T) {
	t.Setenv("APPWRITE_ENDPOINT", "http://example.com")
	t.Setenv("APPWRITE_PROJECT_ID", "project-1")
	t.Setenv("APPWRITE_API_KEY", "key-1")

	resolver := NewTeamResolverFromEnv(nil)
	if _, ok := resolver.(*AppwriteTeamResolver); !ok {
		t.Fatalf("expected AppwriteTeamResolver, got %T", resolver)
	}

	t.Setenv("APPWRITE_API_KEY", "")
	resolver = NewTeamResolverFromEnv(nil)
	if _, ok := resolver.(NoopTeamResolver); !ok {
		t.Fatalf("expected NoopTeamResolver when env is incomplete, got %T", resolver)
	}
}
