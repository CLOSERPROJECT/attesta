package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeTeamResolver struct {
	teamIDs []string
	err     error
}

func (f fakeTeamResolver) TeamIDsForUser(_ context.Context, _ string) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.teamIDs, nil
}

func TestNewCerbosAuthorizerDefaults(t *testing.T) {
	authorizer := NewCerbosAuthorizer("http://localhost:3592", nil, nil)
	if authorizer.client == nil {
		t.Fatal("expected default client to be set")
	}
	if authorizer.now == nil {
		t.Fatal("expected default clock to be set")
	}
	if authorizer.teamResolver == nil {
		t.Fatal("expected default team resolver to be set")
	}
	if authorizer.strictTeamSync {
		t.Fatal("expected strict team sync disabled by default")
	}
}

func TestCerbosAuthorizerCanCompleteBuildsRequestAndMapsAllow(t *testing.T) {
	var captured map[string]interface{}
	var method string
	var path string
	var contentType string

	pdp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		contentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_, _ = w.Write([]byte(`{
  "resourceInstances": {
    "1.1": {"actions": {"complete": "EFFECT_ALLOW"}}
  }
}`))
	}))
	defer pdp.Close()

	fixed := time.Unix(1700000000, 123)
	authorizer := NewCerbosAuthorizer(pdp.URL+"/", pdp.Client(), func() time.Time { return fixed })

	allowed, err := authorizer.CanComplete(context.Background(), Actor{UserID: "u1", Role: "dep1", WorkflowKey: "wf-a"}, "proc-1", "wf-a", WorkflowSub{
		SubstepID: "1.1",
		Order:     2,
		Role:      "dep1",
	}, 1, true)
	if err != nil {
		t.Fatalf("CanComplete returned error: %v", err)
	}
	if !allowed {
		t.Fatal("expected allow result")
	}
	if method != http.MethodPost {
		t.Fatalf("method = %q, want POST", method)
	}
	if path != "/api/check" {
		t.Fatalf("path = %q, want /api/check", path)
	}
	if contentType != "application/json" {
		t.Fatalf("content-type = %q, want application/json", contentType)
	}

	if got := captured["requestId"]; got != "req-1700000000000000123" {
		t.Fatalf("requestId = %#v, want %q", got, "req-1700000000000000123")
	}

	principal := captured["principal"].(map[string]interface{})
	if principal["id"] != "u1" {
		t.Fatalf("principal.id = %#v, want u1", principal["id"])
	}
	roles := principal["roles"].([]interface{})
	if len(roles) != 1 || roles[0] != "dep1" {
		t.Fatalf("principal.roles = %#v, want [dep1]", roles)
	}
	principalAttr := principal["attr"].(map[string]interface{})
	if principalAttr["workflowKey"] != "wf-a" {
		t.Fatalf("principal.attr.workflowKey = %#v, want wf-a", principalAttr["workflowKey"])
	}
	if principalAttr["appwriteTeamsEnabled"] != false {
		t.Fatalf("principal.attr.appwriteTeamsEnabled = %#v, want false", principalAttr["appwriteTeamsEnabled"])
	}
	teamIDs := principalAttr["appwriteTeamIds"].([]interface{})
	if len(teamIDs) != 0 {
		t.Fatalf("principal.attr.appwriteTeamIds = %#v, want []", teamIDs)
	}

	resource := captured["resource"].(map[string]interface{})
	instances := resource["instances"].(map[string]interface{})
	sub := instances["1.1"].(map[string]interface{})
	attr := sub["attr"].(map[string]interface{})
	if attr["roleRequired"] != "dep1" {
		t.Fatalf("roleRequired = %#v, want dep1", attr["roleRequired"])
	}
	if attr["processId"] != "proc-1" || attr["substepId"] != "1.1" {
		t.Fatalf("process/substep attrs = %#v", attr)
	}
	if attr["workflowKey"] != "wf-a" {
		t.Fatalf("workflowKey = %#v, want wf-a", attr["workflowKey"])
	}
	if attr["sequenceOk"] != true {
		t.Fatalf("sequenceOk = %#v, want true", attr["sequenceOk"])
	}
	if attr["stepOrder"] != float64(1) || attr["substepOrder"] != float64(2) {
		t.Fatalf("order attrs = %#v", attr)
	}
	requiredTeamIDs := attr["requiredTeamIds"].([]interface{})
	if len(requiredTeamIDs) != 0 {
		t.Fatalf("requiredTeamIds = %#v, want []", requiredTeamIDs)
	}
}

