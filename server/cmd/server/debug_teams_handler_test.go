package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestHandleDebugTeamsReturnsResolvedMemberships(t *testing.T) {
	cfg := testRuntimeConfig()
	cfg.Workflow.Steps[1].Substep[0].AppwriteTeamIDs = []string{"team-refinery"}
	cfg.Workflow.Steps[2].Substep[0].AppwriteTeamIDs = []string{"team-qa"}

	server := &Server{
		authorizer: NewCerbosAuthorizer("http://localhost:3592", nil, time.Now).WithTeamResolver(fakeTeamResolver{
			teamIDs: []string{"team-refinery"},
		}, false),
		configProvider: func() (RuntimeConfig, error) { return cfg, nil },
	}

	req := httptest.NewRequest(http.MethodGet, "/debug/teams", nil)
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: cfg,
	}))
	req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u2|dep2|workflow"})
	rec := httptest.NewRecorder()

	server.handleDebugTeams(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	appwrite := body["appwrite"].(map[string]interface{})
	if appwrite["enabled"] != true {
		t.Fatalf("appwrite.enabled = %#v, want true", appwrite["enabled"])
	}
	teamIDs := appwrite["teamIds"].([]interface{})
	if len(teamIDs) != 1 || teamIDs[0] != "team-refinery" {
		t.Fatalf("appwrite.teamIds = %#v, want [team-refinery]", teamIDs)
	}

	restricted := body["teamRestrictedSubsteps"].([]interface{})
	if len(restricted) != 2 {
		t.Fatalf("restricted substeps len = %d, want 2", len(restricted))
	}
	first := restricted[0].(map[string]interface{})
	if first["substepId"] != "2.1" || first["teamMatch"] != true {
		t.Fatalf("first restricted substep = %#v, want substep 2.1 with match true", first)
	}
	second := restricted[1].(map[string]interface{})
	if second["substepId"] != "3.1" || second["teamMatch"] != false {
		t.Fatalf("second restricted substep = %#v, want substep 3.1 with match false", second)
	}
}

func TestHandleDebugTeamsMethodNotAllowed(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/debug/teams", nil)
	rec := httptest.NewRecorder()

	server.handleDebugTeams(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleLegacyDebugTeamsScopesWorkflowByQuery(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")
	server := &Server{
		configDir: tempDir,
	}
	req := httptest.NewRequest(http.MethodGet, "/debug/teams?workflow=workflow", nil)
	rec := httptest.NewRecorder()

	server.handleLegacyDebugTeams(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
