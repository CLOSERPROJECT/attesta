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

func TestAppwriteIdentityCreateEmailPasswordSession(t *testing.T) {
	var method string
	var path string
	var projectHeader string
	var keyHeader string
	var body map[string]interface{}

	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		projectHeader = r.Header.Get("X-Appwrite-Project")
		keyHeader = r.Header.Get("X-Appwrite-Key")
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"$id":"session-1",
			"userId":"user-1",
			"expire":"2026-03-18T10:11:12Z",
			"secret":"session-secret"
		}`))
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	session, err := identity.CreateEmailPasswordSession(context.Background(), "user@example.com", "pw-123456789012")
	if err != nil {
		t.Fatalf("CreateEmailPasswordSession error: %v", err)
	}

	if method != http.MethodPost {
		t.Fatalf("method = %q, want POST", method)
	}
	if path != "/v1/account/sessions/email" {
		t.Fatalf("path = %q, want /v1/account/sessions/email", path)
	}
	if projectHeader != "project-1" {
		t.Fatalf("project header = %q, want project-1", projectHeader)
	}
	if keyHeader != "api-key-1" {
		t.Fatalf("key header = %q, want api-key-1", keyHeader)
	}
	if body["email"] != "user@example.com" {
		t.Fatalf("email payload = %#v, want user@example.com", body["email"])
	}
	if body["password"] != "pw-123456789012" {
		t.Fatalf("password payload = %#v", body["password"])
	}
	if session.Secret != "session-secret" {
		t.Fatalf("session secret = %q, want session-secret", session.Secret)
	}
	if session.UserID != "user-1" {
		t.Fatalf("session user id = %q, want user-1", session.UserID)
	}
	if !session.ExpiresAt.Equal(time.Date(2026, time.March, 18, 10, 11, 12, 0, time.UTC)) {
		t.Fatalf("expires at = %s", session.ExpiresAt)
	}
}

func TestAppwriteIdentityGetCurrentUserHydratesMembership(t *testing.T) {
	var accountSessionHeader string
	var membershipsKeyHeader string

	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/account":
			accountSessionHeader = r.Header.Get("X-Appwrite-Session")
			_, _ = w.Write([]byte(`{
				"$id":"user-1",
				"email":"org-admin@example.com",
				"status":true,
				"labels":["attesta:org-admin","attesta:role:qa-reviewer"]
			}`))
		case "/v1/users/user-1/memberships":
			membershipsKeyHeader = r.Header.Get("X-Appwrite-Key")
			_, _ = w.Write([]byte(`{
				"total":1,
				"memberships":[
					{
						"$id":"membership-1",
						"userId":"user-1",
						"teamId":"acme",
						"teamName":"Acme Org",
						"confirm":true,
						"roles":["owner","member"]
					}
				]
			}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	user, err := identity.GetCurrentUser(context.Background(), "session-secret")
	if err != nil {
		t.Fatalf("GetCurrentUser error: %v", err)
	}

	if accountSessionHeader != "session-secret" {
		t.Fatalf("account session header = %q, want session-secret", accountSessionHeader)
	}
	if membershipsKeyHeader != "api-key-1" {
		t.Fatalf("memberships key header = %q, want api-key-1", membershipsKeyHeader)
	}
	if user.ID != "user-1" {
		t.Fatalf("user id = %q, want user-1", user.ID)
	}
	if user.Email != "org-admin@example.com" {
		t.Fatalf("email = %q, want org-admin@example.com", user.Email)
	}
	if user.OrgSlug != "acme" || user.OrgName != "Acme Org" {
		t.Fatalf("org = %#v", user)
	}
	if user.MembershipID != "membership-1" {
		t.Fatalf("membership id = %q, want membership-1", user.MembershipID)
	}
	if len(user.MembershipRoles) != 2 || user.MembershipRoles[0] != "owner" || user.MembershipRoles[1] != "member" {
		t.Fatalf("membership roles = %#v", user.MembershipRoles)
	}
	if !user.IsOrgAdmin {
		t.Fatal("expected org admin")
	}
	if user.Status != "active" {
		t.Fatalf("status = %q, want active", user.Status)
	}
}

func TestAppwriteIdentityListOrganizationsDecodesPrefs(t *testing.T) {
	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/teams" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"total":1,
			"teams":[
				{
					"$id":"acme",
					"name":"Acme Org",
					"prefs":{
						"schemaVersion":1,
						"logoFileId":"logo-file-1",
						"roles":[
							{"slug":"qa-reviewer","name":"QA Reviewer","color":"#123456","border":"solid"},
							{"slug":"qa-approver","name":"QA Approver","color":"#654321","border":"dashed"}
						]
					}
				}
			]
		}`))
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	orgs, err := identity.ListOrganizations(context.Background())
	if err != nil {
		t.Fatalf("ListOrganizations error: %v", err)
	}
	if len(orgs) != 1 {
		t.Fatalf("len(orgs) = %d, want 1", len(orgs))
	}
	if orgs[0].ID != "acme" || orgs[0].Slug != "acme" || orgs[0].Name != "Acme Org" {
		t.Fatalf("org = %#v", orgs[0])
	}
	if orgs[0].LogoFileID != "logo-file-1" {
		t.Fatalf("logo = %q, want logo-file-1", orgs[0].LogoFileID)
	}
	if len(orgs[0].Roles) != 2 {
		t.Fatalf("roles = %#v", orgs[0].Roles)
	}
	if orgs[0].Roles[0].Slug != "qa-reviewer" || orgs[0].Roles[1].Slug != "qa-approver" {
		t.Fatalf("roles = %#v", orgs[0].Roles)
	}
}

func TestAppwriteIdentityNormalizesNotFound(t *testing.T) {
	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	_, err := identity.GetOrganizationBySlug(context.Background(), "missing")
	if !errors.Is(err, ErrIdentityNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrIdentityNotFound)
	}
}