func TestCerbosAuthorizerCanCompleteIncludesAppwriteTeams(t *testing.T) {
	var captured map[string]interface{}
	pdp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_, _ = w.Write([]byte(`{"resourceInstances":{"1.1":{"actions":{"complete":"EFFECT_ALLOW"}}}}`))
	}))
	defer pdp.Close()

	authorizer := NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now).WithTeamResolver(fakeTeamResolver{
		teamIDs: []string{"team-b", "team-a", "team-b"},
	}, false)

	allowed, err := authorizer.CanComplete(context.Background(), Actor{UserID: "u1", Role: "dep1", WorkflowKey: "wf-a"}, "proc-1", "wf-a", WorkflowSub{
		SubstepID:       "1.1",
		Order:           1,
		Role:            "dep1",
		AppwriteTeamIDs: []string{"team-x", "team-y", "team-x"},
	}, 1, true)
	if err != nil {
		t.Fatalf("CanComplete returned error: %v", err)
	}
	if !allowed {
		t.Fatal("expected allow result")
	}

	principal := captured["principal"].(map[string]interface{})
	principalAttr := principal["attr"].(map[string]interface{})
	if principalAttr["appwriteTeamsEnabled"] != true {
		t.Fatalf("principal.attr.appwriteTeamsEnabled = %#v, want true", principalAttr["appwriteTeamsEnabled"])
	}
	teamIDs := principalAttr["appwriteTeamIds"].([]interface{})
	if len(teamIDs) != 2 || teamIDs[0] != "team-a" || teamIDs[1] != "team-b" {
		t.Fatalf("principal.attr.appwriteTeamIds = %#v, want [team-a team-b]", teamIDs)
	}

	resource := captured["resource"].(map[string]interface{})
	instances := resource["instances"].(map[string]interface{})
	sub := instances["1.1"].(map[string]interface{})
	attr := sub["attr"].(map[string]interface{})
	required := attr["requiredTeamIds"].([]interface{})
	if len(required) != 2 || required[0] != "team-x" || required[1] != "team-y" {
		t.Fatalf("resource.attr.requiredTeamIds = %#v, want [team-x team-y]", required)
	}
}

