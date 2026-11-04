package living

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCerbosAuthorizerAllowCommitBuildsLivingCheck(t *testing.T) {
	var captured map[string]any
	pdp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/check" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode: %v", err)
		}
		_, _ = w.Write([]byte(`{"resourceInstances":{"intake/register":{"actions":{"commit":"EFFECT_ALLOW"}}}}`))
	}))
	defer pdp.Close()

	fixed := time.Unix(1700000000, 42)
	auth := NewCerbosAuthorizer(pdp.URL+"/", pdp.Client(), func() time.Time { return fixed })
	allowed, err := auth.AllowCommit(t.Context(), AuthorizeCommit{
		Operator:     OperatorIdentity{UserID: "ada", Org: "operator-org", Roles: []string{"administration", "line"}},
		Org:          "target-org",
		TaskPath:     "intake/register",
		JobID:        "intake",
		TaskID:       "register",
		RolesAllowed: []string{"administration"},
		SequenceOk:   true,
		ReadyWhenOk:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Fatal("expected allow")
	}
	if captured["requestId"] != "living-1700000000000000042" {
		t.Fatalf("requestId = %#v", captured["requestId"])
	}
	principal := captured["principal"].(map[string]any)
	if principal["id"] != "ada" {
		t.Fatalf("principal.id = %#v", principal["id"])
	}
	roles := principal["roles"].([]any)
	if len(roles) != 1 || roles[0] != "authenticated" {
		t.Fatalf("principal.roles = %#v", roles)
	}
	pattr := principal["attr"].(map[string]any)
	if pattr["orgSlug"] != "operator-org" {
		t.Fatalf("principal.orgSlug = %#v", pattr["orgSlug"])
	}
	resource := captured["resource"].(map[string]any)
	if resource["kind"] != "living" {
		t.Fatalf("kind = %#v", resource["kind"])
	}
	inst := resource["instances"].(map[string]any)["intake/register"].(map[string]any)
	attr := inst["attr"].(map[string]any)
	if attr["orgSlug"] != "target-org" {
		t.Fatalf("resource.orgSlug = %#v", attr["orgSlug"])
	}
	if attr["taskPath"] != "intake/register" || attr["jobId"] != "intake" || attr["taskId"] != "register" {
		t.Fatalf("resource attr = %#v", attr)
	}
	allowedRoles := attr["rolesAllowed"].([]any)
	if len(allowedRoles) != 1 || allowedRoles[0] != "administration" {
		t.Fatalf("rolesAllowed = %#v", allowedRoles)
	}
	if attr["sequenceOk"] != true {
		t.Fatalf("sequenceOk = %#v", attr["sequenceOk"])
	}
	if attr["readyWhenOk"] != true {
		t.Fatalf("readyWhenOk = %#v", attr["readyWhenOk"])
	}
	actions := captured["actions"].([]any)
	if len(actions) != 1 || actions[0] != "commit" {
		t.Fatalf("actions = %#v", actions)
	}
}

func TestCerbosAuthorizerAllowCommitMapsDeny(t *testing.T) {
	pdp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"resourceInstances":{"intake/register":{"actions":{"commit":"EFFECT_DENY"}}}}`))
	}))
	defer pdp.Close()

	auth := NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now)
	allowed, err := auth.AllowCommit(t.Context(), AuthorizeCommit{
		Operator:     OperatorIdentity{UserID: "gio", Org: "feragame", Roles: []string{"logistic"}},
		Org:          "feragame",
		TaskPath:     "intake/register",
		RolesAllowed: []string{"administration"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if allowed {
		t.Fatal("expected deny")
	}
}

func TestCerbosAuthorizerAllowCommitErrorsOnBadStatus(t *testing.T) {
	pdp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer pdp.Close()

	auth := NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now)
	_, err := auth.AllowCommit(t.Context(), AuthorizeCommit{
		Operator: OperatorIdentity{UserID: "ada", Org: "feragame"},
		Org:      "feragame",
		TaskPath: "intake/register",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
