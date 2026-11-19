package org

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateTaskConfigsRejectsKindGroup(t *testing.T) {
	err := validateTaskConfigs([]TaskConfig{{
		"id": "lots", "kind": "group",
		"tasks": []TaskConfig{{"id": "register"}},
	}})
	if err == nil || !strings.Contains(err.Error(), "kind") || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("kind group = %v", err)
	}
}

func TestValidateTaskConfigsRejectsGroupOnlyKeys(t *testing.T) {
	cases := []TaskConfig{
		{"id": "lots", "closeRequiresChildSubmission": true},
		{"id": "lots", "closeLabel": "Chiudi"},
		{"id": "lots", "instanceLabelEntity": "lot"},
		{"id": "lots", "tasks": []TaskConfig{{"id": "register"}}},
		{"id": "lots", "substeps": []TaskConfig{{"id": "register"}}},
	}
	for _, tc := range cases {
		err := validateTaskConfigs([]TaskConfig{tc})
		if err == nil {
			t.Fatalf("expected reject for %#v", tc)
		}
		if !strings.Contains(err.Error(), "not supported") {
			t.Fatalf("%#v err = %v, want not supported", tc, err)
		}
	}
}

func TestCommitRejectsSpawningNewMint(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	_, err := c.Commit(t.Context(), FeragameScope(), adminOp(), CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: "T1",
		Value:         map[string]any{},
		Mint:          &CommitMint{Kind: "spawningNew"},
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("spawningNew = %v", err)
	}
	if ce.Message != "mint.kind must be startingNew or none" {
		t.Fatalf("message = %q", ce.Message)
	}
}

func TestCommitRejectsNestedTaskPath(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	_, err := c.Commit(t.Context(), FeragameScope(), adminOp(), CommitRequest{
		TaskPath:      "intake/lots/L1/register",
		JobInstanceID: "T1",
		Value:         map[string]any{},
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unknown-target" {
		t.Fatalf("nested path = %v", err)
	}
}

func TestCloseHTTPNotFound(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	h := &API{
		DomainLoader:  loader,
		TaskCommitter: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return adminOp(), true
		},
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/org/close", strings.NewReader(
		`{"jobId":"intake","jobInstanceId":"T1","groupId":"lots"}`,
	)))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("close = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestProjectWorklistOmitsGroupChrome(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	domain := EmptyDomain()
	domain.JobInstances = []JobInstance{{JobID: "intake", ID: "T1", Label: "truck"}}
	wl := projectWorklistUnfiltered(cfg, domain, FixtureAllRolesOp())
	job := worklistMustJob(t, wl, "intake")
	inst, ok := worklistInstance(job, "T1")
	if !ok {
		t.Fatal("missing intake instance")
	}
	raw, err := json.Marshal(inst)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"groups"`) || strings.Contains(string(raw), "canSpawn") || strings.Contains(string(raw), "canClose") {
		t.Fatalf("Worklist instance still projects group chrome: %s", raw)
	}
}