func TestCerbosAuthorizerCanCompleteMapsDenyAndUnknownToFalse(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "deny effect",
			body: `{"resourceInstances":{"1.1":{"actions":{"complete":"EFFECT_DENY"}}}}`,
		},
		{
			name: "missing action",
			body: `{"resourceInstances":{"1.1":{"actions":{}}}}`,
		},
		{
			name: "missing resource instance",
			body: `{"resourceInstances":{"2.1":{"actions":{"complete":"EFFECT_ALLOW"}}}}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pdp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tc.body))
			}))
			defer pdp.Close()

			authorizer := NewCerbosAuthorizer(pdp.URL, pdp.Client(), func() time.Time {
				return time.Unix(1700000000, 0)
			})
			allowed, err := authorizer.CanComplete(context.Background(), Actor{UserID: "u1", Role: "dep1", WorkflowKey: "wf-a"}, "proc-1", "wf-a", WorkflowSub{
				SubstepID: "1.1",
				Order:     1,
				Role:      "dep1",
			}, 1, true)
			if err != nil {
				t.Fatalf("CanComplete returned error: %v", err)
			}
			if allowed {
				t.Fatal("expected deny/unknown to map to false")
			}
		})
	}
}

func TestCerbosAuthorizerCanCompleteReturnsErrorForBadStatusAndBadJSON(t *testing.T) {
	badStatus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer badStatus.Close()

	authorizer := NewCerbosAuthorizer(badStatus.URL, badStatus.Client(), time.Now)
	_, err := authorizer.CanComplete(context.Background(), Actor{UserID: "u1", Role: "dep1", WorkflowKey: "wf-a"}, "proc-1", "wf-a", WorkflowSub{
		SubstepID: "1.1",
		Order:     1,
		Role:      "dep1",
	}, 1, true)
	if err == nil {
		t.Fatal("expected error for non-200 cerbos status")
	}

	badJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{"))
	}))
	defer badJSON.Close()

	authorizer = NewCerbosAuthorizer(badJSON.URL, badJSON.Client(), time.Now)
	_, err = authorizer.CanComplete(context.Background(), Actor{UserID: "u1", Role: "dep1", WorkflowKey: "wf-a"}, "proc-1", "wf-a", WorkflowSub{
		SubstepID: "1.1",
		Order:     1,
		Role:      "dep1",
	}, 1, true)
	if err == nil {
		t.Fatal("expected JSON decode error")
	}
}

func TestCerbosAuthorizerCanCompleteTeamLookupFailOpenAndFailClosed(t *testing.T) {
	pdp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"resourceInstances":{"1.1":{"actions":{"complete":"EFFECT_ALLOW"}}}}`))
	}))
	defer pdp.Close()

	failOpen := NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now).WithTeamResolver(fakeTeamResolver{
		err: errors.New("boom"),
	}, false)
	allowed, err := failOpen.CanComplete(context.Background(), Actor{UserID: "u1", Role: "dep1", WorkflowKey: "wf-a"}, "proc-1", "wf-a", WorkflowSub{
		SubstepID: "1.1",
		Order:     1,
		Role:      "dep1",
	}, 1, true)
	if err != nil {
		t.Fatalf("expected fail-open mode to ignore team lookup errors, got %v", err)
	}
	if !allowed {
		t.Fatal("expected allow result in fail-open mode")
	}

	failClosed := NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now).WithTeamResolver(fakeTeamResolver{
		err: errors.New("boom"),
	}, true)
	_, err = failClosed.CanComplete(context.Background(), Actor{UserID: "u1", Role: "dep1", WorkflowKey: "wf-a"}, "proc-1", "wf-a", WorkflowSub{
		SubstepID: "1.1",
		Order:     1,
		Role:      "dep1",
	}, 1, true)
	if err == nil {
		t.Fatal("expected fail-closed mode to return team lookup errors")
	}
}

func TestCerbosAuthorizerCanCompleteMarksTeamsDisabledWhenLookupFails(t *testing.T) {
	var captured map[string]interface{}
	pdp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_, _ = w.Write([]byte(`{"resourceInstances":{"1.1":{"actions":{"complete":"EFFECT_ALLOW"}}}}`))
	}))
	defer pdp.Close()

	authorizer := NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now).WithTeamResolver(fakeTeamResolver{
		err: errors.New("network timeout"),
	}, false)
	_, err := authorizer.CanComplete(context.Background(), Actor{UserID: "u1", Role: "dep1", WorkflowKey: "wf-a"}, "proc-1", "wf-a", WorkflowSub{
		SubstepID:       "1.1",
		Order:           1,
		Role:            "dep1",
		AppwriteTeamIDs: []string{"team-a"},
	}, 1, true)
	if err != nil {
		t.Fatalf("expected fail-open mode without error, got %v", err)
	}

	principal := captured["principal"].(map[string]interface{})
	principalAttr := principal["attr"].(map[string]interface{})
	if principalAttr["appwriteTeamsEnabled"] != false {
		t.Fatalf("principal.attr.appwriteTeamsEnabled = %#v, want false", principalAttr["appwriteTeamsEnabled"])
	}
}
